package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"waddle/pkg/capture"
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

// PipelineStats describes capture pipeline runtime status for UI/API consumers.
type PipelineStats struct {
	Running            bool   `json:"running"`
	Source             string `json:"source"`
	ETWFallbackMode    bool   `json:"etwFallbackMode"`
	DroppedEvents      int64  `json:"droppedEvents"`
	ActivityBufferSize int    `json:"activityBufferSize"`
	OCRBufferSize      int    `json:"ocrBufferSize"`
}

// Pipeline orchestrates the hybrid capture pipeline: Sensing → Processing → Storage
type Pipeline struct {
	engine         capture.CaptureEngine
	storage        interface{} // Storage engine interface
	ctx            context.Context
	cancel         context.CancelFunc
	router         *EventRouter
	focusProc      *FocusProcessor
	screenshotProc *ScreenshotProcessor
	mu             sync.RWMutex
	running        bool
}

// NewPipeline creates a new hybrid capture pipeline.
func NewPipeline(storage interface{}, engine capture.CaptureEngine) (*Pipeline, error) {
	ctx, cancel := context.WithCancel(context.Background())

	if engine == nil {
		cancel()
		return nil, fmt.Errorf("capture engine is required")
	}

	router := NewEventRouter(engine)
	focusProc := NewFocusProcessor(engine, router.ScreenshotQueue())
	screenshotProc := NewScreenshotProcessor(engine, router.ScreenshotQueue())

	// Wire the processor to the router
	router.AddProcessor(focusProc)

	p := &Pipeline{
		engine:         engine,
		storage:        storage,
		ctx:            ctx,
		cancel:         cancel,
		router:         router,
		focusProc:      focusProc,
		screenshotProc: screenshotProc,
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

	// Start capture engine
	err := p.engine.Start(p.ctx)
	if err != nil && !p.engine.IsFallbackMode() {
		return fmt.Errorf("failed to start capture engine: %w", err)
	}

	// Start routing and processing
	p.router.Start(p.ctx)
	p.screenshotProc.Start(p.ctx)

	p.running = true
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

	// Cancel context to signal goroutines to stop
	p.cancel()

	// Stop routing and processing gracefully
	p.router.Stop()
	p.screenshotProc.Stop()

	// Stop capture engine
	if err := p.engine.Stop(); err != nil {
		return fmt.Errorf("failed to stop capture engine: %w", err)
	}

	return nil
}

// Close is an alias for Stop to match expected interface
func (p *Pipeline) Close() error {
	return p.Stop()
}

// IsRunning returns true if the pipeline is running
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetPipelineStats returns pipeline statistics.
func (p *Pipeline) GetPipelineStats() PipelineStats {
	running := p.IsRunning()
	fallback := false
	source := "none"
	var dropped int64 = 0

	if p.engine != nil {
		fallback = p.engine.IsFallbackMode()
		dropped = p.engine.DroppedEvents()
	}

	if running {
		if fallback {
			source = string(CaptureSourcePolling)
		} else {
			source = string(CaptureSourceETW)
		}
	}

	return PipelineStats{
		Running:            running,
		Source:             source,
		ETWFallbackMode:    fallback,
		DroppedEvents:      dropped,
		ActivityBufferSize: 0, // Migrated to channels in processor
		OCRBufferSize:      0, // Will be reintegrated with storage layer
	}
}

// Provide storage type for app.go
// This is a minimal stub to keep app.go compiling regarding pipeline features
func (p *Pipeline) GetActivityBuffer() <-chan *ActivityBlock {
	// Not used in new architecture directly
	return make(chan *ActivityBlock)
}
