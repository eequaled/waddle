package processing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ideathon/pkg/ocr"
)

// BatchProcessor handles the background processing of screenshots
type BatchProcessor struct {
	SessionRoot string
}

// NewBatchProcessor creates a new processor
func NewBatchProcessor(root string) *BatchProcessor {
	return &BatchProcessor{
		SessionRoot: root,
	}
}

// ProcessAll scans all sessions and processes them
func (bp *BatchProcessor) ProcessAll() error {
	// Get all date directories (e.g., sessions/2025-11-26)
	entries, err := os.ReadDir(bp.SessionRoot)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it looks like a date directory (simple check)
			if _, err := time.Parse("2006-01-02", entry.Name()); err == nil {
				if err := bp.ProcessSession(filepath.Join(bp.SessionRoot, entry.Name())); err != nil {
					fmt.Printf("Error processing session %s: %v\n", entry.Name(), err)
				}
			}
		}
	}
	return nil
}

// ProcessSession processes a specific date directory
func (bp *BatchProcessor) ProcessSession(sessionDir string) error {
	// Get all app directories
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			appDir := filepath.Join(sessionDir, entry.Name())
			if err := bp.ProcessApp(appDir); err != nil {
				fmt.Printf("Error processing app %s: %v\n", entry.Name(), err)
			}
		}
	}
	return nil
}

// ProcessApp handles screenshots for a specific app
func (bp *BatchProcessor) ProcessApp(appDir string) error {
	// 1. Find all PNG files
	files, err := filepath.Glob(filepath.Join(appDir, "*.png"))
	if err != nil {
		return err
	}

	// Filter out "latest.png" from the list of files to process
	var timestampedFiles []string
	for _, f := range files {
		if filepath.Base(f) != "latest.png" {
			timestampedFiles = append(timestampedFiles, f)
		}
	}

	if len(timestampedFiles) == 0 {
		return nil
	}

	// 2. Sort by modification time (or filename if they are timestamps)
	sort.Strings(timestampedFiles)

	// 3. Process all files
	// We process ALL timestamped files. The very last one (newest) becomes "latest.png"
	// But wait, if we are running continuously, we might want to keep the last one as a timestamped file UNTIL a newer one arrives?
	// No, the requirement is "delete the screenshots that have been extracted. except the latest one".

	// Strategy:
	// - Identify the newest file in timestampedFiles.
	// - If there is already a "latest.png", we can process it (extract text) and delete it?
	// - Actually, "latest.png" is just a copy or rename of the newest state.

	// Revised Strategy:
	// 1. OCR all timestamped files.
	// 2. Append text to ocr.txt.
	// 3. Delete all processed timestamped files.
	// 4. BUT, before deleting the *newest* timestamped file, copy/rename it to "latest.png".

	ocrFile := filepath.Join(appDir, "ocr.txt")
	f, err := os.OpenFile(ocrFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, file := range timestampedFiles {
		// Extract text
		text, err := ocr.ExtractText(file)
		if err != nil {
			fmt.Printf("OCR failed for %s: %v\n", file, err)
			continue
		}

		// Only append if text is found
		if strings.TrimSpace(text) != "" {
			timestamp := strings.TrimSuffix(filepath.Base(file), ".png")
			entry := fmt.Sprintf("\n[%s]\n%s\n", timestamp, text)
			if _, err := f.WriteString(entry); err != nil {
				fmt.Printf("Failed to write to ocr.txt: %v\n", err)
				continue
			}
		}

		// If this is the last (newest) file, make it the new "latest.png"
		if i == len(timestampedFiles)-1 {
			latestPath := filepath.Join(appDir, "latest.png")
			// We copy instead of rename if we want to keep the timestamped one?
			// No, we want to delete the timestamped ones to save space.
			// So we Rename the newest timestamped file to "latest.png".
			// But wait, if we rename it, next time we scan, we won't see it as a timestamped file.
			// That's fine! It's already processed (OCR'd).

			// However, if we rename it, we lose the timestamp in the filename.
			// The UI might need the timestamp.
			// But "latest.png" implies "current state".

			// Let's Rename it to latest.png (overwriting old latest.png)
			// This effectively "deletes" the timestamped file from the file system view,
			// but keeps the image content available as latest.png.

			// Remove old latest.png if exists
			os.Remove(latestPath)
			if err := os.Rename(file, latestPath); err != nil {
				fmt.Printf("Failed to update latest.png: %v\n", err)
			}
		} else {
			// Delete older processed screenshots
			if err := os.Remove(file); err != nil {
				fmt.Printf("Failed to delete %s: %v\n", file, err)
			}
		}
	}

	return nil
}
