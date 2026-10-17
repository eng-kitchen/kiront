package config

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_NewFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg, err := LoadFrom(configPath)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.CurrentMode != "anthropic" {
		t.Errorf("expected CurrentMode 'anthropic', got '%s'", cfg.CurrentMode)
	}
	if cfg.GatewayPort != 8000 {
		t.Errorf("expected GatewayPort 8000, got %d", cfg.GatewayPort)
	}
	if !cfg.AutoStart {
		t.Errorf("expected AutoStart true, got false")
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		KiroAPIKey:  "test-key",
		CurrentMode: "kiro",
		GatewayPort: 8000,
		AutoStart:   true,
	}

	err := SaveTo(configPath, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file contents
	loaded, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected no error loading, got %v", err)
	}
	if loaded.KiroAPIKey != "test-key" {
		t.Errorf("expected KiroAPIKey 'test-key', got '%s'", loaded.KiroAPIKey)
	}
	if loaded.CurrentMode != "kiro" {
		t.Errorf("expected CurrentMode 'kiro', got '%s'", loaded.CurrentMode)
	}
}
