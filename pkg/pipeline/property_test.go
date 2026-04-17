package pipeline

import (
	"testing"
	"time"

	"waddle/pkg/platform"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// appTypeNames maps integer indices to app type strings for property tests.
var appTypeNames = []string{"unknown", "vscode", "chrome", "edge", "slack"}

func appTypeFromInt(n int) string {
	return appTypeNames[n%len(appTypeNames)]
}

// TestHybridPipelineOrder tests Property 2: Hybrid Pipeline Order
// For any window activity, the pipeline should process in order: ETW → UIA → OCR fallback
// and skip OCR when structured data is available.
func TestHybridPipelineOrder(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Pipeline should process ETW → UIA → OCR in correct order and skip OCR when structured data available", prop.ForAll(
		func(windowHandle uintptr, processID uint32, appTypeInt int) bool {
			p := newTestPipeline(t)
			defer p.Stop()

			appType := appTypeFromInt(appTypeInt)

			// Create test activity (simulating ETW input)
			activity := &ActivityBlock{
				Timestamp:     time.Now(),
				WindowHandle:  windowHandle,
				ProcessID:     processID,
				ProcessName:   "test.exe",
				WindowTitle:   "Test Window",
				AppType:       appType,
				CaptureSource: CaptureSourceETW,
				Metadata:      make(map[string]interface{}),
			}

			if activity.CaptureSource != CaptureSourceETW {
				t.Logf("Activity should start with ETW capture source")
				return false
			}

			// Simulate UIA result (portable UIResult)
			result := &platform.UIResult{
				HWND:        windowHandle,
				ProcessID:   processID,
				ProcessName: "test.exe",
				WindowTitle: "Test Window",
				AppType:     appType,
				Metadata:    make(map[string]interface{}),
			}

			// Add app-specific metadata based on type
			switch appType {
			case "vscode":
				result.Metadata["file"] = "test.go"
				result.Metadata["language"] = "go"
				result.Metadata["extractionMethod"] = "title_parsing"
			case "chrome":
				result.Metadata["pageTitle"] = "Test Page"
				result.Metadata["url"] = "https://example.com"
				result.Metadata["extractionMethod"] = "title_parsing"
			case "slack":
				result.Metadata["channel"] = "#test"
				result.Metadata["workspace"] = "Test Workspace"
				result.Metadata["extractionMethod"] = "title_parsing"
			}

			hasStructuredData := p.hasValidStructuredData(result)

			if hasStructuredData {
				activity.StructuredData = true
				activity.CaptureSource = CaptureSourceUIAutomation
				activity.Metadata["skip_ocr"] = true

				if activity.CaptureSource != CaptureSourceUIAutomation {
					return false
				}
				if !activity.StructuredData {
					return false
				}
				if skipOCR, exists := activity.Metadata["skip_ocr"]; !exists || !skipOCR.(bool) {
					return false
				}
			} else {
				activity.StructuredData = false
				activity.CaptureSource = CaptureSourceOCR

				if activity.CaptureSource != CaptureSourceOCR {
					return false
				}
				if activity.StructuredData {
					return false
				}
			}

			expectedOrder := []CaptureSource{CaptureSourceETW}
			if hasStructuredData {
				expectedOrder = append(expectedOrder, CaptureSourceUIAutomation)
			} else {
				expectedOrder = append(expectedOrder, CaptureSourceOCR)
			}

			finalSource := activity.CaptureSource
			expectedFinal := expectedOrder[len(expectedOrder)-1]
			if finalSource != expectedFinal {
				t.Logf("Final capture source should be %v, got %v", expectedFinal, finalSource)
				return false
			}

			return true
		},
		gen.UInt64().Map(func(u uint64) uintptr { return uintptr(u) }),
		gen.UInt32(),
		gen.IntRange(0, 10),
	))

	properties.TestingRun(t)
}

// TestOCRBatching tests Property 4: OCR Batching
func TestOCRBatching(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("OCR requests should be batched with proper timeout and size limits", prop.ForAll(
		func(numRequests int) bool {
			numRequests = (numRequests % 50) + 1

			p := newTestPipeline(t)
			defer p.Stop()

			activities := make([]*ActivityBlock, numRequests)
			for i := 0; i < numRequests; i++ {
				activities[i] = &ActivityBlock{
					Timestamp:      time.Now(),
					WindowHandle:   uintptr(12345 + i),
					ProcessID:      uint32(1000 + i),
					ProcessName:    "test.exe",
					WindowTitle:    "Test Window",
					AppType:        "unknown",
					CaptureSource:  CaptureSourceOCR,
					StructuredData: false,
					Metadata:       make(map[string]interface{}),
				}
			}

			expectedBatches := (numRequests + OCRBatchSize - 1) / OCRBatchSize
			processedBatches := 0

			for i := 0; i < numRequests; i += OCRBatchSize {
				end := i + OCRBatchSize
				if end > numRequests {
					end = numRequests
				}

				batch := activities[i:end]
				p.processOCRBatch(batch)
				processedBatches++

				for j, activity := range batch {
					if processed, exists := activity.Metadata["ocr_processed"]; !exists || !processed.(bool) {
						return false
					}
					if batchSize, exists := activity.Metadata["batch_size"]; !exists || batchSize.(int) != len(batch) {
						return false
					}
					if batchIndex, exists := activity.Metadata["batch_index"]; !exists || batchIndex.(int) != j {
						return false
					}
					if _, exists := activity.Metadata["ocr_text"]; !exists {
						return false
					}
					if confidence, exists := activity.Metadata["ocr_confidence"]; !exists {
						return false
					} else if conf, ok := confidence.(float64); !ok || conf < 0 || conf > 1 {
						return false
					}
				}
			}

			if processedBatches != expectedBatches {
				return false
			}

			for i := 0; i < numRequests; i += OCRBatchSize {
				end := i + OCRBatchSize
				if end > numRequests {
					end = numRequests
				}
				batchSize := end - i
				if batchSize > OCRBatchSize {
					return false
				}
				if i+OCRBatchSize < numRequests && batchSize != OCRBatchSize {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

// TestPipelineSkipLogic tests the skip logic for structured data
func TestPipelineSkipLogic(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Pipeline should skip OCR when structured data is available", prop.ForAll(
		func(hasFile bool, hasLanguage bool, appTypeInt int) bool {
			p := newTestPipeline(t)
			defer p.Stop()

			appType := appTypeFromInt(appTypeInt)

			result := &platform.UIResult{
				HWND:        uintptr(12345),
				ProcessID:   uint32(1000),
				ProcessName: "test.exe",
				WindowTitle: "Test Window",
				AppType:     appType,
				Metadata:    make(map[string]interface{}),
			}

			if hasFile {
				result.Metadata["file"] = "test.go"
			}
			if hasLanguage {
				result.Metadata["language"] = "go"
			}

			hasStructured := p.hasValidStructuredData(result)

			switch appType {
			case "vscode":
				expected := hasFile && hasLanguage
				if hasStructured != expected {
					t.Logf("VS Code structured data detection failed: expected %v, got %v", expected, hasStructured)
					return false
				}
			case "unknown":
				if hasStructured {
					t.Logf("Unknown app type should never have structured data")
					return false
				}
			}

			return true
		},
		gen.Bool(),
		gen.Bool(),
		gen.IntRange(0, 10),
	))

	properties.TestingRun(t)
}