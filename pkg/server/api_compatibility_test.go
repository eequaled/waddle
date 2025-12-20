package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ideathon/pkg/storage"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestAPIResponseCompatibility tests that API responses maintain identical structure
// Property 15: API Response Compatibility
func TestAPIResponseCompatibility(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, ".waddle")
	
	// Initialize storage engine
	config := storage.DefaultStorageConfig(storageDir)
	storageEngine := storage.NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create server
	isPaused := &atomic.Bool{}
	server := NewServer(tempDir, "8080", isPaused, storageEngine)

	properties := gopter.NewProperties(nil)

	// Property: Session list endpoint returns array of date strings
	properties.Property("session list returns date array", prop.ForAll(
		func() bool {
			// Create a test session
			testDate := "2024-01-15"
			storageEngine.DeleteSession(testDate) // Clean up first
			
			if _, err := storageEngine.CreateSession(testDate); err != nil {
				return false
			}

			// Test GET /api/sessions
			req := httptest.NewRequest("GET", "/api/sessions", nil)
			w := httptest.NewRecorder()
			server.handleSessions(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var response []string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Response should be an array of strings containing our test date
			found := false
			for _, date := range response {
				if date == testDate {
					found = true
					break
				}
			}

			return found
		},
	))

	// Property: Session metadata endpoint returns correct structure
	properties.Property("session metadata has correct structure", prop.ForAll(
		func() bool {
			testDate := fmt.Sprintf("2024-01-%02d", time.Now().Nanosecond()%28+1) // Generate unique date
			
			// Clean up first
			storageEngine.DeleteSession(testDate)
			
			// Create test session
			session, err := storageEngine.CreateSession(testDate)
			if err != nil {
				return false
			}

			// Update with test data
			session.CustomTitle = "Test Title"
			session.CustomSummary = "Test Summary"
			session.OriginalSummary = "Original Summary"
			if err := storageEngine.UpdateSession(session); err != nil {
				return false
			}

			// Test GET /api/sessions/{date}/metadata
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/sessions/%s/metadata", testDate), nil)
			w := httptest.NewRecorder()
			server.handleAppDetails(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var metadata SessionMetadata
			if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
				return false
			}

			// Verify structure matches expected API format
			return metadata.CustomTitle == "Test Title" &&
				metadata.CustomSummary == "Test Summary" &&
				metadata.OriginalSummary == "Original Summary" &&
				metadata.ManualNotes != nil // Should be initialized as empty slice
		},
	))

	// Property: Notification endpoint returns correct structure
	properties.Property("notifications have correct structure", prop.ForAll(
		func() bool {
			// Create test notification
			testNotif := &storage.Notification{
				ID:        fmt.Sprintf("test-%d", time.Now().UnixNano()),
				Type:      "test",
				Title:     "Test Title",
				Message:   "Test Message",
				Timestamp: time.Now(),
				Read:      false,
				Metadata:  `{"key":"value"}`,
			}

			if err := storageEngine.AddNotification(testNotif); err != nil {
				return false
			}

			// Test GET /api/notifications
			req := httptest.NewRequest("GET", "/api/notifications", nil)
			w := httptest.NewRecorder()
			server.handleNotifications(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var notifications []Notification
			if err := json.Unmarshal(w.Body.Bytes(), &notifications); err != nil {
				return false
			}

			// Find our test notification
			for _, notif := range notifications {
				if notif.ID == testNotif.ID {
					// Verify structure
					return notif.Type == "test" &&
						notif.Title == "Test Title" &&
						notif.Message == "Test Message" &&
						notif.Timestamp != "" &&
						!notif.Read &&
						notif.Metadata != nil &&
						notif.Metadata["key"] == "value"
				}
			}
			return false
		},
	))

	// Property: Health endpoint returns correct structure
	properties.Property("health endpoint has correct structure", prop.ForAll(
		func() bool {
			// Test GET /api/health
			req := httptest.NewRequest("GET", "/api/health", nil)
			w := httptest.NewRecorder()
			server.handleHealth(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var health storage.HealthStatus
			if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
				return false
			}

			// Verify structure
			return health.Status != "" &&
				health.Checks != nil &&
				!health.Timestamp.IsZero()
		},
	))

	// Property: Search endpoints return correct structure
	properties.Property("search endpoints have correct structure", prop.ForAll(
		func() bool {
			query := "test"

			// Test full-text search with valid query
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/search/fulltext?q=%s", query), nil)
			w := httptest.NewRecorder()
			server.handleFullTextSearch(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var ftsResults []storage.SearchResult
			if err := json.Unmarshal(w.Body.Bytes(), &ftsResults); err != nil {
				return false
			}

			// Test semantic search with valid query
			req = httptest.NewRequest("GET", fmt.Sprintf("/api/search/semantic?q=%s", query), nil)
			w = httptest.NewRecorder()
			server.handleSemanticSearch(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var semanticResults []storage.SearchResult
			if err := json.Unmarshal(w.Body.Bytes(), &semanticResults); err != nil {
				return false
			}

			// Both should return arrays (even if empty)
			return ftsResults != nil && semanticResults != nil
		},
	))

	// Property: Status endpoint maintains structure
	properties.Property("status endpoint has correct structure", prop.ForAll(
		func(paused bool) bool {
			// Test POST /api/status
			body := map[string]bool{"paused": paused}
			bodyBytes, _ := json.Marshal(body)
			req := httptest.NewRequest("POST", "/api/status", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.handleStatus(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var response map[string]bool
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Test GET /api/status
			req = httptest.NewRequest("GET", "/api/status", nil)
			w = httptest.NewRecorder()
			server.handleStatus(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var getResponse map[string]bool
			if err := json.Unmarshal(w.Body.Bytes(), &getResponse); err != nil {
				return false
			}

			// Both should have the same structure
			return response["paused"] == paused &&
				getResponse["paused"] == paused &&
				len(response) == 1 &&
				len(getResponse) == 1
		},
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}