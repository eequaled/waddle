package pipeline

import (
	"testing"
	"time"

	"waddle/pkg/infra/config"
	"waddle/pkg/platform"
)

// newTestPipeline creates a pipeline with a stub platform for tests.
func newTestPipeline(t testing.TB) *Pipeline {
	t.Helper()
	cfg := config.DefaultConfig()
	plat, err := platform.NewPlatform(&cfg)
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	p, err := NewPipeline(nil, plat)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	return p
}

// TestPipelineCreation tests basic pipeline creation and cleanup
func TestPipelineCreation(t *testing.T) {
	p := newTestPipeline(t)
	defer p.Stop()

	if p.IsRunning() {
		t.Errorf("Pipeline should not be running after creation")
	}

	err := p.Start()
	if err != nil {
		t.Logf("Start returned (may be expected on non-Windows): %v", err)
	}

	err = p.Stop()
	if err != nil {
		t.Errorf("Failed to stop pipeline: %v", err)
	}

	if p.IsRunning() {
		t.Errorf("Pipeline should be stopped after stop")
	}
}

// TestPipelineBackpressure tests backpressure handling
func TestPipelineBackpressure(t *testing.T) {
	p := newTestPipeline(t)
	defer p.Stop()

	if p.DroppedEvents() != 0 {
		t.Errorf("DroppedEvents should be 0 initially, got %d", p.DroppedEvents())
	}

	activity := &ActivityBlock{
		Timestamp:     time.Now(),
		WindowHandle:  uintptr(12345),
		ProcessID:     uint32(1000),
		ProcessName:   "test.exe",
		WindowTitle:   "Test Window",
		AppType:       "unknown",
		CaptureSource: CaptureSourceETW,
		Metadata:      make(map[string]interface{}),
	}

	// Send should not panic
	p.sendActivityWithBackpressure(activity)
}

// TestActivityBlockCreation tests ActivityBlock structure
func TestActivityBlockCreation(t *testing.T) {
	activity := &ActivityBlock{
		Timestamp:      time.Now(),
		WindowHandle:   uintptr(12345),
		ProcessID:      uint32(1000),
		ProcessName:    "test.exe",
		WindowTitle:    "Test Window",
		AppType:        "vscode",
		CaptureSource:  CaptureSourceUIAutomation,
		StructuredData: true,
		Metadata: map[string]interface{}{
			"file":     "main.go",
			"language": "go",
		},
	}

	if activity.ProcessID != 1000 {
		t.Errorf("ProcessID should be 1000, got %d", activity.ProcessID)
	}
	if activity.AppType != "vscode" {
		t.Errorf("AppType should be 'vscode', got %v", activity.AppType)
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
	expectedValues := []string{"etw", "polling", "ui_automation", "ocr"}

	for i, source := range sources {
		if string(source) != expectedValues[i] {
			t.Errorf("CaptureSource %d should be %q, got %q", i, expectedValues[i], string(source))
		}
	}
}

// TestPipelineETWFallback tests ETW fallback mode detection
func TestPipelineETWFallback(t *testing.T) {
	p := newTestPipeline(t)
	defer p.Stop()

	isFallback := p.IsETWFallbackMode()
	t.Logf("ETW fallback mode: %v", isFallback)
}

// TestOCRBatchProcessing tests OCR batch processing logic
func TestOCRBatchProcessing(t *testing.T) {
	p := newTestPipeline(t)
	defer p.Stop()

	batch := make([]*ActivityBlock, 3)
	for i := 0; i < 3; i++ {
		batch[i] = &ActivityBlock{
			Timestamp:     time.Now(),
			WindowHandle:  uintptr(12345 + i),
			ProcessID:     uint32(1000 + i),
			ProcessName:   "test.exe",
			WindowTitle:   "Test Window",
			AppType:       "unknown",
			CaptureSource: CaptureSourceOCR,
			Metadata:      make(map[string]interface{}),
		}
	}

	p.processOCRBatch(batch)

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
	p := newTestPipeline(t)
	defer p.Stop()

	activityBuffer := p.GetActivityBuffer()
	if activityBuffer == nil {
		t.Errorf("Activity buffer should not be nil")
	}
	ocrBuffer := p.GetOCRBatchBuffer()
	if ocrBuffer == nil {
		t.Errorf("OCR batch buffer should not be nil")
	}
	if cap(p.activityBuffer) != ActivityBufferSize {
		t.Errorf("Activity buffer capacity should be %d, got %d",
			ActivityBufferSize, cap(p.activityBuffer))
	}
	if cap(p.ocrBatchBuffer) != OCRBatchBufferSize {
		t.Errorf("OCR batch buffer capacity should be %d, got %d",
			OCRBatchBufferSize, cap(p.ocrBatchBuffer))
	}
}

// BenchmarkPipelineBackpressure benchmarks backpressure handling
func BenchmarkPipelineBackpressure(b *testing.B) {
	p := newTestPipeline(b)
	defer p.Stop()

	activity := &ActivityBlock{
		Timestamp:     time.Now(),
		WindowHandle:  uintptr(12345),
		ProcessID:     uint32(1000),
		ProcessName:   "test.exe",
		WindowTitle:   "Test Window",
		AppType:       "unknown",
		CaptureSource: CaptureSourceETW,
		Metadata:      make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.sendActivityWithBackpressure(activity)
	}
}

// BenchmarkOCRBatchProcessing benchmarks OCR batch processing
func BenchmarkOCRBatchProcessing(b *testing.B) {
	p := newTestPipeline(b)
	defer p.Stop()

	batch := make([]*ActivityBlock, OCRBatchSize)
	for i := 0; i < OCRBatchSize; i++ {
		batch[i] = &ActivityBlock{
			Timestamp:     time.Now(),
			WindowHandle:  uintptr(12345 + i),
			ProcessID:     uint32(1000 + i),
			ProcessName:   "test.exe",
			WindowTitle:   "Test Window",
			AppType:       "unknown",
			CaptureSource: CaptureSourceOCR,
			Metadata:      make(map[string]interface{}),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.processOCRBatch(batch)
	}
}