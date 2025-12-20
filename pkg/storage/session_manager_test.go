package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSessionManagerInitialization tests database initialization and schema creation.
func TestSessionManagerInitialization(t *testing.T) {
	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize encryption manager
	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Create session manager
	sm := NewSessionManager(tempDir, em)

	t.Run("Initialize creates database file", func(t *testing.T) {
		if err := sm.Initialize(); err != nil {
			t.Fatalf("Failed to initialize session manager: %v", err)
		}
		defer sm.Close()

		// Check database file exists
		dbPath := filepath.Join(tempDir, "waddle.db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("Database file was not created")
		}
	})
}

// TestSchemaCreation tests that all tables are created correctly.
func TestSchemaCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	expectedTables := []string{
		"schema_version",
		"sessions",
		"app_activities",
		"activity_blocks",
		"chats",
		"notifications",
		"manual_notes",
		"sessions_fts",
		"activity_blocks_fts",
	}

	for _, tableName := range expectedTables {
		t.Run("Table "+tableName+" exists", func(t *testing.T) {
			var name string
			err := sm.db.QueryRow(`
				SELECT name FROM sqlite_master 
				WHERE type IN ('table', 'virtual table') AND name = ?
			`, tableName).Scan(&name)

			if err != nil {
				t.Errorf("Table %s does not exist: %v", tableName, err)
			}
		})
	}
}

// TestSchemaVersion tests schema version tracking.
func TestSchemaVersion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	version, err := sm.GetSchemaVersion()
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}

	// Should be at version 2 (initial schema + FTS5)
	if version != 2 {
		t.Errorf("Expected schema version 2, got %d", version)
	}
}

// TestIntegrityCheck tests the integrity check functionality.
func TestIntegrityCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	// Integrity check should pass on a fresh database
	if err := sm.RunIntegrityCheck(); err != nil {
		t.Errorf("Integrity check failed on fresh database: %v", err)
	}
}

// TestForeignKeyConstraints tests that foreign key constraints are enforced.
func TestForeignKeyConstraints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	t.Run("Insert with invalid foreign key fails", func(t *testing.T) {
		// Try to insert app_activity with non-existent session_id
		_, err := sm.db.Exec(`
			INSERT INTO app_activities (session_id, app_name)
			VALUES (99999, 'TestApp')
		`)

		if err == nil {
			t.Error("Expected foreign key constraint violation, but insert succeeded")
		}
	})

	t.Run("Cascade delete works", func(t *testing.T) {
		// Insert a session
		result, err := sm.db.Exec(`
			INSERT INTO sessions (date, custom_title)
			VALUES ('2025-01-15', 'Test Session')
		`)
		if err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}

		sessionID, _ := result.LastInsertId()

		// Insert an app_activity
		_, err = sm.db.Exec(`
			INSERT INTO app_activities (session_id, app_name)
			VALUES (?, 'TestApp')
		`, sessionID)
		if err != nil {
			t.Fatalf("Failed to insert app_activity: %v", err)
		}

		// Delete the session
		_, err = sm.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		// Verify app_activity was cascade deleted
		var count int
		err = sm.db.QueryRow("SELECT COUNT(*) FROM app_activities WHERE session_id = ?", sessionID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count app_activities: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 app_activities after cascade delete, got %d", count)
		}
	})
}

// TestWALMode tests that WAL mode is enabled.
func TestWALMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	var journalMode string
	err = sm.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to get journal mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("Expected WAL journal mode, got %s", journalMode)
	}
}

// TestIndexesExist tests that all expected indexes are created.
func TestIndexesExist(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	expectedIndexes := []string{
		"idx_sessions_date",
		"idx_sessions_created_at",
		"idx_app_activities_session",
		"idx_app_activities_app_name",
		"idx_activity_blocks_app_activity",
		"idx_activity_blocks_start_time",
		"idx_chats_session",
		"idx_chats_timestamp",
		"idx_notifications_timestamp",
		"idx_notifications_read",
		"idx_manual_notes_session",
	}

	for _, indexName := range expectedIndexes {
		t.Run("Index "+indexName+" exists", func(t *testing.T) {
			var name string
			err := sm.db.QueryRow(`
				SELECT name FROM sqlite_master 
				WHERE type = 'index' AND name = ?
			`, indexName).Scan(&name)

			if err != nil {
				t.Errorf("Index %s does not exist: %v", indexName, err)
			}
		})
	}
}


// TestFTS5Triggers tests that FTS5 triggers sync data correctly.
func TestFTS5Triggers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	t.Run("Insert trigger syncs to FTS", func(t *testing.T) {
		// Insert a session
		_, err := sm.db.Exec(`
			INSERT INTO sessions (date, custom_title, custom_summary, original_summary)
			VALUES ('2025-01-20', 'Test Title', 'Test Summary', 'Original Summary')
		`)
		if err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}

		// Search in FTS table
		var count int
		err = sm.db.QueryRow(`
			SELECT COUNT(*) FROM sessions_fts WHERE sessions_fts MATCH 'Test'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to search FTS: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 FTS result, got %d", count)
		}
	})

	t.Run("Update trigger syncs to FTS", func(t *testing.T) {
		// Update the session
		_, err := sm.db.Exec(`
			UPDATE sessions SET custom_title = 'Updated Title' WHERE date = '2025-01-20'
		`)
		if err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		// Search for updated content
		var count int
		err = sm.db.QueryRow(`
			SELECT COUNT(*) FROM sessions_fts WHERE sessions_fts MATCH 'Updated'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to search FTS: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 FTS result for 'Updated', got %d", count)
		}

		// Old content should not be found
		err = sm.db.QueryRow(`
			SELECT COUNT(*) FROM sessions_fts WHERE sessions_fts MATCH '"Test Title"'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to search FTS: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 FTS results for old title, got %d", count)
		}
	})

	t.Run("Delete trigger removes from FTS", func(t *testing.T) {
		// Delete the session
		_, err := sm.db.Exec(`DELETE FROM sessions WHERE date = '2025-01-20'`)
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		// Search should return no results
		var count int
		err = sm.db.QueryRow(`
			SELECT COUNT(*) FROM sessions_fts WHERE sessions_fts MATCH 'Updated'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to search FTS: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 FTS results after delete, got %d", count)
		}
	})

	t.Run("Activity blocks FTS trigger works", func(t *testing.T) {
		// Insert a session first
		result, err := sm.db.Exec(`
			INSERT INTO sessions (date, custom_title)
			VALUES ('2025-01-21', 'Session for blocks')
		`)
		if err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}
		sessionID, _ := result.LastInsertId()

		// Insert an app_activity
		result, err = sm.db.Exec(`
			INSERT INTO app_activities (session_id, app_name)
			VALUES (?, 'TestApp')
		`, sessionID)
		if err != nil {
			t.Fatalf("Failed to insert app_activity: %v", err)
		}
		appActivityID, _ := result.LastInsertId()

		// Insert an activity block
		_, err = sm.db.Exec(`
			INSERT INTO activity_blocks (app_activity_id, block_id, start_time, end_time, micro_summary)
			VALUES (?, '15-30', '2025-01-21 15:30:00', '2025-01-21 15:45:00', 'Working on important project')
		`, appActivityID)
		if err != nil {
			t.Fatalf("Failed to insert activity_block: %v", err)
		}

		// Search in FTS table
		var count int
		err = sm.db.QueryRow(`
			SELECT COUNT(*) FROM activity_blocks_fts WHERE activity_blocks_fts MATCH 'important'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to search activity_blocks FTS: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 FTS result for activity_blocks, got %d", count)
		}
	})
}

// TestTriggersExist tests that all FTS5 sync triggers are created.
func TestTriggersExist(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sm.Close()

	expectedTriggers := []string{
		"sessions_ai",
		"sessions_ad",
		"sessions_au",
		"activity_blocks_ai",
		"activity_blocks_ad",
		"activity_blocks_au",
	}

	for _, triggerName := range expectedTriggers {
		t.Run("Trigger "+triggerName+" exists", func(t *testing.T) {
			var name string
			err := sm.db.QueryRow(`
				SELECT name FROM sqlite_master 
				WHERE type = 'trigger' AND name = ?
			`, triggerName).Scan(&name)

			if err != nil {
				t.Errorf("Trigger %s does not exist: %v", triggerName, err)
			}
		})
	}
}
