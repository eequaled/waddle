//go:build !windows

package local

import (
	"context"
	"fmt"
)

// LocalInference is the stub implementation when running on non-Windows OS.
// All methods return errors indicating that Windows DLL is required.
type LocalInference struct {
	modelPath string
}

// NewLocalInference returns a stub that always errors on non-Windows.
func NewLocalInference(modelPath string) (*LocalInference, error) {
	return nil, fmt.Errorf("local inference requires Windows (llama.dll)")
}

// Predict is a stub that returns an error.
func (li *LocalInference) Predict(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("local inference requires Windows (llama.dll)")
}

// Close is a no-op in stub mode.
func (li *LocalInference) Close() error {
	return nil
}
