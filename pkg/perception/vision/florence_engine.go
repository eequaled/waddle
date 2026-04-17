//go:build onnx

package vision

import (
	"context"
	"fmt"
)

// FlorenceEngine provides UI element detection using Florence-2 via ONNX Runtime.
// Build with: go build -tags onnx
type FlorenceEngine struct {
	modelPath string
	// session will hold the ONNX Runtime session once integrated.
	// Placeholder until Engineer C audits the onnxruntime-go dependency.
}

// UIElement represents a detected UI element with label, confidence, and bounding box.
type UIElement struct {
	Label      string  `json:"label"`
	Confidence float32 `json:"confidence"`
	BBox       [4]int  `json:"bbox"` // [x1, y1, x2, y2]
}

// NewFlorenceEngine creates a new FlorenceEngine with the given ONNX model path.
// Returns an error if the model cannot be loaded.
func NewFlorenceEngine(modelPath string) (*FlorenceEngine, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("florence: model path cannot be empty")
	}

	engine := &FlorenceEngine{
		modelPath: modelPath,
	}

	// TODO(spike): Initialize ONNX Runtime session here.
	// Blocked on: Engineer C dependency audit of onnxruntime-go.

	return engine, nil
}

// DetectUIElements runs Florence-2 inference on the provided image data
// and returns detected UI elements with bounding boxes.
func (e *FlorenceEngine) DetectUIElements(ctx context.Context, imageData []byte) ([]UIElement, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("florence: image data cannot be empty")
	}

	// TODO(spike): Implement ONNX inference pipeline:
	// 1. Preprocess image (resize, normalize)
	// 2. Run encoder forward pass
	// 3. Run decoder forward pass with "<OD>" prompt
	// 4. Parse output tokens into UIElement structs

	return nil, fmt.Errorf("florence: ONNX inference not yet implemented; awaiting dependency audit")
}

// Close releases the ONNX Runtime session and frees resources.
func (e *FlorenceEngine) Close() error {
	// TODO(spike): Release ONNX Runtime session.
	return nil
}
