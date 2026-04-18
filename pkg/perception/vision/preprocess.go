package vision

import (
	"image"
	"golang.org/x/image/draw"
)

// resizeImage resizes an image.Image to the target width and height using BiLinear interpolation.
func resizeImage(img image.Image, targetWidth, targetHeight int) *image.RGBA {
	bounds := img.Bounds()
	if bounds.Dx() == targetWidth && bounds.Dy() == targetHeight {
		if rgba, ok := img.(*image.RGBA); ok {
			return rgba
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// preprocessImage converts an image.Image to a float32 tensor (NCHW format) for Florence-2.
// 1. Resizes to targetWidth x targetHeight
// 2. Converts from HWC (Go's image format) to NCHW (ONNX format)
// 3. Normalizes pixel values to [0, 1] range
func preprocessImage(img image.Image, targetWidth, targetHeight int) []float32 {
	// Step 1: Resize
	resized := resizeImage(img, targetWidth, targetHeight)

	// Step 2 & 3: HWC -> NCHW transformation and normalization
	// Go image gives us HxWx3 (RGB)
	// ONNX expects Nx3xHxW where N=1
	nchwData := make([]float32, 1*3*targetHeight*targetWidth)

	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			// Get RGBA values (0-65535)
			r, g, b, _ := resized.At(x, y).RGBA()
			
			pixel := y*targetWidth + x

			// Channel-first (NCHW): R plane, G plane, B plane
			// Normalize from [0, 65535] to [0, 1]
			nchwData[0*targetHeight*targetWidth+pixel] = float32(r) / 65535.0 // R
			nchwData[1*targetHeight*targetWidth+pixel] = float32(g) / 65535.0 // G
			nchwData[2*targetHeight*targetWidth+pixel] = float32(b) / 65535.0 // B
		}
	}

	return nchwData
}
