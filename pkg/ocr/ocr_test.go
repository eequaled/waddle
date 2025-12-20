package ocr

import (
	"testing"
)

func TestExtractText(t *testing.T) {
	// Skip test if no test image is available
	// This test requires a real image file to work properly
	t.Skip("OCR test requires a real image file - skipping in automated tests")

	// Example usage (commented out):
	// testImage := `path/to/test/image.png`
	// if _, err := os.Stat(testImage); os.IsNotExist(err) {
	//     t.Skip("Test image not found")
	// }
	//
	// text, err := ExtractText(testImage)
	// if err != nil {
	//     t.Fatalf("ExtractText failed: %v", err)
	// }
	//
	// if text == "" {
	//     t.Error("ExtractText returned empty string")
	// }
	//
	// t.Logf("Extracted text length: %d characters", len(text))
	// t.Logf("First 200 chars: %s", text[:min(200, len(text))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
