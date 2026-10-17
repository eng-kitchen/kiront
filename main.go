package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/sparkreel/kiront/internal/autostart"
	"github.com/sparkreel/kiront/internal/config"
	"github.com/sparkreel/kiront/internal/controller"
	"github.com/sparkreel/kiront/internal/process"
)

var ctrl *controller.Controller

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Initialize controller
	home := os.Getenv("HOME")
	configPath := filepath.Join(home, ".kirotoggle", "config.json")
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	backupPath := filepath.Join(home, ".kirotoggle", "settings.backup.json")

	// Determine gateway directory (relative to executable)
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	gatewayDir := filepath.Join(execDir, "gateway")

	ctrl = controller.NewController(configPath, settingsPath, backupPath, gatewayDir, 8000)

	// Set initial icon/title
	updateMenuBar()

	// Create menu items
	mToggle := systray.AddMenuItem("Toggle Mode", "Switch between Anthropic and Kiro modes")
	systray.AddSeparator()
	mAutoStart := systray.AddMenuItemCheckbox("Start at Login", "Launch KiroToggle on macOS login", ctrl.GetConfig().AutoStart)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit KiroToggle")

	// Handle clicks
	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				handleToggle()
			case <-mAutoStart.ClickedCh:
				handleAutoStartToggle(mAutoStart)
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
		systray.SetTooltip("KiroToggle - Kiro Mode")
	} else {
		systray.SetTitle("A")
		systray.SetTooltip("KiroToggle - Anthropic Mode")
	}
}

func handleToggle() {
	err := ctrl.ToggleMode()
	if err != nil {
		beeep.Alert("KiroToggle Error", err.Error(), "")
		return
	}

	updateMenuBar()

	mode := ctrl.GetCurrentMode()
	msg := fmt.Sprintf("Switched to %s mode", mode)

	if process.IsClaudeCodeRunning() {
		msg += "\n\n⚠️ Please restart Claude Code sessions for changes to take effect."
	}

	beeep.Notify("KiroToggle", msg, "")
}

func handleAutoStartToggle(item *systray.MenuItem) {
	cfg := ctrl.GetConfig()

	if cfg.AutoStart {
		// Disable auto-start
		if err := autostart.Disable(); err != nil {
			beeep.Alert("KiroToggle Error", "Failed to disable auto-start: "+err.Error(), "")
			return
		}
		cfg.AutoStart = false
		item.Uncheck()
	} else {
		// Enable auto-start
		if err := autostart.Enable(); err != nil {
			beeep.Alert("KiroToggle Error", "Failed to enable auto-start: "+err.Error(), "")
			return
		}
		cfg.AutoStart = true
		item.Check()
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		log.Printf("Warning: failed to save config: %v", err)
	}
}

func onExit() {
	if ctrl != nil {
		if err := ctrl.Shutdown(); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}
}
