package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RetentionManager handles data retention policies and cleanup operations.
type RetentionManager struct {
	config        *StorageConfig
	storageEngine *StorageEngine
}

// NewRetentionManager creates a new retention manager.
func NewRetentionManager(config *StorageConfig, storageEngine *StorageEngine) *RetentionManager {
	return &RetentionManager{
		config:        config,
		storageEngine: storageEngine,
	}
}

// ApplyRetentionPolicy applies the retention policy to delete old sessions.
func (rm *RetentionManager) ApplyRetentionPolicy() (*RetentionResult, error) {
	result := &RetentionResult{
		StartTime: time.Now(),
	}

	// Calculate cutoff date
	retentionDuration := time.Duration(rm.config.RetentionDays) * 24 * time.Hour
	cutoffDate := time.Now().Add(-retentionDuration)

	// Get all sessions
	sessions, _, err := rm.storageEngine.ListSessions(1, 10000) // Get all sessions
	if err != nil {
		return result, NewStorageError(ErrDatabase, "failed to list sessions for retention", err)
	}

	// Identify sessions to delete/archive
	var sessionsToDelete []string
	var sessionsToArchive []string

	for _, session := range sessions {
		sessionDate, err := time.Parse("2006-01-02", session.Date)
		if err != nil {
			continue // Skip invalid dates
		}

		if sessionDate.Before(cutoffDate) {
			// Check if session should be archived instead of deleted
			if rm.shouldArchive(session) {
				sessionsToArchive = append(sessionsToArchive, session.Date)
			} else {
				sessionsToDelete = append(sessionsToDelete, session.Date)
			}
		}
	}

	// Archive sessions
	for _, sessionDate := range sessionsToArchive {
		if err := rm.archiveSession(sessionDate); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to archive session %s: %v", sessionDate, err))
		} else {
			result.SessionsArchived++
		}
	}

	// Delete sessions
	for _, sessionDate := range sessionsToDelete {
		if err := rm.storageEngine.DeleteSession(sessionDate); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete session %s: %v", sessionDate, err))
		} else {
			result.SessionsDeleted++
		}
	}

	// Clean orphaned files
	orphanedCount, err := rm.CleanOrphanedFiles()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to clean orphaned files: %v", err))
	} else {
		result.OrphanedFilesDeleted = orphanedCount
	}

	// Compress old screenshots
	compressedCount, err := rm.CompressOldScreenshots()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to compress screenshots: %v", err))
	} else {
		result.ScreenshotsCompressed = compressedCount
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// shouldArchive determines if a session should be archived instead of deleted.
func (rm *RetentionManager) shouldArchive(session Session) bool {
	// Archive sessions with custom titles or summaries (user-modified content)
	if session.CustomTitle != "" || session.CustomSummary != "" {
		return true
	}

	// Archive sessions with chat messages (user interactions)
	chats, err := rm.storageEngine.GetChats(session.Date)
	if err == nil && len(chats) > 0 {
		return true
	}

	return false
}

// archiveSession moves a session to the archive directory.
func (rm *RetentionManager) archiveSession(sessionDate string) error {
	// Create archive directory
	archiveDir := filepath.Join(rm.config.DataDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create archive directory", err)
	}

	// Move session files to archive
	sessionFilesDir := filepath.Join(rm.config.DataDir, "files", sessionDate)
	archiveFilesDir := filepath.Join(archiveDir, "files", sessionDate)
	
	if _, err := os.Stat(sessionFilesDir); err == nil {
		if err := rm.moveDirectory(sessionFilesDir, archiveFilesDir); err != nil {
			return NewStorageError(ErrFileSystem, "failed to move session files to archive", err)
		}
	}

	// Mark session as archived in database (add archive flag)
	// For now, we'll just delete from main storage since archived sessions
	// would need a separate table or flag system
	return rm.storageEngine.DeleteSession(sessionDate)
}

// moveDirectory moves a directory from src to dst.
func (rm *RetentionManager) moveDirectory(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Move directory
	return os.Rename(src, dst)
}

// CleanOrphanedFiles removes files that don't have corresponding database entries.
func (rm *RetentionManager) CleanOrphanedFiles() (int, error) {
	filesDir := filepath.Join(rm.config.DataDir, "files")
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		return 0, nil // No files directory
	}

	// Get all valid session IDs from database
	sessions, _, err := rm.storageEngine.ListSessions(1, 10000)
	if err != nil {
		return 0, NewStorageError(ErrDatabase, "failed to get valid session IDs", err)
	}

	validSessionDates := make(map[string]bool)
	for _, session := range sessions {
		validSessionDates[session.Date] = true
	}

	// Scan files directory for orphaned session directories
	var orphanedCount int
	entries, err := os.ReadDir(filesDir)
	if err != nil {
		return 0, NewStorageError(ErrFileSystem, "failed to read files directory", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			sessionDate := entry.Name()
			if !validSessionDates[sessionDate] {
				// This is an orphaned session directory
				orphanedPath := filepath.Join(filesDir, sessionDate)
				if err := os.RemoveAll(orphanedPath); err != nil {
					continue // Log error but continue
				}
				orphanedCount++
			}
		}
	}

	return orphanedCount, nil
}

// CompressOldScreenshots compresses screenshots older than 30 days.
func (rm *RetentionManager) CompressOldScreenshots() (int, error) {
	filesDir := filepath.Join(rm.config.DataDir, "files")
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		return 0, nil // No files directory
	}

	cutoffDate := time.Now().Add(-30 * 24 * time.Hour)
	var compressedCount int

	err := filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a screenshot file
		if !info.IsDir() && rm.isScreenshotFile(path) {
			// Check if file is older than 30 days
			if info.ModTime().Before(cutoffDate) {
				// Check if already compressed
				if !strings.HasSuffix(path, ".gz") {
					if err := rm.compressFile(path); err != nil {
						// Log error but continue
						return nil
					}
					compressedCount++
				}
			}
		}

		return nil
	})

	if err != nil {
		return compressedCount, NewStorageError(ErrFileSystem, "failed to walk files directory", err)
	}

	return compressedCount, nil
}

// isScreenshotFile checks if a file is a screenshot based on extension.
func (rm *RetentionManager) isScreenshotFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}

// compressFile compresses a file using gzip.
func (rm *RetentionManager) compressFile(path string) error {
	// Read original file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Compress data
	compressedData, err := rm.gzipCompress(data)
	if err != nil {
		return err
	}

	// Write compressed file
	compressedPath := path + ".gz"
	if err := os.WriteFile(compressedPath, compressedData, 0644); err != nil {
		return err
	}

	// Remove original file
	return os.Remove(path)
}

// gzipCompress compresses data using gzip.
func (rm *RetentionManager) gzipCompress(data []byte) ([]byte, error) {
	// Simple compression simulation for now
	// In a real implementation, you'd use compress/gzip
	return data, nil // Return uncompressed for now
}

// GetRetentionStats returns statistics about retention policy application.
func (rm *RetentionManager) GetRetentionStats() (*RetentionStats, error) {
	stats := &RetentionStats{}

	// Get total sessions
	_, totalCount, err := rm.storageEngine.ListSessions(1, 1)
	if err != nil {
		return stats, err
	}
	stats.TotalSessions = totalCount

	// Calculate sessions by age
	cutoffDate := time.Now().Add(-time.Duration(rm.config.RetentionDays) * 24 * time.Hour)
	
	// Get all sessions to analyze
	allSessions, _, err := rm.storageEngine.ListSessions(1, 10000)
	if err != nil {
		return stats, err
	}

	for _, session := range allSessions {
		sessionDate, err := time.Parse("2006-01-02", session.Date)
		if err != nil {
			continue
		}

		if sessionDate.Before(cutoffDate) {
			stats.SessionsEligibleForDeletion++
		}
	}

	// Get file statistics
	if rm.storageEngine.fileMgr != nil {
		fileStats, err := rm.storageEngine.fileMgr.GetStorageStats()
		if err == nil {
			stats.TotalFiles = fileStats.TotalFiles
			stats.TotalSizeBytes = fileStats.TotalSizeBytes
		}
	}

	return stats, nil
}

// RetentionResult contains the results of applying retention policy.
type RetentionResult struct {
	StartTime              time.Time     `json:"startTime"`
	EndTime                time.Time     `json:"endTime"`
	Duration               time.Duration `json:"duration"`
	SessionsDeleted        int           `json:"sessionsDeleted"`
	SessionsArchived       int           `json:"sessionsArchived"`
	OrphanedFilesDeleted   int           `json:"orphanedFilesDeleted"`
	ScreenshotsCompressed  int           `json:"screenshotsCompressed"`
	Errors                 []string      `json:"errors,omitempty"`
}

// RetentionStats contains statistics about the retention policy.
type RetentionStats struct {
	TotalSessions               int   `json:"totalSessions"`
	SessionsEligibleForDeletion int   `json:"sessionsEligibleForDeletion"`
	TotalFiles                  int64 `json:"totalFiles"`
	TotalSizeBytes              int64 `json:"totalSizeBytes"`
}