package capture

import (
	"context"
	"time"
)

// ── Portable event types ────────────────────────────────────────────

// FocusEvent represents a window focus change event from ETW or polling.
type FocusEvent struct {
	Timestamp    time.Time
	WindowHandle uintptr
	ProcessID    uint32
	ProcessName  string
}

// ProcessEvent represents a process lifecycle event.
type ProcessEvent struct {
	Timestamp   time.Time
	ProcessID   uint32
	ProcessName string
	EventType   ProcessEventType
}

// ProcessEventType distinguishes process creation from termination.
type ProcessEventType int

const (
	ProcessCreated ProcessEventType = iota
	ProcessTerminated
)

// ── Window information ──────────────────────────────────────────────

// WindowInfo contains extracted window information from UI Automation.
type WindowInfo struct {
	HWND        uintptr
	ProcessID   uint32
	ProcessName string
	WindowTitle string
	AppType     AppType
	Metadata    map[string]interface{}
}

// AppType represents the type of application detected via title/process heuristics.
type AppType int

const (
	AppTypeUnknown AppType = iota
	AppTypeVSCode
	AppTypeChrome
	AppTypeEdge
	AppTypeSlack
)

// String returns the string representation of AppType.
func (a AppType) String() string {
	switch a {
	case AppTypeVSCode:
		return "vscode"
	case AppTypeChrome:
		return "chrome"
	case AppTypeEdge:
		return "edge"
	case AppTypeSlack:
		return "slack"
	default:
		return "unknown"
	}
}

// ── Constants ───────────────────────────────────────────────────────

const (
	// EventBufferSize is the channel capacity for focus/process event streams.
	// When full, the oldest event is dropped (backpressure).
	EventBufferSize = 1000
)

// ── CaptureEngine ───────────────────────────────────────────────────

// CaptureEngine is the unified capture interface combining ETW tracking,
// UI Automation extraction, and screenshot capture.
type CaptureEngine interface {
	// Start begins the capture engine (ETW subscription + STA thread).
	Start(ctx context.Context) error

	// Stop tears down all capture resources.
	Stop() error

	// FocusEvents returns a receive-only channel of window focus changes.
	FocusEvents() <-chan FocusEvent

	// ProcessEvents returns a receive-only channel of process lifecycle events.
	ProcessEvents() <-chan ProcessEvent

	// GetWindowInfo extracts structured window information via UI Automation.
	// Thread-safe: requests are marshaled to the STA thread.
	GetWindowInfo(hwnd uintptr) (*WindowInfo, error)

	// CaptureWindow captures a specific window as PNG bytes.
	CaptureWindow(hwnd uintptr) ([]byte, error)

	// CaptureScreen captures the full primary display as PNG bytes.
	CaptureScreen() ([]byte, error)

	// IsFallbackMode returns true if ETW failed and polling is active.
	IsFallbackMode() bool

	// DroppedEvents returns count of events dropped due to backpressure.
	DroppedEvents() int64
}
