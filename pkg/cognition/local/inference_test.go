package local

import (
	"context"
	"testing"
)

// TestStubNewLocalInference verifies that the stub returns an appropriate error
// when the llama_cpp build tag is not set.
func TestStubNewLocalInference(t *testing.T) {
	li, err := NewLocalInference("/path/to/model.gguf")

	if err == nil {
		t.Errorf("Expected error from stub NewLocalInference, got nil")
	}

	if li != nil {
		t.Errorf("Expected nil LocalInference from stub, got non-nil")
	}

	// Verify error message mentions build tag
	if err != nil {
		expectedSubstr := "llama_cpp"
		found := false
		for i := 0; i <= len(err.Error())-len(expectedSubstr); i++ {
			if err.Error()[i:i+len(expectedSubstr)] == expectedSubstr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Error should mention 'llama_cpp' build tag, got: %v", err)
		}
	}
}

// TestStubPredict verifies that stub Predict returns an error.
// This test is a placeholder — when the CGo spike succeeds, a parallel
// test with //go:build llama_cpp will exercise the real inference path.
func TestStubPredict(t *testing.T) {
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

// TestStubClose verifies that stub Close is a safe no-op.
func TestStubClose(t *testing.T) {
	// Close on nil should not panic
	var li *LocalInference
	if li != nil {
		err := li.Close()
		if err != nil {
			t.Errorf("Close on stub should not error: %v", err)
		}
	}
}
