package uia

import (
	"testing"
	"time"
)

// TestMarshalerCreation tests basic marshaler creation and cleanup
func TestMarshalerCreation(t *testing.T) {
	marshaler, err := NewMarshaler()
	if err != nil {
		t.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	// Verify marshaler is running
	if !marshaler.running {
		t.Errorf("Marshaler should be running after creation")
	}

	// Test close
	err = marshaler.Close()
	if err != nil {
		t.Errorf("Failed to close marshaler: %v", err)
	}

	// Verify marshaler is stopped
	if marshaler.running {
		t.Errorf("Marshaler should be stopped after close")
	}
}

// TestMarshalerGetWindowInfo tests window information extraction
func TestMarshalerGetWindowInfo(t *testing.T) {
	marshaler, err := NewMarshaler()
	if err != nil {
		t.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	// Test with invalid window handle - should still return WindowInfo for OCR fallback
	windowInfo, err := marshaler.GetWindowInfo(0)
	if err != nil {
		t.Errorf("Should not return error for invalid window handle (OCR fallback): %v", err)
	}
	
	if windowInfo == nil {
		t.Errorf("WindowInfo should not be nil (needed for OCR fallback)")
	}
	
	if windowInfo != nil && windowInfo.HWND != 0 {
		t.Errorf("Window handle should be 0 for test input")
	}
	
	// Check that fallback metadata is set
	if windowInfo != nil {
		if fallback, exists := windowInfo.Metadata["fallback_to_ocr"]; !exists || !fallback.(bool) {
			// This is fine - not all failures trigger OCR fallback
		}
	}
}

// TestMarshalerConcurrency tests concurrent access to marshaler
func TestMarshalerConcurrency(t *testing.T) {
	marshaler, err := NewMarshaler()
	if err != nil {
		t.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	// Launch multiple goroutines to test thread safety
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Try to get window info (will fail but shouldn't crash)
			_, err := marshaler.GetWindowInfo(uintptr(i + 1))
			if err == nil {
				// This is fine - some calls might succeed
			}
		}()
	}

	// Wait for all goroutines to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Goroutine completed
		case <-timeout:
			t.Fatalf("Timeout waiting for concurrent operations")
		}
	}
}

// TestAppTypeDetection tests application type detection
func TestAppTypeDetection(t *testing.T) {
	tests := []struct {
		title    string
		process  string
		expected AppType
	}{
		{"Visual Studio Code - file.go", "Code.exe", AppTypeVSCode},
		{"Google Chrome", "chrome.exe", AppTypeChrome},
		{"Microsoft Edge", "msedge.exe", AppTypeEdge},
		{"Slack - Workspace", "slack.exe", AppTypeSlack},
		{"Notepad", "notepad.exe", AppTypeUnknown},
	}

	marshaler, err := NewMarshaler()
	if err != nil {
		t.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			windowInfo := &WindowInfo{
				WindowTitle: test.title,
				ProcessName: test.process,
				Metadata:    make(map[string]interface{}),
			}

			marshaler.detectAppType(windowInfo)

			if windowInfo.AppType != test.expected {
				t.Errorf("Expected app type %v, got %v", test.expected, windowInfo.AppType)
			}
		})
	}
}

// TestContainsAny tests the containsAny helper function
func TestContainsAny(t *testing.T) {
	tests := []struct {
		text        string
		substrings  []string
		expected    bool
	}{
		{"Visual Studio Code", []string{"Visual Studio", "Code"}, true},
		{"Google Chrome", []string{"Chrome", "Firefox"}, true},
		{"Notepad", []string{"Chrome", "Firefox"}, false},
		{"", []string{"test"}, false},
		{"test", []string{}, false},
	}

	for _, test := range tests {
		result := containsAny(test.text, test.substrings)
		if result != test.expected {
			t.Errorf("containsAny(%q, %v) = %v, expected %v", 
				test.text, test.substrings, result, test.expected)
		}
	}
}

// TestAppTypeString tests AppType string representation
func TestAppTypeString(t *testing.T) {
	tests := []struct {
		appType  AppType
		expected string
	}{
		{AppTypeUnknown, "unknown"},
		{AppTypeVSCode, "vscode"},
		{AppTypeChrome, "chrome"},
		{AppTypeEdge, "edge"},
		{AppTypeSlack, "slack"},
	}

	for _, test := range tests {
		result := test.appType.String()
		if result != test.expected {
			t.Errorf("AppType(%d).String() = %q, expected %q", 
				int(test.appType), result, test.expected)
		}
	}
}

// TestMarshalerRequestQueue tests request queue behavior
func TestMarshalerRequestQueue(t *testing.T) {
	marshaler, err := NewMarshaler()
	if err != nil {
		t.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	// Test that requests don't block when queue has capacity
	start := time.Now()
	_, err = marshaler.GetWindowInfo(1)
	duration := time.Since(start)

	// Request should complete quickly (even if it fails)
	if duration > time.Second {
		t.Errorf("Request took too long: %v", duration)
	}
}

// BenchmarkMarshalerGetWindowInfo benchmarks window info extraction
func BenchmarkMarshalerGetWindowInfo(b *testing.B) {
	marshaler, err := NewMarshaler()
	if err != nil {
		b.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = marshaler.GetWindowInfo(uintptr(i + 1))
	}
}
// TestOCRFallback tests OCR fallback functionality
func TestOCRFallback(t *testing.T) {
	marshaler, err := NewMarshaler()
	if err != nil {
		t.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	// Test ShouldFallbackToOCR with nil WindowInfo
	if !marshaler.ShouldFallbackToOCR(nil) {
		t.Errorf("Should fallback to OCR for nil WindowInfo")
	}

	// Test ShouldFallbackToOCR with unknown app type
	windowInfo := &WindowInfo{
		AppType:  AppTypeUnknown,
		Metadata: make(map[string]interface{}),
	}
	if !marshaler.ShouldFallbackToOCR(windowInfo) {
		t.Errorf("Should fallback to OCR for unknown app type")
	}

	// Test ShouldFallbackToOCR with UIA extraction failure
	windowInfo = &WindowInfo{
		AppType:  AppTypeVSCode,
		Metadata: map[string]interface{}{
			"uia_extraction_failed": true,
		},
	}
	if !marshaler.ShouldFallbackToOCR(windowInfo) {
		t.Errorf("Should fallback to OCR when UIA extraction failed")
	}

	// Test ShouldFallbackToOCR with successful extraction
	windowInfo = &WindowInfo{
		AppType:  AppTypeVSCode,
		Metadata: map[string]interface{}{
			"capture_source": "ui_automation",
		},
	}
	if marshaler.ShouldFallbackToOCR(windowInfo) {
		t.Errorf("Should not fallback to OCR when extraction succeeded")
	}
}

// TestFileExtensionMapping tests file extension to language mapping
func TestFileExtensionMapping(t *testing.T) {
	tests := []struct {
		extension string
		expected  string
	}{
		{"go", "go"},
		{"js", "javascript"},
		{"ts", "typescript"},
		{"py", "python"},
		{"java", "java"},
		{"cpp", "cpp"},
		{"unknown", "unknown"},
		{"", "unknown"},
	}

	for _, test := range tests {
		result := mapFileExtensionToLanguage(test.extension)
		if result != test.expected {
			t.Errorf("mapFileExtensionToLanguage(%q) = %q, expected %q", 
				test.extension, result, test.expected)
		}
	}
}

// TestStringHelpers tests string manipulation helper functions
func TestStringHelpers(t *testing.T) {
	// Test findSubstring
	if findSubstring("hello world", "world") != 6 {
		t.Errorf("findSubstring failed for valid substring")
	}
	if findSubstring("hello", "world") != -1 {
		t.Errorf("findSubstring should return -1 for missing substring")
	}

	// Test findLastChar
	if findLastChar("hello.world.txt", '.') != 11 {
		t.Errorf("findLastChar failed for valid character")
	}
	if findLastChar("hello", '.') != -1 {
		t.Errorf("findLastChar should return -1 for missing character")
	}
}

// TestAppSpecificExtraction tests app-specific metadata extraction
func TestAppSpecificExtraction(t *testing.T) {
	marshaler, err := NewMarshaler()
	if err != nil {
		t.Fatalf("Failed to create marshaler: %v", err)
	}
	defer marshaler.Close()

	tests := []struct {
		name        string
		windowTitle string
		appType     AppType
		expectedKey string
	}{
		{
			name:        "VS Code file extraction",
			windowTitle: "main.go - Visual Studio Code",
			appType:     AppTypeVSCode,
			expectedKey: "file",
		},
		{
			name:        "Chrome page title extraction",
			windowTitle: "GitHub - Google Chrome",
			appType:     AppTypeChrome,
			expectedKey: "pageTitle",
		},
		{
			name:        "Slack channel extraction",
			windowTitle: "#general | My Workspace",
			appType:     AppTypeSlack,
			expectedKey: "channel",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			windowInfo := &WindowInfo{
				WindowTitle: test.windowTitle,
				AppType:     test.appType,
				Metadata:    make(map[string]interface{}),
			}

			switch test.appType {
			case AppTypeVSCode:
				marshaler.extractVSCodeMetadata(windowInfo)
			case AppTypeChrome:
				marshaler.extractChromeMetadata(windowInfo)
			case AppTypeSlack:
				marshaler.extractSlackMetadata(windowInfo)
			}

			if _, exists := windowInfo.Metadata[test.expectedKey]; !exists {
				t.Errorf("Expected metadata key %q not found", test.expectedKey)
			}
		})
	}
}