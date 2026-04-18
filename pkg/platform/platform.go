package platform

import (
	"context"
	"errors"
)

// ErrNotImplemented is returned by stub implementations on unsupported platforms.
var ErrNotImplemented = errors.New("not implemented on this platform")

// ── Portable event types ────────────────────────────────────────────

// FocusEvent is emitted when the user switches to a different window.
type FocusEvent struct {
	Timestamp    int64
	WindowHandle uintptr
	ProcessID    uint32
	ProcessName  string
}

// ProcessEvent is emitted when a process starts or stops.
type ProcessEvent struct {
	Timestamp   int64
	ProcessID   uint32
	ProcessName string
	Type        ProcessEventType
}

// ProcessEventType distinguishes process start from stop.
type ProcessEventType int

const (
	ProcessStart ProcessEventType = iota
	ProcessStop
)

// ── WindowInfo ──────────────────────────────────────────────────────

// WindowInfo is the portable representation of window information extracted
// via UI Automation. Fields mirror uia.WindowInfo but use only stdlib types
// so consumers never import the uia package directly.
type WindowInfo struct {
	HWND        uintptr
	ProcessID   uint32
	ProcessName string
	WindowTitle string
	AppType     string                 // "vscode", "chrome", "edge", "slack", "unknown"
	Metadata    map[string]interface{} // app-specific key/value pairs
}

// ── Sub-interfaces ──────────────────────────────────────────────────

// WindowTracker provides window focus and process lifecycle events.
type WindowTracker interface {
	Start(ctx context.Context) error
	Stop() error
	FocusEvents() <-chan FocusEvent
	ProcessEvents() <-chan ProcessEvent
	IsFallbackMode() bool
	DroppedEvents() int64
}

// UIReader extracts structured data from a window via accessibility APIs.
type UIReader interface {
	GetStructuredData(hwnd uintptr) (*WindowInfo, error)
	Close() error
}

// ScreenCapturer captures screenshots of a window or the full desktop.
type ScreenCapturer interface {
	CaptureWindow(hwnd uintptr) ([]byte, error) // PNG bytes of a specific window
	CaptureScreen() ([]byte, error)             // PNG bytes of full desktop
}

// InputSimulator sends synthetic mouse/keyboard input.
type InputSimulator interface {
	Click(x, y int) error
	TypeText(text string) error
}

// SecretStore provides platform-specific secure storage (DPAPI, libsecret, etc.).
type SecretStore interface {
	Save(key string, data []byte) error
	Load(key string) ([]byte, error)
	Delete(key string) error
}

// ── Composite ───────────────────────────────────────────────────────

// Platform is the full OS abstraction. It embeds all sub-interfaces
// so callers can depend on one value for the entire OS surface.
type Platform interface {
	WindowTracker
	UIReader
	ScreenCapturer
	SecretStore
}
