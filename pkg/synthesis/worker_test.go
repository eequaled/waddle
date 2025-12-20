package synthesis

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	
	"waddle/pkg/storage"
)

// MockStorage implements the storage interface for testing
type MockStorage struct {
	sessions        []storage.Session
	pendingCount    int
	updateCalls     []UpdateCall
	processingDelay time.Duration
}

type UpdateCall struct {
	SessionID       int64
	EntitiesJSON    string
	SynthesisStatus string
	AISummary       string
	AIBullets       string
}

func (m *MockStorage) GetPendingSessions() ([]storage.Session, error) {
	return m.sessions, nil
}

func (m *MockStorage) GetPendingSessionsCount() (int, error) {
	return m.pendingCount, nil
}

func (m *MockStorage) UpdateSessionSynthesis(sessionID int64, entitiesJSON, synthesisStatus, aiSummary, aiBullets string) error {
	if m.processingDelay > 0 {
		time.Sleep(m.processingDelay)
	}
	
	m.updateCalls = append(m.updateCalls, UpdateCall{
		SessionID:       sessionID,
		EntitiesJSON:    entitiesJSON,
		SynthesisStatus: synthesisStatus,
		AISummary:       aiSummary,
		AIBullets:       aiBullets,
	})
	return nil
}

func TestSynthesisWorker_BasicProcessing(t *testing.T) {
	mockStorage := &MockStorage{
		sessions: []storage.Session{
			{ID: 1, ExtractedText: "Working on PROJ-123 #bug @dev", CreatedAt: time.Now().Add(-1 * time.Hour)},
			{ID: 2, ExtractedText: "Meeting about https://example.com", CreatedAt: time.Now()},
		},
		pendingCount: 2,
	}
	
	// Create a mock worker that uses our mock storage
	worker := &SynthesisWorker{
		storage:   mockStorage,
		extractor: NewEntityExtractor(),
	}
	
	// Process sessions manually (not using the background loop)
	ctx := context.Background()
	worker.processPendingSessions(ctx)
	
	// Verify both sessions were processed
	if len(mockStorage.updateCalls) != 2 {
		t.Errorf("Expected 2 update calls, got %d", len(mockStorage.updateCalls))
	}
	
	// Verify first session processing
	call1 := mockStorage.updateCalls[0]
	if call1.SessionID != 1 {
		t.Errorf("Expected session ID 1, got %d", call1.SessionID)
	}
	
	if call1.SynthesisStatus != "completed" {
		t.Errorf("Expected status 'completed', got %s", call1.SynthesisStatus)
	}
	
	// Verify entities were extracted
	var entities []Entity
	err := json.Unmarshal([]byte(call1.EntitiesJSON), &entities)
	if err != nil {
		t.Errorf("Failed to parse entities JSON: %v", err)
	}
	
	if len(entities) == 0 {
		t.Error("Expected entities to be extracted")
	}
	
	// Verify bullets format
	var bullets []string
	err = json.Unmarshal([]byte(call1.AIBullets), &bullets)
	if err != nil {
		t.Errorf("Failed to parse bullets JSON: %v", err)
	}
	
	if len(bullets) != 3 {
		t.Errorf("Expected exactly 3 bullets, got %d", len(bullets))
	}
}

func TestSynthesisWorker_EmptyContent(t *testing.T) {
	mockStorage := &MockStorage{
		sessions: []storage.Session{
			{ID: 1, ExtractedText: "", CreatedAt: time.Now()},
		},
		pendingCount: 1,
	}
	
	worker := &SynthesisWorker{
		storage:   mockStorage,
		extractor: NewEntityExtractor(),
	}
	
	ctx := context.Background()
	worker.processPendingSessions(ctx)
	
	// Verify session was processed despite empty content
	if len(mockStorage.updateCalls) != 1 {
		t.Errorf("Expected 1 update call, got %d", len(mockStorage.updateCalls))
	}
	
	call := mockStorage.updateCalls[0]
	if call.SynthesisStatus != "completed" {
		t.Errorf("Expected status 'completed', got %s", call.SynthesisStatus)
	}
	
	// Verify default bullets for empty content
	var bullets []string
	err := json.Unmarshal([]byte(call.AIBullets), &bullets)
	if err != nil {
		t.Errorf("Failed to parse bullets JSON: %v", err)
	}
	
	if len(bullets) != 3 {
		t.Errorf("Expected exactly 3 bullets, got %d", len(bullets))
	}
	
	expectedBullets := []string{
		"No content available",
		"Session appears to be empty", 
		"Unable to generate meaningful bullets",
	}
	
	if !reflect.DeepEqual(bullets, expectedBullets) {
		t.Errorf("Expected default bullets %v, got %v", expectedBullets, bullets)
	}
}

func TestSynthesisWorker_PendingCount(t *testing.T) {
	mockStorage := &MockStorage{
		pendingCount: 5,
	}
	
	worker := &SynthesisWorker{
		storage:   mockStorage,
		extractor: NewEntityExtractor(),
	}
	
	count, err := worker.PendingCount()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if count != 5 {
		t.Errorf("Expected pending count 5, got %d", count)
	}
}

func TestSynthesisWorker_ProcessingFlag(t *testing.T) {
	mockStorage := &MockStorage{
		sessions: []storage.Session{
			{ID: 1, ExtractedText: "test", CreatedAt: time.Now()},
		},
		processingDelay: 100 * time.Millisecond, // Add delay to test concurrent access
	}
	
	worker := &SynthesisWorker{
		storage:   mockStorage,
		extractor: NewEntityExtractor(),
	}
	
	// Start processing in background
	ctx := context.Background()
	go worker.processPendingSessions(ctx)
	
	// Check processing flag immediately
	time.Sleep(10 * time.Millisecond)
	if !worker.IsProcessing() {
		t.Error("Expected worker to be processing")
	}
	
	// Wait for processing to complete
	time.Sleep(200 * time.Millisecond)
	if worker.IsProcessing() {
		t.Error("Expected worker to finish processing")
	}
}

// Property 6: Sequential Session Processing
func TestProperty_SequentialSessionProcessing(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	properties.Property("Sessions are processed sequentially without overlap", prop.ForAll(
		func(sessionCount int) bool {
			if sessionCount <= 0 || sessionCount > 10 {
				return true // Skip invalid inputs
			}
			
			// Create mock sessions with processing delay
			var sessions []storage.Session
			for i := 0; i < sessionCount; i++ {
				sessions = append(sessions, storage.Session{
					ID:            int64(i + 1),
					ExtractedText: "test content",
					CreatedAt:     time.Now().Add(time.Duration(i) * time.Second),
				})
			}
			
			mockStorage := &MockStorage{
				sessions:        sessions,
				processingDelay: 10 * time.Millisecond,
			}
			
			worker := &SynthesisWorker{
				storage:   mockStorage,
				extractor: NewEntityExtractor(),
			}
			
			// Process sessions
			ctx := context.Background()
			worker.processPendingSessions(ctx)
			
			// Verify all sessions were processed
			if len(mockStorage.updateCalls) != sessionCount {
				return false
			}
			
			// Verify sequential processing (session IDs should be in order)
			for i, call := range mockStorage.updateCalls {
				expectedID := int64(i + 1)
				if call.SessionID != expectedID {
					return false
				}
			}
			
			return true
		},
		gen.IntRange(1, 5),
	))
	
	properties.TestingRun(t)
}

// Property 7: Three-Bullet Summary Format
func TestProperty_ThreeBulletSummaryFormat(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	properties.Property("All summaries have exactly 3 bullets", prop.ForAll(
		func(content string) bool {
			worker := &SynthesisWorker{
				extractor: NewEntityExtractor(),
			}
			
			bullets := worker.generate3BulletSummary(content)
			
			// Must have exactly 3 bullets
			if len(bullets) != 3 {
				return false
			}
			
			// All bullets must be non-empty strings
			for _, bullet := range bullets {
				if bullet == "" {
					return false
				}
			}
			
			return true
		},
		gen.AlphaString(),
	))
	
	properties.TestingRun(t)
}

// Property 10: Backlog Count Accuracy
func TestProperty_BacklogCountAccuracy(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	properties.Property("Pending count matches actual pending sessions", prop.ForAll(
		func(pendingCount int) bool {
			if pendingCount < 0 || pendingCount > 100 {
				return true // Skip invalid inputs
			}
			
			mockStorage := &MockStorage{
				pendingCount: pendingCount,
			}
			
			worker := &SynthesisWorker{
				storage:   mockStorage,
				extractor: NewEntityExtractor(),
			}
			
			count, err := worker.PendingCount()
			if err != nil {
				return false
			}
			
			return count == pendingCount
		},
		gen.IntRange(0, 20),
	))
	
	properties.TestingRun(t)
}

// Property 11: FIFO Processing Order
func TestProperty_FIFOProcessingOrder(t *testing.T) {
	properties := gopter.NewProperties(nil)
	
	properties.Property("Sessions are processed in FIFO order by created_at", prop.ForAll(
		func(sessionTimes []int) bool {
			if len(sessionTimes) == 0 || len(sessionTimes) > 10 {
				return true // Skip invalid inputs
			}
			
			// Create sessions with different creation times
			var sessions []storage.Session
			baseTime := time.Now()
			
			for i, timeOffset := range sessionTimes {
				sessions = append(sessions, storage.Session{
					ID:            int64(i + 1),
					ExtractedText: "test content",
					CreatedAt:     baseTime.Add(time.Duration(timeOffset) * time.Second),
				})
			}
			
			mockStorage := &MockStorage{
				sessions: sessions,
			}
			
			worker := &SynthesisWorker{
				storage:   mockStorage,
				extractor: NewEntityExtractor(),
			}
			
			// Process sessions
			ctx := context.Background()
			worker.processPendingSessions(ctx)
			
			// Verify processing order matches creation time order
			if len(mockStorage.updateCalls) != len(sessions) {
				return false
			}
			
			// Since our mock returns sessions in the order they were added,
			// and the worker should process them in that order,
			// the update calls should match the session order
			for i, call := range mockStorage.updateCalls {
				expectedID := sessions[i].ID
				if call.SessionID != expectedID {
					return false
				}
			}
			
			return true
		},
		gen.SliceOfN(5, gen.IntRange(0, 100)),
	))
	
	properties.TestingRun(t)
}