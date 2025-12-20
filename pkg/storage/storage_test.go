package storage

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// TestGeneratorsCompile verifies that all generators produce valid values.
func TestGeneratorsCompile(t *testing.T) {
	parameters := DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("GenSession produces valid sessions", prop.ForAll(
		func(s Session) bool {
			return s.Date != "" // Basic validation
		},
		GenSession(),
	))

	properties.Property("GenActivityBlock produces valid blocks", prop.ForAll(
		func(b ActivityBlock) bool {
			return b.BlockID != ""
		},
		GenActivityBlock(),
	))

	properties.Property("GenAppActivity produces valid activities", prop.ForAll(
		func(a AppActivity) bool {
			return a.AppName != ""
		},
		GenAppActivity(),
	))

	properties.Property("GenChatMessage produces valid messages", prop.ForAll(
		func(c ChatMessage) bool {
			return ValidChatRoles[c.Role]
		},
		GenChatMessage(),
	))

	properties.Property("GenNotification produces valid notifications", prop.ForAll(
		func(n Notification) bool {
			return n.Type != ""
		},
		GenNotification(),
	))

	properties.Property("GenDateString produces valid dates", prop.ForAll(
		func(date string) bool {
			// Check format YYYY-MM-DD
			if len(date) != 10 {
				return false
			}
			return date[4] == '-' && date[7] == '-'
		},
		GenDateString(),
	))

	properties.Property("GenBlockID produces valid block IDs", prop.ForAll(
		func(blockID string) bool {
			// Check format HH-MM
			if len(blockID) != 5 {
				return false
			}
			return blockID[2] == '-'
		},
		GenBlockID(),
	))

	properties.Property("GenEmbedding produces 768-dimensional vectors", prop.ForAll(
		func(embedding []float32) bool {
			return len(embedding) == 768
		},
		GenEmbedding(),
	))

	properties.Property("GenDateRange produces valid ranges", prop.ForAll(
		func(dr DateRange) bool {
			return dr.StartDate <= dr.EndDate
		},
		GenDateRange(),
	))

	properties.TestingRun(t)
}

// TestErrorTypes verifies error type functionality.
func TestErrorTypes(t *testing.T) {
	t.Run("StorageError implements error interface", func(t *testing.T) {
		err := NewStorageError(ErrDatabase, "test error", nil)
		if err.Error() == "" {
			t.Error("Error() should return non-empty string")
		}
	})

	t.Run("StorageError with cause", func(t *testing.T) {
		cause := NewStorageError(ErrValidation, "validation failed", nil)
		err := NewStorageError(ErrDatabase, "db error", cause)
		if err.Unwrap() != cause {
			t.Error("Unwrap() should return the cause")
		}
	})

	t.Run("IsNotFound helper", func(t *testing.T) {
		err := NewStorageError(ErrNotFound, "not found", nil)
		if !IsNotFound(err) {
			t.Error("IsNotFound should return true for ErrNotFound")
		}
		err2 := NewStorageError(ErrDatabase, "db error", nil)
		if IsNotFound(err2) {
			t.Error("IsNotFound should return false for non-NotFound errors")
		}
	})

	t.Run("IsConflict helper", func(t *testing.T) {
		err := NewStorageError(ErrConflict, "conflict", nil)
		if !IsConflict(err) {
			t.Error("IsConflict should return true for ErrConflict")
		}
	})

	t.Run("IsRetryable helper", func(t *testing.T) {
		dbErr := NewStorageError(ErrDatabase, "db error", nil)
		if !IsRetryable(dbErr) {
			t.Error("Database errors should be retryable")
		}
		validationErr := NewStorageError(ErrValidation, "validation error", nil)
		if IsRetryable(validationErr) {
			t.Error("Validation errors should not be retryable")
		}
	})

	t.Run("ErrorCode String", func(t *testing.T) {
		codes := []ErrorCode{ErrDatabase, ErrEncryption, ErrVector, ErrFileSystem, ErrValidation, ErrNotFound, ErrConflict}
		for _, code := range codes {
			if code.String() == "" || code.String() == "UNKNOWN_ERROR" {
				t.Errorf("ErrorCode %d should have a valid string representation", code)
			}
		}
	})
}

// TestModelValidation verifies model constants and validation.
func TestModelValidation(t *testing.T) {
	t.Run("ValidChatRoles contains expected roles", func(t *testing.T) {
		if !ValidChatRoles[ChatRoleUser] {
			t.Error("ValidChatRoles should contain 'user'")
		}
		if !ValidChatRoles[ChatRoleAssistant] {
			t.Error("ValidChatRoles should contain 'assistant'")
		}
		if ValidChatRoles["invalid"] {
			t.Error("ValidChatRoles should not contain 'invalid'")
		}
	})

	t.Run("SearchMatchType constants", func(t *testing.T) {
		if MatchTypeFullText != "fulltext" {
			t.Error("MatchTypeFullText should be 'fulltext'")
		}
		if MatchTypeSemantic != "semantic" {
			t.Error("MatchTypeSemantic should be 'semantic'")
		}
	})

	t.Run("MigrationStatus constants", func(t *testing.T) {
		statuses := []MigrationStatus{
			MigrationStatusIdle,
			MigrationStatusDetecting,
			MigrationStatusBackingUp,
			MigrationStatusMigrating,
			MigrationStatusVerifying,
			MigrationStatusComplete,
			MigrationStatusFailed,
			MigrationStatusRollingBack,
		}
		for _, status := range statuses {
			if status == "" {
				t.Error("MigrationStatus should not be empty")
			}
		}
	})

	t.Run("HealthStatus constants", func(t *testing.T) {
		if HealthStatusHealthy != "healthy" {
			t.Error("HealthStatusHealthy should be 'healthy'")
		}
		if HealthStatusDegraded != "degraded" {
			t.Error("HealthStatusDegraded should be 'degraded'")
		}
		if HealthStatusUnhealthy != "unhealthy" {
			t.Error("HealthStatusUnhealthy should be 'unhealthy'")
		}
	})
}

// TestDefaultStorageConfig verifies default configuration values.
func TestDefaultStorageConfig(t *testing.T) {
	config := DefaultStorageConfig("/test/path")

	if config.DataDir != "/test/path" {
		t.Errorf("DataDir should be '/test/path', got '%s'", config.DataDir)
	}
	if config.RetentionDays != 365 {
		t.Errorf("RetentionDays should be 365, got %d", config.RetentionDays)
	}
	if config.BackupTime != "02:00" {
		t.Errorf("BackupTime should be '02:00', got '%s'", config.BackupTime)
	}
	if config.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("EmbeddingModel should be 'nomic-embed-text', got '%s'", config.EmbeddingModel)
	}
}
