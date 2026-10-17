package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to write JSON to a file
func writeJSON(t *testing.T, path string, data map[string]interface{}) {
	t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

// Helper function to read JSON from a file
func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	return data
}

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

func TestApplyKiroMode(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	backupPath := filepath.Join(dir, "settings.backup.json")

	// Create fake settings.json
	original := map[string]interface{}{
		"hooks": map[string]interface{}{},
		"tui":   "fullscreen",
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

func TestApplyKiroMode_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	backupPath := filepath.Join(dir, "settings.backup.json")

	// Write malformed JSON
	os.WriteFile(settingsPath, []byte("{invalid json"), 0644)

	mgr := NewManager(settingsPath, backupPath)
	err := mgr.ApplyKiroMode("test-key")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}
