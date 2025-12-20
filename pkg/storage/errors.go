// Package storage provides the hybrid SQLite + LanceDB + filesystem storage system for Waddle.
package storage

import (
	"errors"
	"fmt"
)

// ErrorCode represents categorized error types for the storage system.
type ErrorCode int

const (
	// ErrDatabase indicates a database-level error (connection, query, transaction).
	ErrDatabase ErrorCode = iota
	// ErrEncryption indicates an encryption or decryption failure.
	ErrEncryption
	// ErrVector indicates a vector database (LanceDB) error.
	ErrVector
	// ErrFileSystem indicates a filesystem operation error.
	ErrFileSystem
	// ErrValidation indicates invalid input data.
	ErrValidation
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound
	// ErrConflict indicates a conflict (e.g., duplicate key).
	ErrConflict
	// ErrNotImplemented indicates functionality not yet implemented.
	ErrNotImplemented
	// ErrSerialization indicates JSON serialization/deserialization error.
	ErrSerialization
	// ErrMigration indicates a migration-specific error.
	ErrMigration
)

// String returns a human-readable name for the error code.
func (c ErrorCode) String() string {
	switch c {
	case ErrDatabase:
		return "DATABASE_ERROR"
	case ErrEncryption:
		return "ENCRYPTION_ERROR"
	case ErrVector:
		return "VECTOR_ERROR"
	case ErrFileSystem:
		return "FILESYSTEM_ERROR"
	case ErrValidation:
		return "VALIDATION_ERROR"
	case ErrNotFound:
		return "NOT_FOUND"
	case ErrConflict:
		return "CONFLICT"
	case ErrNotImplemented:
		return "NOT_IMPLEMENTED"
	case ErrSerialization:
		return "SERIALIZATION_ERROR"
	case ErrMigration:
		return "MIGRATION_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}

// StorageError represents a categorized error from the storage system.
type StorageError struct {
	Code      ErrorCode
	Message   string
	Cause     error
	Retryable bool
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewStorageError creates a new StorageError with the given parameters.
func NewStorageError(code ErrorCode, message string, cause error) *StorageError {
	return &StorageError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: isRetryable(code),
	}
}

// isRetryable determines if an error code represents a retryable condition.
func isRetryable(code ErrorCode) bool {
	switch code {
	case ErrDatabase, ErrVector, ErrFileSystem:
		return true
	default:
		return false
	}
}

// Sentinel errors for common conditions.
var (
	ErrSessionNotFound      = NewStorageError(ErrNotFound, "session not found", nil)
	ErrSessionAlreadyExists = NewStorageError(ErrConflict, "session already exists", nil)
	ErrInvalidDate          = NewStorageError(ErrValidation, "invalid date format", nil)
	ErrEmptyRequiredField   = NewStorageError(ErrValidation, "required field is empty", nil)
	ErrDecryptionFailed     = NewStorageError(ErrEncryption, "decryption failed", nil)
	ErrEncryptionFailed     = NewStorageError(ErrEncryption, "encryption failed", nil)
	ErrDatabaseCorruption   = NewStorageError(ErrDatabase, "database corruption detected", nil)
)

// IsNotFound checks if the error is a not-found error.
func IsNotFound(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Code == ErrNotFound
	}
	return false
}

// IsConflict checks if the error is a conflict error.
func IsConflict(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Code == ErrConflict
	}
	return false
}

// IsRetryable checks if the error is retryable.
func IsRetryable(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Retryable
	}
	return false
}
