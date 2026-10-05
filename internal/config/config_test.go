package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"nssm-plus/internal/dpapi"
	"nssm-plus/internal/service"
	"nssm-plus/internal/wrapper"
)

// withTempProgramData points the NSSM-Plus data directory at a temp dir for
// the duration of the test, so tests never touch the real %ProgramData%.
func withTempProgramData(t *testing.T) {
	t.Helper()
	old, had := os.LookupEnv("ProgramData")
	t.Setenv("ProgramData", t.TempDir())
	t.Cleanup(func() {
		if had {
			os.Setenv("ProgramData", old)
		} else {
			os.Unsetenv("ProgramData")
		}
	})
}

func sampleConfig(name string) service.ServiceConfig {
	return service.ServiceConfig{
		ServiceName:  name,
		DisplayName:  name + " Display",
		Description:  "test service",
		AppPath:      `C:\apps\` + name + `.exe`,
		Arguments:    "-run",
		WorkDir:      `C:\apps`,
		StartType:    "auto",
		Environment:  map[string]string{"FOO": "bar"},
		Dependencies: []string{"Tcpip"},
	}
}

func TestSaveServiceAndLoadDirRoundtrip(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	if err := m.SaveService(sampleConfig("SvcA")); err != nil {
		t.Fatalf("SaveService(SvcA) failed: %v", err)
	}
	if err := m.SaveService(sampleConfig("SvcB")); err != nil {
		t.Fatalf("SaveService(SvcB) failed: %v", err)
	}

	files, err := m.LoadDir(Dir())
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(files), files)
	}
	byName := map[string]ServiceFile{}
	for _, f := range files {
		if f.Error != "" {
			t.Errorf("unexpected error for %s: %s", f.FilePath, f.Error)
		}
		byName[f.Config.ServiceName] = f
	}
	a, ok := byName["SvcA"]
	if !ok {
		t.Fatalf("SvcA not loaded: %+v", files)
	}
	if a.Config.Arguments != "-run" || a.Config.Environment["FOO"] != "bar" {
		t.Errorf("SvcA config mismatch: %+v", a.Config)
	}
	if a.FilePath != FilePath("SvcA") {
		t.Errorf("FilePath mismatch: got %q, want %q", a.FilePath, FilePath("SvcA"))
	}
}

func TestLoadDirMissingDirIsEmpty(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	files, err := m.LoadDir(Dir())
	if err != nil {
		t.Fatalf("LoadDir on missing dir should not fail: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected empty result, got %d files", len(files))
	}
}

func TestLoadDirBrokenFileDoesNotAffectOthers(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	if err := m.SaveService(sampleConfig("Good")); err != nil {
		t.Fatalf("SaveService failed: %v", err)
	}
	broken := filepath.Join(Dir(), "Broken.json")
	if err := os.WriteFile(broken, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("failed to write broken file: %v", err)
	}

	files, err := m.LoadDir(Dir())
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	loaded := 0
	for _, f := range files {
		if f.Error == "" {
			loaded++
			if f.Config.ServiceName != "Good" {
				t.Errorf("unexpected loaded service %q", f.Config.ServiceName)
			}
		} else if filepath.Base(f.FilePath) != "Broken.json" {
			t.Errorf("error attributed to wrong file: %s (%s)", f.FilePath, f.Error)
		}
	}
	if loaded != 1 {
		t.Fatalf("expected 1 loaded file, got %d", loaded)
	}
}

func TestLoadDirDuplicateAndMismatch(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	if err := m.SaveService(sampleConfig("Real")); err != nil {
		t.Fatalf("SaveService failed: %v", err)
	}
	// Duplicate: another file claiming the same serviceName
	dupData, _ := json.Marshal(sampleConfig("Real"))
	if err := os.WriteFile(filepath.Join(Dir(), "CopyOfReal.json"), dupData, 0644); err != nil {
		t.Fatalf("failed to write duplicate file: %v", err)
	}
	// Mismatch: file name does not match the serviceName inside
	mismatchData, _ := json.Marshal(sampleConfig("InsideName"))
	if err := os.WriteFile(filepath.Join(Dir(), "OutsideName.json"), mismatchData, 0644); err != nil {
		t.Fatalf("failed to write mismatched file: %v", err)
	}

	files, err := m.LoadDir(Dir())
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	active := 0
	for _, f := range files {
		base := filepath.Base(f.FilePath)
		switch base {
		case "Real.json":
			if f.Error != "" {
				t.Errorf("Real.json should load cleanly, got error: %s", f.Error)
			} else {
				active++
			}
		case "CopyOfReal.json":
			if f.Error == "" {
				t.Errorf("CopyOfReal.json should be flagged as duplicate")
			}
		case "OutsideName.json":
			if f.Error == "" {
				t.Errorf("OutsideName.json should be flagged as mismatched")
			}
		}
	}
	if active != 1 {
		t.Fatalf("expected exactly 1 active config, got %d", active)
	}
}

func TestSaveServiceInvalidName(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	for _, name := range []string{"", "a/b", `c\d`, "svc:1", "svc*"} {
		if err := m.SaveService(sampleConfig(name)); err == nil {
			t.Errorf("SaveService with name %q should fail", name)
		}
	}
}

func TestSaveServiceEncryptsPasswordAndLoadDecrypts(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	cfg := sampleConfig("WithSecret")
	cfg.Password = "plain-secret"
	if err := m.SaveService(cfg); err != nil {
		t.Fatalf("SaveService failed: %v", err)
	}

	// On disk the password must be DPAPI-encrypted, not plaintext
	raw, err := os.ReadFile(FilePath("WithSecret"))
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	var onDisk service.ServiceConfig
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("failed to parse saved file: %v", err)
	}
	if onDisk.Password == "plain-secret" {
		t.Fatalf("password was saved in plaintext")
	}
	if !dpapi.IsEncrypted(onDisk.Password) {
		t.Fatalf("saved password is not DPAPI-encrypted: %q", onDisk.Password)
	}

	files, err := m.LoadDir(Dir())
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if len(files) != 1 || files[0].Error != "" {
		t.Fatalf("unexpected load result: %+v", files)
	}
	if files[0].Config.Password != "plain-secret" {
		t.Errorf("password roundtrip mismatch: got %q", files[0].Config.Password)
	}
}

func TestRenameRemovesOldConfigFile(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	old := sampleConfig("OldName")
	if err := m.SaveService(old); err != nil {
		t.Fatalf("SaveService(OldName) failed: %v", err)
	}
	// Simulate a rename: save under the new name, then delete the old file
	renamed := sampleConfig("NewName")
	renamed.AppPath = old.AppPath
	if err := m.SaveService(renamed); err != nil {
		t.Fatalf("SaveService(NewName) failed: %v", err)
	}
	if err := m.DeleteService("OldName"); err != nil {
		t.Fatalf("DeleteService(OldName) failed: %v", err)
	}

	files, err := m.LoadDir(Dir())
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if len(files) != 1 || files[0].Config.ServiceName != "NewName" {
		t.Fatalf("expected only NewName to remain, got %+v", files)
	}
}

func TestDeleteServiceMissingIsNoError(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	if err := m.DeleteService("NeverExisted"); err != nil {
		t.Fatalf("DeleteService of missing file should not fail: %v", err)
	}
}

func TestLoadFromFileLegacySingleObjectWithMergedAppPath(t *testing.T) {
	withTempProgramData(t)
	m := NewManager()

	legacy := `{"serviceName":"Legacy","appPath":"C:\\apps\\svc.exe -run -verbose"}`
	path := filepath.Join(t.TempDir(), "legacy.json")
	if err := os.WriteFile(path, []byte(legacy), 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	configs, err := m.LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].AppPath != `C:\apps\svc.exe` || configs[0].Arguments != "-run -verbose" {
		t.Errorf("merged appPath not split correctly: %+v", configs[0])
	}
}

func TestDirUsesWrapperBase(t *testing.T) {
	withTempProgramData(t)
	want := filepath.Join(wrapper.ConfigDir(), "configs")
	if Dir() != want {
		t.Errorf("Dir() = %q, want %q", Dir(), want)
	}
}
