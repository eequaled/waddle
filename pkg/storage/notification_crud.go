package storage

import (
	"time"
)

// AddNotification adds a notification to the database.
func (sm *SessionManager) AddNotification(notif *Notification) error {
	if notif.ID == "" {
		return NewStorageError(ErrValidation, "notification ID is required", nil)
	}
	if notif.Type == "" {
		return NewStorageError(ErrValidation, "notification type is required", nil)
	}
	if notif.Title == "" {
		return NewStorageError(ErrValidation, "notification title is required", nil)
	}

	if notif.Timestamp.IsZero() {
		notif.Timestamp = time.Now()
	}

	query := `
		INSERT INTO notifications (id, type, title, message, timestamp, read, session_ref, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			title = excluded.title,
			message = excluded.message,
			timestamp = excluded.timestamp,
			read = excluded.read,
			session_ref = excluded.session_ref,
			metadata = excluded.metadata
	`

	stmt, err := sm.getStmt(query)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to prepare statement", err)
	}

	readInt := 0
	if notif.Read {
		readInt = 1
	}

	_, err = stmt.Exec(
		notif.ID,
		notif.Type,
		notif.Title,
		notif.Message,
		notif.Timestamp,
		readInt,
		notif.SessionRef,
		notif.Metadata,
	)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to add notification", err)
	}

	return nil
}

// GetNotifications retrieves notifications with optional limit.
func (sm *SessionManager) GetNotifications(limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, type, title, message, timestamp, read, session_ref, metadata
		FROM notifications
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := sm.db.Query(query, limit)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to get notifications", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notif Notification
		var readInt int
		var sessionRef, metadata *string
		err := rows.Scan(
			&notif.ID,
			&notif.Type,
			&notif.Title,
			&notif.Message,
			&notif.Timestamp,
			&readInt,
			&sessionRef,
			&metadata,
		)
		if err != nil {
			return nil, NewStorageError(ErrDatabase, "failed to scan notification", err)
		}

		notif.Read = readInt != 0
		if sessionRef != nil {
			notif.SessionRef = *sessionRef
		}
		if metadata != nil {
			notif.Metadata = *metadata
		}

		notifications = append(notifications, notif)
	}

	if err := rows.Err(); err != nil {
		return nil, NewStorageError(ErrDatabase, "error iterating notifications", err)
	}

	return notifications, nil
}

// GetUnreadNotifications retrieves unread notifications.
func (sm *SessionManager) GetUnreadNotifications(limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, type, title, message, timestamp, read, session_ref, metadata
		FROM notifications
		WHERE read = 0
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := sm.db.Query(query, limit)
	if err != nil {
		return nil, NewStorageError(ErrDatabase, "failed to get unread notifications", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notif Notification
		var readInt int
		var sessionRef, metadata *string
		err := rows.Scan(
			&notif.ID,
			&notif.Type,
			&notif.Title,
			&notif.Message,
			&notif.Timestamp,
			&readInt,
			&sessionRef,
			&metadata,
		)
		if err != nil {
			return nil, NewStorageError(ErrDatabase, "failed to scan notification", err)
		}

		notif.Read = readInt != 0
		if sessionRef != nil {
			notif.SessionRef = *sessionRef
		}
		if metadata != nil {
			notif.Metadata = *metadata
		}

		notifications = append(notifications, notif)
	}

	if err := rows.Err(); err != nil {
		return nil, NewStorageError(ErrDatabase, "error iterating notifications", err)
	}

	return notifications, nil
}

// MarkNotificationsRead marks the specified notifications as read.
func (sm *SessionManager) MarkNotificationsRead(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build query with placeholders
	query := "UPDATE notifications SET read = 1 WHERE id IN ("
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ", "
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	_, err := sm.db.Exec(query, args...)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to mark notifications read", err)
	}

	return nil
}

// MarkAllNotificationsRead marks all notifications as read.
func (sm *SessionManager) MarkAllNotificationsRead() error {
	_, err := sm.db.Exec("UPDATE notifications SET read = 1 WHERE read = 0")
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to mark all notifications read", err)
	}
	return nil
}

// DeleteNotification deletes a notification by ID.
func (sm *SessionManager) DeleteNotification(id string) error {
	_, err := sm.db.Exec("DELETE FROM notifications WHERE id = ?", id)
	if err != nil {
		return NewStorageError(ErrDatabase, "failed to delete notification", err)
	}
	return nil
}

// DeleteOldNotifications deletes notifications older than the specified duration.
func (sm *SessionManager) DeleteOldNotifications(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := sm.db.Exec("DELETE FROM notifications WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, NewStorageError(ErrDatabase, "failed to delete old notifications", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, NewStorageError(ErrDatabase, "failed to get rows affected", err)
	}

	return count, nil
}
