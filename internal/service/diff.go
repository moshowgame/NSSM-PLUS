package service

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"

	"nssm-plus/internal/wrapper"
)

// SyncState describes how a file-based service config relates to the
// actually registered service.
type SyncState string

const (
	SyncStateSynced       SyncState = "synced"
	SyncStateDrifted      SyncState = "drifted"
	SyncStateNotInstalled SyncState = "notInstalled"
	SyncStateOrphaned     SyncState = "orphaned"
)

// FieldDiff describes one field that differs between the file config and
// the actually registered service.
type FieldDiff struct {
	Field       string `json:"field"`
	FileValue   string `json:"fileValue"`
	ActualValue string `json:"actualValue"`
}

// ServiceDiff is the reconciliation result for one service.
type ServiceDiff struct {
	ServiceName string      `json:"serviceName"`
	State       SyncState   `json:"state"`
	Fields      []FieldDiff `json:"fields,omitempty"`
}

// DiffServiceConfig compares a file-based config against the actually
// registered service config and returns the list of differing fields.
// Values are normalized before comparison (path case/slashes, argument
// whitespace, start type synonyms, unordered env maps and dependencies),
// so only real drift is reported. Password is skipped because SCM never
// returns it.
func DiffServiceConfig(fileCfg, actual ServiceConfig) []FieldDiff {
	var diffs []FieldDiff
	add := func(field, fileVal, actualVal string) {
		diffs = append(diffs, FieldDiff{Field: field, FileValue: fileVal, ActualValue: actualVal})
	}

	if normDisplayName(fileCfg.DisplayName, fileCfg.ServiceName) != normDisplayName(actual.DisplayName, actual.ServiceName) {
		add("displayName", strings.TrimSpace(fileCfg.DisplayName), strings.TrimSpace(actual.DisplayName))
	}
	if !descEq(fileCfg, actual) {
		add("description", strings.TrimSpace(fileCfg.Description), strings.TrimSpace(actual.Description))
	}
	if !pathEq(fileCfg.AppPath, actual.AppPath) {
		add("appPath", fileCfg.AppPath, actual.AppPath)
	}
	if normArgs(fileCfg.Arguments) != normArgs(actual.Arguments) {
		add("arguments", fileCfg.Arguments, actual.Arguments)
	}
	if !pathEq(fileCfg.WorkDir, actual.WorkDir) {
		add("workDir", fileCfg.WorkDir, actual.WorkDir)
	}
	if normStartType(fileCfg.StartType) != normStartType(actual.StartType) {
		add("startType", fileCfg.StartType, actual.StartType)
	}
	if normAccount(fileCfg.Account) != normAccount(actual.Account) {
		add("account", strings.TrimSpace(fileCfg.Account), strings.TrimSpace(actual.Account))
	}
	if !envEq(fileCfg.Environment, actual.Environment) {
		add("environment", envString(fileCfg.Environment), envString(actual.Environment))
	}
	if !depsEq(fileCfg.Dependencies, actual.Dependencies) {
		add("dependencies", depsString(fileCfg.Dependencies), depsString(actual.Dependencies))
	}
	if !pathEq(fileCfg.LogStdout, actual.LogStdout) {
		add("logStdout", fileCfg.LogStdout, actual.LogStdout)
	}
	if !pathEq(fileCfg.LogStderr, actual.LogStderr) {
		add("logStderr", fileCfg.LogStderr, actual.LogStderr)
	}
	if fileCfg.RotateLog != actual.RotateLog {
		add("rotateLog", strconv.FormatBool(fileCfg.RotateLog), strconv.FormatBool(actual.RotateLog))
	}
	if fileCfg.RestartDelay != actual.RestartDelay {
		add("restartDelay", strconv.Itoa(fileCfg.RestartDelay), strconv.Itoa(actual.RestartDelay))
	}
	if fileCfg.RestartTimeout != actual.RestartTimeout {
		add("restartTimeout", strconv.Itoa(fileCfg.RestartTimeout), strconv.Itoa(actual.RestartTimeout))
	}

	return diffs
}

// GetSyncStates reconciles file-based configs against the actually
// registered services and returns a per-service sync state with
// field-level diffs.
func (m *Manager) GetSyncStates(fileConfigs []ServiceConfig) ([]ServiceDiff, error) {
	if len(fileConfigs) == 0 {
		return nil, nil
	}
	log.Printf("[service] GetSyncStates: %d file config(s)", len(fileConfigs))

	scMgr, err := connectSCM()
	if err != nil {
		return nil, err
	}
	defer scMgr.Disconnect()

	results := make([]ServiceDiff, 0, len(fileConfigs))
	for _, fc := range fileConfigs {
		if fc.ServiceName == "" {
			continue
		}
		result := ServiceDiff{ServiceName: fc.ServiceName}

		s, err := scMgr.OpenService(fc.ServiceName)
		if err != nil {
			if serviceNotExist(err) {
				result.State = SyncStateNotInstalled
			} else {
				result.State = SyncStateDrifted
				result.Fields = []FieldDiff{{Field: "error", ActualValue: err.Error()}}
			}
			results = append(results, result)
			continue
		}

		actual, err := buildServiceConfig(s, fc.ServiceName)
		s.Close()
		if err != nil {
			result.State = SyncStateDrifted
			result.Fields = []FieldDiff{{Field: "error", ActualValue: err.Error()}}
			results = append(results, result)
			continue
		}

		result.Fields = DiffServiceConfig(fc, *actual)
		if len(result.Fields) == 0 {
			result.State = SyncStateSynced
		} else {
			result.State = SyncStateDrifted
		}
		results = append(results, result)
	}

	log.Printf("[service] GetSyncStates: %d result(s)", len(results))
	return results, nil
}

// GetOrphanConfigs lists wrapper config files in the services directory
// that have no matching registered service (e.g. removed via sc delete).
func (m *Manager) GetOrphanConfigs() ([]ServiceInfo, error) {
	entries, err := os.ReadDir(wrapper.ServicesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read services directory: %w", err)
	}

	var candidates []string
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			continue
		}
		candidates = append(candidates, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	scMgr, err := connectSCM()
	if err != nil {
		return nil, err
	}
	defer scMgr.Disconnect()

	var result []ServiceInfo
	for _, name := range candidates {
		s, err := scMgr.OpenService(name)
		if err == nil {
			s.Close()
			continue
		}
		if !serviceNotExist(err) {
			continue
		}
		info := ServiceInfo{Name: name, DisplayName: name, Status: "Orphan"}
		if wCfg, err := wrapper.LoadConfig(name); err == nil {
			info.AppPath = wCfg.AppPath
		}
		result = append(result, info)
	}

	log.Printf("[service] GetOrphanConfigs: found %d orphan config(s)", len(result))
	return result, nil
}

// GetWrapperConfig loads the wrapper config of a service directly from the
// services directory, without requiring the service to be registered.
func (m *Manager) GetWrapperConfig(serviceName string) (*ServiceConfig, error) {
	log.Printf("[service] GetWrapperConfig: name=%q", serviceName)

	wCfg, err := wrapper.LoadConfig(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load wrapper config for '%s': %w", serviceName, err)
	}
	return &ServiceConfig{
		ServiceName:    serviceName,
		DisplayName:    serviceName,
		StartType:      "auto",
		AppPath:        wCfg.AppPath,
		Arguments:      wCfg.Arguments,
		WorkDir:        wCfg.WorkDir,
		Environment:    wCfg.Env,
		LogStdout:      wCfg.LogStdout,
		LogStderr:      wCfg.LogStderr,
		RotateLog:      wCfg.RotateLog,
		RestartDelay:   wCfg.RestartDelay,
		RestartTimeout: wCfg.RestartTimeout,
	}, nil
}

// --- normalization helpers ---

func serviceNotExist(err error) bool {
	return errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) ||
		strings.Contains(err.Error(), "does not exist")
}

// normPath normalizes a filesystem path for comparison: trims spaces and
// surrounding quotes, unifies slash direction, lowercases (Windows paths
// are case-insensitive).
func normPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"`)
	p = strings.ReplaceAll(p, `/`, `\`)
	return strings.ToLower(p)
}

func pathEq(a, b string) bool {
	return normPath(a) == normPath(b)
}

// normArgs collapses all whitespace runs so argument strings that differ
// only in spacing compare equal.
func normArgs(a string) string {
	return strings.Join(strings.Fields(a), " ")
}

func normStartType(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto", "automatic":
		return "auto"
	case "demand", "manual":
		return "manual"
	case "disabled":
		return "disabled"
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

// normAccount maps an empty account to LocalSystem, which is what SCM
// reports for services installed without an explicit account.
func normAccount(a string) string {
	a = strings.ToLower(strings.TrimSpace(a))
	if a == "" {
		return "localsystem"
	}
	return a
}

// normDisplayName falls back to the service name because SCM uses it as
// display name when none was set at install time.
func normDisplayName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.TrimSpace(fallback)
	}
	return strings.ToLower(name)
}

// descEq compares descriptions. When the file description is empty,
// install/modify historically wrote the app path into the SCM description,
// so an actual description matching the installed app path still counts
// as equal.
func descEq(fileCfg, actual ServiceConfig) bool {
	fd := strings.TrimSpace(fileCfg.Description)
	ad := strings.TrimSpace(actual.Description)
	if strings.EqualFold(fd, ad) {
		return true
	}
	if fd == "" {
		return ad == "" || pathEq(ad, actual.AppPath)
	}
	return false
}

func envEq(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func envString(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+env[k])
	}
	return strings.Join(parts, "; ")
}

func normDeps(deps []string) []string {
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			out = append(out, d)
		}
	}
	sort.Strings(out)
	return out
}

func depsEq(a, b []string) bool {
	an := normDeps(a)
	bn := normDeps(b)
	if len(an) != len(bn) {
		return false
	}
	for i := range an {
		if an[i] != bn[i] {
			return false
		}
	}
	return true
}

func depsString(deps []string) string {
	var out []string
	for _, d := range deps {
		if strings.TrimSpace(d) != "" {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ", ")
}
