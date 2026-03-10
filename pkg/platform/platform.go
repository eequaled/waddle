package platform

import (
	"context"
	"errors"
)

// Portable event types
type FocusEvent struct {
	Timestamp    int64
	WindowHandle uintptr
	ProcessID    uint32
	ProcessName  string
}

type ProcessEvent struct {
	Timestamp   int64
	ProcessID   uint32
	ProcessName string
	Type        ProcessEventType
}

type ProcessEventType int

const (
	ProcessStart ProcessEventType = iota
	ProcessStop
)

// Platform Interfaces

type WindowTracker interface {
	Start(ctx context.Context) error
	Stop() error
	FocusEvents() <-chan FocusEvent
	ProcessEvents() <-chan ProcessEvent
	IsFallbackMode() bool
	DroppedEvents() int64
}

type UIReader interface {
	GetWindowInfo(hwnd uintptr) (map[string]interface{}, error)
}

type ScreenCapturer interface {
	CaptureWindow(hwnd uintptr) ([]byte, error)
}

type InputSimulator interface {
	Click(x, y int) error
	TypeText(text string) error
}

type SecretStore interface {
	Save(key, value string) error
	Load(key string) (string, error)
	Delete(key string) error
}

var ErrNotImplemented = errors.New("not implemented on this platform")
