//go:build !windows

package platform

import (
	"context"
)

type StubTracker struct {
	fEvents chan FocusEvent
	pEvents chan ProcessEvent
}

func NewWindowTracker() (WindowTracker, error) {
	return &StubTracker{
		fEvents: make(chan FocusEvent),
		pEvents: make(chan ProcessEvent),
	}, nil
}

func (s *StubTracker) Start(ctx context.Context) error {
	return ErrNotImplemented
}

func (s *StubTracker) Stop() error {
	return nil
}

func (s *StubTracker) FocusEvents() <-chan FocusEvent {
	return s.fEvents
}

func (s *StubTracker) ProcessEvents() <-chan ProcessEvent {
	return s.pEvents
}

func (s *StubTracker) IsFallbackMode() bool {
	return true
}

func (s *StubTracker) DroppedEvents() int64 {
	return 0
}
