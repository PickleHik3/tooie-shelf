# AGENTS.md - tooie-appsbar-go

Guide for AI agents working in this codebase.

## Project Overview

A terminal-based app bar for Android (Termux) that displays app icons using Sixel graphics. Built with Go and the Bubble Tea TUI framework.

**Key Technologies:**
- Go 1.21+
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling/layout
- [go-sixel](https://github.com/mattn/go-sixel) - Sixel graphics encoding
- Sixel-capable terminal required for image display

## Essential Commands

```bash
# Build (optimized binary)
go build -ldflags="-s -w" -o tooie-appsbar ./cmd/launcher

# Run
go run ./cmd/launcher

# Get dependencies
go mod tidy

# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Project Structure

```
tooie-appsbar-go/
├── cmd/launcher/main.go          # Entry point
├── internal/
│   ├── app/
│   │   ├── model.go              # Bubble Tea model (state)
│   │   ├── update.go             # Event handlers (keys, mouse, resize)
│   │   └── view.go               # Render logic (layout, sixel overlay)
│   ├── config/
│   │   ├── config.go             # Config structs and defaults
│   │   └── loader.go             # YAML loading and validation
│   ├── graphics/
│   │   ├── sixel.go              # Image -> Sixel encoding
│   │   └── scaler.go             # Image scaling (CatmullRom/Lanczos)
│   └── sys/
│       ├── android.go            # Android app launching (am command)
│       └── terminal.go           # Terminal geometry via ioctl
├── go.mod
├── README.md                     # User documentation
└── plan.md                       # Original implementation plan
```

## Code Patterns & Conventions

### Bubble Tea Architecture

The app follows the Bubble Tea Model-Update-View pattern:

1. **Model** (`model.go`): Holds all state including:
   - `Config` - loaded from YAML
   - `DisplayApps` - apps in display order
   - `Icons []image.Image` - original high-res icon images
   - `SixelCache` - cached sixel strings keyed by dimensions
   - `CellPx` - terminal cell pixel dimensions (critical for sizing)

2. **Update** (`update.go`): Handles:
   - `tea.KeyMsg` - q/esc/ctrl+c to quit
   - `tea.WindowSizeMsg` - terminal resize (only processed once)
   - `tea.MouseMsg` - click detection via `HitTest()`
   - Custom messages for async operations

3. **View** (`view.go`): Two-pass rendering:
   - Pass 1: Render borders/frames with Lipgloss
   - Pass 2: Overlay sixel images at absolute cursor positions

### Key Implementation Details

**Terminal Geometry:**
- Uses `unix.IoctlGetWinsize` to get pixel dimensions
- Calculates `CellDim` (pixels per character cell)
- Critical for converting cell-based layout to pixel-based sixel sizing

**Sixel Rendering:**
- Images scaled using `ScaleImageAspectFit` (preserves aspect ratio)
- Uses CatmullRom interpolation (Lanczos-like quality)
- Cached by dimensions to avoid re-encoding on every frame
- Positioned using ANSI cursor positioning (`\x1b[row;colH`)

**Visual Feedback:**
- Cell flash on click uses direct ANSI output (not View() redraw)
- Avoids expensive re-render for simple visual feedback
- See `flashCell()` in `update.go`

**Resize Handling:**
- Only processes first `WindowSizeMsg` (ignores soft keyboard popups)
- Clears sixel cache on resize to regenerate at new dimensions

## Configuration

Config file: `~/.config/tooie-appsbar-go/config.yaml`

```yaml
# Display order - only these apps shown, in this order
display:
  - Chrome
  - Files

grid:
  rows: 2
  columns: 4

style:
  border: true
  padding: 1
  icon_scale: 0.8  # Global scale 0.1-1.0

behavior:
  close_on_launch: true

apps:
  - name: Chrome
    icon: /path/to/chrome.png
    package: com.android.chrome

  - name: Files
    icon: /path/to/files.png
    package: com.google.android.apps.nbu.files
    activity: com.google.android.apps.nbu.files.home.HomeActivity

  - name: Htop
    icon: /path/to/htop.png
    command: htop  # Linux command instead of Android app
```

**App Types:**
- Android app: specify `package` (and optional `activity`)
- Linux command: specify `command` (takes priority)
- Per-app `icon_scale` overrides global setting

## Important Gotchas

1. **Sixel strings have no visible width** - Lipgloss can't measure them. Always set explicit `Width`/`Height` on Lipgloss styles containing sixel content.

2. **Resize events flood on soft keyboard** - The app ignores resize events after initial setup to prevent constant redraws when Android keyboard appears/disappears.

3. **Mouse coordinates are in cells, not pixels** - `HitTest()` converts cell coordinates to grid position using `GridCellSize()`.

4. **Icon paths with `~`** - The config loader expands `~` to home directory automatically.

5. **Android app launching** - Uses `am start` command. Requires Termux with `termux-api` or proper Android permissions.

6. **Sixel cache key** - Must include scale: `fmt.Sprintf("%d_%d_%d_%.2f", index, widthCells, heightCells, scale)`

## Testing Approach

This project has no automated tests. Manual testing:

1. Build and run in Termux
2. Verify icons display at correct aspect ratio
3. Click icons - should flash border and launch app
4. Resize terminal - icons should regenerate (once)
5. Test with/without borders, different padding values
6. Test `close_on_launch: true` behavior

## Dependencies

From `go.mod`:
- `github.com/charmbracelet/bubbletea` - TUI runtime
- `github.com/charmbracelet/lipgloss` - Layout/styling
- `github.com/mattn/go-sixel` - Sixel encoding
- `golang.org/x/image` - Image scaling algorithms
- `golang.org/x/sys` - Unix ioctl for terminal geometry
- `gopkg.in/yaml.v3` - Config parsing

## Common Tasks

**Add a new config option:**
1. Add field to struct in `internal/config/config.go`
2. Update `DefaultConfig()` if needed
3. Use in `internal/app/` files

**Modify rendering:**
- Layout calculations: `model.go` (`GridCellSize`, `IconCellSize`)
- Border/frame rendering: `view.go` `renderCellFrame()`
- Sixel positioning: `view.go` second pass in `View()`

**Add new interaction:**
- Add handler in `update.go` `Update()` method
- For mouse: use `m.HitTest(x, y)` to get app index
- For visual feedback without full redraw: use direct ANSI (see `flashCell()`)

**Change icon scaling:**
- Algorithm: `internal/graphics/scaler.go`
- Aspect fit logic: `ScaleImageAspectFit()`
- Quality: Currently uses `draw.CatmullRom` (change to `draw.NearestNeighbor` for speed, `draw.BiLinear` for balance)
