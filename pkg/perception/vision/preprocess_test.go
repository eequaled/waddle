package vision

import (
	"image"
	"image/color"
	"testing"
)

func TestPreprocessImage(t *testing.T) {
	// Create a 2x2 test image
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	
	// Set pixels:
	// Top-left: Red
	// Top-right: Green
	// Bottom-left: Blue
	// Bottom-right: White
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})
	img.Set(0, 1, color.RGBA{0, 0, 255, 255})
	img.Set(1, 1, color.RGBA{255, 255, 255, 255})

	// Preprocess to 2x2 (no resizing happens, just format change)
	nchwData := preprocessImage(img, 2, 2)

	// Expected length: 1 * 3 * 2 * 2 = 12
	if len(nchwData) != 12 {
		t.Fatalf("Expected 12 elements, got %d", len(nchwData))
	}

	// Helper to check closeness (float math)
	isClose := func(a, b float32) bool {
		diff := a - b
		if diff < 0 {
			diff = -diff
		}
		return diff < 0.01
	}

	// Check Red channel plane (indices 0-3)
	if !isClose(nchwData[0], 1.0) { t.Errorf("Expected R[0]~1.0, got %f", nchwData[0]) } // Red pixel
	if !isClose(nchwData[1], 0.0) { t.Errorf("Expected R[1]~0.0, got %f", nchwData[1]) } // Green pixel
	if !isClose(nchwData[2], 0.0) { t.Errorf("Expected R[2]~0.0, got %f", nchwData[2]) } // Blue pixel
	if !isClose(nchwData[3], 1.0) { t.Errorf("Expected R[3]~1.0, got %f", nchwData[3]) } // White pixel

	// Check Green channel plane (indices 4-7)
	if !isClose(nchwData[4], 0.0) { t.Errorf("Expected G[0]~0.0, got %f", nchwData[4]) }
	if !isClose(nchwData[5], 1.0) { t.Errorf("Expected G[1]~1.0, got %f", nchwData[5]) }
	if !isClose(nchwData[6], 0.0) { t.Errorf("Expected G[2]~0.0, got %f", nchwData[6]) }
	if !isClose(nchwData[7], 1.0) { t.Errorf("Expected G[3]~1.0, got %f", nchwData[7]) }

	// Check Blue channel plane (indices 8-11)
	if !isClose(nchwData[8], 0.0) { t.Errorf("Expected B[0]~0.0, got %f", nchwData[8]) }
	if !isClose(nchwData[9], 0.0) { t.Errorf("Expected B[1]~0.0, got %f", nchwData[9]) }
	if !isClose(nchwData[10], 1.0) { t.Errorf("Expected B[2]~1.0, got %f", nchwData[10]) }
	if !isClose(nchwData[11], 1.0) { t.Errorf("Expected B[3]~1.0, got %f", nchwData[11]) }
}
