package main

import (
	"context"
	"log"
	"sort"
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
	_, err := a.storage.GetSession(date)
	if err != nil {
		if storage.IsNotFound(err) {
			return []AppDetail{}, nil
		}
		return nil, err
	}

	activities, err := a.storage.GetSessionAppActivities(date)
	if err != nil {
		return nil, err
	}

	appDetails := make([]AppDetail, 0, len(activities))
	for _, activity := range activities {
		blocks, err := a.storage.GetActivityBlocks(date, activity.AppName)
		if err != nil {
			return nil, err
		}

		appDetails = append(appDetails, AppDetail{
			AppName:    activity.AppName,
			BlockCount: len(blocks),
		})
	}

	sort.Slice(appDetails, func(i, j int) bool {
		if appDetails[i].BlockCount == appDetails[j].BlockCount {
			return appDetails[i].AppName < appDetails[j].AppName
		}
		return appDetails[i].BlockCount > appDetails[j].BlockCount
	})

	return appDetails, nil
}

// GetCaptureStatus returns the current capture pipeline status.
func (a *App) GetCaptureStatus() pipeline.PipelineStats {
	if a.pipeline == nil {
		return pipeline.PipelineStats{Running: false, Source: "none"}
	}
	return a.pipeline.GetPipelineStats()
}

// HealthStatusResponse represents backend health status for UI consumers.
type HealthStatusResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`
}

// GetHealthStatus returns storage health information.
func (a *App) GetHealthStatus() HealthStatusResponse {
	if a.storage == nil {
		return HealthStatusResponse{Status: "unavailable"}
	}
	health, err := a.storage.HealthCheck()
	if err != nil {
		return HealthStatusResponse{Status: "error", Error: err.Error()}
	}
	return HealthStatusResponse{
		Status:    health.Status,
		Timestamp: health.Timestamp.Format(time.RFC3339),
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
