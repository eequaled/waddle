package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// TestBackupRestoreRoundTrip tests that backup and restore preserve data integrity
// Property 16: Backup Restore Round-Trip
func TestBackupRestoreRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("backup restore preserves data integrity", prop.ForAll(
		func() bool {
			// Setup test environment
			tempDir := t.TempDir()
			storageDir := filepath.Join(tempDir, "storage")
			
			// Initialize storage engine
			config := DefaultStorageConfig(storageDir)
			storageEngine := NewStorageEngine(config)
			if err := storageEngine.Initialize(); err != nil {
				t.Logf("Failed to initialize storage engine: %v", err)
				return false
			}
			defer storageEngine.Close()

			// Create test data
			session, err := storageEngine.CreateSession("2024-01-20")
			if err != nil {
				t.Logf("Failed to create session: %v", err)
				return false
			}

			session.CustomTitle = "Test Backup Session"
			session.CustomSummary = "Test backup summary"
			if err := storageEngine.UpdateSession(session); err != nil {
				t.Logf("Failed to update session: %v", err)
				return false
			}

			// Add activity block
			block := &ActivityBlock{
				BlockID:      "10-30",
				StartTime:    time.Now(),
				EndTime:      time.Now().Add(5 * time.Minute),
				OCRText:      "Test OCR content for backup",
				MicroSummary: "Test micro summary",
			}
			if err := storageEngine.AddActivityBlock("2024-01-20", "TestApp", block); err != nil {
				t.Logf("Failed to add activity block: %v", err)
				return false
			}

			// Add notification
			notification := &Notification{
				ID:        "backup-test-notif",
				Type:      "test",
				Title:     "Backup Test",
				Message:   "Test notification for backup",
				Timestamp: time.Now(),
				Read:      false,
			}
			if err := storageEngine.AddNotification(notification); err != nil {
				t.Logf("Failed to add notification: %v", err)
				return false
			}

			// Create backup manager
			backupMgr := NewBackupManager(config, storageEngine)

			// Create backup
			backupPath, err := backupMgr.CreateBackup()
			if err != nil {
				t.Logf("Failed to create backup: %v", err)
				return false
			}

			// Verify backup
			if err := backupMgr.VerifyBackup(backupPath); err != nil {
				t.Logf("Backup verification failed: %v", err)
				return false
			}

			// Modify data after backup
			session.CustomTitle = "Modified Title"
			if err := storageEngine.UpdateSession(session); err != nil {
				t.Logf("Failed to modify session: %v", err)
				return false
			}

			// Restore from backup
			if err := backupMgr.Restore(backupPath); err != nil {
				t.Logf("Failed to restore from backup: %v", err)
				return false
			}

			// Verify data was restored
			restoredSession, err := storageEngine.GetSession("2024-01-20")
			if err != nil {
				t.Logf("Failed to get restored session: %v", err)
				return false
			}

			if restoredSession.CustomTitle != "Test Backup Session" {
				t.Logf("Session title not restored correctly: expected 'Test Backup Session', got '%s'", restoredSession.CustomTitle)
				return false
			}

			// Verify activity blocks were restored
			blocks, err := storageEngine.GetActivityBlocks("2024-01-20", "TestApp")
			if err != nil {
				t.Logf("Failed to get restored activity blocks: %v", err)
				return false
			}

			if len(blocks) != 1 {
				t.Logf("Expected 1 activity block, got %d", len(blocks))
				return false
			}

			if blocks[0].OCRText != "Test OCR content for backup" {
				t.Logf("Activity block OCR text not restored correctly")
				return false
			}

			// Verify notifications were restored
			notifications, err := storageEngine.GetNotifications(10)
			if err != nil {
				t.Logf("Failed to get restored notifications: %v", err)
				return false
			}

			found := false
			for _, notif := range notifications {
				if notif.ID == "backup-test-notif" {
					found = true
					if notif.Title != "Backup Test" {
						t.Logf("Notification title not restored correctly")
						return false
					}
					break
				}
			}

			if !found {
				t.Logf("Test notification not found after restore")
				return false
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestBackupCreation tests basic backup creation functionality
func TestBackupCreation(t *testing.T) {
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "storage")
	
	// Initialize storage engine
	config := DefaultStorageConfig(storageDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create test data
	session, err := storageEngine.CreateSession("2024-01-21")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	session.CustomTitle = "Backup Test Session"
	if err := storageEngine.UpdateSession(session); err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Create backup manager
	backupMgr := NewBackupManager(config, storageEngine)

	// Create backup
	backupPath, err := backupMgr.CreateBackup()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup directory exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}

	// Verify backup metadata exists
	metadataPath := filepath.Join(backupPath, "backup_metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("Backup metadata was not created")
	}

	// Verify database backup exists
	dbBackupPath := filepath.Join(backupPath, "waddle.db")
	if _, err := os.Stat(dbBackupPath); os.IsNotExist(err) {
		t.Error("Database backup was not created")
	}

	// Verify backup
	if err := backupMgr.VerifyBackup(backupPath); err != nil {
		t.Errorf("Backup verification failed: %v", err)
	}
}

// TestBackupRetention tests backup cleanup functionality
func TestBackupRetention(t *testing.T) {
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "storage")
	
	// Initialize storage engine with short retention
	config := DefaultStorageConfig(storageDir)
	config.RetentionDays = 1 // 1 day retention for testing
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create backup manager
	backupMgr := NewBackupManager(config, storageEngine)

	// Create multiple backups with different timestamps
	backupPaths := make([]string, 3)
	for i := 0; i < 3; i++ {
		// Add delay to ensure different timestamps
		if i > 0 {
			time.Sleep(10 * time.Millisecond)
		}
		
		backupPath, err := backupMgr.CreateBackup()
		if err != nil {
			t.Fatalf("Failed to create backup %d: %v", i, err)
		}
		backupPaths[i] = backupPath

		// Modify timestamp to simulate old backups
		if i < 2 {
			oldTime := time.Now().Add(-time.Duration(2+i) * 24 * time.Hour)
			if err := os.Chtimes(backupPath, oldTime, oldTime); err != nil {
				t.Logf("Failed to modify backup timestamp: %v", err)
			}
		}
	}

	// List backups before cleanup
	backups, err := backupMgr.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(backups))
	}

	// Run cleanup
	if err := backupMgr.CleanupOldBackups(); err != nil {
		t.Fatalf("Failed to cleanup old backups: %v", err)
	}

	// List backups after cleanup
	backups, err = backupMgr.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups after cleanup: %v", err)
	}

	// Should have fewer backups now (exact count depends on timing)
	if len(backups) > 2 {
		t.Errorf("Expected cleanup to remove old backups, still have %d", len(backups))
	}
}