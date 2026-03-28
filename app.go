package main

import (
	"context"
	"fmt"
	"nssm-plus/internal/config"
	"nssm-plus/internal/service"
	"sync"
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

// --- Config File Operations ---

// SaveConfigToFile saves a service configuration to a JSON file
func (a *App) SaveConfigToFile(filePath string, cfg service.ServiceConfig) error {
	return a.config.SaveToFile(filePath, cfg)
}

// LoadConfigFromFile loads a service configuration from a JSON file
func (a *App) LoadConfigFromFile(filePath string) (*service.ServiceConfig, error) {
	return a.config.LoadFromFile(filePath)
}

// ExportAllConfigs exports all service configs to a single JSON file
func (a *App) ExportAllConfigs(filePath string) error {
	services, err := a.mgr.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}
	configs := make([]service.ServiceConfig, 0, len(services))
	for _, svc := range services {
		cfg, err := a.mgr.GetServiceConfig(svc.Name)
		if err != nil {
			continue
		}
		configs = append(configs, *cfg)
	}
	return a.config.ExportAll(filePath, configs)
}

// ImportConfigs imports service configs from a JSON file
func (a *App) ImportConfigs(filePath string) ([]service.ServiceConfig, error) {
	return a.config.ImportConfigs(filePath)
}


