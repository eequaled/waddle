package platform

import (
	"testing"
)

func TestTrackerInterfaceCompliance(t *testing.T) {
	wt, err := NewWindowTracker()
	if err != nil {
		t.Logf("Tracker creation info: %v", err)
		// It's okay if creation fails for some reasons (like permissions), but wt should either be nil or a working entity
	}
	
	if wt != nil {
		if wt.FocusEvents() == nil {
			t.Errorf("Tracker.FocusEvents() should not be nil")
		}
		if wt.ProcessEvents() == nil {
			t.Errorf("Tracker.ProcessEvents() should not be nil")
		}
	}
}

func TestNewWindowTracker(t *testing.T) {
	wt, err := NewWindowTracker()
	if err != nil {
		t.Logf("NewWindowTracker info: %v", err)
	}
	
	if wt == nil && err == nil {
		t.Errorf("NewWindowTracker returned both nil wt and nil err")
	}
}

