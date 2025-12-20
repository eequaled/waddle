package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SessionManager handles all SQLite database operations including CRUD and FTS5 search.
type SessionManager struct {
	db            *sql.DB
	encryptionMgr *EncryptionManager
	stmtCache     map[string]*sql.Stmt
	stmtMutex     sync.RWMutex
	dbPath        string
}

// NewSessionManager creates a new SessionManager instance.
func NewSessionManager(dataDir string, encryptionMgr *EncryptionManager) *SessionManager {
	dbPath := filepath.Join(dataDir, "waddle.db")
	return &SessionManager{
		encryptionMgr: encryptionMgr,
		stmtCache:     make(map[string]*sql.Stmt),
		dbPath:        dbPath,
	}
}

// Initialize creates the database and runs migrations.
func (sm *SessionManager) Initialize() error {
	// Ensure directory exists
	dir := filepath.Dir(sm.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create database directory", err)
	}

	// Open database with WAL mode and foreign keys enabled
	db, err := sql.Open("sqlite", sm.dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to open database", err)
	}

	sm.db = db

	// Configure connection pool
	sm.db.SetMaxOpenConns(1) // SQLite works best with single writer
	sm.db.SetMaxIdleConns(1)
	sm.db.SetConnMaxLifetime(time.Hour)

	// Run integrity check on startup
	if err := sm.RunIntegrityCheck(); err != nil {
		return err
	}

	// Run migrations
	if err := sm.runMigrations(); err != nil {
		return err
	}

	return nil
}

// RunIntegrityCheck runs PRAGMA integrity_check on the database.
func (sm *SessionManager) RunIntegrityCheck() error {
	var result string
	err := sm.db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return NewStorageError(ErrDatabase, "integrity check failed", err)
	}

	if result != "ok" {
		return NewStorageError(ErrDatabase, fmt.Sprintf("database corruption detected: %s", result), nil)
	}

	return nil
}

// Vacuum runs VACUUM to optimize the database.
func (sm *SessionManager) Vacuum() error {
	_, err := sm.db.Exec("VACUUM")
	if err != nil {
		return NewStorageError(ErrDatabase, "vacuum failed", err)
	}
	return nil
}

// Close closes the database connection and all prepared statements.
func (sm *SessionManager) Close() error {
	sm.stmtMutex.Lock()
	defer sm.stmtMutex.Unlock()

	// Close all prepared statements
	for _, stmt := range sm.stmtCache {
		stmt.Close()
	}
	sm.stmtCache = make(map[string]*sql.Stmt)

	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}

// getStmt returns a cached prepared statement or creates a new one.
func (sm *SessionManager) getStmt(query string) (*sql.Stmt, error) {
	sm.stmtMutex.RLock()
	stmt, ok := sm.stmtCache[query]
	sm.stmtMutex.RUnlock()

	if ok {
		return stmt, nil
	}

	sm.stmtMutex.Lock()
	defer sm.stmtMutex.Unlock()

	// Double-check after acquiring write lock
	if stmt, ok := sm.stmtCache[query]; ok {
		return stmt, nil
	}

	stmt, err := sm.db.Prepare(query)
	if err != nil {
		return nil, err
	}

	sm.stmtCache[query] = stmt
	return stmt, nil
}

// DB returns the underlying database connection for advanced operations.
func (sm *SessionManager) DB() *sql.DB {
	return sm.db
}
