//go:build !onnx

package vision

import (
	"fmt"
	"waddle/pkg/types"
)

// FlorenceEngine is a stub for environments without ONNX Runtime.
// Build with -tags onnx to enable the real implementation.
type FlorenceEngine struct{}

// UIElement is moved to types package.

// ErrONNXNotAvailable is returned by all stub methods when ONNX Runtime is not compiled in.
var ErrONNXNotAvailable = fmt.Errorf("florence: ONNX Runtime not available; build with -tags onnx")

// NewFlorenceEngine returns an error in stub mode — ONNX Runtime is not compiled in.
func NewFlorenceEngine(modelPath string, dllPath string) (*FlorenceEngine, error) {
	return nil, ErrONNXNotAvailable
}

// DetectUIElements is a stub — returns ErrONNXNotAvailable.
func (e *FlorenceEngine) DetectUIElements(imageData []byte) ([]types.UIElement, error) {
	return nil, ErrONNXNotAvailable
}

// Close is a no-op in stub mode.
func (e *FlorenceEngine) Close() error {
	return nil
}
