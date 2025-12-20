package etw

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestETWFallbackToPolling tests Property 1: ETW Fallback to Polling
// For any ETW initialization failure, the system should activate polling mode and set IsFallbackMode() to true.
// Validates: Requirements 1.7
func TestETWFallbackToPolling(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 1: ETW Fallback to Polling
	properties.Property("For any ETW initialization failure, system should activate polling mode", prop.ForAll(
		func(dummy bool) bool {
			// Create consumer - this may fail due to ETW not being available or insufficient privileges
			consumer, err := NewConsumer()
			
			// If ETW initialization fails, consumer should be in fallback mode
			if err != nil {
				// ETW failed - consumer should be in fallback mode
				if !consumer.IsFallbackMode() {
					t.Logf("ETW failed but fallback mode not set: %v", err)
					return false
				}
				
				// Consumer should still be usable
				if consumer == nil {
					t.Logf("Consumer is nil after ETW failure")
					return false
				}
				
				// Channels should be available even in fallback mode
				if consumer.FocusEvents() == nil {
					t.Logf("FocusEvents channel is nil in fallback mode")
					return false
				}
				
				if consumer.ProcessEvents() == nil {
					t.Logf("ProcessEvents channel is nil in fallback mode")
					return false
				}
				
				return true
			}
			
			// ETW succeeded - consumer should not be in fallback mode
			if consumer.IsFallbackMode() {
				t.Logf("ETW succeeded but consumer is in fallback mode")
				return false
			}
			
			// Clean up
			consumer.Close()
			return true
		},
		gen.Bool(), // Generate a dummy boolean parameter
	))

	properties.TestingRun(t)
}

// TestETWConsumerBasicFunctionality tests basic consumer functionality
func TestETWConsumerBasicFunctionality(t *testing.T) {
	consumer, err := NewConsumer()
	if err != nil {
		// ETW failed - this is expected on systems without admin privileges
		t.Logf("ETW initialization failed (expected on non-admin): %v", err)
		
		// Verify fallback mode is set
		if !consumer.IsFallbackMode() {
			t.Errorf("ETW failed but fallback mode not set")
		}
		return
	}
	
	defer consumer.Close()
	
	// Test basic functionality
	if consumer.IsFallbackMode() {
		t.Errorf("ETW succeeded but consumer reports fallback mode")
	}
	
	if consumer.FocusEvents() == nil {
		t.Errorf("FocusEvents channel is nil")
	}
	
	if consumer.ProcessEvents() == nil {
		t.Errorf("ProcessEvents channel is nil")
	}
	
	if consumer.DroppedEvents() != 0 {
		t.Errorf("DroppedEvents should be 0 initially, got %d", consumer.DroppedEvents())
	}
}

// TestETWConsumerStartStop tests consumer start/stop functionality
func TestETWConsumerStartStop(t *testing.T) {
	consumer, err := NewConsumer()
	if err != nil {
		t.Logf("ETW initialization failed (expected on non-admin): %v", err)
		// Still test start/stop in fallback mode
	}
	
	defer consumer.Close()
	
	// Test start
	err = consumer.Start()
	if err != nil && !consumer.IsFallbackMode() {
		t.Errorf("Start failed in non-fallback mode: %v", err)
	}
	
	// Test double start
	err = consumer.Start()
	if err == nil && !consumer.IsFallbackMode() {
		t.Errorf("Double start should return error in non-fallback mode")
	}
	
	// Test close
	err = consumer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestETWEventBuffering tests event buffering and backpressure
func TestETWEventBuffering(t *testing.T) {
	consumer, err := NewConsumer()
	if err != nil {
		t.Logf("ETW initialization failed (expected on non-admin): %v", err)
	}
	
	defer consumer.Close()
	
	// Test that channels have correct buffer size
	focusChan := consumer.FocusEvents()
	if cap(focusChan) != EventBufferSize {
		t.Errorf("FocusEvents channel buffer size should be %d, got %d", EventBufferSize, cap(focusChan))
	}
	
	processChan := consumer.ProcessEvents()
	if cap(processChan) != EventBufferSize {
		t.Errorf("ProcessEvents channel buffer size should be %d, got %d", EventBufferSize, cap(processChan))
	}
}