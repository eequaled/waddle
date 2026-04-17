package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"waddle/pkg/infra/config"
	"waddle/pkg/pipeline"
	"waddle/pkg/platform"
	"waddle/pkg/server"
	"waddle/pkg/storage"
	"waddle/pkg/synthesis"
)

// App struct
type App struct {
	ctx         context.Context
	cfg         config.Config
	storage     *storage.StorageEngine
	plat        platform.Platform
	pipeline    *pipeline.Pipeline
	synthWorker *synthesis.Worker
	isPaused    *atomic.Bool
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
		a.storage = nil
	}

	// 2. Initialize full Platform (tracker + UIA + screenshot + secrets)
	plat, err := platform.NewPlatform(&a.cfg)
	if err != nil {
		log.Printf("Error initializing platform: %v\n", err)
	}
	a.plat = plat

	// 3. Initialize Capture Pipeline (requires storage + platform)
	if a.storage != nil && a.plat != nil {
		p, err := pipeline.NewPipeline(a.storage, a.plat)
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

	// 6. Start API Server to serve frontend requests
	if a.storage != nil {
		apiServer := server.NewServer(a.cfg.DataDir, a.cfg.Port, a.isPaused, a.storage)
		apiServer.Start()
		log.Printf("API Server started on port %s\n", a.cfg.Port)
	}

	log.Println("Waddle subsystems started successfully")
}

// shutdown is called when the app finishes.
func (a *App) shutdown(ctx context.Context) {
	if a.pipeline != nil {
		a.pipeline.Stop()
	}
	if a.plat != nil {
		a.plat.Close()
	}
	if a.synthWorker != nil {
		a.synthWorker.Close()
	}
	if a.storage != nil {
		a.storage.Close()
	}
}

// ── Wails-bound methods (auto-generates JS bindings) ────────────────

// GetSessions returns all sessions for the Memory view.
func (a *App) GetSessions() ([]storage.Session, error) {
	if a.storage == nil {
		return []storage.Session{}, nil
	}
	sessions, _, err := a.storage.ListSessions(1, 100)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// AppDetail is a simplified app activity for the frontend.
type AppDetail struct {
	AppName    string `json:"appName"`
	BlockCount int    `json:"blockCount"`
}

// GetAppDetails returns app activity summaries for a given date.
func (a *App) GetAppDetails(date string) ([]AppDetail, error) {
	if a.storage == nil {
		return []AppDetail{}, nil
	}
	session, err := a.storage.GetSession(date)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return []AppDetail{}, nil
	}
	// Return basic session info — detailed app breakdown requires
	// additional storage methods that will be added in future phases.
	return []AppDetail{
		{AppName: "Session: " + date, BlockCount: 0},
	}, nil
}

// GetCaptureStatus returns the current capture pipeline status.
func (a *App) GetCaptureStatus() map[string]interface{} {
	if a.pipeline == nil {
		return map[string]interface{}{
			"running": false,
			"source":  "none",
		}
	}
	return a.pipeline.GetPipelineStats()
}

// GetHealthStatus returns storage health information.
func (a *App) GetHealthStatus() map[string]interface{} {
	if a.storage == nil {
		return map[string]interface{}{"status": "unavailable"}
	}
	health, err := a.storage.HealthCheck()
	if err != nil {
		return map[string]interface{}{"status": "error", "error": err.Error()}
	}
	return map[string]interface{}{
		"status":    health.Status,
		"timestamp": health.Timestamp.Format(time.RFC3339),
	}
}

// ToggleCapture pauses or resumes the capture pipeline.
func (a *App) ToggleCapture(paused bool) error {
	a.isPaused.Store(paused)
	// Full pause/resume implementation will come with hot tier integration.
	return nil
}

// Greet returns a greeting for the given name (kept for backward compat).
func (a *App) Greet(name string) string {
	return "Hello " + name + ", Waddle v2 is active!"
}
