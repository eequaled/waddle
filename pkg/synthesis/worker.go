package synthesis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"waddle/pkg/storage"
)

// StorageInterface defines the methods needed by the synthesis worker
type StorageInterface interface {
	GetPendingSessions() ([]storage.Session, error)
	GetPendingSessionsCount() (int, error)
	UpdateSessionSynthesis(sessionID int64, entitiesJSON, synthesisStatus, aiSummary, aiBullets string) error
}

// SynthesisWorker handles sequential processing of sessions for AI synthesis
type SynthesisWorker struct {
	storage    StorageInterface
	extractor  *EntityExtractor
	processing int32 // atomic flag for processing state
}

// NewSynthesisWorker creates a new synthesis worker
func NewSynthesisWorker(storage StorageInterface) *SynthesisWorker {
	return &SynthesisWorker{
		storage:   storage,
		extractor: NewEntityExtractor(),
	}
}

// Start begins the synthesis worker processing loop
func (w *SynthesisWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Check for pending sessions every 5 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processPendingSessions(ctx)
		}
	}
}

// processPendingSessions processes all pending sessions in FIFO order
func (w *SynthesisWorker) processPendingSessions(ctx context.Context) {
	// Use atomic flag to ensure only one processing loop runs at a time
	if !atomic.CompareAndSwapInt32(&w.processing, 0, 1) {
		return // Already processing
	}
	defer atomic.StoreInt32(&w.processing, 0)
	
	// Get pending sessions ordered by created_at (FIFO)
	sessions, err := w.storage.GetPendingSessions()
	if err != nil {
		log.Printf("Error getting pending sessions: %v", err)
		return
	}
	
	for _, session := range sessions {
		select {
		case <-ctx.Done():
			return
		default:
			w.processSession(session)
		}
	}
}

// processSession processes a single session for synthesis
func (w *SynthesisWorker) processSession(session storage.Session) {
	log.Printf("Processing session %d for synthesis", session.ID)
	
	// Extract entities from session content
	entities := w.extractor.ExtractEntities(session.ExtractedText)
	entitiesJSON, err := w.extractor.EntitiesToJSON(entities)
	if err != nil {
		log.Printf("Error converting entities to JSON for session %d: %v", session.ID, err)
		w.markSessionFailed(session.ID, fmt.Sprintf("Entity extraction failed: %v", err))
		return
	}
	
	// Generate AI summary (mock implementation - would use Ollama in production)
	aiSummary := w.generateAISummary(session.ExtractedText)
	
	// Generate 3-bullet summary
	bullets := w.generate3BulletSummary(session.ExtractedText)
	bulletsJSON, err := json.Marshal(bullets)
	if err != nil {
		log.Printf("Error converting bullets to JSON for session %d: %v", session.ID, err)
		w.markSessionFailed(session.ID, fmt.Sprintf("Bullet formatting failed: %v", err))
		return
	}
	
	// Update session with synthesis results
	err = w.storage.UpdateSessionSynthesis(session.ID, entitiesJSON, "completed", aiSummary, string(bulletsJSON))
	if err != nil {
		log.Printf("Error updating session synthesis for session %d: %v", session.ID, err)
		w.markSessionFailed(session.ID, fmt.Sprintf("Database update failed: %v", err))
		return
	}
	
	log.Printf("Successfully processed session %d", session.ID)
}

// generateAISummary generates an AI summary for the session content
func (w *SynthesisWorker) generateAISummary(content string) string {
	// Mock implementation - in production this would call Ollama
	if len(content) == 0 {
		return "No content available for summary"
	}
	
	// Simple extractive summary - take first 200 characters
	summary := strings.TrimSpace(content)
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}
	
	return fmt.Sprintf("AI Summary: %s", summary)
}

// generate3BulletSummary generates exactly 3 bullet points from the content
func (w *SynthesisWorker) generate3BulletSummary(content string) []string {
	// Mock implementation - in production this would use Ollama with specific prompt
	if len(content) == 0 {
		return []string{
			"No content available",
			"Session appears to be empty",
			"Unable to generate meaningful bullets",
		}
	}
	
	// Simple heuristic: split content into sentences and take up to 3
	sentences := strings.Split(content, ".")
	var bullets []string
	
	for i, sentence := range sentences {
		if i >= 3 {
			break
		}
		
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 0 {
			// Limit bullet length
			if len(sentence) > 100 {
				sentence = sentence[:100] + "..."
			}
			bullets = append(bullets, sentence)
		}
	}
	
	// Ensure we always have exactly 3 bullets
	for len(bullets) < 3 {
		bullets = append(bullets, "Additional context needed")
	}
	
	return bullets[:3] // Ensure exactly 3 bullets
}

// markSessionFailed marks a session as failed with error message
func (w *SynthesisWorker) markSessionFailed(sessionID int64, errorMsg string) {
	err := w.storage.UpdateSessionSynthesis(sessionID, "[]", "failed", errorMsg, "[]")
	if err != nil {
		log.Printf("Error marking session %d as failed: %v", sessionID, err)
	}
}

// PendingCount returns the number of sessions pending synthesis
func (w *SynthesisWorker) PendingCount() (int, error) {
	return w.storage.GetPendingSessionsCount()
}

// IsProcessing returns true if the worker is currently processing sessions
func (w *SynthesisWorker) IsProcessing() bool {
	return atomic.LoadInt32(&w.processing) == 1
}