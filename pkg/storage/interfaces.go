package storage

import (
	"time"
)

// StorageConfig holds configuration for the storage engine.
type StorageConfig struct {
	DataDir        string // Base directory (default: ~/.waddle/)
	EncryptionKey  []byte // Derived from DPAPI
	RetentionDays  int    // Default: 365
	BackupTime     string // Default: "02:00"
	EmbeddingModel string // Default: "nomic-embed-text"
}

// DefaultStorageConfig returns a StorageConfig with default values.
func DefaultStorageConfig(dataDir string) *StorageConfig {
	return &StorageConfig{
		DataDir:        dataDir,
		RetentionDays:  365,
		BackupTime:     "02:00",
		EmbeddingModel: "nomic-embed-text",
	}
}

// StorageEngineInterface defines the main entry point for all storage operations.
type StorageEngineInterface interface {
	// Session operations
	CreateSession(date string) (*Session, error)
	GetSession(date string) (*Session, error)
	UpdateSession(session *Session) error
	DeleteSession(date string) error
	ListSessions(page, pageSize int) ([]Session, int, error)

	// Search operations
	FullTextSearch(query string, page, pageSize int) ([]SearchResult, error)
	SemanticSearch(query string, topK int, dateRange *DateRange) ([]SearchResult, error)

	// Activity operations
	AddActivityBlock(sessionDate, appName string, block *ActivityBlock) error
	GetActivityBlocks(sessionDate, appName string) ([]ActivityBlock, error)

	// Chat operations
	AddChat(sessionDate string, chat *ChatMessage) error
	GetChats(sessionDate string) ([]ChatMessage, error)

	// File operations
	SaveScreenshot(sessionDate, appName, filename string, data []byte) (string, error)
	GetScreenshotPath(sessionDate, appName, filename string) string

	// Lifecycle
	Initialize() error
	Close() error
	Backup() error
	Restore(backupPath string) error

	// Health
	HealthCheck() (*HealthStatus, error)
}

// SessionManagerInterface handles all SQLite database operations.
type SessionManagerInterface interface {
	// CRUD operations
	Create(session *Session) error
	Get(date string) (*Session, error)
	Update(session *Session) error
	Delete(date string) error
	List(page, pageSize int) ([]Session, int, error) // Returns sessions and total count

	// FTS5 Search
	Search(query string, page, pageSize int) ([]SearchResult, error)

	// Activity Blocks
	AddBlock(sessionID int64, appName string, block *ActivityBlock) error
	GetBlocks(sessionID int64, appName string) ([]ActivityBlock, error)

	// Chats
	AddChat(sessionID int64, chat *ChatMessage) error
	GetChats(sessionID int64) ([]ChatMessage, error)

	// Notifications
	AddNotification(notif *Notification) error
	GetNotifications(limit int) ([]Notification, error)
	MarkNotificationsRead(ids []string) error

	// Maintenance
	RunIntegrityCheck() error
	Vacuum() error

	// Lifecycle
	Close() error
}

// VectorManagerInterface manages vector embeddings and semantic search using LanceDB.
type VectorManagerInterface interface {
	// Embedding operations
	GenerateEmbedding(text string) ([]float32, error)
	StoreEmbedding(sessionID int64, embedding []float32) error
	UpdateEmbedding(sessionID int64, embedding []float32) error
	DeleteEmbedding(sessionID int64) error

	// Search
	Search(queryEmbedding []float32, topK int) ([]VectorSearchResult, error)

	// Async operations
	QueueEmbedding(sessionID int64, text string) error
	ProcessQueue() error

	// Maintenance
	Reindex(modelVersion string) error

	// Lifecycle
	Close() error
}

// EmbedRequest represents an async embedding generation request.
type EmbedRequest struct {
	SessionID int64
	Text      string
	Callback  func(error)
}

// FileManagerInterface handles filesystem operations for binary assets.
type FileManagerInterface interface {
	// File operations
	SaveFile(sessionID, appName, filename string, data []byte) (string, error)
	GetFilePath(sessionID, appName, filename string) string
	DeleteSessionFiles(sessionID string) error

	// Maintenance
	CleanOrphanedFiles(validSessionIDs []string) (int, error)
	CompressOldScreenshots(olderThan time.Duration) error
	GetStorageStats() (*StorageStats, error)
}

// EncryptionManagerInterface handles data encryption/decryption using Windows DPAPI and AES-256-GCM.
type EncryptionManagerInterface interface {
	// Key management
	InitializeKey() error
	RotateKey(newPassphrase string) error

	// Encryption operations
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)

	// String helpers (for database columns)
	EncryptString(plaintext string) (string, error)
	DecryptString(ciphertext string) (string, error)
}

// HealthStatus represents the health of the storage system.
type HealthStatus struct {
	Status    string           `json:"status"` // "healthy", "degraded", "unhealthy"
	Checks    map[string]Check `json:"checks"`
	Timestamp time.Time        `json:"timestamp"`
}

// Check represents a single health check result.
type Check struct {
	Status  string `json:"status"`
	Latency int64  `json:"latency_ms"`
	Message string `json:"message,omitempty"`
}

// Health status constants.
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
)
