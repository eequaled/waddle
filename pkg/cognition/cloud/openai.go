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

// OpenAIProvider implements Provider for OpenAI-compatible APIs
type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewOpenAIProvider(baseURL, apiKey string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	// remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &OpenAIProvider{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	var apiResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage TokenUsage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, errors.New("no choices returned from API")
	}

	return &CompletionResponse{
		ID:      apiResp.ID,
		Content: apiResp.Choices[0].Message.Content,
		Model:   apiResp.Model,
		Usage:   apiResp.Usage,
	}, nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	bodyMap := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stop":        req.Stop,
		"stream":      true,
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	ch := make(chan StreamChunk)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					ch <- StreamChunk{Done: true}
					return
				}

				var streamResp struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}

				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					continue
				}

				if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
					ch <- StreamChunk{
						Content: streamResp.Choices[0].Delta.Content,
						Done:    false,
					}
				}
			}
		}
	}()

	return ch, nil
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return err
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}
