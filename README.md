# KiroToggle

KiroToggle is a macOS menu bar application that simplifies switching between two Claude Code operational modes with a single click:

- **Anthropic Mode**: Standard Claude Code connecting directly to Anthropic's API
- **Kiro Mode**: Claude Code routing through a local Kiro gateway for enhanced functionality

## Why KiroToggle?

Manually switching between modes requires multiple error-prone steps:
1. Starting/stopping the gateway server
2. Modifying `~/.claude/settings.json` environment variables
3. Managing Python virtual environments
4. Restarting Claude Code sessions

KiroToggle automates this entire workflow into a one-click menu bar toggle.

## Features

- **One-Click Mode Switching**: Toggle between Anthropic and Kiro modes instantly
- **Automatic Gateway Management**: Handles gateway setup, start, and stop automatically
- **Safe Configuration Management**: Backs up and restores settings.json reliably
- **Persistent State**: Remembers current mode across app restarts
- **Crash Recovery**: Detects and recovers from inconsistent states after crashes
- **Auto-Start on Login**: Optional automatic launch when macOS starts
- **Visual Feedback**: Clear menu bar indication of which mode is active (A = Anthropic, K = Kiro)
- **Claude Code Detection**: Warns when Claude Code sessions need restart after mode change

## Requirements

- **macOS**: 10.15 (Catalina) or later
- **Python 3.10+**: Required for the Kiro gateway
- **Claude Code**: Installed and configured

## Installation

### Build from Source

```bash
# Clone the repository
cd /path/to/kiront

# Build the application
make build

# Install to ~/bin (optional)
make install

# Run the application
./bin/kirotoggle
```

The first time you run KiroToggle, it will create the necessary configuration files in `~/.kirotoggle/`.

## Configuration

KiroToggle stores its configuration in `~/.kirotoggle/config.json`:

```json
{
  "kiro_api_key": "your-api-key",
  "gateway_port": 8000,
  "kiro_cli_db_path": "/Users/username/Library/Application Support/kiro-cli/data.sqlite3",
  "current_mode": "anthropic",
  "auto_start": true
}
```

### Configuration Fields

- **kiro_api_key**: Your Kiro API key (must match the key used in Claude Code when in Kiro mode)
- **gateway_port**: Port for the gateway server (default: 8000)
- **kiro_cli_db_path**: Path to the Kiro CLI database file
- **current_mode**: Current operational mode ("anthropic" or "kiro")
- **auto_start**: Whether to launch KiroToggle on macOS login

### Important: API Key Matching

The `kiro_api_key` in KiroToggle's config must match the `ANTHROPIC_API_KEY` that Claude Code will use when in Kiro mode. This ensures proper authentication between Claude Code and the gateway.

## Usage

### Basic Operation

1. **Launch KiroToggle**: The app appears in your macOS menu bar as "A" (Anthropic mode) or "K" (Kiro mode)
2. **Click the menu bar icon**: Opens the menu with options:
   - **Toggle Mode**: Switch between Anthropic and Kiro modes
   - **Start at Login**: Enable/disable auto-start on macOS login
   - **Quit**: Exit KiroToggle

### Toggling Modes

**To Kiro Mode:**
1. Click "Toggle Mode"
2. KiroToggle automatically:
   - Backs up your `~/.claude/settings.json`
   - Sets up the gateway Python environment (if needed)
   - Starts the gateway server
   - Modifies Claude Code settings to route through the gateway
   - Updates menu bar icon to "K"

**To Anthropic Mode:**
1. Click "Toggle Mode"
2. KiroToggle automatically:
   - Stops the gateway server
   - Restores your original `~/.claude/settings.json`
   - Updates menu bar icon to "A"

### After Toggling

**Important**: Claude Code does not hot-reload environment variables. After toggling modes, you must restart any running Claude Code sessions for the change to take effect.

KiroToggle will show a notification if it detects running Claude Code processes:
> ⚠️ Please restart Claude Code sessions for changes to take effect.

### Auto-Start on Login

To have KiroToggle launch automatically when you log into macOS:

1. Click the KiroToggle menu bar icon
2. Check "Start at Login"
3. The app will now launch on every macOS login and restore your last active mode

## What Happens Under the Hood

### When Switching to Kiro Mode

1. **Backup**: `~/.claude/settings.json` → `~/.kirotoggle/settings.backup.json`
2. **Gateway Setup**: Create Python venv in `gateway/.venv` (if not exists), install dependencies
3. **Gateway Start**: Launch `gateway/main.py`, wait for health check at `http://127.0.0.1:8000/health`
4. **Settings Update**: Inject environment variables into `~/.claude/settings.json`:
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

### When Switching to Anthropic Mode

1. **Gateway Stop**: Send SIGTERM to gateway process, wait for graceful shutdown
2. **Settings Restore**: Copy `~/.kirotoggle/settings.backup.json` → `~/.claude/settings.json`
3. **Config Update**: Save current mode as "anthropic"

## Troubleshooting

### KiroToggle won't start

**Problem**: Application crashes or doesn't launch

**Solutions**:
- Check console logs: `Console.app` → filter for "kirotoggle"
- Verify macOS version: requires 10.15+
- Check permissions: ensure the app can access `~/.claude/` and `~/.kirotoggle/`

### Gateway fails to start

**Problem**: Toggle to Kiro mode fails with gateway error

**Solutions**:
- Verify Python 3.10+ is installed: `python3 --version`
- Check if port 8000 is already in use: `lsof -i :8000`
- Manually test gateway setup:
  ```bash
  cd gateway
  python3 -m venv .venv
  .venv/bin/pip install -r requirements.txt
  .venv/bin/python main.py
  ```
- Check gateway logs in `gateway/` directory

### Settings not restored correctly

**Problem**: After switching to Anthropic mode, settings seem wrong

**Solutions**:
- Check if backup exists: `ls ~/.kirotoggle/settings.backup.json`
- If backup is missing, you'll need to manually recreate your Claude Code settings
- KiroToggle always creates a fresh backup before each switch to Kiro mode

### Mode indicator stuck

**Problem**: Menu bar shows "K" but gateway isn't running (or vice versa)

**Solutions**:
- Restart KiroToggle - it has crash recovery that will detect and fix inconsistent state
- Check gateway process: `pgrep -f "gateway/main.py"`
- Manually stop gateway if orphaned: `pkill -f "gateway/main.py"`

### Claude Code still using wrong mode after toggle

**Problem**: Toggled mode but Claude Code behavior hasn't changed

**Solutions**:
- **Critical**: You must restart Claude Code sessions after toggling
- Exit all Claude Code terminals/processes
- Verify settings were updated: `cat ~/.claude/settings.json | grep ANTHROPIC`
- Check that no other process modified settings.json

### Auto-start not working

**Problem**: KiroToggle doesn't launch on login

**Solutions**:
- Check if plist exists: `ls ~/Library/LaunchAgents/com.sparkreel.kirotoggle.plist`
- Verify executable path in plist matches installed location
- Manually load plist: `launchctl load ~/Library/LaunchAgents/com.sparkreel.kirotoggle.plist`

## File Locations

- **Application**: `./bin/kirotoggle` (or wherever you installed it)
- **Configuration**: `~/.kirotoggle/config.json`
- **Settings Backup**: `~/.kirotoggle/settings.backup.json`
- **Gateway PID**: `~/.kirotoggle/gateway.pid`
- **Claude Code Settings**: `~/.claude/settings.json`
- **Gateway Code**: Bundled within the app at `gateway/` (relative to executable)
- **LaunchAgent**: `~/Library/LaunchAgents/com.sparkreel.kirotoggle.plist` (when auto-start enabled)

## Building and Development

### Build Commands

```bash
# Build binary
make build

# Run tests
make test

# Install to ~/bin
make install

# Clean build artifacts
make clean

# Build and run
make run
```

### Development

The codebase is organized as follows:
- `main.go`: Menu bar UI and application entry point
- `internal/config/`: Configuration management
- `internal/settings/`: Claude Code settings backup/restore
- `internal/gateway/`: Gateway lifecycle management
- `internal/controller/`: Main controller coordinating all managers
- `internal/process/`: Claude Code process detection
- `internal/autostart/`: macOS auto-start support
- `gateway/`: Bundled Kiro gateway Python code

## Safety Features

- **Atomic Writes**: All file operations use atomic writes (temp file + rename)
- **Backup Before Modify**: Settings are backed up before every modification
- **Mutex Protection**: Concurrent mode switches are prevented with mutex locks
- **Crash Recovery**: Detects orphaned processes and inconsistent state on startup
- **Graceful Shutdown**: Gateway processes are cleanly stopped with SIGTERM
- **Error Handling**: Failed operations are rolled back to maintain consistency

## License

See LICENSE file for details.

## Support

For issues, questions, or contributions, please open an issue on the GitHub repository.
