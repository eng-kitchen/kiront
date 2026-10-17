package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>uy.ck.kiront</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecutablePath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`

type plistData struct {
	ExecutablePath string
}

// getPlistPath returns the path to the LaunchAgent plist file
func getPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", "uy.ck.kiront.plist"), nil
}

// Enable creates the LaunchAgent plist file to start the app on login
func Enable() error {
	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get plist path
	plistPath, err := getPlistPath()
	if err != nil {
		return err
	}

	// Create LaunchAgents directory if it doesn't exist
	launchAgentsDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Parse template
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse plist template: %w", err)
	}

	// Create plist file
	file, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer file.Close()

	// Execute template
	data := plistData{ExecutablePath: execPath}
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	return nil
}

// Disable removes the LaunchAgent plist file
func Disable() error {
	plistPath, err := getPlistPath()
	if err != nil {
		return err
	}

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, which is fine
			return nil
		}
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

// IsEnabled checks if the LaunchAgent plist file exists
func IsEnabled() bool {
	plistPath, err := getPlistPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(plistPath)
	return err == nil
}
