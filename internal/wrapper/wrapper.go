package wrapper

import (
	"fmt"
	"log"
	"nssm-plus/internal/common"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// serviceHandler implements svc.Handler for the Windows service wrapper.
type serviceHandler struct {
	serviceName string
}

// Execute is called by the Windows SCM when the service is started.
// It supports automatic restart on crash (configurable via RestartDelay).
func (h *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	log.Printf("[%s] Execute: service handler starting", h.serviceName)
	changes <- svc.Status{State: svc.StartPending}

	cfg, err := LoadConfig(h.serviceName)
	if err != nil {
		log.Printf("[%s] Failed to load config: %v", h.serviceName, err)
		changes <- svc.Status{State: svc.Stopped}
		return false, 1
	}
	log.Printf("[%s] Config loaded: appPath=%q, args=%q, dir=%q, restartDelay=%d, logStdout=%q, logStderr=%q, rotateLog=%v, restartTimeout=%d",
		h.serviceName, cfg.AppPath, cfg.Arguments, cfg.WorkDir, cfg.RestartDelay, cfg.LogStdout, cfg.LogStderr, cfg.RotateLog, cfg.RestartTimeout)

	// Ensure log directory exists
	if err := os.MkdirAll(LogsDir(), 0755); err != nil {
		log.Printf("[%s] Failed to create log directory: %v", h.serviceName, err)
	}

	// Setup log file for child process output
	logFile, err := os.OpenFile(LogPath(h.serviceName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[%s] Failed to open log file: %v", h.serviceName, err)
	} else {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	// Main loop: start process and handle events, with optional restart on crash
	for {
		// Reload config on each iteration to pick up changes
		if newCfg, loadErr := LoadConfig(h.serviceName); loadErr == nil {
			cfg = newCfg
		}

		exitCode, stopped := h.runOnce(cfg, r, changes, logFile)
		if stopped {
			return false, 0
		}

		// No restart configured or first start immediate exit
		if cfg.RestartDelay <= 0 {
			changes <- svc.Status{State: svc.Stopped}
			return false, uint32(exitCode)
		}

		// Wait for restart delay, checking for stop requests
		if h.waitForRestart(r, changes, cfg.RestartDelay) {
			return false, 0
		}
	}
}

// runOnce starts the child process and waits for it to exit or a control signal.
// Returns (exitCode, stopped). stopped=true means a Stop/Shutdown was requested.
func (h *serviceHandler) runOnce(cfg *WrapperConfig, r <-chan svc.ChangeRequest, changes chan<- svc.Status, logFile *os.File) (exitCode int, stopped bool) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	log.Printf("[%s] Starting service: %s %s", h.serviceName, cfg.AppPath, cfg.Arguments)

	cmd := exec.Command(cfg.AppPath, splitArgs(cfg.Arguments)...)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if len(cfg.Env) > 0 {
		log.Printf("[%s] Setting %d environment variables: %v", h.serviceName, len(cfg.Env), func() []string {
			keys := make([]string, 0, len(cfg.Env))
			for k := range cfg.Env {
				keys = append(keys, k)
			}
			return keys
		}())
	}

	// Setup stdout/stderr redirection
	stdoutPath := cfg.LogStdout
	stderrPath := cfg.LogStderr
	if stdoutPath == "" {
		stdoutPath = LogPath(h.serviceName)
	}
	if stderrPath == "" {
		stderrPath = stdoutPath
	}
	log.Printf("[%s] Log paths: stdout=%q, stderr=%q", h.serviceName, stdoutPath, stderrPath)

	if err := os.MkdirAll(filepath.Dir(stdoutPath), 0755); err != nil {
		log.Printf("[%s] Failed to create stdout log directory: %v", h.serviceName, err)
	}
	if stderrPath != stdoutPath {
		if err := os.MkdirAll(filepath.Dir(stderrPath), 0755); err != nil {
			log.Printf("[%s] Failed to create stderr log directory: %v", h.serviceName, err)
		}
	}

	// Rotate log files if enabled
	if cfg.RotateLog {
		log.Printf("[%s] Rotating log files (stdout=%q)", h.serviceName, stdoutPath)
		rotateLog(stdoutPath)
		if stderrPath != stdoutPath {
			rotateLog(stderrPath)
		}
	}

	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[%s] Failed to open stdout log file: %v", h.serviceName, err)
	} else {
		cmd.Stdout = stdoutFile
		defer stdoutFile.Close()
	}

	if stderrPath == stdoutPath && stdoutFile != nil {
		cmd.Stderr = stdoutFile
	} else {
		stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("[%s] Failed to open stderr log file: %v", h.serviceName, err)
		} else {
			cmd.Stderr = stderrFile
			defer stderrFile.Close()
		}
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[%s] Failed to start process: %v", h.serviceName, err)
		changes <- svc.Status{State: svc.Stopped}
		return 1, false
	}

	pid := cmd.Process.Pid
	log.Printf("[%s] Started process PID=%d", h.serviceName, pid)
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Monitor process exit in background
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait a brief moment to confirm process started successfully
	select {
	case <-time.After(500 * time.Millisecond):
		// Process is running
	case err := <-done:
		// Process exited immediately
		log.Printf("[%s] Process exited immediately: %v", h.serviceName, err)
		return 1, false
	}

	// Main event loop for this process run
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Printf("[%s] Stopping service (PID=%d)...", h.serviceName, pid)
				changes <- svc.Status{State: svc.StopPending}
				stopProcess(pid, done, cfg.RestartTimeout)
				log.Printf("[%s] Service stopped", h.serviceName)
				changes <- svc.Status{State: svc.Stopped}
				return 0, true
			}
		case err := <-done:
			if err != nil {
				log.Printf("[%s] Process exited with error: %v", h.serviceName, err)
				return 1, false
			}
			log.Printf("[%s] Process exited normally", h.serviceName)
			return 0, false
		}
	}
}

// waitForRestart waits the specified delay before allowing a restart.
// Returns true if a Stop/Shutdown was requested during the wait.
func (h *serviceHandler) waitForRestart(r <-chan svc.ChangeRequest, changes chan<- svc.Status, delaySec int) bool {
	log.Printf("[%s] Process crashed, restarting in %d seconds...", h.serviceName, delaySec)
	changes <- svc.Status{State: svc.StopPending}

	timer := time.NewTimer(time.Duration(delaySec) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			log.Printf("[%s] Restart delay elapsed, restarting process...", h.serviceName)
			changes <- svc.Status{State: svc.StartPending}
			return false
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				log.Printf("[%s] Stop requested during restart delay", h.serviceName)
				changes <- svc.Status{State: svc.Stopped}
				return true
			case svc.Interrogate:
				changes <- c.CurrentStatus
			}
		}
	}
}

// stopProcess kills the process and its entire process tree.
// It first tries a graceful kill, then waits up to timeoutSec seconds before force killing.
// The done channel is used to detect when the process has actually exited.
func stopProcess(pid int, done <-chan error, timeoutSec int) {
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	log.Printf("[wrapper] stopProcess: PID=%d, timeout=%ds", pid, timeoutSec)

	cmd := exec.Command("taskkill", "/T", "/PID", fmt.Sprintf("%d", pid))
	if err := cmd.Run(); err != nil {
		log.Printf("[wrapper] Graceful kill signal for PID %d failed: %v", pid, err)
	} else {
		log.Printf("[wrapper] stopProcess: taskkill /T /PID %d sent successfully", pid)
	}

	select {
	case <-done:
		log.Printf("[wrapper] Process (PID %d) exited gracefully", pid)
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		log.Printf("[wrapper] PID %d graceful shutdown timed out after %ds, force killing...", pid, timeoutSec)
		cmd = exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
		if err := cmd.Run(); err != nil {
			log.Printf("[wrapper] Force kill of PID %d failed: %v", pid, err)
		}
		<-done
	}
}

// Run starts the service wrapper. It detects whether it's running as a
// Windows service or in console mode (for debugging).
func Run(serviceName string) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to detect service mode: %w", err)
	}

	if !isService {
		// Interactive/console mode for debugging
		log.Printf("Running in console mode (not as Windows service)")
		log.Printf("Service name: %s", serviceName)
		log.Printf("Config path: %s", ConfigPath(serviceName))
		return runConsole(serviceName)
	}

	// Service mode — use Windows event log + log file
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open event log: %w", err)
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", serviceName))
	if err := svc.Run(serviceName, &serviceHandler{serviceName: serviceName}); err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", serviceName, err))
		return err
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", serviceName))
	return nil
}

// runConsole simulates the service handler in console mode (for debugging).
func runConsole(serviceName string) error {
	h := &serviceHandler{serviceName: serviceName}

	requests := make(chan svc.ChangeRequest)
	changes := make(chan svc.Status)

	go h.Execute([]string{serviceName}, requests, changes)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case status, ok := <-changes:
			if !ok {
				return nil
			}
			log.Printf("Service status: %v", status.State)
			if status.State == svc.Stopped {
				return nil
			}
		case <-sigChan:
			log.Printf("Received interrupt signal, stopping service...")
			requests <- svc.ChangeRequest{Cmd: svc.Stop}
		}
	}
}

// GetWrapperBinaryPath returns the BinaryPathName that should be set in SCM.
func GetWrapperBinaryPath(exePath, serviceName string) string {
	return common.GetWrapperBinaryPath(exePath, serviceName)
}

// IsWrapperBinaryPath checks if a BinaryPathName belongs to an NSSM-Plus wrapped service.
func IsWrapperBinaryPath(binaryPathName string) bool {
	return common.IsWrapperBinaryPath(binaryPathName)
}

// ExtractServiceName extracts the service name from a wrapper BinaryPathName.
func ExtractServiceName(binaryPathName string) string {
	return common.ExtractServiceName(binaryPathName)
}

// rotateLog performs simple log rotation by renaming the current log file
// with a .1 suffix and shifting older rotated files (.1 -> .2, .2 -> .3, etc.).
// Up to 5 rotated files are kept. Rotation only occurs if the file exceeds defaultLogSize.
func rotateLog(logPath string) {
	info, err := os.Stat(logPath)
	if err != nil || info.Size() < int64(defaultLogSize) {
		if err != nil {
			log.Printf("[wrapper] rotateLog: cannot stat %q: %v", logPath, err)
		}
		return
	}

	log.Printf("[wrapper] rotateLog: rotating %q (size=%d bytes, threshold=%d bytes)", logPath, info.Size(), defaultLogSize)

	maxRotations := 5
	for i := maxRotations - 1; i >= 1; i-- {
		older := fmt.Sprintf("%s.%d", logPath, i)
		newer := fmt.Sprintf("%s.%d", logPath, i+1)
		if i == maxRotations-1 {
			os.Remove(older)
		}
		if _, err := os.Stat(older); err == nil {
			os.Rename(older, newer)
		}
	}
	os.Rename(logPath, fmt.Sprintf("%s.1", logPath))
}
