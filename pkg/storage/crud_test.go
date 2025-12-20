package storage

import (
	"os"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// setupTestDB creates a test database and returns cleanup function.
func setupTestDB(t *testing.T) (*SessionManager, func()) {
	tempDir, err := os.MkdirTemp("", "waddle_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	sm := NewSessionManager(tempDir, em)
	if err := sm.Initialize(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize session manager: %v", err)
	}

	cleanup := func() {
		sm.Close()
		os.RemoveAll(tempDir)
	}

	return sm, cleanup
}

// TestSessionCRUD tests basic session CRUD operations.
func TestSessionCRUD(t *testing.T) {
	sm, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("Create and Get session", func(t *testing.T) {
		session := &Session{
			Date:            "2025-01-15",
			CustomTitle:     "Test Session",
			CustomSummary:   "Test summary",
			OriginalSummary: "Original summary",
			ExtractedText:   "Some extracted text",
		}

		err := sm.Create(session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		if session.ID == 0 {
			t.Error("Session ID should be set after creation")
		}

		// Get by date
		retrieved, err := sm.Get("2025-01-15")
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		if retrieved.CustomTitle != session.CustomTitle {
			t.Errorf("Expected title %q, got %q", session.CustomTitle, retrieved.CustomTitle)
		}
		if retrieved.ExtractedText != session.ExtractedText {
			t.Errorf("Expected extracted text %q, got %q", session.ExtractedText, retrieved.ExtractedText)
		}
	})

	t.Run("Update session", func(t *testing.T) {
		session, err := sm.Get("2025-01-15")
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		session.CustomTitle = "Updated Title"
		session.ExtractedText = "Updated extracted text"

		err = sm.Update(session)
		if err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		retrieved, err := sm.Get("2025-01-15")
		if err != nil {
			t.Fatalf("Failed to get updated session: %v", err)
		}

		if retrieved.CustomTitle != "Updated Title" {
			t.Errorf("Expected title %q, got %q", "Updated Title", retrieved.CustomTitle)
		}
	})

	t.Run("List sessions", func(t *testing.T) {
		// Create more sessions
		for i := 16; i <= 20; i++ {
			session := &Session{
				Date:        "2025-01-" + string(rune('0'+i/10)) + string(rune('0'+i%10)),
				CustomTitle: "Session " + string(rune('0'+i)),
			}
			sm.Create(session)
		}

		sessions, total, err := sm.List(1, 10)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if total < 6 {
			t.Errorf("Expected at least 6 sessions, got %d", total)
		}

		if len(sessions) > 10 {
			t.Errorf("Expected at most 10 sessions per page, got %d", len(sessions))
		}
	})

	t.Run("Delete session", func(t *testing.T) {
		err := sm.Delete("2025-01-15")
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}

		_, err = sm.Get("2025-01-15")
		if !IsNotFound(err) {
			t.Error("Expected session to be deleted")
		}
	})

	t.Run("Create duplicate session fails", func(t *testing.T) {
		session := &Session{Date: "2025-01-16"}
		err := sm.Create(session)
		if err == nil {
			t.Error("Expected error when creating duplicate session")
		}
		if !IsConflict(err) {
			t.Errorf("Expected conflict error, got: %v", err)
		}
	})
}

// TestActivityBlockCRUD tests activity block operations.
func TestActivityBlockCRUD(t *testing.T) {
	sm, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a session first
	session := &Session{Date: "2025-01-15"}
	if err := sm.Create(session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	t.Run("Add and Get blocks", func(t *testing.T) {
		block := &ActivityBlock{
			BlockID:      "15-30",
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(5 * time.Minute),
			OCRText:      "Some OCR text with sensitive data",
			MicroSummary: "User was browsing",
		}

		err := sm.AddBlock(session.ID, "Chrome", block)
		if err != nil {
			t.Fatalf("Failed to add block: %v", err)
		}

		blocks, err := sm.GetBlocks(session.ID, "Chrome")
		if err != nil {
			t.Fatalf("Failed to get blocks: %v", err)
		}

		if len(blocks) != 1 {
			t.Fatalf("Expected 1 block, got %d", len(blocks))
		}

		if blocks[0].OCRText != block.OCRText {
			t.Errorf("Expected OCR text %q, got %q", block.OCRText, blocks[0].OCRText)
		}
	})

	t.Run("Update block on conflict", func(t *testing.T) {
		block := &ActivityBlock{
			BlockID:      "15-30",
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(10 * time.Minute),
			OCRText:      "Updated OCR text",
			MicroSummary: "Updated summary",
		}

		err := sm.AddBlock(session.ID, "Chrome", block)
		if err != nil {
			t.Fatalf("Failed to update block: %v", err)
		}

		blocks, err := sm.GetBlocks(session.ID, "Chrome")
		if err != nil {
			t.Fatalf("Failed to get blocks: %v", err)
		}

		if len(blocks) != 1 {
			t.Fatalf("Expected 1 block after update, got %d", len(blocks))
		}

		if blocks[0].MicroSummary != "Updated summary" {
			t.Errorf("Expected summary %q, got %q", "Updated summary", blocks[0].MicroSummary)
		}
	})
}

// TestChatCRUD tests chat message operations.
func TestChatCRUD(t *testing.T) {
	sm, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a session first
	session := &Session{Date: "2025-01-15"}
	if err := sm.Create(session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	t.Run("Add and Get chats", func(t *testing.T) {
		chat1 := &ChatMessage{
			Role:    ChatRoleUser,
			Content: "Hello, what did I do today?",
		}
		chat2 := &ChatMessage{
			Role:    ChatRoleAssistant,
			Content: "You worked on the storage migration project.",
		}

		if err := sm.AddChat(session.ID, chat1); err != nil {
			t.Fatalf("Failed to add user chat: %v", err)
		}
		if err := sm.AddChat(session.ID, chat2); err != nil {
			t.Fatalf("Failed to add assistant chat: %v", err)
		}

		chats, err := sm.GetChats(session.ID)
		if err != nil {
			t.Fatalf("Failed to get chats: %v", err)
		}

		if len(chats) != 2 {
			t.Fatalf("Expected 2 chats, got %d", len(chats))
		}

		if chats[0].Content != chat1.Content {
			t.Errorf("Expected content %q, got %q", chat1.Content, chats[0].Content)
		}
	})

	t.Run("Invalid role fails", func(t *testing.T) {
		chat := &ChatMessage{
			Role:    "invalid",
			Content: "Test",
		}

		err := sm.AddChat(session.ID, chat)
		if err == nil {
			t.Error("Expected error for invalid role")
		}
	})
}

// TestNotificationCRUD tests notification operations.
func TestNotificationCRUD(t *testing.T) {
	sm, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("Add and Get notifications", func(t *testing.T) {
		notif := &Notification{
			ID:      "notif-1",
			Type:    "info",
			Title:   "Test Notification",
			Message: "This is a test",
		}

		if err := sm.AddNotification(notif); err != nil {
			t.Fatalf("Failed to add notification: %v", err)
		}

		notifications, err := sm.GetNotifications(10)
		if err != nil {
			t.Fatalf("Failed to get notifications: %v", err)
		}

		if len(notifications) != 1 {
			t.Fatalf("Expected 1 notification, got %d", len(notifications))
		}

		if notifications[0].Title != notif.Title {
			t.Errorf("Expected title %q, got %q", notif.Title, notifications[0].Title)
		}
	})

	t.Run("Mark notifications read", func(t *testing.T) {
		// Add more notifications
		for i := 2; i <= 5; i++ {
			notif := &Notification{
				ID:    "notif-" + string(rune('0'+i)),
				Type:  "info",
				Title: "Notification " + string(rune('0'+i)),
			}
			sm.AddNotification(notif)
		}

		// Mark some as read
		err := sm.MarkNotificationsRead([]string{"notif-1", "notif-2"})
		if err != nil {
			t.Fatalf("Failed to mark notifications read: %v", err)
		}

		unread, err := sm.GetUnreadNotifications(10)
		if err != nil {
			t.Fatalf("Failed to get unread notifications: %v", err)
		}

		if len(unread) != 3 {
			t.Errorf("Expected 3 unread notifications, got %d", len(unread))
		}
	})
}

// TestTransactionAtomicity is a property-based test for transaction atomicity.
// **Property 12: Transaction Atomicity**
// **Validates: Requirements 10.1, 10.3**
func TestTransactionAtomicity(t *testing.T) {
	parameters := DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("Valid session data is persisted atomically", prop.ForAll(
		func(date string, title string) bool {
			sm, cleanup := setupTestDB(t)
			defer cleanup()

			session := &Session{
				Date:        date,
				CustomTitle: title,
			}

			err := sm.Create(session)
			if err != nil {
				// If creation fails, verify no partial data
				_, getErr := sm.Get(date)
				return IsNotFound(getErr)
			}

			// If creation succeeds, verify data is complete
			retrieved, err := sm.Get(date)
			if err != nil {
				return false
			}

			return retrieved.Date == date && retrieved.CustomTitle == title
		},
		GenDateString(),
		GenNonEmptyString(),
	))

	properties.TestingRun(t)
}

// TestForeignKeyIntegrity is a property-based test for foreign key constraints.
// **Property 13: Foreign Key Integrity**
// **Validates: Requirements 10.4**
func TestForeignKeyIntegrity(t *testing.T) {
	parameters := DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("Insert with invalid foreign key fails", prop.ForAll(
		func(invalidSessionID int64) bool {
			sm, cleanup := setupTestDB(t)
			defer cleanup()

			// Try to insert app_activity with non-existent session_id
			_, err := sm.db.Exec(`
				INSERT INTO app_activities (session_id, app_name)
				VALUES (?, 'TestApp')
			`, invalidSessionID)

			// Should fail due to foreign key constraint
			return err != nil
		},
		gopter.Gen(func(params *gopter.GenParameters) *gopter.GenResult {
			// Generate IDs that definitely don't exist
			id := int64(params.Rng.Intn(1000000) + 1000000)
			return gopter.NewGenResult(id, gopter.NoShrinker)
		}),
	))

	properties.Property("Cascade delete removes all children", prop.ForAll(
		func(date string, appName string) bool {
			sm, cleanup := setupTestDB(t)
			defer cleanup()

			// Create session
			session := &Session{Date: date}
			if err := sm.Create(session); err != nil {
				return true // Skip if date conflict
			}

			// Add activity block
			block := &ActivityBlock{
				BlockID:   "15-30",
				StartTime: time.Now(),
				EndTime:   time.Now().Add(5 * time.Minute),
				OCRText:   "Test OCR",
			}
			if err := sm.AddBlock(session.ID, appName, block); err != nil {
				return false
			}

			// Add chat
			chat := &ChatMessage{Role: ChatRoleUser, Content: "Test"}
			if err := sm.AddChat(session.ID, chat); err != nil {
				return false
			}

			// Delete session
			if err := sm.Delete(date); err != nil {
				return false
			}

			// Verify all children are deleted
			var activityCount, blockCount, chatCount int
			sm.db.QueryRow("SELECT COUNT(*) FROM app_activities WHERE session_id = ?", session.ID).Scan(&activityCount)
			sm.db.QueryRow("SELECT COUNT(*) FROM activity_blocks WHERE app_activity_id IN (SELECT id FROM app_activities WHERE session_id = ?)", session.ID).Scan(&blockCount)
			sm.db.QueryRow("SELECT COUNT(*) FROM chats WHERE session_id = ?", session.ID).Scan(&chatCount)

			return activityCount == 0 && blockCount == 0 && chatCount == 0
		},
		GenDateString(),
		GenAppName(),
	))

	properties.TestingRun(t)
}

// TestDataValidation is a property-based test for data validation.
// **Property 14: Data Validation**
// **Validates: Requirements 10.5**
func TestDataValidation(t *testing.T) {
	parameters := DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("Empty date is rejected", prop.ForAll(
		func(_ bool) bool {
			sm, cleanup := setupTestDB(t)
			defer cleanup()

			session := &Session{Date: ""}
			err := sm.Create(session)
			return err != nil
		},
		gopter.Gen(func(*gopter.GenParameters) *gopter.GenResult {
			return gopter.NewGenResult(true, gopter.NoShrinker)
		}),
	))

	properties.Property("Invalid chat role is rejected", prop.ForAll(
		func(invalidRole string) bool {
			if ValidChatRoles[invalidRole] {
				return true // Skip valid roles
			}

			sm, cleanup := setupTestDB(t)
			defer cleanup()

			session := &Session{Date: "2025-01-15"}
			sm.Create(session)

			chat := &ChatMessage{Role: invalidRole, Content: "Test"}
			err := sm.AddChat(session.ID, chat)
			return err != nil
		},
		gopter.Gen(func(params *gopter.GenParameters) *gopter.GenResult {
			roles := []string{"admin", "system", "bot", "invalid", ""}
			role := roles[params.Rng.Intn(len(roles))]
			return gopter.NewGenResult(role, gopter.NoShrinker)
		}),
	))

	properties.Property("Empty notification ID is rejected", prop.ForAll(
		func(_ bool) bool {
			sm, cleanup := setupTestDB(t)
			defer cleanup()

			notif := &Notification{ID: "", Type: "info", Title: "Test"}
			err := sm.AddNotification(notif)
			return err != nil
		},
		gopter.Gen(func(*gopter.GenParameters) *gopter.GenResult {
			return gopter.NewGenResult(true, gopter.NoShrinker)
		}),
	))

	properties.TestingRun(t)
}
