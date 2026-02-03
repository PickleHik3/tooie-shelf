package graphics

import (
	"bytes"
	"image"
	"image/png"
	_ "image/gif"
	_ "image/jpeg"
	"os"

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
func RenderSixelWithDimensions(src image.Image, widthCells, heightCells int, cellPx sys.CellDim) SixelResult {
	targetW := widthCells * cellPx.Width
	targetH := heightCells * cellPx.Height

	if targetW <= 0 || targetH <= 0 {
		return SixelResult{}
	}

	scaled := ScaleImageAspectFit(src, targetW, targetH)
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
