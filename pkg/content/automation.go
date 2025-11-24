package content

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractContext executes the bundled PowerShell script to get text from the active window.
func ExtractContext() (string, error) {
	// Locate the script relative to the executable or current working directory
	// For development (go run), it might be in pkg/content/scripts
	// For production, we might need a more robust path resolution strategy
	scriptPath := filepath.Join("pkg", "content", "scripts", "get_text.ps1")

	// Prepare the command
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// Clean up output
	output := strings.TrimSpace(out.String())
	return output, nil
}
