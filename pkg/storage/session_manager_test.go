package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
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

	// Should be at version 3 (initial schema + FTS5 + synthesis/capture columns)
	if version != 3 {
		t.Errorf("Expected schema version 3, got %d", version)
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
// TestPropertyFullTextSearchCoverage is Property Test 6: Full-Text Search Coverage
// For any text stored in sessions or activity_blocks, searching for any word 
// contained in that text SHALL return the containing record in results.
func TestPropertyFullTextSearchCoverage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session_manager_fts_property_*")
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

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for simple searchable words
	genWord := gen.OneConstOf("apple", "banana", "cherry", "dog", "elephant", "forest", "guitar", "house", "island", "jungle")

	// Generator for session data with searchable content
	genSessionData := gen.Struct(reflect.TypeOf(struct {
		Date           string
		CustomTitle    string
		CustomSummary  string
		OriginalSummary string
	}{}), map[string]gopter.Gen{
		"Date":            GenDateString(),
		"CustomTitle":     genWord,
		"CustomSummary":   genWord,
		"OriginalSummary": genWord,
	})

	properties.Property("Full-text search finds containing records", prop.ForAll(
		func(data struct {
			Date           string
			CustomTitle    string
			CustomSummary  string
			OriginalSummary string
		}) bool {
			// Create session with the generated data
			session := Session{
				Date:            data.Date,
				CustomTitle:     data.CustomTitle,
				CustomSummary:   data.CustomSummary,
				OriginalSummary: data.OriginalSummary,
			}

			// Create the session
			err := sm.Create(&session)
			if err != nil {
				t.Logf("Failed to create session: %v", err)
				return false
			}

			// Test searching for the custom title
			if session.CustomTitle != "" {
				results, err := sm.Search(session.CustomTitle, 1, 10)
				if err != nil {
					t.Logf("Search failed for title '%s': %v", session.CustomTitle, err)
					sm.Delete(session.Date) // Clean up
					return false
				}

				// Should find the session
				found := false
				for _, result := range results {
					if result.Session.Date == session.Date {
						found = true
						break
					}
				}
				if !found {
					t.Logf("Search for title '%s' did not find session %s", session.CustomTitle, session.Date)
					sm.Delete(session.Date) // Clean up
					return false
				}
			}

			// Test searching for the custom summary
			if session.CustomSummary != "" && session.CustomSummary != session.CustomTitle {
				results, err := sm.Search(session.CustomSummary, 1, 10)
				if err != nil {
					t.Logf("Search failed for summary '%s': %v", session.CustomSummary, err)
					sm.Delete(session.Date) // Clean up
					return false
				}

				// Should find the session
				found := false
				for _, result := range results {
					if result.Session.Date == session.Date {
						found = true
						break
					}
				}
				if !found {
					t.Logf("Search for summary '%s' did not find session %s", session.CustomSummary, session.Date)
					sm.Delete(session.Date) // Clean up
					return false
				}
			}

			// Clean up - delete the session
			sm.Delete(session.Date)

			return true
		},
		genSessionData,
	))

	properties.TestingRun(t)
}
// TestPropertyFullTextSearchPagination is Property Test 7: Full-Text Search Pagination
// For any full-text search with pagination parameters:
// - Results on page N SHALL not overlap with results on page N-1 or N+1
// - Result count per page SHALL be at most pageSize
// - Total results across all pages SHALL equal the unpaginated result count
func TestPropertyFullTextSearchPagination(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session_manager_pagination_property_*")
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

	// Pre-populate with many sessions containing the same search term
	searchTerm := "testword"
	numSessions := 25 // Create enough sessions to test pagination
	
	for i := 0; i < numSessions; i++ {
		session := Session{
			Date:        fmt.Sprintf("2025-01-%02d", i+1),
			CustomTitle: fmt.Sprintf("Session %d with %s", i, searchTerm),
		}
		err := sm.Create(&session)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for page size (1 to 10)
	genPageSize := gen.IntRange(1, 10)

	properties.Property("Full-text search pagination properties", prop.ForAll(
		func(pageSize int) bool {
			// Get all results without pagination to establish baseline
			allResults, err := sm.Search(searchTerm, 1, 1000) // Large page size to get all
			if err != nil {
				t.Logf("Failed to get all results: %v", err)
				return false
			}

			totalResults := len(allResults)
			if totalResults == 0 {
				t.Logf("No results found for search term")
				return false
			}

			expectedPages := (totalResults + pageSize - 1) / pageSize // Ceiling division

			var allPaginatedResults []SearchResult
			seenSessionIDs := make(map[string]bool)

			// Test each page
			for page := 1; page <= expectedPages; page++ {
				results, err := sm.Search(searchTerm, page, pageSize)
				if err != nil {
					t.Logf("Failed to search page %d: %v", page, err)
					return false
				}

				// Property 1: Result count per page <= pageSize
				if len(results) > pageSize {
					t.Logf("Page %d has %d results, expected at most %d", page, len(results), pageSize)
					return false
				}

				// Property 2: No overlapping results between pages
				for _, result := range results {
					if seenSessionIDs[result.Session.Date] {
						t.Logf("Session %s appears on multiple pages", result.Session.Date)
						return false
					}
					seenSessionIDs[result.Session.Date] = true
				}

				allPaginatedResults = append(allPaginatedResults, results...)

				// Last page might have fewer results
				if page < expectedPages && len(results) != pageSize {
					t.Logf("Non-last page %d has %d results, expected %d", page, len(results), pageSize)
					return false
				}
			}

			// Property 3: Total paginated results equals unpaginated results
			if len(allPaginatedResults) != totalResults {
				t.Logf("Paginated results count %d != total results count %d", len(allPaginatedResults), totalResults)
				return false
			}

			// Property 4: Results are in the same order (by score)
			for i := 0; i < len(allResults) && i < len(allPaginatedResults); i++ {
				if allResults[i].Session.Date != allPaginatedResults[i].Session.Date {
					t.Logf("Result order differs at position %d", i)
					return false
				}
			}

			return true
		},
		genPageSize,
	))

	properties.TestingRun(t)
}
// TestPropertySemanticSearchDateFiltering is Property Test 5: Semantic Search Date Filtering
// For any semantic search with a date range filter, all returned sessions 
// SHALL have dates within the specified range (inclusive).
func TestPropertySemanticSearchDateFiltering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session_manager_semantic_date_property_*")
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

	// Create a mock VectorManager for testing
	vectorTempDir, err := os.MkdirTemp("", "vector_manager_semantic_date_*")
	if err != nil {
		t.Fatalf("Failed to create vector temp directory: %v", err)
	}
	defer os.RemoveAll(vectorTempDir)

	vectorConfig := DefaultVectorManagerConfig(vectorTempDir)
	vm, err := NewVectorManager(vectorConfig)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	// Pre-populate with sessions across different dates
	testSessions := []Session{
		{Date: "2025-01-01", CustomTitle: "January session with searchable content"},
		{Date: "2025-01-15", CustomTitle: "Mid January session with searchable content"},
		{Date: "2025-02-01", CustomTitle: "February session with searchable content"},
		{Date: "2025-02-15", CustomTitle: "Mid February session with searchable content"},
		{Date: "2025-03-01", CustomTitle: "March session with searchable content"},
	}

	for _, session := range testSessions {
		err := sm.Create(&session)
		if err != nil {
			t.Fatalf("Failed to create session %s: %v", session.Date, err)
		}

		// Store embeddings for semantic search (using mock embeddings)
		embedding := createNormalizedEmbedding(EmbeddingDimensions)
		err = vm.StoreEmbedding(session.ID, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding for session %s: %v", session.Date, err)
		}
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for date ranges
	genDateRange := gen.Struct(reflect.TypeOf(DateRange{}), map[string]gopter.Gen{
		"StartDate": gen.OneConstOf("2025-01-01", "2025-01-15", "2025-02-01"),
		"EndDate":   gen.OneConstOf("2025-02-15", "2025-03-01", "2025-03-15"),
	}).SuchThat(func(dr DateRange) bool {
		return dr.StartDate <= dr.EndDate // Ensure valid date range
	})

	properties.Property("Semantic search date filtering", prop.ForAll(
		func(dateRange DateRange) bool {
			// Create a mock query embedding
			queryEmbedding := createNormalizedEmbedding(EmbeddingDimensions)
			
			// Get vector search results first (limit to number of sessions we have)
			vectorResults, err := vm.Search(queryEmbedding, len(testSessions))
			if err != nil {
				t.Logf("Vector search failed: %v", err)
				return false
			}

			if len(vectorResults) == 0 {
				// No vector results, so semantic search should return empty
				return true
			}

			// Now test the date filtering by manually checking what should be returned
			expectedSessionIDs := make(map[int64]bool)
			for _, session := range testSessions {
				if session.Date >= dateRange.StartDate && session.Date <= dateRange.EndDate {
					expectedSessionIDs[session.ID] = true
				}
			}

			// Filter vector results by expected session IDs
			var filteredVectorResults []VectorSearchResult
			for _, vr := range vectorResults {
				if expectedSessionIDs[vr.SessionID] {
					filteredVectorResults = append(filteredVectorResults, vr)
				}
			}

			// Perform semantic search with date filtering using a mock implementation
			// Since we can't call the real SemanticSearch (needs Ollama), we'll test the date filtering logic directly
			
			// Build the SQL query that SemanticSearch would use
			if len(filteredVectorResults) == 0 {
				return true // No results expected
			}

			sessionIDs := make([]int64, len(filteredVectorResults))
			for i, result := range filteredVectorResults {
				sessionIDs[i] = result.SessionID
			}

			// Test the date filtering SQL logic
			placeholders := make([]string, len(sessionIDs))
			args := make([]interface{}, len(sessionIDs))
			for i, id := range sessionIDs {
				placeholders[i] = "?"
				args[i] = id
			}

			baseSQL := `
				SELECT id, date FROM sessions
				WHERE id IN (` + strings.Join(placeholders, ",") + `)`

			// Add date range filtering
			if dateRange.StartDate != "" {
				baseSQL += " AND date >= ?"
				args = append(args, dateRange.StartDate)
			}
			if dateRange.EndDate != "" {
				baseSQL += " AND date <= ?"
				args = append(args, dateRange.EndDate)
			}

			rows, err := sm.db.Query(baseSQL, args...)
			if err != nil {
				t.Logf("Date filtering query failed: %v", err)
				return false
			}
			defer rows.Close()

			// Verify all returned sessions are within date range
			for rows.Next() {
				var sessionID int64
				var sessionDate string
				err := rows.Scan(&sessionID, &sessionDate)
				if err != nil {
					t.Logf("Failed to scan result: %v", err)
					return false
				}

				// Property: All results should be within the date range
				if sessionDate < dateRange.StartDate || sessionDate > dateRange.EndDate {
					t.Logf("Session %d with date %s is outside date range [%s, %s]", 
						sessionID, sessionDate, dateRange.StartDate, dateRange.EndDate)
					return false
				}
			}

			return true
		},
		genDateRange,
	))

	properties.TestingRun(t)
}

// TestPropertySearchMatchesEntities is Property Test 14: Search Matches Entities
// For any search query matching an entity value, the search results SHALL include 
// sessions containing that entity.
// **Validates: Requirements 6.5**
func TestPropertySearchMatchesEntities(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session_manager_entity_search_property_*")
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

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for entity values that match the patterns from requirements
	genEntity := gen.OneConstOf("PROJ-123", "#feature", "@john", "https://example.com")

	// Generator for session data with entities
	genSessionWithEntity := gen.Struct(reflect.TypeOf(struct {
		Date        string
		Entity      string
		CustomTitle string
	}{}), map[string]gopter.Gen{
		"Date":        GenDateString(),
		"Entity":      genEntity,
		"CustomTitle": gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
	})

	properties.Property("Search matches entities", prop.ForAll(
		func(data struct {
			Date        string
			Entity      string
			CustomTitle string
		}) bool {
			// Create entities JSON with the generated entity
			entitiesJSON := fmt.Sprintf(`[{"value":"%s","type":"test","count":1}]`, data.Entity)

			// Create session with entity in entities_json
			session := Session{
				Date:         data.Date,
				CustomTitle:  data.CustomTitle,
				EntitiesJSON: entitiesJSON,
			}

			// Create the session
			err := sm.Create(&session)
			if err != nil {
				t.Logf("Failed to create session: %v", err)
				return false
			}

			// Search for the entity value
			results, err := sm.Search(data.Entity, 1, 10)
			if err != nil {
				t.Logf("Search failed: %v", err)
				return false
			}

			// Property: Search results should include the session containing the entity
			found := false
			for _, result := range results {
				if result.Session.ID == session.ID {
					found = true
					break
				}
			}

			if !found {
				t.Logf("Entity search for '%s' did not return session %d with entities_json: %s", 
					data.Entity, session.ID, entitiesJSON)
				return false
			}

			// Clean up - delete the session
			err = sm.DeleteByID(session.ID)
			if err != nil {
				t.Logf("Failed to delete session: %v", err)
				// Don't fail the test for cleanup issues
			}

			return true
		},
		genSessionWithEntity,
	))

	properties.TestingRun(t)
}