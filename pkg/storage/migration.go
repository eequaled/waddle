package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MigrationManager handles the migration from JSON-based storage to SQLite storage.
type MigrationManager struct {
	config      *StorageConfig
	statePath   string
	legacyPath  string // Path to old JSON data (e.g., ~/Documents/Waddle/sessions/)
	backupPath  string // Path for backup during migration
}

// NewMigrationManager creates a new migration manager.
func NewMigrationManager(config *StorageConfig, legacyPath string) *MigrationManager {
	statePath := filepath.Join(config.DataDir, "migration_state.json")
	backupPath := filepath.Join(config.DataDir, "backup", fmt.Sprintf("migration-%d", time.Now().Unix()))
	
	return &MigrationManager{
		config:     config,
		statePath:  statePath,
		legacyPath: legacyPath,
		backupPath: backupPath,
	}
}

// GetState loads the current migration state from disk.
func (mm *MigrationManager) GetState() (*MigrationState, error) {
	// Check if state file exists
	if _, err := os.Stat(mm.statePath); os.IsNotExist(err) {
		// Return default idle state
		return &MigrationState{
			Status:      MigrationStatusIdle,
			StartedAt:   time.Time{},
			Checkpoints: []MigrationCheckpoint{},
		}, nil
	}

	// Load existing state
	data, err := os.ReadFile(mm.statePath)
	if err != nil {
		return nil, NewStorageError(ErrFileSystem, "failed to read migration state", err)
	}

	var state MigrationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, NewStorageError(ErrSerialization, "failed to parse migration state", err)
	}

	return &state, nil
}

// SaveState persists the migration state to disk.
func (mm *MigrationManager) SaveState(state *MigrationState) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(mm.statePath), 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create state directory", err)
	}

	// Serialize state
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return NewStorageError(ErrSerialization, "failed to serialize migration state", err)
	}

	// Write to file
	if err := os.WriteFile(mm.statePath, data, 0644); err != nil {
		return NewStorageError(ErrFileSystem, "failed to write migration state", err)
	}

	return nil
}

// TransitionTo transitions the migration state to a new status.
func (mm *MigrationManager) TransitionTo(currentState *MigrationState, newStatus MigrationStatus, details string) error {
	// Validate state transition
	if !mm.isValidTransition(currentState.Status, newStatus) {
		return NewStorageError(ErrValidation, 
			fmt.Sprintf("invalid state transition from %s to %s", currentState.Status, newStatus), nil)
	}

	// Update state
	currentState.Status = newStatus
	
	// Set timestamps
	if newStatus == MigrationStatusDetecting && currentState.StartedAt.IsZero() {
		currentState.StartedAt = time.Now()
	}
	
	if newStatus == MigrationStatusComplete || newStatus == MigrationStatusFailed {
		now := time.Now()
		currentState.CompletedAt = &now
	}

	// Add checkpoint
	checkpoint := MigrationCheckpoint{
		Name:      string(newStatus),
		Timestamp: time.Now(),
		Success:   newStatus != MigrationStatusFailed,
		Details:   details,
	}
	currentState.Checkpoints = append(currentState.Checkpoints, checkpoint)

	// Save state
	return mm.SaveState(currentState)
}

// isValidTransition checks if a state transition is valid.
func (mm *MigrationManager) isValidTransition(from, to MigrationStatus) bool {
	validTransitions := map[MigrationStatus][]MigrationStatus{
		MigrationStatusIdle: {
			MigrationStatusDetecting,
		},
		MigrationStatusDetecting: {
			MigrationStatusBackingUp,
			MigrationStatusComplete, // No migration needed
			MigrationStatusFailed,
		},
		MigrationStatusBackingUp: {
			MigrationStatusMigrating,
			MigrationStatusFailed,
		},
		MigrationStatusMigrating: {
			MigrationStatusVerifying,
			MigrationStatusFailed,
			MigrationStatusRollingBack,
		},
		MigrationStatusVerifying: {
			MigrationStatusComplete,
			MigrationStatusFailed,
			MigrationStatusRollingBack,
		},
		MigrationStatusFailed: {
			MigrationStatusRollingBack,
			MigrationStatusIdle, // Reset for retry
		},
		MigrationStatusRollingBack: {
			MigrationStatusIdle,
			MigrationStatusFailed,
		},
		MigrationStatusComplete: {
			// Terminal state - no transitions allowed
		},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedTo := range allowed {
		if allowedTo == to {
			return true
		}
	}

	return false
}

// DetectLegacyData checks if there's legacy JSON data that needs migration.
func (mm *MigrationManager) DetectLegacyData() (bool, error) {
	// Check if legacy sessions directory exists
	if _, err := os.Stat(mm.legacyPath); os.IsNotExist(err) {
		return false, nil
	}

	// Check if there are any session directories
	entries, err := os.ReadDir(mm.legacyPath)
	if err != nil {
		return false, NewStorageError(ErrFileSystem, "failed to read legacy directory", err)
	}

	// Look for date-formatted directories (YYYY-MM-DD)
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 10 && entry.Name()[4] == '-' && entry.Name()[7] == '-' {
			return true, nil
		}
	}

	return false, nil
}

// CreateBackup creates a backup of the existing legacy data.
func (mm *MigrationManager) CreateBackup(state *MigrationState) error {
	// Create backup directory
	if err := os.MkdirAll(mm.backupPath, 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create backup directory", err)
	}

	// Update state with backup path
	state.BackupPath = mm.backupPath

	// Copy legacy data directory
	if err := mm.copyDirectory(mm.legacyPath, filepath.Join(mm.backupPath, "sessions")); err != nil {
		return NewStorageError(ErrFileSystem, "failed to copy legacy data", err)
	}

	// Copy notifications.json if it exists
	notificationsPath := filepath.Join(filepath.Dir(mm.legacyPath), "notifications.json")
	if _, err := os.Stat(notificationsPath); err == nil {
		destPath := filepath.Join(mm.backupPath, "notifications.json")
		if err := mm.copyFile(notificationsPath, destPath); err != nil {
			return NewStorageError(ErrFileSystem, "failed to copy notifications", err)
		}
	}

	// Verify backup integrity
	if err := mm.verifyBackup(state); err != nil {
		return err
	}

	return mm.AddCheckpoint(state, "backup_created", fmt.Sprintf("Backup created at %s", mm.backupPath), true)
}

// copyDirectory recursively copies a directory.
func (mm *MigrationManager) copyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := mm.copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := mm.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func (mm *MigrationManager) copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy file contents
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// verifyBackup verifies the integrity of the backup.
func (mm *MigrationManager) verifyBackup(state *MigrationState) error {
	// Compare file counts
	originalCount, err := mm.countFiles(mm.legacyPath)
	if err != nil {
		return NewStorageError(ErrFileSystem, "failed to count original files", err)
	}

	backupCount, err := mm.countFiles(filepath.Join(mm.backupPath, "sessions"))
	if err != nil {
		return NewStorageError(ErrFileSystem, "failed to count backup files", err)
	}

	if originalCount != backupCount {
		return NewStorageError(ErrValidation, 
			fmt.Sprintf("backup file count mismatch: original=%d, backup=%d", originalCount, backupCount), nil)
	}

	// Verify a few key files by checksum
	if err := mm.verifyKeyFiles(); err != nil {
		return err
	}

	return nil
}

// countFiles recursively counts files in a directory.
func (mm *MigrationManager) countFiles(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// verifyKeyFiles verifies checksums of key files.
func (mm *MigrationManager) verifyKeyFiles() error {
	// Find a few sample files to verify
	sampleFiles := []string{}
	
	err := filepath.Walk(mm.legacyPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".json" || filepath.Ext(path) == ".txt") {
			sampleFiles = append(sampleFiles, path)
			if len(sampleFiles) >= 5 { // Verify up to 5 files
				return filepath.SkipDir
			}
		}
		return nil
	})
	
	if err != nil {
		return err
	}

	// Verify checksums
	for _, originalPath := range sampleFiles {
		relativePath, err := filepath.Rel(mm.legacyPath, originalPath)
		if err != nil {
			continue
		}
		
		backupPath := filepath.Join(mm.backupPath, "sessions", relativePath)
		
		if err := mm.compareFileChecksums(originalPath, backupPath); err != nil {
			return NewStorageError(ErrValidation, 
				fmt.Sprintf("checksum mismatch for file %s", relativePath), err)
		}
	}

	return nil
}

// compareFileChecksums compares SHA256 checksums of two files.
func (mm *MigrationManager) compareFileChecksums(file1, file2 string) error {
	hash1, err := mm.calculateFileChecksum(file1)
	if err != nil {
		return err
	}

	hash2, err := mm.calculateFileChecksum(file2)
	if err != nil {
		return err
	}

	if hash1 != hash2 {
		return fmt.Errorf("checksums do not match")
	}

	return nil
}

// calculateFileChecksum calculates SHA256 checksum of a file.
func (mm *MigrationManager) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// SetError sets an error message in the migration state.
func (mm *MigrationManager) SetError(state *MigrationState, err error) error {
	state.LastError = err.Error()
	return mm.SaveState(state)
}

// AddCheckpoint adds a checkpoint to the migration state.
func (mm *MigrationManager) AddCheckpoint(state *MigrationState, name, details string, success bool) error {
	checkpoint := MigrationCheckpoint{
		Name:      name,
		Timestamp: time.Now(),
		Success:   success,
		Details:   details,
	}
	state.Checkpoints = append(state.Checkpoints, checkpoint)
	return mm.SaveState(state)
}

// Reset resets the migration state to idle (for retry).
func (mm *MigrationManager) Reset() error {
	state := &MigrationState{
		Status:      MigrationStatusIdle,
		StartedAt:   time.Time{},
		Checkpoints: []MigrationCheckpoint{},
	}
	return mm.SaveState(state)
}

// MigrateData performs the actual data migration from JSON to SQLite.
func (mm *MigrationManager) MigrateData(state *MigrationState, storageEngine *StorageEngine) error {
	// Reset counters
	state.SessionsMigrated = 0
	state.BlocksMigrated = 0
	state.FilesCopied = 0

	// Migrate sessions
	if err := mm.migrateSessions(state, storageEngine); err != nil {
		return err
	}

	// Migrate notifications
	if err := mm.migrateNotifications(state, storageEngine); err != nil {
		return err
	}

	// Generate embeddings for migrated sessions
	if err := mm.generateEmbeddings(state, storageEngine); err != nil {
		return err
	}

	return mm.AddCheckpoint(state, "data_migration_complete", 
		fmt.Sprintf("Migrated %d sessions, %d blocks, %d files", 
			state.SessionsMigrated, state.BlocksMigrated, state.FilesCopied), true)
}

// migrateSessions migrates session data from JSON to SQLite.
func (mm *MigrationManager) migrateSessions(state *MigrationState, storageEngine *StorageEngine) error {
	// Read all session directories
	entries, err := os.ReadDir(mm.legacyPath)
	if err != nil {
		return NewStorageError(ErrFileSystem, "failed to read legacy sessions", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it's a date directory (YYYY-MM-DD format)
		sessionDate := entry.Name()
		if len(sessionDate) != 10 || sessionDate[4] != '-' || sessionDate[7] != '-' {
			continue
		}

		if err := mm.migrateSession(sessionDate, state, storageEngine); err != nil {
			return NewStorageError(ErrMigration, 
				fmt.Sprintf("failed to migrate session %s", sessionDate), err)
		}

		state.SessionsMigrated++
		if err := mm.SaveState(state); err != nil {
			return err
		}
	}

	return nil
}

// migrateSession migrates a single session.
func (mm *MigrationManager) migrateSession(sessionDate string, state *MigrationState, storageEngine *StorageEngine) error {
	sessionDir := filepath.Join(mm.legacyPath, sessionDate)

	// Create session in new storage
	session, err := storageEngine.CreateSession(sessionDate)
	if err != nil {
		return err
	}

	// Look for metadata.json
	metadataPath := filepath.Join(sessionDir, "metadata.json")
	if _, err := os.Stat(metadataPath); err == nil {
		if err := mm.migrateSessionMetadata(metadataPath, session, storageEngine); err != nil {
			return err
		}
	}

	// Migrate app activities
	if err := mm.migrateAppActivities(sessionDir, sessionDate, state, storageEngine); err != nil {
		return err
	}

	return nil
}

// migrateSessionMetadata migrates session metadata from JSON.
func (mm *MigrationManager) migrateSessionMetadata(metadataPath string, session *Session, storageEngine *StorageEngine) error {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}

	var metadata struct {
		CustomTitle     string `json:"customTitle,omitempty"`
		CustomSummary   string `json:"customSummary,omitempty"`
		OriginalSummary string `json:"originalSummary,omitempty"`
		ExtractedText   string `json:"extractedText,omitempty"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}

	// Update session with metadata
	session.CustomTitle = metadata.CustomTitle
	session.CustomSummary = metadata.CustomSummary
	session.OriginalSummary = metadata.OriginalSummary
	session.ExtractedText = metadata.ExtractedText

	return storageEngine.UpdateSession(session)
}

// migrateAppActivities migrates app activities and blocks.
func (mm *MigrationManager) migrateAppActivities(sessionDir, sessionDate string, state *MigrationState, storageEngine *StorageEngine) error {
	// Read app directories
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appName := entry.Name()
		appDir := filepath.Join(sessionDir, appName)

		// Migrate activity blocks
		if err := mm.migrateActivityBlocks(appDir, sessionDate, appName, state, storageEngine); err != nil {
			return err
		}

		// Copy screenshot files
		if err := mm.copyScreenshots(appDir, sessionDate, appName, state, storageEngine); err != nil {
			return err
		}
	}

	return nil
}

// migrateActivityBlocks migrates activity blocks from JSON files.
func (mm *MigrationManager) migrateActivityBlocks(appDir, sessionDate, appName string, state *MigrationState, storageEngine *StorageEngine) error {
	blocksDir := filepath.Join(appDir, "blocks")
	if _, err := os.Stat(blocksDir); os.IsNotExist(err) {
		return nil // No blocks directory
	}

	entries, err := os.ReadDir(blocksDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			blockPath := filepath.Join(blocksDir, entry.Name())
			if err := mm.migrateActivityBlock(blockPath, sessionDate, appName, state, storageEngine); err != nil {
				return err
			}
			state.BlocksMigrated++
		}
	}

	return nil
}

// migrateActivityBlock migrates a single activity block.
func (mm *MigrationManager) migrateActivityBlock(blockPath, sessionDate, appName string, state *MigrationState, storageEngine *StorageEngine) error {
	data, err := os.ReadFile(blockPath)
	if err != nil {
		return err
	}

	var blockData struct {
		BlockID      string    `json:"blockId"`
		StartTime    time.Time `json:"startTime"`
		EndTime      time.Time `json:"endTime"`
		OCRText      string    `json:"ocrText"`
		MicroSummary string    `json:"microSummary"`
	}

	if err := json.Unmarshal(data, &blockData); err != nil {
		return err
	}

	// Create activity block
	block := &ActivityBlock{
		BlockID:      blockData.BlockID,
		StartTime:    blockData.StartTime,
		EndTime:      blockData.EndTime,
		OCRText:      blockData.OCRText,
		MicroSummary: blockData.MicroSummary,
	}

	return storageEngine.AddActivityBlock(sessionDate, appName, block)
}

// copyScreenshots copies screenshot files to the new file structure.
func (mm *MigrationManager) copyScreenshots(appDir, sessionDate, appName string, state *MigrationState, storageEngine *StorageEngine) error {
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's a screenshot file
		filename := entry.Name()
		if filepath.Ext(filename) == ".png" || filepath.Ext(filename) == ".jpg" || filepath.Ext(filename) == ".jpeg" {
			srcPath := filepath.Join(appDir, filename)
			
			// Read file data
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}

			// Save using StorageEngine
			if _, err := storageEngine.SaveScreenshot(sessionDate, appName, filename, data); err != nil {
				return err
			}

			state.FilesCopied++
		}
	}

	return nil
}

// migrateNotifications migrates notifications from JSON file.
func (mm *MigrationManager) migrateNotifications(state *MigrationState, storageEngine *StorageEngine) error {
	notificationsPath := filepath.Join(filepath.Dir(mm.legacyPath), "notifications.json")
	if _, err := os.Stat(notificationsPath); os.IsNotExist(err) {
		return nil // No notifications file
	}

	data, err := os.ReadFile(notificationsPath)
	if err != nil {
		return err
	}

	var notifications []struct {
		ID         string            `json:"id"`
		Type       string            `json:"type"`
		Title      string            `json:"title"`
		Message    string            `json:"message"`
		Timestamp  string            `json:"timestamp"`
		Read       bool              `json:"read"`
		SessionRef string            `json:"sessionRef,omitempty"`
		Metadata   map[string]string `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(data, &notifications); err != nil {
		return err
	}

	for _, notif := range notifications {
		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, notif.Timestamp)
		if err != nil {
			timestamp = time.Now() // Fallback
		}

		// Convert metadata to JSON string
		metadataStr := ""
		if notif.Metadata != nil {
			metadataBytes, err := json.Marshal(notif.Metadata)
			if err == nil {
				metadataStr = string(metadataBytes)
			}
		}

		// Create notification
		storageNotif := &Notification{
			ID:         notif.ID,
			Type:       notif.Type,
			Title:      notif.Title,
			Message:    notif.Message,
			Timestamp:  timestamp,
			Read:       notif.Read,
			SessionRef: notif.SessionRef,
			Metadata:   metadataStr,
		}

		if err := storageEngine.AddNotification(storageNotif); err != nil {
			return err
		}
	}

	return mm.AddCheckpoint(state, "notifications_migrated", 
		fmt.Sprintf("Migrated %d notifications", len(notifications)), true)
}

// generateEmbeddings generates embeddings for migrated sessions.
func (mm *MigrationManager) generateEmbeddings(state *MigrationState, storageEngine *StorageEngine) error {
	// Get all sessions
	sessions, _, err := storageEngine.ListSessions(1, 10000) // Get all sessions
	if err != nil {
		return err
	}

	for _, session := range sessions {
		// Generate text for embedding
		text := session.CustomSummary + " " + session.OriginalSummary + " " + session.ExtractedText
		if strings.TrimSpace(text) == "" {
			continue // Skip sessions with no text
		}

		// Queue embedding generation (async) - access through vector manager
		if err := storageEngine.vectorMgr.QueueEmbedding(session.ID, text); err != nil {
			// Log error but continue
			fmt.Printf("Warning: Failed to queue embedding for session %s: %v\n", session.Date, err)
		}
	}

	return mm.AddCheckpoint(state, "embeddings_queued", 
		fmt.Sprintf("Queued embeddings for %d sessions", len(sessions)), true)
}

// GetProgress returns the current migration progress as a percentage.
func (mm *MigrationManager) GetProgress(state *MigrationState) float64 {
	switch state.Status {
	case MigrationStatusIdle:
		return 0.0
	case MigrationStatusDetecting:
		return 10.0
	case MigrationStatusBackingUp:
		return 20.0
	case MigrationStatusMigrating:
		// Calculate based on migrated items
		total := state.SessionsMigrated + state.BlocksMigrated + state.FilesCopied
		if total == 0 {
			return 30.0
		}
		// Assume migration is 30-80% of total progress
		return 30.0 + (50.0 * float64(total) / 1000.0) // Rough estimate
	case MigrationStatusVerifying:
		return 90.0
	case MigrationStatusComplete:
		return 100.0
	case MigrationStatusFailed, MigrationStatusRollingBack:
		return float64(len(state.Checkpoints)) * 10.0 // Rough estimate based on checkpoints
	default:
		return 0.0
	}
}

// Rollback restores the system to its pre-migration state using the backup.
func (mm *MigrationManager) Rollback(state *MigrationState, storageEngine *StorageEngine) error {
	if state.BackupPath == "" {
		return NewStorageError(ErrValidation, "no backup path available for rollback", nil)
	}

	// Verify backup exists and is valid
	if _, err := os.Stat(state.BackupPath); os.IsNotExist(err) {
		return NewStorageError(ErrFileSystem, "backup directory not found", err)
	}

	// Close storage engine to release locks
	if err := storageEngine.Close(); err != nil {
		return NewStorageError(ErrDatabase, "failed to close storage engine", err)
	}

	// Remove current SQLite database
	dbPath := filepath.Join(storageEngine.config.DataDir, "waddle.db")
	if err := mm.removeIfExists(dbPath); err != nil {
		return err
	}

	// Remove current vector database
	vectorPath := filepath.Join(storageEngine.config.DataDir, "vectors")
	if err := mm.removeDirectoryIfExists(vectorPath); err != nil {
		return err
	}

	// Remove current files directory
	filesPath := filepath.Join(storageEngine.config.DataDir, "files")
	if err := mm.removeDirectoryIfExists(filesPath); err != nil {
		return err
	}

	// Restore legacy data from backup
	legacyParentDir := filepath.Dir(mm.legacyPath)
	if err := mm.removeDirectoryIfExists(mm.legacyPath); err != nil {
		return err
	}

	// Copy sessions back from backup
	backupSessionsPath := filepath.Join(state.BackupPath, "sessions")
	if err := mm.copyDirectory(backupSessionsPath, mm.legacyPath); err != nil {
		return NewStorageError(ErrFileSystem, "failed to restore sessions from backup", err)
	}

	// Copy notifications.json back from backup if it exists
	backupNotificationsPath := filepath.Join(state.BackupPath, "notifications.json")
	if _, err := os.Stat(backupNotificationsPath); err == nil {
		notificationsPath := filepath.Join(legacyParentDir, "notifications.json")
		if err := mm.copyFile(backupNotificationsPath, notificationsPath); err != nil {
			return NewStorageError(ErrFileSystem, "failed to restore notifications from backup", err)
		}
	}

	// Verify rollback integrity
	if err := mm.verifyRollback(state); err != nil {
		return err
	}

	return mm.AddCheckpoint(state, "rollback_complete", 
		fmt.Sprintf("Successfully rolled back from backup at %s", state.BackupPath), true)
}

// removeIfExists removes a file if it exists, ignoring not-found errors.
func (mm *MigrationManager) removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return NewStorageError(ErrFileSystem, fmt.Sprintf("failed to remove %s", path), err)
	}
	return nil
}

// removeDirectoryIfExists removes a directory if it exists, ignoring not-found errors.
func (mm *MigrationManager) removeDirectoryIfExists(path string) error {
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return NewStorageError(ErrFileSystem, fmt.Sprintf("failed to remove directory %s", path), err)
	}
	return nil
}

// verifyRollback verifies that the rollback was successful.
func (mm *MigrationManager) verifyRollback(state *MigrationState) error {
	// Check that legacy data was restored
	if _, err := os.Stat(mm.legacyPath); os.IsNotExist(err) {
		return NewStorageError(ErrValidation, "legacy data not restored after rollback", nil)
	}

	// Check that SQLite database was removed
	dbPath := filepath.Join(mm.config.DataDir, "waddle.db")
	if _, err := os.Stat(dbPath); err == nil {
		return NewStorageError(ErrValidation, "SQLite database still exists after rollback", nil)
	}

	// Verify file counts match backup
	originalCount, err := mm.countFiles(filepath.Join(state.BackupPath, "sessions"))
	if err != nil {
		return NewStorageError(ErrFileSystem, "failed to count backup files", err)
	}

	restoredCount, err := mm.countFiles(mm.legacyPath)
	if err != nil {
		return NewStorageError(ErrFileSystem, "failed to count restored files", err)
	}

	if originalCount != restoredCount {
		return NewStorageError(ErrValidation, 
			fmt.Sprintf("rollback file count mismatch: backup=%d, restored=%d", originalCount, restoredCount), nil)
	}

	return nil
}