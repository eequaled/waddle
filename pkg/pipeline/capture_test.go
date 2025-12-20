package pipeline

import (
	"testing"
	"time"
	"waddle/pkg/capture/uia"
)

// TestPipelineCreation tests basic pipeline creation and cleanup
func TestPipelineCreation(t *testing.T) {
	pipeline, err := NewPipeline()
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Verify pipeline is not running initially
	if pipeline.IsRunning() {
		t.Errorf("Pipeline should not be running after creation")
	}

	// Test start
	err = pipeline.Start()
	if err != nil {
		t.Errorf("Failed to start pipeline: %v", err)
	}

	// Verify pipeline is running
	if !pipeline.IsRunning() {
		t.Errorf("Pipeline should be running after start")
	}

	// Test stop
	err = pipeline.Stop()
	if err != nil {
		t.Errorf("Failed to stop pipeline: %v", err)
	}

	// Verify pipeline is stopped
	if pipeline.IsRunning() {
		t.Errorf("Pipeline should be stopped after stop")
	}
}

// TestPipelineDoubleStart tests double start protection
func TestPipelineDoubleStart(t *testing.T) {
	pipeline, err := NewPipeline()
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Start pipeline
	err = pipeline.Start()
	if err != nil {
		t.Errorf("Failed to start pipeline: %v", err)
	}

	// Try to start again
	err = pipeline.Start()
	if err == nil {
		t.Errorf("Double start should return error")
	}
}

// TestPipelineBackpressure tests backpressure handling
func TestPipelineBackpressure(t *testing.T) {
	pipeline, err := NewPipeline()
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Test that dropped events counter starts at 0
	if pipeline.DroppedEvents() != 0 {
		t.Errorf("DroppedEvents should be 0 initially, got %d", pipeline.DroppedEvents())
	}

	// Create test activity
	activity := &ActivityBlock{
		Timestamp:     time.Now(),
		WindowHandle:  uintptr(12345),
		ProcessID:     uint32(1000),
		ProcessName:   "test.exe",
		WindowTitle:   "Test Window",
		AppType:       uia.AppTypeUnknown,
		CaptureSource: CaptureSourceETW,
		Metadata:      make(map[string]interface{}),
	}

	// Test backpressure handling by sending activity when pipeline is not started
	pipeline.sendActivityWithBackpressure(activity)

	// The activity should be buffered or dropped, but no panic should occur
}

// TestActivityBlockCreation tests ActivityBlock structure
func TestActivityBlockCreation(t *testing.T) {
	activity := &ActivityBlock{
		Timestamp:     time.Now(),
		WindowHandle:  uintptr(12345),
		ProcessID:     uint32(1000),
		ProcessName:   "test.exe",
		WindowTitle:   "Test Window",
		AppType:       uia.AppTypeVSCode,
		CaptureSource: CaptureSourceUIAutomation,
		StructuredData: true,
		Metadata:      map[string]interface{}{
			"file":     "main.go",
			"language": "go",
		},
	}

	// Verify all fields are set correctly
	if activity.ProcessID != 1000 {
		t.Errorf("ProcessID should be 1000, got %d", activity.ProcessID)
	}

	if activity.AppType != uia.AppTypeVSCode {
		t.Errorf("AppType should be VSCode, got %v", activity.AppType)
	}

	if activity.CaptureSource != CaptureSourceUIAutomation {
		t.Errorf("CaptureSource should be UI Automation, got %v", activity.CaptureSource)
	}

	if !activity.StructuredData {
		t.Errorf("StructuredData should be true")
	}

	if activity.Metadata["file"] != "main.go" {
		t.Errorf("Metadata file should be 'main.go', got %v", activity.Metadata["file"])
	}
}

// TestCaptureSourceTypes tests CaptureSource type values
func TestCaptureSourceTypes(t *testing.T) {
	sources := []CaptureSource{
		CaptureSourceETW,
		CaptureSourcePolling,
		CaptureSourceUIAutomation,
		CaptureSourceOCR,
	}

	expectedValues := []string{
		"etw",
		"polling",
		"ui_automation",
		"ocr",
	}

	for i, source := range sources {
		if string(source) != expectedValues[i] {
			t.Errorf("CaptureSource %d should be %q, got %q", i, expectedValues[i], string(source))
		}
	}
}

// TestPipelineETWFallback tests ETW fallback mode detection
func TestPipelineETWFallback(t *testing.T) {
	pipeline, err := NewPipeline()
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Check if ETW is in fallback mode (expected on non-admin systems)
	isFallback := pipeline.IsETWFallbackMode()
	t.Logf("ETW fallback mode: %v", isFallback)

	// This is informational - both true and false are valid depending on system privileges
}

// TestOCRBatchProcessing tests OCR batch processing logic
func TestOCRBatchProcessing(t *testing.T) {
	pipeline, err := NewPipeline()
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Create test batch
	batch := make([]*ActivityBlock, 3)
	for i := 0; i < 3; i++ {
		batch[i] = &ActivityBlock{
			Timestamp:     time.Now(),
			WindowHandle:  uintptr(12345 + i),
			ProcessID:     uint32(1000 + i),
			ProcessName:   "test.exe",
			WindowTitle:   "Test Window",
			AppType:       uia.AppTypeUnknown,
			CaptureSource: CaptureSourceOCR,
			Metadata:      make(map[string]interface{}),
		}
	}

	// Process batch
	pipeline.processOCRBatch(batch)

	// Verify batch processing metadata
	for i, activity := range batch {
		if processed, exists := activity.Metadata["ocr_processed"]; !exists || !processed.(bool) {
			t.Errorf("Activity %d should be marked as OCR processed", i)
		}

		if batchSize, exists := activity.Metadata["batch_size"]; !exists || batchSize.(int) != 3 {
			t.Errorf("Activity %d should have batch_size=3, got %v", i, batchSize)
		}
	}
}

// TestPipelineChannelBuffers tests channel buffer access
func TestPipelineChannelBuffers(t *testing.T) {
	pipeline, err := NewPipeline()
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Test activity buffer access
	activityBuffer := pipeline.GetActivityBuffer()
	if activityBuffer == nil {
		t.Errorf("Activity buffer should not be nil")
	}

	// Test OCR batch buffer access
	ocrBuffer := pipeline.GetOCRBatchBuffer()
	if ocrBuffer == nil {
		t.Errorf("OCR batch buffer should not be nil")
	}

	// Verify buffer capacities
	if cap(pipeline.activityBuffer) != ActivityBufferSize {
		t.Errorf("Activity buffer capacity should be %d, got %d", 
			ActivityBufferSize, cap(pipeline.activityBuffer))
	}

	if cap(pipeline.ocrBatchBuffer) != OCRBatchBufferSize {
		t.Errorf("OCR batch buffer capacity should be %d, got %d", 
			OCRBatchBufferSize, cap(pipeline.ocrBatchBuffer))
	}
}

// BenchmarkPipelineBackpressure benchmarks backpressure handling
func BenchmarkPipelineBackpressure(b *testing.B) {
	pipeline, err := NewPipeline()
	if err != nil {
		b.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	activity := &ActivityBlock{
		Timestamp:     time.Now(),
		WindowHandle:  uintptr(12345),
		ProcessID:     uint32(1000),
		ProcessName:   "test.exe",
		WindowTitle:   "Test Window",
		AppType:       uia.AppTypeUnknown,
		CaptureSource: CaptureSourceETW,
		Metadata:      make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.sendActivityWithBackpressure(activity)
	}
}

// BenchmarkOCRBatchProcessing benchmarks OCR batch processing
func BenchmarkOCRBatchProcessing(b *testing.B) {
	pipeline, err := NewPipeline()
	if err != nil {
		b.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Create test batch
	batch := make([]*ActivityBlock, OCRBatchSize)
	for i := 0; i < OCRBatchSize; i++ {
		batch[i] = &ActivityBlock{
			Timestamp:     time.Now(),
			WindowHandle:  uintptr(12345 + i),
			ProcessID:     uint32(1000 + i),
			ProcessName:   "test.exe",
			WindowTitle:   "Test Window",
			AppType:       uia.AppTypeUnknown,
			CaptureSource: CaptureSourceOCR,
			Metadata:      make(map[string]interface{}),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.processOCRBatch(batch)
	}
}