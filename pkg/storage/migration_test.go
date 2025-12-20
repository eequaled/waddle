package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// TestMigrationDataIntegrity tests that migration preserves data integrity
// Property 11: Migration Data Integrity
func TestMigrationDataIntegrity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Migration preserves session data integrity
	properties.Property("migration preserves session data", prop.ForAll(
		func() bool {
			// Setup test environment
			tempDir := t.TempDir()
			legacyDir := filepath.Join(tempDir, "legacy", "sessions")
			storageDir := filepath.Join(tempDir, "storage")

			// Create test legacy data
			if err := createTestLegacyData(legacyDir); err != nil {
				t.Logf("Failed to create test legacy data: %v", err)
				return false
			}

			// Initialize storage engine
			config := DefaultStorageConfig(storageDir)
			storageEngine := NewStorageEngine(config)
			if err := storageEngine.Initialize(); err != nil {
				t.Logf("Failed to initialize storage engine: %v", err)
				return false
			}
			defer storageEngine.Close()

			// Initialize migration manager
			migrationMgr := NewMigrationManager(config, legacyDir)

			// Get initial state
			state, err := migrationMgr.GetState()
			if err != nil {
				t.Logf("Failed to get migration state: %v", err)
				return false
			}

			// Detect legacy data
			hasLegacy, err := migrationMgr.DetectLegacyData()
			if err != nil {
				t.Logf("Failed to detect legacy data: %v", err)
				return false
			}
			if !hasLegacy {
				t.Logf("No legacy data detected")
				return false
			}

			// Transition to detecting
			if err := migrationMgr.TransitionTo(state, MigrationStatusDetecting, "Starting migration"); err != nil {
				t.Logf("Failed to transition to detecting: %v", err)
				return false
			}

			// Create backup
			if err := migrationMgr.TransitionTo(state, MigrationStatusBackingUp, "Creating backup"); err != nil {
				t.Logf("Failed to transition to backing up: %v", err)
				return false
			}

			if err := migrationMgr.CreateBackup(state); err != nil {
				t.Logf("Failed to create backup: %v", err)
				return false
			}

			// Migrate data
			if err := migrationMgr.TransitionTo(state, MigrationStatusMigrating, "Migrating data"); err != nil {
				t.Logf("Failed to transition to migrating: %v", err)
				return false
			}

			if err := migrationMgr.MigrateData(state, storageEngine); err != nil {
				t.Logf("Failed to migrate data: %v", err)
				return false
			}

			// Verify data integrity
			return verifyMigrationIntegrity(legacyDir, storageEngine, t)
		},
	))

	// Property: Migration handles empty sessions correctly
	properties.Property("migration handles empty sessions", prop.ForAll(
		func() bool {
			// Setup test environment
			tempDir := t.TempDir()
			legacyDir := filepath.Join(tempDir, "legacy", "sessions")
			storageDir := filepath.Join(tempDir, "storage")

			// Create empty session directory
			sessionDir := filepath.Join(legacyDir, "2024-01-15")
			if err := os.MkdirAll(sessionDir, 0755); err != nil {
				return false
			}

			// Initialize storage engine
			config := DefaultStorageConfig(storageDir)
			storageEngine := NewStorageEngine(config)
			if err := storageEngine.Initialize(); err != nil {
				return false
			}
			defer storageEngine.Close()

			// Initialize migration manager
			migrationMgr := NewMigrationManager(config, legacyDir)

			// Get initial state
			state, err := migrationMgr.GetState()
			if err != nil {
				return false
			}

			// Migrate data
			if err := migrationMgr.MigrateData(state, storageEngine); err != nil {
				return false
			}

			// Verify session was created
			session, err := storageEngine.GetSession("2024-01-15")
			if err != nil {
				return false
			}

			return session.Date == "2024-01-15"
		},
	))

	// Property: Migration preserves file checksums
	properties.Property("migration preserves file checksums", prop.ForAll(
		func() bool {
			// Setup test environment
			tempDir := t.TempDir()
			legacyDir := filepath.Join(tempDir, "legacy", "sessions")
			storageDir := filepath.Join(tempDir, "storage")

			// Create test session with screenshot
			sessionDir := filepath.Join(legacyDir, "2024-01-16")
			appDir := filepath.Join(sessionDir, "TestApp")
			if err := os.MkdirAll(appDir, 0755); err != nil {
				return false
			}

			// Create test screenshot
			screenshotData := []byte("fake screenshot data for testing")
			screenshotPath := filepath.Join(appDir, "test.png")
			if err := os.WriteFile(screenshotPath, screenshotData, 0644); err != nil {
				return false
			}

			// Initialize storage engine
			config := DefaultStorageConfig(storageDir)
			storageEngine := NewStorageEngine(config)
			if err := storageEngine.Initialize(); err != nil {
				return false
			}
			defer storageEngine.Close()

			// Initialize migration manager
			migrationMgr := NewMigrationManager(config, legacyDir)

			// Get initial state
			state, err := migrationMgr.GetState()
			if err != nil {
				return false
			}

			// Migrate data
			if err := migrationMgr.MigrateData(state, storageEngine); err != nil {
				return false
			}

			// Verify file was copied correctly
			newPath := storageEngine.GetScreenshotPath("2024-01-16", "TestApp", "test.png")
			newData, err := os.ReadFile(newPath)
			if err != nil {
				return false
			}

			// Compare data
			if len(newData) != len(screenshotData) {
				return false
			}

			for i, b := range screenshotData {
				if newData[i] != b {
					return false
				}
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// createTestLegacyData creates test legacy data structure
func createTestLegacyData(legacyDir string) error {
	// Create session directory
	sessionDir := filepath.Join(legacyDir, "2024-01-15")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	// Create metadata.json
	metadata := map[string]interface{}{
		"customTitle":     "Test Session",
		"customSummary":   "Test custom summary",
		"originalSummary": "Test original summary",
		"extractedText":   "Test extracted text content",
	}
	metadataBytes, _ := json.MarshalIndent(metadata, "", "  ")
	if err := os.WriteFile(filepath.Join(sessionDir, "metadata.json"), metadataBytes, 0644); err != nil {
		return err
	}

	// Create app directory
	appDir := filepath.Join(sessionDir, "TestApp")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}

	// Create blocks directory
	blocksDir := filepath.Join(appDir, "blocks")
	if err := os.MkdirAll(blocksDir, 0755); err != nil {
		return err
	}

	// Create test block
	block := map[string]interface{}{
		"blockId":      "15-30",
		"startTime":    "2024-01-15T15:30:00Z",
		"endTime":      "2024-01-15T15:35:00Z",
		"ocrText":      "Test OCR text content",
		"microSummary": "Test micro summary",
	}
	blockBytes, _ := json.MarshalIndent(block, "", "  ")
	if err := os.WriteFile(filepath.Join(blocksDir, "15-30.json"), blockBytes, 0644); err != nil {
		return err
	}

	// Create test screenshot
	screenshotData := []byte("fake screenshot data")
	if err := os.WriteFile(filepath.Join(appDir, "15-30-00.png"), screenshotData, 0644); err != nil {
		return err
	}

	// Create notifications.json in parent directory
	notificationsDir := filepath.Dir(legacyDir)
	notifications := []map[string]interface{}{
		{
			"id":        "test-notif-1",
			"type":      "test",
			"title":     "Test Notification",
			"message":   "Test notification message",
			"timestamp": "2024-01-15T15:30:00Z",
			"read":      false,
			"metadata":  map[string]string{"key": "value"},
		},
	}
	notifBytes, _ := json.MarshalIndent(notifications, "", "  ")
	if err := os.WriteFile(filepath.Join(notificationsDir, "notifications.json"), notifBytes, 0644); err != nil {
		return err
	}

	return nil
}

// verifyMigrationIntegrity verifies that migrated data matches original data
func verifyMigrationIntegrity(legacyDir string, storageEngine *StorageEngine, t *testing.T) bool {
	// Verify session was migrated
	session, err := storageEngine.GetSession("2024-01-15")
	if err != nil {
		t.Logf("Failed to get migrated session: %v", err)
		return false
	}

	// Verify session metadata
	if session.CustomTitle != "Test Session" {
		t.Logf("Custom title mismatch: expected 'Test Session', got '%s'", session.CustomTitle)
		return false
	}
	if session.CustomSummary != "Test custom summary" {
		t.Logf("Custom summary mismatch: expected 'Test custom summary', got '%s'", session.CustomSummary)
		return false
	}
	if session.OriginalSummary != "Test original summary" {
		t.Logf("Original summary mismatch: expected 'Test original summary', got '%s'", session.OriginalSummary)
		return false
	}
	if session.ExtractedText != "Test extracted text content" {
		t.Logf("Extracted text mismatch: expected 'Test extracted text content', got '%s'", session.ExtractedText)
		return false
	}

	// Verify activity blocks were migrated
	blocks, err := storageEngine.GetActivityBlocks("2024-01-15", "TestApp")
	if err != nil {
		t.Logf("Failed to get activity blocks: %v", err)
		return false
	}

	if len(blocks) != 1 {
		t.Logf("Expected 1 activity block, got %d", len(blocks))
		return false
	}

	block := blocks[0]
	if block.BlockID != "15-30" {
		t.Logf("Block ID mismatch: expected '15-30', got '%s'", block.BlockID)
		return false
	}
	if block.OCRText != "Test OCR text content" {
		t.Logf("OCR text mismatch: expected 'Test OCR text content', got '%s'", block.OCRText)
		return false
	}
	if block.MicroSummary != "Test micro summary" {
		t.Logf("Micro summary mismatch: expected 'Test micro summary', got '%s'", block.MicroSummary)
		return false
	}

	// Verify screenshot was copied
	screenshotPath := storageEngine.GetScreenshotPath("2024-01-15", "TestApp", "15-30-00.png")
	if _, err := os.Stat(screenshotPath); os.IsNotExist(err) {
		t.Logf("Screenshot was not copied: %s", screenshotPath)
		return false
	}

	// Verify notifications were migrated
	notifications, err := storageEngine.GetNotifications(10)
	if err != nil {
		t.Logf("Failed to get notifications: %v", err)
		return false
	}

	found := false
	for _, notif := range notifications {
		if notif.ID == "test-notif-1" {
			found = true
			if notif.Type != "test" {
				t.Logf("Notification type mismatch: expected 'test', got '%s'", notif.Type)
				return false
			}
			if notif.Title != "Test Notification" {
				t.Logf("Notification title mismatch: expected 'Test Notification', got '%s'", notif.Title)
				return false
			}
			break
		}
	}

	if !found {
		t.Logf("Test notification was not migrated")
		return false
	}

	return true
}

// TestMigrationStateTransitions tests the migration state machine
func TestMigrationStateTransitions(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultStorageConfig(tempDir)
	migrationMgr := NewMigrationManager(config, filepath.Join(tempDir, "legacy"))

	// Test initial state
	state, err := migrationMgr.GetState()
	if err != nil {
		t.Fatalf("Failed to get initial state: %v", err)
	}

	if state.Status != MigrationStatusIdle {
		t.Errorf("Expected initial status to be idle, got %s", state.Status)
	}

	// Test valid transition
	err = migrationMgr.TransitionTo(state, MigrationStatusDetecting, "Starting detection")
	if err != nil {
		t.Errorf("Valid transition failed: %v", err)
	}

	if state.Status != MigrationStatusDetecting {
		t.Errorf("Expected status to be detecting, got %s", state.Status)
	}

	// Test invalid transition (from detecting to rolling_back is invalid)
	err = migrationMgr.TransitionTo(state, MigrationStatusRollingBack, "Invalid transition")
	if err == nil {
		t.Error("Invalid transition should have failed")
	}

	// Test checkpoint was added (should still be detecting since invalid transition failed)
	if len(state.Checkpoints) == 0 {
		t.Error("Expected checkpoint to be added")
	}

	checkpoint := state.Checkpoints[len(state.Checkpoints)-1]
	if checkpoint.Name != string(MigrationStatusDetecting) {
		t.Errorf("Expected checkpoint name to be %s, got %s", MigrationStatusDetecting, checkpoint.Name)
	}
}

// TestBackupIntegrity tests backup creation and verification
func TestBackupIntegrity(t *testing.T) {
	tempDir := t.TempDir()
	legacyDir := filepath.Join(tempDir, "legacy", "sessions")
	storageDir := filepath.Join(tempDir, "storage")

	// Create test legacy data
	if err := createTestLegacyData(legacyDir); err != nil {
		t.Fatalf("Failed to create test legacy data: %v", err)
	}

	// Initialize migration manager
	config := DefaultStorageConfig(storageDir)
	migrationMgr := NewMigrationManager(config, legacyDir)

	// Get initial state
	state, err := migrationMgr.GetState()
	if err != nil {
		t.Fatalf("Failed to get migration state: %v", err)
	}

	// Create backup
	if err := migrationMgr.CreateBackup(state); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup path was set
	if state.BackupPath == "" {
		t.Error("Backup path was not set")
	}

	// Verify backup directory exists
	if _, err := os.Stat(state.BackupPath); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}

	// Verify backup contains expected files
	backupSessionsDir := filepath.Join(state.BackupPath, "sessions")
	if _, err := os.Stat(backupSessionsDir); os.IsNotExist(err) {
		t.Error("Backup sessions directory was not created")
	}

	// Verify specific files were backed up
	metadataBackup := filepath.Join(backupSessionsDir, "2024-01-15", "metadata.json")
	if _, err := os.Stat(metadataBackup); os.IsNotExist(err) {
		t.Error("Metadata file was not backed up")
	}

	blockBackup := filepath.Join(backupSessionsDir, "2024-01-15", "TestApp", "blocks", "15-30.json")
	if _, err := os.Stat(blockBackup); os.IsNotExist(err) {
		t.Error("Block file was not backed up")
	}

	screenshotBackup := filepath.Join(backupSessionsDir, "2024-01-15", "TestApp", "15-30-00.png")
	if _, err := os.Stat(screenshotBackup); os.IsNotExist(err) {
		t.Error("Screenshot file was not backed up")
	}
}

// TestRollbackFunctionality tests the rollback functionality
func TestRollbackFunctionality(t *testing.T) {
	tempDir := t.TempDir()
	legacyDir := filepath.Join(tempDir, "legacy", "sessions")
	storageDir := filepath.Join(tempDir, "storage")

	// Create test legacy data
	if err := createTestLegacyData(legacyDir); err != nil {
		t.Fatalf("Failed to create test legacy data: %v", err)
	}

	// Initialize storage engine
	config := DefaultStorageConfig(storageDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}

	// Initialize migration manager
	migrationMgr := NewMigrationManager(config, legacyDir)

	// Get initial state
	state, err := migrationMgr.GetState()
	if err != nil {
		t.Fatalf("Failed to get migration state: %v", err)
	}

	// Follow proper state transitions
	if err := migrationMgr.TransitionTo(state, MigrationStatusDetecting, "Starting migration"); err != nil {
		t.Fatalf("Failed to transition to detecting: %v", err)
	}

	if err := migrationMgr.TransitionTo(state, MigrationStatusBackingUp, "Creating backup"); err != nil {
		t.Fatalf("Failed to transition to backing up: %v", err)
	}

	// Create backup
	if err := migrationMgr.CreateBackup(state); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if err := migrationMgr.TransitionTo(state, MigrationStatusMigrating, "Migrating data"); err != nil {
		t.Fatalf("Failed to transition to migrating: %v", err)
	}

	// Migrate data
	if err := migrationMgr.MigrateData(state, storageEngine); err != nil {
		t.Fatalf("Failed to migrate data: %v", err)
	}

	// Verify migration worked
	session, err := storageEngine.GetSession("2024-01-15")
	if err != nil {
		t.Fatalf("Failed to get migrated session: %v", err)
	}
	if session.CustomTitle != "Test Session" {
		t.Errorf("Expected custom title 'Test Session', got '%s'", session.CustomTitle)
	}

	// Modify the legacy data to simulate changes
	originalMetadataPath := filepath.Join(legacyDir, "2024-01-15", "metadata.json")
	if err := os.Remove(originalMetadataPath); err != nil {
		t.Fatalf("Failed to remove original metadata: %v", err)
	}

	// Transition to rollback state (from migrating)
	if err := migrationMgr.TransitionTo(state, MigrationStatusRollingBack, "Testing rollback"); err != nil {
		t.Fatalf("Failed to transition to rollback: %v", err)
	}

	// Perform rollback
	if err := migrationMgr.Rollback(state, storageEngine); err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify rollback worked
	// Check that legacy data was restored
	if _, err := os.Stat(originalMetadataPath); os.IsNotExist(err) {
		t.Error("Original metadata file was not restored")
	}

	// Check that SQLite database was removed
	dbPath := filepath.Join(storageDir, "waddle.db")
	if _, err := os.Stat(dbPath); err == nil {
		t.Error("SQLite database still exists after rollback")
	}

	// Verify restored metadata content
	restoredData, err := os.ReadFile(originalMetadataPath)
	if err != nil {
		t.Fatalf("Failed to read restored metadata: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(restoredData, &metadata); err != nil {
		t.Fatalf("Failed to parse restored metadata: %v", err)
	}

	if metadata["customTitle"] != "Test Session" {
		t.Errorf("Expected restored custom title 'Test Session', got '%v'", metadata["customTitle"])
	}
}