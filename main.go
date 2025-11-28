package main

import (
	"fmt"
	"ideathon/pkg/ai"
	"ideathon/pkg/capture"
	"ideathon/pkg/content"
	"ideathon/pkg/ocr"
	"ideathon/pkg/processing"
	"ideathon/pkg/server"
	"ideathon/pkg/tracker"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	fmt.Println("Starting Session Tracker...")

	// 1. Initialize Storage Root
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}
	// Base directory for all sessions
	sessionRootDir := filepath.Join(homeDir, "OneDrive", "Documents", "ideathon", "sessions")
	if err := os.MkdirAll(sessionRootDir, 0755); err != nil {
		fmt.Printf("Error creating session root directory: %v\n", err)
		return
	}
	fmt.Printf("Saving sessions to: %s\n", sessionRootDir)

	// 2. Initialize Modules
	trackerPoller := tracker.NewPoller()
	clipboardMonitor := content.NewMonitor()

	// 3. Start Background Processors
	// OCR Batch Processor (Clean up screenshots, extract text)
	batchProcessor := processing.NewBatchProcessor(sessionRootDir)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := batchProcessor.ProcessAll(); err != nil {
				fmt.Printf("Batch processing error: %v\n", err)
			}
		}
	}()

	// AI Memory Manager (Summarize text)
	// Using gemma2:2b as a small, efficient local model
	ollamaClient := ai.NewOllamaClient("", "gemma2:2b")
	memoryManager := processing.NewMemoryManager(sessionRootDir, ollamaClient)
	go func() {
		// Initial run to process existing data
		if err := memoryManager.ProcessMemories(); err != nil {
			fmt.Printf("Memory processing error: %v\n", err)
		}

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Println("Running AI Memory processing...")
			if err := memoryManager.ProcessMemories(); err != nil {
				fmt.Printf("Memory processing error: %v\n", err)
			}
		}
	}()

	// 4. Start API Server
	isPaused := &atomic.Bool{}
	apiServer := server.NewServer(sessionRootDir, "8080", isPaused)
	apiServer.Start()

	// 5. Start Monitoring
	focusChan := trackerPoller.Start()
	clipboardChan := clipboardMonitor.Start()

	// 6. Handle OS Signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Monitoring active... Press Ctrl+C to stop.")

	// 7. State Management
	var (
		currentAppName string
		currentTitle   string
		currentHandle  syscall.Handle
		currentDir     string
	)

	// Ticker for screenshots (e.g., every 5 seconds)
	screenshotTicker := time.NewTicker(5 * time.Second)
	defer screenshotTicker.Stop()

	// Helper to sanitize filenames
	sanitize := func(name string) string {
		invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
		for _, char := range invalid {
			name = strings.ReplaceAll(name, char, "_")
		}
		return strings.TrimSpace(name)
	}

	// Helper to load blacklist
	loadBlacklist := func() map[string]bool {
		bl := make(map[string]bool)
		path := filepath.Join(sessionRootDir, "blacklist.txt")
		content, err := os.ReadFile(path)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					bl[strings.ToLower(trimmed)] = true
				}
			}
		}
		return bl
	}

	for {
		select {
		case focusEvent := <-focusChan:
			fmt.Printf("[DEBUG] Focus Event: %s (%s)\n", focusEvent.AppName, focusEvent.Title)
			// Check if paused
			if isPaused.Load() {
				currentDir = ""
				currentHandle = 0
				continue
			}

			// Check Blacklist
			blacklist := loadBlacklist()
			if blacklist[strings.ToLower(focusEvent.AppName)] {
				currentDir = ""
				currentHandle = 0
				continue
			}

			// Update State
			currentAppName = focusEvent.AppName
			currentTitle = focusEvent.Title
			currentHandle = focusEvent.Handle

			// Create Directory: sessions/YYYY-MM-DD/AppName
			dateStr := time.Now().Format("2006-01-02")
			safeAppName := sanitize(currentAppName)
			if safeAppName == "" {
				safeAppName = "UnknownApp"
			}

			currentDir = filepath.Join(sessionRootDir, dateStr, safeAppName)
			if err := os.MkdirAll(currentDir, 0755); err != nil {
				fmt.Printf("Error creating app directory: %v\n", err)
				currentDir = ""
			} else {
				fmt.Printf("\n[FOCUS] %s (%s)\n", currentAppName, currentTitle)
			}

		case <-screenshotTicker.C:
			if isPaused.Load() {
				continue
			}
			if currentDir != "" && currentHandle != 0 {
				// Capture Screenshot
				timestamp := time.Now().Format("15-04-05")
				filename := fmt.Sprintf("%s.png", timestamp)
				fullPath := filepath.Join(currentDir, filename)

				// Capture
				err := capture.SaveActiveWindow(currentHandle, fullPath)
				if err != nil {
					fmt.Printf("[DEBUG] Failed to capture: %v\n", err)
				} else {
					fmt.Printf(".") // Progress indicator

					// Trigger OCR in background
					go func(img, dir string) {
						text, err := ocr.ExtractText(img)
						if err != nil {
							return
						}
						if text == "" {
							return
						}

						// Append to ocr.txt
						ocrPath := filepath.Join(dir, "ocr.txt")
						f, err := os.OpenFile(ocrPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
						if err != nil {
							return
						}
						defer f.Close()

						entry := fmt.Sprintf("[%s] %s\n\n", filepath.Base(img), text)
						f.WriteString(entry)
					}(fullPath, currentDir)
				}
			}

		case clipboardEvent := <-clipboardChan:
			if currentDir != "" {
				// Save clipboard content
				logPath := filepath.Join(currentDir, "clipboard.txt")

				f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Printf("Error opening clipboard log: %v\n", err)
					continue
				}

				timestamp := time.Now().Format("15:04:05")
				entry := fmt.Sprintf("[%s] %s\n\n", timestamp, clipboardEvent.Content)

				if _, err := f.WriteString(entry); err != nil {
					fmt.Printf("Error writing to clipboard log: %v\n", err)
				}
				f.Close()

				fmt.Printf("\n[CLIPBOARD] Saved to %s\n", filepath.Base(currentDir))
			}

		case <-sigChan:
			fmt.Println("\nShutting down...")
			return
		}
	}
}
