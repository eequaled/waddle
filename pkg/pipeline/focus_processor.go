package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"waddle/pkg/capture"
)

// FocusProcessor implements EventProcessor to handle window focus changes.
type FocusProcessor struct {
	engine      capture.CaptureEngine
	screenshotQ chan ScreenshotRequest
	lastHWND    uintptr
	mu          sync.Mutex
}

// NewFocusProcessor creates a new FocusProcessor.
func NewFocusProcessor(engine capture.CaptureEngine, screenshotQ chan ScreenshotRequest) *FocusProcessor {
	return &FocusProcessor{
		engine:      engine,
		screenshotQ: screenshotQ,
	}
}

// ProcessFocusEvent handles a new focus event.
func (p *FocusProcessor) ProcessFocusEvent(ctx context.Context, event capture.FocusEvent) error {
	p.mu.Lock()
	// Debounce: ignore if it's the exact same window
	if p.lastHWND == event.WindowHandle {
		p.mu.Unlock()
		return nil
	}
	p.lastHWND = event.WindowHandle
	p.mu.Unlock()

	// Extract window info
	info, err := p.engine.GetWindowInfo(event.WindowHandle)
	if err != nil {
		return fmt.Errorf("failed to get window info: %w", err)
	}

	// Dispatch screenshot request with backpressure
	req := ScreenshotRequest{
		HWND:       event.WindowHandle,
		WindowInfo: info,
		Timestamp:  time.Now(),
	}

	select {
	case p.screenshotQ <- req:
		// Successfully queued
	default:
		// Queue full - drop oldest and retry
		select {
		case <-p.screenshotQ:
		default:
		}
		select {
		case p.screenshotQ <- req:
		default:
			// Still full, drop
		}
	}

	return nil
}

// ProcessProcessEvent ignores process events.
func (p *FocusProcessor) ProcessProcessEvent(ctx context.Context, event capture.ProcessEvent) error {
	// Not handled by FocusProcessor
	return nil
}
