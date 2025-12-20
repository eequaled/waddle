package synthesis

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"waddle/pkg/ai"
	"waddle/pkg/storage"
)

// Worker processes sessions sequentially for AI synthesis
type Worker struct {
	storage    *storage.StorageEngine
	ollama     *ai.OllamaClient
	extractor  *Extractor
	processing atomic.Bool
	pending    atomic.Int64
	quit       chan struct{}
	wg         sync.WaitGroup
}

// NewWorker creates a new synthesis worker
func NewWorker(storageEngine *storage.StorageEngine) *Worker {
	return &Worker{
		storage:   storageEngine,
		ollama:    ai.NewOllamaClient("", "gemma2:2b"),
		extractor: NewExtractor(),
		quit:      make(chan struct{}),
	}
}

// Start begins background processing
func (w *Worker) Start() error {
	w.wg.Add(1)
	go w.processLoop()
	return nil
}

// Close stops the worker gracefully
func (w *Worker) Close() error {
	close(w.quit)
	w.wg.Wait()
	return nil
}

// PendingCount returns number of sessions awaiting synthesis
func (w *Worker) PendingCount() int64 {
	return w.pending.Load()
}

// processLoop runs the background processing
func (w *Worker) processLoop() {
	defer w.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second) // Process every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-w.quit:
			return
		case <-ticker.C:
			if err := w.ProcessNext(); err != nil {
				log.Printf("Synthesis processing error: %v", err)
			}
		}
	}
}

// ProcessNext processes the oldest pending session (FIFO)
func (w *Worker) ProcessNext() error {
	if !w.processing.CompareAndSwap(false, true) {
		return nil // Already processing
	}
	defer w.processing.Store(false)

	// Get oldest pending session
	sessions, _, err := w.storage.ListSessions(1, 100)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	var pendingSession *storage.Session
	for _, session := range sessions {
		// Check if synthesis is pending (assuming we add this field to Session)
		if session.CustomSummary == "" && session.ExtractedText != "" {
			pendingSession = &session
			break
		}
	}

	if pendingSession == nil {
		w.pending.Store(0)
		return nil // No pending sessions
	}

	// Generate 3-bullet summary
	summary, err := w.generate3BulletSummary(pendingSession.ExtractedText)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Extract entities
	entities := w.extractor.Extract(pendingSession.ExtractedText)
	_ = entities // TODO: Store entities in session when schema is updated

	// Update session with synthesis results
	pendingSession.CustomSummary = summary
	// Note: We'd need to add entities field to Session struct
	
	if err := w.storage.UpdateSession(pendingSession); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	log.Printf("Synthesized session: %s", pendingSession.Date)
	return nil
}

// generate3BulletSummary creates exactly 3 bullet points
func (w *Worker) generate3BulletSummary(text string) (string, error) {
	prompt := fmt.Sprintf(`Analyze this activity session and create exactly 3 bullet points summarizing what was accomplished:

%s

Format your response as exactly 3 bullet points starting with "•". Be concise and factual.`, text)

	summary, err := w.ollama.Summarize("Session", prompt)
	if err != nil {
		return "", err
	}

	return summary, nil
}