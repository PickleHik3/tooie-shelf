package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigPath returns the default config file path.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "tooie-shelf", "config.yaml")
}

// Load reads and parses the configuration file.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if no config exists
		}
		return cfg, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand ~ in icon paths
	for i := range cfg.Apps {
		cfg.Apps[i].Icon = expandPath(cfg.Apps[i].Icon)
	}

	// Validate configuration
	if err := validate(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// validate checks the configuration for errors.
func validate(cfg Config) error {
	if cfg.Grid.Rows < 1 {
		return fmt.Errorf("grid.rows must be at least 1")
	}
	if cfg.Grid.Columns < 1 {
		return fmt.Errorf("grid.columns must be at least 1")
	}

	for i, app := range cfg.Apps {
		// Android apps require both package and activity
		if app.Command == "" {
			if app.Package == "" {
				return fmt.Errorf("app %d (%s): package name is required for Android apps", i, app.Name)
			}
			if app.Activity == "" {
				return fmt.Errorf("app %d (%s): activity is required for Android apps (use 'command' for Linux commands)", i, app.Name)
			}
		}
		if app.Icon != "" {
			if _, err := os.Stat(app.Icon); err != nil {
				return fmt.Errorf("app %d (%s): icon file not found: %s", i, app.Name, app.Icon)
			}
		}
	}

	return nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "tooie-shelf", "icons")
	return os.MkdirAll(dir, 0755)
}
