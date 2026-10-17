package controller

import (
	"fmt"
	"log"
	"sync"

	"github.com/sparkreel/kiront/internal/config"
	"github.com/sparkreel/kiront/internal/gateway"
	"github.com/sparkreel/kiront/internal/process"
	"github.com/sparkreel/kiront/internal/settings"
)

type Controller struct {
	configPath string
	cfg        *config.Config
	settings   *settings.Manager
	gateway    *gateway.Manager
	mu         sync.Mutex
}

// NewController creates a new controller with the given paths and port
func NewController(configPath, settingsPath, backupPath, gatewayDir string, port int) *Controller {
	// Load or create config
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		log.Printf("Failed to load config, using defaults: %v", err)
		cfg = &config.Config{
			CurrentMode: "anthropic",
			GatewayPort: port,
			AutoStart:   true,
		}
	}

	return &Controller{
		configPath: configPath,
		cfg:        cfg,
		settings:   settings.NewManager(settingsPath, backupPath),
		gateway:    gateway.NewManager(gatewayDir, port),
	}
}

// GetCurrentMode returns the current operational mode
func (c *Controller) GetCurrentMode() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg.CurrentMode
}

// GetConfig returns the current configuration (for UI access)
func (c *Controller) GetConfig() *config.Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg
}

// ToggleMode switches between Anthropic and Kiro modes
func (c *Controller) ToggleMode() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cfg.CurrentMode == "anthropic" {
		// Switching to Kiro mode
		return c.switchToKiroMode()
	} else {
		// Switching to Anthropic mode
		return c.switchToAnthropicMode()
	}
}

func (c *Controller) switchToKiroMode() error {
	// Step 1: Backup settings.json
	if err := c.settings.Backup(); err != nil {
		return fmt.Errorf("failed to backup settings: %w", err)
	}

	// Step 2: Setup gateway
	if err := c.gateway.Setup(c.cfg.KiroAPIKey, c.cfg.KiroCLIDBPath); err != nil {
		return fmt.Errorf("failed to setup gateway: %w", err)
	}

	// Step 3: Start gateway
	if err := c.gateway.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// Step 4: Apply Kiro mode to settings.json
	if err := c.settings.ApplyKiroMode(c.cfg.KiroAPIKey); err != nil {
		// Rollback: stop gateway
		c.gateway.Stop()
		return fmt.Errorf("failed to apply Kiro mode to settings: %w", err)
	}

	// Step 5: Update config
	c.cfg.CurrentMode = "kiro"
	if err := config.SaveTo(c.configPath, c.cfg); err != nil {
		log.Printf("Warning: failed to save config: %v", err)
	}

	// Step 6: Check if Claude Code is running
	if process.IsClaudeCodeRunning() {
		log.Println("⚠️ Claude Code is running. Please restart your Claude Code sessions for the mode change to take effect.")
	}

	return nil
}

func (c *Controller) switchToAnthropicMode() error {
	// Step 1: Stop gateway
	if err := c.gateway.Stop(); err != nil {
		log.Printf("Warning: failed to stop gateway cleanly: %v", err)
	}

	// Step 2: Restore settings.json from backup
	if !c.settings.HasBackup() {
		return fmt.Errorf("cannot restore to Anthropic mode: backup not found")
	}

	if err := c.settings.Restore(); err != nil {
		return fmt.Errorf("failed to restore settings: %w", err)
	}

	// Step 3: Update config
	c.cfg.CurrentMode = "anthropic"
	if err := config.SaveTo(c.configPath, c.cfg); err != nil {
		log.Printf("Warning: failed to save config: %v", err)
	}

	// Step 4: Check if Claude Code is running
	if process.IsClaudeCodeRunning() {
		log.Println("⚠️ Claude Code is running. Please restart your Claude Code sessions for the mode change to take effect.")
	}

	return nil
}

// Shutdown cleanly shuts down the controller
func (c *Controller) Shutdown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If in Kiro mode, stop the gateway
	if c.cfg.CurrentMode == "kiro" {
		if err := c.gateway.Stop(); err != nil {
			return fmt.Errorf("failed to stop gateway during shutdown: %w", err)
		}
	}

	return nil
}
