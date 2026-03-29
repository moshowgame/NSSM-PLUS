package wrapper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	baseDirName = "NSSM-Plus"
	servicesDir = "services"
	logsDirName = "logs"
)

// ConfigDir returns the base directory for NSSM-Plus data.
func ConfigDir() string {
	// Use ProgramData for system-wide storage
	base := os.Getenv("ProgramData")
	if base == "" {
		base = filepath.Join(os.Getenv("SystemDrive"), "ProgramData")
	}
	return filepath.Join(base, baseDirName)
}

// ServicesDir returns the directory containing service config files.
func ServicesDir() string {
	return filepath.Join(ConfigDir(), servicesDir)
}

// LogsDir returns the directory for service log files.
func LogsDir() string {
	return filepath.Join(ConfigDir(), logsDirName)
}

// ConfigPath returns the config file path for a given service name.
func ConfigPath(serviceName string) string {
	return filepath.Join(ServicesDir(), serviceName+".json")
}

// LogPath returns the log file path for a given service name.
func LogPath(serviceName string) string {
	return filepath.Join(LogsDir(), serviceName+".log")
}

// WrapperConfig holds the application launch configuration for a service.
type WrapperConfig struct {
	AppPath   string            `json:"appPath"`
	Arguments string            `json:"arguments"`
	WorkDir   string            `json:"workDir"`
	Env       map[string]string `json:"env,omitempty"`
}

// SaveConfig writes the wrapper config to disk.
func SaveConfig(serviceName string, cfg *WrapperConfig) error {
	if err := os.MkdirAll(ServicesDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(serviceName), data, 0644)
}

// LoadConfig reads the wrapper config from disk.
func LoadConfig(serviceName string) (*WrapperConfig, error) {
	data, err := os.ReadFile(ConfigPath(serviceName))
	if err != nil {
		return nil, err
	}
	var cfg WrapperConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ConfigExists checks if a wrapper config file exists for the service.
func ConfigExists(serviceName string) bool {
	_, err := os.Stat(ConfigPath(serviceName))
	return err == nil
}

// DeleteConfig removes the wrapper config file.
func DeleteConfig(serviceName string) error {
	return os.Remove(ConfigPath(serviceName))
}

// splitArgs splits a command-line argument string into individual arguments.
// It handles simple whitespace splitting. For quoted arguments, a more
// sophisticated parser would be needed, but this covers common cases.
func splitArgs(argStr string) []string {
	argStr = strings.TrimSpace(argStr)
	if argStr == "" {
		return nil
	}
	return strings.Fields(argStr)
}
