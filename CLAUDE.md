# Kiront

A macOS menu bar app that toggles Claude Code between Anthropic and Kiro modes with one click.
Written in Go. No dock icon (`LSUIElement = true`).

## Project structure

| File | Purpose |
|---|---|
| `main.go` | Menu bar UI and application entry point |
| `internal/config/` | Configuration management |
| `internal/settings/` | Claude Code settings backup/restore |
| `internal/gateway/` | Gateway lifecycle management |
| `internal/controller/` | Main controller coordinating all managers |
| `internal/process/` | Claude Code process detection |
| `internal/autostart/` | macOS auto-start support |
| `gateway/` | Bundled Kiro gateway Python code |
| `Info.plist` | App bundle metadata |
| `Makefile` | Build targets |
| `docs/index.html` | Landing page template — contains `{{VERSION}}` placeholder |
| `.github/workflows/release.yml` | Release pipeline |

## Build & local dev workflow

```bash
make build          # compile + assemble Kiront.app in the project root
make install        # build → ad-hoc sign → install to ~/Applications → launch
make test           # run tests
make clean          # remove binary, Kiront.app, and build artifacts
make run            # build and run
```

`make install` clears the Gatekeeper quarantine flag, so the app opens without
needing "Open Anyway" or right-click → Open. Use this during development.

Requires macOS 10.15+ and Go 1.21+.

## Run tests

```bash
go test ./... -v
```

## Release workflow

### When to create a new release

**Claude/AI assistants should automatically create and push a version tag when:**

1. **New features are added** → Bump minor version (v0.1.0 → v0.2.0)
   - Examples: new menu items, new modes, new configuration options, new auto-start features

2. **Bug fixes or improvements** → Bump patch version (v0.1.0 → v0.1.1)
   - Examples: crash fixes, settings restore bugs, gateway startup issues, UI fixes

3. **Breaking changes** → Bump major version (v0.1.0 → v1.0.0)
   - Examples: config file format changes, API changes, removal of features

4. **Documentation-only changes** → No version bump needed
   - Examples: README updates, comment changes, CLAUDE.md updates

### How to tag and release

After committing changes that warrant a release:

```bash
# Determine the next version based on the changes
# Current version: v0.1.0
# For feature: v0.2.0
# For bugfix: v0.1.1
# For breaking change: v1.0.0

git tag v0.x.x
git push origin v0.x.x
```

GitHub Actions takes it from there:
1. Builds `Kiront.app` on a macOS runner
2. Packages `Kiront.dmg`
3. Injects the version into the landing page and deploys it to GitHub Pages
4. Creates a GitHub Release and attaches `Kiront.dmg`

**Important:** `docs/index.html` contains a `{{VERSION}}` placeholder that
Actions replaces at deploy time. Do not manually edit the version string in
that file — Actions processes it at deploy time into `_site/index.html`; `docs/index.html` always stays as the `{{VERSION}}` template.

### Checking current version

```bash
git describe --tags --abbrev=0
```

## GitHub Pages

The landing page is served from GitHub Pages.
Source must be set to **"GitHub Actions"** in repo Settings → Pages.

## Configuration

On first run, Kiront creates `~/.kiront/config.json` with:
- `kiro_api_key`: Kiro API key
- `gateway_port`: Gateway server port (default: 8000)
- `kiro_cli_db_path`: Path to Kiro CLI database
- `current_mode`: Current mode ("anthropic" or "kiro")
- `auto_start`: Auto-launch on macOS login
