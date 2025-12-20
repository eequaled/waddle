package server

import (
	"encoding/json"
	"fmt"
	"waddle/pkg/ai"
	"waddle/pkg/storage"
	"net/http"
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
	// Use StorageEngine to get chat history
	// For now, return empty array since we need to implement global chat storage
	// TODO: Implement global chat storage in StorageEngine
	json.NewEncoder(w).Encode([]ChatSession{})
}

func (s *Server) processChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Gather Context using StorageEngine
	contextText := ""
	if req.Context == "global" {
		contextText = s.gatherGlobalContextFromStorage(req.Message)
	} else {
		contextText = s.gatherSessionContextFromStorage(req.Context, req.Message)
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

	// 3. Save to History using StorageEngine
	chatMsg := storage.ChatMessage{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	}
	userMsg := storage.ChatMessage{
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	}

	s.saveChatToStorage(req.Context, userMsg, chatMsg)

	// 4. Return Response (convert to API format)
	apiChatMsg := ChatMessage{
		Role:      chatMsg.Role,
		Content:   chatMsg.Content,
		Timestamp: chatMsg.Timestamp,
	}

	json.NewEncoder(w).Encode(apiChatMsg)
}

func (s *Server) saveChatToStorage(context string, userMsg, aiMsg storage.ChatMessage) {
	// For session-specific chats, save to the session
	if context != "global" {
		// Save user message
		if err := s.storageEngine.AddChat(context, &userMsg); err != nil {
			// Log error but don't fail
			fmt.Printf("Error saving user chat message: %v\n", err)
		}
		
		// Save AI message
		if err := s.storageEngine.AddChat(context, &aiMsg); err != nil {
			// Log error but don't fail
			fmt.Printf("Error saving AI chat message: %v\n", err)
		}
	}
	// TODO: Implement global chat storage in StorageEngine
}

func (s *Server) gatherGlobalContextFromStorage(query string) string {
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Recent Activity Summaries:\n")

	// Get recent sessions using StorageEngine
	sessions, _, err := s.storageEngine.ListSessions(1, 10) // Get last 10 sessions
	if err != nil {
		return "No recent activity found."
	}

	for _, session := range sessions {
		// Add session summary if available
		if session.OriginalSummary != "" {
			contextBuilder.WriteString(fmt.Sprintf("[%s] %s\n", session.Date, session.OriginalSummary))
		}
		if session.CustomSummary != "" {
			contextBuilder.WriteString(fmt.Sprintf("[%s] Custom: %s\n", session.Date, session.CustomSummary))
		}
		
		// Add some extracted text if available
		if session.ExtractedText != "" {
			// Truncate to first 200 chars for context
			text := session.ExtractedText
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			contextBuilder.WriteString(fmt.Sprintf("[%s] Text: %s\n", session.Date, text))
		}
	}

	return contextBuilder.String()
}

func (s *Server) gatherSessionContextFromStorage(date string, query string) string {
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Activity for %s:\n", date))

	// Get session details
	session, err := s.storageEngine.GetSession(date)
	if err != nil {
		return fmt.Sprintf("No data found for %s", date)
	}

	// Add session summaries
	if session.OriginalSummary != "" {
		contextBuilder.WriteString(fmt.Sprintf("Original Summary: %s\n", session.OriginalSummary))
	}
	if session.CustomSummary != "" {
		contextBuilder.WriteString(fmt.Sprintf("Custom Summary: %s\n", session.CustomSummary))
	}
	if session.ExtractedText != "" {
		contextBuilder.WriteString(fmt.Sprintf("Extracted Text: %s\n", session.ExtractedText))
	}

	// Get chat history for this session
	chats, err := s.storageEngine.GetChats(date)
	if err == nil && len(chats) > 0 {
		contextBuilder.WriteString("\nPrevious Chat History:\n")
		for _, chat := range chats {
			contextBuilder.WriteString(fmt.Sprintf("[%s] %s: %s\n", 
				chat.Timestamp.Format("15:04"), chat.Role, chat.Content))
		}
	}

	return contextBuilder.String()
}
