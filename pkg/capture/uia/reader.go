package uia

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// Reader provides UI Automation functionality for extracting window information
type Reader struct {
	automation *ole.IDispatch
	marshaler  *Marshaler
}

// NewReader creates a new UI Automation reader
func NewReader() (*Reader, error) {
	marshaler, err := NewMarshaler()
	if err != nil {
		return nil, fmt.Errorf("failed to create marshaler: %w", err)
	}

	r := &Reader{
		marshaler: marshaler,
	}

	return r, nil
}

// GetWindowInfo extracts comprehensive window information
func (r *Reader) GetWindowInfo(hwnd uintptr) (*WindowInfo, error) {
	// Use marshaler to safely call UI Automation on STA thread
	return r.marshaler.GetWindowInfo(hwnd)
}

// ExtractVSCodeInfo extracts VS Code specific information using UI Automation
func (r *Reader) ExtractVSCodeInfo(hwnd uintptr) (map[string]interface{}, error) {
	// This would be implemented to use UI Automation to navigate the VS Code UI tree
	// and extract file name, language, git branch, etc.
	
	metadata := make(map[string]interface{})
	
	// Placeholder implementation - real implementation would:
	// 1. Get IUIAutomationElement from window handle
	// 2. Navigate to tab bar to get current file
	// 3. Navigate to status bar to get language and git branch
	// 4. Use accessibility properties to extract text content
	
	metadata["file"] = "example.go"
	metadata["language"] = "go"
	metadata["gitBranch"] = "main"
	metadata["extractionMethod"] = "uia"
	
	return metadata, nil
}

// ExtractChromeInfo extracts Chrome specific information using UI Automation
func (r *Reader) ExtractChromeInfo(hwnd uintptr) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	
	// Placeholder implementation - real implementation would:
	// 1. Get IUIAutomationElement from window handle
	// 2. Navigate to address bar to get URL
	// 3. Get page title from document
	// 4. Extract tab information if multiple tabs
	
	metadata["url"] = "https://example.com"
	metadata["pageTitle"] = "Example Page"
	metadata["extractionMethod"] = "uia"
	
	return metadata, nil
}

// ExtractEdgeInfo extracts Edge specific information using UI Automation
func (r *Reader) ExtractEdgeInfo(hwnd uintptr) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	
	// Similar to Chrome implementation
	metadata["url"] = "https://example.com"
	metadata["pageTitle"] = "Example Page"
	metadata["extractionMethod"] = "uia"
	
	return metadata, nil
}

// ExtractSlackInfo extracts Slack specific information using UI Automation
func (r *Reader) ExtractSlackInfo(hwnd uintptr) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	
	// Placeholder implementation - real implementation would:
	// 1. Get IUIAutomationElement from window handle
	// 2. Navigate to channel header to get channel name
	// 3. Navigate to workspace selector to get workspace name
	// 4. Extract current conversation context
	
	metadata["channel"] = "#general"
	metadata["workspace"] = "Example Workspace"
	metadata["extractionMethod"] = "uia"
	
	return metadata, nil
}

// IsUIAutomationAvailable checks if UI Automation is available for the window
func (r *Reader) IsUIAutomationAvailable(hwnd uintptr) bool {
	// Check if the window supports UI Automation
	// This is a simplified check - real implementation would:
	// 1. Try to get IUIAutomationElement from window handle
	// 2. Check if element has accessible properties
	// 3. Verify that the application exposes UI Automation patterns
	
	// For now, assume UI Automation is available for known applications
	windowInfo, err := r.GetWindowInfo(hwnd)
	if err != nil {
		return false
	}
	
	// UI Automation is typically available for modern applications
	return windowInfo.AppType != AppTypeUnknown
}

// GetAccessibilityTree gets the accessibility tree for debugging/inspection
func (r *Reader) GetAccessibilityTree(hwnd uintptr) (map[string]interface{}, error) {
	// This would return a structured representation of the UI Automation tree
	// Useful for debugging and understanding application structure
	
	tree := map[string]interface{}{
		"hwnd": hwnd,
		"root": map[string]interface{}{
			"name":         "Root Element",
			"controlType":  "Window",
			"children":     []interface{}{},
			"properties":   map[string]interface{}{},
		},
	}
	
	return tree, nil
}

// Close cleans up the reader and its resources
func (r *Reader) Close() error {
	if r.marshaler != nil {
		return r.marshaler.Close()
	}
	return nil
}

// Helper functions for UI Automation COM interface management

// createUIAutomation creates the UI Automation COM interface
func createUIAutomation() (*ole.IDispatch, error) {
	// Create IUIAutomation interface
	unknown, err := oleutil.CreateObject("UIAutomation.CUIAutomation")
	if err != nil {
		return nil, fmt.Errorf("failed to create UI Automation object: %w", err)
	}
	
	automation, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		unknown.Release()
		return nil, fmt.Errorf("failed to query IDispatch interface: %w", err)
	}
	
	return automation, nil
}

// getElementFromHandle gets UI Automation element from window handle
func getElementFromHandle(automation *ole.IDispatch, hwnd uintptr) (*ole.IDispatch, error) {
	// Call IUIAutomation::ElementFromHandle
	result, err := oleutil.CallMethod(automation, "ElementFromHandle", hwnd)
	if err != nil {
		return nil, fmt.Errorf("ElementFromHandle failed: %w", err)
	}
	
	element := result.ToIDispatch()
	if element == nil {
		return nil, fmt.Errorf("ElementFromHandle returned null")
	}
	
	return element, nil
}

// getElementProperty gets a property from a UI Automation element
func getElementProperty(element *ole.IDispatch, propertyID int32) (*ole.VARIANT, error) {
	// Call IUIAutomationElement::GetCurrentPropertyValue
	result, err := oleutil.CallMethod(element, "GetCurrentPropertyValue", propertyID)
	if err != nil {
		return nil, fmt.Errorf("GetCurrentPropertyValue failed: %w", err)
	}
	
	return result, nil
}

// findChildElements finds child elements matching criteria
func findChildElements(element *ole.IDispatch, condition *ole.IDispatch) ([]*ole.IDispatch, error) {
	// Call IUIAutomationElement::FindAll
	result, err := oleutil.CallMethod(element, "FindAll", 4, condition) // TreeScope_Descendants = 4
	if err != nil {
		return nil, fmt.Errorf("FindAll failed: %w", err)
	}
	
	elementArray := result.ToIDispatch()
	if elementArray == nil {
		return nil, fmt.Errorf("FindAll returned null")
	}
	defer elementArray.Release()
	
	// Get length of array
	lengthResult, err := oleutil.GetProperty(elementArray, "Length")
	if err != nil {
		return nil, fmt.Errorf("failed to get array length: %w", err)
	}
	
	length := int(lengthResult.Val)
	elements := make([]*ole.IDispatch, 0, length)
	
	// Get each element from array
	for i := 0; i < length; i++ {
		itemResult, err := oleutil.CallMethod(elementArray, "GetElement", i)
		if err != nil {
			continue // Skip failed elements
		}
		
		item := itemResult.ToIDispatch()
		if item != nil {
			elements = append(elements, item)
		}
	}
	
	return elements, nil
}

// Win32 API helpers for basic window information

// getWindowText gets the window title using Win32 API
func getWindowText(hwnd uintptr) (string, error) {
	titleBuf := make([]uint16, 256)
	ret, _, err := syscall.NewLazyDLL("user32.dll").NewProc("GetWindowTextW").Call(
		hwnd,
		uintptr(unsafe.Pointer(&titleBuf[0])),
		uintptr(len(titleBuf)),
	)
	if err != nil && err != syscall.Errno(0) {
		return "", fmt.Errorf("GetWindowTextW failed: %w", err)
	}
	if ret == 0 {
		return "", nil
	}
	return syscall.UTF16ToString(titleBuf[:ret]), nil
}

// getWindowProcessID gets the process ID of the window owner
func getWindowProcessID(hwnd uintptr) (uint32, error) {
	var processID uint32
	_, _, err := syscall.NewLazyDLL("user32.dll").NewProc("GetWindowThreadProcessId").Call(
		hwnd,
		uintptr(unsafe.Pointer(&processID)),
	)
	if err != nil && err != syscall.Errno(0) {
		return 0, fmt.Errorf("GetWindowThreadProcessId failed: %w", err)
	}
	return processID, nil
}