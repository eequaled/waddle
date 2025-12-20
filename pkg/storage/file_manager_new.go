package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileManager handles filesystem operations for binary assets (screenshots).
type FileManager struct {
	baseDir string // ~/.waddle/files/
}

// NewFileManager creates a new FileManager instance.
func NewFileManager(dataDir string) *FileManager {
	return &FileManager{
		baseDir: filepath.Join(dataDir, "files"),
	}
}

// Initialize creates the base directory if it doesn't exist.
func (fm *FileManager) Initialize() error {
	if err := os.MkdirAll(fm.baseDir, 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create files directory", err)
	}
	return nil
}

// SaveFile saves a file and returns the path where it was stored.
// Path format: {baseDir}/{sessionID}/{appName}/screenshots/{filename}
func (fm *FileManager) SaveFile(sessionID, appName, filename string, data []byte) (string, error) {
	if sessionID == "" {
		return "", NewStorageError(ErrValidation, "session ID is required", nil)
	}
	if appName == "" {
		return "", NewStorageError(ErrValidation, "app name is required", nil)
	}
	if filename == "" {
		return "", NewStorageError(ErrValidation, "filename is required", nil)
	}

	// Sanitize inputs
	safeSessionID := sanitizePathComponent(sessionID)
	safeAppName := sanitizePathComponent(appName)
	safeFilename := sanitizePathComponent(filename)

	// Build path
	dir := filepath.Join(fm.baseDir, safeSessionID, safeAppName, "screenshots")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", NewStorageError(ErrFileSystem, "failed to create directory", err)
	}

	fullPath := filepath.Join(dir, safeFilename)

	// Write file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", NewStorageError(ErrFileSystem, "failed to write file", err)
	}

	// Return relative path from baseDir
	relPath, _ := filepath.Rel(fm.baseDir, fullPath)
	return relPath, nil
}

// SaveLatestScreenshot saves the latest screenshot for an app.
// Path format: {baseDir}/{sessionID}/{appName}/latest.png
func (fm *FileManager) SaveLatestScreenshot(sessionID, appName string, data []byte) (string, error) {
	if sessionID == "" {
		return "", NewStorageError(ErrValidation, "session ID is required", nil)
	}
	if appName == "" {
		return "", NewStorageError(ErrValidation, "app name is required", nil)
	}

	safeSessionID := sanitizePathComponent(sessionID)
	safeAppName := sanitizePathComponent(appName)

	dir := filepath.Join(fm.baseDir, safeSessionID, safeAppName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", NewStorageError(ErrFileSystem, "failed to create directory", err)
	}

	fullPath := filepath.Join(dir, "latest.png")

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", NewStorageError(ErrFileSystem, "failed to write file", err)
	}

	relPath, _ := filepath.Rel(fm.baseDir, fullPath)
	return relPath, nil
}

// GetFilePath returns the full filesystem path for a file.
func (fm *FileManager) GetFilePath(sessionID, appName, filename string) string {
	safeSessionID := sanitizePathComponent(sessionID)
	safeAppName := sanitizePathComponent(appName)
	safeFilename := sanitizePathComponent(filename)

	return filepath.Join(fm.baseDir, safeSessionID, safeAppName, "screenshots", safeFilename)
}

// GetLatestScreenshotPath returns the path to the latest screenshot for an app.
func (fm *FileManager) GetLatestScreenshotPath(sessionID, appName string) string {
	safeSessionID := sanitizePathComponent(sessionID)
	safeAppName := sanitizePathComponent(appName)

	return filepath.Join(fm.baseDir, safeSessionID, safeAppName, "latest.png")
}

// FileExists checks if a file exists at the given path.
func (fm *FileManager) FileExists(path string) bool {
	fullPath := filepath.Join(fm.baseDir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ReadFile reads a file from the storage.
func (fm *FileManager) ReadFile(path string) ([]byte, error) {
	fullPath := filepath.Join(fm.baseDir, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewStorageError(ErrNotFound, "file not found", err)
		}
		return nil, NewStorageError(ErrFileSystem, "failed to read file", err)
	}
	return data, nil
}

// DeleteSessionFiles deletes all files for a session.
func (fm *FileManager) DeleteSessionFiles(sessionID string) error {
	if sessionID == "" {
		return NewStorageError(ErrValidation, "session ID is required", nil)
	}

	safeSessionID := sanitizePathComponent(sessionID)
	sessionDir := filepath.Join(fm.baseDir, safeSessionID)

	// Check if directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil // Nothing to delete
	}

	if err := os.RemoveAll(sessionDir); err != nil {
		return NewStorageError(ErrFileSystem, "failed to delete session files", err)
	}

	return nil
}

// CleanOrphanedFiles removes files that don't have corresponding session IDs.
// Returns the number of files deleted.
func (fm *FileManager) CleanOrphanedFiles(validSessionIDs []string) (int, error) {
	validSet := make(map[string]bool)
	for _, id := range validSessionIDs {
		validSet[sanitizePathComponent(id)] = true
	}

	entries, err := os.ReadDir(fm.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, NewStorageError(ErrFileSystem, "failed to read files directory", err)
	}

	deletedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionID := entry.Name()
		if !validSet[sessionID] {
			sessionDir := filepath.Join(fm.baseDir, sessionID)
			if err := os.RemoveAll(sessionDir); err != nil {
				// Log but continue
				continue
			}
			deletedCount++
		}
	}

	return deletedCount, nil
}

// CompressOldScreenshots compresses screenshots older than the specified duration.
// Note: This is a placeholder - actual compression would require image processing.
func (fm *FileManager) CompressOldScreenshots(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	return filepath.Walk(fm.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Only process PNG files
		if !strings.HasSuffix(strings.ToLower(path), ".png") {
			return nil
		}

		// Skip if not old enough
		if info.ModTime().After(cutoff) {
			return nil
		}

		// TODO: Implement actual compression (e.g., convert to JPEG with lower quality)
		// For now, this is a no-op placeholder

		return nil
	})
}

// GetStorageStats returns statistics about file storage.
func (fm *FileManager) GetStorageStats() (*StorageStats, error) {
	stats := &StorageStats{}

	err := filepath.Walk(fm.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		stats.TotalFiles++
		stats.TotalSizeBytes += info.Size()

		if strings.HasSuffix(strings.ToLower(path), ".png") {
			stats.ScreenshotCount++
		}

		if stats.OldestFile.IsZero() || info.ModTime().Before(stats.OldestFile) {
			stats.OldestFile = info.ModTime()
		}

		return nil
	})

	if err != nil {
		return nil, NewStorageError(ErrFileSystem, "failed to get storage stats", err)
	}

	return stats, nil
}

// ListSessionFiles lists all files for a session.
func (fm *FileManager) ListSessionFiles(sessionID string) ([]string, error) {
	safeSessionID := sanitizePathComponent(sessionID)
	sessionDir := filepath.Join(fm.baseDir, safeSessionID)

	var files []string

	err := filepath.Walk(sessionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(fm.baseDir, path)
		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, NewStorageError(ErrFileSystem, "failed to list session files", err)
	}

	return files, nil
}

// CopyFile copies a file from src to dst.
func (fm *FileManager) CopyFile(src, dst string) error {
	srcPath := filepath.Join(fm.baseDir, src)
	dstPath := filepath.Join(fm.baseDir, dst)

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create destination directory", err)
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return NewStorageError(ErrFileSystem, "failed to open source file", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return NewStorageError(ErrFileSystem, "failed to create destination file", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return NewStorageError(ErrFileSystem, "failed to copy file", err)
	}

	return nil
}

// sanitizePathComponent removes or replaces characters that are unsafe for file paths.
func sanitizePathComponent(name string) string {
	// Replace unsafe characters
	unsafe := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*", "..", "\x00"}
	result := name
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	return strings.TrimSpace(result)
}

// GetBaseDir returns the base directory for file storage.
func (fm *FileManager) GetBaseDir() string {
	return fm.baseDir
}

// GetFullPath returns the full path for a relative path.
func (fm *FileManager) GetFullPath(relPath string) string {
	return filepath.Join(fm.baseDir, relPath)
}

// EnsureDir ensures a directory exists within the base directory.
func (fm *FileManager) EnsureDir(relPath string) error {
	fullPath := filepath.Join(fm.baseDir, relPath)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return NewStorageError(ErrFileSystem, fmt.Sprintf("failed to create directory: %s", relPath), err)
	}
	return nil
}
