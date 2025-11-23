package main

import (
	"fmt"
	"ideathon/pkg/content"
	"ideathon/pkg/storage"
	"ideathon/pkg/tracker"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
	for {
		select {
		case focusEvent := <-focusChan:
			// Log Focus Change
			header := fmt.Sprintf("\n[%s] === FOCUS: %s (%s) ===\n",
				focusEvent.Timestamp.Format("15:04:05"),
				focusEvent.AppName,
				focusEvent.Title,
			)

			fmt.Print(header) // Echo to console
			if err := logger.Log(header); err != nil {
				fmt.Printf("Error writing to log: %v\n", err)
			}

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
			return
		}
	}
}
