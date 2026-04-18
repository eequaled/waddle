//go:build !windows

package platform

import (
	"context"

	"waddle/pkg/infra/config"
)

// ── StubTracker (unchanged from Week 1) ─────────────────────────────

// StubTracker provides a no-op WindowTracker for non-Windows platforms.
type StubTracker struct {
	fEvents chan FocusEvent
	pEvents chan ProcessEvent
}

// NewWindowTracker returns a stub tracker on non-Windows platforms.
func NewWindowTracker() (WindowTracker, error) {
	return &StubTracker{
		fEvents: make(chan FocusEvent),
		pEvents: make(chan ProcessEvent),
	}, nil
}

func (s *StubTracker) Start(ctx context.Context) error     { return ErrNotImplemented }
func (s *StubTracker) Stop() error                          { return nil }
func (s *StubTracker) FocusEvents() <-chan FocusEvent       { return s.fEvents }
func (s *StubTracker) ProcessEvents() <-chan ProcessEvent   { return s.pEvents }
func (s *StubTracker) IsFallbackMode() bool                 { return true }
func (s *StubTracker) DroppedEvents() int64                 { return 0 }

// ── Full Platform stub ──────────────────────────────────────────────

// stubPlatform returns ErrNotImplemented for all platform operations
// on non-Windows builds. Allows the app to compile and gracefully degrade.
type stubPlatform struct {
	StubTracker
}

// NewPlatform returns a stub platform on non-Windows builds.
func NewPlatform(cfg *config.Config) (Platform, error) {
	return &stubPlatform{
		StubTracker: StubTracker{
			fEvents: make(chan FocusEvent),
			pEvents: make(chan ProcessEvent),
		},
	}, nil
}

// UIReader
func (s *stubPlatform) GetStructuredData(hwnd uintptr) (*WindowInfo, error) {
	return nil, ErrNotImplemented
}
func (s *stubPlatform) Close() error { return nil }

// ScreenCapturer
func (s *stubPlatform) CaptureWindow(hwnd uintptr) ([]byte, error) {
	return nil, ErrNotImplemented
}
func (s *stubPlatform) CaptureScreen() ([]byte, error) {
	return nil, ErrNotImplemented
}

// SecretStore
func (s *stubPlatform) Save(key string, data []byte) error  { return ErrNotImplemented }
func (s *stubPlatform) Load(key string) ([]byte, error)     { return nil, ErrNotImplemented }
func (s *stubPlatform) Delete(key string) error              { return ErrNotImplemented }
