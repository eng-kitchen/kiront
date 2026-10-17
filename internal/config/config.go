package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	KiroAPIKey    string `json:"kiro_api_key"`
	GatewayPort   int    `json:"gateway_port"`
	KiroCLIDBPath string `json:"kiro_cli_db_path"`
	CurrentMode   string `json:"current_mode"`
	AutoStart     bool   `json:"auto_start"`
}

func LoadFrom(path string) (*Config, error) {
	// Check if file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Return default config
		return &Config{
			CurrentMode:   "anthropic",
			GatewayPort:   8000,
			AutoStart:     true,
			KiroCLIDBPath: "/Users/yuri/Library/Application Support/kiro-cli/data.sqlite3",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	// Read and unmarshal existing config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveTo(path string, cfg *Config) error {
	// Marshal config to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to temp file in same directory
	tempFile, err := os.CreateTemp(dir, "config-*.json.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	// Ensure temp file is cleaned up on error
	defer func() {
		if tempFile != nil {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		return err
	}

	// Sync to disk
	if err := tempFile.Sync(); err != nil {
		return err
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return err
	}

	// Atomically rename temp file to target path
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}

	// Clear tempFile so defer doesn't remove it
	tempFile = nil

	return nil
}

func Load() (*Config, error) {
	return LoadFrom(filepath.Join(os.Getenv("HOME"), ".kiront", "config.json"))
}

func Save(cfg *Config) error {
	return SaveTo(filepath.Join(os.Getenv("HOME"), ".kiront", "config.json"), cfg)
}
