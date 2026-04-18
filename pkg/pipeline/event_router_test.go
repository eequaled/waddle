package pipeline

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"waddle/pkg/capture"
)

// mockProcessor keeps track of received events for testing
type mockProcessor struct {
	focusCount   atomic.Int32
	processCount atomic.Int32
}

func (m *mockProcessor) ProcessFocusEvent(ctx context.Context, event capture.FocusEvent) error {
	m.focusCount.Add(1)
	return nil
}

func (m *mockProcessor) ProcessProcessEvent(ctx context.Context, event capture.ProcessEvent) error {
	m.processCount.Add(1)
	return nil
}

func TestEventRouter(t *testing.T) {
	mockEngine := NewMockCaptureEngine()
	router := NewEventRouter(mockEngine)
	processor := &mockProcessor{}
	router.AddProcessor(processor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router.Start(ctx)

	// Send events
	mockEngine.focusEvents <- capture.FocusEvent{WindowHandle: 1}
	mockEngine.processEvents <- capture.ProcessEvent{ProcessID: 2}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	if processor.focusCount.Load() != 1 {
		t.Errorf("Expected 1 focus event, got %d", processor.focusCount.Load())
	}
	if processor.processCount.Load() != 1 {
		t.Errorf("Expected 1 process event, got %d", processor.processCount.Load())
	}

	router.Stop()
}

func TestFocusProcessorDebounce(t *testing.T) {
	mockEngine := NewMockCaptureEngine()
	screenshotQ := make(chan ScreenshotRequest, 10)
	processor := NewFocusProcessor(mockEngine, screenshotQ)

	ctx := context.Background()

	// First event should trigger screenshot
	_ = processor.ProcessFocusEvent(ctx, capture.FocusEvent{WindowHandle: 100})
	select {
	case <-screenshotQ:
		// success
	default:
		t.Errorf("Expected screenshot request for new window")
	}

	// Second event for same window should be debounced
	_ = processor.ProcessFocusEvent(ctx, capture.FocusEvent{WindowHandle: 100})
	select {
	case <-screenshotQ:
		t.Errorf("Expected screenshot request to be debounced")
	default:
		// success
	}

	// Third event for different window should trigger
	_ = processor.ProcessFocusEvent(ctx, capture.FocusEvent{WindowHandle: 200})
	select {
	case <-screenshotQ:
		// success
	default:
		t.Errorf("Expected screenshot request for new window")
	}
}

func TestScreenshotProcessorRateLimit(t *testing.T) {
	mockEngine := NewMockCaptureEngine()
	var captures atomic.Int32
	mockEngine.CaptureWindowFn = func(hwnd uintptr) ([]byte, error) {
		captures.Add(1)
		return []byte("test"), nil
	}

	screenshotQ := make(chan ScreenshotRequest, 10)
	processor := NewScreenshotProcessor(mockEngine, screenshotQ)
	// Override rate limit for faster test
	processor.rateLimit = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processor.Start(ctx)

	// Send two requests back to back for the same window
	screenshotQ <- ScreenshotRequest{HWND: 500}
	screenshotQ <- ScreenshotRequest{HWND: 500}

	time.Sleep(50 * time.Millisecond) // Let goroutine process

	if captures.Load() != 1 {
		t.Errorf("Expected exactly 1 capture due to rate limiting, got %d", captures.Load())
	}

	// Wait for rate limit to expire
	time.Sleep(100 * time.Millisecond)

	screenshotQ <- ScreenshotRequest{HWND: 500}
	time.Sleep(50 * time.Millisecond)

	if captures.Load() != 2 {
		t.Errorf("Expected 2 captures after rate limit expired, got %d", captures.Load())
	}

	cancel()
	processor.Stop()
}

func TestFocusProcessorBackpressure(t *testing.T) {
	mockEngine := NewMockCaptureEngine()
	// Small buffer for testing backpressure
	screenshotQ := make(chan ScreenshotRequest, 2)
	processor := NewFocusProcessor(mockEngine, screenshotQ)

	ctx := context.Background()

	// Fill the queue
	_ = processor.ProcessFocusEvent(ctx, capture.FocusEvent{WindowHandle: 100})
	_ = processor.ProcessFocusEvent(ctx, capture.FocusEvent{WindowHandle: 101})

	if len(screenshotQ) != 2 {
		t.Fatalf("Expected queue to be full (2), got %d", len(screenshotQ))
	}

	// Trigger backpressure (drop oldest)
	_ = processor.ProcessFocusEvent(ctx, capture.FocusEvent{WindowHandle: 102})

	if len(screenshotQ) != 2 {
		t.Fatalf("Expected queue to be full (2) after backpressure, got %d", len(screenshotQ))
	}

	// Check that the oldest was dropped and replaced
	newFirstReq := <-screenshotQ
	if newFirstReq.HWND == 100 {
		t.Errorf("Oldest item (HWND 100) was not dropped")
	}
	if newFirstReq.HWND != 101 {
		t.Errorf("Expected new oldest item to be HWND 101, got %d", newFirstReq.HWND)
	}

	secondReq := <-screenshotQ
	if secondReq.HWND != 102 {
		t.Errorf("Expected newest item to be HWND 102, got %d", secondReq.HWND)
	}
}
