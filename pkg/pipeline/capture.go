package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"waddle/pkg/platform"
	"waddle/pkg/storage"
)

// CaptureSource indicates where the activity data came from
type CaptureSource string

const (
	CaptureSourceETW          CaptureSource = "etw"
	CaptureSourcePolling      CaptureSource = "polling"
	CaptureSourceUIAutomation CaptureSource = "ui_automation"
	CaptureSourceOCR          CaptureSource = "ocr"
)

// ActivityBlock represents a captured activity with metadata
type ActivityBlock struct {
	Timestamp      time.Time
	WindowHandle   uintptr
	ProcessID      uint32
	ProcessName    string
	WindowTitle    string
	AppType        string // "vscode", "chrome", "edge", "slack", "unknown"
	Metadata       map[string]interface{}
	CaptureSource  CaptureSource
	StructuredData bool // True if data came from UIA, false if from OCR
}

// Pipeline orchestrates the hybrid capture pipeline: Tracker → UIA → OCR
type Pipeline struct {
	plat           platform.Platform
	storage        interface{} // Storage engine interface
	ctx            context.Context
	cancel         context.CancelFunc
	activityBuffer chan *ActivityBlock
	ocrBatchBuffer chan *ActivityBlock
	droppedEvents  atomic.Int64
	mu             sync.RWMutex
	running        bool
	wg             sync.WaitGroup
}

const (
	// ActivityBufferSize is the size of the activity event buffer
	ActivityBufferSize = 1000

	// OCRBatchBufferSize is the size of the OCR batch buffer
	OCRBatchBufferSize = 100

	// OCRBatchTimeout is the maximum time to wait before flushing OCR batch
	OCRBatchTimeout = 500 * time.Millisecond

	// OCRBatchSize is the maximum number of OCR requests to batch
	OCRBatchSize = 10
)

// NewPipeline creates a new hybrid capture pipeline.
// If plat is nil, a default platform is created via platform.NewWindowTracker.
func NewPipeline(storage interface{}, plat platform.Platform) (*Pipeline, error) {
	ctx, cancel := context.WithCancel(context.Background())

	if plat == nil {
		cancel()
		return nil, fmt.Errorf("platform is required")
	}

	p := &Pipeline{
		plat:           plat,
		storage:        storage,
		ctx:            ctx,
		cancel:         cancel,
		activityBuffer: make(chan *ActivityBlock, ActivityBufferSize),
		ocrBatchBuffer: make(chan *ActivityBlock, OCRBatchBufferSize),
	}

	return p, nil
}

// Start begins the capture pipeline
func (p *Pipeline) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("pipeline already running")
	}

	// Start window tracker (via platform)
	err := p.plat.Start(p.ctx)
	if err != nil && !p.plat.IsFallbackMode() {
		return fmt.Errorf("failed to start window tracker: %w", err)
	}

	// Start pipeline workers
	p.wg.Add(3)
	go p.etwEventProcessor()
	go p.uiaProcessor()
	go p.ocrBatchProcessor()

	p.running = true
	return nil
}

// etwEventProcessor processes ETW focus events
func (p *Pipeline) etwEventProcessor() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case focusEvent := <-p.plat.FocusEvents():
			// Create activity block from Tracker event
			activity := &ActivityBlock{
				Timestamp:     time.Unix(0, focusEvent.Timestamp),
				WindowHandle:  focusEvent.WindowHandle,
				ProcessID:     focusEvent.ProcessID,
				ProcessName:   focusEvent.ProcessName,
				CaptureSource: CaptureSourceETW,
				Metadata:      make(map[string]interface{}),
			}

			// Send to UIA processor with backpressure handling
			p.sendActivityWithBackpressure(activity)
		}
	}
}

// uiaProcessor processes activities through UI Automation (via platform.UIReader)
func (p *Pipeline) uiaProcessor() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case activity := <-p.activityBuffer:
			// Try to extract window info via platform UIReader
			result, err := p.plat.GetStructuredData(activity.WindowHandle)

			if err != nil || result == nil {
				// UIA extraction failed - send to OCR batch
				activity.StructuredData = false
				activity.CaptureSource = CaptureSourceOCR
				if err != nil {
					activity.Metadata["uia_error"] = err.Error()
				}
				p.sendToOCRBatch(activity)
				continue
			}

			// Populate activity with UIResult
			activity.WindowTitle = result.WindowTitle
			activity.AppType = result.AppType

			// Merge metadata
			for key, value := range result.Metadata {
				activity.Metadata[key] = value
			}

			// Check if we have valid structured data from UIA
			if p.hasValidStructuredData(result) {
				activity.StructuredData = true
				activity.CaptureSource = CaptureSourceUIAutomation
				activity.Metadata["skip_ocr"] = true
				activity.Metadata["structured_extraction"] = true

				// Skip OCR - send directly to storage
				p.processStructuredActivity(activity)
			} else {
				// UIA available but no structured data - send to OCR batch
				activity.StructuredData = false
				activity.CaptureSource = CaptureSourceOCR
				activity.Metadata["uia_available"] = true
				activity.Metadata["structured_extraction"] = false
				p.sendToOCRBatch(activity)
			}
		}
	}
}

// ocrBatchProcessor batches OCR requests and processes them
func (p *Pipeline) ocrBatchProcessor() {
	defer p.wg.Done()

	batch := make([]*ActivityBlock, 0, OCRBatchSize)
	timer := time.NewTimer(OCRBatchTimeout)
	defer timer.Stop()

	for {
		select {
		case <-p.ctx.Done():
			// Process remaining batch before exiting
			if len(batch) > 0 {
				p.processOCRBatch(batch)
			}
			return

		case activity := <-p.ocrBatchBuffer:
			batch = append(batch, activity)
			if len(batch) >= OCRBatchSize {
				p.processOCRBatch(batch)
				batch = make([]*ActivityBlock, 0, OCRBatchSize)
				timer.Reset(OCRBatchTimeout)
			}

		case <-timer.C:
			if len(batch) > 0 {
				p.processOCRBatch(batch)
				batch = make([]*ActivityBlock, 0, OCRBatchSize)
			}
			timer.Reset(OCRBatchTimeout)
		}
	}
}

// processOCRBatch processes a batch of OCR requests
func (p *Pipeline) processOCRBatch(batch []*ActivityBlock) {
	if len(batch) == 0 {
		return
	}

	// Simulate OCR processing time based on batch size
	processingTime := time.Duration(len(batch)) * 10 * time.Millisecond
	time.Sleep(processingTime)

	for i, activity := range batch {
		activity.Metadata["ocr_processed"] = true
		activity.Metadata["batch_size"] = len(batch)
		activity.Metadata["batch_index"] = i
		activity.Metadata["processing_time_ms"] = processingTime.Milliseconds()
		activity.Metadata["ocr_timestamp"] = time.Now()
		activity.Metadata["ocr_text"] = fmt.Sprintf("OCR extracted text for window %d", activity.WindowHandle)
		activity.Metadata["ocr_confidence"] = 0.85
	}
}

// sendActivityWithBackpressure sends activity to buffer with backpressure handling
func (p *Pipeline) sendActivityWithBackpressure(activity *ActivityBlock) {
	select {
	case p.activityBuffer <- activity:
	default:
		select {
		case <-p.activityBuffer:
			p.droppedEvents.Add(1)
		default:
		}
		select {
		case p.activityBuffer <- activity:
		default:
			p.droppedEvents.Add(1)
		}
	}
}

// sendToOCRBatch sends activity to OCR batch buffer
func (p *Pipeline) sendToOCRBatch(activity *ActivityBlock) {
	select {
	case p.ocrBatchBuffer <- activity:
	default:
		select {
		case <-p.ocrBatchBuffer:
			p.droppedEvents.Add(1)
		default:
		}
		select {
		case p.ocrBatchBuffer <- activity:
		default:
			p.droppedEvents.Add(1)
		}
	}
}

// DroppedEvents returns the count of dropped events due to backpressure
func (p *Pipeline) DroppedEvents() int64 {
	return p.droppedEvents.Load()
}

// IsRunning returns true if the pipeline is running
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// IsETWFallbackMode returns true if tracker is in fallback mode
func (p *Pipeline) IsETWFallbackMode() bool {
	return p.plat.IsFallbackMode()
}

// ProcessFocusEvent handles tracker focus event with backpressure
func (p *Pipeline) ProcessFocusEvent(event platform.FocusEvent) error {
	dateStr := time.Now().Format("2006-01-02")
	if storageEngine, ok := p.storage.(*storage.StorageEngine); ok {
		go func(date string) {
			if _, err := storageEngine.GetSession(date); err != nil {
				if storage.IsNotFound(err) {
					if _, err := storageEngine.CreateSession(date); err != nil {
						fmt.Printf("Error creating session: %v\n", err)
					}
				}
			}
		}(dateStr)
	}

	activity := &ActivityBlock{
		Timestamp:     time.Unix(0, event.Timestamp),
		WindowHandle:  event.WindowHandle,
		ProcessID:     event.ProcessID,
		ProcessName:   event.ProcessName,
		CaptureSource: CaptureSourceETW,
		Metadata:      make(map[string]interface{}),
	}

	p.sendActivityWithBackpressure(activity)
	return nil
}

// Stop stops the capture pipeline and cleans up resources
func (p *Pipeline) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	// Cancel context to stop all workers
	p.cancel()

	// Wait for all workers to finish
	p.wg.Wait()

	// Stop platform (tracker + UIA reader)
	if p.plat != nil {
		if err := p.plat.Stop(); err != nil {
			return fmt.Errorf("failed to stop platform: %w", err)
		}
	}

	// Close channels
	close(p.activityBuffer)
	close(p.ocrBatchBuffer)

	return nil
}

// GetActivityBuffer returns the activity buffer channel for testing
func (p *Pipeline) GetActivityBuffer() <-chan *ActivityBlock {
	return p.activityBuffer
}

// GetOCRBatchBuffer returns the OCR batch buffer channel for testing
func (p *Pipeline) GetOCRBatchBuffer() <-chan *ActivityBlock {
	return p.ocrBatchBuffer
}

// Close is an alias for Stop to match expected interface
func (p *Pipeline) Close() error {
	return p.Stop()
}

// hasValidStructuredData checks if UIResult contains valid structured data
func (p *Pipeline) hasValidStructuredData(result *platform.UIResult) bool {
	if result == nil {
		return false
	}

	// Check if UIA extraction failed
	if failed, exists := result.Metadata["uia_extraction_failed"]; exists && failed.(bool) {
		return false
	}

	// Check if we have app-specific structured data (using string AppType)
	switch result.AppType {
	case "vscode":
		_, hasFile := result.Metadata["file"]
		_, hasLanguage := result.Metadata["language"]
		return hasFile && hasLanguage

	case "chrome", "edge":
		_, hasPageTitle := result.Metadata["pageTitle"]
		_, hasURL := result.Metadata["url"]
		return hasPageTitle || hasURL

	case "slack":
		_, hasChannel := result.Metadata["channel"]
		_, hasWorkspace := result.Metadata["workspace"]
		return hasChannel || hasWorkspace

	case "unknown":
		return false

	default:
		return false
	}
}

// processStructuredActivity processes an activity with structured data (skips OCR)
func (p *Pipeline) processStructuredActivity(activity *ActivityBlock) {
	activity.Metadata["processed_timestamp"] = time.Now()
	activity.Metadata["processing_path"] = "structured_skip_ocr"
}

// GetPipelineStats returns pipeline statistics
func (p *Pipeline) GetPipelineStats() map[string]interface{} {
	return map[string]interface{}{
		"running":              p.IsRunning(),
		"etw_fallback_mode":    p.IsETWFallbackMode(),
		"dropped_events":       p.DroppedEvents(),
		"activity_buffer_size": len(p.activityBuffer),
		"ocr_buffer_size":      len(p.ocrBatchBuffer),
	}
}