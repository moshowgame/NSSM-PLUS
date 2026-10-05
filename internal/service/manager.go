package service

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"nssm-plus/internal/common"
	"nssm-plus/internal/wrapper"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	// Service marker to identify NSSM Plus managed services
	nssmPlusMarker = "NSSM-Plus"

	// Windows SERVICE_NO_CHANGE: tells ChangeServiceConfig not to modify this field
	serviceNoChange = 0xFFFFFFFF
)

// ServiceConfig holds all configuration for a Windows service
type ServiceConfig struct {
	ServiceName    string            `json:"serviceName"`
	DisplayName    string            `json:"displayName"`
	Description    string            `json:"description"`
	AppPath        string            `json:"appPath"`
	Arguments      string            `json:"arguments"`
	WorkDir        string            `json:"workDir"`
	StartType      string            `json:"startType"` // auto, demand, disabled
	Account        string            `json:"account"`
	Password       string            `json:"password"`
	Environment    map[string]string `json:"environment"`
	LogStdout      string            `json:"logStdout"`
	LogStderr      string            `json:"logStderr"`
	RotateLog      bool              `json:"rotateLog"`
	RestartDelay   int               `json:"restartDelay"`   // seconds, 0 = no restart
	RestartTimeout int               `json:"restartTimeout"` // seconds
	Dependencies   []string          `json:"dependencies"`
}

// ServiceInfo contains basic service information for listing
type ServiceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
	StartType   string `json:"startType"`
	AppPath     string `json:"appPath"`
}

// Manager handles Windows service operations
type Manager struct{}

// NewManager creates a new service manager
func NewManager() *Manager {
	return &Manager{}
}

// connectSCM connects to the Service Control Manager
func connectSCM() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SCM (need admin privileges): %w", err)
	}
	return m, nil
}

// toMgrStartType converts our string start type to mgr constant
func toMgrStartType(startType string) uint32 {
	switch startType {
	case "demand", "manual":
		return mgr.StartManual
	case "disabled":
		return mgr.StartDisabled
	default:
		return mgr.StartAutomatic
	}
}

// buildDescription adds NSSM Plus marker to description
func buildDescription(desc, appPath string) string {
	if desc == "" {
		return fmt.Sprintf("[%s] %s", nssmPlusMarker, appPath)
	}
	return fmt.Sprintf("[%s] %s", nssmPlusMarker, desc)
}

// validateServiceName checks that a service name contains only valid characters.
// Windows service names: up to 256 chars, letters, digits, spaces, and limited special chars.
func validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("service name must be 256 characters or less")
	}
	for _, ch := range name {
		if !isValidServiceNameChar(ch) {
			return fmt.Errorf("service name contains invalid character: '%c' (allowed: letters, digits, spaces, _-.)", ch)
		}
	}
	return nil
}

func isValidServiceNameChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == ' ' || ch == '_' || ch == '-' || ch == '.'
}

// validateAppPath checks that the application path exists on disk.
func validateAppPath(appPath string) error {
	if appPath == "" {
		return fmt.Errorf("application path is required")
	}
	trimmed := strings.TrimSpace(appPath)
	// Remove surrounding quotes for the check
	trimmed = strings.Trim(trimmed, `"`)
	if _, err := os.Stat(trimmed); err != nil {
		return fmt.Errorf("application path does not exist: %s", trimmed)
	}
	return nil
}

// Install creates a new Windows service
func (m *Manager) Install(cfg ServiceConfig) error {
	log.Printf("[service] Install: name=%q, appPath=%q, args=%q, dir=%q, startType=%q, account=%q",
		cfg.ServiceName, cfg.AppPath, cfg.Arguments, cfg.WorkDir, cfg.StartType, cfg.Account)

	if err := validateServiceName(cfg.ServiceName); err != nil {
		log.Printf("[service] Install: validation failed for service name: %v", err)
		return err
	}
	if err := validateAppPath(cfg.AppPath); err != nil {
		log.Printf("[service] Install: validation failed for app path: %v", err)
		return err
	}

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] Install: failed to connect to SCM: %v", err)
		return err
	}
	defer scMgr.Disconnect()

	existingSvc, err := scMgr.OpenService(cfg.ServiceName)
	if err == nil {
		existingSvc.Close()
		log.Printf("[service] Install: service '%s' already exists", cfg.ServiceName)
		return fmt.Errorf("service '%s' already exists. Use Modify to update it", cfg.ServiceName)
	}

	wrapperCfg := &wrapper.WrapperConfig{
		AppPath:        strings.TrimSpace(cfg.AppPath),
		Arguments:      strings.TrimSpace(cfg.Arguments),
		WorkDir:        strings.TrimSpace(cfg.WorkDir),
		Env:            cfg.Environment,
		RestartDelay:   cfg.RestartDelay,
		LogStdout:      strings.TrimSpace(cfg.LogStdout),
		LogStderr:      strings.TrimSpace(cfg.LogStderr),
		RotateLog:      cfg.RotateLog,
		RestartTimeout: cfg.RestartTimeout,
	}
	if err := wrapper.SaveConfig(cfg.ServiceName, wrapperCfg); err != nil {
		log.Printf("[service] Install: failed to save wrapper config for '%s': %v", cfg.ServiceName, err)
		return fmt.Errorf("failed to save wrapper config: %w", err)
	}
	log.Printf("[service] Install: wrapper config saved for '%s'", cfg.ServiceName)

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[service] Install: failed to get executable path: %v", err)
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	binaryPath := wrapper.GetWrapperBinaryPath(exePath, cfg.ServiceName)
	log.Printf("[service] Install: binaryPath=%q", binaryPath)

	desc := buildDescription(cfg.Description, cfg.AppPath)
	h, err := windows.CreateService(
		scMgr.Handle,
		syscall.StringToUTF16Ptr(cfg.ServiceName),
		syscall.StringToUTF16Ptr(cfg.DisplayName),
		windows.SERVICE_ALL_ACCESS,
		windows.SERVICE_WIN32_OWN_PROCESS,
		toMgrStartType(cfg.StartType),
		windows.SERVICE_ERROR_NORMAL,
		syscall.StringToUTF16Ptr(binaryPath),
		nil,
		nil,
		toStringBlock(cfg.Dependencies),
		syscall.StringToUTF16Ptr(cfg.Account),
		syscall.StringToUTF16Ptr(cfg.Password),
	)
	if err != nil {
		log.Printf("[service] Install: CreateService failed for '%s': %v", cfg.ServiceName, err)
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer windows.CloseServiceHandle(h)

	if desc != "" {
		d := windows.SERVICE_DESCRIPTION{Description: syscall.StringToUTF16Ptr(desc)}
		windows.ChangeServiceConfig2(h, windows.SERVICE_CONFIG_DESCRIPTION, (*byte)(unsafe.Pointer(&d)))
	}

	log.Printf("[service] Install: service '%s' installed successfully", cfg.ServiceName)
	return nil
}

// Remove deletes an existing Windows service
func (m *Manager) Remove(serviceName string) error {
	log.Printf("[service] Remove: name=%q", serviceName)

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] Remove: failed to connect to SCM: %v", err)
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		log.Printf("[service] Remove: failed to open service '%s': %v", serviceName, err)
		return fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}

	status, err := s.Query()
	if err != nil {
		s.Close()
		log.Printf("[service] Remove: failed to query service '%s' status: %v", serviceName, err)
		return fmt.Errorf("failed to query service status: %w", err)
	}

	if status.State != svc.Stopped {
		log.Printf("[service] Remove: service '%s' is running (state=%d), stopping first...", serviceName, status.State)
		_, err = s.Control(svc.Stop)
		if err != nil {
			if !strings.Contains(err.Error(), "not been started") &&
				!strings.Contains(err.Error(), "not running") {
				s.Close()
				log.Printf("[service] Remove: failed to stop service '%s': %v", serviceName, err)
				return fmt.Errorf("failed to stop service before removal: %w", err)
			}
		}
		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			st, err := s.Query()
			if err != nil {
				break
			}
			if st.State == svc.Stopped {
				log.Printf("[service] Remove: service '%s' stopped (waited %dms)", serviceName, (i+1)*500)
				break
			}
		}
	}

	err = s.Delete()
	s.Close()
	if err != nil {
		if strings.Contains(err.Error(), "marked for deletion") {
			log.Printf("[service] Remove: service '%s' already marked for deletion", serviceName)
			return fmt.Errorf("service '%s' is already marked for deletion (will be removed on next restart or after handles are released)", serviceName)
		}
		log.Printf("[service] Remove: failed to delete service '%s': %v", serviceName, err)
		return fmt.Errorf("failed to delete service: %w", err)
	}

	wrapper.DeleteConfig(serviceName)
	log.Printf("[service] Remove: service '%s' removed successfully", serviceName)
	return nil
}

// Start starts a Windows service
func (m *Manager) Start(serviceName string) error {
	log.Printf("[service] Start: name=%q", serviceName)

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] Start: failed to connect to SCM: %v", err)
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		log.Printf("[service] Start: failed to open service '%s': %v", serviceName, err)
		return fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		log.Printf("[service] Start: failed to start service '%s': %v", serviceName, err)
		return fmt.Errorf("failed to start service: %w", err)
	}

	log.Printf("[service] Start: service '%s' started successfully", serviceName)
	return nil
}

// Stop stops a running Windows service
func (m *Manager) Stop(serviceName string) error {
	log.Printf("[service] Stop: name=%q", serviceName)

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] Stop: failed to connect to SCM: %v", err)
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		log.Printf("[service] Stop: failed to open service '%s': %v", serviceName, err)
		return fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		log.Printf("[service] Stop: failed to stop service '%s': %v", serviceName, err)
		return fmt.Errorf("failed to stop service: %w", err)
	}

	log.Printf("[service] Stop: stop signal sent to service '%s'", serviceName)
	return nil
}

// Restart stops and then starts a Windows service
func (m *Manager) Restart(serviceName string) error {
	log.Printf("[service] Restart: name=%q", serviceName)

	err := m.Stop(serviceName)
	if err != nil {
		if !strings.Contains(err.Error(), "not been started") &&
			!strings.Contains(err.Error(), "not running") {
			log.Printf("[service] Restart: failed to stop service '%s': %v", serviceName, err)
			return fmt.Errorf("failed to stop service for restart: %w", err)
		}
		log.Printf("[service] Restart: service '%s' was not running, proceeding with start", serviceName)
	}

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] Restart: failed to connect to SCM for wait: %v", err)
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err == nil {
		for i := 0; i < 30; i++ {
			status, qErr := s.Query()
			if qErr != nil || status.State == svc.Stopped {
				if qErr == nil {
					log.Printf("[service] Restart: service '%s' reached Stopped state (waited %dms)", serviceName, (i+1)*500)
				}
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		s.Close()
	}

	log.Printf("[service] Restart: starting service '%s'...", serviceName)
	return m.Start(serviceName)
}

// GetStatus queries the current status of a service
func (m *Manager) GetStatus(serviceName string) (string, error) {
	log.Printf("[service] GetStatus: name=%q", serviceName)

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] GetStatus: failed to connect to SCM: %v", err)
		return "", err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		log.Printf("[service] GetStatus: failed to open service '%s': %v", serviceName, err)
		return "", fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		log.Printf("[service] GetStatus: failed to query service '%s': %v", serviceName, err)
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	result := statusToString(status.State)
	log.Printf("[service] GetStatus: service '%s' status=%s", serviceName, result)
	return result, nil
}

// ListServices lists all services managed by NSSM Plus
func (m *Manager) ListServices() ([]ServiceInfo, error) {
	log.Printf("[service] ListServices: scanning for NSSM Plus managed services")

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] ListServices: failed to connect to SCM: %v", err)
		return nil, err
	}
	defer scMgr.Disconnect()

	services, err := scMgr.ListServices()
	if err != nil {
		log.Printf("[service] ListServices: failed to list services: %v", err)
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	log.Printf("[service] ListServices: total Windows services found: %d", len(services))

	var result []ServiceInfo
	for _, name := range services {
		s, err := scMgr.OpenService(name)
		if err != nil {
			continue
		}

		cfg, err := s.Config()
		if err != nil {
			s.Close()
			continue
		}

		if !isNssmPlusService(cfg.Description) {
			s.Close()
			continue
		}

		status, err := s.Query()
		if err != nil {
			s.Close()
			continue
		}

		info := ServiceInfo{
			Name:        name,
			DisplayName: cfg.DisplayName,
			Status:      statusToString(status.State),
			StartType:   startTypeToString(cfg.StartType),
		}
		if wrapper.IsWrapperBinaryPath(cfg.BinaryPathName) {
			svcName := wrapper.ExtractServiceName(cfg.BinaryPathName)
			if svcName != "" && wrapper.ConfigExists(svcName) {
				if wCfg, err := wrapper.LoadConfig(svcName); err == nil {
					info.AppPath = wCfg.AppPath
				}
			}
		} else {
			info.AppPath = cfg.BinaryPathName
		}
		result = append(result, info)
		s.Close()
	}

	log.Printf("[service] ListServices: found %d NSSM Plus managed services", len(result))
	return result, nil
}

// Modify updates an existing service configuration
func (m *Manager) Modify(oldName string, cfg ServiceConfig) error {
	log.Printf("[service] Modify: oldName=%q, newName=%q, appPath=%q, startType=%q",
		oldName, cfg.ServiceName, cfg.AppPath, cfg.StartType)

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] Modify: failed to connect to SCM: %v", err)
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(oldName)
	if err != nil {
		log.Printf("[service] Modify: failed to open service '%s': %v", oldName, err)
		return fmt.Errorf("failed to open service '%s': %w", oldName, err)
	}
	defer s.Close()

	currentCfg, err := s.Config()
	if err != nil {
		log.Printf("[service] Modify: failed to get current config for '%s': %v", oldName, err)
		return fmt.Errorf("failed to get current service config: %w", err)
	}

	var binaryPath string
	if common.IsWrapperBinaryPath(currentCfg.BinaryPathName) {
		exePath, err := os.Executable()
		if err != nil {
			log.Printf("[service] Modify: failed to get executable path: %v", err)
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		binaryPath = common.GetWrapperBinaryPath(exePath, oldName)
	} else {
		binaryPath = strings.TrimSpace(cfg.AppPath)
		args := strings.TrimSpace(cfg.Arguments)
		if args != "" {
			binaryPath = binaryPath + " " + args
		}
	}
	log.Printf("[service] Modify: binaryPath=%q", binaryPath)

	wrapperCfg := &wrapper.WrapperConfig{
		AppPath:        strings.TrimSpace(cfg.AppPath),
		Arguments:      strings.TrimSpace(cfg.Arguments),
		WorkDir:        strings.TrimSpace(cfg.WorkDir),
		Env:            cfg.Environment,
		RestartDelay:   cfg.RestartDelay,
		LogStdout:      strings.TrimSpace(cfg.LogStdout),
		LogStderr:      strings.TrimSpace(cfg.LogStderr),
		RotateLog:      cfg.RotateLog,
		RestartTimeout: cfg.RestartTimeout,
	}
	if err := wrapper.SaveConfig(oldName, wrapperCfg); err != nil {
		log.Printf("[service] Modify: failed to update wrapper config for '%s': %v", oldName, err)
		return fmt.Errorf("failed to update wrapper config: %w", err)
	}
	log.Printf("[service] Modify: wrapper config updated for '%s'", oldName)

	err = s.UpdateConfig(mgr.Config{
		ServiceType:      serviceNoChange,
		StartType:        toMgrStartType(cfg.StartType),
		ErrorControl:     serviceNoChange,
		BinaryPathName:   binaryPath,
		DisplayName:      cfg.DisplayName,
		Description:      buildDescription(cfg.Description, cfg.AppPath),
		ServiceStartName: cfg.Account,
		Password:         cfg.Password,
		Dependencies:     cfg.Dependencies,
	})
	if err != nil {
		log.Printf("[service] Modify: UpdateConfig failed for '%s': %v", oldName, err)
		return fmt.Errorf("failed to update service config: %w", err)
	}

	log.Printf("[service] Modify: service '%s' updated successfully", oldName)
	return nil
}

// GetServiceConfig retrieves the full configuration of a service
func (m *Manager) GetServiceConfig(serviceName string) (*ServiceConfig, error) {
	log.Printf("[service] GetServiceConfig: name=%q", serviceName)

	scMgr, err := connectSCM()
	if err != nil {
		log.Printf("[service] GetServiceConfig: failed to connect to SCM: %v", err)
		return nil, err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		log.Printf("[service] GetServiceConfig: failed to open service '%s': %v", serviceName, err)
		return nil, fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	return buildServiceConfig(s, serviceName)
}

// buildServiceConfig reads the full configuration from an open service
// handle, combining SCM properties with the wrapper config when the
// service is NSSM-Plus managed.
func buildServiceConfig(s *mgr.Service, serviceName string) (*ServiceConfig, error) {
	cfg, err := s.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get service config: %w", err)
	}

	result := &ServiceConfig{
		ServiceName:  serviceName,
		DisplayName:  cfg.DisplayName,
		Description:  cleanDescription(cfg.Description),
		StartType:    startTypeToString(cfg.StartType),
		Account:      cfg.ServiceStartName,
		Dependencies: cfg.Dependencies,
	}

	if wrapper.IsWrapperBinaryPath(cfg.BinaryPathName) {
		svcName := wrapper.ExtractServiceName(cfg.BinaryPathName)
		if svcName != "" && wrapper.ConfigExists(svcName) {
			wCfg, err := wrapper.LoadConfig(svcName)
			if err == nil {
				result.AppPath = wCfg.AppPath
				result.Arguments = wCfg.Arguments
				result.WorkDir = wCfg.WorkDir
				result.Environment = wCfg.Env
				result.LogStdout = wCfg.LogStdout
				result.LogStderr = wCfg.LogStderr
				result.RotateLog = wCfg.RotateLog
				result.RestartDelay = wCfg.RestartDelay
				result.RestartTimeout = wCfg.RestartTimeout
				return result, nil
			}
		}
	}

	result.AppPath, result.Arguments = parseBinaryPathName(cfg.BinaryPathName)
	return result, nil
}

// --- Helper functions ---

func statusToString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "Start Pending"
	case svc.Running:
		return "Running"
	case svc.StopPending:
		return "Stop Pending"
	case svc.ContinuePending:
		return "Continue Pending"
	case svc.PausePending:
		return "Pause Pending"
	case svc.Paused:
		return "Paused"
	default:
		return "Unknown"
	}
}

func startTypeToString(startType uint32) string {
	switch startType {
	case windows.SERVICE_AUTO_START:
		return "Automatic"
	case windows.SERVICE_DEMAND_START:
		return "Manual"
	case windows.SERVICE_DISABLED:
		return "Disabled"
	default:
		return "Unknown"
	}
}

func isNssmPlusService(description string) bool {
	if description == "" {
		return false
	}
	return strings.HasPrefix(description, "["+nssmPlusMarker+"]")
}

func cleanDescription(description string) string {
	prefix := "[" + nssmPlusMarker + "] "
	if strings.HasPrefix(description, prefix) {
		return strings.TrimPrefix(description, prefix)
	}
	return description
}

// parseBinaryPathName splits BinaryPathName into appPath and arguments.
// BinaryPathName format: "C:\path\app.exe" --arg1 --arg2
func parseBinaryPathName(binaryPathName string) (appPath string, arguments string) {
	binaryPathName = strings.TrimSpace(binaryPathName)
	if binaryPathName == "" {
		return "", ""
	}
	// If quoted, extract the quoted part as appPath
	if strings.HasPrefix(binaryPathName, `"`) {
		endQuote := strings.Index(binaryPathName[1:], `"`)
		if endQuote >= 0 {
			appPath = binaryPathName[1 : endQuote+1]
			arguments = strings.TrimSpace(binaryPathName[endQuote+2:])
			return appPath, arguments
		}
	}
	// No quotes — split on first space
	idx := strings.Index(binaryPathName, " ")
	if idx >= 0 {
		return binaryPathName[:idx], strings.TrimSpace(binaryPathName[idx+1:])
	}
	return binaryPathName, ""
}

// toStringBlock converts a string slice to a null-terminated multi-string block
// (required by Windows service APIs for dependencies).
func toStringBlock(ss []string) *uint16 {
	if len(ss) == 0 {
		return nil
	}
	t := ""
	for _, s := range ss {
		if s != "" {
			t += s + "\x00"
		}
	}
	if t == "" {
		return nil
	}
	t += "\x00"
	return &utf16.Encode([]rune(t))[0]
}
