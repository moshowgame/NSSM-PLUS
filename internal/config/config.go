package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nssm-plus/internal/service"
)

// ConfigFile represents a multi-service configuration file
type ConfigFile struct {
	Services []service.ServiceConfig `json:"services"`
}

// Manager handles configuration file operations
type Manager struct{}

// NewManager creates a new config manager
func NewManager() *Manager {
	return &Manager{}
}

// SaveToFile saves multiple service configurations to a JSON file
func (m *Manager) SaveToFile(filePath string, configs []service.ServiceConfig) error {
	data, err := json.MarshalIndent(ConfigFile{Services: configs}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadFromFile loads multiple service configurations from a JSON file.
// Supports three formats for backward compatibility:
//  1. New format: {"services": [...]}
//  2. Bare array: [...]
//  3. Old single-service format: {...}
// For backward compat with old merged appPath format, if appPath contains
// a space and arguments is empty, the first token is kept as appPath
// and the rest is moved to arguments.
func (m *Manager) LoadFromFile(filePath string) ([]service.ServiceConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configs []service.ServiceConfig

	// Try new multi-service format
	var cf ConfigFile
	if err := json.Unmarshal(data, &cf); err == nil && len(cf.Services) > 0 {
		configs = cf.Services
	} else {
		// Try bare array format
		var arr []service.ServiceConfig
		if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
			configs = arr
		} else {
			// Fallback: old single-service format
			var single service.ServiceConfig
			if err := json.Unmarshal(data, &single); err == nil && single.ServiceName != "" {
				configs = []service.ServiceConfig{single}
			} else {
				return nil, fmt.Errorf("failed to parse config file: unrecognized format")
			}
		}
	}

	// Backward compat: if old config had merged appPath (contains space) and no arguments,
	// split it — but only if appPath doesn't look like a simple unquoted exe name
	for i := range configs {
		if configs[i].Arguments == "" && configs[i].AppPath != "" {
			if idx := strings.Index(configs[i].AppPath, " "); idx >= 0 {
				configs[i].Arguments = configs[i].AppPath[idx+1:]
				configs[i].AppPath = configs[i].AppPath[:idx]
			}
		}
	}

	return configs, nil
}
