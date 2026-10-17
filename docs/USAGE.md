# KiroToggle Usage Guide

This guide provides detailed instructions for using KiroToggle to switch between Anthropic and Kiro modes for Claude Code.

## Table of Contents

1. [First-Time Setup](#first-time-setup)
2. [Basic Usage](#basic-usage)
3. [Understanding Modes](#understanding-modes)
4. [Toggling Between Modes](#toggling-between-modes)
5. [Auto-Start Configuration](#auto-start-configuration)
6. [Restarting Claude Code Sessions](#restarting-claude-code-sessions)
7. [Best Practices](#best-practices)
8. [Common Workflows](#common-workflows)

## First-Time Setup

### 1. Install KiroToggle

```bash
# Build and install
cd /path/to/kiront
make build
make install
```

### 2. Initial Launch

When you first run KiroToggle:

```bash
~/bin/kirotoggle
```

The app will:
- Create `~/.kirotoggle/` directory
- Generate default `config.json`
- Appear in your menu bar as "A" (Anthropic mode - the default)

### 3. Configure Your API Key

Edit `~/.kirotoggle/config.json` to set your Kiro API key:

```json
{
  "kiro_api_key": "your-actual-kiro-api-key",
  "gateway_port": 8000,
  "kiro_cli_db_path": "/Users/yourname/Library/Application Support/kiro-cli/data.sqlite3",
  "current_mode": "anthropic",
  "auto_start": true
}
```

**Important**: This API key must match the `ANTHROPIC_API_KEY` you want Claude Code to use when connecting through the gateway.

### 4. Verify Gateway Bundle

The gateway code is bundled within KiroToggle. Verify it's present:

```bash
ls "$(dirname $(which kirotoggle))/gateway/main.py"
```

If missing, ensure you built KiroToggle from the correct directory with the bundled gateway.

## Basic Usage

### Menu Bar Icon

KiroToggle shows its status in the macOS menu bar:

- **A** = Anthropic Mode (direct connection to Anthropic API)
- **K** = Kiro Mode (routing through local gateway)

### Opening the Menu

Click the menu bar icon to see available options:

```
Toggle Mode           ← Switch between Anthropic and Kiro
─────────────────
☑ Start at Login     ← Auto-start on macOS login (checkable)
─────────────────
Quit                  ← Exit KiroToggle
```

## Understanding Modes

### Anthropic Mode (Default)

**What it is**: Claude Code connects directly to Anthropic's API servers.

**When active**:
- Menu bar shows "A"
- Gateway server is stopped
- `~/.claude/settings.json` has no custom `env` variables (or original env restored)
- Claude Code uses default Anthropic endpoints

**Use this mode when**:
- You want standard Claude Code behavior
- You're not using Kiro-specific features
- You're troubleshooting issues

### Kiro Mode

**What it is**: Claude Code routes requests through a local gateway server running on port 8000.

**When active**:
- Menu bar shows "K"
- Gateway server is running (`http://127.0.0.1:8000`)
- `~/.claude/settings.json` has `env` variables pointing to gateway
- Claude Code sends all requests to the local gateway

**Use this mode when**:
- You want Kiro-enhanced functionality
- You need features the gateway provides
- You're working with Kiro-specific models

## Toggling Between Modes

### Switching to Kiro Mode

**Steps**:
1. Click the KiroToggle menu bar icon (shows "A")
2. Click "Toggle Mode"
3. Wait for the notification: "Switched to kiro mode"
4. Icon changes to "K"

**What happens automatically**:
1. ✅ Your current `~/.claude/settings.json` is backed up to `~/.kirotoggle/settings.backup.json`
2. ✅ Gateway Python environment is set up (first time only: creates venv, installs dependencies)
3. ✅ Gateway `.env` file is created with your API key and configuration
4. ✅ Gateway server starts on port 8000
5. ✅ Health check confirms gateway is ready (`/health` endpoint)
6. ✅ `~/.claude/settings.json` is updated with gateway environment variables:
   ```json
   {
     "env": {
       "ANTHROPIC_BASE_URL": "http://127.0.0.1:8000",
       "ANTHROPIC_API_KEY": "your-kiro-api-key",
       "ANTHROPIC_MODEL": "claude-sonnet-4-5",
       "ANTHROPIC_SMALL_FAST_MODEL": "claude-haiku-4-5"
     }
   }
   ```

**Timeline**: First toggle takes 5-10 seconds (venv setup), subsequent toggles take 2-3 seconds.

### Switching to Anthropic Mode

**Steps**:
1. Click the KiroToggle menu bar icon (shows "K")
2. Click "Toggle Mode"
3. Wait for the notification: "Switched to anthropic mode"
4. Icon changes to "A"

**What happens automatically**:
1. ✅ Gateway server receives SIGTERM (graceful shutdown request)
2. ✅ Wait up to 5 seconds for gateway to shut down cleanly
3. ✅ If still running, send SIGKILL (force stop)
4. ✅ Verify port 8000 is no longer in use
5. ✅ `~/.claude/settings.json` is restored from `~/.kirotoggle/settings.backup.json`
6. ✅ Gateway environment variables are removed from Claude Code config

**Timeline**: Usually completes in 1-2 seconds.

### If Toggle Fails

If you see an error notification:

1. **Check the error message** - it will indicate what went wrong
2. **Common issues**:
   - Port 8000 already in use → stop the conflicting process
   - Gateway won't start → check Python version, dependencies
   - Backup not found → you'll need to manually recreate your settings
3. **Safe to retry** - failed toggles are rolled back automatically

## Auto-Start Configuration

### Enabling Auto-Start

**To launch KiroToggle automatically when you log into macOS**:

1. Click the KiroToggle menu bar icon
2. Click "Start at Login" (checkbox will be checked ☑)
3. A LaunchAgent plist is created at `~/Library/LaunchAgents/com.sparkreel.kirotoggle.plist`

**Result**: 
- KiroToggle launches automatically on every macOS login
- Restores the mode you were in when you last quit (A or K)
- If in Kiro mode, automatically starts the gateway

### Disabling Auto-Start

1. Click the KiroToggle menu bar icon
2. Click "Start at Login" again (checkbox will be unchecked ☐)
3. The LaunchAgent plist is removed

### Persistent Mode

KiroToggle remembers your current mode across:
- App restarts
- macOS reboots
- Crashes (with recovery)

If you quit in Kiro mode, the next launch will:
1. Detect you were in Kiro mode
2. Attempt to restart the gateway
3. If gateway start fails, safely fall back to Anthropic mode

## Restarting Claude Code Sessions

### Why Restart is Needed

**Critical**: Claude Code does NOT hot-reload environment variables.

When you toggle modes, KiroToggle updates `~/.claude/settings.json`, but Claude Code processes that are already running:
- Still use the old environment variables they loaded at startup
- Will not see the new `ANTHROPIC_BASE_URL` or `ANTHROPIC_API_KEY`
- Will continue sending requests to the wrong endpoint

### The Warning

After toggling, if KiroToggle detects running Claude Code processes, you'll see:

```
⚠️ Please restart Claude Code sessions for changes to take effect.
```

### How to Restart Claude Code

**Terminal/CLI sessions**:
```bash
# Exit your current Claude session
exit

# Or press Ctrl+D

# Start a new session
claude
```

**IDE integrations (VS Code, JetBrains)**:
1. Close all Claude panels/windows
2. Restart the IDE (or reload the window)
3. Reopen Claude

**Desktop app**:
1. Quit the Claude desktop app completely (Cmd+Q)
2. Relaunch it

### Verifying the Mode Change

To confirm Claude Code is using the correct endpoint:

**In Kiro mode**, check your settings:
```bash
cat ~/.claude/settings.json | grep ANTHROPIC_BASE_URL
# Should show: "ANTHROPIC_BASE_URL": "http://127.0.0.1:8000"
```

**In Anthropic mode**, check your settings:
```bash
cat ~/.claude/settings.json | grep ANTHROPIC_BASE_URL
# Should show nothing (or the field won't exist)
```

**Test a Claude request** and watch the gateway logs (if in Kiro mode):
```bash
# In another terminal
tail -f ~/path/to/kirotoggle/gateway/logs/*.log
```

## Best Practices

### 1. Toggle Before Starting Work

**Recommended workflow**:
1. Decide which mode you need
2. Toggle to that mode via KiroToggle
3. Then start your Claude Code sessions
4. This avoids the restart step

### 2. Keep Auto-Start Enabled

If you frequently use KiroToggle:
- Enable "Start at Login"
- The app will be ready when you need it
- Minimal memory footprint when idle

### 3. Check Mode Before Important Tasks

Before starting a long coding session:
- Glance at the menu bar icon (A or K)
- Verify you're in the mode you expect
- Toggle if needed

### 4. Don't Manually Edit settings.json

While KiroToggle is managing your modes:
- Avoid manually editing `~/.claude/settings.json`
- Let KiroToggle handle the `env` variables
- Other settings are safe to edit

If you must manually edit:
1. Switch to Anthropic mode first
2. Make your changes
3. Toggle to Kiro mode if needed (creates fresh backup)

### 5. Monitor Gateway Health (Kiro Mode)

If you're in Kiro mode and see issues:

```bash
# Check if gateway is running
curl http://127.0.0.1:8000/health
# Should return: {"status": "healthy"}

# Check gateway process
pgrep -f "gateway/main.py"
# Should show a process ID

# View gateway logs
ls ~/path/to/gateway/
```

## Common Workflows

### Daily Developer Workflow

**Morning**:
1. Log into macOS (KiroToggle auto-starts)
2. Menu bar shows "K" (you left it in Kiro mode yesterday)
3. Gateway is already running
4. Start Claude Code → immediately connected to gateway

**End of day**:
1. Finish work in Claude Code
2. No need to toggle - KiroToggle stays in Kiro mode
3. Quit or leave it running

### Troubleshooting a Bug (Need Fresh Environment)

**Problem**: Claude Code behaving strangely, want to rule out gateway issues.

**Steps**:
1. Click KiroToggle → "Toggle Mode" (switch to Anthropic)
2. Restart Claude Code
3. Test if issue persists
4. If solved → gateway was the issue
5. If persists → problem is elsewhere
6. Toggle back to Kiro mode when done

### Switching Between Projects

**Scenario**: One project uses Anthropic, another uses Kiro.

**Workflow**:
1. Check menu bar - see current mode
2. Toggle if switching projects
3. Restart Claude Code
4. Work on the project
5. Toggle again when switching back

### First Time Testing Kiro

**Steps**:
1. Ensure config.json has your API key
2. Click Toggle Mode (A → K)
3. Wait for notification
4. Start a new Claude Code session
5. Send a test message
6. Check that it works

If you want to go back:
1. Click Toggle Mode (K → A)
2. Restart Claude Code
3. Back to normal Anthropic behavior

## Summary

**Key Points**:
- ✅ Menu bar icon shows current mode (A/K)
- ✅ One-click toggle handles everything automatically
- ✅ Must restart Claude Code after toggling
- ✅ State persists across restarts
- ✅ Auto-start available for convenience
- ✅ Safe to toggle back and forth anytime

**Remember**:
- Check the icon before starting work
- Restart Claude Code after every toggle
- Use auto-start if you toggle frequently
- KiroToggle handles all the complexity for you

For more information, see the main [README.md](../README.md) or open an issue on GitHub.
