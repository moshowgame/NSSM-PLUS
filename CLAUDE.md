# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Development with hot-reload (requires admin terminal)
wails dev

# Production build → build/bin/nssm-plus.exe
wails build

# Frontend-only dev server (for UI work without Go backend)
cd frontend && npm install && npm run dev

# Manual Go build (after frontend is built)
go build -o nssm-plus.exe .
```

## Architecture

This is a **Windows-only** Wails v2.12 desktop app — a GUI replacement for NSSM. Go backend + Vue 3/Vite 5 frontend, communicating via Wails auto-generated bindings.

### Entry point (`main.go`)

Three modes, dispatched by CLI args:

| Mode | Trigger | Behavior |
|------|---------|----------|
| **GUI** | `nssm-plus.exe` (no args) | `wails.Run()` — WebView2 window |
| **Service wrapper** | `nssm-plus.exe service <Name>` | `wrapper.Run()` — `svc.Handler` that launches & monitors a child process |
| **CLI** | `nssm-plus.exe install/remove/start/stop/restart/status/list` | `cli.Run()` — direct SCM operations, no GUI |

### Self-hosted wrapper pattern (core design)

When a service is installed, SCM's `BinaryPathName` points to `nssm-plus.exe` itself with a `service <Name>` argument. When SCM starts the service, it runs `nssm-plus.exe service MySvc`, which enters the wrapper path in `main.go`. The wrapper reads `%ProgramData%\NSSM-Plus\services\<name>.json` to get the real app path/args, launches the child process, redirects stdout/stderr to `%ProgramData%\NSSM-Plus\logs\<name>.log`, and responds to SCM stop signals.

### Module dependency graph

```
main.go
  ├── app.go (Wails bindings; all public methods auto-exposed to frontend)
  │     ├── internal/service/manager.go   — SCM operations (Install/Remove/Start/Stop/Modify/List)
  │     │     └── internal/wrapper/       — persistence and splitArgs helper
  │     └── internal/config/config.go     — per-service config files in ProgramData\NSSM-Plus\configs + multi-service JSON import/export
  │           └── internal/dpapi/dpapi.go — Windows DPAPI password encrypt/decrypt
  ├── internal/cli/cli.go                 — CLI arg parsing
  └── internal/wrapper/wrapper.go         — svc.Handler, process lifecycle, graceful/force kill
```

**Import cycle avoidance**: `internal/service/manager.go` imports `internal/wrapper` for config persistence, but `wrapper` does NOT import `service`. Two helper functions (`IsWrapperBinaryPathCurrent`, `GetWrapperBinaryPathCurrent`) are duplicated in `manager.go` to avoid the cycle.

### NSSM-Plus service identification

Services are identified by a `[NSSM-Plus]` prefix in the Description field, set via `ChangeServiceConfig2`. `ListServices()` enumerates ALL system services and filters by this prefix. The real app path is stored in `ProgramData\NSSM-Plus\services\<name>.json`, not derived from the SCM BinaryPathName.

### Frontend

Single-component Vue 3 app (`frontend/src/App.vue`). Calls backend via `window.go.main.App.MethodName(...)`. Wails generates the JS bindings from `app.go`'s public methods at build time.

### Config file model

There are two distinct config stores under `%ProgramData%\NSSM-Plus\`:

- **`configs\<serviceName>.json`** — the authoritative per-service config files (one single `ServiceConfig` object per file). Scanned at startup via `Manager.LoadDir()`; written by `SaveService()` / removed by `DeleteService()`. `app.go` persists them automatically on Install/Modify, and deletes the old file on rename. Files that fail to parse, have a filename mismatching their `serviceName`, or duplicate another file's `serviceName` come back with `ServiceFile.Error` set and never block the other files.
- **`services\<serviceName>.json`** — the runtime wrapper config (`WrapperConfig`, app path/args/env/log/restart) read by `internal/wrapper` when SCM starts a service. Not user-facing.

`internal/config/config.go` also handles the legacy multi-service JSON bundle formats for import/export (`LoadFromFile` / `SaveToFile`):
1. `{"services": [...]}` — multi-service bundle
2. `[...]` — bare array
3. `{...}` — legacy single-service object

Passwords are encrypted with Windows DPAPI when saved to file, decrypted when loaded.

### Frontend

Single-component Vue 3 app (`frontend/src/App.vue`). Calls backend via `window.go.main.App.MethodName(...)`. Wails generates the JS bindings from `app.go`'s public methods at build time. The frontend state is directory-based: `serviceFiles` (raw `LoadConfigDir` result) → `loadedServices` (valid configs) and `configErrors` (broken files, shown as a ⚠ badge in the header); **Import Config** splits a legacy bundle into per-service files, **Export Config** aggregates them back.

## Key constraints

- **Windows-only**, requires WebView2 Runtime and admin privileges for SCM operations
- `app.go` public methods must accept/return JSON-serializable types only
- Wails binding names are PascalCase and match Go method names exactly
- Go module is `nssm-plus` (no external import path)
- `taskkill /T /PID` is used for process tree termination (5s graceful, then `/F` forced)
