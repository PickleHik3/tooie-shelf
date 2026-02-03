package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"tooie-appsbar-go/internal/graphics"
)

// ANSI escape codes for cursor positioning and sync output
const (
	cursorTo   = "\x1b[%d;%dH" // row;col (1-indexed)
	cursorHome = "\x1b[H"      // Move cursor to top-left
	hideCursor = "\x1b[?25l"
	syncStart  = "\x1b[?2026h" // Begin synchronized update
	syncEnd    = "\x1b[?2026l" // End synchronized update
)

// View renders the launcher UI.
func (m Model) View() string {
	if !m.Ready {
		return "Loading..."
	}

	if len(m.DisplayApps) == 0 {
		return "No apps configured. Edit ~/.config/tooie-appsbar-go/config.yaml"
	}

	cellW, cellH := m.GridCellSize()

	if cellW <= 0 || cellH <= 0 {
		return "Terminal too small"
	}

	var b strings.Builder
	b.WriteString(syncStart)
	b.WriteString(hideCursor)
	b.WriteString(cursorHome)

	// Calculate inner dimensions for lipgloss (accounting for border)
	innerW := cellW
	innerH := cellH
	if m.Config.Style.Border {
		innerW = cellW - 2 // borders take 2 chars horizontally
		innerH = cellH - 2 // borders take 2 chars vertically
	}
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	// First pass: render all borders/frames
	var rows []string
	appIndex := 0

	for row := 0; row < m.Config.Grid.Rows; row++ {
		var cells []string

		for col := 0; col < m.Config.Grid.Columns; col++ {
			var cell string
			if appIndex < len(m.DisplayApps) {
				cell = m.renderCellFrame(appIndex, innerW, innerH)
				appIndex++
			} else {
				cell = m.renderEmptyCell(innerW, innerH)
			}
			cells = append(cells, cell)
		}

		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}

	b.WriteString(lipgloss.JoinVertical(lipgloss.Left, rows...))

	// Second pass: overlay sixel images at absolute positions (only if not already drawn)
	// This ensures sixels are drawn once and persist across renders
	if !m.SixelsDrawn {
		m.drawSixelsDirectly(&b)
	}

	// Move cursor to bottom
	b.WriteString(fmt.Sprintf(cursorTo, m.TermHeight, 1))
	b.WriteString(syncEnd)

	return b.String()
}

// drawSixelsDirectly renders sixel images directly to the output.
// This is called once and the sixels persist in the terminal buffer.
func (m *Model) drawSixelsDirectly(b *strings.Builder) {
	// Check if icons are loaded
	iconsLoaded := false
	for _, icon := range m.Icons {
		if icon != nil {
			iconsLoaded = true
			break
		}
	}

	if !iconsLoaded {
		return
	}

	cellW, cellH := m.GridCellSize()
	iconW, iconH := m.IconCellSize()

	if cellW <= 0 || cellH <= 0 {
		return
	}

	appIndex := 0
	for row := 0; row < m.Config.Grid.Rows && appIndex < len(m.DisplayApps); row++ {
		for col := 0; col < m.Config.Grid.Columns && appIndex < len(m.DisplayApps); col++ {
			if appIndex < len(m.Icons) && m.Icons[appIndex] != nil {
				// Apply icon scale
				scale := m.GetIconScale(appIndex)
				scaledIconW := int(float64(iconW) * scale)
				scaledIconH := int(float64(iconH) * scale)
				if scaledIconW < 1 {
					scaledIconW = 1
				}
				if scaledIconH < 1 {
					scaledIconH = 1
				}

				sixelResult := m.getSixelContentWithDimensions(appIndex, scaledIconW, scaledIconH, scale)
				if sixelResult.Sixel != "" {
					// Calculate absolute position for this icon
					borderOffset := 0
					if m.Config.Style.Border {
						borderOffset = 1
					}
					padOffset := m.Config.Style.Padding

					// Calculate centering offset based on actual sixel pixel dimensions
					sixelWidthCells := sixelResult.Width / m.CellPx.Width
					sixelHeightCells := sixelResult.Height / m.CellPx.Height
					centerOffsetX := (iconW - sixelWidthCells) / 2
					centerOffsetY := (iconH - sixelHeightCells) / 2

					// Position: centered within the icon area
					// +1 because terminal positions are 1-indexed
					posX := col*cellW + borderOffset + padOffset + centerOffsetX + 1
					posY := row*cellH + borderOffset + padOffset + centerOffsetY + 1

					if posX < 1 {
						posX = 1
					}
					if posY < 1 {
						posY = 1
					}

					// Move cursor and render sixel
					b.WriteString(fmt.Sprintf(cursorTo, posY, posX))
					b.WriteString(sixelResult.Sixel)
				}
			}
			appIndex++
		}
	}

	m.SixelsDrawn = true
}

// renderCellFrame renders just the border/frame of a cell.
// Note: Border colors are now handled via direct ANSI in flashCell/drawNormalBorder
// to avoid triggering full View() redraws on interaction.
func (m *Model) renderCellFrame(index, innerW, innerH int) string {
	style := lipgloss.NewStyle().
		Width(innerW).
		Height(innerH)

	if m.Config.Style.Border {
		borderStyle := lipgloss.RoundedBorder()
		// Use configured border color for initial render
		borderColor := lipgloss.Color(m.Config.GetBorderColor())

		if m.ErrorFlash[index] {
			borderColor = lipgloss.Color("196")
		}
		// Note: m.Selected highlighting is now handled via direct ANSI

		style = style.
			Border(borderStyle).
			BorderForeground(borderColor)
	}

	return style.Render("")
}

// renderEmptyCell renders an empty placeholder cell.
func (m *Model) renderEmptyCell(innerW, innerH int) string {
	style := lipgloss.NewStyle().
		Width(innerW).
		Height(innerH)

	if m.Config.Style.Border {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(m.Config.GetBorderColor()))
	}

	return style.Render("")
}

// getSixelContentWithDimensions retrieves cached sixel with dimensions or generates new.
func (m *Model) getSixelContentWithDimensions(index, widthCells, heightCells int, scale float64) graphics.SixelResult {
	key := fmt.Sprintf("%d_%d_%d_%.2f", index, widthCells, heightCells, scale)

	if cached, ok := m.SixelCache[key]; ok {
		return cached
	}

	var result graphics.SixelResult
	if index < len(m.Icons) && m.Icons[index] != nil {
		result = graphics.RenderSixelWithDimensions(m.Icons[index], widthCells, heightCells, m.CellPx)
	}

	m.SixelCache[key] = result
	return result
}
