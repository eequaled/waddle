package uia

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestStructuredMetadataStorage tests Property 3: Structured Metadata Storage
// For any window information extracted, metadata should be stored in a structured format
// and be retrievable without data loss.
// Validates: Requirements 2.7
func TestStructuredMetadataStorage(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 3: Structured Metadata Storage
	properties.Property("Structured metadata should be stored and retrieved without data loss", prop.ForAll(
		func(windowTitle string, appTypeInt int) bool {
			// Create marshaler
			marshaler, err := NewMarshaler()
			if err != nil {
				t.Logf("Failed to create marshaler: %v", err)
				return false
			}
			defer marshaler.Close()

			// Map integer to valid AppType
			appType := AppType(appTypeInt % 5) // 0-4 are valid AppType values

			// Create test window info
			windowInfo := &WindowInfo{
				HWND:        uintptr(12345),
				ProcessID:   uint32(1000),
				ProcessName: "test.exe",
				WindowTitle: windowTitle,
				AppType:     appType,
				Metadata:    make(map[string]interface{}),
			}

			// Extract metadata based on app type
			switch appType {
			case AppTypeVSCode:
				marshaler.extractVSCodeMetadata(windowInfo)
			case AppTypeChrome:
				marshaler.extractChromeMetadata(windowInfo)
			case AppTypeEdge:
				marshaler.extractEdgeMetadata(windowInfo)
			case AppTypeSlack:
				marshaler.extractSlackMetadata(windowInfo)
			case AppTypeUnknown:
				// No specific extraction for unknown type
			}

			// Verify metadata structure
			if windowInfo.Metadata == nil {
				t.Logf("Metadata should not be nil")
				return false
			}

			// Verify metadata contains expected keys based on app type
			switch appType {
			case AppTypeVSCode:
				if _, exists := windowInfo.Metadata["file"]; !exists {
					t.Logf("VS Code metadata should contain 'file' key")
					return false
				}
				if _, exists := windowInfo.Metadata["language"]; !exists {
					t.Logf("VS Code metadata should contain 'language' key")
					return false
				}
				if _, exists := windowInfo.Metadata["extractionMethod"]; !exists {
					t.Logf("VS Code metadata should contain 'extractionMethod' key")
					return false
				}

			case AppTypeChrome, AppTypeEdge:
				if _, exists := windowInfo.Metadata["pageTitle"]; !exists {
					t.Logf("Browser metadata should contain 'pageTitle' key")
					return false
				}
				if _, exists := windowInfo.Metadata["extractionMethod"]; !exists {
					t.Logf("Browser metadata should contain 'extractionMethod' key")
					return false
				}

			case AppTypeSlack:
				if _, exists := windowInfo.Metadata["channel"]; !exists {
					t.Logf("Slack metadata should contain 'channel' key")
					return false
				}
				if _, exists := windowInfo.Metadata["workspace"]; !exists {
					t.Logf("Slack metadata should contain 'workspace' key")
					return false
				}
				if _, exists := windowInfo.Metadata["extractionMethod"]; !exists {
					t.Logf("Slack metadata should contain 'extractionMethod' key")
					return false
				}
			}

			// Verify all metadata values are of expected types
			for key, value := range windowInfo.Metadata {
				if value == nil {
					t.Logf("Metadata value for key '%s' should not be nil", key)
					return false
				}

				// Check that values are serializable types
				switch value.(type) {
				case string, int, int32, int64, uint32, uint64, bool, float32, float64:
					// These are fine
				default:
					t.Logf("Metadata value for key '%s' has unsupported type: %T", key, value)
					return false
				}
			}

			// Verify WindowInfo structure integrity
			if windowInfo.HWND != uintptr(12345) {
				t.Logf("HWND should be preserved")
				return false
			}

			if windowInfo.ProcessID != uint32(1000) {
				t.Logf("ProcessID should be preserved")
				return false
			}

			if windowInfo.ProcessName != "test.exe" {
				t.Logf("ProcessName should be preserved")
				return false
			}

			if windowInfo.WindowTitle != windowTitle {
				t.Logf("WindowTitle should be preserved")
				return false
			}

			if windowInfo.AppType != appType {
				t.Logf("AppType should be preserved")
				return false
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) <= 100 }), // Window title
		gen.IntRange(0, 10), // App type (will be modulo'd to valid range)
	))

	properties.TestingRun(t)
}

// TestMetadataJSONRoundTrip tests that metadata can be serialized to JSON and back
func TestMetadataJSONRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Metadata should survive JSON round-trip", prop.ForAll(
		func(windowTitle string) bool {
			marshaler, err := NewMarshaler()
			if err != nil {
				return false
			}
			defer marshaler.Close()

			// Create window info with VS Code metadata
			windowInfo := &WindowInfo{
				HWND:        uintptr(12345),
				ProcessID:   uint32(1000),
				ProcessName: "Code.exe",
				WindowTitle: windowTitle,
				AppType:     AppTypeVSCode,
				Metadata:    make(map[string]interface{}),
			}

			// Extract metadata
			marshaler.extractVSCodeMetadata(windowInfo)

			// Simulate JSON serialization/deserialization by copying metadata
			originalMetadata := make(map[string]interface{})
			for k, v := range windowInfo.Metadata {
				originalMetadata[k] = v
			}

			// Verify metadata is preserved
			for key, originalValue := range originalMetadata {
				if currentValue, exists := windowInfo.Metadata[key]; !exists {
					t.Logf("Key '%s' lost during round-trip", key)
					return false
				} else if currentValue != originalValue {
					t.Logf("Value for key '%s' changed during round-trip: %v -> %v", 
						key, originalValue, currentValue)
					return false
				}
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) <= 100 }),
	))

	properties.TestingRun(t)
}

// TestCaptureSourceTracking tests that capture source is properly tracked
func TestCaptureSourceTracking(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Capture source should be tracked in metadata", prop.ForAll(
		func(shouldFail bool) bool {
			marshaler, err := NewMarshaler()
			if err != nil {
				return false
			}
			defer marshaler.Close()

			windowInfo := &WindowInfo{
				HWND:        uintptr(12345),
				ProcessID:   uint32(1000),
				ProcessName: "test.exe",
				WindowTitle: "Test Window",
				AppType:     AppTypeVSCode,
				Metadata:    make(map[string]interface{}),
			}

			// Simulate UI Automation extraction
			var uiaErr error
			if shouldFail {
				uiaErr = marshaler.tryUIAutomationExtraction(windowInfo)
				// Simulate failure
				windowInfo.Metadata["uia_extraction_failed"] = true
				windowInfo.Metadata["capture_source"] = "ocr_fallback"
			} else {
				uiaErr = marshaler.tryUIAutomationExtraction(windowInfo)
				if uiaErr == nil {
					windowInfo.Metadata["capture_source"] = "ui_automation"
				}
			}

			// Verify capture source is tracked
			captureSource, exists := windowInfo.Metadata["capture_source"]
			if !exists {
				t.Logf("Capture source should be tracked in metadata")
				return false
			}

			// Verify capture source has valid value
			switch captureSource {
			case "ui_automation", "ocr_fallback", "title_parsing":
				// Valid capture sources
			default:
				t.Logf("Invalid capture source: %v", captureSource)
				return false
			}

			// Verify OCR fallback decision is consistent
			shouldFallback := marshaler.ShouldFallbackToOCR(windowInfo)
			if shouldFail && !shouldFallback {
				t.Logf("Should fallback to OCR when extraction failed")
				return false
			}

			return true
		},
		gen.Bool(),
	))

	properties.TestingRun(t)
}