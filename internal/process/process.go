package process

import "os/exec"

// IsClaudeCodeRunning checks if any Claude Code processes are currently running.
// Returns true if Claude Code is detected, false otherwise.
func IsClaudeCodeRunning() bool {
	cmd := exec.Command("pgrep", "-f", "claude")
	err := cmd.Run()
	// pgrep returns exit code 0 if process found, 1 if not found
	return err == nil
}
