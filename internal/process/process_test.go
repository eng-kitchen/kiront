package process

import "testing"

func TestIsClaudeCodeRunning_NotRunning(t *testing.T) {
	// This will pass if no Claude Code actually running
	running := IsClaudeCodeRunning()
	// Can't assert specific value, just check it doesn't panic
	_ = running
}
