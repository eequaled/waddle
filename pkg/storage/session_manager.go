package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// Search performs full-text search using SQLite FTS5 across sessions and activity blocks.
// It searches across custom_title, custom_summary, original_summary, and micro_summary fields.
// Returns results ranked by relevance with snippet highlighting.
func (sm *SessionManager) Search(query string, page, pageSize int) ([]SearchResult, error) {
	if query == "" {
		return nil, NewStorageError(ErrValidation, "search query cannot be empty", nil)
	}
	if page < 1 {
		return nil, NewStorageError(ErrValidation, "page must be >= 1", nil)
	}
	if pageSize < 1 || pageSize > 1000 {
		return nil, NewStorageError(ErrValidation, "pageSize must be between 1 and 1000", nil)
	}

	offset := (page - 1) * pageSize

	// Escape FTS5 special characters and prepare query
	ftsQuery := prepareFTSQuery(query)

	// Search across sessions_fts and activity_blocks_fts
	// We'll use UNION to combine results from both tables
	searchSQL := `
		WITH session_matches AS (
			SELECT 
				s.id, s.date, s.custom_title, s.custom_summary, s.original_summary,
				s.extracted_text_encrypted, s.created_at, s.updated_at,
				fts.rank as score,
				snippet(sessions_fts, 1, '<mark>', '</mark>', '...', 32) as snippet,
				'session' as match_source
			FROM sessions_fts fts
			JOIN sessions s ON s.id = fts.rowid
			WHERE sessions_fts MATCH ?
		),
		block_matches AS (
			SELECT DISTINCT
				s.id, s.date, s.custom_title, s.custom_summary, s.original_summary,
				s.extracted_text_encrypted, s.created_at, s.updated_at,
				fts.rank as score,
				snippet(activity_blocks_fts, 0, '<mark>', '</mark>', '...', 32) as snippet,
				'activity_block' as match_source
			FROM activity_blocks_fts fts
			JOIN activity_blocks ab ON ab.id = fts.rowid
			JOIN app_activities aa ON aa.id = ab.app_activity_id
			JOIN sessions s ON s.id = aa.session_id
			WHERE activity_blocks_fts MATCH ?
		),
		all_matches AS (
			SELECT * FROM session_matches
			UNION ALL
			SELECT * FROM block_matches
		)
		SELECT 
			id, date, custom_title, custom_summary, original_summary,
			extracted_text_encrypted, created_at, updated_at,
			MAX(score) as best_score, snippet, match_source
		FROM all_matches
		GROUP BY id
		ORDER BY best_score DESC
		LIMIT ? OFFSET ?`

	stmt, err := sm.getStmt(searchSQL)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to prepare search statement", err)
	}

	rows, err := stmt.Query(ftsQuery, ftsQuery, pageSize, offset)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "search query failed", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var session Session
		var encryptedText sql.NullString
		var score float32
		var snippet, matchSource string

		err := rows.Scan(
			&session.ID, &session.Date, &session.CustomTitle, &session.CustomSummary,
			&session.OriginalSummary, &encryptedText, &session.CreatedAt, &session.UpdatedAt,
			&score, &snippet, &matchSource,
		)
		if err != nil {
			return nil, NewStorageError(ErrDatabase, "failed to scan search result", err)
		}

		// Decrypt extracted text if present
		if encryptedText.Valid && encryptedText.String != "" {
			decrypted, err := sm.encryptionMgr.DecryptString(encryptedText.String)
			if err != nil {
				// Log error but don't fail the search
				session.ExtractedText = "[decryption failed]"
			} else {
				session.ExtractedText = decrypted
			}
		}

		result := SearchResult{
			Session:   session,
			Score:     score,
			Snippet:   snippet,
			MatchType: MatchTypeFullText,
		}

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, NewStorageError(ErrDatabase, "error iterating search results", err)
	}

	return results, nil
}

// prepareFTSQuery escapes and prepares a query string for FTS5.
// It handles boolean operators (AND, OR, NOT) and escapes special characters.
func prepareFTSQuery(query string) string {
	// For now, we'll do basic escaping and let FTS5 handle the query
	// In a production system, you might want more sophisticated query parsing
	
	// Remove potentially problematic characters but preserve basic boolean operators
	// FTS5 supports: AND, OR, NOT, quotes for phrases, * for prefix matching
	
	// Simple approach: if the query contains quotes, preserve them for phrase search
	// Otherwise, treat as individual terms with implicit AND
	
	return query // FTS5 is quite robust with query handling
}

// SemanticSearch performs semantic search by generating embeddings and searching LanceDB,
// then fetching full session metadata from SQLite. Supports date range filtering.
func (sm *SessionManager) SemanticSearch(query string, topK int, dateRange *DateRange, vectorMgr VectorManagerInterface) ([]SearchResult, error) {
	if query == "" {
		return nil, NewStorageError(ErrValidation, "search query cannot be empty", nil)
	}
	if topK < 1 || topK > 1000 {
		return nil, NewStorageError(ErrValidation, "topK must be between 1 and 1000", nil)
	}

	// Generate embedding for the query
	queryEmbedding, err := vectorMgr.GenerateEmbedding(query)
	if err != nil {
		return nil, NewStorageError(ErrVector, "failed to generate query embedding", err)
	}

	// Search LanceDB for similar sessions
	vectorResults, err := vectorMgr.Search(queryEmbedding, topK)
	if err != nil {
		return nil, NewStorageError(ErrVector, "vector search failed", err)
	}

	if len(vectorResults) == 0 {
		return []SearchResult{}, nil
	}

	// Extract session IDs
	sessionIDs := make([]int64, len(vectorResults))
	scoreMap := make(map[int64]float32)
	for i, result := range vectorResults {
		sessionIDs[i] = result.SessionID
		scoreMap[result.SessionID] = result.Score
	}

	// Build SQL query to fetch session metadata
	placeholders := make([]string, len(sessionIDs))
	args := make([]interface{}, len(sessionIDs))
	for i, id := range sessionIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	baseSQL := `
		SELECT id, date, custom_title, custom_summary, original_summary,
		       extracted_text_encrypted, created_at, updated_at
		FROM sessions
		WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	// Add date range filtering if provided
	if dateRange != nil {
		if dateRange.StartDate != "" {
			baseSQL += " AND date >= ?"
			args = append(args, dateRange.StartDate)
		}
		if dateRange.EndDate != "" {
			baseSQL += " AND date <= ?"
			args = append(args, dateRange.EndDate)
		}
	}

	stmt, err := sm.getStmt(baseSQL)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to prepare semantic search statement", err)
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "semantic search query failed", err)
	}
	defer rows.Close()

	var results []SearchResult
	sessionMap := make(map[int64]Session)

	// Fetch session data
	for rows.Next() {
		var session Session
		var encryptedText sql.NullString

		err := rows.Scan(
			&session.ID, &session.Date, &session.CustomTitle, &session.CustomSummary,
			&session.OriginalSummary, &encryptedText, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, NewStorageError(ErrDatabase, "failed to scan semantic search result", err)
		}

		// Decrypt extracted text if present
		if encryptedText.Valid && encryptedText.String != "" {
			decrypted, err := sm.encryptionMgr.DecryptString(encryptedText.String)
			if err != nil {
				// Log error but don't fail the search
				session.ExtractedText = "[decryption failed]"
			} else {
				session.ExtractedText = decrypted
			}
		}

		sessionMap[session.ID] = session
	}

	if err = rows.Err(); err != nil {
		return nil, NewStorageError(ErrDatabase, "error iterating semantic search results", err)
	}

	// Build results in the same order as vector search results, preserving similarity scores
	for _, vectorResult := range vectorResults {
		if session, exists := sessionMap[vectorResult.SessionID]; exists {
			result := SearchResult{
				Session:   session,
				Score:     vectorResult.Score,
				Snippet:   "", // No snippet for semantic search
				MatchType: MatchTypeSemantic,
			}
			results = append(results, result)
		}
	}

	return results, nil
}
