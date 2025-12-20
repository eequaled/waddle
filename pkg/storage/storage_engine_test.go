package storage

import (
	"os"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestStorageEngineInitialization tests basic StorageEngine initialization.
func TestStorageEngineInitialization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_engine_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultStorageConfig(tempDir)
	se := NewStorageEngine(config)

	t.Run("Initialize creates all components", func(t *testing.T) {
		if err := se.Initialize(); err != nil {
			t.Fatalf("Failed to initialize storage engine: %v", err)
		}
		defer se.Close()

		if se.sessionMgr == nil {
			t.Error("SessionManager not initialized")
		}
		if se.vectorMgr == nil {
			t.Error("VectorManager not initialized")
		}
		if se.fileMgr == nil {
			t.Error("FileManager not initialized")
		}
		if se.encryptionMgr == nil {
			t.Error("EncryptionManager not initialized")
		}
	})
}

// TestStorageEngineBasicOperations tests basic CRUD operations.
func TestStorageEngineBasicOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_engine_crud_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultStorageConfig(tempDir)
	se := NewStorageEngine(config)
	if err := se.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer se.Close()

	t.Run("Create and get session", func(t *testing.T) {
		session, err := se.CreateSession("2025-01-20")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		retrieved, err := se.GetSession("2025-01-20")
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		if retrieved.Date != session.Date {
			t.Errorf("Expected date %s, got %s", session.Date, retrieved.Date)
		}
	})

	t.Run("Update session", func(t *testing.T) {
		session, err := se.CreateSession("2025-01-21")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		session.CustomTitle = "Updated Title"
		if err := se.UpdateSession(session); err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		retrieved, err := se.GetSession("2025-01-21")
		if err != nil {
			t.Fatalf("Failed to get updated session: %v", err)
		}

		if retrieved.CustomTitle != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got %s", retrieved.CustomTitle)
		}
	})

	t.Run("List sessions", func(t *testing.T) {
		sessions, total, err := se.ListSessions(1, 10)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(sessions) == 0 {
			t.Error("Expected some sessions")
		}
		if total == 0 {
			t.Error("Expected total > 0")
		}
	})
}

// TestPropertyCascadeDeleteCompleteness is Property Test 10: Cascade Delete Completeness
// For any deleted session:
// - All SQLite records (session, app_activities, activity_blocks, chats) SHALL be removed
// - The LanceDB embedding for that session SHALL be removed
// - The filesystem directory for that session SHALL be removed
func TestPropertyCascadeDeleteCompleteness(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_engine_cascade_property_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultStorageConfig(tempDir)
	se := NewStorageEngine(config)
	if err := se.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer se.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for session data
	genSessionData := gen.Struct(reflect.TypeOf(struct {
		Date           string
		CustomTitle    string
		CustomSummary  string
		AppName        string
		BlockID        string
		ChatContent    string
		FileName       string
	}{}), map[string]gopter.Gen{
		"Date":        GenDateString(),
		"CustomTitle": gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		"CustomSummary": gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		"AppName":     gen.OneConstOf("TestApp", "AnotherApp", "ThirdApp"),
		"BlockID":     GenBlockID(),
		"ChatContent": gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		"FileName":    gen.OneConstOf("screenshot1.png", "screenshot2.png", "latest.png"),
	})

	properties.Property("Cascade delete removes all related data", prop.ForAll(
		func(data struct {
			Date           string
			CustomTitle    string
			CustomSummary  string
			AppName        string
			BlockID        string
			ChatContent    string
			FileName       string
		}) bool {
			// Create session with all types of related data
			session, err := se.CreateSession(data.Date)
			if err != nil {
				t.Logf("Failed to create session: %v", err)
				return false
			}

			// Update session with content
			session.CustomTitle = data.CustomTitle
			session.CustomSummary = data.CustomSummary
			if err := se.UpdateSession(session); err != nil {
				t.Logf("Failed to update session: %v", err)
				return false
			}

			// Add activity block
			block := &ActivityBlock{
				BlockID:      data.BlockID,
				OCRText:      "Test OCR content",
				MicroSummary: "Test summary",
			}
			if err := se.AddActivityBlock(data.Date, data.AppName, block); err != nil {
				t.Logf("Failed to add activity block: %v", err)
				return false
			}

			// Add chat message
			chat := &ChatMessage{
				Role:    "user",
				Content: data.ChatContent,
			}
			if err := se.AddChat(data.Date, chat); err != nil {
				t.Logf("Failed to add chat: %v", err)
				return false
			}

			// Save file
			fileData := []byte("test file content")
			_, err = se.SaveScreenshot(data.Date, data.AppName, data.FileName, fileData)
			if err != nil {
				t.Logf("Failed to save screenshot: %v", err)
				return false
			}

			// Store embedding (simulate text content generating embedding)
			embedding := createNormalizedEmbedding(EmbeddingDimensions)
			if err := se.vectorMgr.StoreEmbedding(session.ID, embedding); err != nil {
				t.Logf("Failed to store embedding: %v", err)
				return false
			}

			// Verify data exists before deletion
			if !se.vectorMgr.HasEmbedding(session.ID) {
				t.Logf("Embedding should exist before deletion")
				return false
			}

			// Check if file exists
			filePath := se.GetScreenshotPath(data.Date, data.AppName, data.FileName)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Logf("File should exist before deletion")
				return false
			}

			// Delete the session (cascade delete)
			if err := se.DeleteSession(data.Date); err != nil {
				t.Logf("Failed to delete session: %v", err)
				return false
			}

			// Property 1: Session should not exist in SQLite
			_, err = se.GetSession(data.Date)
			if !IsNotFound(err) {
				t.Logf("Session should not exist after deletion")
				return false
			}

			// Property 2: Embedding should not exist in LanceDB
			if se.vectorMgr.HasEmbedding(session.ID) {
				t.Logf("Embedding should not exist after deletion")
				return false
			}

			// Property 3: Files should not exist
			if _, err := os.Stat(filePath); !os.IsNotExist(err) {
				t.Logf("File should not exist after deletion")
				return false
			}

			// Property 4: Activity blocks should not exist (tested via foreign key cascade)
			blocks, err := se.GetActivityBlocks(data.Date, data.AppName)
			if err == nil && len(blocks) > 0 {
				t.Logf("Activity blocks should not exist after session deletion")
				return false
			}

			// Property 5: Chats should not exist (tested via foreign key cascade)
			chats, err := se.GetChats(data.Date)
			if err == nil && len(chats) > 0 {
				t.Logf("Chats should not exist after session deletion")
				return false
			}

			return true
		},
		genSessionData,
	))

	properties.TestingRun(t)
}

// TestStorageEngineHealthCheck tests the health check functionality.
func TestStorageEngineHealthCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_engine_health_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultStorageConfig(tempDir)
	se := NewStorageEngine(config)
	if err := se.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer se.Close()

	t.Run("Health check returns status", func(t *testing.T) {
		health, err := se.HealthCheck()
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		if health.Status == "" {
			t.Error("Health status should not be empty")
		}

		expectedChecks := []string{"database", "vector_db", "filesystem"}
		for _, checkName := range expectedChecks {
			if _, exists := health.Checks[checkName]; !exists {
				t.Errorf("Missing health check: %s", checkName)
			}
		}
	})
}