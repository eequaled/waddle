package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// Migration represents a database schema migration.
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// migrations contains all database migrations in order.
var migrations = []Migration{
	{
		Version:     1,
		Description: "Initial schema with sessions, app_activities, activity_blocks, chats, notifications",
		SQL: `
-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT,
    checksum TEXT
);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT UNIQUE NOT NULL,
    custom_title TEXT,
    custom_summary TEXT,
    original_summary TEXT,
    extracted_text_encrypted BLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_date ON sessions(date);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);

-- App activities table
CREATE TABLE IF NOT EXISTS app_activities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    app_name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    UNIQUE(session_id, app_name)
);

CREATE INDEX IF NOT EXISTS idx_app_activities_session ON app_activities(session_id);
CREATE INDEX IF NOT EXISTS idx_app_activities_app_name ON app_activities(app_name);

-- Activity blocks table
CREATE TABLE IF NOT EXISTS activity_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_activity_id INTEGER NOT NULL,
    block_id TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    ocr_text_encrypted BLOB,
    micro_summary TEXT,
    FOREIGN KEY (app_activity_id) REFERENCES app_activities(id) ON DELETE CASCADE,
    UNIQUE(app_activity_id, block_id)
);

CREATE INDEX IF NOT EXISTS idx_activity_blocks_app_activity ON activity_blocks(app_activity_id);
CREATE INDEX IF NOT EXISTS idx_activity_blocks_start_time ON activity_blocks(start_time);

-- Chats table
CREATE TABLE IF NOT EXISTS chats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('user', 'assistant')),
    content_encrypted BLOB NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chats_session ON chats(session_id);
CREATE INDEX IF NOT EXISTS idx_chats_timestamp ON chats(timestamp);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    read INTEGER DEFAULT 0,
    session_ref TEXT,
    metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_notifications_timestamp ON notifications(timestamp);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);

-- Manual notes table
CREATE TABLE IF NOT EXISTS manual_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_manual_notes_session ON manual_notes(session_id);
`,
	},
	{
		Version:     2,
		Description: "Add FTS5 virtual tables for full-text search",
		SQL: `
-- FTS5 virtual tables for full-text search on non-encrypted fields
CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(
    date,
    custom_title,
    custom_summary,
    original_summary,
    content='sessions',
    content_rowid='id'
);

CREATE VIRTUAL TABLE IF NOT EXISTS activity_blocks_fts USING fts5(
    micro_summary,
    content='activity_blocks',
    content_rowid='id'
);

-- Triggers to keep sessions_fts in sync
CREATE TRIGGER IF NOT EXISTS sessions_ai AFTER INSERT ON sessions BEGIN
    INSERT INTO sessions_fts(rowid, date, custom_title, custom_summary, original_summary)
    VALUES (new.id, new.date, new.custom_title, new.custom_summary, new.original_summary);
END;

CREATE TRIGGER IF NOT EXISTS sessions_ad AFTER DELETE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, date, custom_title, custom_summary, original_summary)
    VALUES ('delete', old.id, old.date, old.custom_title, old.custom_summary, old.original_summary);
END;

CREATE TRIGGER IF NOT EXISTS sessions_au AFTER UPDATE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, date, custom_title, custom_summary, original_summary)
    VALUES ('delete', old.id, old.date, old.custom_title, old.custom_summary, old.original_summary);
    INSERT INTO sessions_fts(rowid, date, custom_title, custom_summary, original_summary)
    VALUES (new.id, new.date, new.custom_title, new.custom_summary, new.original_summary);
END;

-- Triggers to keep activity_blocks_fts in sync
CREATE TRIGGER IF NOT EXISTS activity_blocks_ai AFTER INSERT ON activity_blocks BEGIN
    INSERT INTO activity_blocks_fts(rowid, micro_summary)
    VALUES (new.id, new.micro_summary);
END;

CREATE TRIGGER IF NOT EXISTS activity_blocks_ad AFTER DELETE ON activity_blocks BEGIN
    INSERT INTO activity_blocks_fts(activity_blocks_fts, rowid, micro_summary)
    VALUES ('delete', old.id, old.micro_summary);
END;

CREATE TRIGGER IF NOT EXISTS activity_blocks_au AFTER UPDATE ON activity_blocks BEGIN
    INSERT INTO activity_blocks_fts(activity_blocks_fts, rowid, micro_summary)
    VALUES ('delete', old.id, old.micro_summary);
    INSERT INTO activity_blocks_fts(rowid, micro_summary)
    VALUES (new.id, new.micro_summary);
END;
`,
	},
	{
		Version:     3,
		Description: "Add synthesis columns to sessions and capture columns to activity_blocks, create knowledge_cards table",
		SQL: `
-- Add synthesis columns to sessions table
ALTER TABLE sessions ADD COLUMN entities_json TEXT DEFAULT '[]';
ALTER TABLE sessions ADD COLUMN synthesis_status TEXT DEFAULT 'pending';
ALTER TABLE sessions ADD COLUMN ai_summary TEXT DEFAULT '';
ALTER TABLE sessions ADD COLUMN ai_bullets TEXT DEFAULT '[]';

-- Add capture columns to activity_blocks table
ALTER TABLE activity_blocks ADD COLUMN capture_source TEXT DEFAULT 'polling_ocr';
ALTER TABLE activity_blocks ADD COLUMN structured_metadata TEXT DEFAULT '{}';

-- Create knowledge_cards table
CREATE TABLE IF NOT EXISTS knowledge_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    bullets TEXT NOT NULL, -- JSON array of 3 bullet points
    entities TEXT NOT NULL, -- JSON array of extracted entities
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'completed', 'failed')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_knowledge_cards_session ON knowledge_cards(session_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_cards_status ON knowledge_cards(status);

-- Update FTS5 triggers to include new synthesis columns
DROP TRIGGER IF EXISTS sessions_ai;
DROP TRIGGER IF EXISTS sessions_ad;
DROP TRIGGER IF EXISTS sessions_au;

-- Recreate FTS5 virtual table with new columns
DROP TABLE IF EXISTS sessions_fts;
CREATE VIRTUAL TABLE sessions_fts USING fts5(
    date,
    custom_title,
    custom_summary,
    original_summary,
    ai_summary,
    entities_json,
    content='sessions',
    content_rowid='id'
);

-- Recreate triggers with new columns
CREATE TRIGGER sessions_ai AFTER INSERT ON sessions BEGIN
    INSERT INTO sessions_fts(rowid, date, custom_title, custom_summary, original_summary, ai_summary, entities_json)
    VALUES (new.id, new.date, new.custom_title, new.custom_summary, new.original_summary, new.ai_summary, new.entities_json);
END;

CREATE TRIGGER sessions_ad AFTER DELETE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, date, custom_title, custom_summary, original_summary, ai_summary, entities_json)
    VALUES ('delete', old.id, old.date, old.custom_title, old.custom_summary, old.original_summary, old.ai_summary, old.entities_json);
END;

CREATE TRIGGER sessions_au AFTER UPDATE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, date, custom_title, custom_summary, original_summary, ai_summary, entities_json)
    VALUES ('delete', old.id, old.date, old.custom_title, old.custom_summary, old.original_summary, old.ai_summary, old.entities_json);
    INSERT INTO sessions_fts(rowid, date, custom_title, custom_summary, original_summary, ai_summary, entities_json)
    VALUES (new.id, new.date, new.custom_title, new.custom_summary, new.original_summary, new.ai_summary, new.entities_json);
END;

-- Update activity_blocks FTS5 to include structured metadata
DROP TRIGGER IF EXISTS activity_blocks_ai;
DROP TRIGGER IF EXISTS activity_blocks_ad;
DROP TRIGGER IF EXISTS activity_blocks_au;

DROP TABLE IF EXISTS activity_blocks_fts;
CREATE VIRTUAL TABLE activity_blocks_fts USING fts5(
    micro_summary,
    structured_metadata,
    content='activity_blocks',
    content_rowid='id'
);

CREATE TRIGGER activity_blocks_ai AFTER INSERT ON activity_blocks BEGIN
    INSERT INTO activity_blocks_fts(rowid, micro_summary, structured_metadata)
    VALUES (new.id, new.micro_summary, new.structured_metadata);
END;

CREATE TRIGGER activity_blocks_ad AFTER DELETE ON activity_blocks BEGIN
    INSERT INTO activity_blocks_fts(activity_blocks_fts, rowid, micro_summary, structured_metadata)
    VALUES ('delete', old.id, old.micro_summary, old.structured_metadata);
END;

CREATE TRIGGER activity_blocks_au AFTER UPDATE ON activity_blocks BEGIN
    INSERT INTO activity_blocks_fts(activity_blocks_fts, rowid, micro_summary, structured_metadata)
    VALUES ('delete', old.id, old.micro_summary, old.structured_metadata);
    INSERT INTO activity_blocks_fts(rowid, micro_summary, structured_metadata)
    VALUES (new.id, new.micro_summary, new.structured_metadata);
END;
`,
	},
}

// runMigrations runs all pending database migrations.
func (sm *SessionManager) runMigrations() error {
	// Get current schema version
	currentVersion := sm.getCurrentSchemaVersion()

	for _, migration := range migrations {
		if migration.Version > currentVersion {
			if err := sm.runMigration(migration); err != nil {
				return err
			}
		}
	}

	return nil
}

// getCurrentSchemaVersion returns the current schema version from the database.
func (sm *SessionManager) getCurrentSchemaVersion() int {
	// Check if schema_version table exists
	var tableName string
	err := sm.db.QueryRow(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='schema_version'
	`).Scan(&tableName)

	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		return 0
	}

	// Get the latest version
	var version int
	err = sm.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		return 0
	}

	return version
}

// runMigration executes a single migration within a transaction.
func (sm *SessionManager) runMigration(migration Migration) error {
	tx, err := sm.db.Begin()
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to begin migration transaction", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	_, err = tx.Exec(migration.SQL)
	if err != nil {
		return NewStorageError(ErrDatabase, fmt.Sprintf("migration %d failed: %s", migration.Version, migration.Description), err)
	}

	// Calculate checksum
	checksum := calculateChecksum(migration.SQL)

	// Record migration
	_, err = tx.Exec(`
		INSERT INTO schema_version (version, description, checksum)
		VALUES (?, ?, ?)
	`, migration.Version, migration.Description, checksum)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to record migration", err)
	}

	if err := tx.Commit(); err != nil {
		return NewStorageError(ErrDatabase, "failed to commit migration", err)
	}

	return nil
}

// calculateChecksum calculates SHA256 checksum of the migration SQL.
func calculateChecksum(sql string) string {
	hash := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(hash[:])
}

// GetSchemaVersion returns the current schema version.
func (sm *SessionManager) GetSchemaVersion() (int, error) {
	var version int
	err := sm.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		return 0, NewStorageError(ErrDatabase, "failed to get schema version", err)
	}
	return version, nil
}
