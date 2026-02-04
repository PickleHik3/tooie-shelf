package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"tooie-shelf/internal/sys"
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
// Auto-detects package/activity for apps that don't specify them.
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

	// Expand ~ in icon paths and auto-detect missing package/activity
	for i := range cfg.Apps {
		cfg.Apps[i].Icon = expandPath(cfg.Apps[i].Icon)

		// Auto-detect package and activity if not specified and not a command
		if cfg.Apps[i].Command == "" && (cfg.Apps[i].Package == "" || cfg.Apps[i].Activity == "") {
			if err := autoDetectAppInfo(&cfg.Apps[i]); err != nil {
				// Log warning but don't fail - app may be optional
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}

	// Validate configuration
	if err := validate(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// autoDetectAppInfo attempts to detect package and activity for an app.
func autoDetectAppInfo(app *AppConfig) error {
	// Skip if already has both package and activity
	if app.Package != "" && app.Activity != "" {
		return nil
	}

	// Skip if it's a command-based app
	if app.Command != "" {
		return nil
	}

	fmt.Fprintf(os.Stderr, "Auto-detecting package/activity for '%s'...\n", app.Name)

	// Try to detect package
	if app.Package == "" {
		pkg, err := sys.AutoDetectPackage(app.Name)
		if err != nil {
			return fmt.Errorf("could not auto-detect package for '%s': %w", app.Name, err)
		}
		app.Package = pkg
		fmt.Fprintf(os.Stderr, "  Found package: %s\n", pkg)
	}

	// Try to detect activity
	if app.Activity == "" {
		activity, err := sys.AutoDetectActivity(app.Package)
		if err != nil {
			return fmt.Errorf("could not auto-detect activity for '%s' (%s): %w", app.Name, app.Package, err)
		}
		app.Activity = activity
		fmt.Fprintf(os.Stderr, "  Found activity: %s\n", activity)
	}

	return nil
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
				return fmt.Errorf("app %d (%s): package name is required for Android apps (or use auto-detect by omitting package/activity)", i, app.Name)
			}
			if app.Activity == "" {
				return fmt.Errorf("app %d (%s): activity is required for Android apps (or use auto-detect by omitting package/activity)", i, app.Name)
			}
		}
		if app.Icon != "" {
			// Skip file validation for special icon sources
			isSpecialSource := strings.HasPrefix(app.Icon, "dashboard:") ||
				strings.HasPrefix(app.Icon, "http://") ||
				strings.HasPrefix(app.Icon, "https://")

			if !isSpecialSource {
				if _, err := os.Stat(app.Icon); err != nil {
					return fmt.Errorf("app %d (%s): icon file not found: %s", i, app.Name, app.Icon)
				}
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
