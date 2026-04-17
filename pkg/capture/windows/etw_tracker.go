//go:build windows

package windows

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"waddle/pkg/capture"

	"github.com/tekert/golang-etw/etw"
)

// ETWTracker subscribes to Windows kernel events for zero-overhead capture.
type ETWTracker struct {
	session       *etw.RealTimeSession
	consumer      *etw.Consumer
	ctx           context.Context
	cancel        context.CancelFunc
	focusEvents   chan capture.FocusEvent
	processEvents chan capture.ProcessEvent
	fallbackMode  bool
	droppedEvents atomic.Int64
	mu            sync.RWMutex
	running       bool
}

const (
	// ETW Provider GUIDs
	Win32kProviderGUID        = "{8c416c79-d49b-4f01-a467-e56d3aa8234c}" // Microsoft-Windows-Win32k
	KernelProcessProviderGUID = "{22fb2cd6-0e7b-422b-a0c7-2fad1fd0e716}" // Microsoft-Windows-Kernel-Process
)

// NewETWTracker creates ETW consumer subscribed to Win32k and Kernel-Process providers.
func NewETWTracker() (*ETWTracker, error) {
	ctx, cancel := context.WithCancel(context.Background())

	t := &ETWTracker{
		ctx:           ctx,
		cancel:        cancel,
		focusEvents:   make(chan capture.FocusEvent, capture.EventBufferSize),
		processEvents: make(chan capture.ProcessEvent, capture.EventBufferSize),
		fallbackMode:  false,
	}

	// Try to create ETW session
	session := etw.NewRealTimeSession("WaddleETWSession")
	if session == nil {
		// ETW initialization failed - set fallback mode
		t.fallbackMode = true
		// Log warning notification to user
		fmt.Printf("⚠️  ETW session creation failed, falling back to polling mode\n")
		fmt.Printf("   Performance may be reduced. ETW requires administrator privileges.\n")
		return t, fmt.Errorf("ETW session creation failed, falling back to polling")
	}

	t.session = session

	// Try to create ETW consumer
	consumer := etw.NewConsumer(ctx)
	if consumer == nil {
		// ETW consumer creation failed - set fallback mode
		t.fallbackMode = true
		// Log warning notification to user
		fmt.Printf("⚠️  ETW consumer creation failed, falling back to polling mode\n")
		fmt.Printf("   Performance may be reduced. ETW requires administrator privileges.\n")
		return t, fmt.Errorf("ETW consumer creation failed, falling back to polling")
	}

	t.consumer = consumer
	return t, nil
}

// Start begins consuming ETW events (non-blocking).
func (t *ETWTracker) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("consumer already running")
	}

	if t.fallbackMode {
		// In fallback mode, don't start ETW - polling will be used instead
		t.running = true
		return nil
	}

	// Enable providers on the session
	err := t.session.EnableProvider(etw.MustParseProvider(Win32kProviderGUID))
	if err != nil {
		t.fallbackMode = true
		fmt.Printf("⚠️  Failed to enable Win32k provider, falling back to polling: %v\n", err)
		return fmt.Errorf("failed to enable Win32k provider, falling back to polling: %w", err)
	}

	err = t.session.EnableProvider(etw.MustParseProvider(KernelProcessProviderGUID))
	if err != nil {
		// Log warning but continue - we can work with just Win32k
		fmt.Printf("Warning: failed to enable Kernel-Process provider: %v\n", err)
	}

	// Configure consumer to use our session
	t.consumer.FromSessions(t.session)

	// Set up event processing
	t.consumer.ProcessEvents(t.handleETWEvent)

	// Start ETW consumer in background goroutine
	go func() {
		err := t.consumer.Start()
		if err != nil {
			t.mu.Lock()
			t.fallbackMode = true
			t.mu.Unlock()
			fmt.Printf("⚠️  ETW consumer failed to start, falling back to polling: %v\n", err)
		}
	}()

	t.running = true
	return nil
}

// handleETWEvent processes incoming ETW events.
func (t *ETWTracker) handleETWEvent(e *etw.Event) {
	defer e.Release()

	// Parse event based on provider GUID
	providerGUID := e.System.Provider.Guid.String()

	switch providerGUID {
	case Win32kProviderGUID:
		// Window focus event
		t.handleFocusEvent(e)
	case KernelProcessProviderGUID:
		// Process lifecycle event
		t.handleProcessEvent(e)
	}
}

// handleFocusEvent converts ETW focus event to capture.FocusEvent.
func (t *ETWTracker) handleFocusEvent(e *etw.Event) {
	// Extract window handle and process info from ETW event
	// This is a simplified implementation - real ETW parsing would be more complex

	focusEvent := capture.FocusEvent{
		Timestamp:    e.System.TimeCreated.SystemTime,
		WindowHandle: 0, // Would extract from event data
		ProcessID:    e.System.Execution.ProcessID,
		ProcessName:  "", // Would extract from event data
	}

	t.sendFocusEvent(focusEvent)
}

// handleProcessEvent converts ETW process event to capture.ProcessEvent.
func (t *ETWTracker) handleProcessEvent(e *etw.Event) {
	// Extract process info from ETW event
	// This is a simplified implementation - real ETW parsing would be more complex

	eventType := capture.ProcessCreated
	if e.System.Opcode.Name == "Stop" {
		eventType = capture.ProcessTerminated
	}

	processEvent := capture.ProcessEvent{
		Timestamp:   e.System.TimeCreated.SystemTime,
		ProcessID:   e.System.Execution.ProcessID,
		ProcessName: "", // Would extract from event data
		EventType:   eventType,
	}

	t.sendProcessEvent(processEvent)
}

// FocusEvents returns channel of window focus change events.
func (t *ETWTracker) FocusEvents() <-chan capture.FocusEvent {
	return t.focusEvents
}

// ProcessEvents returns channel of process lifecycle events.
func (t *ETWTracker) ProcessEvents() <-chan capture.ProcessEvent {
	return t.processEvents
}

// IsFallbackMode returns true if ETW failed and polling is active.
func (t *ETWTracker) IsFallbackMode() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.fallbackMode
}

// DroppedEvents returns count of events dropped due to backpressure.
func (t *ETWTracker) DroppedEvents() int64 {
	return t.droppedEvents.Load()
}

// sendFocusEvent sends a focus event, dropping oldest if buffer is full.
func (t *ETWTracker) sendFocusEvent(event capture.FocusEvent) {
	select {
	case t.focusEvents <- event:
		// Event sent successfully
	default:
		// Buffer is full - drop oldest event and add new one
		select {
		case <-t.focusEvents:
			t.droppedEvents.Add(1)
		default:
		}

		select {
		case t.focusEvents <- event:
		default:
			// Still couldn't send - increment dropped counter
			t.droppedEvents.Add(1)
		}
	}
}

// sendProcessEvent sends a process event, dropping oldest if buffer is full.
func (t *ETWTracker) sendProcessEvent(event capture.ProcessEvent) {
	select {
	case t.processEvents <- event:
		// Event sent successfully
	default:
		// Buffer is full - drop oldest event and add new one
		select {
		case <-t.processEvents:
			t.droppedEvents.Add(1)
		default:
		}

		select {
		case t.processEvents <- event:
		default:
			// Still couldn't send - increment dropped counter
			t.droppedEvents.Add(1)
		}
	}
}

// Close stops ETW session and cleans up.
func (t *ETWTracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	// Cancel context to stop ETW consumer
	if t.cancel != nil {
		t.cancel()
	}

	// Stop ETW consumer
	if t.consumer != nil && !t.fallbackMode {
		t.consumer.Stop()
	}

	// Stop ETW session
	if t.session != nil && !t.fallbackMode {
		t.session.Stop()
	}

	// Close channels
	close(t.focusEvents)
	close(t.processEvents)

	return nil
}
