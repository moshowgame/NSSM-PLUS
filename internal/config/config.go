package config

import (
	"encoding/json"
	"fmt"
	"os"
	"nssm-plus/internal/service"
)

// Manager handles configuration file operations
type Manager struct{}

// NewManager creates a new config manager
func NewManager() *Manager {
	return &Manager{}
}

// SaveToFile saves a service configuration to a JSON file
func (m *Manager) SaveToFile(filePath string, cfg service.ServiceConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadFromFile loads a service configuration from a JSON file
func (m *Manager) LoadFromFile(filePath string) (*service.ServiceConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg service.ServiceConfig
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// ExportAll exports multiple service configs to a single JSON file
func (m *Manager) ExportAll(filePath string, configs []service.ServiceConfig) error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportConfigs imports service configs from a JSON file
func (m *Manager) ImportConfigs(filePath string) ([]service.ServiceConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file: %w", err)
	}

	var configs []service.ServiceConfig
	err = json.Unmarshal(data, &configs)
	if err != nil {
		// Try single config format
		var single service.ServiceConfig
		if singleErr := json.Unmarshal(data, &single); singleErr == nil {
			return []service.ServiceConfig{single}, nil
		}
		return nil, fmt.Errorf("failed to parse import file: %w", err)
	}

	return configs, nil
}
