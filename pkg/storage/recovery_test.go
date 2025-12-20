package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestRecoveryManagerCreation tests basic recovery manager creation
func TestRecoveryManagerCreation(t *testing.T) {
	sqlitePath := "test.db"
	lanceDBPath := "test_lancedb"

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	if rm.sqlitePath != sqlitePath {
		t.Errorf("Expected SQLite path %s, got %s", sqlitePath, rm.sqlitePath)
	}

	if rm.lanceDBPath != lanceDBPath {
		t.Errorf("Expected LanceDB path %s, got %s", lanceDBPath, rm.lanceDBPath)
	}
}

// TestCorruptionTypeString tests CorruptionType string representation
func TestCorruptionTypeString(t *testing.T) {
	tests := []struct {
		corruption CorruptionType
		expected   string
	}{
		{CorruptionNone, "none"},
		{CorruptionSQLite, "sqlite"},
		{CorruptionLanceDB, "lancedb"},
		{CorruptionBoth, "both"},
		{CorruptionType(999), "unknown"},
	}

	for _, test := range tests {
		result := test.corruption.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

// TestValidateDatabasesNonExistent tests validation with non-existent databases
func TestValidateDatabasesNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "nonexistent.db")
	lanceDBPath := filepath.Join(tempDir, "nonexistent_lancedb")

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	result, err := rm.ValidateDatabases()
	if err != nil {
		t.Fatalf("ValidateDatabases should not return error: %v", err)
	}

	if result.SQLiteValid {
		t.Errorf("SQLite should be invalid for non-existent database")
	}

	if result.LanceDBValid {
		t.Errorf("LanceDB should be invalid for non-existent database")
	}

	if result.Corruption != CorruptionBoth {
		t.Errorf("Expected corruption type %s, got %s", CorruptionBoth, result.Corruption)
	}
}

// TestValidateSQLiteValid tests SQLite validation with valid database
func TestValidateSQLiteValid(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	// Create valid SQLite database
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		t.Fatalf("Failed to create SQLite database: %v", err)
	}

	// Create required tables
	schema := `
		CREATE TABLE sessions (
			id INTEGER PRIMARY KEY,
			title TEXT
		);
		CREATE TABLE activity_blocks (
			id INTEGER PRIMARY KEY,
			session_id INTEGER
		);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	db.Close()

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	// Test SQLite validation
	valid, err := rm.validateSQLite()
	if err != nil {
		t.Errorf("SQLite validation should succeed: %v", err)
	}

	if !valid {
		t.Errorf("SQLite should be valid")
	}
}

// TestValidateLanceDBValid tests LanceDB validation with valid structure
func TestValidateLanceDBValid(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	// Create valid LanceDB structure
	err = os.MkdirAll(lanceDBPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create LanceDB directory: %v", err)
	}

	err = os.MkdirAll(filepath.Join(lanceDBPath, "_metadata"), 0755)
	if err != nil {
		t.Fatalf("Failed to create metadata directory: %v", err)
	}

	err = os.MkdirAll(filepath.Join(lanceDBPath, "_versions"), 0755)
	if err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	// Test LanceDB validation
	valid, err := rm.validateLanceDB()
	if err != nil {
		t.Errorf("LanceDB validation should succeed: %v", err)
	}

	if !valid {
		t.Errorf("LanceDB should be valid")
	}
}

// TestRecoverFromCorruptionNone tests recovery when no corruption exists
func TestRecoverFromCorruptionNone(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	status, err := rm.RecoverFromCorruption(CorruptionNone)
	if err != nil {
		t.Errorf("Recovery should succeed for no corruption: %v", err)
	}

	if !status.Success {
		t.Errorf("Recovery should be successful")
	}

	if status.Type != CorruptionNone {
		t.Errorf("Expected corruption type %s, got %s", CorruptionNone, status.Type)
	}

	if status.Progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", status.Progress)
	}
}

// TestRecoverSQLite tests SQLite recovery
func TestRecoverSQLite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	// Create corrupted SQLite file (empty file)
	file, err := os.Create(sqlitePath)
	if err != nil {
		t.Fatalf("Failed to create corrupted SQLite file: %v", err)
	}
	file.Close()

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	status, err := rm.RecoverFromCorruption(CorruptionSQLite)
	if err != nil {
		t.Errorf("SQLite recovery should succeed: %v", err)
	}

	if !status.Success {
		t.Errorf("SQLite recovery should be successful")
	}

	if status.Type != CorruptionSQLite {
		t.Errorf("Expected corruption type %s, got %s", CorruptionSQLite, status.Type)
	}

	// Verify new database was created
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		t.Errorf("Failed to open recovered SQLite database: %v", err)
	}
	defer db.Close()

	// Verify tables exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query sessions table: %v", err)
	}
	if count != 1 {
		t.Errorf("Sessions table should exist after recovery")
	}
}

// TestRecoverLanceDB tests LanceDB recovery
func TestRecoverLanceDB(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	// Create corrupted LanceDB directory (empty directory)
	err = os.MkdirAll(lanceDBPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create corrupted LanceDB directory: %v", err)
	}

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	status, err := rm.RecoverFromCorruption(CorruptionLanceDB)
	if err != nil {
		t.Errorf("LanceDB recovery should succeed: %v", err)
	}

	if !status.Success {
		t.Errorf("LanceDB recovery should be successful")
	}

	if status.Type != CorruptionLanceDB {
		t.Errorf("Expected corruption type %s, got %s", CorruptionLanceDB, status.Type)
	}

	// Verify directories were created
	metadataPath := filepath.Join(lanceDBPath, "_metadata")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Errorf("Metadata directory should exist after recovery")
	}

	versionsPath := filepath.Join(lanceDBPath, "_versions")
	if _, err := os.Stat(versionsPath); os.IsNotExist(err) {
		t.Errorf("Versions directory should exist after recovery")
	}
}

// TestRebuildVectorsFromSessions tests vector rebuilding
func TestRebuildVectorsFromSessions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	// Create SQLite database with test data
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		t.Fatalf("Failed to create SQLite database: %v", err)
	}

	// Create schema and insert test data
	schema := `
		CREATE TABLE sessions (
			id INTEGER PRIMARY KEY,
			title TEXT,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Insert test sessions
	for i := 1; i <= 5; i++ {
		_, err = db.Exec("INSERT INTO sessions (title, summary) VALUES (?, ?)",
			fmt.Sprintf("Session %d", i),
			fmt.Sprintf("Summary for session %d", i))
		if err != nil {
			t.Fatalf("Failed to insert test session: %v", err)
		}
	}
	db.Close()

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	status, err := rm.RebuildVectorsFromSessions()
	if err != nil {
		t.Errorf("Vector rebuilding should succeed: %v", err)
	}

	if !status.Success {
		t.Errorf("Vector rebuilding should be successful")
	}

	if status.SessionsFound != 5 {
		t.Errorf("Expected 5 sessions found, got %d", status.SessionsFound)
	}

	if status.VectorsRebuilt != 5 {
		t.Errorf("Expected 5 vectors rebuilt, got %d", status.VectorsRebuilt)
	}

	if status.Progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", status.Progress)
	}
}

// TestGetRecoveryStats tests recovery statistics
func TestGetRecoveryStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	// Create test files
	file, err := os.Create(sqlitePath)
	if err != nil {
		t.Fatalf("Failed to create SQLite file: %v", err)
	}
	file.WriteString("test data")
	file.Close()

	err = os.MkdirAll(lanceDBPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create LanceDB directory: %v", err)
	}

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	stats := rm.GetRecoveryStats()

	if _, exists := stats["sqlite_size_bytes"]; !exists {
		t.Errorf("Stats should include sqlite_size_bytes")
	}

	if _, exists := stats["sqlite_modified"]; !exists {
		t.Errorf("Stats should include sqlite_modified")
	}

	if _, exists := stats["lancedb_modified"]; !exists {
		t.Errorf("Stats should include lancedb_modified")
	}
}

// TestRecoveryStatusTiming tests recovery status timing
func TestRecoveryStatusTiming(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	startTime := time.Now()
	status, err := rm.RecoverFromCorruption(CorruptionNone)
	endTime := time.Now()

	if err != nil {
		t.Errorf("Recovery should succeed: %v", err)
	}

	if status.StartTime.Before(startTime) {
		t.Errorf("Start time should be after test start")
	}

	if status.EndTime.After(endTime) {
		t.Errorf("End time should be before test end")
	}

	if status.EndTime.Before(status.StartTime) {
		t.Errorf("End time should be after start time")
	}
}

// BenchmarkValidateDatabases benchmarks database validation
func BenchmarkValidateDatabases(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "recovery_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sqlitePath := filepath.Join(tempDir, "test.db")
	lanceDBPath := filepath.Join(tempDir, "test_lancedb")

	rm := NewRecoveryManager(sqlitePath, lanceDBPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.ValidateDatabases()
	}
}