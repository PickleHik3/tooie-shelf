package app

import (
	"image"

	"tooie-shelf/internal/config"
	"tooie-shelf/internal/graphics"
	"tooie-shelf/internal/sys"
)

// Model represents the application state.
type Model struct {
	Config      config.Config
	DisplayApps []config.AppConfig // Apps in display order
	TermWidth   int                // Terminal columns
	TermHeight  int                // Terminal rows
	CellPx      sys.CellDim        // Pixel dimensions per cell

	Icons      []image.Image                  // Original high-res images
	SixelCache map[string]graphics.SixelResult // Cached sixel data with dimensions

	ErrorFlash []bool // Per-app error indicator
	Selected   int    // Currently selected app index (-1 for none)

	Ready           bool // Terminal geometry acquired
	NeedsFullRedraw bool // When true, redraw icons; when false, only redraw borders
	SixelsDrawn     bool // True if sixels have been drawn to screen (static mode)
}

// launchResultMsg carries the result of an app launch attempt.
type launchResultMsg struct {
	Index int
	Err   error
}

// NewModel creates a new launcher model.
func NewModel(cfg config.Config) Model {
	displayApps := cfg.GetDisplayApps()
	numApps := len(displayApps)

	return Model{
		Config:          cfg,
		DisplayApps:     displayApps,
		Icons:           make([]image.Image, numApps),
		SixelCache:      make(map[string]graphics.SixelResult),
		ErrorFlash:      make([]bool, numApps),
		Selected:        -1,
		Ready:           false,
		NeedsFullRedraw: true,
		SixelsDrawn:     false,
	}
}

// CacheKey generates a cache key for a sixel render.
func CacheKey(appIndex, widthCells, heightCells int) string {
	return string(rune(appIndex)) + "_" + string(rune(widthCells)) + "_" + string(rune(heightCells))
}

// ClearCache invalidates all cached sixel data.
func (m *Model) ClearCache() {
	m.SixelCache = make(map[string]graphics.SixelResult)
}

// GridCellSize calculates the size of each grid cell in terminal cells.
func (m *Model) GridCellSize() (width, height int) {
	if m.Config.Grid.Columns <= 0 || m.Config.Grid.Rows <= 0 {
		return 0, 0
	}
	width = m.TermWidth / m.Config.Grid.Columns
	// Use TermHeight - 1 to avoid bottom border being cut off
	height = (m.TermHeight - 1) / m.Config.Grid.Rows
	return
}

// IconCellSize calculates the available space for icons within a cell.
func (m *Model) IconCellSize() (width, height int) {
	cellW, cellH := m.GridCellSize()

	// Subtract padding and borders
	padding := m.Config.Style.Padding
	borderSize := 0
	if m.Config.Style.Border {
		borderSize = 2 // 1 char on each side
	}

	width = cellW - 2*padding - borderSize
	height = cellH - 2*padding - borderSize

	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return
}

// HitTest returns the app index at the given terminal coordinates, or -1 if none.
func (m *Model) HitTest(x, y int) int {
	cellW, cellH := m.GridCellSize()
	if cellW <= 0 || cellH <= 0 {
		return -1
	}

	col := x / cellW
	row := y / cellH

	if col < 0 || col >= m.Config.Grid.Columns {
		return -1
	}
	if row < 0 || row >= m.Config.Grid.Rows {
		return -1
	}

	index := row*m.Config.Grid.Columns + col
	if index >= len(m.DisplayApps) {
		return -1
	}

	return index
}

// GetIconScale returns the icon scale for the app at the given display index.
func (m *Model) GetIconScale(index int) float64 {
	if index < 0 || index >= len(m.DisplayApps) {
		return 1.0
	}
	return m.Config.GetIconScale(m.DisplayApps[index])
}
