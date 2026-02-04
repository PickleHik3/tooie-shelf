package sys

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// cache for auto-detected packages to avoid repeated lookups
var packageCache = make(map[string]string)
var activityCache = make(map[string]string)
var cacheExpiry = make(map[string]time.Time)
const cacheDuration = 24 * time.Hour

// AutoDetectPackage finds the package name from an app name using fuzzy matching.
// It searches installed packages and returns the best match.
func AutoDetectPackage(appName string) (string, error) {
	// Check cache first
	if cached, ok := packageCache[appName]; ok {
		if time.Since(cacheExpiry[appName]) < cacheDuration {
			return cached, nil
		}
	}

	// Get all installed packages
	cmd := exec.Command("pm", "list", "packages")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list packages: %w", err)
	}

	// Normalize app name for matching
	searchName := strings.ToLower(appName)
	searchName = strings.ReplaceAll(searchName, " ", "")
	searchName = strings.ReplaceAll(searchName, "-", "")

	var bestMatch string
	bestScore := 0

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "package:")
		if line == "" {
			continue
		}

		// Calculate match score
		pkgLower := strings.ToLower(line)
		score := calculateMatchScore(pkgLower, searchName)

		if score > bestScore {
			bestScore = score
			bestMatch = line
		}
	}

	if bestMatch == "" || bestScore < 2 {
		return "", fmt.Errorf("no matching package found for '%s'", appName)
	}

	// Cache the result
	packageCache[appName] = bestMatch
	cacheExpiry[appName] = time.Now()

	return bestMatch, nil
}

// calculateMatchScore returns a score based on how well the package matches the search name.
// Higher score = better match.
func calculateMatchScore(pkg, search string) int {
	score := 0

	// Exact match after removing dots
	pkgNoDots := strings.ReplaceAll(pkg, ".", "")
	if pkgNoDots == search {
		return 100
	}

	// Contains search term
	if strings.Contains(pkgNoDots, search) {
		score += 50
	}

	// Search term contains package name (reverse)
	if strings.Contains(search, pkgNoDots) {
		score += 30
	}

	// Word boundaries match
	searchParts := strings.Split(search, "")
	for _, part := range searchParts {
		if len(part) > 2 && strings.Contains(pkgNoDots, part) {
			score += 5
		}
	}

	// Prefer shorter package names (more specific)
	score -= len(pkg) / 10

	return score
}

// AutoDetectActivity finds the main launcher activity for a package.
// It uses pm dump to find the activity with MAIN/LAUNCHER intent filter.
func AutoDetectActivity(pkg string) (string, error) {
	// Check cache first
	cacheKey := pkg
	if cached, ok := activityCache[cacheKey]; ok {
		if time.Since(cacheExpiry[cacheKey+"_act"]) < cacheDuration {
			return cached, nil
		}
	}

	// Try to find main activity using pm dump
	cmd := exec.Command("pm", "dump", pkg)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pm dump failed: %w", err)
	}

	activity := parseMainActivity(string(output), pkg)
	if activity == "" {
		return "", fmt.Errorf("no main activity found for package %s", pkg)
	}

	// Cache the result
	activityCache[cacheKey] = activity
	cacheExpiry[cacheKey+"_act"] = time.Now()

	return activity, nil
}

// parseMainActivity extracts the main launcher activity from pm dump output.
func parseMainActivity(output, pkg string) string {
	lines := strings.Split(output, "\n")
	
	// Look for MAIN/LAUNCHER intent filter
	inMainFilter := false
	var currentActivity string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check for activity declaration
		if strings.HasPrefix(line, "Activity #") {
			// Extract activity name from previous section if we found MAIN/LAUNCHER
			if inMainFilter && currentActivity != "" {
				return currentActivity
			}
			inMainFilter = false
			currentActivity = ""
			
			// Parse activity name from line like:
			// Activity #0: com.package.name/.ActivityName
			if idx := strings.Index(line, pkg); idx != -1 {
				rest := line[idx:]
				if spaceIdx := strings.Index(rest, " "); spaceIdx != -1 {
					rest = rest[:spaceIdx]
				}
				// Handle both full name and shorthand (.ActivityName)
				if strings.Contains(rest, "/") {
					parts := strings.Split(rest, "/")
					if len(parts) == 2 {
						if strings.HasPrefix(parts[1], ".") {
							currentActivity = parts[0] + parts[1]
						} else {
							currentActivity = parts[1]
						}
					}
				}
			}
		}
		
		// Check for MAIN action
		if strings.Contains(line, "android.intent.action.MAIN") {
			inMainFilter = true
		}
		
		// Check for LAUNCHER category
		if strings.Contains(line, "android.intent.category.LAUNCHER") && inMainFilter {
			if currentActivity != "" {
				return currentActivity
			}
		}
	}
	
	// Return last found activity if it was marked as MAIN
	if inMainFilter && currentActivity != "" {
		return currentActivity
	}
	
	return ""
}

// GetAppInfo attempts to auto-detect package and activity for an app.
// Returns the detected package and activity, or error if detection fails.
func GetAppInfo(appName string) (pkg, activity string, err error) {
	// Try to detect package
	pkg, err = AutoDetectPackage(appName)
	if err != nil {
		return "", "", fmt.Errorf("package detection failed: %w", err)
	}

	// Try to detect activity
	activity, err = AutoDetectActivity(pkg)
	if err != nil {
		return pkg, "", fmt.Errorf("activity detection failed: %w", err)
	}

	return pkg, activity, nil
}

// GetCachedAppInfo returns cached app info if available, or empty strings if not cached.
func GetCachedAppInfo(appName string) (pkg, activity string, valid bool) {
	pkg, pkgOk := packageCache[appName]
	if !pkgOk || time.Since(cacheExpiry[appName]) >= cacheDuration {
		return "", "", false
	}
	
	activity, actOk := activityCache[pkg]
	if !actOk || time.Since(cacheExpiry[pkg+"_act"]) >= cacheDuration {
		return pkg, "", false
	}
	
	return pkg, activity, true
}

// ClearAppInfoCache clears the auto-detection cache.
func ClearAppInfoCache() {
	packageCache = make(map[string]string)
	activityCache = make(map[string]string)
	cacheExpiry = make(map[string]time.Time)
}

// GetCachePath returns the path for storing auto-detected app info cache.
func GetCachePath() string {
	home, _ := filepath.Abs("~")
	if home == "" || home == "~" {
		home = "/data/data/com.termux/files/home"
	}
	return filepath.Join(home, ".config", "tooie-shelf", "app-cache.yaml")
}
