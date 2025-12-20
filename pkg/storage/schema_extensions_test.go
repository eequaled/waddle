package storage

import (
	"os"
	"testing"
	"time"
)

// TestSchemaExtensions tests the new synthesis and capture columns.
func TestSchemaExtensions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema_extensions_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := DefaultStorageConfig(tempDir)
	encryptionMgr := NewEncryptionManager()
	err = encryptionMgr.InitializeKey()
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}
	
	sm := NewSessionManager(config.DataDir, encryptionMgr)
	err = sm.Initialize()
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}
	defer sm.Close()
	
	// Test that knowledge_cards table exists
	t.Run("Table knowledge_cards exists", func(t *testing.T) {
		var tableName string
		err := sm.db.QueryRow(`
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name='knowledge_cards'
		`).Scan(&tableName)
		
		if err != nil {
			t.Fatalf("knowledge_cards table does not exist: %v", err)
		}
		
		if tableName != "knowledge_cards" {
			t.Errorf("Expected table name 'knowledge_cards', got '%s'", tableName)
		}
	})
	
	// Test that sessions table has synthesis columns
	t.Run("Sessions table has synthesis columns", func(t *testing.T) {
		columns := []string{"entities_json", "synthesis_status", "ai_summary", "ai_bullets"}
		
		for _, column := range columns {
			var columnExists int
			err := sm.db.QueryRow(`
				SELECT COUNT(*) FROM pragma_table_info('sessions') 
				WHERE name = ?
			`, column).Scan(&columnExists)
			
			if err != nil {
				t.Fatalf("Failed to check column %s: %v", column, err)
			}
			
			if columnExists == 0 {
				t.Errorf("Column %s does not exist in sessions table", column)
			}
		}
	})
	
	// Test that activity_blocks table has capture columns
	t.Run("Activity blocks table has capture columns", func(t *testing.T) {
		columns := []string{"capture_source", "structured_metadata"}
		
		for _, column := range columns {
			var columnExists int
			err := sm.db.QueryRow(`
				SELECT COUNT(*) FROM pragma_table_info('activity_blocks') 
				WHERE name = ?
			`, column).Scan(&columnExists)
			
			if err != nil {
				t.Fatalf("Failed to check column %s: %v", column, err)
			}
			
			if columnExists == 0 {
				t.Errorf("Column %s does not exist in activity_blocks table", column)
			}
		}
	})
	
	// Test knowledge_cards table structure
	t.Run("Knowledge cards table has correct structure", func(t *testing.T) {
		expectedColumns := []string{
			"id", "session_id", "title", "bullets", "entities", 
			"status", "created_at", "updated_at",
		}
		
		for _, column := range expectedColumns {
			var columnExists int
			err := sm.db.QueryRow(`
				SELECT COUNT(*) FROM pragma_table_info('knowledge_cards') 
				WHERE name = ?
			`, column).Scan(&columnExists)
			
			if err != nil {
				t.Fatalf("Failed to check column %s: %v", column, err)
			}
			
			if columnExists == 0 {
				t.Errorf("Column %s does not exist in knowledge_cards table", column)
			}
		}
	})
	
	// Test knowledge_cards indexes exist
	t.Run("Knowledge cards indexes exist", func(t *testing.T) {
		indexes := []string{
			"idx_knowledge_cards_session",
			"idx_knowledge_cards_status",
		}
		
		for _, index := range indexes {
			var indexExists int
			err := sm.db.QueryRow(`
				SELECT COUNT(*) FROM sqlite_master 
				WHERE type='index' AND name = ?
			`, index).Scan(&indexExists)
			
			if err != nil {
				t.Fatalf("Failed to check index %s: %v", index, err)
			}
			
			if indexExists == 0 {
				t.Errorf("Index %s does not exist", index)
			}
		}
	})
}

// TestSynthesisColumnsDefaults tests that synthesis columns have correct defaults.
func TestSynthesisColumnsDefaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synthesis_defaults_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := DefaultStorageConfig(tempDir)
	encryptionMgr := NewEncryptionManager()
	err = encryptionMgr.InitializeKey()
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}
	
	sm := NewSessionManager(config.DataDir, encryptionMgr)
	err = sm.Initialize()
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}
	defer sm.Close()
	
	// Create a test session
	session := Session{
		Date:         "2025-12-20",
		CustomTitle:  "Test Session",
		CustomSummary: "Test summary",
	}
	
	err = sm.Create(&session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	
	// Retrieve the session and check defaults
	retrieved, err := sm.GetByID(session.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}
	
	// Check synthesis column defaults
	if retrieved.EntitiesJSON != "[]" {
		t.Errorf("Expected entities_json default '[]', got '%s'", retrieved.EntitiesJSON)
	}
	
	if retrieved.SynthesisStatus != "pending" {
		t.Errorf("Expected synthesis_status default 'pending', got '%s'", retrieved.SynthesisStatus)
	}
	
	if retrieved.AISummary != "" {
		t.Errorf("Expected ai_summary default '', got '%s'", retrieved.AISummary)
	}
	
	if retrieved.AIBullets != "[]" {
		t.Errorf("Expected ai_bullets default '[]', got '%s'", retrieved.AIBullets)
	}
}

// TestCaptureColumnsDefaults tests that capture columns have correct defaults.
func TestCaptureColumnsDefaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "capture_defaults_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := DefaultStorageConfig(tempDir)
	encryptionMgr := NewEncryptionManager()
	err = encryptionMgr.InitializeKey()
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}
	
	sm := NewSessionManager(config.DataDir, encryptionMgr)
	err = sm.Initialize()
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}
	defer sm.Close()
	
	// Create a test session first
	session := Session{
		Date:         "2025-12-20",
		CustomTitle:  "Test Session",
	}
	
	err = sm.Create(&session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	
	// Create an activity block
	startTime, _ := time.Parse(time.RFC3339, "2025-12-20T10:00:00Z")
	endTime, _ := time.Parse(time.RFC3339, "2025-12-20T10:01:00Z")
	
	block := ActivityBlock{
		BlockID:       "test-block-1",
		StartTime:     startTime,
		EndTime:       endTime,
		MicroSummary:  "Test block",
	}
	
	err = sm.AddBlock(session.ID, "test-app", &block)
	if err != nil {
		t.Fatalf("Failed to add activity block: %v", err)
	}
	
	// Retrieve the block and check defaults
	blocks, err := sm.GetBlocks(session.ID, "test-app")
	if err != nil {
		t.Fatalf("Failed to retrieve activity blocks: %v", err)
	}
	
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}
	
	retrieved := blocks[0]
	
	// Check capture column defaults
	if retrieved.CaptureSource != "polling_ocr" {
		t.Errorf("Expected capture_source default 'polling_ocr', got '%s'", retrieved.CaptureSource)
	}
	
	if retrieved.StructuredMetadata != "{}" {
		t.Errorf("Expected structured_metadata default '{}', got '%s'", retrieved.StructuredMetadata)
	}
}

// TestKnowledgeCardsCRUD tests basic CRUD operations on knowledge_cards table.
func TestKnowledgeCardsCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "knowledge_cards_crud_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := DefaultStorageConfig(tempDir)
	encryptionMgr := NewEncryptionManager()
	err = encryptionMgr.InitializeKey()
	if err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}
	
	sm := NewSessionManager(config.DataDir, encryptionMgr)
	err = sm.Initialize()
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}
	defer sm.Close()
	
	// Create a test session
	session := Session{
		Date:         "2025-12-20",
		CustomTitle:  "Test Session",
	}
	
	err = sm.Create(&session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	
	// Test creating a knowledge card
	t.Run("Create knowledge card", func(t *testing.T) {
		_, err := sm.db.Exec(`
			INSERT INTO knowledge_cards (session_id, title, bullets, entities, status)
			VALUES (?, ?, ?, ?, ?)
		`, session.ID, "Test Card", `["Bullet 1", "Bullet 2", "Bullet 3"]`, `["#test", "@user"]`, "completed")
		
		if err != nil {
			t.Fatalf("Failed to create knowledge card: %v", err)
		}
	})
	
	// Test retrieving knowledge cards
	t.Run("Retrieve knowledge cards", func(t *testing.T) {
		rows, err := sm.db.Query(`
			SELECT id, session_id, title, bullets, entities, status
			FROM knowledge_cards
			WHERE session_id = ?
		`, session.ID)
		
		if err != nil {
			t.Fatalf("Failed to query knowledge cards: %v", err)
		}
		defer rows.Close()
		
		var count int
		for rows.Next() {
			var id, sessionID int64
			var title, bullets, entities, status string
			
			err := rows.Scan(&id, &sessionID, &title, &bullets, &entities, &status)
			if err != nil {
				t.Fatalf("Failed to scan knowledge card: %v", err)
			}
			
			if sessionID != session.ID {
				t.Errorf("Expected session_id %d, got %d", session.ID, sessionID)
			}
			
			if title != "Test Card" {
				t.Errorf("Expected title 'Test Card', got '%s'", title)
			}
			
			if status != "completed" {
				t.Errorf("Expected status 'completed', got '%s'", status)
			}
			
			count++
		}
		
		if count != 1 {
			t.Errorf("Expected 1 knowledge card, got %d", count)
		}
	})
	
	// Test foreign key constraint
	t.Run("Foreign key constraint works", func(t *testing.T) {
		_, err := sm.db.Exec(`
			INSERT INTO knowledge_cards (session_id, title, bullets, entities)
			VALUES (?, ?, ?, ?)
		`, 99999, "Invalid Card", "[]", "[]") // Non-existent session_id
		
		if err == nil {
			t.Error("Expected foreign key constraint error, but insert succeeded")
		}
	})
}