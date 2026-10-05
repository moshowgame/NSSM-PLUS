package main

import (
	"context"
	"fmt"
	"nssm-plus/internal/config"
	"nssm-plus/internal/service"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	mu     sync.Mutex
	config *config.Manager
	mgr    *service.Manager
}

func NewApp() *App {
	return &App{
		config: config.NewManager(),
		mgr:    service.NewManager(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// --- Service Operations ---

// InstallService installs a new Windows service
func (a *App) InstallService(cfg service.ServiceConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mgr.Install(cfg)
}

// RemoveService removes an existing Windows service
func (a *App) RemoveService(serviceName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mgr.Remove(serviceName)
}

// StartService starts a Windows service
func (a *App) StartService(serviceName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mgr.Start(serviceName)
}

// StopService stops a running Windows service
func (a *App) StopService(serviceName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mgr.Stop(serviceName)
}

// RestartService restarts a Windows service
func (a *App) RestartService(serviceName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mgr.Restart(serviceName)
}

// GetServiceStatus queries the status of a Windows service
func (a *App) GetServiceStatus(serviceName string) (string, error) {
	return a.mgr.GetStatus(serviceName)
}

// GetInstalledServices lists all services managed by NSSM Plus
func (a *App) GetInstalledServices() ([]service.ServiceInfo, error) {
	return a.mgr.ListServices()
}

// ModifyService updates an existing service configuration
func (a *App) ModifyService(oldName string, cfg service.ServiceConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mgr.Modify(oldName, cfg)
}

// GetServiceConfig retrieves the configuration of an existing service
func (a *App) GetServiceConfig(serviceName string) (*service.ServiceConfig, error) {
	return a.mgr.GetServiceConfig(serviceName)
}

// GetSyncStates reconciles file-based configs against the actually
// registered services, returning per-service sync state and field diffs.
func (a *App) GetSyncStates(configs []service.ServiceConfig) ([]service.ServiceDiff, error) {
	return a.mgr.GetSyncStates(configs)
}

// GetOrphanConfigs lists wrapper configs that have no registered service.
func (a *App) GetOrphanConfigs() ([]service.ServiceInfo, error) {
	return a.mgr.GetOrphanConfigs()
}

// GetWrapperConfig loads the wrapper config of a service directly from disk,
// without requiring the service to be registered.
func (a *App) GetWrapperConfig(serviceName string) (*service.ServiceConfig, error) {
	return a.mgr.GetWrapperConfig(serviceName)
}

// --- File Dialog Operations (via Wails Go runtime) ---

// ShowOpenDialog opens a native Open File dialog filtered for JSON config files.
func (a *App) ShowOpenDialog(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

// ShowSaveDialog opens a native Save File dialog and returns the selected path
func (a *App) ShowSaveDialog(title string, defaultFilename string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
}

// ShowOpenAppDialog opens a native Open File dialog filtered for executables.
func (a *App) ShowOpenAppDialog(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "Executables (*.exe;*.bat;*.cmd;*.ps1)", Pattern: "*.exe;*.bat;*.cmd;*.ps1"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

// ShowOpenDirectoryDialog opens a native directory picker dialog.
func (a *App) ShowOpenDirectoryDialog(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

// ValidateAppPath checks if a given application path exists on disk.
func (a *App) ValidateAppPath(appPath string) error {
	trimmed := strings.TrimSpace(appPath)
	if trimmed == "" {
		return fmt.Errorf("application path is required")
	}
	trimmed = strings.Trim(trimmed, `"`)
	if _, err := os.Stat(trimmed); err != nil {
		return fmt.Errorf("application path does not exist: %s", trimmed)
	}
	return nil
}

// --- Config File Operations (multi-service) ---

// SaveConfigToFile saves the given service configurations to a multi-service JSON file.
// The configs are provided by the frontend and include the currently edited form state.
func (a *App) SaveConfigToFile(filePath string, configs []service.ServiceConfig) error {
	return a.config.SaveToFile(filePath, configs)
}

// LoadConfigFromFile loads service configs from a multi-service JSON file
func (a *App) LoadConfigFromFile(filePath string) ([]service.ServiceConfig, error) {
	return a.config.LoadFromFile(filePath)
}

// OpenInExplorer opens Windows File Explorer and selects the specified file
func (a *App) OpenInExplorer(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("no file path specified")
	}
	// Use cmd /c to ensure proper argument passing to explorer.exe
	cmd := exec.Command("cmd", "/c", "explorer.exe", "/select,"+filePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
