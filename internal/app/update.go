package app

import (
	"fmt"
	"image"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"tooie-shelf/internal/config"
	"tooie-shelf/internal/graphics"
	"tooie-shelf/internal/sys"
)

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		queryTerminal,
		loadIcons(m.DisplayApps),
	)
}

// Update handles events and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// Only set dimensions on first receive, ignore resizes (e.g., soft keyboard)
		if m.TermWidth == 0 && m.TermHeight == 0 {
			m.TermWidth = msg.Width
			m.TermHeight = msg.Height
			return m, queryTerminal
		}
		// Ignore subsequent resize events to prevent redraws
		return m, nil

	case terminalGeometryMsg:
		// Debounce: only update if dimensions actually changed
		if m.CellPx.Width == msg.CellDim.Width &&
			m.CellPx.Height == msg.CellDim.Height &&
			m.Ready {
			return m, nil
		}
		m.CellPx = msg.CellDim
		m.Ready = true
		m.ClearCache()
		m.SixelsDrawn = false // Force sixel redraw at new positions
		return m, tea.ClearScreen

	case iconsLoadedMsg:
		m.Icons = msg.Icons

	case tea.MouseMsg:
		// Only handle release events, ignore press/motion to avoid extra redraws
		if msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		index := m.HitTest(msg.X, msg.Y)
		if index >= 0 && index < len(m.DisplayApps) {
			// Flash visual feedback directly via ANSI (no View() redraw)
			m.flashCell(index)

			app := m.DisplayApps[index]
			if app.IsCommand() {
				// Run command/script/binary
				go sys.RunCommand(app.Command)
			} else {
				// Launch Android app
				go sys.LaunchApp(app.Package, app.Activity)
			}

			if m.Config.Behavior.CloseOnLaunch {
				return m, tea.Quit
			}
		}
		return m, nil
	}

	return m, nil
}

// terminalGeometryMsg carries terminal pixel dimensions.
type terminalGeometryMsg struct {
	CellDim sys.CellDim
}

// iconsLoadedMsg carries loaded icon images.
type iconsLoadedMsg struct {
	Icons []image.Image
}

// queryTerminal queries terminal geometry.
func queryTerminal() tea.Msg {
	geom, err := sys.GetTerminalGeometry()
	if err != nil {
		// Use fallback dimensions
		return terminalGeometryMsg{
			CellDim: sys.CellDim{Width: 10, Height: 20},
		}
	}
	return terminalGeometryMsg{CellDim: geom.CellDim}
}

// loadIcons loads all icon images for the display apps in parallel.
// Icon sources (in priority order):
// 1. User-specified Dashboard Icons (icon: "dashboard:icon-name")
// 2. User-specified URL (icon: "https://...")
// 3. User-specified local file path
// 4. Cached/extracted APK icon (if package specified and no user icon)
// 5. Placeholder (fallback)
func loadIcons(apps []config.AppConfig) tea.Cmd {
	return func() tea.Msg {
		type iconResult struct {
			index int
			img   image.Image
		}

		icons := make([]image.Image, len(apps))
		resultChan := make(chan iconResult, len(apps))

		// Launch goroutines for parallel loading
		for i, app := range apps {
			go func(index int, app config.AppConfig) {
				img := loadSingleIcon(app)
				resultChan <- iconResult{index: index, img: img}
			}(i, app)
		}

		// Collect results
		for range apps {
			result := <-resultChan
			icons[result.index] = result.img
		}

		return iconsLoadedMsg{Icons: icons}
	}
}

// loadSingleIcon loads a single icon for an app.
func loadSingleIcon(app config.AppConfig) image.Image {
	var img image.Image
	var err error

	// Priority 1, 2, 3: User-specified icon takes priority
	if app.Icon != "" {
		switch {
		// Dashboard Icons: "dashboard:icon-name"
		case strings.HasPrefix(app.Icon, "dashboard:"):
			iconName := strings.TrimPrefix(app.Icon, "dashboard:")
			img, err = graphics.FetchDashboardIcon(iconName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to fetch dashboard icon '%s': %v\n", iconName, err)
			}

		// Direct URL: "https://..."
		case strings.HasPrefix(app.Icon, "http://") || strings.HasPrefix(app.Icon, "https://"):
			img, err = graphics.FetchIconFromURL(app.Icon)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to fetch icon from URL '%s': %v\n", app.Icon, err)
			}

		// Local file path
		default:
			img, err = graphics.LoadImage(app.Icon)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load icon '%s': %v\n", app.Icon, err)
			}
		}
	}

	// Priority 4: If no user-specified icon loaded, try APK extraction (uses cache)
	if img == nil && app.Package != "" {
		img, err = graphics.ExtractAPKIcon(app.Package)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to extract icon for %s: %v\n", app.Name, err)
		}
	}

	// Fallback to placeholder
	if img == nil {
		img = graphics.CreatePlaceholder(64, 64)
	}

	return img
}

// flashCell provides visual feedback by briefly highlighting the cell border.
// Uses direct ANSI output to avoid triggering a full View() redraw.
func (m *Model) flashCell(index int) {
	if !m.Config.Style.Border {
		return
	}

	cellW, cellH := m.GridCellSize()
	if cellW <= 0 || cellH <= 0 {
		return
	}

	col := index % m.Config.Grid.Columns
	row := index / m.Config.Grid.Columns

	// Calculate top-left position of the cell (1-indexed for ANSI)
	startX := col*cellW + 1
	startY := row*cellH + 1

	// Get highlight color from config
	highlightColor := m.Config.GetHighlightColor()
	highlight := fmt.Sprintf("\x1b[38;5;%sm", highlightColor)
	reset := "\x1b[0m"

	// Rounded border characters
	topLeft := "╭"
	topRight := "╮"
	bottomLeft := "╰"
	bottomRight := "╯"
	horizontal := "─"
	vertical := "│"

	var output string

	// Top border
	output += fmt.Sprintf("\x1b[%d;%dH%s%s", startY, startX, highlight, topLeft)
	for x := 1; x < cellW-1; x++ {
		output += horizontal
	}
	output += topRight

	// Side borders
	for y := 1; y < cellH-1; y++ {
		output += fmt.Sprintf("\x1b[%d;%dH%s", startY+y, startX, vertical)
		output += fmt.Sprintf("\x1b[%d;%dH%s", startY+y, startX+cellW-1, vertical)
	}

	// Bottom border
	output += fmt.Sprintf("\x1b[%d;%dH%s", startY+cellH-1, startX, bottomLeft)
	for x := 1; x < cellW-1; x++ {
		output += horizontal
	}
	output += bottomRight + reset

	// Move cursor to bottom
	output += fmt.Sprintf("\x1b[%d;1H", m.TermHeight)

	// Write directly to stdout
	fmt.Fprint(os.Stdout, output)

	// Schedule reset to normal border after delay
	go func() {
		time.Sleep(150 * time.Millisecond)
		m.drawNormalBorder(index)
	}()
}

// drawNormalBorder draws the normal border color for a cell via direct ANSI.
func (m *Model) drawNormalBorder(index int) {
	if !m.Config.Style.Border {
		return
	}

	cellW, cellH := m.GridCellSize()
	if cellW <= 0 || cellH <= 0 {
		return
	}

	col := index % m.Config.Grid.Columns
	row := index / m.Config.Grid.Columns

	startX := col*cellW + 1
	startY := row*cellH + 1

	// Get normal border color from config
	borderColor := m.Config.GetBorderColor()
	color := fmt.Sprintf("\x1b[38;5;%sm", borderColor)
	reset := "\x1b[0m"

	topLeft := "╭"
	topRight := "╮"
	bottomLeft := "╰"
	bottomRight := "╯"
	horizontal := "─"
	vertical := "│"

	var output string

	// Top border
	output += fmt.Sprintf("\x1b[%d;%dH%s%s", startY, startX, color, topLeft)
	for x := 1; x < cellW-1; x++ {
		output += horizontal
	}
	output += topRight

	// Side borders
	for y := 1; y < cellH-1; y++ {
		output += fmt.Sprintf("\x1b[%d;%dH%s", startY+y, startX, vertical)
		output += fmt.Sprintf("\x1b[%d;%dH%s", startY+y, startX+cellW-1, vertical)
	}

	// Bottom border
	output += fmt.Sprintf("\x1b[%d;%dH%s", startY+cellH-1, startX, bottomLeft)
	for x := 1; x < cellW-1; x++ {
		output += horizontal
	}
	output += bottomRight + reset

	output += fmt.Sprintf("\x1b[%d;1H", m.TermHeight)
	fmt.Fprint(os.Stdout, output)
}
