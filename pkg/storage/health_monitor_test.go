package storage

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHealthCheck tests the health check functionality
func TestHealthCheck(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create health monitor
	healthMon, err := NewHealthMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}
	defer healthMon.Close()

	// Perform health check
	status, err := healthMon.HealthCheck()
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Verify health status
	if status.Status == "" {
		t.Error("Expected health status to be set")
	}

	if status.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	// Verify individual checks
	expectedChecks := []string{"sqlite", "vector", "filesystem", "ollama"}
	for _, checkName := range expectedChecks {
		if check, exists := status.Checks[checkName]; exists {
			if check.Status == "" {
				t.Errorf("Expected status for check %s", checkName)
			}
			if check.Latency < 0 {
				t.Errorf("Expected non-negative latency for check %s", checkName)
			}
		} else {
			t.Errorf("Expected check %s to be present", checkName)
		}
	}
}

// TestHealthEndpoint tests the HTTP health endpoint
func TestHealthEndpoint(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create health monitor
	healthMon, err := NewHealthMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}
	defer healthMon.Close()

	// Create test server
	handler := healthMon.GetHealthEndpoint()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make request to health endpoint
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make health check request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Error("Expected JSON content type")
	}
}

// TestStructuredLogger tests the structured logging functionality
func TestStructuredLogger(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create structured logger
	logger, err := NewStructuredLogger(tempDir)
	if err != nil {
		t.Fatalf("Failed to create structured logger: %v", err)
	}
	defer logger.Close()

	// Test different log levels
	logger.Info("test", "info_operation", map[string]interface{}{
		"key": "value",
	})

	logger.Error("test", "error_operation", NewStorageError(ErrValidation, "test error", nil), map[string]interface{}{
		"error_code": "TEST_ERROR",
	})

	logger.Performance("test", "perf_operation", 150*time.Millisecond, map[string]interface{}{
		"iterations": 100,
	})

	// Verify log files were created
	logTypes := []string{"waddle", "performance", "error"}
	for _, logType := range logTypes {
		if _, exists := logger.logFiles[logType]; !exists {
			t.Errorf("Expected log file for type %s", logType)
		}
	}
}

// TestHealthCheckWithUnhealthyComponents tests health check with failing components
func TestHealthCheckWithUnhealthyComponents(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine but don't initialize it properly
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	// Don't call Initialize() to simulate unhealthy state

	// Create health monitor
	healthMon, err := NewHealthMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}
	defer healthMon.Close()

	// Perform health check
	status, err := healthMon.HealthCheck()
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Should detect unhealthy components
	if status.Status == HealthStatusHealthy {
		t.Error("Expected unhealthy status due to uninitialized components")
	}

	// Check that SQLite check failed
	if sqliteCheck, exists := status.Checks["sqlite"]; exists {
		if sqliteCheck.Status == HealthStatusHealthy {
			t.Error("Expected SQLite check to be unhealthy")
		}
	}
}

// TestHealthCheckLatency tests that health checks complete within reasonable time
func TestHealthCheckLatency(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create health monitor
	healthMon, err := NewHealthMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create health monitor: %v", err)
	}
	defer healthMon.Close()

	// Measure health check time
	start := time.Now()
	status, err := healthMon.HealthCheck()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Health check should complete within 10 seconds
	if duration > 10*time.Second {
		t.Errorf("Health check took too long: %v", duration)
	}

	// Individual checks should have reasonable latencies
	for checkName, check := range status.Checks {
		if check.Latency > 5000 { // 5 seconds
			t.Errorf("Check %s took too long: %dms", checkName, check.Latency)
		}
	}
}