package storage

import (
	"time"

	"waddle/pkg/types"
)

// ════════════════════════════════════════════════════════════════════════
// TYPE ALIASES — Re-exported from pkg/types (single source of truth).
// Using type aliases (=) means storage.Session IS types.Session,
// zero code changes needed at call sites, no marshaling differences.
// ════════════════════════════════════════════════════════════════════════

type Session = types.Session
type AppActivity = types.AppActivity
type ActivityBlock = types.ActivityBlock
type ChatMessage = types.ChatMessage
type ManualNote = types.ManualNote
type KnowledgeCard = types.KnowledgeCard
type Notification = types.Notification
type SearchResult = types.SearchResult
type VectorSearchResult = types.VectorSearchResult
type DateRange = types.DateRange
type Entity = types.Entity
type EntityType = types.EntityType

// Re-export chat role constants from types.
const (
	ChatRoleUser      = types.ChatRoleUser
	ChatRoleAssistant = types.ChatRoleAssistant
)

// Re-export ValidChatRoles from types.
var ValidChatRoles = types.ValidChatRoles

// Re-export search match type constants from types.
const (
	MatchTypeFullText = types.MatchTypeFullText
	MatchTypeSemantic = types.MatchTypeSemantic
)

// ════════════════════════════════════════════════════════════════════════
// STORAGE-SPECIFIC TYPES — These belong only in the storage layer.
// ════════════════════════════════════════════════════════════════════════

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
