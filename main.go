package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/eng-kitchen/kiront/internal/autostart"
	"github.com/eng-kitchen/kiront/internal/config"
	"github.com/eng-kitchen/kiront/internal/controller"
	"github.com/eng-kitchen/kiront/internal/process"
)

var ctrl *controller.Controller

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Initialize controller
	home := os.Getenv("HOME")
	configPath := filepath.Join(home, ".kiront", "config.json")
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	backupPath := filepath.Join(home, ".kiront", "settings.backup.json")

	// Determine gateway directory (relative to executable)
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	gatewayDir := filepath.Join(execDir, "gateway")

	ctrl = controller.NewController(configPath, settingsPath, backupPath, gatewayDir, 8000)

	// Recover state on startup (handle crashes)
	if err := ctrl.RecoverState(); err != nil {
		log.Printf("State recovery failed: %v", err)
	}

	// Set initial icon/title
	updateMenuBar()

	// Create menu items
	mToggle := systray.AddMenuItem("Toggle Mode", "Switch between Anthropic and Kiro modes")
	systray.AddSeparator()
	mAutoStart := systray.AddMenuItemCheckbox("Start at Login", "Launch Kiront on macOS login", ctrl.GetConfig().AutoStart)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Kiront")

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
		systray.SetTooltip("Kiront - Kiro Mode")
	} else {
		systray.SetTitle("A")
		systray.SetTooltip("Kiront - Anthropic Mode")
	}
}

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

func handleAutoStartToggle(item *systray.MenuItem) {
	cfg := ctrl.GetConfig()

	if cfg.AutoStart {
		// Disable auto-start
		if err := autostart.Disable(); err != nil {
			beeep.Alert("Kiront Error", "Failed to disable auto-start: "+err.Error(), "")
			return
		}
		cfg.AutoStart = false
		item.Uncheck()
	} else {
		// Enable auto-start
		if err := autostart.Enable(); err != nil {
			beeep.Alert("Kiront Error", "Failed to enable auto-start: "+err.Error(), "")
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
