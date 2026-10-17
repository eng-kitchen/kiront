package gateway

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestStart(t *testing.T) {
	// This test requires actual gateway code, so we'll mock it
	dir := t.TempDir()
	gatewayDir := filepath.Join(dir, "gateway")
	os.MkdirAll(gatewayDir, 0755)

	// Create fake venv structure
	venvBinDir := filepath.Join(gatewayDir, ".venv", "bin")
	os.MkdirAll(venvBinDir, 0755)

	// Create a fake python executable that's a wrapper script calling system python3
	pythonWrapper := filepath.Join(venvBinDir, "python")
	wrapperScript := "#!/bin/sh\nexec python3 \"$@\"\n"
	os.WriteFile(pythonWrapper, []byte(wrapperScript), 0755)

	// Create fake Python script that responds to health checks
	mainPy := `import http.server
import json

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "healthy"}).encode())
    def log_message(self, format, *args):
        pass

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

func TestStop(t *testing.T) {
	dir := t.TempDir()
	gatewayDir := filepath.Join(dir, "gateway")
	os.MkdirAll(gatewayDir, 0755)

	// Create fake venv structure
	venvBinDir := filepath.Join(gatewayDir, ".venv", "bin")
	os.MkdirAll(venvBinDir, 0755)

	// Create a fake python executable that's a wrapper script calling system python3
	pythonWrapper := filepath.Join(venvBinDir, "python")
	wrapperScript := "#!/bin/sh\nexec python3 \"$@\"\n"
	os.WriteFile(pythonWrapper, []byte(wrapperScript), 0755)

	// Create fake Python script that responds to health checks
	mainPy := `import http.server
import json

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "healthy"}).encode())
    def log_message(self, format, *args):
        pass

server = http.server.HTTPServer(('127.0.0.1', 8000), Handler)
server.serve_forever()
`
	os.WriteFile(filepath.Join(gatewayDir, "main.py"), []byte(mainPy), 0644)

	mgr := NewManager(gatewayDir, 8000)
	mgr.Start()
	assert.True(t, mgr.IsRunning())

	err := mgr.Stop()

	assert.NoError(t, err)
	assert.False(t, mgr.IsRunning())
}

func TestStart_Timeout(t *testing.T) {
	dir := t.TempDir()
	gatewayDir := filepath.Join(dir, "gateway")
	os.MkdirAll(gatewayDir, 0755)

	// Create fake venv structure
	venvBinDir := filepath.Join(gatewayDir, ".venv", "bin")
	os.MkdirAll(venvBinDir, 0755)

	// Create a fake python executable that's a wrapper script calling system python3
	pythonWrapper := filepath.Join(venvBinDir, "python")
	wrapperScript := "#!/bin/sh\nexec python3 \"$@\"\n"
	os.WriteFile(pythonWrapper, []byte(wrapperScript), 0755)

	// Create Python script that sleeps forever without opening any port
	mainPy := `import time
time.sleep(1000)
`
	os.WriteFile(filepath.Join(gatewayDir, "main.py"), []byte(mainPy), 0644)

	// Use a different port to avoid conflicts with other tests
	mgr := NewManager(gatewayDir, 9876)
	err := mgr.Start()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")

	// Ensure process is cleaned up - read PID file and kill if exists
	if pidData, err := os.ReadFile(mgr.pidFile); err == nil {
		if pid, err := strconv.Atoi(string(pidData)); err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				process.Kill()
			}
		}
		os.Remove(mgr.pidFile)
	}
}
