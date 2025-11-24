package main

import (
	"context"
	"fmt"
	"ideathon/pkg/content"
	"ideathon/pkg/storage"
	"ideathon/pkg/tracker"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	fmt.Println("Starting Memory App...")

	// 1. Initialize Storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}
	// Use the specific path requested by the user
	// In a production app, we might want to make this configurable via flags or config file
	logDir := filepath.Join(homeDir, "OneDrive", "Documents", "ideathon txt experiments")
	logger, err := storage.NewLogger(logDir)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		return
	}
	fmt.Printf("Logging to: %s\n", logDir)

	// 2. Initialize Modules
	trackerPoller := tracker.NewPoller()
	clipboardMonitor := content.NewMonitor()

	// 3. Start Monitoring
	focusChan := trackerPoller.Start()
	clipboardChan := clipboardMonitor.Start()

	// 4. Handle OS Signals for Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Monitoring active... Press Ctrl+C to stop.")

	// 5. Main Event Loop

	// Session Management
	type Session struct {
		sync.Mutex
		AppName   string
		Title     string
		StartTime time.Time
		LastText  string
	}

	var (
		currentSession *Session
		cancelFunc     context.CancelFunc
	)

	for {
		select {
		case focusEvent := <-focusChan:
			// 1. Close Previous Session
			if cancelFunc != nil {
				cancelFunc() // Stop the background poller
			}

			if currentSession != nil {
				currentSession.Lock()
				text := currentSession.LastText
				startTime := currentSession.StartTime
				appName := currentSession.AppName
				currentSession.Unlock()

				if text != "" {
					// Save to file
					safeApp := filepath.Base(appName)
					safeApp = strings.TrimSuffix(safeApp, filepath.Ext(safeApp))
					filename := fmt.Sprintf("%s_%s.txt", startTime.Format("2006-01-02_15-04-05"), safeApp)
					fullPath := filepath.Join(logDir, filename)

					if err := os.WriteFile(fullPath, []byte(text), 0644); err != nil {
						fmt.Printf("Error saving session file: %v\n", err)
					} else {
						fmt.Printf("   > Session saved: %s (%d chars)\n", filename, len(text))
					}
				}
			}

			// 2. Log New Focus
			header := fmt.Sprintf("\n[%s] === FOCUS: %s (%s) ===\n",
				focusEvent.Timestamp.Format("15:04:05"),
				focusEvent.AppName,
				focusEvent.Title,
			)
			fmt.Print(header)
			if err := logger.Log(header); err != nil {
				fmt.Printf("Error writing to log: %v\n", err)
			}

			// 3. Start New Session
			currentSession = &Session{
				AppName:   focusEvent.AppName,
				Title:     focusEvent.Title,
				StartTime: focusEvent.Timestamp,
			}

			var ctx context.Context
			ctx, cancelFunc = context.WithCancel(context.Background())

			// Start Background Poller for this session
			go func(ctx context.Context, sess *Session) {
				// Initial delay
				time.Sleep(2 * time.Second)

				ticker := time.NewTicker(10 * time.Second)
				defer ticker.Stop()

				// Immediate first run after delay
				capture := func() {
					text, err := content.ExtractContext()
					if err != nil {
						// fmt.Printf("Error extracting context: %v\n", err) // Optional: reduce noise
						return
					}
					if text != "" {
						sess.Lock()
						sess.LastText = text
						sess.Unlock()
						// fmt.Printf("   > Captured %d chars\n", len(text)) // Debug feedback
					}
				}

				// Check if context is already done before first capture
				select {
				case <-ctx.Done():
					return
				default:
					capture()
				}

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						// Check for 2-hour session limit
						sess.Lock()
						if time.Since(sess.StartTime) > 2*time.Hour {
							// Rotate session
							oldText := sess.LastText
							oldStart := sess.StartTime
							oldApp := sess.AppName

							// Reset for new chunk
							sess.StartTime = time.Now()
							sess.LastText = "" // Optional: clear text to avoid duplicate logging if no new text comes in?
							// Actually, keep LastText so we don't lose context if user is idle.
							// But if we write the file, we effectively "archived" that text.
							// Let's keep it, but the new file will start with it.
							// User said "wont write the same session forever".
							// If we save now, we save 0-2h.
							// Next save (at 4h or close) will save 2h-4h.
							// If text hasn't changed, 2h-4h file will be identical to 0-2h file.
							// That might be what "same session forever" means.
							// But if the user is active, text changes.
							sess.Unlock()

							if oldText != "" {
								safeApp := filepath.Base(oldApp)
								safeApp = strings.TrimSuffix(safeApp, filepath.Ext(safeApp))
								filename := fmt.Sprintf("%s_%s.txt", oldStart.Format("2006-01-02_15-04-05"), safeApp)
								fullPath := filepath.Join(logDir, filename)
								if err := os.WriteFile(fullPath, []byte(oldText), 0644); err != nil {
									fmt.Printf("Error saving session chunk: %v\n", err)
								} else {
									fmt.Printf("   > Session chunk saved: %s\n", filename)
								}
							}
						} else {
							sess.Unlock()
						}

						capture()
					}
				}
			}(ctx, currentSession)

		case clipboardEvent := <-clipboardChan:
			// Log Captured Content
			logEntry := fmt.Sprintf("   > \"Captured text: %s\"\n", clipboardEvent.Content)

			// Console feedback (truncated)
			displayContent := clipboardEvent.Content
			if len(displayContent) > 50 {
				displayContent = displayContent[:50] + "..."
			}
			fmt.Printf("   > Captured: %q\n", displayContent)

			if err := logger.Log(logEntry); err != nil {
				fmt.Printf("Error writing to log: %v\n", err)
			}

		case <-sigChan:
			fmt.Println("\nShutting down...")
			// Save pending session on exit
			if cancelFunc != nil {
				cancelFunc()
			}
			if currentSession != nil {
				currentSession.Lock()
				text := currentSession.LastText
				startTime := currentSession.StartTime
				appName := currentSession.AppName
				currentSession.Unlock()

				if text != "" {
					safeApp := filepath.Base(appName)
					safeApp = strings.TrimSuffix(safeApp, filepath.Ext(safeApp))
					filename := fmt.Sprintf("%s_%s.txt", startTime.Format("2006-01-02_15-04-05"), safeApp)
					fullPath := filepath.Join(logDir, filename)
					os.WriteFile(fullPath, []byte(text), 0644)
					fmt.Printf("   > Final session saved: %s\n", filename)
				}
			}
			return
		}
	}
}
