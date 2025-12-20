package storage

import (
	"database/sql"
	"time"
)

// Create creates a new session in the database.
func (sm *SessionManager) Create(session *Session) error {
	if session.Date == "" {
		return ErrEmptyRequiredField
	}

	// Encrypt sensitive fields
	var encryptedText []byte
	var err error
	if session.ExtractedText != "" && sm.encryptionMgr != nil {
		encryptedText, err = sm.encryptionMgr.Encrypt([]byte(session.ExtractedText))
		if err != nil {
			return NewStorageError(ErrEncryption, "failed to encrypt extracted text", err)
		}
	}

	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now

	query := `
		INSERT INTO sessions (date, custom_title, custom_summary, original_summary, extracted_text_encrypted, 
		                     entities_json, synthesis_status, ai_summary, ai_bullets, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := sm.getStmt(query)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	// Set defaults for synthesis columns if not provided
	if session.EntitiesJSON == "" {
		session.EntitiesJSON = "[]"
	}
	if session.SynthesisStatus == "" {
		session.SynthesisStatus = "pending"
	}
	if session.AISummary == "" {
		session.AISummary = ""
	}
	if session.AIBullets == "" {
		session.AIBullets = "[]"
	}

	result, err := stmt.Exec(
		session.Date,
		session.CustomTitle,
		session.CustomSummary,
		session.OriginalSummary,
		encryptedText,
		session.EntitiesJSON,
		session.SynthesisStatus,
		session.AISummary,
		session.AIBullets,
		session.CreatedAt,
		session.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation
		if isUniqueConstraintError(err) {
			return ErrSessionAlreadyExists
		}
		return NewStorageError(ErrDatabase, "failed to create session", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to get last insert id", err)
	}

	session.ID = id
	return nil
}

// Get retrieves a session by date.
func (sm *SessionManager) Get(date string) (*Session, error) {
	if date == "" {
		return nil, ErrInvalidDate
	}

	query := `
		SELECT id, date, custom_title, custom_summary, original_summary, extracted_text_encrypted, 
		       entities_json, synthesis_status, ai_summary, ai_bullets, created_at, updated_at
		FROM sessions
		WHERE date = ?
	`

	stmt, err := sm.getStmt(query)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	var session Session
	var encryptedText []byte
	err = stmt.QueryRow(date).Scan(
		&session.ID,
		&session.Date,
		&session.CustomTitle,
		&session.CustomSummary,
		&session.OriginalSummary,
		&encryptedText,
		&session.EntitiesJSON,
		&session.SynthesisStatus,
		&session.AISummary,
		&session.AIBullets,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to get session", err)
	}

	// Decrypt sensitive fields
	if len(encryptedText) > 0 && sm.encryptionMgr != nil {
		decrypted, err := sm.encryptionMgr.Decrypt(encryptedText)
		if err != nil {
			return nil, NewStorageError(ErrEncryption, "failed to decrypt extracted text", err)
		}
		session.ExtractedText = string(decrypted)
	}

	return &session, nil
}

// GetByID retrieves a session by ID.
func (sm *SessionManager) GetByID(id int64) (*Session, error) {
	query := `
		SELECT id, date, custom_title, custom_summary, original_summary, extracted_text_encrypted, 
		       entities_json, synthesis_status, ai_summary, ai_bullets, created_at, updated_at
		FROM sessions
		WHERE id = ?
	`

	stmt, err := sm.getStmt(query)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	var session Session
	var encryptedText []byte
	err = stmt.QueryRow(id).Scan(
		&session.ID,
		&session.Date,
		&session.CustomTitle,
		&session.CustomSummary,
		&session.OriginalSummary,
		&encryptedText,
		&session.EntitiesJSON,
		&session.SynthesisStatus,
		&session.AISummary,
		&session.AIBullets,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to get session", err)
	}

	// Decrypt sensitive fields
	if len(encryptedText) > 0 && sm.encryptionMgr != nil {
		decrypted, err := sm.encryptionMgr.Decrypt(encryptedText)
		if err != nil {
			return nil, NewStorageError(ErrEncryption, "failed to decrypt extracted text", err)
		}
		session.ExtractedText = string(decrypted)
	}

	return &session, nil
}

// Update updates an existing session.
func (sm *SessionManager) Update(session *Session) error {
	if session.ID == 0 {
		return NewStorageError(ErrValidation, "session ID is required for update", nil)
	}

	// Encrypt sensitive fields
	var encryptedText []byte
	var err error
	if session.ExtractedText != "" && sm.encryptionMgr != nil {
		encryptedText, err = sm.encryptionMgr.Encrypt([]byte(session.ExtractedText))
		if err != nil {
			return NewStorageError(ErrEncryption, "failed to encrypt extracted text", err)
		}
	}

	session.UpdatedAt = time.Now()

	query := `
		UPDATE sessions
		SET custom_title = ?, custom_summary = ?, original_summary = ?, extracted_text_encrypted = ?, 
		    entities_json = ?, synthesis_status = ?, ai_summary = ?, ai_bullets = ?, updated_at = ?
		WHERE id = ?
	`

	stmt, err := sm.getStmt(query)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	result, err := stmt.Exec(
		session.CustomTitle,
		session.CustomSummary,
		session.OriginalSummary,
		encryptedText,
		session.EntitiesJSON,
		session.SynthesisStatus,
		session.AISummary,
		session.AIBullets,
		session.UpdatedAt,
		session.ID,
	)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to update session", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// Delete deletes a session by date (cascade deletes related records).
func (sm *SessionManager) Delete(date string) error {
	if date == "" {
		return ErrInvalidDate
	}

	query := "DELETE FROM sessions WHERE date = ?"

	stmt, err := sm.getStmt(query)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	result, err := stmt.Exec(date)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to delete session", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteByID deletes a session by ID (cascade deletes related records).
func (sm *SessionManager) DeleteByID(id int64) error {
	query := "DELETE FROM sessions WHERE id = ?"

	stmt, err := sm.getStmt(query)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	result, err := stmt.Exec(id)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to delete session", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to get rows affected", err)
	}

	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// List returns paginated sessions with total count.
func (sm *SessionManager) List(page, pageSize int) ([]Session, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	// Get total count
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM sessions"
	err := sm.db.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, NewStorageError(ErrDatabase, "failed to count sessions", err)
	}

	// Get paginated sessions
	query := `
		SELECT id, date, custom_title, custom_summary, original_summary, extracted_text_encrypted, 
		       entities_json, synthesis_status, ai_summary, ai_bullets, created_at, updated_at
		FROM sessions
		ORDER BY date DESC
		LIMIT ? OFFSET ?
	`

	rows, err := sm.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, NewStorageError(ErrDatabase, "failed to list sessions", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var encryptedText []byte
		err := rows.Scan(
			&session.ID,
			&session.Date,
			&session.CustomTitle,
			&session.CustomSummary,
			&session.OriginalSummary,
			&encryptedText,
			&session.EntitiesJSON,
			&session.SynthesisStatus,
			&session.AISummary,
			&session.AIBullets,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, 0, NewStorageError(ErrDatabase, "failed to scan session", err)
		}

		// Decrypt sensitive fields
		if len(encryptedText) > 0 && sm.encryptionMgr != nil {
			decrypted, err := sm.encryptionMgr.Decrypt(encryptedText)
			if err != nil {
				return nil, 0, NewStorageError(ErrEncryption, "failed to decrypt extracted text", err)
			}
			session.ExtractedText = string(decrypted)
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, NewStorageError(ErrDatabase, "error iterating sessions", err)
	}

	return sessions, totalCount, nil
}

// isUniqueConstraintError checks if the error is a unique constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "UNIQUE constraint failed") || contains(errStr, "unique constraint")
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
