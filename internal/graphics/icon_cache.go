package graphics

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Cache TTL for icon path cache (7 days)
const iconPathCacheTTL = 7 * 24 * time.Hour

// getCachedIconPath returns the path for a cached icon PNG (Tier 1 cache).
func getCachedIconPath(pkg string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tooie-shelf", "icons", pkg+".png")
}

// getIconPathCachePath returns the path for cached icon resource path (Tier 2 cache).
func getIconPathCachePath(pkg string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tooie-shelf", "icon-paths", pkg+".txt")
}

// getCachedIconPathWithTTL returns cached icon path if valid, or empty if expired/missing.
func getCachedIconPathWithTTL(pkg string) string {
	cachePath := getIconPathCachePath(pkg)

	info, err := os.Stat(cachePath)
	if err != nil {
		return ""
	}

	// Check if cache is expired
	if time.Since(info.ModTime()) > iconPathCacheTTL {
		return ""
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// saveIconPathCache saves the icon resource path to cache.
func saveIconPathCache(pkg, iconPath string) error {
	cachePath := getIconPathCachePath(pkg)
	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
	return os.WriteFile(cachePath, []byte(iconPath), 0644)
}

// getIconPathViaADB uses rish to query icon path via pm dump (avoids aapt2 dependency).
// This is expensive, so results are cached aggressively.
func getIconPathViaADB(pkg string) (string, error) {
	// Check Tier 2 cache first
	if cached := getCachedIconPathWithTTL(pkg); cached != "" {
		return cached, nil
	}

	// Query via rish (adb shell)
	// rish is at $HOME/.rish/rish
	home, _ := os.UserHomeDir()
	rishPath := filepath.Join(home, ".rish", "rish")

	// Check if rish exists
	if _, err := os.Stat(rishPath); err != nil {
		return "", fmt.Errorf("rish not found at %s", rishPath)
	}

	// Run pm dump via rish
	cmd := exec.Command(rishPath, "-c", fmt.Sprintf("pm dump %s", pkg))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rish pm dump failed: %w", err)
	}

	// Parse pm dump output for icon resource
	iconPath := parseIconFromPMDump(string(output))
	if iconPath == "" {
		return "", fmt.Errorf("no icon found in pm dump output")
	}

	// Save to Tier 2 cache
	_ = saveIconPathCache(pkg, iconPath)

	return iconPath, nil
}

// parseIconFromPMDump extracts icon resource path from pm dump output.
func parseIconFromPMDump(output string) string {
	lines := strings.Split(output, "\n")

	// Look for icon resource in various formats
	// Primary: "icon=res/mipmap-xxxhdpi/ic_launcher.png"
	// Alternative: "applicationIcon=..." or in activity sections

	var bestIcon string
	highestRes := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for icon= pattern
		if strings.Contains(line, "icon=") {
			parts := strings.Split(line, "icon=")
			if len(parts) >= 2 {
				iconPart := strings.TrimSpace(parts[1])
				// Extract just the resource path
				if idx := strings.Index(iconPart, " "); idx != -1 {
					iconPart = iconPart[:idx]
				}

				// Check if it's a PNG (prefer over XML)
				if strings.HasSuffix(iconPart, ".png") {
					res := extractResolution(iconPart)
					if res > highestRes {
						highestRes = res
						bestIcon = iconPart
					}
				} else if strings.HasSuffix(iconPart, ".xml") && bestIcon == "" {
					// XML fallback - try PNG version
					bestIcon = strings.TrimSuffix(iconPart, ".xml") + ".png"
				}
			}
		}
	}

	return bestIcon
}

// extractResolution extracts DPI resolution from resource path.
// e.g., "mipmap-xxxhdpi" -> 640
func extractResolution(path string) int {
	// Map density names to approximate DPI values
	densityMap := map[string]int{
		"ldpi":    120,
		"mdpi":    160,
		"hdpi":    240,
		"xhdpi":   320,
		"xxhdpi":  480,
		"xxxhdpi": 640,
		"anydpi":  0,
	}

	for name, dpi := range densityMap {
		if strings.Contains(path, "-"+name) {
			return dpi
		}
	}
	return 0
}
