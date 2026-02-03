package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tooie-shelf/internal/app"
	"tooie-shelf/internal/config"
)

func main() {
	// Ensure config directory exists
	if err := config.EnsureConfigDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create config directory: %v\n", err)
	}

	// Load configuration
	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create model
	model := app.NewModel(cfg)

	// Create program with mouse support
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
