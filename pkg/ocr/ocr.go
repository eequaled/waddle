package ocr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractText takes an image path and returns the text found in it using Windows OCR.
func ExtractText(imagePath string) (string, error) {
	// Resolve script path
	// Assuming the binary is run from the project root or the script is in a known location relative to the executable.
	// For dev (go run), it's in pkg/ocr/scripts/ocr.ps1

	// Try to find the script
	scriptPath := filepath.Join("pkg", "ocr", "scripts", "ocr.ps1")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// Try absolute path if we are in a different working directory
		// Or maybe we should embed it? For now, let's assume CWD is project root.
		return "", fmt.Errorf("OCR script not found at %s", scriptPath)
	}

	// Prepare command
	// -NoProfile -ExecutionPolicy Bypass -File <script> -ImagePath <image>
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "-ImagePath", imagePath)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("OCR execution failed: %v, stderr: %s", err, stderr.String())
	}

	// Clean output
	output := strings.TrimSpace(out.String())

	// Check for script errors that might be printed to stdout
	if strings.HasPrefix(output, "Error:") {
		return "", fmt.Errorf("OCR script error: %s", output)
	}

	return output, nil
}
