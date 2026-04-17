package vision

import (
	"context"
	"testing"
)

// TestStubReturnsError verifies the stub implementation returns the expected error.
func TestStubReturnsError(t *testing.T) {
	t.Run("NewFlorenceEngine returns error without ONNX", func(t *testing.T) {
		engine, err := NewFlorenceEngine("/fake/model/path")
		if err == nil {
			t.Fatal("Expected error from stub NewFlorenceEngine, got nil")
		}
		if engine != nil {
			t.Fatal("Expected nil engine from stub, got non-nil")
		}
		if err != ErrONNXNotAvailable {
			t.Errorf("Expected ErrONNXNotAvailable, got: %v", err)
		}
	})

	t.Run("DetectUIElements returns error without ONNX", func(t *testing.T) {
		// Even though NewFlorenceEngine fails, test DetectUIElements on a zero-value engine
		engine := &FlorenceEngine{}
		results, err := engine.DetectUIElements(context.Background(), []byte("fake image data"))
		if err == nil {
			t.Fatal("Expected error from stub DetectUIElements, got nil")
		}
		if results != nil {
			t.Fatal("Expected nil results from stub, got non-nil")
		}
		if err != ErrONNXNotAvailable {
			t.Errorf("Expected ErrONNXNotAvailable, got: %v", err)
		}
	})

	t.Run("Close is safe on stub", func(t *testing.T) {
		engine := &FlorenceEngine{}
		if err := engine.Close(); err != nil {
			t.Errorf("Expected Close to succeed on stub, got: %v", err)
		}
	})
}

// TestUIElementStruct verifies UIElement struct fields are accessible.
func TestUIElementStruct(t *testing.T) {
	elem := UIElement{
		Label:      "button",
		Confidence: 0.95,
		BBox:       [4]int{10, 20, 100, 50},
	}

	if elem.Label != "button" {
		t.Errorf("Expected label 'button', got '%s'", elem.Label)
	}
	if elem.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", elem.Confidence)
	}
	if elem.BBox != [4]int{10, 20, 100, 50} {
		t.Errorf("Expected bbox [10, 20, 100, 50], got %v", elem.BBox)
	}
}
