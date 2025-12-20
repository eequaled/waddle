package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"waddle/pkg/capture/uia"
	"waddle/pkg/storage"
	"waddle/pkg/tracker/etw"
)

// CaptureSource indicates where the activity data came from
type CaptureSource string

const (
	CaptureSourceETW        CaptureSource = "etw"
	CaptureSourcePolling    CaptureSource = "polling"
	CaptureSourceUIAutomation CaptureSource = "ui_automation"
	CaptureSourceOCR        CaptureSource = "ocr"
)

// ActivityBlock represents a captured activity with metadata
type ActivityBlock struct {
	Timestamp      time.Time
	WindowHandle   uintptr
	ProcessID      uint32
	ProcessName    string
	WindowTitle    string
	AppType        uia.AppType
	Metadata       map[string]interface{}
	CaptureSource  CaptureSource
	StructuredData bool // True if data came from UIA, false if from OCR
}

// Pipeline orchestrates the hybrid capture pipeline: ETW → UIA → OCR
type Pipeline struct {
	etwConsumer    *etw.Consumer
	uiaReader      *uia.Reader
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

// NewPipeline creates a new hybrid capture pipeline
func NewPipeline(storage interface{}) (*Pipeline, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create ETW consumer
	etwConsumer, err := etw.NewConsumer()
	if err != nil {
		// ETW failed but consumer is in fallback mode - this is OK
		// Continue with polling mode
	}
	
	// Create UIA reader
	uiaReader, err := uia.NewReader()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create UIA reader: %w", err)
	}
	
	p := &Pipeline{
		etwConsumer:    etwConsumer,
		uiaReader:      uiaReader,
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
	
	// Start ETW consumer
	err := p.etwConsumer.Start()
	if err != nil && !p.etwConsumer.IsFallbackMode() {
		return fmt.Errorf("failed to start ETW consumer: %w", err)
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
			
		case focusEvent := <-p.etwConsumer.FocusEvents():
			// Create activity block from ETW event
			activity := &ActivityBlock{
				Timestamp:     focusEvent.Timestamp,
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

// uiaProcessor processes activities through UI Automation
func (p *Pipeline) uiaProcessor() {
	defer p.wg.Done()
	
	for {
		select {
		case <-p.ctx.Done():
			return
			
		case activity := <-p.activityBuffer:
			// Try to extract window info using UI Automation
			windowInfo, err := p.uiaReader.GetWindowInfo(activity.WindowHandle)
			
			if err != nil || windowInfo == nil {
				// UIA extraction failed - send to OCR batch
				activity.StructuredData = false
				activity.CaptureSource = CaptureSourceOCR
				activity.Metadata["uia_error"] = err.Error()
				p.sendToOCRBatch(activity)
				continue
			}
			
			// Populate activity with window info
			activity.WindowTitle = windowInfo.WindowTitle
			activity.AppType = windowInfo.AppType
			
			// Merge metadata from window info
			for key, value := range windowInfo.Metadata {
				activity.Metadata[key] = value
			}
			
			// Check if we have valid structured data from UIA
			if p.hasValidStructuredData(windowInfo) {
				// UIA extraction succeeded - we have structured data
				activity.StructuredData = true
				activity.CaptureSource = CaptureSourceUIAutomation
				activity.Metadata["skip_ocr"] = true
				activity.Metadata["structured_extraction"] = true
				
				// Skip OCR - send directly to storage (simulated)
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
			// Add to batch
			batch = append(batch, activity)
			
			// Flush if batch is full
			if len(batch) >= OCRBatchSize {
				p.processOCRBatch(batch)
				batch = make([]*ActivityBlock, 0, OCRBatchSize)
				timer.Reset(OCRBatchTimeout)
			}
			
		case <-timer.C:
			// Flush batch on timeout
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
	
	// In a real implementation, this would:
	// 1. Take screenshots of all windows in batch
	// 2. Send screenshots to OCR service in single batch request
	// 3. Parse OCR results and populate activity metadata
	// 4. Send completed activities to storage
	
	// Simulate OCR processing time based on batch size
	processingTime := time.Duration(len(batch)) * 10 * time.Millisecond
	time.Sleep(processingTime)
	
	// Mark all activities as OCR processed
	for i, activity := range batch {
		activity.Metadata["ocr_processed"] = true
		activity.Metadata["batch_size"] = len(batch)
		activity.Metadata["batch_index"] = i
		activity.Metadata["processing_time_ms"] = processingTime.Milliseconds()
		activity.Metadata["ocr_timestamp"] = time.Now()
		
		// Simulate OCR text extraction
		activity.Metadata["ocr_text"] = fmt.Sprintf("OCR extracted text for window %d", activity.WindowHandle)
		activity.Metadata["ocr_confidence"] = 0.85 // Simulated confidence score
	}
}

// sendActivityWithBackpressure sends activity to buffer with backpressure handling
func (p *Pipeline) sendActivityWithBackpressure(activity *ActivityBlock) {
	select {
	case p.activityBuffer <- activity:
		// Activity sent successfully
	default:
		// Buffer is full - drop oldest activity and add new one
		select {
		case <-p.activityBuffer:
			p.droppedEvents.Add(1)
		default:
		}
		
		select {
		case p.activityBuffer <- activity:
		default:
			// Still couldn't send - increment dropped counter
			p.droppedEvents.Add(1)
		}
	}
}

// sendToOCRBatch sends activity to OCR batch buffer
func (p *Pipeline) sendToOCRBatch(activity *ActivityBlock) {
	select {
	case p.ocrBatchBuffer <- activity:
		// Activity sent successfully
	default:
		// Buffer is full - drop oldest activity and add new one
		select {
		case <-p.ocrBatchBuffer:
			p.droppedEvents.Add(1)
		default:
		}
		
		select {
		case p.ocrBatchBuffer <- activity:
		default:
			// Still couldn't send - increment dropped counter
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

// IsETWFallbackMode returns true if ETW is in fallback mode
func (p *Pipeline) IsETWFallbackMode() bool {
	return p.etwConsumer.IsFallbackMode()
}

// ProcessFocusEvent handles ETW focus event with backpressure
func (p *Pipeline) ProcessFocusEvent(event etw.FocusEvent) error {
	// Create session for today's date if it doesn't exist
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
	
	// Create activity block from ETW event
	activity := &ActivityBlock{
		Timestamp:     event.Timestamp,
		WindowHandle:  event.WindowHandle,
		ProcessID:     event.ProcessID,
		ProcessName:   event.ProcessName,
		CaptureSource: CaptureSourceETW,
		Metadata:      make(map[string]interface{}),
	}
	
	// Send to UIA processor with backpressure handling
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
	
	// Stop ETW consumer
	if p.etwConsumer != nil {
		err := p.etwConsumer.Close()
		if err != nil {
			return fmt.Errorf("failed to close ETW consumer: %w", err)
		}
	}
	
	// Stop UIA reader
	if p.uiaReader != nil {
		err := p.uiaReader.Close()
		if err != nil {
			return fmt.Errorf("failed to close UIA reader: %w", err)
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

// hasValidStructuredData checks if window info contains valid structured data
func (p *Pipeline) hasValidStructuredData(windowInfo *uia.WindowInfo) bool {
	if windowInfo == nil {
		return false
	}
	
	// Check if UIA extraction failed
	if failed, exists := windowInfo.Metadata["uia_extraction_failed"]; exists && failed.(bool) {
		return false
	}
	
	// Check if we have app-specific structured data
	switch windowInfo.AppType {
	case uia.AppTypeVSCode:
		// VS Code should have file and language info
		_, hasFile := windowInfo.Metadata["file"]
		_, hasLanguage := windowInfo.Metadata["language"]
		return hasFile && hasLanguage
		
	case uia.AppTypeChrome, uia.AppTypeEdge:
		// Browsers should have page title or URL
		_, hasPageTitle := windowInfo.Metadata["pageTitle"]
		_, hasURL := windowInfo.Metadata["url"]
		return hasPageTitle || hasURL
		
	case uia.AppTypeSlack:
		// Slack should have channel or workspace info
		_, hasChannel := windowInfo.Metadata["channel"]
		_, hasWorkspace := windowInfo.Metadata["workspace"]
		return hasChannel || hasWorkspace
		
	case uia.AppTypeUnknown:
		// Unknown apps don't have structured data
		return false
		
	default:
		return false
	}
}

// processStructuredActivity processes an activity with structured data (skips OCR)
func (p *Pipeline) processStructuredActivity(activity *ActivityBlock) {
	// In a real implementation, this would send the activity directly to storage
	// since we have structured data and don't need OCR
	
	// Add processing metadata
	activity.Metadata["processed_timestamp"] = time.Now()
	activity.Metadata["processing_path"] = "structured_skip_ocr"
	
	// Simulate storage operation
	// storage.SaveActivity(activity)
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