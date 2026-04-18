package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"waddle/pkg/capture"
)

// ScreenshotProcessor handles capturing screenshots asynchronously.
type ScreenshotProcessor struct {
	engine      capture.CaptureEngine
	screenshotQ <-chan ScreenshotRequest
	lastCapture map[uintptr]time.Time
	mu          sync.Mutex
	wg          sync.WaitGroup
	// rateLimit specifies the minimum time between captures of the same window.
	rateLimit time.Duration
}

// NewScreenshotProcessor creates a new ScreenshotProcessor.
func NewScreenshotProcessor(engine capture.CaptureEngine, screenshotQ <-chan ScreenshotRequest) *ScreenshotProcessor {
	return &ScreenshotProcessor{
		engine:      engine,
		screenshotQ: screenshotQ,
		lastCapture: make(map[uintptr]time.Time),
		rateLimit:   5 * time.Second,
	}
}

// Start begins the background processing loop for screenshots.
func (p *ScreenshotProcessor) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.processLoop(ctx)
}

// Stop waits for the processing loop to gracefully shutdown.
func (p *ScreenshotProcessor) Stop() {
	p.wg.Wait()
}

func (p *ScreenshotProcessor) processLoop(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-p.screenshotQ:
			if !ok {
				return
			}
			p.handleScreenshotRequest(ctx, req)
		}
	}
}

func (p *ScreenshotProcessor) handleScreenshotRequest(ctx context.Context, req ScreenshotRequest) {
	p.mu.Lock()
	last, exists := p.lastCapture[req.HWND]
	now := time.Now()

	// Rate limiting: 1 screenshot per window per 5 seconds
	if exists && now.Sub(last) < p.rateLimit {
		p.mu.Unlock()
		return
	}
	// Cleanup map if it grows too large (prevent unbounded growth)
	if len(p.lastCapture) > 1000 {
		cutoff := now.Add(-10 * time.Second)
		for k, v := range p.lastCapture {
			if v.Before(cutoff) {
				delete(p.lastCapture, k)
			}
		}
	}

	p.lastCapture[req.HWND] = now
	p.mu.Unlock()

	// Capture the screenshot bytes
	bytes, err := p.engine.CaptureWindow(req.HWND)
	if err != nil {
		fmt.Printf("Warning: failed to capture screenshot for HWND %d: %v\n", req.HWND, err)
		return
	}

	// Downstream processing would occur here.
	// For now, we simulate processing by logging or passing to storage.
	_ = bytes
}
