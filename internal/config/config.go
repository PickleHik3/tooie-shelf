package config

// Config represents the launcher configuration.
type Config struct {
	Display  []string       `yaml:"display,omitempty"`  // App names in display order (if empty, show all)
	Grid     GridConfig     `yaml:"grid"`
	Style    StyleConfig    `yaml:"style"`
	Behavior BehaviorConfig `yaml:"behavior"`
	Apps     []AppConfig    `yaml:"apps"`
}

// BehaviorConfig defines behavior options.
type BehaviorConfig struct {
	CloseOnLaunch bool `yaml:"close_on_launch"`
}

// GridConfig defines the grid layout.
type GridConfig struct {
	Rows    int `yaml:"rows"`
	Columns int `yaml:"columns"`
}

// StyleConfig defines visual styling options.
type StyleConfig struct {
	Border          bool   `yaml:"border"`
	Padding         int    `yaml:"padding"`
	IconScale       float64 `yaml:"icon_scale,omitempty"` // Global icon scale (0.1-1.0), default 1.0
	BorderColor     string `yaml:"border_color,omitempty"`     // Normal border color (ANSI 256 color or "default")
	HighlightColor  string `yaml:"highlight_color,omitempty"`  // Click highlight color (ANSI 256 color or "default")
}

// AppConfig defines a single app entry.
type AppConfig struct {
	Name      string  `yaml:"name"`
	Icon      string  `yaml:"icon"`
	Package   string  `yaml:"package,omitempty"`           // Android package name
	Activity  string  `yaml:"activity,omitempty"`          // Android activity
	Command   string  `yaml:"command,omitempty"`           // Linux command/script/binary (takes priority over package)
	IconScale float64 `yaml:"icon_scale,omitempty"`        // Per-app override (0.1-1.0)
}

// IsCommand returns true if this app runs a command instead of launching an Android app.
func (a *AppConfig) IsCommand() bool {
	return a.Command != ""
}

// GetIconScale returns the effective icon scale for an app (per-app or global).
func (c *Config) GetIconScale(app AppConfig) float64 {
	if app.IconScale > 0 {
		return clampScale(app.IconScale)
	}
	if c.Style.IconScale > 0 {
		return clampScale(c.Style.IconScale)
	}
	return 1.0
}

// GetDisplayApps returns apps in display order. If Display is empty, returns all apps.
func (c *Config) GetDisplayApps() []AppConfig {
	if len(c.Display) == 0 {
		return c.Apps
	}

	// Build a map of apps by name
	appMap := make(map[string]AppConfig)
	for _, app := range c.Apps {
		appMap[app.Name] = app
	}

	// Return apps in display order
	var result []AppConfig
	for _, name := range c.Display {
		if app, ok := appMap[name]; ok {
			result = append(result, app)
		}
	}
	return result
}

// clampScale ensures scale is within valid range.
func clampScale(s float64) float64 {
	if s < 0.1 {
		return 0.1
	}
	if s > 1.0 {
		return 1.0
	}
	return s
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Display: []string{},
		Grid: GridConfig{
			Rows:    1,
			Columns: 5,
		},
		Style: StyleConfig{
			Border:         true,
			Padding:        1,
			IconScale:      1.0,
			BorderColor:    "240",
			HighlightColor: "96",
		},
		Behavior: BehaviorConfig{
			CloseOnLaunch: false,
		},
		Apps: []AppConfig{},
	}
}

// GetBorderColor returns the border color, or default if not set.
func (c *Config) GetBorderColor() string {
	if c.Style.BorderColor == "" || c.Style.BorderColor == "default" {
		return "240"
	}
	return c.Style.BorderColor
}

// GetHighlightColor returns the highlight color, or default if not set.
func (c *Config) GetHighlightColor() string {
	if c.Style.HighlightColor == "" || c.Style.HighlightColor == "default" {
		return "96"
	}
	return c.Style.HighlightColor
}
