# Kiront Design

## Intent

Kiront is a macOS menu bar application that simplifies switching between two Claude Code operational modes: standard Anthropic API mode and Kiro mode (which routes through a local gateway). Currently, switching requires manual steps including activating Python environments, starting gateway servers, and modifying configuration files. This app reduces the entire workflow to single-click toggles.

## Why Now

The current manual process is time-consuming and error-prone. Each switch involves:
1. Pulling gateway updates
2. Setting up Python virtual environments
3. Starting/stopping the gateway server
4. Backing up and modifying `~/.claude/settings.json`
5. Managing environment variables

This happens frequently enough that the friction adds up. A dedicated app eliminates the context switching cost.

## Goals

- **One-click mode switching**: Toggle between Anthropic and Kiro modes with a single menu bar click
- **Gateway lifecycle management**: Automatically pull, setup, start, and stop the gateway as needed
- **Safe configuration management**: Back up and restore `~/.claude/settings.json` reliably
- **Persistent state**: Remember current mode across app restarts
- **Visual feedback**: Clear indication of which mode is active

## Non-Goals

- **Multi-gateway support**: Only one gateway configuration needed
- **Cross-platform**: macOS only (user is on macOS)
- **Claude Code integration beyond config**: We only modify settings.json, not integrating deeper into Claude Code
- **Gateway development/debugging tools**: This is strictly for toggling, not developing the gateway

## Anti-Goals (what would make this a failure)

- **Data loss**: Corrupting or losing the original `~/.claude/settings.json`
- **Orphaned processes**: Leaving gateway processes running after app quit or crash
- **Silent failures**: User thinks they're in Kiro mode but gateway isn't running
- **Brittle assumptions**: Hardcoding paths or behaviors that break if the gateway structure changes

## Constraints

- Gateway will be bundled within the kiront repository (not external)
- Gateway requires Python venv and runs via `python main.py`
- Gateway needs `.env` file with: PROXY_API_KEY, ACCOUNT_SYSTEM=true, KIRO_CLI_DB_FILE path
- Claude Code settings at `~/.claude/settings.json` must be modified atomically
- Gateway runs on port 8000 by default
- **Claude Code does not hot-reload environment variables** - must detect running Claude Code sessions and warn user to restart them after toggling modes
- App should auto-start on macOS login and remember the last active mode

## Technical Decisions

### Platform and Language
**Go** for the implementation. Go provides:
- Native macOS menu bar support via libraries like `github.com/getlantern/systray`
- Simple process management for the Python gateway
- Easy JSON manipulation for settings.json
- Single-binary distribution

### Architecture

**Menu Bar App** with three states:
1. **Anthropic Mode** (default): Gateway stopped, original settings.json active
2. **Transitioning**: Visual feedback during mode switch
3. **Kiro Mode**: Gateway running, settings.json modified with Kiro variables

**Components:**
- **Menu Bar UI**: Shows current mode, offers toggle action, access to settings
- **Settings Store**: Persists user configuration (API key, port, paths)
- **Gateway Manager**: Handles git pull, venv setup, process start/stop
- **Config Manager**: Backs up, modifies, and restores settings.json
- **State Manager**: Tracks current mode, persists across restarts

### Settings Storage

The app stores its own configuration in `~/.kiront/config.json`:
```json
{
  "kiro_api_key": "s13b45d1c-87499akj694cubHjb-k8ujbli8j",
  "gateway_port": 8000,
  "kiro_cli_db_path": "/Users/yuri/Library/Application Support/kiro-cli/data.sqlite3",
  "current_mode": "anthropic",
  "auto_start": true
}
```

Gateway `.env` file will contain:
```
PROXY_API_KEY=s13b45d1c-87499akj694cubHjb-k8ujbli8j
ACCOUNT_SYSTEM=true
KIRO_CLI_DB_FILE=/Users/yuri/Library/Application Support/kiro-cli/data.sqlite3
```

Note: `PROXY_API_KEY` in gateway/.env must match `ANTHROPIC_API_KEY` in Claude Code settings.json for authentication to work.

### Claude Code Configuration Management

**Example of actual settings.json structure:**
```json
{
  "hooks": { ... },
  "enabledPlugins": { ... },
  "extraKnownMarketplaces": { ... },
  "tui": "fullscreen",
  "agentPushNotifEnabled": true,
  "autoMode": { ... },
  "mcpServers": { ... },
  "defaultMode": "auto",
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8000",
    "ANTHROPIC_API_KEY": "s13b45d1c-87499akj694cubHjb-k8ujbli8j",
    "ANTHROPIC_MODEL": "claude-sonnet-4-5",
    "ANTHROPIC_SMALL_FAST_MODEL": "claude-haiku-4-5"
  }
}
```

**Backup Strategy:**
- Before first Kiro mode switch, copy `~/.claude/settings.json` to `~/.kiront/settings.backup.json`
- Restore from backup when switching back to Anthropic mode
- Re-backup each time we switch to Kiro mode (in case user manually edited settings while in Anthropic mode)

**Modification Strategy:**
- Read existing settings.json as JSON
- When switching to Kiro mode: merge in the `env` object with Kiro variables (or replace if `env` key exists)
- When switching to Anthropic mode: restore from backup completely
- Write atomically (write to temp file, then rename)

**Claude Code Process Detection:**
- Claude Code does not hot-reload environment variables (it hot-reloads some settings, but not `env`)
- After toggling modes, check if `claude` processes are running: `pgrep -f "claude"`
- If found, show notification: "⚠️ Claude Code is running. Please restart your Claude Code sessions for the mode change to take effect."
- This is informational only - don't block the toggle, just inform the user

### Gateway Lifecycle

**Bundling:**
The gateway code will be bundled within the kiront repository itself (not as an external dependency). This eliminates the need for git cloning and ensures version compatibility.

**Setup (one-time or on-demand):**
1. Gateway code is in `kiront/gateway/` directory (bundled)
2. Check if `gateway/.venv` exists
3. If not, run `python3 -m venv gateway/.venv`
4. Install dependencies: `gateway/.venv/bin/pip install -r gateway/requirements.txt`
5. Create/update `gateway/.env` file from app settings with:
   - `PROXY_API_KEY` (must match Claude Code's ANTHROPIC_API_KEY)
   - `ACCOUNT_SYSTEM=true`
   - `KIRO_CLI_DB_FILE="/Users/yuri/Library/Application Support/kiro-cli/data.sqlite3"`

**Start:**
1. Run setup if needed
2. Start process: `cd gateway && ../.venv/bin/python main.py`
3. Wait for port 8000 to be listening (TCP check)
4. Poll `GET http://127.0.0.1:8000/health` until it returns `{"status": "healthy"}`
5. Store PID for later shutdown

**Stop:**
1. Send SIGTERM to gateway process
2. Wait up to 5 seconds for graceful shutdown
3. If still running, send SIGKILL
4. Verify port 8000 is no longer listening

### Error Handling

- If gateway fails to start, show notification and revert to Anthropic mode
- If settings.json backup is missing, warn user and refuse to toggle
- If gateway process dies unexpectedly, update menu bar to show disconnected state
- All file operations use atomic writes where possible

## What's Left to the Builder

**Implementation Details:**
- Which Go menu bar library to use (systray vs. others)
- Settings UI approach (separate window, dialog, or menu-driven)
- Gateway stdout/stderr logging - gateway logs to stderr via loguru, decide whether to capture to file for debugging
- Process management details (PTY vs. pipes for gateway process)
- Icon design for menu bar (text indicator vs. custom icon)
- Update mechanism for the app itself
- Python version handling - default to `python3`, optionally check for minimum version if needed

**Edge Cases to Handle:**
- Orphaned gateway processes after app crash: search by process name or port binding on restart
- If config says "kiro" mode but gateway isn't running on restart: auto-start gateway or switch to "disconnected" state
- Port 8000 already in use: fail gracefully with clear error message
- Multiple Kiront instances: use lock file or process check to prevent
- Gateway dependency installation failures: show error and recovery steps
- settings.json malformed or read-only: detect and show clear error
- Missing settings.backup.json when restoring: warn and refuse to toggle back

**Distribution:**
- How to package (standalone binary, .app bundle, Homebrew)
- Code signing and notarization for macOS
- First-run setup flow for API key configuration
