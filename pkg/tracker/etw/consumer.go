package etw

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tekert/golang-etw/etw"
)

// Consumer subscribes to Windows kernel events for zero-overhead capture
type Consumer struct {
	session       *etw.RealTimeSession
	consumer      *etw.Consumer
	ctx           context.Context
	cancel        context.CancelFunc
	focusEvents   chan FocusEvent
	processEvents chan ProcessEvent
	fallbackMode  bool
	droppedEvents atomic.Int64
	mu            sync.RWMutex
	running       bool
}

// FocusEvent represents a window focus change event
type FocusEvent struct {
	Timestamp    time.Time
	WindowHandle uintptr
	ProcessID    uint32
	ProcessName  string
}

// ProcessEvent represents a process lifecycle event
type ProcessEvent struct {
	Timestamp   time.Time
	ProcessID   uint32
	ProcessName string
	EventType   ProcessEventType
}

// ProcessEventType represents the type of process event
type ProcessEventType int

const (
	ProcessCreated ProcessEventType = iota
	ProcessTerminated
)

const (
	// Event buffer size - when full, oldest events are dropped
	EventBufferSize = 1000
	
	// ETW Provider GUIDs
	Win32kProviderGUID         = "{8c416c79-d49b-4f01-a467-e56d3aa8234c}" // Microsoft-Windows-Win32k
	KernelProcessProviderGUID  = "{22fb2cd6-0e7b-422b-a0c7-2fad1fd0e716}" // Microsoft-Windows-Kernel-Process
)

// NewConsumer creates ETW consumer subscribed to Win32k and Kernel-Process providers
func NewConsumer() (*Consumer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	c := &Consumer{
		ctx:           ctx,
		cancel:        cancel,
		focusEvents:   make(chan FocusEvent, EventBufferSize),
		processEvents: make(chan ProcessEvent, EventBufferSize),
		fallbackMode:  false,
	}

	// Try to create ETW session
	session := etw.NewRealTimeSession("WaddleETWSession")
	if session == nil {
		// ETW initialization failed - set fallback mode
		c.fallbackMode = true
		// Log warning notification to user
		fmt.Printf("⚠️  ETW session creation failed, falling back to polling mode\n")
		fmt.Printf("   Performance may be reduced. ETW requires administrator privileges.\n")
		return c, fmt.Errorf("ETW session creation failed, falling back to polling")
	}

	c.session = session

	// Try to create ETW consumer
	consumer := etw.NewConsumer(ctx)
	if consumer == nil {
		// ETW consumer creation failed - set fallback mode
		c.fallbackMode = true
		// Log warning notification to user
		fmt.Printf("⚠️  ETW consumer creation failed, falling back to polling mode\n")
		fmt.Printf("   Performance may be reduced. ETW requires administrator privileges.\n")
		return c, fmt.Errorf("ETW consumer creation failed, falling back to polling")
	}

	c.consumer = consumer
	return c, nil
}

// Start begins consuming ETW events (non-blocking)
func (c *Consumer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("consumer already running")
	}

	if c.fallbackMode {
		// In fallback mode, don't start ETW - polling will be used instead
		c.running = true
		return nil
	}

	// Enable providers on the session
	err := c.session.EnableProvider(etw.MustParseProvider(Win32kProviderGUID))
	if err != nil {
		c.fallbackMode = true
		fmt.Printf("⚠️  Failed to enable Win32k provider, falling back to polling: %v\n", err)
		return fmt.Errorf("failed to enable Win32k provider, falling back to polling: %w", err)
	}

	err = c.session.EnableProvider(etw.MustParseProvider(KernelProcessProviderGUID))
	if err != nil {
		// Log warning but continue - we can work with just Win32k
		fmt.Printf("Warning: failed to enable Kernel-Process provider: %v\n", err)
	}

	// Configure consumer to use our session
	c.consumer.FromSessions(c.session)

	// Set up event processing
	c.consumer.ProcessEvents(c.handleETWEvent)

	// Start ETW consumer in background goroutine
	go func() {
		err := c.consumer.Start()
		if err != nil {
			c.mu.Lock()
			c.fallbackMode = true
			c.mu.Unlock()
			fmt.Printf("⚠️  ETW consumer failed to start, falling back to polling: %v\n", err)
		}
	}()

	c.running = true
	return nil
}

// handleETWEvent processes incoming ETW events
func (c *Consumer) handleETWEvent(e *etw.Event) {
	defer e.Release()

	// Parse event based on provider GUID
	providerGUID := e.System.Provider.Guid.String()
	
	switch providerGUID {
	case Win32kProviderGUID:
		// Window focus event
		c.handleFocusEvent(e)
	case KernelProcessProviderGUID:
		// Process lifecycle event
		c.handleProcessEvent(e)
	}
}

// handleFocusEvent converts ETW focus event to FocusEvent
func (c *Consumer) handleFocusEvent(e *etw.Event) {
	// Extract window handle and process info from ETW event
	// This is a simplified implementation - real ETW parsing would be more complex
	
	focusEvent := FocusEvent{
		Timestamp:    e.System.TimeCreated.SystemTime,
		WindowHandle: 0, // Would extract from event data
		ProcessID:    e.System.Execution.ProcessID,
		ProcessName:  "", // Would extract from event data
	}
	
	c.sendFocusEvent(focusEvent)
}

// handleProcessEvent converts ETW process event to ProcessEvent
func (c *Consumer) handleProcessEvent(e *etw.Event) {
	// Extract process info from ETW event
	// This is a simplified implementation - real ETW parsing would be more complex
	
	eventType := ProcessCreated
	if e.System.Opcode.Name == "Stop" {
		eventType = ProcessTerminated
	}
	
	processEvent := ProcessEvent{
		Timestamp:   e.System.TimeCreated.SystemTime,
		ProcessID:   e.System.Execution.ProcessID,
		ProcessName: "", // Would extract from event data
		EventType:   eventType,
	}
	
	c.sendProcessEvent(processEvent)
}

// FocusEvents returns channel of window focus change events
func (c *Consumer) FocusEvents() <-chan FocusEvent {
	return c.focusEvents
}

// ProcessEvents returns channel of process lifecycle events
func (c *Consumer) ProcessEvents() <-chan ProcessEvent {
	return c.processEvents
}

// IsFallbackMode returns true if ETW failed and polling is active
func (c *Consumer) IsFallbackMode() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fallbackMode
}

// DroppedEvents returns count of events dropped due to backpressure
func (c *Consumer) DroppedEvents() int64 {
	return c.droppedEvents.Load()
}

// sendFocusEvent sends a focus event, dropping oldest if buffer is full
func (c *Consumer) sendFocusEvent(event FocusEvent) {
	select {
	case c.focusEvents <- event:
		// Event sent successfully
	default:
		// Buffer is full - drop oldest event and add new one
		select {
		case <-c.focusEvents:
			c.droppedEvents.Add(1)
		default:
		}
		
		select {
		case c.focusEvents <- event:
		default:
			// Still couldn't send - increment dropped counter
			c.droppedEvents.Add(1)
		}
	}
}

// sendProcessEvent sends a process event, dropping oldest if buffer is full
func (c *Consumer) sendProcessEvent(event ProcessEvent) {
	select {
	case c.processEvents <- event:
		// Event sent successfully
	default:
		// Buffer is full - drop oldest event and add new one
		select {
		case <-c.processEvents:
			c.droppedEvents.Add(1)
		default:
		}
		
		select {
		case c.processEvents <- event:
		default:
			// Still couldn't send - increment dropped counter
			c.droppedEvents.Add(1)
		}
	}
}

// Close stops ETW session and cleans up
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.running = false

	// Cancel context to stop ETW consumer
	if c.cancel != nil {
		c.cancel()
	}

	// Stop ETW consumer
	if c.consumer != nil && !c.fallbackMode {
		c.consumer.Stop()
	}

	// Stop ETW session
	if c.session != nil && !c.fallbackMode {
		c.session.Stop()
	}

	// Close channels
	close(c.focusEvents)
	close(c.processEvents)

	return nil
}