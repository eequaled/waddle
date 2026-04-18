package pipeline

import (
	"context"
	"sync"
	"time"

	"waddle/pkg/capture"
)

// EventProcessor defines the interface for handling pipeline events.
type EventProcessor interface {
	ProcessFocusEvent(ctx context.Context, event capture.FocusEvent) error
	ProcessProcessEvent(ctx context.Context, event capture.ProcessEvent) error
}

// ScreenshotRequest represents a request to capture a window.
type ScreenshotRequest struct {
	HWND       uintptr
	WindowInfo *capture.WindowInfo
	Timestamp  time.Time
}

// EventRouter reads events from the CaptureEngine and routes them to processors.
type EventRouter struct {
	engine      capture.CaptureEngine
	processors  []EventProcessor
	screenshotQ chan ScreenshotRequest // buffered, 100
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewEventRouter creates a new EventRouter.
func NewEventRouter(engine capture.CaptureEngine) *EventRouter {
	return &EventRouter{
		engine:      engine,
		processors:  make([]EventProcessor, 0),
		screenshotQ: make(chan ScreenshotRequest, 100),
		stopCh:      make(chan struct{}),
	}
}

// AddProcessor registers an EventProcessor.
func (r *EventRouter) AddProcessor(p EventProcessor) {
	r.processors = append(r.processors, p)
}

// ScreenshotQueue returns the screenshot request channel (for wiring to ScreenshotProcessor).
func (r *EventRouter) ScreenshotQueue() chan ScreenshotRequest {
	return r.screenshotQ
}

// Start begins reading events from the CaptureEngine.
func (r *EventRouter) Start(ctx context.Context) {
	r.wg.Add(2)
	go r.routeFocusEvents(ctx)
	go r.routeProcessEvents(ctx)
}

// Stop gracefully stops the router.
func (r *EventRouter) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

func (r *EventRouter) routeFocusEvents(ctx context.Context) {
	defer r.wg.Done()
	focusChan := r.engine.FocusEvents()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case ev, ok := <-focusChan:
			if !ok {
				return
			}
			for _, p := range r.processors {
				_ = p.ProcessFocusEvent(ctx, ev) // Errors are ignored at router level
			}
		}
	}
}

func (r *EventRouter) routeProcessEvents(ctx context.Context) {
	defer r.wg.Done()
	processChan := r.engine.ProcessEvents()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case ev, ok := <-processChan:
			if !ok {
				return
			}
			for _, p := range r.processors {
				_ = p.ProcessProcessEvent(ctx, ev)
			}
		}
	}
}
