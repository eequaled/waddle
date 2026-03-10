//go:build windows

package platform

import (
	"context"
	"waddle/pkg/tracker/etw"
)

type WindowsTracker struct {
	consumer *etw.Consumer
	fEvents  chan FocusEvent
	pEvents  chan ProcessEvent
}

func NewWindowTracker() (WindowTracker, error) {
	consumer, err := etw.NewConsumer()
	if err != nil && consumer == nil {
		return nil, err
	}
	
	wt := &WindowsTracker{
		consumer: consumer,
		fEvents:  make(chan FocusEvent, etw.EventBufferSize),
		pEvents:  make(chan ProcessEvent, etw.EventBufferSize),
	}
	
	return wt, nil
}

func (w *WindowsTracker) Start(ctx context.Context) error {
	if err := w.consumer.Start(); err != nil {
		return err
	}
	// Bridge goroutines are started only after the consumer is running
	go w.bridgeFocusEvents()
	go w.bridgeProcessEvents()
	return nil
}

func (w *WindowsTracker) Stop() error {
	return w.consumer.Close()
}

func (w *WindowsTracker) FocusEvents() <-chan FocusEvent {
	return w.fEvents
}

func (w *WindowsTracker) ProcessEvents() <-chan ProcessEvent {
	return w.pEvents
}

func (w *WindowsTracker) IsFallbackMode() bool {
	return w.consumer.IsFallbackMode()
}

func (w *WindowsTracker) DroppedEvents() int64 {
	return w.consumer.DroppedEvents()
}

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
		if e.EventType == etw.ProcessCreated {
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
