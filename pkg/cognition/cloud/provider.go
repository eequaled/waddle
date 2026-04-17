package cloud

import "context"

// Provider defines the interface for LLM inference backends
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
	Name() string
	HealthCheck(ctx context.Context) error
}

type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionResponse struct {
	ID      string     `json:"id"`
	Content string     `json:"content"`
	Model   string     `json:"model"`
	Usage   TokenUsage `json:"usage"`
}

type StreamChunk struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
