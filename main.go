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
	log.Println("onReady: Starting initialization...")

	// Initialize controller
	home := os.Getenv("HOME")
	configPath := filepath.Join(home, ".kiront", "config.json")
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	backupPath := filepath.Join(home, ".kiront", "settings.backup.json")

	log.Printf("onReady: Config path: %s", configPath)

	// Determine gateway directory (relative to executable)
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)

	// For .app bundles, gateway is in Contents/Resources
	if filepath.Base(execDir) == "MacOS" {
		execDir = filepath.Join(filepath.Dir(execDir), "Resources")
	}

	gatewayDir := filepath.Join(execDir, "gateway")
	log.Printf("onReady: Gateway dir: %s", gatewayDir)

	log.Println("onReady: Creating controller...")
	ctrl = controller.NewController(configPath, settingsPath, backupPath, gatewayDir, 8000)

	log.Println("onReady: Recovering state...")
	// Recover state on startup (handle crashes)
	if err := ctrl.RecoverState(); err != nil {
		log.Printf("State recovery failed: %v", err)
	}

	log.Println("onReady: Updating menu bar...")
	// Set initial icon/title
	updateMenuBar()
	log.Println("onReady: Menu bar updated")

	log.Println("onReady: Creating menu items...")
	// Create menu items
	mToggle := systray.AddMenuItem("Toggle Mode", "Switch between Anthropic and Kiro modes")
	systray.AddSeparator()
	mAutoStart := systray.AddMenuItemCheckbox("Start at Login", "Launch Kiront on macOS login", ctrl.GetConfig().AutoStart)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Kiront")
	log.Println("onReady: Menu items created, entering event loop...")

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
		systray.SetTemplateIcon(iconKiroPNG, iconKiroPNG)
		systray.SetTooltip("Kiront - Kiro Mode")
	} else {
		systray.SetTemplateIcon(iconAnthropicPNG, iconAnthropicPNG)
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
