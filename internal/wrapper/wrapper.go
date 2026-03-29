package wrapper

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// serviceHandler implements svc.Handler for the Windows service wrapper.
type serviceHandler struct {
	serviceName string
}

// Execute is called by the Windows SCM when the service is started.
func (h *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	cfg, err := LoadConfig(h.serviceName)
	if err != nil {
		log.Printf("[%s] Failed to load config: %v", h.serviceName, err)
		changes <- svc.Status{State: svc.Stopped}
		return false, 1
	}

	// Ensure log directory exists
	os.MkdirAll(LogsDir(), 0755)

	// Setup log file for child process output
	logFile, err := os.OpenFile(LogPath(h.serviceName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[%s] Failed to open log file: %v", h.serviceName, err)
	} else {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Printf("[%s] Starting service: %s %s", h.serviceName, cfg.AppPath, cfg.Arguments)

	// Build and start child process
	cmd := exec.Command(cfg.AppPath, splitArgs(cfg.Arguments)...)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}
	// Inherit environment and add custom vars
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// Redirect stdout/stderr to log file
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[%s] Failed to start process: %v", h.serviceName, err)
		changes <- svc.Status{State: svc.Stopped}
		return false, 1
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
		changes <- svc.Status{State: svc.Stopped}
		return false, 1
	}

	// Main event loop
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Printf("[%s] Stopping service (PID=%d)...", h.serviceName, pid)
				changes <- svc.Status{State: svc.StopPending}
				stopProcess(pid)
				<-done
				log.Printf("[%s] Service stopped", h.serviceName)
				changes <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		case err := <-done:
			exitCode := 0
			if err != nil {
				exitCode = 1
				log.Printf("[%s] Process exited with error: %v", h.serviceName, err)
			} else {
				log.Printf("[%s] Process exited normally", h.serviceName)
			}
			changes <- svc.Status{State: svc.StopPending}
			changes <- svc.Status{State: svc.Stopped}
			return false, uint32(exitCode)
		}
	}
}

// stopProcess kills the process and its entire process tree.
func stopProcess(pid int) {
	// Try graceful kill first using taskkill without /F
	cmd := exec.Command("taskkill", "/T", "/PID", fmt.Sprintf("%d", pid))
	cmd.Run()

	// Wait briefly for graceful shutdown
	time.Sleep(5 * time.Second)

	// Force kill remaining processes
	cmd = exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	cmd.Run()
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

	// Create mock channels
	requests := make(chan svc.ChangeRequest)
	changes := make(chan svc.Status)

	go h.Execute([]string{serviceName}, requests, changes)

	// In console mode, just wait for the process to finish
	for status := range changes {
		log.Printf("Service status: %v", status.State)
		if status.State == svc.Stopped {
			return nil
		}
	}
	return nil
}

// GetWrapperBinaryPath returns the BinaryPathName that should be set in SCM.
// Format: "<path-to-exe>" service <serviceName>
func GetWrapperBinaryPath(exePath, serviceName string) string {
	// Quote the exe path if needed
	if !strings.HasPrefix(exePath, `"`) {
		exePath = `"` + exePath + `"`
	}
	return exePath + " service " + serviceName
}

// IsWrapperBinaryPath checks if a BinaryPathName belongs to an NSSM-Plus wrapped service.
func IsWrapperBinaryPath(binaryPathName string) bool {
	return strings.Contains(binaryPathName, " service ")
}

// ExtractServiceName extracts the service name from a wrapper BinaryPathName.
func ExtractServiceName(binaryPathName string) string {
	idx := strings.LastIndex(binaryPathName, " service ")
	if idx < 0 {
		return ""
	}
	name := binaryPathName[idx+len(" service "):]
	name = strings.TrimSpace(name)
	// Remove surrounding quotes if present
	if strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`) {
		name = name[1 : len(name)-1]
	}
	return name
}
