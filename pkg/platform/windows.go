//go:build windows

package platform

import (
	"context"
	"path/filepath"

	"waddle/pkg/capture"
	capwin "waddle/pkg/capture/windows"
	"waddle/pkg/infra/config"
)

// ── WindowsTracker ──────────────────────────────────────────────────

// WindowsTracker wraps the ETW consumer behind the WindowTracker interface.
type WindowsTracker struct {
	consumer *capwin.ETWTracker
	fEvents  chan FocusEvent
	pEvents  chan ProcessEvent
}

// NewWindowTracker creates a standalone WindowTracker (backward compat).
func NewWindowTracker() (WindowTracker, error) {
	consumer, err := capwin.NewETWTracker()
	if err != nil && consumer == nil {
		return nil, err
	}

	wt := &WindowsTracker{
		consumer: consumer,
		fEvents:  make(chan FocusEvent, capture.EventBufferSize),
		pEvents:  make(chan ProcessEvent, capture.EventBufferSize),
	}

	return wt, nil
}

func (w *WindowsTracker) Start(ctx context.Context) error {
	if err := w.consumer.Start(); err != nil {
		return err
	}
	// Bridge goroutines start only after the consumer is running
	go w.bridgeFocusEvents()
	go w.bridgeProcessEvents()
	return nil
}

func (w *WindowsTracker) Stop() error {
	return w.consumer.Close()
}

func (w *WindowsTracker) FocusEvents() <-chan FocusEvent    { return w.fEvents }
func (w *WindowsTracker) ProcessEvents() <-chan ProcessEvent { return w.pEvents }
func (w *WindowsTracker) IsFallbackMode() bool               { return w.consumer.IsFallbackMode() }
func (w *WindowsTracker) DroppedEvents() int64               { return w.consumer.DroppedEvents() }

func (w *WindowsTracker) bridgeFocusEvents() {
	for e := range w.consumer.FocusEvents() {
		w.fEvents <- FocusEvent{
			Timestamp:    e.Timestamp.UnixNano(),
			WindowHandle: e.WindowHandle,
			ProcessID:    e.ProcessID,
			ProcessName:  e.ProcessName,
		}
	}
}

func (w *WindowsTracker) bridgeProcessEvents() {
	for e := range w.consumer.ProcessEvents() {
		var eventType ProcessEventType
		if e.EventType == capture.ProcessCreated {
			eventType = ProcessStart
		} else {
			eventType = ProcessStop
		}

		w.pEvents <- ProcessEvent{
			Timestamp:   e.Timestamp.UnixNano(),
			ProcessID:   e.ProcessID,
			ProcessName: e.ProcessName,
			Type:        eventType,
		}
	}
}

// ── Full Platform composite ─────────────────────────────────────────

// windowsPlatform composes all 4 sub-implementations into the Platform interface.
type windowsPlatform struct {
	tracker *WindowsTracker
	uia     *windowsUIReader
	screen  *windowsScreenCapturer
	secrets *windowsSecretStore
}

// NewPlatform creates a fully-composed Platform for Windows.
func NewPlatform(cfg *config.Config) (Platform, error) {
	// 1. Window tracker (ETW)
	tracker, err := NewWindowTracker()
	if err != nil {
		return nil, err
	}

	// 2. UI reader (UIA marshaler + reader)
	uiaReader, err := newWindowsUIReader()
	if err != nil {
		return nil, err
	}

	// 3. Screen capturer
	screen := &windowsScreenCapturer{}

	// 4. Secret store (DPAPI vault)
	secrets := newWindowsSecretStore(filepath.Join(cfg.DataDir, "secrets"))

	return &windowsPlatform{
		tracker: tracker.(*WindowsTracker),
		uia:     uiaReader,
		screen:  screen,
		secrets: secrets,
	}, nil
}

// ── WindowTracker delegation ────────────────────────────────────────

func (p *windowsPlatform) Start(ctx context.Context) error   { return p.tracker.Start(ctx) }
func (p *windowsPlatform) FocusEvents() <-chan FocusEvent     { return p.tracker.FocusEvents() }
func (p *windowsPlatform) ProcessEvents() <-chan ProcessEvent { return p.tracker.ProcessEvents() }
func (p *windowsPlatform) IsFallbackMode() bool               { return p.tracker.IsFallbackMode() }
func (p *windowsPlatform) DroppedEvents() int64               { return p.tracker.DroppedEvents() }

func (p *windowsPlatform) Stop() error {
	// Stop tracker first, then close UIA (which tears down the STA thread)
	err := p.tracker.Stop()
	if p.uia != nil {
		_ = p.uia.Close()
	}
	return err
}

// ── UIReader delegation ─────────────────────────────────────────────

func (p *windowsPlatform) GetStructuredData(hwnd uintptr) (*UIResult, error) {
	return p.uia.GetStructuredData(hwnd)
}

func (p *windowsPlatform) Close() error {
	return p.uia.Close()
}

// ── ScreenCapturer delegation ───────────────────────────────────────

func (p *windowsPlatform) CaptureWindow(hwnd uintptr) ([]byte, error) {
	return p.screen.CaptureWindow(hwnd)
}

func (p *windowsPlatform) CaptureScreen() ([]byte, error) {
	return p.screen.CaptureScreen()
}

// ── SecretStore delegation ──────────────────────────────────────────

func (p *windowsPlatform) Save(key string, data []byte) error {
	return p.secrets.Save(key, data)
}

func (p *windowsPlatform) Load(key string) ([]byte, error) {
	return p.secrets.Load(key)
}

func (p *windowsPlatform) Delete(key string) error {
	return p.secrets.Delete(key)
}
