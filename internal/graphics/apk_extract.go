package graphics

import (
	"archive/zip"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	_ "golang.org/x/image/webp"
)

// getAPKPaths returns all APK paths for a given package using pm path command.
// For App Bundles, this returns multiple paths (base + split APKs).
func getAPKPaths(pkg string) ([]string, error) {
	cmd := exec.Command("pm", "path", pkg)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pm path failed: %w", err)
	}

	// Output format: "package:/data/app/.../base.apk\npackage:/data/app/.../split_config.xxhdpi.apk"
	var paths []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "package:")
		if line != "" {
			paths = append(paths, line)
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("could not find APK for package %s", pkg)
	}
	return paths, nil
}

// getIconPathFromAAPT2 uses aapt2 to get the icon resource path from the APK.
// First tries the application: line (most accurate), then falls back to application-icon lines.
func getIconPathFromAAPT2(apkPath string) (string, error) {
	cmd := exec.Command("aapt2", "dump", "badging", apkPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("aapt2 failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")

	// First, try to get icon from the application: line
	// Format: application: label='AppName' icon='res/XX.png' ...
	for _, line := range lines {
		if !strings.HasPrefix(line, "application:") {
			continue
		}

		// Extract icon='...' from the line
		if idx := strings.Index(line, "icon='"); idx != -1 {
			start := idx + 6 // len("icon='")
			end := strings.Index(line[start:], "'")
			if end != -1 {
				iconPath := line[start : start+end]
				// If it's XML, try PNG version
				if strings.HasSuffix(iconPath, ".xml") {
					pngPath := strings.TrimSuffix(iconPath, ".xml") + ".png"
					return pngPath, nil
				}
				if strings.HasSuffix(iconPath, ".png") {
					return iconPath, nil
				}
			}
		}
	}

	// Fallback: Parse application-icon lines
	// Format: application-icon-640:'res/BW.xml' or application-icon-640:'res/BW.png'
	var highestResIcon string
	highestRes := 0

	for _, line := range lines {
		if !strings.HasPrefix(line, "application-icon-") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		resPart := strings.TrimPrefix(parts[0], "application-icon-")
		res, _ := strconv.Atoi(resPart)
		path := strings.Trim(parts[1], "'")

		if strings.HasSuffix(path, ".png") {
			if res > highestRes {
				highestRes = res
				highestResIcon = path
			}
		} else if strings.HasSuffix(path, ".xml") && highestResIcon == "" {
			pngPath := strings.TrimSuffix(path, ".xml") + ".png"
			highestResIcon = pngPath
			highestRes = res
		}
	}

	if highestResIcon == "" {
		return "", fmt.Errorf("no icon found via aapt2")
	}

	return highestResIcon, nil
}

// extractIconFromAPK extracts icon directly from APK using mipmap patterns.
// This mimics Activity Launcher's approach: look for mipmap/ic_launcher in highest density.
func extractIconFromAPK(apkPath string, pkg string) (image.Image, string, error) {
	logIconExtraction(pkg, "Opening APK", apkPath)

	r, err := zip.OpenReader(apkPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open APK: %w", err)
	}
	defer r.Close()

	// Build a map of all files for quick lookup
	fileMap := make(map[string]*zip.File)
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	// Log all mipmap files found (for debugging)
	if debugEnabled {
		var mipmapFiles []string
		for name := range fileMap {
			if strings.Contains(name, "mipmap") && (strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".webp")) {
				mipmapFiles = append(mipmapFiles, name)
			}
		}
		logIconExtraction(pkg, "Mipmap files found", fmt.Sprintf("%d files", len(mipmapFiles)))
		for _, f := range mipmapFiles {
			logDebug("  - %s", f)
		}
	}

	// PRIORITY 1: Direct mipmap/ic_launcher lookup (like Activity Launcher)
	// This is the standard Android app icon location
	// Check WebP first (modern apps), then PNG
	// Also check app_icon (used by some OEMs like Nothing Phone)
	mipmapPatterns := []string{
		// Standard ic_launcher - WebP variants first
		"res/mipmap-xxxhdpi-v4/ic_launcher.webp",
		"res/mipmap-xxxhdpi/ic_launcher.webp",
		"res/mipmap-xxhdpi-v4/ic_launcher.webp",
		"res/mipmap-xxhdpi/ic_launcher.webp",
		"res/mipmap-xhdpi-v4/ic_launcher.webp",
		"res/mipmap-xhdpi/ic_launcher.webp",
		"res/mipmap-hdpi-v4/ic_launcher.webp",
		"res/mipmap-hdpi/ic_launcher.webp",
		"res/mipmap-mdpi-v4/ic_launcher.webp",
		"res/mipmap-mdpi/ic_launcher.webp",
		"res/mipmap/ic_launcher.webp",
		// Standard ic_launcher - PNG variants
		"res/mipmap-xxxhdpi-v4/ic_launcher.png",
		"res/mipmap-xxxhdpi/ic_launcher.png",
		"res/mipmap-xxhdpi-v4/ic_launcher.png",
		"res/mipmap-xxhdpi/ic_launcher.png",
		"res/mipmap-xhdpi-v4/ic_launcher.png",
		"res/mipmap-xhdpi/ic_launcher.png",
		"res/mipmap-hdpi-v4/ic_launcher.png",
		"res/mipmap-hdpi/ic_launcher.png",
		"res/mipmap-mdpi-v4/ic_launcher.png",
		"res/mipmap-mdpi/ic_launcher.png",
		"res/mipmap/ic_launcher.png",
		// Alternative app_icon (used by Nothing Phone and some OEMs) - WebP
		"res/mipmap-xxxhdpi-v4/app_icon.webp",
		"res/mipmap-xxxhdpi/app_icon.webp",
		"res/mipmap-xxhdpi-v4/app_icon.webp",
		"res/mipmap-xxhdpi/app_icon.webp",
		"res/mipmap-xhdpi-v4/app_icon.webp",
		"res/mipmap-xhdpi/app_icon.webp",
		"res/mipmap-hdpi-v4/app_icon.webp",
		"res/mipmap-hdpi/app_icon.webp",
		"res/mipmap-mdpi-v4/app_icon.webp",
		"res/mipmap-mdpi/app_icon.webp",
		"res/mipmap/app_icon.webp",
		// Alternative app_icon - PNG
		"res/mipmap-xxxhdpi-v4/app_icon.png",
		"res/mipmap-xxxhdpi/app_icon.png",
		"res/mipmap-xxhdpi-v4/app_icon.png",
		"res/mipmap-xxhdpi/app_icon.png",
		"res/mipmap-xhdpi-v4/app_icon.png",
		"res/mipmap-xhdpi/app_icon.png",
		"res/mipmap-hdpi-v4/app_icon.png",
		"res/mipmap-hdpi/app_icon.png",
		"res/mipmap-mdpi-v4/app_icon.png",
		"res/mipmap-mdpi/app_icon.png",
		"res/mipmap/app_icon.png",
	}

	for _, pattern := range mipmapPatterns {
		if f, ok := fileMap[pattern]; ok {
			logIconExtraction(pkg, "Found mipmap/ic_launcher", pattern)
			rc, err := f.Open()
			if err == nil {
				defer rc.Close()
				img, _, err := image.Decode(rc)
				if err == nil {
					logIconExtraction(pkg, "Successfully decoded", pattern)
					return img, pattern, nil
				}
				logIconExtraction(pkg, "Failed to decode", pattern, err.Error())
			}
		}
	}

	// PRIORITY 2: Try ADB (pm dump) to get icon path
	if pkg != "" {
		logIconExtraction(pkg, "Trying ADB (pm dump)")
		iconPath, err := getIconPathViaADB(pkg)
		if err == nil && iconPath != "" {
			logIconExtraction(pkg, "ADB returned path", iconPath)
			if f, ok := fileMap[iconPath]; ok {
				rc, err := f.Open()
				if err == nil {
					defer rc.Close()
					img, _, err := image.Decode(rc)
					if err == nil {
						logIconExtraction(pkg, "Successfully decoded from ADB path", iconPath)
						return img, iconPath, nil
					}
				}
			}
			// If ADB returned XML path, try PNG version
			if strings.HasSuffix(iconPath, ".xml") {
				pngPath := strings.TrimSuffix(iconPath, ".xml") + ".png"
				logIconExtraction(pkg, "Trying PNG version of XML", pngPath)
				if f, ok := fileMap[pngPath]; ok {
					rc, err := f.Open()
					if err == nil {
						defer rc.Close()
						img, _, err := image.Decode(rc)
						if err == nil {
							logIconExtraction(pkg, "Successfully decoded PNG", pngPath)
							return img, pngPath, nil
						}
					}
				}
			}
		} else if err != nil {
			logIconExtraction(pkg, "ADB failed", err.Error())
		}
	}

	// PRIORITY 3: Try aapt2
	logIconExtraction(pkg, "Trying aapt2")
	iconPath, err := getIconPathFromAAPT2(apkPath)
	if err == nil && iconPath != "" {
		logIconExtraction(pkg, "aapt2 returned path", iconPath)
		if f, ok := fileMap[iconPath]; ok {
			rc, err := f.Open()
			if err == nil {
				defer rc.Close()
				img, _, err := image.Decode(rc)
				if err == nil {
					logIconExtraction(pkg, "Successfully decoded from aapt2 path", iconPath)
					return img, iconPath, nil
				}
			}
		}
		// If aapt2 returned XML path, try PNG version
		if strings.HasSuffix(iconPath, ".xml") {
			pngPath := strings.TrimSuffix(iconPath, ".xml") + ".png"
			if f, ok := fileMap[pngPath]; ok {
				rc, err := f.Open()
				if err == nil {
					defer rc.Close()
					img, _, err := image.Decode(rc)
					if err == nil {
						return img, pngPath, nil
					}
				}
			}
		}
	} else if err != nil {
		logIconExtraction(pkg, "aapt2 failed", err.Error())
	}

	// PRIORITY 4: Drawable fallbacks
	drawablePatterns := []string{
		"res/drawable-xxxhdpi-v4/ic_launcher.png",
		"res/drawable-xxxhdpi/ic_launcher.png",
		"res/drawable-xxhdpi-v4/ic_launcher.png",
		"res/drawable-xxhdpi/ic_launcher.png",
		"res/drawable-xhdpi-v4/ic_launcher.png",
		"res/drawable-xhdpi/ic_launcher.png",
		"res/drawable-hdpi-v4/ic_launcher.png",
		"res/drawable-hdpi/ic_launcher.png",
		"res/drawable-mdpi-v4/ic_launcher.png",
		"res/drawable-mdpi/ic_launcher.png",
		"res/drawable/ic_launcher.png",
		// WebP variants
		"res/drawable-xxxhdpi-v4/ic_launcher.webp",
		"res/drawable-xxxhdpi/ic_launcher.webp",
		"res/drawable-xxhdpi-v4/ic_launcher.webp",
		"res/drawable-xxhdpi/ic_launcher.webp",
		"res/drawable-xhdpi-v4/ic_launcher.webp",
		"res/drawable-xhdpi/ic_launcher.webp",
		"res/drawable-hdpi-v4/ic_launcher.webp",
		"res/drawable-hdpi/ic_launcher.webp",
		"res/drawable-mdpi-v4/ic_launcher.webp",
		"res/drawable-mdpi/ic_launcher.webp",
		"res/drawable/ic_launcher.webp",
	}

	for _, pattern := range drawablePatterns {
		if f, ok := fileMap[pattern]; ok {
			logIconExtraction(pkg, "Found drawable/ic_launcher", pattern)
			rc, err := f.Open()
			if err == nil {
				defer rc.Close()
				img, _, err := image.Decode(rc)
				if err == nil {
					logIconExtraction(pkg, "Successfully decoded", pattern)
					return img, pattern, nil
				}
			}
		}
	}

	// PRIORITY 5: Any PNG/WebP in mipmap (largest) - ONLY in base APK
	// Skip this for split APKs to avoid picking up random images
	if !strings.Contains(apkPath, "split_config.") {
		logIconExtraction(pkg, "Looking for any mipmap image in base APK")
		var largestMipmap *zip.File
		var largestMipmapSize uint64
		var largestMipmapName string

		for name, f := range fileMap {
			if strings.Contains(name, "mipmap") &&
				(strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".webp")) {
				if f.UncompressedSize64 > largestMipmapSize {
					largestMipmapSize = f.UncompressedSize64
					largestMipmap = f
					largestMipmapName = name
				}
			}
		}

		if largestMipmap != nil {
			logIconExtraction(pkg, "Found largest mipmap", largestMipmapName)
			rc, err := largestMipmap.Open()
			if err == nil {
				defer rc.Close()
				img, _, err := image.Decode(rc)
				if err == nil {
					logIconExtraction(pkg, "Successfully decoded largest mipmap", largestMipmapName)
					return img, largestMipmapName, nil
				}
			}
		}
	} else {
		logIconExtraction(pkg, "Skipping mipmap search in split APK")
	}

	return nil, "", fmt.Errorf("no icon found in APK")
}

// ExtractAPKIcon extracts the app icon from an APK file.
// For App Bundles, searches through all split APKs.
// Icons are cached to avoid repeated extraction.
func ExtractAPKIcon(pkg string) (image.Image, error) {
	if pkg == "" {
		return nil, fmt.Errorf("empty package name")
	}

	logIconExtraction(pkg, "Starting icon extraction")

	// Check Tier 1 cache first (PNG icon)
	cachePath := getCachedIconPath(pkg)
	if cached, err := LoadImage(cachePath); err == nil {
		logIconExtraction(pkg, "Tier 1 cache hit", cachePath)
		return cached, nil
	}
	logIconExtraction(pkg, "Tier 1 cache miss")

	// Get all APK paths (base + splits for App Bundles)
	apkPaths, err := getAPKPaths(pkg)
	if err != nil {
		logIconExtraction(pkg, "Failed to get APK paths", err.Error())
		return nil, err
	}
	logIconExtraction(pkg, "Found APKs", fmt.Sprintf("%d paths", len(apkPaths)))
	for i, path := range apkPaths {
		logDebug("  APK[%d]: %s", i, path)
	}

	// Try to extract icon from each APK (base first, then splits)
	var img image.Image
	var iconSource string
	var lastErr error
	for _, apkPath := range apkPaths {
		img, iconSource, err = extractIconFromAPK(apkPath, pkg)
		if err == nil {
			logIconExtraction(pkg, "Icon extracted successfully", iconSource)
			break
		}
		logIconExtraction(pkg, "Failed to extract from APK", apkPath, err.Error())
		lastErr = err
	}

	if img == nil {
		logIconExtraction(pkg, "All extraction methods failed", lastErr.Error())
		return nil, fmt.Errorf("could not extract icon from any APK: %w", lastErr)
	}

	// Save to Tier 1 cache
	logIconExtraction(pkg, "Saving to Tier 1 cache", cachePath)
	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
	_ = SaveImage(img, cachePath)

	return img, nil
}
