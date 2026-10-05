package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"nssm-plus/internal/dpapi"
	"nssm-plus/internal/service"
	"nssm-plus/internal/wrapper"
)

// ConfigFile represents a multi-service configuration file
type ConfigFile struct {
	Services []service.ServiceConfig `json:"services"`
}

// ServiceFile represents one single-service config file with its on-disk
// path. Error is set when the file could not be fully loaded (broken JSON,
// missing serviceName, mismatched or duplicated service name); Config may
// still carry whatever could be parsed for diagnostics.
type ServiceFile struct {
	FilePath string                `json:"filePath"`
	Config   service.ServiceConfig `json:"config"`
	Error    string                `json:"error,omitempty"`
}

// Manager handles configuration file operations
type Manager struct{}

// NewManager creates a new config manager
func NewManager() *Manager {
	return &Manager{}
}

// SaveToFile saves multiple service configurations to a JSON file.
// Passwords are encrypted using Windows DPAPI before saving.
// The input configs slice is not modified; a copy is used internally.
func (m *Manager) SaveToFile(filePath string, configs []service.ServiceConfig) error {
	log.Printf("[config] SaveToFile: path=%q, serviceCount=%d", filePath, len(configs))

	configsCopy := make([]service.ServiceConfig, len(configs))
	copy(configsCopy, configs)

	for i := range configsCopy {
		if configsCopy[i].Password != "" && !dpapi.IsEncrypted(configsCopy[i].Password) {
			encrypted, err := dpapi.Encrypt(configsCopy[i].Password)
			if err != nil {
				log.Printf("[config] SaveToFile: failed to encrypt password for service %q: %v", configsCopy[i].ServiceName, err)
				return fmt.Errorf("failed to encrypt password for service '%s': %w", configsCopy[i].ServiceName, err)
			}
			configsCopy[i].Password = encrypted
		}
	}

	data, err := json.MarshalIndent(ConfigFile{Services: configsCopy}, "", "  ")
	if err != nil {
		log.Printf("[config] SaveToFile: failed to marshal config: %v", err)
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		log.Printf("[config] SaveToFile: failed to write file %q: %v", filePath, err)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Printf("[config] SaveToFile: saved %d service(s) to %q", len(configsCopy), filePath)
	return nil
}

// LoadFromFile loads multiple service configurations from a JSON file.
// Supports three formats for backward compatibility:
//  1. New format: {"services": [...]}
//  2. Bare array: [...]
//  3. Old single-service format: {...}
//
// For backward compat with old merged appPath format, if appPath contains
// a space and arguments is empty, the first token is kept as appPath
// and the rest is moved to arguments.
func (m *Manager) LoadFromFile(filePath string) ([]service.ServiceConfig, error) {
	log.Printf("[config] LoadFromFile: path=%q", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("[config] LoadFromFile: failed to read file %q: %v", filePath, err)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configs []service.ServiceConfig

	var cf ConfigFile
	if err := json.Unmarshal(data, &cf); err == nil && len(cf.Services) > 0 {
		configs = cf.Services
		log.Printf("[config] LoadFromFile: parsed as multi-service format, found %d service(s)", len(configs))
	} else {
		var arr []service.ServiceConfig
		if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
			configs = arr
			log.Printf("[config] LoadFromFile: parsed as bare array format, found %d service(s)", len(configs))
		} else {
			var single service.ServiceConfig
			if err := json.Unmarshal(data, &single); err == nil && single.ServiceName != "" {
				configs = []service.ServiceConfig{single}
				log.Printf("[config] LoadFromFile: parsed as single-service format, name=%q", single.ServiceName)
			} else {
				log.Printf("[config] LoadFromFile: unrecognized format in %q", filePath)
				return nil, fmt.Errorf("failed to parse config file: unrecognized format")
			}
		}
	}

	for i := range configs {
		normalizeConfig(&configs[i])
	}

	log.Printf("[config] LoadFromFile: loaded %d service(s) from %q", len(configs), filePath)
	return configs, nil
}

// normalizeConfig applies backward-compatible fixups to a loaded config:
// split a legacy merged appPath into appPath + arguments, and decrypt a
// DPAPI-encrypted password. The config is modified in place.
func normalizeConfig(cfg *service.ServiceConfig) {
	if cfg.Arguments == "" && cfg.AppPath != "" {
		if idx := strings.Index(cfg.AppPath, " "); idx >= 0 {
			log.Printf("[config] normalizeConfig: splitting merged appPath for service %q (space at index %d)", cfg.ServiceName, idx)
			cfg.Arguments = cfg.AppPath[idx+1:]
			cfg.AppPath = cfg.AppPath[:idx]
		}
	}

	if cfg.Password != "" && dpapi.IsEncrypted(cfg.Password) {
		decrypted, err := dpapi.Decrypt(cfg.Password)
		if err != nil {
			log.Printf("[config] normalizeConfig: failed to decrypt password for service %q: %v", cfg.ServiceName, err)
			fmt.Fprintf(os.Stderr, "Warning: failed to decrypt password for service '%s': %v\n", cfg.ServiceName, err)
		} else {
			cfg.Password = decrypted
		}
	}
}

// --- Single-service config files (one file per service) ---

// configDirName is the subdirectory of the NSSM-Plus data directory that
// holds the per-service config files.
const configDirName = "configs"

// maxServiceFileNameLen is the maximum service name length usable as a
// config file name (255-char NTFS filename limit minus the ".json" suffix).
const maxServiceFileNameLen = 250

// Dir returns the directory holding the per-service config files,
// %ProgramData%\NSSM-Plus\configs. It is created on demand.
func Dir() string {
	return filepath.Join(wrapper.ConfigDir(), configDirName)
}

// FilePath returns the single-service config file path for a service name.
func FilePath(serviceName string) string {
	return filepath.Join(Dir(), serviceName+".json")
}

// LoadDir scans dir for *.json files, loading each as a single-service
// config. A broken or conflicting file is reported via ServiceFile.Error
// and never prevents the other files from loading. A missing directory is
// treated as an empty configuration set.
func (m *Manager) LoadDir(dir string) ([]ServiceFile, error) {
	log.Printf("[config] LoadDir: dir=%q", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[config] LoadDir: dir does not exist yet, returning empty")
			return []ServiceFile{}, nil
		}
		log.Printf("[config] LoadDir: failed to read dir %q: %v", dir, err)
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	files := []ServiceFile{}
	seen := map[string]string{} // serviceName -> file path that claimed it
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		sf := ServiceFile{FilePath: path}

		cfg, err := parseServiceFile(path)
		if err != nil {
			log.Printf("[config] LoadDir: failed to parse %q: %v", path, err)
			sf.Error = err.Error()
			files = append(files, sf)
			continue
		}
		sf.Config = *cfg

		baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if !strings.EqualFold(baseName, cfg.ServiceName) {
			err := fmt.Errorf("file name %q does not match serviceName %q", entry.Name(), cfg.ServiceName)
			log.Printf("[config] LoadDir: %v", err)
			sf.Error = err.Error()
			files = append(files, sf)
			continue
		}
		if prev, dup := seen[cfg.ServiceName]; dup {
			err := fmt.Errorf("duplicate serviceName %q (already loaded from %s)", cfg.ServiceName, filepath.Base(prev))
			log.Printf("[config] LoadDir: %v", err)
			sf.Error = err.Error()
			files = append(files, sf)
			continue
		}

		seen[cfg.ServiceName] = path
		files = append(files, sf)
	}

	log.Printf("[config] LoadDir: loaded %d file(s) from %q", len(files), dir)
	return files, nil
}

// parseServiceFile parses one single-service config file. The file must
// contain a single service object with a non-empty serviceName; multi-service
// bundle files are rejected.
func parseServiceFile(filePath string) (*service.ServiceConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var cfg service.ServiceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("missing serviceName")
	}
	normalizeConfig(&cfg)
	return &cfg, nil
}

// SaveService saves a single service config to its own file in the default
// config directory. Passwords are encrypted with DPAPI before saving.
func (m *Manager) SaveService(cfg service.ServiceConfig) error {
	return m.SaveServiceTo(Dir(), cfg)
}

// SaveServiceTo saves a single service config as <serviceName>.json inside
// the given directory. Passwords are encrypted with DPAPI before saving.
func (m *Manager) SaveServiceTo(dir string, cfg service.ServiceConfig) error {
	if err := validateServiceFileName(cfg.ServiceName); err != nil {
		return err
	}
	log.Printf("[config] SaveServiceTo: dir=%q, service=%q", dir, cfg.ServiceName)

	cfgCopy := cfg
	if cfgCopy.Password != "" && !dpapi.IsEncrypted(cfgCopy.Password) {
		encrypted, err := dpapi.Encrypt(cfgCopy.Password)
		if err != nil {
			log.Printf("[config] SaveServiceTo: failed to encrypt password for service %q: %v", cfgCopy.ServiceName, err)
			return fmt.Errorf("failed to encrypt password for service '%s': %w", cfgCopy.ServiceName, err)
		}
		cfgCopy.Password = encrypted
	}

	data, err := json.MarshalIndent(cfgCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	path := filepath.Join(dir, cfgCopy.ServiceName+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[config] SaveServiceTo: failed to write file %q: %v", path, err)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Printf("[config] SaveServiceTo: saved service %q to %q", cfgCopy.ServiceName, path)
	return nil
}

// DeleteService removes the config file of a service from the default
// config directory. Deleting a non-existent file is not an error.
func (m *Manager) DeleteService(serviceName string) error {
	if err := validateServiceFileName(serviceName); err != nil {
		return err
	}
	err := os.Remove(FilePath(serviceName))
	if err != nil && !os.IsNotExist(err) {
		log.Printf("[config] DeleteService: failed to remove %q: %v", FilePath(serviceName), err)
		return fmt.Errorf("failed to delete config file: %w", err)
	}
	log.Printf("[config] DeleteService: removed %q (or not present)", FilePath(serviceName))
	return nil
}

// validateServiceFileName checks that a service name can be used as a
// config file name. The allowed character set matches the Windows service
// name rules enforced at install time.
func validateServiceFileName(name string) error {
	if name == "" {
		return fmt.Errorf("service name is required")
	}
	if len(name) > maxServiceFileNameLen {
		return fmt.Errorf("service name must be %d characters or less", maxServiceFileNameLen)
	}
	for _, ch := range name {
		isValid := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == ' ' || ch == '_' || ch == '-' || ch == '.'
		if !isValid {
			return fmt.Errorf("service name contains invalid character: '%c' (allowed: letters, digits, spaces, _-.)", ch)
		}
	}
	return nil
}
