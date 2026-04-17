//go:build !llama_cpp

package local

import (
	"context"
	"fmt"
)

// LocalInference is the stub implementation when llama_cpp build tag is not set.
// All methods return errors indicating that CGo mode is required.
type LocalInference struct {
	modelPath string
}

// NewLocalInference returns a stub that always errors without the llama_cpp build tag.
func NewLocalInference(modelPath string) (*LocalInference, error) {
	return nil, fmt.Errorf("local inference requires build tag: -tags llama_cpp")
}

// Predict is a stub that returns an error when llama_cpp is not enabled.
func (li *LocalInference) Predict(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("local inference requires build tag: -tags llama_cpp")
}

// Close is a no-op in stub mode.
func (li *LocalInference) Close() error {
	return nil
}
