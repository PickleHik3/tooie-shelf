package graphics

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	_ "image/gif"
	_ "image/jpeg"
	"os"
	"os/exec"

	"github.com/mattn/go-sixel"

	"tooie-shelf/internal/sys"
)

// SixelResult contains the sixel string and its pixel dimensions.
type SixelResult struct {
	Sixel  string
	Width  int // Actual pixel width of rendered sixel
	Height int // Actual pixel height of rendered sixel
}

// RenderSixel converts an image to a sixel string sized for the given cell dimensions.
func RenderSixel(src image.Image, widthCells, heightCells int, cellPx sys.CellDim) string {
	result := RenderSixelWithDimensions(src, widthCells, heightCells, cellPx)
	return result.Sixel
}

// RenderSixelWithDimensions converts an image to a sixel string and returns the actual pixel dimensions.
// All icons are standardized to a square format before scaling to ensure consistent sizing.
func RenderSixelWithDimensions(src image.Image, widthCells, heightCells int, cellPx sys.CellDim) SixelResult {
	targetW := widthCells * cellPx.Width
	targetH := heightCells * cellPx.Height

	if targetW <= 0 || targetH <= 0 {
		return SixelResult{}
	}

	// Standardize to square format first to ensure all icons have same aspect ratio
	// Use the larger dimension as the standard size
	stdSize := targetW
	if targetH > targetW {
		stdSize = targetH
	}

	// Create standardized square icon
	standardized := StandardizeImage(src, stdSize)

	// Now scale to fit exactly within target dimensions
	scaled := ScaleImageAspectFit(standardized, targetW, targetH)
	bounds := scaled.Bounds()

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	if err := enc.Encode(scaled); err != nil {
		return SixelResult{}
	}
	return SixelResult{
		Sixel:  buf.String(),
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}
}

// LoadImage loads an image from a file path.
func LoadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// CreatePlaceholder creates a simple colored placeholder image.
func CreatePlaceholder(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a gray color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, image.White)
		}
	}
	return img
}

// SaveImage saves an image to a PNG file.
func SaveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// FetchDashboardIcon downloads an icon from the Dashboard Icons CDN.
// Format: "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/{name}.png"
func FetchDashboardIcon(iconName string) (image.Image, error) {
	if iconName == "" {
		return nil, fmt.Errorf("empty icon name")
	}

	url := fmt.Sprintf("https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/%s.png", iconName)
	return FetchIconFromURL(url)
}

// FetchIconFromURL downloads an icon from a URL.
func FetchIconFromURL(url string) (image.Image, error) {
	if url == "" {
		return nil, fmt.Errorf("empty URL")
	}

	cmd := exec.Command("curl", "-sL", "--max-time", "10", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch icon from %s: %w", url, err)
	}

	img, _, err := image.Decode(bytes.NewReader(output))
	if err != nil {
		return nil, fmt.Errorf("failed to decode icon: %w", err)
	}

	return img, nil
}
