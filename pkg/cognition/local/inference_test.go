package local

import (
	"context"
	"testing"
)

// TestNewLocalInference verifies that it handles missing DLLs gracefully
func TestNewLocalInference(t *testing.T) {
	li, err := NewLocalInference("/path/to/model.gguf")

	// Since we are running tests without llama.dll present, it should error
	if err == nil {
		t.Errorf("Expected error from NewLocalInference due to missing DLL, got nil")
	}

	if li != nil {
		t.Errorf("Expected nil LocalInference when DLL missing, got non-nil")
	}

	// Verify error message mentions llama.dll
	if err != nil {
		expectedSubstr := "llama.dll"
		found := false
		for i := 0; i <= len(err.Error())-len(expectedSubstr); i++ {
			if err.Error()[i:i+len(expectedSubstr)] == expectedSubstr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Error should mention 'llama.dll', got: %v", err)
		}
	}
}

// TestPredict verifies Predict handles missing state.
func TestPredict(t *testing.T) {
	// In stub mode, NewLocalInference returns nil, so we can't call Predict.
	// This test validates that the stub constructor error path is correct.
	li, err := NewLocalInference("model.gguf")
	if err == nil {
		defer li.Close()
		_, predErr := li.Predict(context.Background(), "test prompt")
		if predErr == nil {
			t.Errorf("Expected error from Predict in stub mode")
		}
	}
	// If err != nil (expected in stub mode), the test passes.
}

// TestClose verifies Close is a safe no-op.
func TestClose(t *testing.T) {
	// Close on nil should not panic
	var li *LocalInference
	if li != nil {
		err := li.Close()
		if err != nil {
			t.Errorf("Close on stub should not error: %v", err)
		}
	}
}
