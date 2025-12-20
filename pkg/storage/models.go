package storage

import (
	"time"
)

// Session represents a date-based collection of app activities.
type Session struct {
	ID              int64         `json:"id"`
	Date            string        `json:"date"` // Format: "2006-01-02"
	CustomTitle     string        `json:"customTitle"`
	CustomSummary   string        `json:"customSummary"`
	OriginalSummary string        `json:"originalSummary"`
	ExtractedText   string        `json:"-"` // Encrypted, not in JSON response
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`

	// Relationships (loaded on demand)
	Activities  []AppActivity `json:"activities,omitempty"`
	ManualNotes []ManualNote  `json:"manualNotes,omitempty"`
}

// AppActivity represents activity data for a specific application within a session.
type AppActivity struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"sessionId"`
	AppName   string    `json:"appName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Relationships
	Blocks []ActivityBlock `json:"blocks,omitempty"`
}

// ActivityBlock represents a time-bounded chunk of OCR text with AI-generated summary.
type ActivityBlock struct {
	ID            int64     `json:"id"`
	AppActivityID int64     `json:"appActivityId"`
	BlockID       string    `json:"blockId"` // Format: "HH-MM" (e.g., "15-04")
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	OCRText       string    `json:"ocrText"`      // Encrypted in DB
	MicroSummary  string    `json:"microSummary"`
}

// ChatMessage represents a chat message in a session.
type ChatMessage struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"sessionId"`
	Role      string    `json:"role"` // "user" or "assistant"
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
	ID        int64     `json:"id"`
	SessionID int64     `json:"sessionId"`
	Content   string    `json:"content"`
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

// VectorSearchResult represents a result from LanceDB semantic search.
type VectorSearchResult struct {
	SessionID    int64   `json:"sessionId"`
	Score        float32 `json:"score"` // Cosine similarity
	ModelVersion string  `json:"modelVersion"`
}

// DateRange represents a date range filter for searches.
type DateRange struct {
	StartDate string `json:"startDate"` // Format: "2006-01-02"
	EndDate   string `json:"endDate"`   // Format: "2006-01-02"
}

// StorageStats represents storage usage statistics.
type StorageStats struct {
	TotalFiles      int64     `json:"totalFiles"`
	TotalSizeBytes  int64     `json:"totalSizeBytes"`
	ScreenshotCount int64     `json:"screenshotCount"`
	OldestFile      time.Time `json:"oldestFile"`
}

// MigrationStatus represents the state of a data migration.
type MigrationStatus string

const (
	MigrationStatusIdle        MigrationStatus = "idle"
	MigrationStatusDetecting   MigrationStatus = "detecting"
	MigrationStatusBackingUp   MigrationStatus = "backing_up"
	MigrationStatusMigrating   MigrationStatus = "migrating"
	MigrationStatusVerifying   MigrationStatus = "verifying"
	MigrationStatusComplete    MigrationStatus = "complete"
	MigrationStatusFailed      MigrationStatus = "failed"
	MigrationStatusRollingBack MigrationStatus = "rolling_back"
)

// MigrationState tracks the progress of a data migration.
type MigrationState struct {
	Status           MigrationStatus       `json:"status"`
	StartedAt        time.Time             `json:"startedAt"`
	CompletedAt      *time.Time            `json:"completedAt,omitempty"`
	BackupPath       string                `json:"backupPath"`
	SessionsMigrated int                   `json:"sessionsMigrated"`
	BlocksMigrated   int                   `json:"blocksMigrated"`
	FilesCopied      int                   `json:"filesCopied"`
	LastError        string                `json:"lastError,omitempty"`
	Checkpoints      []MigrationCheckpoint `json:"checkpoints"`
}

// MigrationCheckpoint represents a checkpoint during migration.
type MigrationCheckpoint struct {
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Details   string    `json:"details,omitempty"`
}
