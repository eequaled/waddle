package uia

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
)

// Marshaler provides thread-safe access to UI Automation COM interfaces
// All UI Automation calls must be marshaled to a dedicated STA thread
type Marshaler struct {
	requests  chan *staRequest
	quit      chan struct{}
	doneChan  chan struct{} // Signals STA thread shutdown complete
	wg        sync.WaitGroup
	mu        sync.RWMutex
	running   bool
	closed    atomic.Bool // Atomic flag to prevent Close() races
}

// staRequest represents a request to execute on the STA thread
type staRequest struct {
	operation string
	hwnd      uintptr
	response  chan *staResponse
}

// staResponse represents a response from the STA thread
type staResponse struct {
	windowInfo *WindowInfo
	err        error
}

// WindowInfo contains extracted window information
type WindowInfo struct {
	HWND        uintptr
	ProcessID   uint32
	ProcessName string
	WindowTitle string
	AppType     AppType
	Metadata    map[string]interface{}
}

// AppType represents the type of application
type AppType int

const (
	AppTypeUnknown AppType = iota
	AppTypeVSCode
	AppTypeChrome
	AppTypeEdge
	AppTypeSlack
)

// String returns the string representation of AppType
func (a AppType) String() string {
	switch a {
	case AppTypeVSCode:
		return "vscode"
	case AppTypeChrome:
		return "chrome"
	case AppTypeEdge:
		return "edge"
	case AppTypeSlack:
		return "slack"
	default:
		return "unknown"
	}
}

// COM interface constants
const (
	COINIT_APARTMENTTHREADED = 0x2
	COINIT_DISABLE_OLE1DDE   = 0x4
)

// NewMarshaler creates a new UI Automation marshaler with dedicated STA thread
func NewMarshaler() (*Marshaler, error) {
	m := &Marshaler{
		requests: make(chan *staRequest, 100),
		quit:     make(chan struct{}),
		doneChan: make(chan struct{}), // Add done channel for graceful shutdown
	}

	// Start the dedicated STA thread
	m.wg.Add(1)
	go m.staThread()

	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	return m, nil
}

// staThread runs the dedicated STA thread for UI Automation COM calls
func (m *Marshaler) staThread() {
	defer m.wg.Done()
	defer func() {
		// Signal shutdown complete
		close(m.doneChan)
	}()

	// Lock this goroutine to the OS thread for STA
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Initialize COM with apartment threading
	err := ole.CoInitializeEx(0, COINIT_APARTMENTTHREADED|COINIT_DISABLE_OLE1DDE)
	if err != nil {
		// COM initialization failed - log and drain any pending requests
		// Send error response to any pending requests before exiting
		for {
			select {
			case req := <-m.requests:
				req.response <- &staResponse{err: fmt.Errorf("COM initialization failed: %w", err)}
			case <-m.quit:
				return
			default:
				return
			}
		}
	}
	defer ole.CoUninitialize()

	// Process requests on the STA thread
	for {
		select {
		case req := <-m.requests:
			m.handleSTARequest(req)
		case <-m.quit:
			// Drain remaining requests with error before exiting
			m.drainPendingRequests()
			return
		}
	}
}

// drainPendingRequests sends error responses to any remaining requests in the queue
func (m *Marshaler) drainPendingRequests() {
	for {
		select {
		case req := <-m.requests:
			req.response <- &staResponse{err: fmt.Errorf("marshaler shutting down")}
		default:
			return
		}
	}
}

// handleSTARequest processes a UI Automation request on the STA thread
// Includes panic recovery to prevent goroutine leaks
func (m *Marshaler) handleSTARequest(req *staRequest) {
	// Panic recovery - COM calls can panic
	defer func() {
		if r := recover(); r != nil {
			req.response <- &staResponse{err: fmt.Errorf("UIA panic recovered: %v", r)}
		}
	}()

	switch req.operation {
	case "GetWindowInfo":
		windowInfo, err := m.getWindowInfoSTA(req.hwnd)
		req.response <- &staResponse{windowInfo: windowInfo, err: err}
	default:
		req.response <- &staResponse{err: fmt.Errorf("unknown operation: %s", req.operation)}
	}
}

// getWindowInfoSTA extracts window information using UI Automation on STA thread
func (m *Marshaler) getWindowInfoSTA(hwnd uintptr) (*WindowInfo, error) {
	windowInfo := &WindowInfo{
		HWND:        hwnd,
		ProcessID:   0, // Will be filled by getBasicWindowInfo
		ProcessName: "", // Will be filled by getBasicWindowInfo
		WindowTitle: "", // Will be filled by getBasicWindowInfo
		AppType:     AppTypeUnknown,
		Metadata:    make(map[string]interface{}),
	}

	// Get basic window information using Win32 API
	err := m.getBasicWindowInfo(windowInfo)
	if err != nil {
		// Even if basic info fails, return windowInfo for OCR fallback
		windowInfo.Metadata["basic_info_failed"] = true
		windowInfo.Metadata["basic_info_error"] = err.Error()
	}

	// Try UI Automation extraction
	uiaErr := m.tryUIAutomationExtraction(windowInfo)
	if uiaErr != nil {
		// UI Automation failed - mark for OCR fallback
		windowInfo.Metadata["uia_extraction_failed"] = true
		windowInfo.Metadata["uia_extraction_error"] = uiaErr.Error()
		windowInfo.Metadata["capture_source"] = "ocr_fallback"
	} else {
		windowInfo.Metadata["capture_source"] = "ui_automation"
	}

	// Detect application type and extract specific metadata
	m.detectAppType(windowInfo)

	return windowInfo, nil // Always return windowInfo, even if extraction failed
}

// getBasicWindowInfo extracts basic window information using Win32 API
func (m *Marshaler) getBasicWindowInfo(info *WindowInfo) error {
	// Get process ID
	var processID uint32
	_, _, err := syscall.NewLazyDLL("user32.dll").NewProc("GetWindowThreadProcessId").Call(
		info.HWND,
		uintptr(unsafe.Pointer(&processID)),
	)
	if err != nil && err != syscall.Errno(0) {
		return fmt.Errorf("GetWindowThreadProcessId failed: %w", err)
	}
	info.ProcessID = processID

	// Get window title
	titleBuf := make([]uint16, 256)
	ret, _, err := syscall.NewLazyDLL("user32.dll").NewProc("GetWindowTextW").Call(
		info.HWND,
		uintptr(unsafe.Pointer(&titleBuf[0])),
		uintptr(len(titleBuf)),
	)
	if err != nil && err != syscall.Errno(0) {
		return fmt.Errorf("GetWindowTextW failed: %w", err)
	}
	if ret > 0 {
		info.WindowTitle = syscall.UTF16ToString(titleBuf[:ret])
	}

	// Get process name (simplified - would use proper process enumeration)
	info.ProcessName = "unknown.exe"

	return nil
}

// detectAppType detects the application type based on window title and process name
func (m *Marshaler) detectAppType(info *WindowInfo) {
	title := info.WindowTitle
	process := info.ProcessName

	// Detect VS Code
	if containsAny(title, []string{"Visual Studio Code", "VSCode"}) ||
		containsAny(process, []string{"Code.exe", "code.exe"}) {
		info.AppType = AppTypeVSCode
		m.extractVSCodeMetadata(info)
		return
	}

	// Detect Chrome
	if containsAny(title, []string{"Google Chrome"}) ||
		containsAny(process, []string{"chrome.exe"}) {
		info.AppType = AppTypeChrome
		m.extractChromeMetadata(info)
		return
	}

	// Detect Edge
	if containsAny(title, []string{"Microsoft Edge"}) ||
		containsAny(process, []string{"msedge.exe"}) {
		info.AppType = AppTypeEdge
		m.extractEdgeMetadata(info)
		return
	}

	// Detect Slack
	if containsAny(title, []string{"Slack"}) ||
		containsAny(process, []string{"slack.exe"}) {
		info.AppType = AppTypeSlack
		m.extractSlackMetadata(info)
		return
	}

	info.AppType = AppTypeUnknown
}

// extractVSCodeMetadata extracts VS Code specific metadata
func (m *Marshaler) extractVSCodeMetadata(info *WindowInfo) {
	// Extract file name from window title
	// VS Code titles are typically: "filename.ext - Visual Studio Code"
	title := info.WindowTitle
	if len(title) > 0 {
		// Find the first " - " separator
		if dashIndex := findSubstring(title, " - "); dashIndex != -1 {
			filename := title[:dashIndex]
			info.Metadata["file"] = filename
			
			// Extract language from file extension
			if dotIndex := findLastChar(filename, '.'); dotIndex != -1 {
				ext := filename[dotIndex+1:]
				info.Metadata["language"] = mapFileExtensionToLanguage(ext)
			} else {
				info.Metadata["language"] = "unknown"
			}
		} else {
			info.Metadata["file"] = "unknown.txt"
			info.Metadata["language"] = "unknown"
		}
	} else {
		info.Metadata["file"] = "unknown.txt"
		info.Metadata["language"] = "unknown"
	}
	
	// Git branch would be extracted from status bar via UI Automation
	// For now, use placeholder
	info.Metadata["gitBranch"] = "main"
	info.Metadata["extractionMethod"] = "title_parsing"
}

// extractChromeMetadata extracts Chrome specific metadata
func (m *Marshaler) extractChromeMetadata(info *WindowInfo) {
	// Extract URL and page title from window title
	// Chrome titles are typically: "Page Title - Google Chrome"
	title := info.WindowTitle
	if len(title) > 0 {
		// Find the last " - Google Chrome" separator
		chromeIndex := findSubstring(title, " - Google Chrome")
		if chromeIndex != -1 {
			pageTitle := title[:chromeIndex]
			info.Metadata["pageTitle"] = pageTitle
		} else {
			info.Metadata["pageTitle"] = title
		}
	} else {
		info.Metadata["pageTitle"] = "unknown"
	}
	
	// URL would be extracted from address bar via UI Automation
	// For now, use placeholder
	info.Metadata["url"] = "unknown"
	info.Metadata["extractionMethod"] = "title_parsing"
}

// extractEdgeMetadata extracts Edge specific metadata
func (m *Marshaler) extractEdgeMetadata(info *WindowInfo) {
	// Extract URL and page title from window title
	// Edge titles are typically: "Page Title - Microsoft Edge"
	title := info.WindowTitle
	if len(title) > 0 {
		// Find the last " - Microsoft Edge" separator
		edgeIndex := findSubstring(title, " - Microsoft Edge")
		if edgeIndex != -1 {
			pageTitle := title[:edgeIndex]
			info.Metadata["pageTitle"] = pageTitle
		} else {
			info.Metadata["pageTitle"] = title
		}
	} else {
		info.Metadata["pageTitle"] = "unknown"
	}
	
	// URL would be extracted from address bar via UI Automation
	// For now, use placeholder
	info.Metadata["url"] = "unknown"
	info.Metadata["extractionMethod"] = "title_parsing"
}

// extractSlackMetadata extracts Slack specific metadata
func (m *Marshaler) extractSlackMetadata(info *WindowInfo) {
	// Extract channel and workspace from window title
	// Slack titles are typically: "#channel-name | Workspace Name"
	title := info.WindowTitle
	if len(title) > 0 {
		// Find the " | " separator
		if pipeIndex := findSubstring(title, " | "); pipeIndex != -1 {
			channelPart := title[:pipeIndex]
			workspacePart := title[pipeIndex+3:]
			
			info.Metadata["channel"] = channelPart
			info.Metadata["workspace"] = workspacePart
		} else {
			// Fallback: use entire title as workspace
			info.Metadata["channel"] = "unknown"
			info.Metadata["workspace"] = title
		}
	} else {
		info.Metadata["channel"] = "unknown"
		info.Metadata["workspace"] = "unknown"
	}
	
	info.Metadata["extractionMethod"] = "title_parsing"
}

// containsAny checks if the text contains any of the substrings
func containsAny(text string, substrings []string) bool {
	for _, substr := range substrings {
		if len(text) >= len(substr) {
			for i := 0; i <= len(text)-len(substr); i++ {
				if text[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// GetWindowInfo extracts window information using UI Automation (thread-safe)
func (m *Marshaler) GetWindowInfo(hwnd uintptr) (*WindowInfo, error) {
	// Check if marshaler is closed using atomic flag
	if m.closed.Load() {
		return nil, fmt.Errorf("marshaler is closed")
	}

	m.mu.RLock()
	if !m.running {
		m.mu.RUnlock()
		return nil, fmt.Errorf("marshaler not running")
	}
	m.mu.RUnlock()

	// Create request with BUFFERED response channel (size 1) to prevent deadlock
	// This is critical: if STA thread sends response before we read, it won't block
	response := make(chan *staResponse, 1)
	req := &staRequest{
		operation: "GetWindowInfo",
		hwnd:      hwnd,
		response:  response,
	}

	// Send request to STA thread with timeout
	select {
	case m.requests <- req:
		// Request sent successfully
	case <-m.doneChan:
		// Marshaler is shutting down
		return nil, fmt.Errorf("marshaler closed during request")
	case <-time.After(5 * time.Second):
		// Request queue full or blocked
		return nil, fmt.Errorf("request queue full or timeout")
	}

	// Wait for response with timeout
	select {
	case resp := <-response:
		// If UI Automation extraction failed, mark for OCR fallback
		if resp.err != nil && resp.windowInfo != nil {
			resp.windowInfo.Metadata["uia_failed"] = true
			resp.windowInfo.Metadata["fallback_to_ocr"] = true
			resp.windowInfo.Metadata["uia_error"] = resp.err.Error()
		}
		return resp.windowInfo, resp.err
	case <-m.doneChan:
		// Marshaler closed while waiting for response
		return nil, fmt.Errorf("marshaler closed while waiting for response")
	case <-time.After(10 * time.Second):
		// Response timeout
		return nil, fmt.Errorf("response timeout")
	}
}

// Close stops the marshaler and cleans up resources
func (m *Marshaler) Close() error {
	// Use atomic flag to prevent double-close and race conditions
	if m.closed.Swap(true) {
		// Already closed
		return nil
	}

	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	// Signal STA thread to quit
	close(m.quit)

	// Wait for STA thread to finish with timeout
	select {
	case <-m.doneChan:
		// STA thread finished gracefully
	case <-time.After(5 * time.Second):
		// Timeout waiting for STA thread - force continue
	}

	// Wait for WaitGroup
	m.wg.Wait()

	// Close request channel (safe now that STA thread is done)
	close(m.requests)

	return nil
}

// Helper functions for string manipulation

// findSubstring finds the first occurrence of substr in text
func findSubstring(text, substr string) int {
	if len(substr) == 0 || len(text) < len(substr) {
		return -1
	}
	
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// findLastChar finds the last occurrence of char in text
func findLastChar(text string, char byte) int {
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == char {
			return i
		}
	}
	return -1
}

// mapFileExtensionToLanguage maps file extensions to programming languages
func mapFileExtensionToLanguage(ext string) string {
	languageMap := map[string]string{
		"go":     "go",
		"js":     "javascript",
		"ts":     "typescript",
		"py":     "python",
		"java":   "java",
		"cpp":    "cpp",
		"c":      "c",
		"cs":     "csharp",
		"php":    "php",
		"rb":     "ruby",
		"rs":     "rust",
		"swift":  "swift",
		"kt":     "kotlin",
		"scala":  "scala",
		"html":   "html",
		"css":    "css",
		"scss":   "scss",
		"sass":   "sass",
		"less":   "less",
		"json":   "json",
		"xml":    "xml",
		"yaml":   "yaml",
		"yml":    "yaml",
		"toml":   "toml",
		"md":     "markdown",
		"txt":    "plaintext",
		"sql":    "sql",
		"sh":     "shell",
		"bash":   "shell",
		"zsh":    "shell",
		"ps1":    "powershell",
		"bat":    "batch",
		"cmd":    "batch",
	}
	
	if language, exists := languageMap[ext]; exists {
		return language
	}
	return "unknown"
}
// tryUIAutomationExtraction attempts to extract information using UI Automation
func (m *Marshaler) tryUIAutomationExtraction(windowInfo *WindowInfo) error {
	// This is where we would implement actual UI Automation extraction
	// For now, simulate the attempt and potential failure
	
	// Simulate UI Automation availability check
	if windowInfo.AppType == AppTypeUnknown {
		return fmt.Errorf("UI Automation not available for unknown application type")
	}
	
	// Simulate extraction based on app type
	switch windowInfo.AppType {
	case AppTypeVSCode:
		return m.tryVSCodeUIAutomation(windowInfo)
	case AppTypeChrome:
		return m.tryChromeUIAutomation(windowInfo)
	case AppTypeEdge:
		return m.tryEdgeUIAutomation(windowInfo)
	case AppTypeSlack:
		return m.trySlackUIAutomation(windowInfo)
	default:
		return fmt.Errorf("UI Automation not implemented for app type: %s", windowInfo.AppType.String())
	}
}

// tryVSCodeUIAutomation attempts VS Code specific UI Automation extraction
func (m *Marshaler) tryVSCodeUIAutomation(windowInfo *WindowInfo) error {
	// In a real implementation, this would:
	// 1. Get IUIAutomationElement from window handle
	// 2. Navigate to tab bar to find active tab
	// 3. Extract file name from tab text
	// 4. Navigate to status bar to get language and git branch
	// 5. Use accessibility patterns to get text content
	
	// For now, simulate success/failure based on window title availability
	if windowInfo.WindowTitle == "" {
		return fmt.Errorf("cannot extract VS Code info: window title empty")
	}
	
	// Enhanced metadata would be set here
	windowInfo.Metadata["uia_vscode_extraction"] = "simulated"
	return nil
}

// tryChromeUIAutomation attempts Chrome specific UI Automation extraction
func (m *Marshaler) tryChromeUIAutomation(windowInfo *WindowInfo) error {
	// In a real implementation, this would:
	// 1. Get IUIAutomationElement from window handle
	// 2. Navigate to address bar (Omnibox)
	// 3. Extract URL from address bar value
	// 4. Get page title from document
	// 5. Handle multiple tabs if present
	
	// For now, simulate success/failure
	if windowInfo.WindowTitle == "" {
		return fmt.Errorf("cannot extract Chrome info: window title empty")
	}
	
	windowInfo.Metadata["uia_chrome_extraction"] = "simulated"
	return nil
}

// tryEdgeUIAutomation attempts Edge specific UI Automation extraction
func (m *Marshaler) tryEdgeUIAutomation(windowInfo *WindowInfo) error {
	// Similar to Chrome implementation
	if windowInfo.WindowTitle == "" {
		return fmt.Errorf("cannot extract Edge info: window title empty")
	}
	
	windowInfo.Metadata["uia_edge_extraction"] = "simulated"
	return nil
}

// trySlackUIAutomation attempts Slack specific UI Automation extraction
func (m *Marshaler) trySlackUIAutomation(windowInfo *WindowInfo) error {
	// In a real implementation, this would:
	// 1. Get IUIAutomationElement from window handle
	// 2. Navigate to channel header
	// 3. Extract channel name and workspace
	// 4. Get current conversation context
	
	if windowInfo.WindowTitle == "" {
		return fmt.Errorf("cannot extract Slack info: window title empty")
	}
	
	windowInfo.Metadata["uia_slack_extraction"] = "simulated"
	return nil
}

// ShouldFallbackToOCR determines if OCR fallback should be used
func (m *Marshaler) ShouldFallbackToOCR(windowInfo *WindowInfo) bool {
	if windowInfo == nil {
		return true
	}
	
	// Check if UI Automation extraction failed
	if failed, exists := windowInfo.Metadata["uia_extraction_failed"]; exists && failed.(bool) {
		return true
	}
	
	// Check if basic info extraction failed
	if failed, exists := windowInfo.Metadata["basic_info_failed"]; exists && failed.(bool) {
		return true
	}
	
	// Check if app type is unknown (no structured extraction available)
	if windowInfo.AppType == AppTypeUnknown {
		return true
	}
	
	return false
}