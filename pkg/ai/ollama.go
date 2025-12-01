package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultOllamaURL = "http://localhost:11434"
	DefaultModel     = "gemma2:2b" // Can be overridden
)

type OllamaClient struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewOllamaClient(url, model string) *OllamaClient {
	if url == "" {
		url = DefaultOllamaURL
	}
	if model == "" {
		model = DefaultModel
	}
	return &OllamaClient{
		BaseURL: url,
		Model:   model,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type GenerateRequest struct {
	Model   string  `json:"model"`
	Prompt  string  `json:"prompt"`
	Stream  bool    `json:"stream"`
	Options Options `json:"options,omitempty"`
}

type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Summarize generates a concise summary of the provided text
func (c *OllamaClient) Summarize(appName, contextText string) (string, error) {
	if strings.TrimSpace(contextText) == "" {
		return "", fmt.Errorf("empty context provided")
	}

	prompt := fmt.Sprintf(`You are analyzing screen captures from %s.

OCR Text:
%s

Extract the following from the EXACT text above (do not invent information):
1. **Project/Work**: What specific project, file, or task is shown? Quote exact names/titles.
2. **Links/Titles**: List any URLs, file paths, or document titles you see.
3. **Topics**: What technologies/languages/tools are mentioned or visible?

Be FACTUAL. Only report what you can directly see in the text. If something is unclear, say so.
Summary:`, appName, contextText)

	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
		Options: Options{
			Temperature: 0.1, // Very low for factual extraction
			NumPredict:  400, // Increased for detailed extraction
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := c.Client.Post(fmt.Sprintf("%s/api/generate", c.BaseURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return strings.TrimSpace(genResp.Response), nil
}
