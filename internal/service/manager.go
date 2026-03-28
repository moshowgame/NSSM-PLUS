package service

import (
	"fmt"
	"strings"
	"time"

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

// Install creates a new Windows service
func (m *Manager) Install(cfg ServiceConfig) error {
	if cfg.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if cfg.AppPath == "" {
		return fmt.Errorf("application path is required")
	}

	scMgr, err := connectSCM()
	if err != nil {
		return err
	}
	defer scMgr.Disconnect()

	// Check if service already exists
	existingSvc, err := scMgr.OpenService(cfg.ServiceName)
	if err == nil {
		existingSvc.Close()
		return fmt.Errorf("service '%s' already exists. Use Modify to update it", cfg.ServiceName)
	}

	// Build binary path: AppPath contains the full command line (e.g. "java -server -jar app.jar")
	binaryPath := cfg.AppPath
	if cfg.Arguments != "" {
		binaryPath = cfg.AppPath + " " + cfg.Arguments
	}
	// Quote the executable part if it contains spaces
	binaryPath = quoteExeInCmdLine(binaryPath)

	// Create service
	s, err := scMgr.CreateService(
		cfg.ServiceName,
		binaryPath,
		mgr.Config{
			ServiceType:      windows.SERVICE_WIN32_OWN_PROCESS,
			StartType:        toMgrStartType(cfg.StartType),
			ErrorControl:     windows.SERVICE_ERROR_NORMAL,
			BinaryPathName:   binaryPath,
			DisplayName:      cfg.DisplayName,
			Description:      buildDescription(cfg.Description, cfg.AppPath),
			ServiceStartName: cfg.Account,
			Password:         cfg.Password,
			Dependencies:     cfg.Dependencies,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	return nil
}

// Remove deletes an existing Windows service
func (m *Manager) Remove(serviceName string) error {
	scMgr, err := connectSCM()
	if err != nil {
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}

	// Stop the service first if it's running, then wait for it to fully stop
	status, err := s.Query()
	if err != nil {
		s.Close()
		return fmt.Errorf("failed to query service status: %w", err)
	}
	if status.State != svc.Stopped {
		_, err = s.Control(svc.Stop)
		if err != nil {
			if !strings.Contains(err.Error(), "not been started") &&
				!strings.Contains(err.Error(), "not running") {
				s.Close()
				return fmt.Errorf("failed to stop service before removal: %w", err)
			}
		}
		// Wait for the service to fully stop (up to 15 seconds)
		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			st, err := s.Query()
			if err != nil {
				break
			}
			if st.State == svc.Stopped {
				break
			}
		}
	}

	// Delete must be called before closing the handle, but close immediately after
	err = s.Delete()
	s.Close()
	if err != nil {
		if strings.Contains(err.Error(), "marked for deletion") {
			return fmt.Errorf("service '%s' is already marked for deletion (will be removed on next restart or after handles are released)", serviceName)
		}
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// Start starts a Windows service
func (m *Manager) Start(serviceName string) error {
	scMgr, err := connectSCM()
	if err != nil {
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// Stop stops a running Windows service
func (m *Manager) Stop(serviceName string) error {
	scMgr, err := connectSCM()
	if err != nil {
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

// Restart stops and then starts a Windows service
func (m *Manager) Restart(serviceName string) error {
	err := m.Stop(serviceName)
	if err != nil {
		// If service is not running, just try starting it
		if !strings.Contains(err.Error(), "not been started") &&
			!strings.Contains(err.Error(), "not running") {
			return fmt.Errorf("failed to stop service for restart: %w", err)
		}
	}
	return m.Start(serviceName)
}

// GetStatus queries the current status of a service
func (m *Manager) GetStatus(serviceName string) (string, error) {
	scMgr, err := connectSCM()
	if err != nil {
		return "", err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	return statusToString(status.State), nil
}

// ListServices lists all services managed by NSSM Plus
func (m *Manager) ListServices() ([]ServiceInfo, error) {
	scMgr, err := connectSCM()
	if err != nil {
		return nil, err
	}
	defer scMgr.Disconnect()

	services, err := scMgr.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

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
			AppPath:     cfg.BinaryPathName,
		}
		result = append(result, info)
		s.Close()
	}

	return result, nil
}

// Modify updates an existing service configuration
func (m *Manager) Modify(oldName string, cfg ServiceConfig) error {
	scMgr, err := connectSCM()
	if err != nil {
		return err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(oldName)
	if err != nil {
		return fmt.Errorf("failed to open service '%s': %w", oldName, err)
	}
	defer s.Close()

	// Build binary path: AppPath contains the full command line (e.g. "java -server -jar app.jar")
	binaryPath := cfg.AppPath
	if cfg.Arguments != "" {
		binaryPath = cfg.AppPath + " " + cfg.Arguments
	}
	// Quote the executable part if it contains spaces
	binaryPath = quoteExeInCmdLine(binaryPath)

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
		return fmt.Errorf("failed to update service config: %w", err)
	}

	return nil
}

// GetServiceConfig retrieves the full configuration of a service
func (m *Manager) GetServiceConfig(serviceName string) (*ServiceConfig, error) {
	scMgr, err := connectSCM()
	if err != nil {
		return nil, err
	}
	defer scMgr.Disconnect()

	s, err := scMgr.OpenService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open service '%s': %w", serviceName, err)
	}
	defer s.Close()

	cfg, err := s.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get service config: %w", err)
	}

	result := &ServiceConfig{
		ServiceName: serviceName,
		DisplayName: cfg.DisplayName,
		Description: cleanDescription(cfg.Description),
		AppPath:     stripOuterQuotes(cfg.BinaryPathName),
		StartType:   startTypeToString(cfg.StartType),
		Account:     cfg.ServiceStartName,
	}

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
	// Remove "[NSSM-Plus] " prefix from description
	prefix := "[" + nssmPlusMarker + "] "
	if strings.HasPrefix(description, prefix) {
		return strings.TrimPrefix(description, prefix)
	}
	// Also handle "[NSSM-Plus]" case for "Managed by NSSM-Plus - path"
	if strings.HasPrefix(description, "["+nssmPlusMarker+"] ") {
		rest := description[len(nssmPlusMarker)+3:]
		return rest
	}
	return description
}

// quoteExeInCmdLine quotes the executable part of a command line if it contains spaces.
// e.g. "C:\Program Files\app.exe --port 8080" -> "\"C:\Program Files\app.exe\" --port 8080"
// e.g. "java -server -jar app.jar" -> "java -server -jar app.jar" (no change)
func quoteExeInCmdLine(cmdLine string) string {
	cmdLine = strings.TrimSpace(cmdLine)
	if cmdLine == "" {
		return ""
	}

	var exe string
	var rest string

	if strings.HasPrefix(cmdLine, `"`) {
		// Already quoted — keep as-is
		endQuote := strings.Index(cmdLine[1:], `"`)
		if endQuote >= 0 {
			return cmdLine
		}
		// Unclosed quote, take as-is
		return cmdLine
	}

	// Split on first space
	idx := strings.Index(cmdLine, " ")
	if idx < 0 {
		exe = cmdLine
	} else {
		exe = cmdLine[:idx]
		rest = cmdLine[idx:]
	}

	// Quote the exe if it contains spaces and isn't already quoted
	if strings.Contains(exe, " ") && !strings.HasPrefix(exe, `"`) {
		exe = `"` + exe + `"`
	}

	if rest == "" {
		return exe
	}
	return exe + rest
}

// stripOuterQuotes removes surrounding quotes from a path string.
// Windows SCM may store BinaryPathName as "java" -jar app.jar,
// this strips the outer quotes to get: java -jar app.jar
func stripOuterQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
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
