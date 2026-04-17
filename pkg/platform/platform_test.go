package platform

import (
	"testing"

	"waddle/pkg/infra/config"
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

func TestPlatformInterfaceCompliance(t *testing.T) {
	cfg := config.DefaultConfig()
	plat, err := NewPlatform(&cfg)
	if err != nil {
		t.Logf("Platform creation info: %v", err)
	}
	if plat == nil && err == nil {
		t.Errorf("NewPlatform returned both nil plat and nil err")
	}
	if plat != nil {
		if plat.FocusEvents() == nil {
			t.Errorf("Platform.FocusEvents() should not be nil")
		}
		if plat.ProcessEvents() == nil {
			t.Errorf("Platform.ProcessEvents() should not be nil")
		}
	}
}
