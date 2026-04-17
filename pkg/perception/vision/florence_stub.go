//go:build !onnx

package vision

import (
	"context"
	"fmt"
)

// FlorenceEngine is a stub for environments without ONNX Runtime.
// Build with -tags onnx to enable the real implementation.
type FlorenceEngine struct{}

// UIElement represents a detected UI element with label, confidence, and bounding box.
type UIElement struct {
	Label      string  `json:"label"`
	Confidence float32 `json:"confidence"`
	BBox       [4]int  `json:"bbox"` // [x1, y1, x2, y2]
}

// ErrONNXNotAvailable is returned by all stub methods when ONNX Runtime is not compiled in.
var ErrONNXNotAvailable = fmt.Errorf("florence: ONNX Runtime not available; build with -tags onnx")

// NewFlorenceEngine returns an error in stub mode — ONNX Runtime is not compiled in.
func NewFlorenceEngine(modelPath string) (*FlorenceEngine, error) {
	return nil, ErrONNXNotAvailable
}

// DetectUIElements is a stub — returns ErrONNXNotAvailable.
func (e *FlorenceEngine) DetectUIElements(ctx context.Context, imageData []byte) ([]UIElement, error) {
	return nil, ErrONNXNotAvailable
}

// Close is a no-op in stub mode.
func (e *FlorenceEngine) Close() error {
	return nil
}
