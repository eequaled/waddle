//go:build onnx

package vision

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"sync"

	// Needed for image formats
	_ "image/jpeg"
	_ "image/png"

	ort "github.com/yalue/onnxruntime_go"
	"waddle/pkg/types"
)

// TODO: Replace with actual tensor names from Florence-2 ONNX export.
// Input names can be found by running: python -c "import onnx; m = onnx.load('model.onnx'); print([i.name for i in m.graph.input])"
const inputName = "TODO_INPUT"   // Expected: "pixel_values"
const outputName = "TODO_OUTPUT" // Expected: "logits" or "pred_boxes"

// FlorenceEngine performs UI element detection using Florence-2 via ONNX Runtime.
// On Windows, it uses DirectML for GPU-agnostic acceleration.
type FlorenceEngine struct {
	modelPath string
	session   *ort.AdvancedSession
	options   *ort.SessionOptions
	mu        sync.Mutex
	closed    bool
}

// UIElement is moved to types package.

// NewFlorenceEngine creates a new Florence-2 inference engine.
// modelPath points to the Florence-2 ONNX model file.
// dllPath points to onnxruntime.dll (DirectML build).
func NewFlorenceEngine(modelPath string, dllPath string) (*FlorenceEngine, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("florence: model path cannot be empty")
	}

	// Validate model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("florence-2 model not found: %s", modelPath)
	}

	// Set ONNX Runtime DLL path (must be called BEFORE InitializeEnvironment)
	if dllPath != "" {
		ort.SetSharedLibraryPath(dllPath)
	}

	// Initialize ONNX Runtime environment
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Create session options with DirectML
	options, err := ort.NewSessionOptions()
	if err != nil {
		ort.DestroyEnvironment()
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}

	// Enable DirectML (deviceID 0 = primary GPU)
	if err := options.AppendExecutionProviderDirectML(0); err != nil {
		// DirectML not available — fall back to CPU
		fmt.Printf("[WARN] DirectML unavailable: %v. Falling back to CPU.\n", err)
	}

	// Create advanced session to inspect inputs and outputs
	session, err := ort.NewAdvancedSession(modelPath, []string{inputName}, []string{outputName}, nil, nil, options)
	if err != nil {
		options.Destroy()
		ort.DestroyEnvironment()
		return nil, fmt.Errorf("failed to create ONNX session (verify tensor names): %w", err)
	}

	return &FlorenceEngine{
		modelPath: modelPath,
		session:   session,
		options:   options,
	}, nil
}

// DetectUIElements runs Florence-2 inference on the provided image data
// and returns detected UI elements with bounding boxes.
func (e *FlorenceEngine) DetectUIElements(imageData []byte) ([]types.UIElement, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("florence: image data cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil, fmt.Errorf("FlorenceEngine is closed")
	}

	if e.session == nil {
		return nil, fmt.Errorf("ONNX session not initialized")
	}

	// 1. Decode image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// 2. Preprocess: HWC -> NCHW
	inputData := preprocessImage(img, 768, 768) // Florence-2 vision encoder input size

	// 3. Create ONNX input tensor
	inputShape := ort.NewShape(1, 3, 768, 768)
	tensor, err := ort.NewTensor(inputShape, inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}
	defer tensor.Destroy()

	// 4. Run inference
	// (Note: Currently waiting for exact tensor names and multi-session logic if encoder/decoder are separate)
	// Because the exact tensor shapes for output are unknown, we leave the post-processing stubbed.
	/*
	err = e.session.Run([]ort.ArbitraryTensor{tensor}, []ort.ArbitraryTensor{outputTensor})
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}
	*/

	return nil, fmt.Errorf("florence: ONNX post-processing not yet implemented; awaiting exact tensor names and model structure")
}

// Close releases the ONNX Runtime session and frees resources.
func (e *FlorenceEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil
	}

	e.closed = true

	if e.session != nil {
		e.session.Destroy()
		e.session = nil
	}

	if e.options != nil {
		e.options.Destroy()
		e.options = nil
	}

	return ort.DestroyEnvironment()
}
