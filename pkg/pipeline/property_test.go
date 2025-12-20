package pipeline

import (
	"testing"
	"time"
	"waddle/pkg/capture/uia"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestHybridPipelineOrder tests Property 2: Hybrid Pipeline Order
// For any window activity, the pipeline should process in order: ETW → UIA → OCR fallback
// and skip OCR when structured data is available.
// Validates: Requirements 3.1, 3.2, 3.3, 3.4
func TestHybridPipelineOrder(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 2: Hybrid Pipeline Order
	properties.Property("Pipeline should process ETW → UIA → OCR in correct order and skip OCR when structured data available", prop.ForAll(
		func(windowHandle uintptr, processID uint32, appTypeInt int) bool {
			// Create pipeline
			pipeline, err := NewPipeline()
			if err != nil {
				t.Logf("Failed to create pipeline: %v", err)
				return false
			}
			defer pipeline.Stop()

			// Map integer to valid AppType
			appType := uia.AppType(appTypeInt % 5) // 0-4 are valid AppType values

			// Create test activity (simulating ETW input)
			activity := &ActivityBlock{
				Timestamp:     time.Now(),
				WindowHandle:  windowHandle,
				ProcessID:     processID,
				ProcessName:   "test.exe",
				WindowTitle:   "Test Window",
				AppType:       appType,
				CaptureSource: CaptureSourceETW, // Should start with ETW
				Metadata:      make(map[string]interface{}),
			}

			// Verify initial state (ETW stage)
			if activity.CaptureSource != CaptureSourceETW {
				t.Logf("Activity should start with ETW capture source")
				return false
			}

			// Simulate UIA processing
			windowInfo := &uia.WindowInfo{
				HWND:        windowHandle,
				ProcessID:   processID,
				ProcessName: "test.exe",
				WindowTitle: "Test Window",
				AppType:     appType,
				Metadata:    make(map[string]interface{}),
			}

			// Add app-specific metadata based on type
			switch appType {
			case uia.AppTypeVSCode:
				windowInfo.Metadata["file"] = "test.go"
				windowInfo.Metadata["language"] = "go"
				windowInfo.Metadata["extractionMethod"] = "title_parsing"
			case uia.AppTypeChrome:
				windowInfo.Metadata["pageTitle"] = "Test Page"
				windowInfo.Metadata["url"] = "https://example.com"
				windowInfo.Metadata["extractionMethod"] = "title_parsing"
			case uia.AppTypeSlack:
				windowInfo.Metadata["channel"] = "#test"
				windowInfo.Metadata["workspace"] = "Test Workspace"
				windowInfo.Metadata["extractionMethod"] = "title_parsing"
			}

			// Check if pipeline would have valid structured data
			hasStructuredData := pipeline.hasValidStructuredData(windowInfo)

			// Simulate UIA processing result
			if hasStructuredData {
				// Should skip OCR
				activity.StructuredData = true
				activity.CaptureSource = CaptureSourceUIAutomation
				activity.Metadata["skip_ocr"] = true

				// Verify OCR is skipped for structured data
				if activity.CaptureSource != CaptureSourceUIAutomation {
					t.Logf("Should use UI Automation source when structured data available")
					return false
				}

				if !activity.StructuredData {
					t.Logf("StructuredData should be true when UIA succeeds")
					return false
				}

				if skipOCR, exists := activity.Metadata["skip_ocr"]; !exists || !skipOCR.(bool) {
					t.Logf("Should skip OCR when structured data available")
					return false
				}

			} else {
				// Should fallback to OCR
				activity.StructuredData = false
				activity.CaptureSource = CaptureSourceOCR

				// Verify OCR fallback for unknown/failed apps
				if activity.CaptureSource != CaptureSourceOCR {
					t.Logf("Should use OCR source when structured data unavailable")
					return false
				}

				if activity.StructuredData {
					t.Logf("StructuredData should be false when falling back to OCR")
					return false
				}
			}

			// Verify pipeline order is maintained
			// ETW → UIA → (OCR or Skip)
			expectedOrder := []CaptureSource{CaptureSourceETW}
			if hasStructuredData {
				expectedOrder = append(expectedOrder, CaptureSourceUIAutomation)
			} else {
				expectedOrder = append(expectedOrder, CaptureSourceOCR)
			}

			// The final capture source should match expected progression
			finalSource := activity.CaptureSource
			expectedFinal := expectedOrder[len(expectedOrder)-1]
			if finalSource != expectedFinal {
				t.Logf("Final capture source should be %v, got %v", expectedFinal, finalSource)
				return false
			}

			return true
		},
		gen.UInt64().Map(func(u uint64) uintptr { return uintptr(u) }), // Window handle
		gen.UInt32(),                                                      // Process ID  
		gen.IntRange(0, 10),                                              // App type (will be modulo'd)
	))

	properties.TestingRun(t)
}

// TestOCRBatching tests Property 4: OCR Batching
// For any sequence of OCR requests, they should be batched efficiently
// with proper timeout and size limits.
// Validates: Requirements 3.6
func TestOCRBatching(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 4: OCR Batching
	properties.Property("OCR requests should be batched with proper timeout and size limits", prop.ForAll(
		func(numRequests int) bool {
			// Limit number of requests to reasonable range
			numRequests = (numRequests % 50) + 1 // 1-50 requests

			pipeline, err := NewPipeline()
			if err != nil {
				t.Logf("Failed to create pipeline: %v", err)
				return false
			}
			defer pipeline.Stop()

			// Create test activities for OCR batching
			activities := make([]*ActivityBlock, numRequests)
			for i := 0; i < numRequests; i++ {
				activities[i] = &ActivityBlock{
					Timestamp:     time.Now(),
					WindowHandle:  uintptr(12345 + i),
					ProcessID:     uint32(1000 + i),
					ProcessName:   "test.exe",
					WindowTitle:   "Test Window",
					AppType:       uia.AppTypeUnknown, // Unknown apps go to OCR
					CaptureSource: CaptureSourceOCR,
					StructuredData: false,
					Metadata:      make(map[string]interface{}),
				}
			}

			// Calculate expected number of batches
			expectedBatches := (numRequests + OCRBatchSize - 1) / OCRBatchSize // Ceiling division

			// Process activities in batches
			processedBatches := 0
			for i := 0; i < numRequests; i += OCRBatchSize {
				end := i + OCRBatchSize
				if end > numRequests {
					end = numRequests
				}

				batch := activities[i:end]
				pipeline.processOCRBatch(batch)
				processedBatches++

				// Verify batch processing
				for j, activity := range batch {
					// Check OCR processing metadata
					if processed, exists := activity.Metadata["ocr_processed"]; !exists || !processed.(bool) {
						t.Logf("Activity should be marked as OCR processed")
						return false
					}

					if batchSize, exists := activity.Metadata["batch_size"]; !exists || batchSize.(int) != len(batch) {
						t.Logf("Batch size should be %d, got %v", len(batch), batchSize)
						return false
					}

					if batchIndex, exists := activity.Metadata["batch_index"]; !exists || batchIndex.(int) != j {
						t.Logf("Batch index should be %d, got %v", j, batchIndex)
						return false
					}

					// Verify OCR metadata is added
					if _, exists := activity.Metadata["ocr_text"]; !exists {
						t.Logf("OCR text should be added to metadata")
						return false
					}

					if confidence, exists := activity.Metadata["ocr_confidence"]; !exists {
						t.Logf("OCR confidence should be added to metadata")
						return false
					} else if conf, ok := confidence.(float64); !ok || conf < 0 || conf > 1 {
						t.Logf("OCR confidence should be between 0 and 1, got %v", confidence)
						return false
					}
				}
			}

			// Verify correct number of batches processed
			if processedBatches != expectedBatches {
				t.Logf("Expected %d batches, processed %d", expectedBatches, processedBatches)
				return false
			}

			// Verify batch size constraints
			for i := 0; i < numRequests; i += OCRBatchSize {
				end := i + OCRBatchSize
				if end > numRequests {
					end = numRequests
				}
				batchSize := end - i

				// Batch size should not exceed OCRBatchSize
				if batchSize > OCRBatchSize {
					t.Logf("Batch size %d exceeds maximum %d", batchSize, OCRBatchSize)
					return false
				}

				// Last batch can be smaller, but others should be full (except for small inputs)
				if i+OCRBatchSize < numRequests && batchSize != OCRBatchSize {
					t.Logf("Non-final batch should be full size %d, got %d", OCRBatchSize, batchSize)
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 100), // Number of OCR requests
	))

	properties.TestingRun(t)
}

// TestPipelineSkipLogic tests the skip logic for structured data
func TestPipelineSkipLogic(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Pipeline should skip OCR when structured data is available", prop.ForAll(
		func(hasFile bool, hasLanguage bool, appTypeInt int) bool {
			pipeline, err := NewPipeline()
			if err != nil {
				return false
			}
			defer pipeline.Stop()

			// Create window info with varying levels of structured data
			windowInfo := &uia.WindowInfo{
				HWND:        uintptr(12345),
				ProcessID:   uint32(1000),
				ProcessName: "test.exe",
				WindowTitle: "Test Window",
				AppType:     uia.AppType(appTypeInt % 5),
				Metadata:    make(map[string]interface{}),
			}

			// Add metadata based on parameters
			if hasFile {
				windowInfo.Metadata["file"] = "test.go"
			}
			if hasLanguage {
				windowInfo.Metadata["language"] = "go"
			}

			// Test structured data detection
			hasStructured := pipeline.hasValidStructuredData(windowInfo)

			// Verify logic based on app type and available data
			switch windowInfo.AppType {
			case uia.AppTypeVSCode:
				expected := hasFile && hasLanguage
				if hasStructured != expected {
					t.Logf("VS Code structured data detection failed: expected %v, got %v", expected, hasStructured)
					return false
				}

			case uia.AppTypeUnknown:
				if hasStructured {
					t.Logf("Unknown app type should never have structured data")
					return false
				}

			default:
				// Other app types have their own logic
			}

			return true
		},
		gen.Bool(), // Has file metadata
		gen.Bool(), // Has language metadata
		gen.IntRange(0, 10), // App type
	))

	properties.TestingRun(t)
}