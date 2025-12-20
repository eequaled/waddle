package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// HealthMonitor provides health checking and observability for the storage system.
type HealthMonitor struct {
	config        *StorageConfig
	storageEngine *StorageEngine
	logger        *StructuredLogger
}

// StructuredLogger provides structured JSON logging.
type StructuredLogger struct {
	logFiles map[string]*os.File
}

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Operation string                 `json:"operation"`
	Duration  int64                  `json:"duration_ms,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(config *StorageConfig, storageEngine *StorageEngine) (*HealthMonitor, error) {
	logger, err := NewStructuredLogger(config.DataDir)
	if err != nil {
		return nil, err
	}

	return &HealthMonitor{
		config:        config,
		storageEngine: storageEngine,
		logger:        logger,
	}, nil
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger(dataDir string) (*StructuredLogger, error) {
	logFiles := make(map[string]*os.File)

	// Create log files
	logTypes := []string{"waddle", "performance", "error"}
	for _, logType := range logTypes {
		logPath := filepath.Join(dataDir, fmt.Sprintf("%s.log", logType))
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// Close any already opened files
			for _, f := range logFiles {
				f.Close()
			}
			return nil, NewStorageError(ErrFileSystem, fmt.Sprintf("failed to create %s log", logType), err)
		}
		logFiles[logType] = file
	}

	return &StructuredLogger{
		logFiles: logFiles,
	}, nil
}

// Log writes a structured log entry.
func (sl *StructuredLogger) Log(logType, level, component, operation string, duration time.Duration, err error, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Component: component,
		Operation: operation,
		Fields:    fields,
	}

	if duration > 0 {
		entry.Duration = duration.Milliseconds()
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Write to appropriate log file
	file, exists := sl.logFiles[logType]
	if !exists {
		file = sl.logFiles["waddle"] // Default to main log
	}

	if file != nil {
		jsonData, _ := json.Marshal(entry)
		file.WriteString(string(jsonData) + "\n")
		file.Sync()
	}
}

// Info logs an info-level message.
func (sl *StructuredLogger) Info(component, operation string, fields map[string]interface{}) {
	sl.Log("waddle", "INFO", component, operation, 0, nil, fields)
}

// Error logs an error-level message.
func (sl *StructuredLogger) Error(component, operation string, err error, fields map[string]interface{}) {
	sl.Log("error", "ERROR", component, operation, 0, err, fields)
}

// Performance logs a performance measurement.
func (sl *StructuredLogger) Performance(component, operation string, duration time.Duration, fields map[string]interface{}) {
	sl.Log("performance", "PERF", component, operation, duration, nil, fields)
}

// Close closes all log files.
func (sl *StructuredLogger) Close() error {
	for _, file := range sl.logFiles {
		file.Close()
	}
	return nil
}

// HealthCheck performs a comprehensive health check of the storage system.
func (hm *HealthMonitor) HealthCheck() (*HealthStatus, error) {
	status := &HealthStatus{
		Status:    HealthStatusHealthy,
		Checks:    make(map[string]Check),
		Timestamp: time.Now(),
	}

	// Check SQLite database
	hm.checkSQLiteHealth(status)

	// Check vector database
	hm.checkVectorHealth(status)

	// Check filesystem
	hm.checkFilesystemHealth(status)

	// Check Ollama connectivity (for embeddings)
	hm.checkOllamaHealth(status)

	// Determine overall status
	hm.determineOverallStatus(status)

	// Log health check results
	hm.logger.Info("health", "check_complete", map[string]interface{}{
		"status":       status.Status,
		"checks_count": len(status.Checks),
	})

	return status, nil
}

// checkSQLiteHealth checks the health of the SQLite database.
func (hm *HealthMonitor) checkSQLiteHealth(status *HealthStatus) {
	start := time.Now()
	check := Check{Status: HealthStatusHealthy}

	if hm.storageEngine.sessionMgr == nil {
		check.Status = HealthStatusUnhealthy
		check.Message = "Session manager not initialized"
	} else {
		// Test database connectivity
		db := hm.storageEngine.sessionMgr.DB()
		if db == nil {
			check.Status = HealthStatusUnhealthy
			check.Message = "Database connection not available"
		} else {
			// Test simple query
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var result int
			err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				check.Status = HealthStatusUnhealthy
				check.Message = fmt.Sprintf("Database query failed: %v", err)
			} else {
				// Run integrity check
				if err := hm.storageEngine.sessionMgr.RunIntegrityCheck(); err != nil {
					check.Status = HealthStatusDegraded
					check.Message = fmt.Sprintf("Database integrity check failed: %v", err)
				}
			}
		}
	}

	check.Latency = time.Since(start).Milliseconds()
	status.Checks["sqlite"] = check
}

// checkVectorHealth checks the health of the vector database.
func (hm *HealthMonitor) checkVectorHealth(status *HealthStatus) {
	start := time.Now()
	check := Check{Status: HealthStatusHealthy}

	if hm.storageEngine.vectorMgr == nil {
		check.Status = HealthStatusDegraded
		check.Message = "Vector manager not initialized"
	} else {
		// Test vector database by attempting a simple search
		testEmbedding := make([]float32, 768)
		for i := range testEmbedding {
			testEmbedding[i] = 0.1
		}

		_, err := hm.storageEngine.vectorMgr.Search(testEmbedding, 1)
		if err != nil {
			check.Status = HealthStatusDegraded
			check.Message = fmt.Sprintf("Vector search failed: %v", err)
		}
	}

	check.Latency = time.Since(start).Milliseconds()
	status.Checks["vector"] = check
}

// checkFilesystemHealth checks the health of the filesystem.
func (hm *HealthMonitor) checkFilesystemHealth(status *HealthStatus) {
	start := time.Now()
	check := Check{Status: HealthStatusHealthy}

	// Check data directory accessibility
	if _, err := os.Stat(hm.config.DataDir); os.IsNotExist(err) {
		check.Status = HealthStatusUnhealthy
		check.Message = "Data directory not accessible"
	} else {
		// Test write permissions
		testFile := filepath.Join(hm.config.DataDir, ".health_check")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			check.Status = HealthStatusDegraded
			check.Message = fmt.Sprintf("Write permission test failed: %v", err)
		} else {
			os.Remove(testFile) // Clean up
		}

		// Check disk space (warn if less than 1GB free)
		if hm.storageEngine.fileMgr != nil {
			stats, err := hm.storageEngine.fileMgr.GetStorageStats()
			if err == nil && stats.TotalSizeBytes > 0 {
				// This is a simplified check - in production you'd check actual disk space
				if stats.TotalSizeBytes > 10*1024*1024*1024 { // 10GB
					check.Status = HealthStatusDegraded
					check.Message = "High disk usage detected"
				}
			}
		}
	}

	check.Latency = time.Since(start).Milliseconds()
	status.Checks["filesystem"] = check
}

// checkOllamaHealth checks the health of Ollama connectivity.
func (hm *HealthMonitor) checkOllamaHealth(status *HealthStatus) {
	start := time.Now()
	check := Check{Status: HealthStatusHealthy}

	if hm.storageEngine.vectorMgr == nil {
		check.Status = HealthStatusDegraded
		check.Message = "Vector manager not available"
	} else {
		// Test embedding generation
		_, err := hm.storageEngine.vectorMgr.GenerateEmbedding("health check test")
		if err != nil {
			check.Status = HealthStatusDegraded
			check.Message = fmt.Sprintf("Embedding generation failed: %v", err)
		}
	}

	check.Latency = time.Since(start).Milliseconds()
	status.Checks["ollama"] = check
}

// determineOverallStatus determines the overall health status based on individual checks.
func (hm *HealthMonitor) determineOverallStatus(status *HealthStatus) {
	hasUnhealthy := false
	hasDegraded := false

	for _, check := range status.Checks {
		switch check.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		status.Status = HealthStatusUnhealthy
	} else if hasDegraded {
		status.Status = HealthStatusDegraded
	} else {
		status.Status = HealthStatusHealthy
	}
}

// GetHealthEndpoint returns an HTTP handler for the health check endpoint.
func (hm *HealthMonitor) GetHealthEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := hm.HealthCheck()
		if err != nil {
			http.Error(w, fmt.Sprintf("Health check failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		
		// Set HTTP status based on health
		switch status.Status {
		case HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still OK, but with warnings
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(status)
	}
}

// StartHealthCheckScheduler starts a background scheduler for periodic health checks.
func (hm *HealthMonitor) StartHealthCheckScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			status, err := hm.HealthCheck()
			if err != nil {
				hm.logger.Error("health", "scheduled_check_failed", err, nil)
				continue
			}

			// Log if status is not healthy
			if status.Status != HealthStatusHealthy {
				hm.logger.Error("health", "unhealthy_status", nil, map[string]interface{}{
					"status": status.Status,
					"checks": status.Checks,
				})
			}
		}
	}()
}

// Close closes the health monitor and its logger.
func (hm *HealthMonitor) Close() error {
	if hm.logger != nil {
		return hm.logger.Close()
	}
	return nil
}