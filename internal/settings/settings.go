package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manager handles Claude Code settings.json backup, modification, and restoration
type Manager struct {
	settingsPath string
	backupPath   string
}

// NewManager creates a new settings manager with specified paths
func NewManager(settingsPath, backupPath string) *Manager {
	return &Manager{
		settingsPath: settingsPath,
		backupPath:   backupPath,
	}
}

// New creates a manager using default paths (~/.claude/settings.json and ~/.kiront/settings.backup.json)
func New() *Manager {
	home := os.Getenv("HOME")
	return NewManager(
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".kiront", "settings.backup.json"),
	)
}

// Backup creates a backup copy of the current settings file
func (m *Manager) Backup() error {
	// Read the current settings
	data, err := os.ReadFile(m.settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read settings file: %w", err)
	}

	// Ensure the backup directory exists
	backupDir := filepath.Dir(m.backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Write atomically: temp file + rename
	tempPath := m.backupPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup temp file: %w", err)
	}

	if err := os.Rename(tempPath, m.backupPath); err != nil {
		os.Remove(tempPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename backup temp file: %w", err)
	}

	return nil
}

// HasBackup returns true if a backup file exists
func (m *Manager) HasBackup() bool {
	_, err := os.Stat(m.backupPath)
	return err == nil
}

// Restore restores settings from the backup file
func (m *Manager) Restore() error {
	if !m.HasBackup() {
		return fmt.Errorf("backup not found at %s", m.backupPath)
	}

	// Read the backup
	data, err := os.ReadFile(m.backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write atomically: temp file + rename
	settingsDir := filepath.Dir(m.settingsPath)
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	tempPath := m.settingsPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings temp file: %w", err)
	}

	if err := os.Rename(tempPath, m.settingsPath); err != nil {
		os.Remove(tempPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename settings temp file: %w", err)
	}

	return nil
}

// ApplyKiroMode modifies settings to use Kiro gateway
func (m *Manager) ApplyKiroMode(apiKey string) error {
	// Read current settings
	data, err := os.ReadFile(m.settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read settings file: %w", err)
	}

	// Parse JSON
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse settings JSON: %w", err)
	}

	// Get existing env map or create new one
	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		env = make(map[string]interface{})
	}

	// Merge Kiro vars into existing env
	env["ANTHROPIC_BASE_URL"] = "http://127.0.0.1:8000"
	env["ANTHROPIC_API_KEY"] = apiKey
	env["ANTHROPIC_MODEL"] = "claude-sonnet-4-5"
	env["ANTHROPIC_SMALL_FAST_MODEL"] = "claude-haiku-4-5"

	settings["env"] = env

	// Marshal back to JSON
	modifiedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings JSON: %w", err)
	}

	// Write atomically: temp file + rename
	tempPath := m.settingsPath + ".tmp"
	if err := os.WriteFile(tempPath, modifiedData, 0644); err != nil {
		return fmt.Errorf("failed to write settings temp file: %w", err)
	}

	if err := os.Rename(tempPath, m.settingsPath); err != nil {
		os.Remove(tempPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename settings temp file: %w", err)
	}

	return nil
}
