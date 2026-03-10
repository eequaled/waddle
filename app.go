package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"waddle/pkg/infra/config"
	"waddle/pkg/pipeline"
	"waddle/pkg/platform"
	"waddle/pkg/storage"
	"waddle/pkg/synthesis"
)

// App struct
type App struct {
	ctx           context.Context
	cfg           config.Config
	storage       *storage.StorageEngine
	tracker       platform.WindowTracker
	pipeline      *pipeline.Pipeline
	synthWorker   *synthesis.Worker
	isPaused      *atomic.Bool
}

// NewApp creates a new App application struct
func NewApp(cfg config.Config) *App {
	return &App{
		cfg:      cfg,
		isPaused: &atomic.Bool{},
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 1. Initialize Storage Engine
	storageConfig := storage.DefaultStorageConfig(a.cfg.DataDir)
	a.storage = storage.NewStorageEngine(storageConfig)
	if err := a.storage.Initialize(); err != nil {
		log.Printf("Error initializing storage engine: %v\n", err)
		// Storage is critical — zero it out so downstream nil-checks work
		a.storage = nil
	}

	// 2. Initialize Platform Tracker
	tracker, err := platform.NewWindowTracker()
	if err != nil {
		log.Printf("Error initializing platform tracker: %v\n", err)
	}
	a.tracker = tracker

	// 3. Initialize Capture Pipeline (requires storage, tracker is optional)
	if a.storage != nil {
		p, err := pipeline.NewPipeline(a.storage, a.tracker)
		if err != nil {
			log.Printf("Error initializing capture pipeline: %v\n", err)
		} else {
			a.pipeline = p
		}
	}

	// 4. Initialize Synthesis Worker (requires storage)
	if a.storage != nil {
		a.synthWorker = synthesis.NewWorker(a.storage)
		if err := a.synthWorker.Start(); err != nil {
			log.Printf("Error starting synthesis worker: %v\n", err)
		}
	}

	// 5. Start Pipeline
	if a.pipeline != nil {
		if err := a.pipeline.Start(); err != nil {
			log.Printf("Error starting capture pipeline: %v\n", err)
		}
	}

	log.Println("Waddle subsystems started successfully")
}

// shutdown is called when the app finishes.
func (a *App) shutdown(ctx context.Context) {
	if a.pipeline != nil {
		a.pipeline.Stop()
	}
	if a.tracker != nil {
		a.tracker.Stop()
	}
	if a.synthWorker != nil {
		a.synthWorker.Close()
	}
	if a.storage != nil {
		a.storage.Close()
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Waddle v2 is active!", name)
}
