# Kiront Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a macOS menu bar app in Go that toggles Claude Code between Anthropic API mode and Kiro gateway mode with one click.

**Architecture:** Menu bar app with five managers (Config, Settings, Gateway, State, Process) coordinated by a main controller. Gateway Python code bundled in `gateway/` directory. All file operations atomic. Process lifecycle tracked via PID file.

**Tech Stack:** Go 1.21+, `github.com/getlantern/systray` for menu bar, standard library for process management and JSON handling.

**Spec:** `/Users/yuri/Projects/kiront/2026-10-17-kiront-design.md`

## Global Constraints

- Gateway bundled at `gateway/` directory relative to executable
- Claude Code settings at `~/.claude/settings.json` - must use atomic writes (temp file + rename)
- App config at `~/.kiront/config.json`
- Settings backup at `~/.kiront/settings.backup.json`
- Gateway runs on port 8000
- PROXY_API_KEY must match ANTHROPIC_API_KEY
- Python 3.x required, invoked as `python3`
- Health check endpoint: `GET http://127.0.0.1:8000/health` returns `{"status": "healthy"}`

## Review Focus

1. **Concurrent mode switches:** If user clicks toggle twice rapidly, should not corrupt state or leave orphaned processes. One toggle at a time via mutex.
2. **Crash recovery:** App crashes in Kiro mode leaves gateway running and settings.json modified. On restart, detect orphaned gateway (port check), re-acquire PID or kill by port, restore consistent state.
3. **Malformed settings.json:** User or other tool corrupts settings.json. Atomic write should prevent partial writes; read should validate JSON before proceeding.
4. **Missing backup:** User manually deletes settings.backup.json while in Kiro mode. Toggle back should fail gracefully with clear error, not destroy settings.json.
5. **Gateway startup timeout:** Gateway takes >30s to start or never returns healthy status. Should timeout, show error, clean up process, revert mode.

---

## File Structure

```
kiront/
├── main.go                    # Entry point, menu bar setup
├── internal/
│   ├── config/
│   │   └── config.go          # Config manager (config.json)
│   ├── settings/
│   │   └── settings.go        # Claude Code settings manager (settings.json backup/restore)
│   ├── gateway/
│   │   └── gateway.go         # Gateway lifecycle manager (venv, start/stop)
│   ├── state/
│   │   └── state.go           # State manager (mode tracking)
│   ├── process/
│   │   └── process.go         # Process detection (Claude Code processes)
│   └── controller/
│       └── controller.go      # Main controller (coordinates all managers)
├── gateway/                   # Bundled Python gateway code
│   ├── main.py
│   ├── requirements.txt
│   └── ... (all gateway files)
└── docs/superpowers/plans/
    └── 2026-10-17-kiront-implementation.md
```

---

### Task 1: Project Setup and Config Manager

**Files:**
- Create: `go.mod`, `go.sum`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Interfaces:**
- Consumes: none
- Produces: `config.Load() (*Config, error)`, `config.Save(*Config) error`, `Config` struct with fields: `KiroAPIKey string`, `GatewayPort int`, `KiroCLIDBPath string`, `CurrentMode string`, `AutoStart bool`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/yuri/Projects/kiront
go mod init github.com/sparkreel/kiront
go get github.com/getlantern/systray@latest
```

Expected: `go.mod` and `go.sum` created with systray dependency

- [ ] **Step 2: Write failing test for config loading**

Create `internal/config/config_test.go`:
```go
func TestLoadConfig_NewFile(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "config.json")
    
    cfg, err := LoadFrom(configPath)
    
    assert.NoError(t, err)
    assert.Equal(t, "anthropic", cfg.CurrentMode)
    assert.Equal(t, 8000, cfg.GatewayPort)
    assert.True(t, cfg.AutoStart)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/config -v`
Expected: FAIL - "LoadFrom" undefined

- [ ] **Step 4: Implement Config struct and LoadFrom in `internal/config/config.go`**

```go
type Config struct {
    KiroAPIKey     string `json:"kiro_api_key"`
    GatewayPort    int    `json:"gateway_port"`
    KiroCLIDBPath  string `json:"kiro_cli_db_path"`
    CurrentMode    string `json:"current_mode"`
    AutoStart      bool   `json:"auto_start"`
}

func LoadFrom(path string) (*Config, error)
```

Implementation: If file doesn't exist, return default config with `CurrentMode: "anthropic"`, `GatewayPort: 8000`, `AutoStart: true`, `KiroCLIDBPath: "/Users/yuri/Library/Application Support/kiro-cli/data.sqlite3"`

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config -v`
Expected: PASS

- [ ] **Step 6: Write failing test for config saving**

Add to `internal/config/config_test.go`:
```go
func TestSaveConfig(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "config.json")
    
    cfg := &Config{
        KiroAPIKey: "test-key",
        CurrentMode: "kiro",
        GatewayPort: 8000,
        AutoStart: true,
    }
    
    err := SaveTo(configPath, cfg)
    assert.NoError(t, err)
    
    // Verify file contents
    loaded, err := LoadFrom(configPath)
    assert.NoError(t, err)
    assert.Equal(t, "test-key", loaded.KiroAPIKey)
    assert.Equal(t, "kiro", loaded.CurrentMode)
}
```

- [ ] **Step 7: Run test to verify it fails**

Run: `go test ./internal/config -v`
Expected: FAIL - "SaveTo" undefined

- [ ] **Step 8: Implement SaveTo with atomic write**

```go
func SaveTo(path string, cfg *Config) error
```

Implementation: Marshal to JSON, write to temp file in same directory, rename to target path atomically. Create parent directory if needed.

- [ ] **Step 9: Run test to verify it passes**

Run: `go test ./internal/config -v`
Expected: PASS

- [ ] **Step 10: Add convenience functions Load() and Save()**

```go
func Load() (*Config, error) {
    return LoadFrom(filepath.Join(os.Getenv("HOME"), ".kiront", "config.json"))
}

func Save(cfg *Config) error {
    return SaveTo(filepath.Join(os.Getenv("HOME"), ".kiront", "config.json"), cfg)
}
```

- [ ] **Step 11: Commit**

```bash
git add go.mod go.sum internal/config/
git commit -m "feat: add config manager with atomic save"
```

---

### Task 2: Claude Code Settings Manager

**Files:**
- Create: `internal/settings/settings.go`
- Create: `internal/settings/settings_test.go`

**Interfaces:**
- Consumes: none
- Produces: `settings.Backup() error`, `settings.Restore() error`, `settings.ApplyKiroMode(apiKey string) error`, `settings.HasBackup() bool`

- [ ] **Step 1: Write failing test for settings backup**

Create `internal/settings/settings_test.go`:
```go
func TestBackup(t *testing.T) {
    dir := t.TempDir()
    settingsPath := filepath.Join(dir, "settings.json")
    backupPath := filepath.Join(dir, "settings.backup.json")
    
    // Create fake settings.json
    original := map[string]interface{}{"test": "value"}
    writeJSON(t, settingsPath, original)
    
    mgr := NewManager(settingsPath, backupPath)
    err := mgr.Backup()
    
    assert.NoError(t, err)
    assert.FileExists(t, backupPath)
    
    // Verify backup contents match original
    backup := readJSON(t, backupPath)
    assert.Equal(t, original, backup)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/settings -v`
Expected: FAIL - "NewManager" undefined

- [ ] **Step 3: Implement Manager struct and Backup in `internal/settings/settings.go`**

```go
type Manager struct {
    settingsPath string
    backupPath   string
}

func NewManager(settingsPath, backupPath string) *Manager
func (m *Manager) Backup() error
func (m *Manager) HasBackup() bool
```

Implementation: Copy settings.json to backup.json atomically

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/settings -v`
Expected: PASS

- [ ] **Step 5: Write failing test for ApplyKiroMode**

Add to `internal/settings/settings_test.go`:
```go
func TestApplyKiroMode(t *testing.T) {
    dir := t.TempDir()
    settingsPath := filepath.Join(dir, "settings.json")
    backupPath := filepath.Join(dir, "settings.backup.json")
    
    // Create fake settings.json
    original := map[string]interface{}{
        "hooks": map[string]interface{}{},
        "tui": "fullscreen",
    }
    writeJSON(t, settingsPath, original)
    
    mgr := NewManager(settingsPath, backupPath)
    err := mgr.ApplyKiroMode("test-api-key")
    
    assert.NoError(t, err)
    
    // Verify env vars added
    modified := readJSON(t, settingsPath)
    env := modified["env"].(map[string]interface{})
    assert.Equal(t, "http://127.0.0.1:8000", env["ANTHROPIC_BASE_URL"])
    assert.Equal(t, "test-api-key", env["ANTHROPIC_API_KEY"])
    assert.Equal(t, "claude-sonnet-4-5", env["ANTHROPIC_MODEL"])
    assert.Equal(t, "claude-haiku-4-5", env["ANTHROPIC_SMALL_FAST_MODEL"])
    
    // Verify other settings preserved
    assert.Equal(t, "fullscreen", modified["tui"])
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/settings -v`
Expected: FAIL - "ApplyKiroMode" undefined

- [ ] **Step 7: Implement ApplyKiroMode with atomic write**

```go
func (m *Manager) ApplyKiroMode(apiKey string) error
```

Implementation: Read settings.json, merge/replace `env` key with Kiro variables, write atomically

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/settings -v`
Expected: PASS

- [ ] **Step 9: Write failing test for Restore**

Add to `internal/settings/settings_test.go`:
```go
func TestRestore(t *testing.T) {
    dir := t.TempDir()
    settingsPath := filepath.Join(dir, "settings.json")
    backupPath := filepath.Join(dir, "settings.backup.json")
    
    // Create backup
    original := map[string]interface{}{"test": "original"}
    writeJSON(t, backupPath, original)
    
    // Create modified settings
    modified := map[string]interface{}{"test": "modified"}
    writeJSON(t, settingsPath, modified)
    
    mgr := NewManager(settingsPath, backupPath)
    err := mgr.Restore()
    
    assert.NoError(t, err)
    
    // Verify settings restored
    restored := readJSON(t, settingsPath)
    assert.Equal(t, "original", restored["test"])
}

func TestRestore_NoBackup(t *testing.T) {
    dir := t.TempDir()
    settingsPath := filepath.Join(dir, "settings.json")
    backupPath := filepath.Join(dir, "settings.backup.json")
    
    mgr := NewManager(settingsPath, backupPath)
    err := mgr.Restore()
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "backup not found")
}
```

- [ ] **Step 10: Run test to verify it fails**

Run: `go test ./internal/settings -v`
Expected: FAIL - "Restore" undefined

- [ ] **Step 11: Implement Restore**

```go
func (m *Manager) Restore() error
```

Implementation: Return error if backup doesn't exist, otherwise copy backup to settings atomically

- [ ] **Step 12: Run test to verify it passes**

Run: `go test ./internal/settings -v`
Expected: PASS

- [ ] **Step 13: Add test for malformed JSON handling**

Add to `internal/settings/settings_test.go`:
```go
func TestApplyKiroMode_MalformedJSON(t *testing.T) {
    dir := t.TempDir()
    settingsPath := filepath.Join(dir, "settings.json")
    backupPath := filepath.Join(dir, "settings.backup.json")
    
    // Write malformed JSON
    os.WriteFile(settingsPath, []byte("{invalid json"), 0644)
    
    mgr := NewManager(settingsPath, backupPath)
    err := mgr.ApplyKiroMode("test-key")
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid")
}
```

Run test, verify it passes (implementation should already handle this).

- [ ] **Step 14: Add convenience New() function**

```go
func New() *Manager {
    home := os.Getenv("HOME")
    return NewManager(
        filepath.Join(home, ".claude", "settings.json"),
        filepath.Join(home, ".kiront", "settings.backup.json"),
    )
}
```

- [ ] **Step 15: Commit**

```bash
git add internal/settings/
git commit -m "feat: add settings manager with backup/restore"
```

---

### Task 3: Gateway Manager

**Files:**
- Create: `internal/gateway/gateway.go`
- Create: `internal/gateway/gateway_test.go`

**Interfaces:**
- Consumes: none
- Produces: `gateway.NewManager(gatewayDir string, port int) *Manager`, `Manager.Setup(apiKey, cliDBPath string) error`, `Manager.Start() error`, `Manager.Stop() error`, `Manager.IsRunning() bool`

- [ ] **Step 1: Write failing test for gateway setup**

Create `internal/gateway/gateway_test.go`:
```go
func TestSetup(t *testing.T) {
    dir := t.TempDir()
    gatewayDir := filepath.Join(dir, "gateway")
    os.MkdirAll(gatewayDir, 0755)
    
    // Create fake requirements.txt
    os.WriteFile(filepath.Join(gatewayDir, "requirements.txt"), []byte("fastapi==0.100.0"), 0644)
    
    mgr := NewManager(gatewayDir, 8000)
    err := mgr.Setup("test-api-key", "/path/to/db.sqlite3")
    
    assert.NoError(t, err)
    assert.DirExists(t, filepath.Join(gatewayDir, ".venv"))
    assert.FileExists(t, filepath.Join(gatewayDir, ".env"))
    
    // Verify .env contents
    envContent, _ := os.ReadFile(filepath.Join(gatewayDir, ".env"))
    assert.Contains(t, string(envContent), "PROXY_API_KEY=test-api-key")
    assert.Contains(t, string(envContent), "ACCOUNT_SYSTEM=true")
    assert.Contains(t, string(envContent), "KIRO_CLI_DB_FILE=/path/to/db.sqlite3")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/gateway -v`
Expected: FAIL - "NewManager" undefined

- [ ] **Step 3: Implement Manager struct and Setup in `internal/gateway/gateway.go`**

```go
type Manager struct {
    gatewayDir string
    port       int
    cmd        *exec.Cmd
    pidFile    string
}

func NewManager(gatewayDir string, port int) *Manager
func (m *Manager) Setup(apiKey, cliDBPath string) error
```

Implementation: 
1. Check if .venv exists, if not run `python3 -m venv .venv`
2. Run `.venv/bin/pip install -r requirements.txt` (skip if already installed - check for site-packages)
3. Write .env file with three variables

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/gateway -v`
Expected: PASS (may be slow due to venv creation, consider skipping venv in test or mocking)

- [ ] **Step 5: Write failing test for Start**

Add to `internal/gateway/gateway_test.go`:
```go
func TestStart(t *testing.T) {
    // This test requires actual gateway code, so we'll mock it
    dir := t.TempDir()
    gatewayDir := filepath.Join(dir, "gateway")
    os.MkdirAll(gatewayDir, 0755)
    
    // Create fake Python script that responds to health checks
    mainPy := `
import http.server
import json

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "healthy"}).encode())

server = http.server.HTTPServer(('127.0.0.1', 8000), Handler)
server.serve_forever()
`
    os.WriteFile(filepath.Join(gatewayDir, "main.py"), []byte(mainPy), 0644)
    
    mgr := NewManager(gatewayDir, 8000)
    err := mgr.Start()
    
    assert.NoError(t, err)
    assert.True(t, mgr.IsRunning())
    
    // Cleanup
    mgr.Stop()
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/gateway -v`
Expected: FAIL - "Start" undefined

- [ ] **Step 7: Implement Start with health check**

```go
func (m *Manager) Start() error
func (m *Manager) IsRunning() bool
```

Implementation:
1. Start process: `cd gateway && .venv/bin/python main.py`
2. TCP dial port 8000 until successful (timeout 10s)
3. Poll GET http://127.0.0.1:8000/health until returns `{"status": "healthy"}` (timeout 20s)
4. Store PID in pidFile (`~/.kiront/gateway.pid`)
5. Return error if any step fails or times out

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/gateway -v -timeout 30s`
Expected: PASS

- [ ] **Step 9: Write failing test for Stop**

Add to `internal/gateway/gateway_test.go`:
```go
func TestStop(t *testing.T) {
    dir := t.TempDir()
    gatewayDir := filepath.Join(dir, "gateway")
    os.MkdirAll(gatewayDir, 0755)
    
    // Create and start fake gateway
    mainPy := `// ... same as above`
    os.WriteFile(filepath.Join(gatewayDir, "main.py"), []byte(mainPy), 0644)
    
    mgr := NewManager(gatewayDir, 8000)
    mgr.Start()
    assert.True(t, mgr.IsRunning())
    
    err := mgr.Stop()
    
    assert.NoError(t, err)
    assert.False(t, mgr.IsRunning())
}
```

- [ ] **Step 10: Run test to verify it fails**

Run: `go test ./internal/gateway -v`
Expected: FAIL - "Stop" undefined

- [ ] **Step 11: Implement Stop with graceful shutdown**

```go
func (m *Manager) Stop() error
```

Implementation:
1. Read PID from pidFile
2. Send SIGTERM
3. Wait up to 5 seconds for process to exit
4. If still running, send SIGKILL
5. Remove pidFile
6. Verify port 8000 no longer listening

- [ ] **Step 12: Run test to verify it passes**

Run: `go test ./internal/gateway -v`
Expected: PASS

- [ ] **Step 13: Add test for timeout during Start**

Add to `internal/gateway/gateway_test.go`:
```go
func TestStart_Timeout(t *testing.T) {
    dir := t.TempDir()
    gatewayDir := filepath.Join(dir, "gateway")
    os.MkdirAll(gatewayDir, 0755)
    
    // Create Python script that never responds to health checks
    mainPy := `import time; time.sleep(1000)`
    os.WriteFile(filepath.Join(gatewayDir, "main.py"), []byte(mainPy), 0644)
    
    mgr := NewManager(gatewayDir, 8000)
    err := mgr.Start()
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "timeout")
    
    // Ensure process is cleaned up
    mgr.Stop()
}
```

Run test, verify it passes (implementation should already timeout).

- [ ] **Step 14: Commit**

```bash
git add internal/gateway/
git commit -m "feat: add gateway manager with lifecycle control"
```

---

### Task 4: Process Detector

**Files:**
- Create: `internal/process/process.go`
- Create: `internal/process/process_test.go`

**Interfaces:**
- Consumes: none
- Produces: `process.IsClaudeCodeRunning() bool`

- [ ] **Step 1: Write failing test for Claude Code detection**

Create `internal/process/process_test.go`:
```go
func TestIsClaudeCodeRunning_NotRunning(t *testing.T) {
    // This will pass if no Claude Code actually running
    running := IsClaudeCodeRunning()
    // Can't assert specific value, just check it doesn't panic
    _ = running
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/process -v`
Expected: FAIL - "IsClaudeCodeRunning" undefined

- [ ] **Step 3: Implement IsClaudeCodeRunning in `internal/process/process.go`**

```go
func IsClaudeCodeRunning() bool
```

Implementation: Run `pgrep -f "claude"`, return true if exit code 0 (process found), false otherwise

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/process -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/process/
git commit -m "feat: add Claude Code process detection"
```

---

### Task 5: State Manager and Controller

**Files:**
- Create: `internal/state/state.go`
- Create: `internal/controller/controller.go`
- Create: `internal/controller/controller_test.go`

**Interfaces:**
- Consumes: `config.Config`, `settings.Manager`, `gateway.Manager`, `process.IsClaudeCodeRunning()`
- Produces: `controller.New() *Controller`, `Controller.ToggleMode() error`, `Controller.GetCurrentMode() string`, `Controller.Shutdown() error`

- [ ] **Step 1: Write failing test for controller toggle to Kiro mode**

Create `internal/controller/controller_test.go`:
```go
func TestToggleToKiroMode(t *testing.T) {
    dir := t.TempDir()
    
    // Setup test environment
    configPath := filepath.Join(dir, "config.json")
    settingsPath := filepath.Join(dir, "settings.json")
    backupPath := filepath.Join(dir, "settings.backup.json")
    gatewayDir := filepath.Join(dir, "gateway")
    
    // Create initial settings.json
    writeJSON(t, settingsPath, map[string]interface{}{"tui": "fullscreen"})
    
    // Create fake gateway
    setupFakeGateway(t, gatewayDir)
    
    ctrl := NewController(configPath, settingsPath, backupPath, gatewayDir, 8000)
    
    assert.Equal(t, "anthropic", ctrl.GetCurrentMode())
    
    err := ctrl.ToggleMode()
    
    assert.NoError(t, err)
    assert.Equal(t, "kiro", ctrl.GetCurrentMode())
    
    // Verify backup created
    assert.FileExists(t, backupPath)
    
    // Verify settings modified
    settings := readJSON(t, settingsPath)
    assert.NotNil(t, settings["env"])
    
    // Cleanup
    ctrl.Shutdown()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/controller -v`
Expected: FAIL - "NewController" undefined

- [ ] **Step 3: Implement Controller struct in `internal/controller/controller.go`**

```go
type Controller struct {
    config      *config.Config
    settings    *settings.Manager
    gateway     *gateway.Manager
    mu          sync.Mutex
}

func NewController(configPath, settingsPath, backupPath, gatewayDir string, port int) *Controller
func (c *Controller) GetCurrentMode() string
func (c *Controller) ToggleMode() error
func (c *Controller) Shutdown() error
```

Implementation of ToggleMode:
1. Lock mutex (prevent concurrent toggles)
2. If current mode is "anthropic":
   - Backup settings.json
   - Setup gateway
   - Start gateway
   - Apply Kiro mode to settings.json
   - Update config.CurrentMode to "kiro"
   - Save config
3. If current mode is "kiro":
   - Stop gateway
   - Restore settings.json from backup
   - Update config.CurrentMode to "anthropic"
   - Save config
4. Check if Claude Code running, log warning if so
5. Unlock mutex

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/controller -v -timeout 30s`
Expected: PASS

- [ ] **Step 5: Write test for toggle back to Anthropic mode**

Add to `internal/controller/controller_test.go`:
```go
func TestToggleBackToAnthropicMode(t *testing.T) {
    dir := t.TempDir()
    
    // Setup and toggle to Kiro
    ctrl := setupControllerInKiroMode(t, dir)
    
    err := ctrl.ToggleMode()
    
    assert.NoError(t, err)
    assert.Equal(t, "anthropic", ctrl.GetCurrentMode())
    
    // Verify settings restored
    settings := readJSON(t, filepath.Join(dir, "settings.json"))
    assert.Nil(t, settings["env"])
    
    ctrl.Shutdown()
}
```

Run test, verify it passes.

- [ ] **Step 6: Add test for concurrent toggle prevention**

Add to `internal/controller/controller_test.go`:
```go
func TestToggleMode_Concurrent(t *testing.T) {
    dir := t.TempDir()
    ctrl := setupController(t, dir)
    
    var wg sync.WaitGroup
    errors := make([]error, 2)
    
    // Try to toggle twice concurrently
    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            errors[idx] = ctrl.ToggleMode()
        }(i)
    }
    
    wg.Wait()
    
    // Both should succeed (mutex prevents race)
    // Final state should be consistent
    mode := ctrl.GetCurrentMode()
    assert.Contains(t, []string{"anthropic", "kiro"}, mode)
    
    ctrl.Shutdown()
}
```

Run test, verify it passes (mutex should handle this).

- [ ] **Step 7: Add test for missing backup error**

Add to `internal/controller/controller_test.go`:
```go
func TestToggleToAnthropicMode_MissingBackup(t *testing.T) {
    dir := t.TempDir()
    
    // Setup in Kiro mode but delete backup
    ctrl := setupControllerInKiroMode(t, dir)
    os.Remove(filepath.Join(dir, "settings.backup.json"))
    
    err := ctrl.ToggleMode()
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "backup")
    
    // Mode should remain "kiro" on error
    assert.Equal(t, "kiro", ctrl.GetCurrentMode())
    
    ctrl.Shutdown()
}
```

Run test, verify it passes.

- [ ] **Step 8: Commit**

```bash
git add internal/state/ internal/controller/
git commit -m "feat: add controller with mode toggling logic"
```

---

### Task 6: Menu Bar UI

**Files:**
- Create: `main.go`

**Interfaces:**
- Consumes: `controller.Controller`, `controller.ToggleMode()`, `controller.GetCurrentMode()`
- Produces: Runnable macOS menu bar application

- [ ] **Step 1: Write basic main.go with systray**

Create `main.go`:
```go
package main

import (
    "log"
    "os"
    "path/filepath"
    
    "github.com/getlantern/systray"
    "github.com/sparkreel/kiront/internal/controller"
)

func main() {
    systray.Run(onReady, onExit)
}

func onReady() {
    // TODO: implement
}

func onExit() {
    // TODO: implement
}
```

- [ ] **Step 2: Build to verify it compiles**

Run: `go build -o kiront .`
Expected: Binary created successfully

- [ ] **Step 3: Implement onReady with menu bar items**

Update `main.go`:
```go
var ctrl *controller.Controller

func onReady() {
    // Initialize controller
    home := os.Getenv("HOME")
    configPath := filepath.Join(home, ".kiront", "config.json")
    settingsPath := filepath.Join(home, ".claude", "settings.json")
    backupPath := filepath.Join(home, ".kiront", "settings.backup.json")
    
    // Determine gateway directory (relative to executable)
    execPath, _ := os.Executable()
    execDir := filepath.Dir(execPath)
    gatewayDir := filepath.Join(execDir, "gateway")
    
    ctrl = controller.NewController(configPath, settingsPath, backupPath, gatewayDir, 8000)
    
    // Set initial icon/title
    updateMenuBar()
    
    // Create menu items
    mToggle := systray.AddMenuItem("Toggle Mode", "Switch between Anthropic and Kiro modes")
    systray.AddSeparator()
    mQuit := systray.AddMenuItem("Quit", "Quit Kiront")
    
    // Handle clicks
    go func() {
        for {
            select {
            case <-mToggle.ClickedCh:
                handleToggle()
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }()
}

func updateMenuBar() {
    mode := ctrl.GetCurrentMode()
    if mode == "kiro" {
        systray.SetTitle("K")
        systray.SetTooltip("Kiront - Kiro Mode")
    } else {
        systray.SetTitle("A")
        systray.SetTooltip("Kiront - Anthropic Mode")
    }
}

func handleToggle() {
    err := ctrl.ToggleMode()
    if err != nil {
        log.Printf("Toggle failed: %v", err)
        // TODO: Show notification
        return
    }
    
    updateMenuBar()
    
    // Check for running Claude Code processes
    if process.IsClaudeCodeRunning() {
        // TODO: Show notification about restarting Claude Code
        log.Println("⚠️ Claude Code is running. Please restart your Claude Code sessions.")
    }
}

func onExit() {
    if ctrl != nil {
        ctrl.Shutdown()
    }
}
```

- [ ] **Step 4: Build and manually test**

Run: `go build -o kiront .`
Expected: Binary runs, shows menu bar icon, can toggle modes

- [ ] **Step 5: Add macOS notification support**

Install notification library:
```bash
go get github.com/gen2brain/beeep@latest
```

Update handleToggle to show notification:
```go
import "github.com/gen2brain/beeep"

func handleToggle() {
    err := ctrl.ToggleMode()
    if err != nil {
        beeep.Alert("Kiront Error", err.Error(), "")
        return
    }
    
    updateMenuBar()
    
    mode := ctrl.GetCurrentMode()
    msg := fmt.Sprintf("Switched to %s mode", mode)
    
    if process.IsClaudeCodeRunning() {
        msg += "\n\n⚠️ Please restart Claude Code sessions for changes to take effect."
    }
    
    beeep.Notify("Kiront", msg, "")
}
```

- [ ] **Step 6: Test manually**

Run: `go build -o kiront . && ./kiront`
Test: Toggle modes, verify notifications appear

- [ ] **Step 7: Commit**

```bash
git add main.go go.mod go.sum
git commit -m "feat: add menu bar UI with toggle and notifications"
```

---

### Task 7: Bundle Gateway Code

**Files:**
- Create: `gateway/` directory with all Python files copied from `/Users/yuri/kiro-gateway`

**Interfaces:**
- Consumes: Gateway source at `/Users/yuri/kiro-gateway`
- Produces: Bundled gateway at `gateway/` in kiront repo

- [ ] **Step 1: Copy gateway files**

```bash
mkdir -p gateway
cp -r /Users/yuri/kiro-gateway/* gateway/
# Exclude venv if it exists
rm -rf gateway/.venv
rm -rf gateway/.git
```

- [ ] **Step 2: Verify gateway structure**

Check that `gateway/main.py` and `gateway/requirements.txt` exist:
```bash
ls -la gateway/
```

Expected: All Python files present

- [ ] **Step 3: Add .gitignore for gateway venv**

Create `gateway/.gitignore`:
```
.venv/
*.pyc
__pycache__/
.env
```

- [ ] **Step 4: Test gateway setup works**

Run setup manually:
```bash
cd gateway
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add gateway/
git commit -m "feat: bundle kiro-gateway Python code"
```

---

### Task 8: Auto-Start on Login

**Files:**
- Modify: `main.go`
- Create: `internal/autostart/autostart.go`

**Interfaces:**
- Consumes: `config.Config.AutoStart`
- Produces: `autostart.Enable() error`, `autostart.Disable() error`, `autostart.IsEnabled() bool`

- [ ] **Step 1: Write autostart module**

Create `internal/autostart/autostart.go`:
```go
package autostart

import (
    "os"
    "path/filepath"
    "text/template"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>uy.ck.kiront</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecutablePath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`

func Enable() error
func Disable() error
func IsEnabled() bool
```

Implementation: 
- Enable: Write plist to `~/Library/LaunchAgents/uy.ck.kiront.plist`
- Disable: Remove plist file
- IsEnabled: Check if plist file exists

- [ ] **Step 2: Add autostart menu item to main.go**

Update `onReady()` in `main.go`:
```go
mToggle := systray.AddMenuItem("Toggle Mode", "Switch between Anthropic and Kiro modes")
systray.AddSeparator()
mAutoStart := systray.AddMenuItemCheckbox("Start at Login", "Launch Kiront on macOS login", ctrl.GetConfig().AutoStart)
systray.AddSeparator()
mQuit := systray.AddMenuItem("Quit", "Quit Kiront")

// In event loop:
case <-mAutoStart.ClickedCh:
    handleAutoStartToggle(mAutoStart)
```

Add handler:
```go
func handleAutoStartToggle(item *systray.MenuItem) {
    cfg := ctrl.GetConfig()
    
    if cfg.AutoStart {
        // Disable
        autostart.Disable()
        cfg.AutoStart = false
        item.Uncheck()
    } else {
        // Enable
        err := autostart.Enable()
        if err != nil {
            beeep.Alert("Kiront Error", "Failed to enable auto-start: "+err.Error(), "")
            return
        }
        cfg.AutoStart = true
        item.Check()
    }
    
    config.Save(cfg)
}
```

- [ ] **Step 3: Test manually**

Build and run, toggle auto-start, verify plist created/removed in `~/Library/LaunchAgents/`

- [ ] **Step 4: Commit**

```bash
git add internal/autostart/ main.go
git commit -m "feat: add auto-start on login support"
```

---

### Task 9: Crash Recovery and State Restoration

**Files:**
- Modify: `internal/controller/controller.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: `config.Config`, `gateway.Manager`
- Produces: `controller.RecoverState() error`

- [ ] **Step 1: Add RecoverState method to controller**

Add to `internal/controller/controller.go`:
```go
func (c *Controller) RecoverState() error {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    c.config = cfg
    
    // If config says Kiro mode, check if gateway is actually running
    if cfg.CurrentMode == "kiro" {
        if !c.gateway.IsRunning() {
            // Orphaned state: gateway not running but config says Kiro
            // Try to start gateway
            err := c.gateway.Start()
            if err != nil {
                // Gateway start failed, switch to Anthropic mode
                log.Printf("Failed to restart gateway on recovery: %v", err)
                c.settings.Restore()
                cfg.CurrentMode = "anthropic"
                config.Save(cfg)
            }
        }
    }
    
    return nil
}
```

- [ ] **Step 2: Call RecoverState in main.go onReady**

Update `onReady()`:
```go
ctrl = controller.NewController(...)

// Recover state on startup
err := ctrl.RecoverState()
if err != nil {
    log.Printf("State recovery failed: %v", err)
}

updateMenuBar()
```

- [ ] **Step 3: Test crash recovery manually**

1. Build and run Kiront
2. Toggle to Kiro mode
3. Kill the app with `kill -9 <pid>`
4. Kill gateway process manually
5. Restart Kiront
6. Verify it attempts to restart gateway or falls back to Anthropic mode

- [ ] **Step 4: Commit**

```bash
git add internal/controller/controller.go main.go
git commit -m "feat: add crash recovery and state restoration"
```

---

### Task 10: README and Documentation

**Files:**
- Create: `README.md`
- Create: `docs/USAGE.md`

**Interfaces:**
- Consumes: none
- Produces: User-facing documentation

- [ ] **Step 1: Write README.md**

Create `README.md` with sections:
- What is Kiront
- Requirements (macOS, Python 3.x, Claude Code)
- Installation (build from source, future: download binary)
- Usage (menu bar icon, toggle modes, settings)
- Configuration (config.json structure)
- Troubleshooting (common issues)

- [ ] **Step 2: Write USAGE.md**

Create `docs/USAGE.md` with detailed usage instructions:
- First-time setup
- Toggling modes
- Auto-start configuration
- What happens when you toggle
- Restarting Claude Code sessions

- [ ] **Step 3: Commit**

```bash
git add README.md docs/USAGE.md
git commit -m "docs: add README and usage documentation"
```

---

### Task 11: Build and Distribution

**Files:**
- Create: `Makefile`
- Create: `.goreleaser.yml` (optional, for future releases)

**Interfaces:**
- Consumes: All source files
- Produces: Distributable binary

- [ ] **Step 1: Create Makefile**

Create `Makefile`:
```makefile
.PHONY: build clean install test

build:
	go build -o bin/kiront .

clean:
	rm -rf bin/

install: build
	mkdir -p ~/bin
	cp bin/kiront ~/bin/
	chmod +x ~/bin/kiront

test:
	go test ./... -v

run: build
	./bin/kiront
```

- [ ] **Step 2: Test build**

Run: `make build`
Expected: Binary created at `bin/kiront`

- [ ] **Step 3: Create install target**

Run: `make install`
Expected: Binary copied to `~/bin/kiront`

- [ ] **Step 4: Verify installed binary works**

Run: `~/bin/kiront`
Expected: Menu bar app launches successfully

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "build: add Makefile for build and install"
```

---

## Final Integration Test

After all tasks complete, perform end-to-end testing:

1. **Clean state test:**
   - Remove `~/.kiront/` directory
   - Remove any gateway processes
   - Launch Kiront
   - Verify default Anthropic mode
   - Toggle to Kiro mode
   - Verify gateway starts, settings.json modified, backup created
   - Toggle back to Anthropic mode
   - Verify gateway stops, settings.json restored

2. **Crash recovery test:**
   - Toggle to Kiro mode
   - Kill Kiront with `kill -9`
   - Restart Kiront
   - Verify state recovered correctly

3. **Claude Code integration test:**
   - Start Claude Code session
   - Toggle Kiront to Kiro mode
   - Verify notification appears about restarting Claude Code
   - Exit Claude Code
   - Start new Claude Code session
   - Verify it connects through gateway (check logs)

4. **Auto-start test:**
   - Enable auto-start
   - Log out and log back in
   - Verify Kiront launches automatically
   - Verify it's in the mode it was before logout
