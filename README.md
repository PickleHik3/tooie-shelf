# Tooie Shelf

A terminal-based app bar for Android (Termux) that displays app icons using Sixel graphics.

## Requirements

- Termux with a Sixel-capable terminal
- Go 1.21+

## Build

```bash
go build -ldflags="-s -w" -o tooie-shelf ./cmd/launcher
```

## Configuration

Config file: `~/.config/tooie-shelf/config.yaml`

```yaml
# Display order - only these apps shown, in this order
display:
  - Chrome
  - Files
  - WhatsApp

grid:
  rows: 2
  columns: 4

style:
  border: true
  padding: 1
  icon_scale: 0.8              # Global icon scale (0.1-1.0)
  border_color: "240"          # Normal border color (ANSI 256)
  highlight_color: "96"        # Click highlight color (ANSI 256)

behavior:
  close_on_launch: true

apps:
  # Android app
  - name: Chrome
    icon: /path/to/chrome.png
    package: com.android.chrome

  # Android app with specific activity
  - name: Files
    icon: /path/to/files.png
    package: com.google.android.apps.nbu.files
    activity: com.google.android.apps.nbu.files.home.HomeActivity

  # Linux command
  - name: Htop
    icon: /path/to/htop.png
    command: htop

  # Script or binary
  - name: Backup
    icon: /path/to/backup.png
    command: ~/scripts/backup.sh

  # Command with arguments
  - name: SSH Server
    icon: /path/to/terminal.png
    command: sshd -D
```

### Options

| Field | Description |
|-------|-------------|
| `display` | App names in display order (if empty, show all apps) |
| `grid.rows` | Number of rows in the grid |
| `grid.columns` | Number of columns in the grid |
| `style.border` | Show borders around cells |
| `style.padding` | Padding inside cells (in characters) |
| `style.icon_scale` | Global icon scale 0.1-1.0 (default: 1.0) |
| `style.border_color` | Normal border color - ANSI 256 color code or "default" (default: "240") |
| `style.highlight_color` | Click highlight color - ANSI 256 color code or "default" (default: "96") |
| `behavior.close_on_launch` | Exit after launching an app (default: false) |
| `apps[].name` | Display name (used for display order matching) |
| `apps[].icon` | Path to icon image (PNG, JPG, GIF) |
| `apps[].package` | Android package name (for Android apps) |
| `apps[].activity` | Optional: specific Android activity to launch |
| `apps[].command` | Linux command/script/binary (takes priority over package) |
| `apps[].icon_scale` | Per-app icon scale override (0.1-1.0) |

## Usage

```bash
./tooie-shelf
```

- Touch an icon to launch the app
- Press `q` or `Esc` to quit

## Version

0.1

## Features

- **Sixel graphics** - High-quality icon display in supported terminals
- **Zero flicker** - Static sixel rendering with direct ANSI border feedback
- **Customizable colors** - Configurable border and highlight colors (ANSI 256)
- **Android + Linux support** - Launch Android apps or Linux commands/scripts
- **Flexible layout** - Configurable grid, padding, and icon scaling
- **Soft keyboard friendly** - Debounced resize handling prevents redraws

## Roadmap

- [x] Icon scale option (global and per-app)
- [x] Display order configuration
- [x] Linux command/script/binary support
- [x] Customizable border/highlight colors
- [x] Flicker-free rendering for home launcher use
