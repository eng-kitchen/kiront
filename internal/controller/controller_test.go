package controller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Test helper: write JSON to file
func writeJSON(t *testing.T, path string, data map[string]interface{}) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

// Test helper: read JSON from file
func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	return result
}

// Test helper: setup fake gateway for testing
func setupFakeGateway(t *testing.T, gatewayDir string) {
	t.Helper()
	if err := os.MkdirAll(gatewayDir, 0755); err != nil {
		t.Fatalf("failed to create gateway dir: %v", err)
	}

	// Create fake main.py that responds to health checks
	mainPy := `#!/usr/bin/env python3
import http.server
import json
import sys

class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        pass  # Suppress logs

    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "healthy"}).encode())
        else:
            self.send_response(404)
            self.end_headers()

try:
    server = http.server.HTTPServer(('127.0.0.1', 8000), Handler)
    server.serve_forever()
except KeyboardInterrupt:
    sys.exit(0)
`
	if err := os.WriteFile(filepath.Join(gatewayDir, "main.py"), []byte(mainPy), 0755); err != nil {
		t.Fatalf("failed to write main.py: %v", err)
	}

	// Create fake requirements.txt
	requirements := "# No dependencies for test\n"
	if err := os.WriteFile(filepath.Join(gatewayDir, "requirements.txt"), []byte(requirements), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}
}

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

	if ctrl.GetCurrentMode() != "anthropic" {
		t.Errorf("expected initial mode to be 'anthropic', got %q", ctrl.GetCurrentMode())
	}

	err := ctrl.ToggleMode()
	if err != nil {
		t.Fatalf("ToggleMode failed: %v", err)
	}

	if ctrl.GetCurrentMode() != "kiro" {
		t.Errorf("expected mode after toggle to be 'kiro', got %q", ctrl.GetCurrentMode())
	}

	// Verify backup created
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file not created")
	}

	// Verify settings modified
	settings := readJSON(t, settingsPath)
	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		t.Fatal("env key not found or not a map")
	}

	if env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:8000" {
		t.Errorf("ANTHROPIC_BASE_URL not set correctly")
	}

	// Cleanup
	ctrl.Shutdown()
}

// Test helper: setup controller already in Kiro mode
func setupControllerInKiroMode(t *testing.T, dir string) *Controller {
	t.Helper()

	configPath := filepath.Join(dir, "config.json")
	settingsPath := filepath.Join(dir, "settings.json")
	backupPath := filepath.Join(dir, "settings.backup.json")
	gatewayDir := filepath.Join(dir, "gateway")

	// Create initial settings.json
	writeJSON(t, settingsPath, map[string]interface{}{"tui": "fullscreen"})

	// Create fake gateway
	setupFakeGateway(t, gatewayDir)

	// Create controller and toggle to Kiro
	ctrl := NewController(configPath, settingsPath, backupPath, gatewayDir, 8000)
	if err := ctrl.ToggleMode(); err != nil {
		t.Fatalf("Failed to toggle to Kiro mode: %v", err)
	}

	return ctrl
}

func TestToggleBackToAnthropicMode(t *testing.T) {
	dir := t.TempDir()

	// Setup paths
	settingsPath := filepath.Join(dir, "settings.json")

	// Setup controller in Kiro mode
	ctrl := setupControllerInKiroMode(t, dir)
	defer ctrl.Shutdown()

	if ctrl.GetCurrentMode() != "kiro" {
		t.Fatalf("expected mode to be 'kiro', got %q", ctrl.GetCurrentMode())
	}

	// Toggle back to Anthropic
	err := ctrl.ToggleMode()
	if err != nil {
		t.Fatalf("ToggleMode failed: %v", err)
	}

	if ctrl.GetCurrentMode() != "anthropic" {
		t.Errorf("expected mode after toggle to be 'anthropic', got %q", ctrl.GetCurrentMode())
	}

	// Verify settings restored (env should be removed)
	settings := readJSON(t, settingsPath)
	if env, ok := settings["env"].(map[string]interface{}); ok {
		if _, hasBaseURL := env["ANTHROPIC_BASE_URL"]; hasBaseURL {
			t.Error("ANTHROPIC_BASE_URL should be removed after restoring to Anthropic mode")
		}
	}

	// Verify original settings preserved
	if settings["tui"] != "fullscreen" {
		t.Error("original settings not preserved")
	}
}

func TestToggleMode_Concurrent(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, "config.json")
	settingsPath := filepath.Join(dir, "settings.json")
	backupPath := filepath.Join(dir, "settings.backup.json")
	gatewayDir := filepath.Join(dir, "gateway")

	// Create initial settings.json
	writeJSON(t, settingsPath, map[string]interface{}{"tui": "fullscreen"})

	// Create fake gateway
	setupFakeGateway(t, gatewayDir)

	ctrl := NewController(configPath, settingsPath, backupPath, gatewayDir, 8000)
	defer ctrl.Shutdown()

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
	for i, err := range errors {
		if err != nil {
			t.Errorf("toggle %d failed: %v", i, err)
		}
	}

	// Final state should be consistent (toggled twice means back to original, or one succeeded)
	mode := ctrl.GetCurrentMode()
	if mode != "anthropic" && mode != "kiro" {
		t.Errorf("unexpected mode after concurrent toggles: %q", mode)
	}
}

func TestToggleToAnthropicMode_MissingBackup(t *testing.T) {
	dir := t.TempDir()

	backupPath := filepath.Join(dir, "settings.backup.json")

	// Setup in Kiro mode but delete backup
	ctrl := setupControllerInKiroMode(t, dir)
	defer ctrl.Shutdown()

	// Delete the backup file
	if err := os.Remove(backupPath); err != nil {
		t.Fatalf("failed to remove backup: %v", err)
	}

	// Try to toggle back
	err := ctrl.ToggleMode()

	if err == nil {
		t.Fatal("expected error when backup is missing, got nil")
	}

	if !contains(err.Error(), "backup") {
		t.Errorf("expected error to mention 'backup', got: %v", err)
	}

	// Mode should remain "kiro" on error
	if ctrl.GetCurrentMode() != "kiro" {
		t.Errorf("expected mode to remain 'kiro' on error, got %q", ctrl.GetCurrentMode())
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
