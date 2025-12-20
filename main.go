package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"waddle/pkg/capture/uia"
	"waddle/pkg/pipeline"
	"waddle/pkg/server"
	"waddle/pkg/storage"
	"waddle/pkg/synthesis"
	"waddle/pkg/tracker/etw"
)

func main() {
	// Parse flags
	dataDirFlag := flag.String("data-dir", "", "Path to data directory (default: ~/.waddle)")
	portFlag := flag.String("port", "8080", "API Server port")
	flag.Parse()

	fmt.Println("Starting Waddle...")

	// 1. Initialize Storage Engine
	var storageDataDir string
	if *dataDirFlag != "" {
		storageDataDir = *dataDirFlag
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Error getting home directory: %v", err)
		}
		storageDataDir = filepath.Join(homeDir, ".waddle")
	}

	storageConfig := storage.DefaultStorageConfig(storageDataDir)
	storageEngine := storage.NewStorageEngine(storageConfig)
	if err := storageEngine.Initialize(); err != nil {
		log.Fatalf("Error initializing storage engine: %v", err)
	}
	defer storageEngine.Close()
	fmt.Printf("Storage engine initialized at: %s\n", storageDataDir)

	// 2. Check Ollama availability
	if err := checkOllama(); err != nil {
		fmt.Printf("⚠️  Ollama not available: %v\n", err)
		fmt.Printf("   Semantic search will be disabled. To enable:\n")
		fmt.Printf("   1. Install Ollama: https://ollama.ai\n")
		fmt.Printf("   2. Run: ollama serve\n")
		fmt.Printf("   3. Pull model: ollama pull nomic-embed-text\n")
	} else {
		fmt.Printf("✅ Ollama available - semantic search enabled\n")
	}

	// 3. Initialize ETW Consumer
	etwConsumer, err := etw.NewConsumer()
	if err != nil {
		fmt.Printf("⚠️  ETW initialization failed: %v\n", err)
		fmt.Printf("   Falling back to polling mode\n")
		// TODO: Initialize polling fallback
	}

	// 4. Initialize UI Automation Marshaler
	uiaMarshaler, err := uia.NewMarshaler()
	if err != nil {
		log.Fatalf("Error initializing UI Automation: %v", err)
	}
	defer uiaMarshaler.Close()

	// 5. Initialize Hybrid Capture Pipeline
	capturePipeline, err := pipeline.NewPipeline(storageEngine)
	if err != nil {
		log.Fatalf("Error initializing capture pipeline: %v", err)
	}
	defer capturePipeline.Close()

	// 6. Initialize Synthesis Worker
	synthesisWorker := synthesis.NewWorker(storageEngine)
	if err := synthesisWorker.Start(); err != nil {
		log.Fatalf("Error starting synthesis worker: %v", err)
	}
	defer synthesisWorker.Close()

	// 7. Start API Server
	isPaused := &atomic.Bool{}
	apiServer := server.NewServer("", *portFlag, isPaused, storageEngine)
	go apiServer.Start()

	// 8. Start ETW Consumer
	if etwConsumer != nil {
		if err := etwConsumer.Start(); err != nil {
			fmt.Printf("Error starting ETW consumer: %v\n", err)
		}
	}

	// 9. Start Capture Pipeline
	if err := capturePipeline.Start(); err != nil {
		log.Fatalf("Error starting capture pipeline: %v", err)
	}

	// 10. Handle OS Signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Monitoring active... Press Ctrl+C to stop.")

	// 11. Main event loop - wire ETW events to pipeline
	if etwConsumer != nil {
		go func() {
			for {
				select {
				case focusEvent := <-etwConsumer.FocusEvents():
					// Convert ETW event to pipeline format and process
					if err := capturePipeline.ProcessFocusEvent(focusEvent); err != nil {
						log.Printf("Error processing focus event: %v", err)
					}
				case <-sigChan:
					return
				}
			}
		}()
	}

	// 12. Wait for shutdown signal
	<-sigChan
	fmt.Println("\nShutting down...")
}

// checkOllama verifies if Ollama is available for semantic search.
func checkOllama() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return fmt.Errorf("connection failed (run 'ollama serve')")
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("server error (status %d)", resp.StatusCode)
	}
	
	return nil
}
