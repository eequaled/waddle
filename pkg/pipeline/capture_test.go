package pipeline

import (
	"testing"
	"time"
)

// newTestPipeline creates a pipeline with a mock capture engine for tests.
func newTestPipeline(t testing.TB) *Pipeline {
	t.Helper()
	engine := NewMockCaptureEngine()
	p, err := NewPipeline(nil, engine)
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
		t.Logf("Start returned error: %v", err)
	}

	err = p.Stop()
	if err != nil {
		t.Errorf("Failed to stop pipeline: %v", err)
	}

	if p.IsRunning() {
		t.Errorf("Pipeline should be stopped after stop")
	}
}

// TestPipelineStats tests statistics reporting
func TestPipelineStats(t *testing.T) {
	p := newTestPipeline(t)
	defer p.Stop()

	stats := p.GetPipelineStats()
	if stats.Running {
		t.Errorf("Pipeline should not be reported as running")
	}
	if stats.Source != "none" {
		t.Errorf("Pipeline source should be 'none', got %q", stats.Source)
	}

	_ = p.Start()
	stats = p.GetPipelineStats()
	if !stats.Running {
		t.Errorf("Pipeline should be reported as running")
	}
	if stats.Source != "etw" && stats.Source != "polling" {
		t.Errorf("Pipeline source should be 'etw' or 'polling', got %q", stats.Source)
	}
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