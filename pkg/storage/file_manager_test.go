package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// setupTestFileManager creates a test FileManager and returns cleanup function.
func setupTestFileManager(t *testing.T) (*FileManager, func()) {
	tempDir, err := os.MkdirTemp("", "waddle_files_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	fm := NewFileManager(tempDir)
	if err := fm.Initialize(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize file manager: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return fm, cleanup
}

// TestFileManagerBasicOperations tests basic file operations.
func TestFileManagerBasicOperations(t *testing.T) {
	fm, cleanup := setupTestFileManager(t)
	defer cleanup()

	t.Run("SaveFile and read back", func(t *testing.T) {
		data := []byte("test screenshot data")
		path, err := fm.SaveFile("2025-01-15", "Chrome", "15-30-00.png", data)
		if err != nil {
			t.Fatalf("Failed to save file: %v", err)
		}

		if path == "" {
			t.Error("Expected non-empty path")
		}

		// Verify file exists
		if !fm.FileExists(path) {
			t.Error("File should exist after save")
		}

		// Read back
		readData, err := fm.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(readData) != string(data) {
			t.Errorf("Data mismatch: expected %q, got %q", data, readData)
		}
	})

	t.Run("SaveLatestScreenshot", func(t *testing.T) {
		data := []byte("latest screenshot")
		path, err := fm.SaveLatestScreenshot("2025-01-15", "Chrome", data)
		if err != nil {
			t.Fatalf("Failed to save latest screenshot: %v", err)
		}

		expectedPath := fm.GetLatestScreenshotPath("2025-01-15", "Chrome")
		if !fm.FileExists(path) {
			t.Error("Latest screenshot should exist")
		}

		// Verify it's at the expected location
		fullPath := fm.GetFullPath(path)
		if fullPath != expectedPath {
			t.Errorf("Path mismatch: expected %q, got %q", expectedPath, fullPath)
		}
	})

	t.Run("DeleteSessionFiles", func(t *testing.T) {
		// Create some files
		fm.SaveFile("2025-01-16", "App1", "file1.png", []byte("data1"))
		fm.SaveFile("2025-01-16", "App2", "file2.png", []byte("data2"))

		// Delete session
		err := fm.DeleteSessionFiles("2025-01-16")
		if err != nil {
			t.Fatalf("Failed to delete session files: %v", err)
		}

		// Verify files are gone
		files, _ := fm.ListSessionFiles("2025-01-16")
		if len(files) != 0 {
			t.Errorf("Expected 0 files after delete, got %d", len(files))
		}
	})

	t.Run("GetStorageStats", func(t *testing.T) {
		// Create some files
		fm.SaveFile("2025-01-17", "App1", "file1.png", []byte("data1"))
		fm.SaveFile("2025-01-17", "App2", "file2.png", []byte("data2data2"))

		stats, err := fm.GetStorageStats()
		if err != nil {
			t.Fatalf("Failed to get storage stats: %v", err)
		}

		if stats.TotalFiles < 2 {
			t.Errorf("Expected at least 2 files, got %d", stats.TotalFiles)
		}

		if stats.ScreenshotCount < 2 {
			t.Errorf("Expected at least 2 screenshots, got %d", stats.ScreenshotCount)
		}
	})
}

// TestFileStoragePathCorrectness is a property-based test.
// **Property 8: File Storage Path Correctness**
// **Validates: Requirements 3.1, 3.4**
func TestFileStoragePathCorrectness(t *testing.T) {
	parameters := DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("Saved file exists at expected path", prop.ForAll(
		func(sessionDate string, appName string) bool {
			fm, cleanup := setupTestFileManager(t)
			defer cleanup()

			filename := "test-file.png"
			data := []byte("test data")

			relPath, err := fm.SaveFile(sessionDate, appName, filename, data)
			if err != nil {
				return false
			}

			// Verify file exists
			if !fm.FileExists(relPath) {
				return false
			}

			// Verify path structure
			expectedPath := fm.GetFilePath(sessionDate, appName, filename)
			actualPath := fm.GetFullPath(relPath)

			return actualPath == expectedPath
		},
		GenDateString(),
		GenAppName(),
	))

	properties.TestingRun(t)
}

// TestFileReferenceIntegrity is a property-based test.
// **Property 9: File Reference Integrity**
// **Validates: Requirements 3.3**
func TestFileReferenceIntegrity(t *testing.T) {
	parameters := DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("File path is a string reference, not binary", prop.ForAll(
		func(sessionDate string, appName string) bool {
			fm, cleanup := setupTestFileManager(t)
			defer cleanup()

			filename := "screenshot.png"
			data := []byte("binary screenshot data that should not be in path")

			relPath, err := fm.SaveFile(sessionDate, appName, filename, data)
			if err != nil {
				return false
			}

			// Path should be a string, not contain binary data
			if len(relPath) > 500 {
				return false // Path too long, might contain binary
			}

			// Path should be valid UTF-8 string
			for _, r := range relPath {
				if r < 32 && r != '\t' && r != '\n' && r != '\r' {
					return false // Contains control characters
				}
			}

			// Verify actual file location matches path
			fullPath := fm.GetFullPath(relPath)
			_, err = os.Stat(fullPath)
			return err == nil
		},
		GenDateString(),
		GenAppName(),
	))

	properties.TestingRun(t)
}

// TestCleanOrphanedFiles tests orphaned file cleanup.
func TestCleanOrphanedFiles(t *testing.T) {
	fm, cleanup := setupTestFileManager(t)
	defer cleanup()

	// Create files for sessions
	fm.SaveFile("session-1", "App", "file.png", []byte("data"))
	fm.SaveFile("session-2", "App", "file.png", []byte("data"))
	fm.SaveFile("session-3", "App", "file.png", []byte("data"))

	// Only session-1 and session-2 are valid
	validIDs := []string{"session-1", "session-2"}

	deleted, err := fm.CleanOrphanedFiles(validIDs)
	if err != nil {
		t.Fatalf("Failed to clean orphaned files: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleted)
	}

	// Verify session-3 is gone
	files, _ := fm.ListSessionFiles("session-3")
	if len(files) != 0 {
		t.Error("Orphaned session files should be deleted")
	}

	// Verify session-1 and session-2 still exist
	files1, _ := fm.ListSessionFiles("session-1")
	files2, _ := fm.ListSessionFiles("session-2")
	if len(files1) == 0 || len(files2) == 0 {
		t.Error("Valid session files should not be deleted")
	}
}

// TestSanitizePathComponent tests path sanitization.
func TestSanitizePathComponent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal-name", "normal-name"},
		{"name with spaces", "name with spaces"},
		{"name<with>special", "name_with_special"},
		{"name:with:colons", "name_with_colons"},
		{"name/with/slashes", "name_with_slashes"},
		{"name\\with\\backslashes", "name_with_backslashes"},
		{"name|with|pipes", "name_with_pipes"},
		{"name?with?questions", "name_with_questions"},
		{"name*with*stars", "name_with_stars"},
		{"name..with..dots", "name_with_dots"},
		{"  trimmed  ", "trimmed"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePathComponent(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePathComponent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCopyFile tests file copying.
func TestCopyFile(t *testing.T) {
	fm, cleanup := setupTestFileManager(t)
	defer cleanup()

	// Create source file
	srcData := []byte("source file content")
	srcPath, err := fm.SaveFile("session-1", "App", "source.png", srcData)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy to destination
	dstPath := filepath.Join("session-2", "App", "screenshots", "dest.png")
	err = fm.CopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination exists and has same content
	dstData, err := fm.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstData) != string(srcData) {
		t.Error("Copied file content doesn't match source")
	}
}
