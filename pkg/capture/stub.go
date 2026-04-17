//go:build !windows

package capture

import "context"

// StubCaptureEngine provides a no-op CaptureEngine for non-Windows platforms.
type StubCaptureEngine struct {
	fEvents chan FocusEvent
	pEvents chan ProcessEvent
}

// NewStubCaptureEngine creates a stub capture engine that returns errors.
func NewStubCaptureEngine() *StubCaptureEngine {
	return &StubCaptureEngine{
		fEvents: make(chan FocusEvent),
		pEvents: make(chan ProcessEvent),
	}
}

func (s *StubCaptureEngine) Start(ctx context.Context) error          { return nil }
func (s *StubCaptureEngine) Stop() error                               { return nil }
func (s *StubCaptureEngine) FocusEvents() <-chan FocusEvent            { return s.fEvents }
func (s *StubCaptureEngine) ProcessEvents() <-chan ProcessEvent        { return s.pEvents }
func (s *StubCaptureEngine) GetWindowInfo(hwnd uintptr) (*WindowInfo, error) { return nil, nil }
func (s *StubCaptureEngine) CaptureWindow(hwnd uintptr) ([]byte, error) { return nil, nil }
func (s *StubCaptureEngine) CaptureScreen() ([]byte, error)            { return nil, nil }
func (s *StubCaptureEngine) IsFallbackMode() bool                      { return true }
func (s *StubCaptureEngine) DroppedEvents() int64                      { return 0 }
