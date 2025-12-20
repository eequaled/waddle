package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StorageEngine is the central coordinator that provides a unified interface for all storage operations.
type StorageEngine struct {
	config        *StorageConfig
	sessionMgr    *SessionManager
	vectorMgr     *VectorManager
	fileMgr       *FileManager
	encryptionMgr *EncryptionManager
}

// NewStorageEngine creates a new StorageEngine instance with the given configuration.
func NewStorageEngine(config *StorageConfig) *StorageEngine {
	return &StorageEngine{
		config: config,
	}
}

// Initialize initializes all storage components and creates necessary directories.
func (se *StorageEngine) Initialize() error {
	// Ensure data directory exists
	if err := os.MkdirAll(se.config.DataDir, 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create data directory", err)
	}

	// Initialize encryption manager first (needed by others)
	se.encryptionMgr = NewEncryptionManager()
	if err := se.encryptionMgr.InitializeKey(); err != nil {
		return NewStorageError(ErrEncryption, "failed to initialize encryption", err)
	}

	// Initialize session manager
	se.sessionMgr = NewSessionManager(se.config.DataDir, se.encryptionMgr)
	if err := se.sessionMgr.Initialize(); err != nil {
		return NewStorageError(ErrDatabase, "failed to initialize session manager", err)
	}

	// Initialize vector manager
	vectorConfig := DefaultVectorManagerConfig(se.config.DataDir)
	vectorConfig.ModelVersion = se.config.EmbeddingModel
	var err error
	se.vectorMgr, err = NewVectorManager(vectorConfig)
	if err != nil {
		return NewStorageError(ErrVector, "failed to initialize vector manager", err)
	}

	// Initialize file manager
	se.fileMgr = NewFileManager(se.config.DataDir)

	return nil
}

// Close closes all storage components and cleans up resources.
func (se *StorageEngine) Close() error {
	var lastErr error

	if se.sessionMgr != nil {
		if err := se.sessionMgr.Close(); err != nil {
			lastErr = err
		}
	}

	if se.vectorMgr != nil {
		if err := se.vectorMgr.Close(); err != nil {
			lastErr = err
		}
	}

	// File manager and encryption manager don't need explicit closing

	return lastErr
}

// Session operations

// CreateSession creates a new session for the given date.
func (se *StorageEngine) CreateSession(date string) (*Session, error) {
	session := &Session{
		Date:      date,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := se.sessionMgr.Create(session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session by date.
func (se *StorageEngine) GetSession(date string) (*Session, error) {
	return se.sessionMgr.Get(date)
}

// UpdateSession updates an existing session.
func (se *StorageEngine) UpdateSession(session *Session) error {
	session.UpdatedAt = time.Now()
	
	// Update in database
	if err := se.sessionMgr.Update(session); err != nil {
		return err
	}

	// Update embedding if text content changed
	if session.CustomSummary != "" || session.OriginalSummary != "" || session.ExtractedText != "" {
		text := session.CustomSummary + " " + session.OriginalSummary + " " + session.ExtractedText
		if err := se.vectorMgr.QueueEmbedding(session.ID, text); err != nil {
			// Log error but don't fail the update
			// In a production system, you'd want proper logging here
		}
	}

	return nil
}

// DeleteSession deletes a session and all associated data.
func (se *StorageEngine) DeleteSession(date string) error {
	// Get session first to get ID
	session, err := se.sessionMgr.Get(date)
	if err != nil {
		return err
	}

	// Delete from all stores
	if err := se.sessionMgr.Delete(date); err != nil {
		return err
	}

	if err := se.vectorMgr.DeleteEmbedding(session.ID); err != nil {
		// Log error but continue - vector might not exist
	}

	if err := se.fileMgr.DeleteSessionFiles(date); err != nil {
		// Log error but continue - files might not exist
	}

	return nil
}

// ListSessions returns a paginated list of sessions.
func (se *StorageEngine) ListSessions(page, pageSize int) ([]Session, int, error) {
	return se.sessionMgr.List(page, pageSize)
}

// Search operations

// FullTextSearch performs full-text search across sessions and activity blocks.
func (se *StorageEngine) FullTextSearch(query string, page, pageSize int) ([]SearchResult, error) {
	return se.sessionMgr.Search(query, page, pageSize)
}

// SemanticSearch performs semantic search using vector embeddings.
func (se *StorageEngine) SemanticSearch(query string, topK int, dateRange *DateRange) ([]SearchResult, error) {
	return se.sessionMgr.SemanticSearch(query, topK, dateRange, se.vectorMgr)
}

// Activity operations

// AddActivityBlock adds an activity block to a session.
func (se *StorageEngine) AddActivityBlock(sessionDate, appName string, block *ActivityBlock) error {
	// Get session to get ID
	session, err := se.sessionMgr.Get(sessionDate)
	if err != nil {
		return err
	}

	return se.sessionMgr.AddBlock(session.ID, appName, block)
}

// GetActivityBlocks retrieves activity blocks for a session and app.
func (se *StorageEngine) GetActivityBlocks(sessionDate, appName string) ([]ActivityBlock, error) {
	// Get session to get ID
	session, err := se.sessionMgr.Get(sessionDate)
	if err != nil {
		return nil, err
	}

	return se.sessionMgr.GetBlocks(session.ID, appName)
}

// Chat operations

// AddChat adds a chat message to a session.
func (se *StorageEngine) AddChat(sessionDate string, chat *ChatMessage) error {
	// Get session to get ID
	session, err := se.sessionMgr.Get(sessionDate)
	if err != nil {
		return err
	}

	return se.sessionMgr.AddChat(session.ID, chat)
}

// GetChats retrieves chat messages for a session.
func (se *StorageEngine) GetChats(sessionDate string) ([]ChatMessage, error) {
	// Get session to get ID
	session, err := se.sessionMgr.Get(sessionDate)
	if err != nil {
		return nil, err
	}

	return se.sessionMgr.GetChats(session.ID)
}

// Notification operations

// AddNotification adds a notification.
func (se *StorageEngine) AddNotification(notif *Notification) error {
	return se.sessionMgr.AddNotification(notif)
}

// GetNotifications retrieves notifications.
func (se *StorageEngine) GetNotifications(limit int) ([]Notification, error) {
	return se.sessionMgr.GetNotifications(limit)
}

// MarkNotificationsRead marks notifications as read.
func (se *StorageEngine) MarkNotificationsRead(ids []string) error {
	return se.sessionMgr.MarkNotificationsRead(ids)
}

// File operations

// SaveScreenshot saves a screenshot file and returns the file path.
func (se *StorageEngine) SaveScreenshot(sessionDate, appName, filename string, data []byte) (string, error) {
	return se.fileMgr.SaveFile(sessionDate, appName, filename, data)
}

// GetScreenshotPath returns the path to a screenshot file.
func (se *StorageEngine) GetScreenshotPath(sessionDate, appName, filename string) string {
	return se.fileMgr.GetFilePath(sessionDate, appName, filename)
}

// Backup creates a backup of all storage components.
func (se *StorageEngine) Backup() error {
	// Create backup directory with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(se.config.DataDir, "backups", timestamp)
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create backup directory", err)
	}

	// Backup SQLite database using VACUUM INTO
	dbBackupPath := filepath.Join(backupDir, "waddle.db")
	_, err := se.sessionMgr.DB().Exec("VACUUM INTO ?", dbBackupPath)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to backup database", err)
	}

	// TODO: Backup vector database and files
	// This would involve copying the LanceDB directory and files directory

	return nil
}

// Restore restores from a backup.
func (se *StorageEngine) Restore(backupPath string) error {
	// TODO: Implement restore functionality
	// This would involve:
	// 1. Validating backup integrity
	// 2. Stopping current operations
	// 3. Replacing current data with backup
	// 4. Reinitializing components
	
	return NewStorageError(ErrNotImplemented, "restore not yet implemented", nil)
}

// HealthCheck performs a health check on all storage components.
func (se *StorageEngine) HealthCheck() (*HealthStatus, error) {
	status := &HealthStatus{
		Status:    HealthStatusHealthy,
		Checks:    make(map[string]Check),
		Timestamp: time.Now(),
	}

	// Check SQLite database
	start := time.Now()
	err := se.sessionMgr.RunIntegrityCheck()
	latency := time.Since(start).Milliseconds()
	
	if err != nil {
		status.Checks["database"] = Check{
			Status:  HealthStatusUnhealthy,
			Latency: latency,
			Message: err.Error(),
		}
		status.Status = HealthStatusUnhealthy
	} else {
		status.Checks["database"] = Check{
			Status:  HealthStatusHealthy,
			Latency: latency,
		}
	}

	// Check vector database
	start = time.Now()
	count := se.vectorMgr.Count()
	latency = time.Since(start).Milliseconds()
	
	status.Checks["vector_db"] = Check{
		Status:  HealthStatusHealthy,
		Latency: latency,
		Message: fmt.Sprintf("%d embeddings", count),
	}

	// Check file system
	start = time.Now()
	stats, err := se.fileMgr.GetStorageStats()
	latency = time.Since(start).Milliseconds()
	
	if err != nil {
		status.Checks["filesystem"] = Check{
			Status:  HealthStatusDegraded,
			Latency: latency,
			Message: err.Error(),
		}
		if status.Status == HealthStatusHealthy {
			status.Status = HealthStatusDegraded
		}
	} else {
		status.Checks["filesystem"] = Check{
			Status:  HealthStatusHealthy,
			Latency: latency,
			Message: fmt.Sprintf("%d files, %d bytes", stats.TotalFiles, stats.TotalSizeBytes),
		}
	}

	return status, nil
}