package ocr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractText uses Tesseract OCR to extract text from an image
func ExtractText(imagePath string) (string, error) {
	// Find tesseract executable
	tesseractPath := findTesseract()
	if tesseractPath == "" {
		return "", fmt.Errorf("tesseract not found. Please install from: https://github.com/UB-Mannheim/tesseract/wiki")
	}

	// Prepare command
	// tesseract image.png stdout (outputs to stdout instead of file)
	cmd := exec.Command(tesseractPath, imagePath, "stdout", "-l", "eng", "--psm", "3")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("tesseract execution failed: %v, stderr: %s", err, stderr.String())
	}

	// Clean output
	text := strings.TrimSpace(out.String())

	return text, nil
}

// findTesseract locates the tesseract executable
// Checks: bundled with Electron app, bundled bin, common install locations, PATH
func findTesseract() string {
	// Get executable directory for bundled app detection
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	// 1. Check Electron bundled location (resources/tesseract)
	electronBundled := filepath.Join(exeDir, "resources", "tesseract", "tesseract.exe")
	if _, err := os.Stat(electronBundled); err == nil {
		return electronBundled
	}

	// 2. Check relative to exe (for portable/dev)
	relativeBundled := filepath.Join(exeDir, "tesseract", "tesseract.exe")
	if _, err := os.Stat(relativeBundled); err == nil {
		return relativeBundled
	}

	// 3. Check old bundled location (for distribution)
	bundledPath := filepath.Join("pkg", "ocr", "bin", "tesseract.exe")
	if _, err := os.Stat(bundledPath); err == nil {
		return bundledPath
	}

	// 4. Check common install locations
	commonPaths := []string{
		`C:\Program Files\Tesseract-OCR\tesseract.exe`,
		`C:\Program Files (x86)\Tesseract-OCR\tesseract.exe`,
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// 5. Check PATH
	path, err := exec.LookPath("tesseract")
	if err == nil {
		return path
	}

	return ""
}
