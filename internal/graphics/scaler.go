package graphics

import (
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

// ScaleImage scales an image to the target dimensions using CatmullRom (Lanczos-like).
func ScaleImage(src image.Image, targetW, targetH int) image.Image {
	if targetW <= 0 || targetH <= 0 {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// ScaleImageAspectFit scales an image to fit within target dimensions while preserving aspect ratio.
func ScaleImageAspectFit(src image.Image, maxW, maxH int) image.Image {
	if maxW <= 0 || maxH <= 0 {
		return src
	}

	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Calculate scale factor to fit within bounds
	scaleW := float64(maxW) / float64(srcW)
	scaleH := float64(maxH) / float64(srcH)
	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}

	targetW := int(float64(srcW) * scale)
	targetH := int(float64(srcH) * scale)

	if targetW <= 0 {
		targetW = 1
	}
	if targetH <= 0 {
		targetH = 1
	}

	return ScaleImage(src, targetW, targetH)
}

// StandardizeImage creates a square image by adding transparent padding to center the source image.
// This ensures all icons have the same aspect ratio for consistent scaling and positioning.
func StandardizeImage(src image.Image, size int) image.Image {
	if size <= 0 {
		size = 256 // Default standard size
	}

	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Create a square destination with transparent background
	dst := image.NewRGBA(image.Rect(0, 0, size, size))

	// Fill with transparent background
	transparent := color.RGBA{0, 0, 0, 0}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dst.Set(x, y, transparent)
		}
	}

	// Calculate position to center the source image
	// Scale to fit within the square while preserving aspect ratio
	scaleW := float64(size) / float64(srcW)
	scaleH := float64(size) / float64(srcH)
	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}

	// Allow upscaling to fill the target area - this ensures all icons are the same size

	scaledW := int(float64(srcW) * scale)
	scaledH := int(float64(srcH) * scale)

	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}

	// Center the scaled image
	offsetX := (size - scaledW) / 2
	offsetY := (size - scaledH) / 2

	// Scale and draw the source image centered
	scaled := ScaleImage(src, scaledW, scaledH)
	draw.Draw(dst, image.Rect(offsetX, offsetY, offsetX+scaledW, offsetY+scaledH), scaled, image.Point{}, draw.Over)

	return dst
}
