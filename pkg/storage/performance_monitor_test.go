package storage

import (
	"testing"
	"time"
)

// TestPerformanceMonitoring tests performance monitoring functionality
func TestPerformanceMonitoring(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create performance monitor
	perfMon, err := NewPerformanceMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create performance monitor: %v", err)
	}
	defer perfMon.Close()

	// Log some test operations
	perfMon.LogOperation("session", "create", 50*time.Millisecond, "date=2024-01-20", nil)
	perfMon.LogOperation("session", "get", 10*time.Millisecond, "date=2024-01-20", nil)
	perfMon.LogOperation("search", "fulltext", 150*time.Millisecond, "query=test", nil)
	perfMon.LogOperation("vector", "search", 200*time.Millisecond, "query=semantic", nil)

	// Get performance stats
	stats := perfMon.GetStats(time.Now().Add(-1 * time.Hour))

	// Verify stats
	if stats.TotalOperations != 4 {
		t.Errorf("Expected 4 total operations, got %d", stats.TotalOperations)
	}

	if stats.SlowQueries != 2 {
		t.Errorf("Expected 2 slow queries (>100ms), got %d", stats.SlowQueries)
	}

	if stats.ErrorRate != 0.0 {
		t.Errorf("Expected 0%% error rate, got %.2f%%", stats.ErrorRate*100)
	}

	// Verify component stats
	if sessionStats, exists := stats.ByComponent["session"]; exists {
		if sessionStats.Operations != 2 {
			t.Errorf("Expected 2 session operations, got %d", sessionStats.Operations)
		}
	} else {
		t.Error("Expected session component stats")
	}
}

// TestBenchmarkExecution tests benchmark execution
func TestBenchmarkExecution(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create performance monitor
	perfMon, err := NewPerformanceMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create performance monitor: %v", err)
	}
	defer perfMon.Close()

	// Run benchmark with small dataset
	result, err := perfMon.RunBenchmark(10)
	if err != nil {
		t.Fatalf("Failed to run benchmark: %v", err)
	}

	// Verify benchmark results
	if result.TestDataCount != 10 {
		t.Errorf("Expected test data count 10, got %d", result.TestDataCount)
	}

	if result.TotalDuration <= 0 {
		t.Error("Expected positive total duration")
	}

	// Verify operation benchmarks exist
	expectedOps := []string{"session_lookup", "fulltext_search", "semantic_search", "file_save"}
	for _, op := range expectedOps {
		if benchmark, exists := result.Operations[op]; exists {
			if benchmark.Iterations <= 0 {
				t.Errorf("Expected positive iterations for %s", op)
			}
			if benchmark.AverageDuration <= 0 {
				t.Errorf("Expected positive average duration for %s", op)
			}
		} else {
			t.Errorf("Expected benchmark results for operation: %s", op)
		}
	}
}

// TestStorageMetrics tests storage metrics collection
func TestStorageMetrics(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create some test data
	session, err := storageEngine.CreateSession("2024-01-20")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	session.CustomTitle = "Test Session"
	if err := storageEngine.UpdateSession(session); err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Create performance monitor
	perfMon, err := NewPerformanceMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create performance monitor: %v", err)
	}
	defer perfMon.Close()

	// Get storage metrics
	metrics, err := perfMon.GetStorageMetrics()
	if err != nil {
		t.Fatalf("Failed to get storage metrics: %v", err)
	}

	// Verify metrics
	if metrics.SessionCount != 1 {
		t.Errorf("Expected 1 session, got %d", metrics.SessionCount)
	}

	if metrics.DatabaseSize <= 0 {
		t.Error("Expected positive database size")
	}

	if metrics.TotalSize <= 0 {
		t.Error("Expected positive total size")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Expected valid timestamp")
	}
}

// TestPerformanceLogging tests slow query logging
func TestPerformanceLogging(t *testing.T) {
	tempDir := t.TempDir()
	
	// Initialize storage engine
	config := DefaultStorageConfig(tempDir)
	storageEngine := NewStorageEngine(config)
	if err := storageEngine.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage engine: %v", err)
	}
	defer storageEngine.Close()

	// Create performance monitor
	perfMon, err := NewPerformanceMonitor(config, storageEngine)
	if err != nil {
		t.Fatalf("Failed to create performance monitor: %v", err)
	}
	defer perfMon.Close()

	// Log a slow operation (>100ms)
	perfMon.LogOperation("test", "slow_operation", 250*time.Millisecond, "test=true", nil)

	// Log a fast operation (<100ms)
	perfMon.LogOperation("test", "fast_operation", 50*time.Millisecond, "test=true", nil)

	// Get stats
	stats := perfMon.GetStats(time.Now().Add(-1 * time.Hour))

	// Verify slow query was logged
	if stats.SlowQueries != 1 {
		t.Errorf("Expected 1 slow query, got %d", stats.SlowQueries)
	}

	if stats.TotalOperations != 2 {
		t.Errorf("Expected 2 total operations, got %d", stats.TotalOperations)
	}
}