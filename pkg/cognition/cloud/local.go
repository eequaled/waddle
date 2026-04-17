package cloud

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// LocalProvider wraps the existing Ollama client as a Provider
type LocalProvider struct {
	ollamaURL  string
	httpClient *http.Client
}

func NewLocalProvider(ollamaURL string) *LocalProvider {
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollamaURL = strings.TrimSuffix(ollamaURL, "/")

	return &LocalProvider{
		ollamaURL:  ollamaURL,
		httpClient: &http.Client{},
	}
}

func (p *LocalProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Translate CompletionRequest to Ollama API format
	bodyMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": req.Temperature,
		},
	}
	
	if req.MaxTokens > 0 {
		bodyMap["options"].(map[string]interface{})["num_predict"] = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		bodyMap["options"].(map[string]interface{})["stop"] = req.Stop
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.ollamaURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp struct {
		Model   string  `json:"model"`
		Message Message `json:"message"`
		Done    bool    `json:"done"`
		// Ollama provides different usage metrics but we try to map them if possible
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Done {
		return nil, errors.New("ollama request not fully completed")
	}

	return &CompletionResponse{
		ID:      "local-ollama-" + apiResp.Model, // Ollama doesn't return a specific ID
		Content: apiResp.Message.Content,
		Model:   apiResp.Model,
		Usage: TokenUsage{
			PromptTokens:     apiResp.PromptEvalCount,
			CompletionTokens: apiResp.EvalCount,
			TotalTokens:      apiResp.PromptEvalCount + apiResp.EvalCount,
		},
	}, nil
}

func (p *LocalProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	bodyMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
		"options": map[string]interface{}{
			"temperature": req.Temperature,
		},
	}
	
	if req.MaxTokens > 0 {
		bodyMap["options"].(map[string]interface{})["num_predict"] = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		bodyMap["options"].(map[string]interface{})["stop"] = req.Stop
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.ollamaURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("Ollama API returned non-200 status: %d", resp.StatusCode)
	}

	ch := make(chan StreamChunk)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			var chunkResp struct {
				Message Message `json:"message"`
				Done    bool    `json:"done"`
			}
			
			if err := json.Unmarshal(scanner.Bytes(), &chunkResp); err != nil {
				continue
			}

			if chunkResp.Message.Content != "" || chunkResp.Done {
				ch <- StreamChunk{
					Content: chunkResp.Message.Content,
					Done:    chunkResp.Done,
				}
			}
			
			if chunkResp.Done {
				return
			}
		}
	}()

	return ch, nil
}

func (p *LocalProvider) Name() string { return "local" }

func (p *LocalProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.ollamaURL+"/", nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama health check failed with status: %d", resp.StatusCode)
	}

	return nil
}
