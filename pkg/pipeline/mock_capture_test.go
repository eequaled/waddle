package pipeline

import (
	"context"
	"sync/atomic"

	"waddle/pkg/capture"
)

// MockCaptureEngine implements capture.CaptureEngine for testing purposes.
type MockCaptureEngine struct {
	focusEvents   chan capture.FocusEvent
	processEvents chan capture.ProcessEvent
	droppedEvents atomic.Int64
	fallbackMode  bool

	// Mocks for method calls
	GetWindowInfoFn func(hwnd uintptr) (*capture.WindowInfo, error)
	CaptureWindowFn func(hwnd uintptr) ([]byte, error)
	CaptureScreenFn func() ([]byte, error)
}

func NewMockCaptureEngine() *MockCaptureEngine {
	return &MockCaptureEngine{
		focusEvents:   make(chan capture.FocusEvent, 10),
		processEvents: make(chan capture.ProcessEvent, 10),
	}
}

func (m *MockCaptureEngine) Start(ctx context.Context) error {
	return nil
}

func (m *MockCaptureEngine) Stop() error {
	close(m.focusEvents)
	close(m.processEvents)
	return nil
}

func (m *MockCaptureEngine) FocusEvents() <-chan capture.FocusEvent {
	return m.focusEvents
}

func (m *MockCaptureEngine) ProcessEvents() <-chan capture.ProcessEvent {
	return m.processEvents
}

func (m *MockCaptureEngine) GetWindowInfo(hwnd uintptr) (*capture.WindowInfo, error) {
	if m.GetWindowInfoFn != nil {
		return m.GetWindowInfoFn(hwnd)
	}
	return &capture.WindowInfo{HWND: hwnd}, nil
}

func (m *MockCaptureEngine) CaptureWindow(hwnd uintptr) ([]byte, error) {
	if m.CaptureWindowFn != nil {
		return m.CaptureWindowFn(hwnd)
	}
	return []byte("mock_png_bytes"), nil
}

func (m *MockCaptureEngine) CaptureScreen() ([]byte, error) {
	if m.CaptureScreenFn != nil {
		return m.CaptureScreenFn()
	}
	return []byte("mock_screen_png_bytes"), nil
}

func (m *MockCaptureEngine) IsFallbackMode() bool {
	return m.fallbackMode
}

func (m *MockCaptureEngine) DroppedEvents() int64 {
	return m.droppedEvents.Load()
}
