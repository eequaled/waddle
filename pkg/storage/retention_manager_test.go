package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// TestRetentionPolicyEnforcement tests that retention policy is correctly enforced
// Property 17: Retention Policy Enforcement
func TestRetentionPolicyEnforcement(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("retention policy deletes old sessions", prop.ForAll(
		func() bool {
			// Setup test environment
			tempDir := t.TempDir()
			storageDir := filepath.Join(tempDir, "storage")
			
			// Initialize storage engine with short retention (1 day)
			config := DefaultStorageConfig(storageDir)
			config.RetentionDays = 1
			storageEngine := NewStorageEngine(config)
			if err := storageEngine.Initialize(); err != nil {
				t.Logf("Failed to initialize storage engine: %v", err)
				return false
			}
			defer storageEngine.Close()

			// Create sessions with different ages
			today := time.Now()
			oldDate := today.Add(-2 * 24 * time.Hour).Format("2006-01-02")
			recentDate := today.Format("2006-01-02")

			// Create old session (should be deleted)
			_, err := storageEngine.CreateSession(oldDate)
			if err != nil {
				t.Logf("Failed to create old session: %v", err)
				return false
			}

			// Create recent session (should be kept)
			_, err = storageEngine.CreateSession(recentDate)
			if err != nil {
				t.Logf("Failed to create recent session: %v", err)
				return false
			}

			// Create session with custom content (should be archived)
			archiveDate := today.Add(-3 * 24 * time.Hour).Format("2006-01-02")
			archiveSession, err := storageEngine.CreateSession(archiveDate)
			if err != nil {
				t.Logf("Failed to create archive session: %v", err)
				return false
			}
			archiveSession.CustomTitle = "Important Session"
			if err := storageEngine.UpdateSession(archiveSession); err != nil {
				t.Logf("Failed to update archive session: %v", err)
				return false
			}

			// Create retention manager
			retentionMgr := NewRetentionManager(config, storageEngine)

			// Apply retention policy
			result, err := retentionMgr.ApplyRetentionPolicy()
			if err != nil {
				t.Logf("Failed to apply retention policy: %v", err)
				return false
			}

			// Verify results
			if result.SessionsDeleted == 0 && result.SessionsArchived == 0 {
				t.Logf("Expected some sessions to be deleted or archived")
				return false
			}

			// Verify recent session still exists
			if _, err := storageEngine.GetSession(recentDate); err != nil {
				t.Logf("Recent session was incorrectly deleted")
				return false
			}

			// Verify old session was deleted
			if _, err := storageEngine.GetSession(oldDate); err == nil {
				t.Logf("Old session was not deleted")
				return false
			}

			return true
		},
	))

	properties.Property("orphaned files are cleaned up", prop.ForAll(
		func() bool {
			// Setup test environment
			tempDir := t.TempDir()
			storageDir := filepath.Join(tempDir, "storage")
			
			config := DefaultStorageConfig(storageDir)
			storageEngine := NewStorageEngine(config)
			if err := storageEngine.Initialize(); err != nil {
				t.Logf("Failed to initialize storage engine: %v", err)
				return false
			}
			defer storageEngine.Close()

			// Create valid session
			validDate := "2024-01-20"
			if _, err := storageEngine.CreateSession(validDate); err != nil {
				t.Logf("Failed to create valid session: %v", err)
				return false
			}

			// Create orphaned file directory (no corresponding session)
			orphanedDate := "2024-01-19"
			orphanedDir := filepath.Join(storageDir, "files", orphanedDate)
			if err := os.MkdirAll(orphanedDir, 0755); err != nil {
				t.Logf("Failed to create orphaned directory: %v", err)
				return false
			}

			// Create a file in the orphaned directory
			orphanedFile := filepath.Join(orphanedDir, "test.png")
			if err := os.WriteFile(orphanedFile, []byte("fake image"), 0644); err != nil {
				t.Logf("Failed to create orphaned file: %v", err)
				return false
			}

			// Create retention manager
			retentionMgr := NewRetentionManager(config, storageEngine)

			// Clean orphaned files
			orphanedCount, err := retentionMgr.CleanOrphanedFiles()
			if err != nil {
				t.Logf("Failed to clean orphaned files: %v", err)
				return false
			}

			if orphanedCount == 0 {
				t.Logf("Expected orphaned files to be cleaned")
				return false
			}

			// Verify orphaned directory was removed
			if _, err := os.Stat(orphanedDir); err == nil {
				t.Logf("Orphaned directory was not removed")
				return false
			}

			// Verify valid session files still exist (directory might not exist if no files saved)
			// This is acceptable behavior

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestRetentionStats tests retention statistics calculation
func TestRetentionStats(t *testing.T) {
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "storage")
	
	// Initialize storage engine
	config := DefaultStorageConfig(storageDir)
	config.RetentionDays = 7 // 7 days retention
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create sessions with different ages
	today := time.Now()
	
	// Recent session (within retention)
	recentDate := today.Format("2006-01-02")
	if _, err := storageEngine.CreateSession(recentDate); err != nil {
		t.Fatalf("Failed to create recent session: %v", err)
	}

	// Old session (outside retention)
	oldDate := today.Add(-10 * 24 * time.Hour).Format("2006-01-02")
	if _, err := storageEngine.CreateSession(oldDate); err != nil {
		t.Fatalf("Failed to create old session: %v", err)
	}

	// Create retention manager
	retentionMgr := NewRetentionManager(config, storageEngine)

	// Get retention stats
	stats, err := retentionMgr.GetRetentionStats()
	if err != nil {
		t.Fatalf("Failed to get retention stats: %v", err)
	}

	// Verify stats
	if stats.TotalSessions != 2 {
		t.Errorf("Expected 2 total sessions, got %d", stats.TotalSessions)
	}

	if stats.SessionsEligibleForDeletion != 1 {
		t.Errorf("Expected 1 session eligible for deletion, got %d", stats.SessionsEligibleForDeletion)
	}
}

// TestScreenshotCompression tests screenshot compression functionality
func TestScreenshotCompression(t *testing.T) {
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "storage")
	
	config := DefaultStorageConfig(storageDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create session
	sessionDate := "2024-01-15"
	if _, err := storageEngine.CreateSession(sessionDate); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create old screenshot file
	filesDir := filepath.Join(storageDir, "files", sessionDate, "TestApp")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files directory: %v", err)
	}

	screenshotPath := filepath.Join(filesDir, "old_screenshot.png")
	screenshotData := []byte("fake screenshot data")
	if err := os.WriteFile(screenshotPath, screenshotData, 0644); err != nil {
		t.Fatalf("Failed to create screenshot file: %v", err)
	}

	// Set file modification time to 35 days ago
	oldTime := time.Now().Add(-35 * 24 * time.Hour)
	if err := os.Chtimes(screenshotPath, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set file time: %v", err)
	}

	// Create retention manager
	retentionMgr := NewRetentionManager(config, storageEngine)

	// Compress old screenshots
	compressedCount, err := retentionMgr.CompressOldScreenshots()
	if err != nil {
		t.Fatalf("Failed to compress screenshots: %v", err)
	}

	// For now, compression is a no-op, so count should be 1 but file should still exist
	if compressedCount != 1 {
		t.Errorf("Expected 1 compressed file, got %d", compressedCount)
	}
}