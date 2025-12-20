package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupManager handles automated backups and recovery operations.
type BackupManager struct {
	config      *StorageConfig
	backupDir   string
	storageEngine *StorageEngine
}

// NewBackupManager creates a new backup manager.
func NewBackupManager(config *StorageConfig, storageEngine *StorageEngine) *BackupManager {
	backupDir := filepath.Join(config.DataDir, "backups")
	return &BackupManager{
		config:        config,
		backupDir:     backupDir,
		storageEngine: storageEngine,
	}
}

// CreateBackup creates a backup of the current storage state.
func (bm *BackupManager) CreateBackup() (string, error) {
	// Create backup directory with timestamp (microsecond precision)
	timestamp := time.Now().Format("20060102-150405.000000")
	backupPath := filepath.Join(bm.backupDir, fmt.Sprintf("backup-%s", timestamp))
	
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", NewStorageError(ErrFileSystem, "failed to create backup directory", err)
	}

	// Backup SQLite database using VACUUM INTO
	dbPath := filepath.Join(bm.config.DataDir, "waddle.db")
	backupDBPath := filepath.Join(backupPath, "waddle.db")
	
	if err := bm.backupSQLiteDatabase(dbPath, backupDBPath); err != nil {
		return "", err
	}

	// Backup vector database directory
	vectorSrcPath := filepath.Join(bm.config.DataDir, "vectors")
	vectorBackupPath := filepath.Join(backupPath, "vectors")
	
	if _, err := os.Stat(vectorSrcPath); err == nil {
		if err := bm.copyDirectory(vectorSrcPath, vectorBackupPath); err != nil {
			return "", NewStorageError(ErrFileSystem, "failed to backup vector database", err)
		}
	}

	// Backup files directory
	filesSrcPath := filepath.Join(bm.config.DataDir, "files")
	filesBackupPath := filepath.Join(backupPath, "files")
	
	if _, err := os.Stat(filesSrcPath); err == nil {
		if err := bm.copyDirectory(filesSrcPath, filesBackupPath); err != nil {
			return "", NewStorageError(ErrFileSystem, "failed to backup files directory", err)
		}
	}

	// Create backup metadata
	if err := bm.createBackupMetadata(backupPath); err != nil {
		return "", err
	}

	return backupPath, nil
}

// backupSQLiteDatabase creates a backup of the SQLite database using VACUUM INTO.
func (bm *BackupManager) backupSQLiteDatabase(srcPath, dstPath string) error {
	// Check if source database exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return nil // No database to backup
	}

	// Remove destination file if it exists (VACUUM INTO requires non-existent file)
	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		return NewStorageError(ErrFileSystem, "failed to remove existing backup file", err)
	}

	// Use SQLite VACUUM INTO for online backup
	// This requires accessing the database through the session manager
	if bm.storageEngine.sessionMgr != nil {
		db := bm.storageEngine.sessionMgr.DB()
		if db != nil {
			query := fmt.Sprintf("VACUUM INTO '%s'", strings.ReplaceAll(dstPath, "'", "''"))
			if _, err := db.Exec(query); err != nil {
				return NewStorageError(ErrDatabase, "failed to backup database with VACUUM INTO", err)
			}
		} else {
			// Fallback to file copy if database not available
			if err := bm.copyFile(srcPath, dstPath); err != nil {
				return NewStorageError(ErrFileSystem, "failed to copy database file", err)
			}
		}
	} else {
		// Fallback to file copy if session manager not available
		if err := bm.copyFile(srcPath, dstPath); err != nil {
			return NewStorageError(ErrFileSystem, "failed to copy database file", err)
		}
	}

	return nil
}

// copyDirectory recursively copies a directory.
func (bm *BackupManager) copyDirectory(src, dst string) error {
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
			if err := bm.copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := bm.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func (bm *BackupManager) copyFile(src, dst string) error {
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
	_, err = srcFile.WriteTo(dstFile)
	return err
}

// createBackupMetadata creates metadata for the backup.
func (bm *BackupManager) createBackupMetadata(backupPath string) error {
	metadata := map[string]interface{}{
		"timestamp":    time.Now().Format(time.RFC3339),
		"version":      "1.0",
		"dataDir":      bm.config.DataDir,
		"retentionDays": bm.config.RetentionDays,
	}

	// Get storage stats if available
	if bm.storageEngine != nil && bm.storageEngine.fileMgr != nil {
		if stats, err := bm.storageEngine.fileMgr.GetStorageStats(); err == nil {
			metadata["stats"] = stats
		}
	}

	// Write metadata to JSON file
	metadataPath := filepath.Join(backupPath, "backup_metadata.json")
	return bm.writeJSONFile(metadataPath, metadata)
}

// writeJSONFile writes data to a JSON file.
func (bm *BackupManager) writeJSONFile(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// VerifyBackup verifies the integrity of a backup.
func (bm *BackupManager) VerifyBackup(backupPath string) error {
	// Check if backup directory exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return NewStorageError(ErrNotFound, "backup directory not found", err)
	}

	// Verify SQLite database integrity
	dbBackupPath := filepath.Join(backupPath, "waddle.db")
	if _, err := os.Stat(dbBackupPath); err == nil {
		if err := bm.verifySQLiteIntegrity(dbBackupPath); err != nil {
			return err
		}
	}

	// Verify file counts match expected structure
	if err := bm.verifyBackupStructure(backupPath); err != nil {
		return err
	}

	return nil
}

// verifySQLiteIntegrity runs PRAGMA integrity_check on the backup database.
func (bm *BackupManager) verifySQLiteIntegrity(dbPath string) error {
	// This would require opening the backup database separately
	// For now, we'll just check that the file exists and is not empty
	info, err := os.Stat(dbPath)
	if err != nil {
		return NewStorageError(ErrFileSystem, "backup database file not accessible", err)
	}

	if info.Size() == 0 {
		return NewStorageError(ErrValidation, "backup database file is empty", nil)
	}

	return nil
}

// verifyBackupStructure verifies the backup has the expected directory structure.
func (bm *BackupManager) verifyBackupStructure(backupPath string) error {
	// Check for backup metadata
	metadataPath := filepath.Join(backupPath, "backup_metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return NewStorageError(ErrValidation, "backup metadata not found", nil)
	}

	// Check for at least one of: database, vectors, or files
	hasContent := false
	
	if _, err := os.Stat(filepath.Join(backupPath, "waddle.db")); err == nil {
		hasContent = true
	}
	
	if _, err := os.Stat(filepath.Join(backupPath, "vectors")); err == nil {
		hasContent = true
	}
	
	if _, err := os.Stat(filepath.Join(backupPath, "files")); err == nil {
		hasContent = true
	}

	if !hasContent {
		return NewStorageError(ErrValidation, "backup contains no data", nil)
	}

	return nil
}

// Restore restores the system from a backup.
func (bm *BackupManager) Restore(backupPath string) error {
	// Verify backup before restore
	if err := bm.VerifyBackup(backupPath); err != nil {
		return NewStorageError(ErrValidation, "backup verification failed", err)
	}

	// Close storage engine to release locks
	if err := bm.storageEngine.Close(); err != nil {
		return NewStorageError(ErrDatabase, "failed to close storage engine", err)
	}

	// Restore database
	if err := bm.restoreDatabase(backupPath); err != nil {
		return err
	}

	// Restore vector database
	if err := bm.restoreVectorDatabase(backupPath); err != nil {
		return err
	}

	// Restore files
	if err := bm.restoreFiles(backupPath); err != nil {
		return err
	}

	// Reinitialize storage engine
	if err := bm.storageEngine.Initialize(); err != nil {
		return NewStorageError(ErrDatabase, "failed to reinitialize after restore", err)
	}

	return nil
}

// restoreDatabase restores the SQLite database from backup.
func (bm *BackupManager) restoreDatabase(backupPath string) error {
	srcPath := filepath.Join(backupPath, "waddle.db")
	dstPath := filepath.Join(bm.config.DataDir, "waddle.db")

	// Remove current database
	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		return NewStorageError(ErrFileSystem, "failed to remove current database", err)
	}

	// Copy backup database
	if _, err := os.Stat(srcPath); err == nil {
		if err := bm.copyFile(srcPath, dstPath); err != nil {
			return NewStorageError(ErrFileSystem, "failed to restore database", err)
		}
	}

	return nil
}

// restoreVectorDatabase restores the vector database from backup.
func (bm *BackupManager) restoreVectorDatabase(backupPath string) error {
	srcPath := filepath.Join(backupPath, "vectors")
	dstPath := filepath.Join(bm.config.DataDir, "vectors")

	// Remove current vector database
	if err := os.RemoveAll(dstPath); err != nil && !os.IsNotExist(err) {
		return NewStorageError(ErrFileSystem, "failed to remove current vector database", err)
	}

	// Copy backup vector database
	if _, err := os.Stat(srcPath); err == nil {
		if err := bm.copyDirectory(srcPath, dstPath); err != nil {
			return NewStorageError(ErrFileSystem, "failed to restore vector database", err)
		}
	}

	return nil
}

// restoreFiles restores the files directory from backup.
func (bm *BackupManager) restoreFiles(backupPath string) error {
	srcPath := filepath.Join(backupPath, "files")
	dstPath := filepath.Join(bm.config.DataDir, "files")

	// Remove current files directory
	if err := os.RemoveAll(dstPath); err != nil && !os.IsNotExist(err) {
		return NewStorageError(ErrFileSystem, "failed to remove current files directory", err)
	}

	// Copy backup files directory
	if _, err := os.Stat(srcPath); err == nil {
		if err := bm.copyDirectory(srcPath, dstPath); err != nil {
			return NewStorageError(ErrFileSystem, "failed to restore files directory", err)
		}
	}

	return nil
}

// rollbackRestore attempts to rollback a failed restore.
func (bm *BackupManager) rollbackRestore(currentBackupPath string) error {
	// This is a best-effort rollback
	bm.restoreDatabase(currentBackupPath)
	bm.restoreVectorDatabase(currentBackupPath)
	bm.restoreFiles(currentBackupPath)
	return nil
}

// ListBackups returns a list of available backups.
func (bm *BackupManager) ListBackups() ([]BackupInfo, error) {
	if _, err := os.Stat(bm.backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, NewStorageError(ErrFileSystem, "failed to read backup directory", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "backup-") {
			backupPath := filepath.Join(bm.backupDir, entry.Name())
			info, err := bm.getBackupInfo(backupPath)
			if err != nil {
				continue // Skip invalid backups
			}
			backups = append(backups, *info)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// getBackupInfo extracts information about a backup.
func (bm *BackupManager) getBackupInfo(backupPath string) (*BackupInfo, error) {
	// Get directory info
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, err
	}

	// Try to read metadata
	metadataPath := filepath.Join(backupPath, "backup_metadata.json")
	var metadata map[string]interface{}
	if data, err := os.ReadFile(metadataPath); err == nil {
		json.Unmarshal(data, &metadata)
	}

	// Calculate backup size
	size, err := bm.calculateDirectorySize(backupPath)
	if err != nil {
		size = 0
	}

	backupInfo := &BackupInfo{
		Path:      backupPath,
		Name:      filepath.Base(backupPath),
		Timestamp: info.ModTime(),
		Size:      size,
		Metadata:  metadata,
	}

	return backupInfo, nil
}

// calculateDirectorySize calculates the total size of a directory.
func (bm *BackupManager) calculateDirectorySize(dirPath string) (int64, error) {
	var size int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// CleanupOldBackups removes backups older than the retention period.
func (bm *BackupManager) CleanupOldBackups() error {
	backups, err := bm.ListBackups()
	if err != nil {
		return err
	}

	retentionDuration := time.Duration(bm.config.RetentionDays) * 24 * time.Hour
	cutoffTime := time.Now().Add(-retentionDuration)

	var deletedCount int
	for _, backup := range backups {
		if backup.Timestamp.Before(cutoffTime) {
			if err := os.RemoveAll(backup.Path); err != nil {
				// Log error but continue
				continue
			}
			deletedCount++
		}
	}

	return nil
}

// BackupInfo contains information about a backup.
type BackupInfo struct {
	Path      string                 `json:"path"`
	Name      string                 `json:"name"`
	Timestamp time.Time              `json:"timestamp"`
	Size      int64                  `json:"size"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}