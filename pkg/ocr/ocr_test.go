package ocr

import (
	"os"
	"testing"
)

func TestExtractText(t *testing.T) {
	// Find a test image
	testImage := `C:\Users\himez\OneDrive\Documents\ideathon\sessions\2025-11-25\Antigravity.exe\22-45-27.png`

	if _, err := os.Stat(testImage); os.IsNotExist(err) {
		t.Skip("Test image not found")
	}

	text, err := ExtractText(testImage)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	if text == "" {
		t.Error("ExtractText returned empty string")
	}

	t.Logf("Extracted text length: %d characters", len(text))
	t.Logf("First 200 chars: %s", text[:min(200, len(text))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
