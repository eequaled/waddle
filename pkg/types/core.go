package types

import (
	"encoding/json"
	"strconv"
	"time"
)

// ════════════════════════════════════════════════════════════════════════
// ID TYPES — These enforce string serialization at JSON boundaries.
// Wails v2 uses struct reflection (not MarshalJSON) for TS generation,
// so the ts_type:"string" struct tag is REQUIRED on every field using
// these types.
// ════════════════════════════════════════════════════════════════════════

// SessionID is a type-safe session identifier that serializes as a JSON string.
type SessionID int64

// ElementID is a type-safe element identifier that serializes as a JSON string.
type ElementID int64

// BlockID is a string-typed block identifier in "HH-MM" format.
type BlockID string

// NotificationID is a string-typed notification identifier.
type NotificationID string

func (id SessionID) String() string { return strconv.FormatInt(int64(id), 10) }
func (id ElementID) String() string { return strconv.FormatInt(int64(id), 10) }

func (id SessionID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

func (id *SessionID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*id = SessionID(parsed)
	return nil
}

func (id ElementID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

func (id *ElementID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*id = ElementID(parsed)
	return nil
}

// ════════════════════════════════════════════════════════════════════════
// CANONICAL STRUCT DEFINITIONS — Single source of truth.
// All packages MUST use these (or type aliases) instead of redefining.
// ════════════════════════════════════════════════════════════════════════

// Session represents a date-based collection of app activities.
type Session struct {
	ID              SessionID `json:"id" ts_type:"string"`
	Date            string    `json:"date"` // Format: "2006-01-02"
	CustomTitle     string    `json:"customTitle"`
	CustomSummary   string    `json:"customSummary"`
	OriginalSummary string    `json:"originalSummary"`
	ExtractedText   string    `json:"-"` // Encrypted, not in JSON response
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`

	// Synthesis columns
	EntitiesJSON     string `json:"entitiesJson"`    // JSON array of extracted entities
	SynthesisStatus  string `json:"synthesisStatus"` // "pending", "completed", "failed"
	AISummary        string `json:"aiSummary"`       // AI-generated summary
	AIBullets        string `json:"aiBullets"`       // JSON array of 3 bullet points
	EncryptionStatus string `json:"encryptionStatus,omitempty"` // "stale" if decryption failed

	// Relationships (loaded on demand)
	Activities  []AppActivity `json:"activities,omitempty"`
	ManualNotes []ManualNote  `json:"manualNotes,omitempty"`
}

// AppActivity represents activity data for a specific application within a session.
type AppActivity struct {
	ID        ElementID `json:"id" ts_type:"string"`
	SessionID SessionID `json:"sessionId" ts_type:"string"`
	AppName   string    `json:"appName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Relationships
	Blocks []ActivityBlock `json:"blocks,omitempty"`
}

// ActivityBlock represents a time-bounded chunk of OCR text with AI-generated summary.
type ActivityBlock struct {
	ID            ElementID `json:"id" ts_type:"string"`
	AppActivityID ElementID `json:"appActivityId" ts_type:"string"`
	BlockID       string    `json:"blockId"` // Format: "HH-MM" (e.g., "15-04")
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	OCRText       string    `json:"ocrText"`      // Encrypted in DB
	MicroSummary  string    `json:"microSummary"`

	// Capture columns (P0 requirements)
	CaptureSource      string `json:"captureSource"`      // "etw_uia", "uia_fallback", "polling_ocr"
	StructuredMetadata string `json:"structuredMetadata"` // JSON object with app-specific data
}

// ChatMessage represents a chat message in a session.
type ChatMessage struct {
	ID        ElementID `json:"id" ts_type:"string"`
	SessionID SessionID `json:"sessionId" ts_type:"string"`
	Role      string    `json:"role"`    // "user" or "assistant"
	Content   string    `json:"content"` // Encrypted in DB
	Timestamp time.Time `json:"timestamp"`
}

// ChatRole constants for validation.
const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
)

// ValidChatRoles contains all valid chat roles.
var ValidChatRoles = map[string]bool{
	ChatRoleUser:      true,
	ChatRoleAssistant: true,
}

// Notification represents a system notification.
type Notification struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	Read       bool      `json:"read"`
	SessionRef string    `json:"sessionRef,omitempty"`
	Metadata   string    `json:"metadata,omitempty"` // JSON string
}

// ManualNote represents a user-created note within a session.
type ManualNote struct {
	ID        ElementID `json:"id" ts_type:"string"`
	SessionID SessionID `json:"sessionId" ts_type:"string"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// KnowledgeCard represents an AI-generated summary card for a session.
type KnowledgeCard struct {
	ID        ElementID `json:"id" ts_type:"string"`
	SessionID SessionID `json:"sessionId" ts_type:"string"`
	Title     string    `json:"title"`
	Bullets   string    `json:"bullets"`  // JSON array of 3 bullet points
	Entities  string    `json:"entities"` // JSON array of extracted entities
	Status    string    `json:"status"`   // "pending", "completed", "failed"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SearchResult represents a search result from full-text or semantic search.
type SearchResult struct {
	Session   Session `json:"session"`
	Score     float32 `json:"score"`     // Relevance/similarity score
	Snippet   string  `json:"snippet"`   // Highlighted text snippet
	MatchType string  `json:"matchType"` // "fulltext" or "semantic"
}

// SearchMatchType constants.
const (
	MatchTypeFullText = "fulltext"
	MatchTypeSemantic = "semantic"
)

// VectorSearchResult represents a result from vector semantic search.
type VectorSearchResult struct {
	SessionID    SessionID `json:"sessionId" ts_type:"string"`
	Score        float32   `json:"score"` // Cosine similarity
	ModelVersion string    `json:"modelVersion"`
}

// DateRange represents a date range filter for searches.
type DateRange struct {
	StartDate string `json:"startDate"` // Format: "2006-01-02"
	EndDate   string `json:"endDate"`   // Format: "2006-01-02"
}

// Entity represents an extracted entity.
type Entity struct {
	Value string     `json:"value"`
	Type  EntityType `json:"type"`
	Count int        `json:"count"`
}

// EntityType enumerates entity categories for extraction.
type EntityType string

const (
	EntityTypeJiraTicket EntityType = "jira_ticket"
	EntityTypeHashtag    EntityType = "hashtag"
	EntityTypeMention    EntityType = "mention"
	EntityTypeURL        EntityType = "url"
)

// FocusEvent represents a focus event for UI grounding.
type FocusEvent struct {
	ID        ElementID `json:"id" ts_type:"string"`
	SessionID SessionID `json:"sessionId" ts_type:"string"`
	Timestamp time.Time `json:"timestamp"`
	Element   string    `json:"element"`
}

// UIElement represents a detected UI element from Florence-2.
type UIElement struct {
	BBox       [4]float32 `json:"bbox"`       // [x1, y1, x2, y2] normalized coordinates
	Label      string     `json:"label"`      // e.g., "button", "text", "input"
	Confidence float32    `json:"confidence"` // confidence score
	Text       string     `json:"text"`       // OCR text if applicable
}

// ActionSpec represents an action specification.
type ActionSpec struct {
	ID   ElementID `json:"id" ts_type:"string"`
	Name string    `json:"name"`
	Data string    `json:"data"`
}

// ScreenState represents the state of the screen at a given time.
type ScreenState struct {
	ID        ElementID `json:"id" ts_type:"string"`
	SessionID SessionID `json:"sessionId" ts_type:"string"`
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data"`
}
