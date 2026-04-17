package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProvider_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("Expected path /v1/models, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "test-api-key")
	err := provider.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestLocalProvider_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ollama is running"))
	}))
	defer server.Close()

	provider := NewLocalProvider(server.URL)
	err := provider.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRouter_SelectProvider(t *testing.T) {
	local := NewLocalProvider("http://local")
	cloud := NewOpenAIProvider("http://cloud", "key")

	tests := []struct {
		name     string
		mode     string
		local    Provider
		cloud    Provider
		expected Provider
		wantErr  bool
	}{
		{"local mode with local provider", "local", local, cloud, local, false},
		{"local mode without local provider", "local", nil, cloud, nil, true},
		{"cloud mode with cloud provider", "cloud", local, cloud, cloud, false},
		{"hybrid mode prefers cloud", "hybrid", local, cloud, cloud, false},
		{"hybrid mode falls back to local", "hybrid", local, nil, local, false},
		{"invalid mode", "invalid", local, cloud, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.local, tt.cloud, tt.mode)
			provider, err := router.selectProvider()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if provider != tt.expected {
					t.Errorf("Expected provider %v, got %v", tt.expected, provider)
				}
			}
		})
	}
}

func TestOpenAIProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"id":    "test-id",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello, world!",
					},
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "test-api-key")
	
	req := CompletionRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hi"},
		},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", resp.Content)
	}
	if resp.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", resp.ID)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}
