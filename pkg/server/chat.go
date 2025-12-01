package server

import (
	"encoding/json"
	"fmt"
	"ideathon/pkg/ai"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ChatRequest struct {
	Context   string `json:"context"` // "global" or "session_id" (e.g., "2023-10-27")
	Message   string `json:"message"`
	SessionID string `json:"sessionId,omitempty"` // Optional, specific session ID if context is "session"
}

type ChatMessage struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatSession struct {
	ID        string        `json:"id"`
	Context   string        `json:"context"`
	Messages  []ChatMessage `json:"messages"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// handleChat handles POST /api/chat and GET /api/chat/history
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.getChatHistory(w, r)
		return
	}
	if r.Method == "POST" {
		s.processChat(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) getChatHistory(w http.ResponseWriter, _ *http.Request) {
	chatPath := filepath.Join(s.rootDir, "global_chats", "history.json")

	content, err := os.ReadFile(chatPath)
	if err != nil {
		if os.IsNotExist(err) {
			json.NewEncoder(w).Encode([]ChatSession{})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}

func (s *Server) processChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Gather Context
	contextText := ""
	if req.Context == "global" {
		contextText = s.gatherGlobalContext(req.Message)
	} else {
		contextText = s.gatherSessionContext(req.Context, req.Message)
	}

	// 2. Call AI
	ollama := ai.NewOllamaClient("", "gemma2:2b")

	prompt := fmt.Sprintf(`You are an intelligent assistant helping a developer recall their work.
Context:
%s

User Question: %s

Answer the question based on the context provided. If the context doesn't contain the answer, say so.`, contextText, req.Message)

	response, err := ollama.Summarize("Chat", prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI Error: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Save to History
	chatMsg := ChatMessage{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	}
	userMsg := ChatMessage{
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	}

	s.saveChat(req.Context, userMsg, chatMsg)

	// 4. Return Response
	json.NewEncoder(w).Encode(chatMsg)
}

func (s *Server) saveChat(context string, userMsg, aiMsg ChatMessage) {
	chatDir := filepath.Join(s.rootDir, "global_chats")
	os.MkdirAll(chatDir, 0755)
	chatPath := filepath.Join(chatDir, "history.json")

	var sessions []ChatSession
	content, err := os.ReadFile(chatPath)
	if err == nil {
		json.Unmarshal(content, &sessions)
	}

	found := false
	for i := range sessions {
		if sessions[i].Context == context {
			sessions[i].Messages = append(sessions[i].Messages, userMsg, aiMsg)
			sessions[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		sessions = append(sessions, ChatSession{
			ID:        fmt.Sprintf("chat_%d", time.Now().Unix()),
			Context:   context,
			Messages:  []ChatMessage{userMsg, aiMsg},
			UpdatedAt: time.Now(),
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	data, _ := json.MarshalIndent(sessions, "", "  ")
	os.WriteFile(chatPath, data, 0644)
}

func (s *Server) gatherGlobalContext(_ string) string {
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Recent Activity Summaries:\n")

	entries, _ := os.ReadDir(s.rootDir)

	count := 0
	for i := len(entries) - 1; i >= 0; i-- {
		if !entries[i].IsDir() {
			continue
		}
		date := entries[i].Name()

		if count > 5 {
			break
		}
		count++

		dateDir := filepath.Join(s.rootDir, date)
		apps, _ := os.ReadDir(dateDir)
		for _, app := range apps {
			if !app.IsDir() {
				continue
			}

			blocksDir := filepath.Join(dateDir, app.Name(), "blocks")
			blocks, _ := os.ReadDir(blocksDir)

			for _, block := range blocks {
				if strings.HasSuffix(block.Name(), ".json") {
					content, _ := os.ReadFile(filepath.Join(blocksDir, block.Name()))
					var b struct {
						MicroSummary string `json:"microSummary"`
						StartTime    string `json:"startTime"`
					}
					json.Unmarshal(content, &b)
					if b.MicroSummary != "" {
						contextBuilder.WriteString(fmt.Sprintf("[%s %s] %s: %s\n", date, b.StartTime, app.Name(), b.MicroSummary))
					}
				}
			}
		}
	}
	return contextBuilder.String()
}

func (s *Server) gatherSessionContext(date string, _ string) string {
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Activity for %s:\n", date))

	dateDir := filepath.Join(s.rootDir, date)
	apps, _ := os.ReadDir(dateDir)
	for _, app := range apps {
		if !app.IsDir() {
			continue
		}

		blocksDir := filepath.Join(dateDir, app.Name(), "blocks")
		blocks, _ := os.ReadDir(blocksDir)

		for _, block := range blocks {
			if strings.HasSuffix(block.Name(), ".json") {
				content, _ := os.ReadFile(filepath.Join(blocksDir, block.Name()))
				var b struct {
					MicroSummary string `json:"microSummary"`
					StartTime    string `json:"startTime"`
					OCRText      string `json:"ocrText"`
				}
				json.Unmarshal(content, &b)

				// Include full OCR text for session-specific chat for better detail
				contextBuilder.WriteString(fmt.Sprintf("[%s] %s Summary: %s\n", b.StartTime, app.Name(), b.MicroSummary))
				// Include full OCR text for better context (no truncation)
				if b.OCRText != "" {
					contextBuilder.WriteString(fmt.Sprintf("Full Text:\n%s\n\n", b.OCRText))
				}
			}
		}
	}
	return contextBuilder.String()
}
