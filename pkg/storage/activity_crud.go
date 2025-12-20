package storage

import (
	"database/sql"
	"time"
)

// AddBlock adds an activity block to a session's app activity.
func (sm *SessionManager) AddBlock(sessionID int64, appName string, block *ActivityBlock) error {
	if appName == "" {
		return NewStorageError(ErrValidation, "app name is required", nil)
	}
	if block.BlockID == "" {
		return NewStorageError(ErrValidation, "block ID is required", nil)
	}

	// Get or create app activity
	appActivityID, err := sm.getOrCreateAppActivity(sessionID, appName)
	if err != nil {
		return err
	}

	// Encrypt OCR text
	var encryptedOCR []byte
	if block.OCRText != "" && sm.encryptionMgr != nil {
		encryptedOCR, err = sm.encryptionMgr.Encrypt([]byte(block.OCRText))
		if err != nil {
			return NewStorageError(ErrEncryption, "failed to encrypt OCR text", err)
		}
	}

	query := `
		INSERT INTO activity_blocks (app_activity_id, block_id, start_time, end_time, ocr_text_encrypted, micro_summary)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(app_activity_id, block_id) DO UPDATE SET
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			ocr_text_encrypted = excluded.ocr_text_encrypted,
			micro_summary = excluded.micro_summary
	`

	stmt, err := sm.getStmt(query)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	result, err := stmt.Exec(
		appActivityID,
		block.BlockID,
		block.StartTime,
		block.EndTime,
		encryptedOCR,
		block.MicroSummary,
	)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to add block", err)
	}

	id, err := result.LastInsertId()
	if err == nil && id > 0 {
		block.ID = id
	}
	block.AppActivityID = appActivityID

	return nil
}

// GetBlocks retrieves all activity blocks for a session's app.
func (sm *SessionManager) GetBlocks(sessionID int64, appName string) ([]ActivityBlock, error) {
	query := `
		SELECT ab.id, ab.app_activity_id, ab.block_id, ab.start_time, ab.end_time, ab.ocr_text_encrypted, ab.micro_summary
		FROM activity_blocks ab
		JOIN app_activities aa ON ab.app_activity_id = aa.id
		WHERE aa.session_id = ? AND aa.app_name = ?
		ORDER BY ab.start_time ASC
	`

	rows, err := sm.db.Query(query, sessionID, appName)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to get blocks", err)
	}
	defer rows.Close()

	var blocks []ActivityBlock
	for rows.Next() {
		var block ActivityBlock
		var encryptedOCR []byte
		err := rows.Scan(
			&block.ID,
			&block.AppActivityID,
			&block.BlockID,
			&block.StartTime,
			&block.EndTime,
			&encryptedOCR,
			&block.MicroSummary,
		)
		if err != nil {
			return nil, NewStorageError(ErrDatabase, "failed to scan block", err)
		}

		// Decrypt OCR text
		if len(encryptedOCR) > 0 && sm.encryptionMgr != nil {
			decrypted, err := sm.encryptionMgr.Decrypt(encryptedOCR)
			if err != nil {
				return nil, NewStorageError(ErrEncryption, "failed to decrypt OCR text", err)
			}
			block.OCRText = string(decrypted)
		}

		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, NewStorageError(ErrDatabase, "error iterating blocks", err)
	}

	return blocks, nil
}

// getOrCreateAppActivity gets or creates an app activity for a session.
func (sm *SessionManager) getOrCreateAppActivity(sessionID int64, appName string) (int64, error) {
	// Try to get existing
	var id int64
	err := sm.db.QueryRow(`
		SELECT id FROM app_activities WHERE session_id = ? AND app_name = ?
	`, sessionID, appName).Scan(&id)

	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return 0, NewStorageError(ErrDatabase, "failed to get app activity", err)
	}

	// Create new
	now := time.Now()
	result, err := sm.db.Exec(`
		INSERT INTO app_activities (session_id, app_name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, appName, now, now)
	if err != nil {
		// Check if it was created by another goroutine
		if isUniqueConstraintError(err) {
			err := sm.db.QueryRow(`
				SELECT id FROM app_activities WHERE session_id = ? AND app_name = ?
			`, sessionID, appName).Scan(&id)
			if err == nil {
				return id, nil
			}
		}
		return 0, NewStorageError(ErrDatabase, "failed to create app activity", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, NewStorageError(ErrDatabase, "failed to get last insert id", err)
	}

	return id, nil
}

// GetAppActivities retrieves all app activities for a session.
func (sm *SessionManager) GetAppActivities(sessionID int64) ([]AppActivity, error) {
	query := `
		SELECT id, session_id, app_name, created_at, updated_at
		FROM app_activities
		WHERE session_id = ?
		ORDER BY app_name ASC
	`

	rows, err := sm.db.Query(query, sessionID)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to get app activities", err)
	}
	defer rows.Close()

	var activities []AppActivity
	for rows.Next() {
		var activity AppActivity
		err := rows.Scan(
			&activity.ID,
			&activity.SessionID,
			&activity.AppName,
			&activity.CreatedAt,
			&activity.UpdatedAt,
		)
		if err != nil {
			return nil, NewStorageError(ErrDatabase, "failed to scan app activity", err)
		}
		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, NewStorageError(ErrDatabase, "error iterating app activities", err)
	}

	return activities, nil
}

// AddChat adds a chat message to a session.
func (sm *SessionManager) AddChat(sessionID int64, chat *ChatMessage) error {
	if !ValidChatRoles[chat.Role] {
		return NewStorageError(ErrValidation, "invalid chat role", nil)
	}
	if chat.Content == "" {
		return NewStorageError(ErrValidation, "chat content is required", nil)
	}

	// Encrypt content
	var encryptedContent []byte
	var err error
	if sm.encryptionMgr != nil {
		encryptedContent, err = sm.encryptionMgr.Encrypt([]byte(chat.Content))
		if err != nil {
			return NewStorageError(ErrEncryption, "failed to encrypt chat content", err)
		}
	} else {
		encryptedContent = []byte(chat.Content)
	}

	if chat.Timestamp.IsZero() {
		chat.Timestamp = time.Now()
	}

	query := `
		INSERT INTO chats (session_id, role, content_encrypted, timestamp)
		VALUES (?, ?, ?, ?)
	`

	stmt, err := sm.getStmt(query)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	result, err := stmt.Exec(sessionID, chat.Role, encryptedContent, chat.Timestamp)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to add chat", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to get last insert id", err)
	}

	chat.ID = id
	chat.SessionID = sessionID
	return nil
}

// GetChats retrieves all chat messages for a session.
func (sm *SessionManager) GetChats(sessionID int64) ([]ChatMessage, error) {
	query := `
		SELECT id, session_id, role, content_encrypted, timestamp
		FROM chats
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`

	rows, err := sm.db.Query(query, sessionID)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to get chats", err)
	}
	defer rows.Close()

	var chats []ChatMessage
	for rows.Next() {
		var chat ChatMessage
		var encryptedContent []byte
		err := rows.Scan(
			&chat.ID,
			&chat.SessionID,
			&chat.Role,
			&encryptedContent,
			&chat.Timestamp,
		)
		if err != nil {
			return nil, NewStorageError(ErrDatabase, "failed to scan chat", err)
		}

		// Decrypt content
		if len(encryptedContent) > 0 && sm.encryptionMgr != nil {
			decrypted, err := sm.encryptionMgr.Decrypt(encryptedContent)
			if err != nil {
				return nil, NewStorageError(ErrEncryption, "failed to decrypt chat content", err)
			}
			chat.Content = string(decrypted)
		} else {
			chat.Content = string(encryptedContent)
		}

		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, NewStorageError(ErrDatabase, "error iterating chats", err)
	}

	return chats, nil
}
