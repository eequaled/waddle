package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// PerformanceMonitor tracks and logs performance metrics for storage operations.
type PerformanceMonitor struct {
	config        *StorageConfig
	logFile       *os.File
	metrics       []PerformanceMetric
	metricsMutex  sync.RWMutex
	storageEngine *StorageEngine
}

// PerformanceMetric represents a single performance measurement.
type PerformanceMetric struct {
	Timestamp   time.Time `json:"timestamp"`
	Operation   string    `json:"operation"`
	Duration    int64     `json:"duration_ms"`
	Parameters  string    `json:"parameters,omitempty"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Component   string    `json:"component"` // "session", "vector", "file", "encryption"
}

// PerformanceStats contains aggregated performance statistics.
type PerformanceStats struct {
	TotalOperations int64                    `json:"totalOperations"`
	AverageDuration float64                  `json:"averageDuration"`
	P50Duration     int64                    `json:"p50Duration"`
	P95Duration     int64                    `json:"p95Duration"`
	P99Duration     int64                    `json:"p99Duration"`
	SlowQueries     int64                    `json:"slowQueries"`
	ErrorRate       float64                  `json:"errorRate"`
	ByComponent     map[string]ComponentStats `json:"byComponent"`
}

// ComponentStats contains performance statistics for a specific component.
type ComponentStats struct {
	Operations      int64   `json:"operations"`
	AverageDuration float64 `json:"averageDuration"`
	ErrorRate       float64 `json:"errorRate"`
}

// NewPerformanceMonitor creates a new performance monitor.
func NewPerformanceMonitor(config *StorageConfig, storageEngine *StorageEngine) (*PerformanceMonitor, error) {
	// Create performance log file
	logPath := filepath.Join(config.DataDir, "performance.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, NewStorageError(ErrFileSystem, "failed to create performance log", err)
	}

	return &PerformanceMonitor{
		config:        config,
		logFile:       logFile,
		metrics:       make([]PerformanceMetric, 0),
		storageEngine: storageEngine,
	}, nil
}

// LogOperation logs a performance metric for an operation.
func (pm *PerformanceMonitor) LogOperation(component, operation string, duration time.Duration, parameters string, err error) {
	metric := PerformanceMetric{
		Timestamp:  time.Now(),
		Operation:  operation,
		Duration:   duration.Milliseconds(),
		Parameters: parameters,
		Success:    err == nil,
		Component:  component,
	}

	if err != nil {
		metric.Error = err.Error()
	}

	// Add to in-memory metrics
	pm.metricsMutex.Lock()
	pm.metrics = append(pm.metrics, metric)
	// Keep only last 10000 metrics in memory
	if len(pm.metrics) > 10000 {
		pm.metrics = pm.metrics[len(pm.metrics)-10000:]
	}
	pm.metricsMutex.Unlock()

	// Log to file if duration exceeds threshold (100ms)
	if duration.Milliseconds() > 100 {
		pm.logSlowQuery(metric)
	}
}

// logSlowQuery logs slow queries to the performance log file.
func (pm *PerformanceMonitor) logSlowQuery(metric PerformanceMetric) {
	if pm.logFile == nil {
		return
	}

	logEntry := map[string]interface{}{
		"timestamp":  metric.Timestamp.Format(time.RFC3339),
		"component":  metric.Component,
		"operation":  metric.Operation,
		"duration":   metric.Duration,
		"parameters": metric.Parameters,
		"success":    metric.Success,
	}

	if metric.Error != "" {
		logEntry["error"] = metric.Error
	}

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		return
	}

	pm.logFile.WriteString(string(jsonData) + "\n")
	pm.logFile.Sync()
}

// GetStats returns aggregated performance statistics.
func (pm *PerformanceMonitor) GetStats(since time.Time) *PerformanceStats {
	pm.metricsMutex.RLock()
	defer pm.metricsMutex.RUnlock()

	// Filter metrics by time
	var filteredMetrics []PerformanceMetric
	for _, metric := range pm.metrics {
		if metric.Timestamp.After(since) {
			filteredMetrics = append(filteredMetrics, metric)
		}
	}

	if len(filteredMetrics) == 0 {
		return &PerformanceStats{
			ByComponent: make(map[string]ComponentStats),
		}
	}

	// Calculate overall stats
	stats := &PerformanceStats{
		TotalOperations: int64(len(filteredMetrics)),
		ByComponent:     make(map[string]ComponentStats),
	}

	// Calculate durations and error counts
	var durations []int64
	var totalDuration int64
	var errorCount int64
	var slowQueryCount int64
	componentMetrics := make(map[string][]PerformanceMetric)

	for _, metric := range filteredMetrics {
		durations = append(durations, metric.Duration)
		totalDuration += metric.Duration

		if !metric.Success {
			errorCount++
		}

		if metric.Duration > 100 {
			slowQueryCount++
		}

		// Group by component
		componentMetrics[metric.Component] = append(componentMetrics[metric.Component], metric)
	}

	// Calculate percentiles
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	stats.AverageDuration = float64(totalDuration) / float64(len(filteredMetrics))
	stats.P50Duration = durations[len(durations)*50/100]
	stats.P95Duration = durations[len(durations)*95/100]
	stats.P99Duration = durations[len(durations)*99/100]
	stats.SlowQueries = slowQueryCount
	stats.ErrorRate = float64(errorCount) / float64(len(filteredMetrics))

	// Calculate component stats
	for component, metrics := range componentMetrics {
		var componentDuration int64
		var componentErrors int64

		for _, metric := range metrics {
			componentDuration += metric.Duration
			if !metric.Success {
				componentErrors++
			}
		}

		stats.ByComponent[component] = ComponentStats{
			Operations:      int64(len(metrics)),
			AverageDuration: float64(componentDuration) / float64(len(metrics)),
			ErrorRate:       float64(componentErrors) / float64(len(metrics)),
		}
	}

	return stats
}

// RunBenchmark runs performance benchmarks and returns results.
func (pm *PerformanceMonitor) RunBenchmark(testDataCount int) (*BenchmarkResult, error) {
	result := &BenchmarkResult{
		StartTime:     time.Now(),
		TestDataCount: testDataCount,
		Operations:    make(map[string]OperationBenchmark),
	}

	// Generate test data if needed
	if err := pm.generateTestData(testDataCount); err != nil {
		return result, err
	}

	// Benchmark session operations
	if err := pm.benchmarkSessionOperations(result); err != nil {
		return result, err
	}

	// Benchmark search operations
	if err := pm.benchmarkSearchOperations(result); err != nil {
		return result, err
	}

	// Benchmark file operations
	if err := pm.benchmarkFileOperations(result); err != nil {
		return result, err
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// generateTestData creates test data for benchmarking.
func (pm *PerformanceMonitor) generateTestData(count int) error {
	log.Printf("Generating %d test sessions for benchmarking...", count)

	for i := 0; i < count; i++ {
		// Create session with date
		date := time.Now().Add(-time.Duration(i) * 24 * time.Hour).Format("2006-01-02")
		
		session, err := pm.storageEngine.CreateSession(date)
		if err != nil {
			continue // Skip if session already exists
		}

		// Add some content
		session.CustomTitle = fmt.Sprintf("Benchmark Session %d", i)
		session.CustomSummary = fmt.Sprintf("This is a test session for benchmarking purposes. Session number %d.", i)
		session.ExtractedText = fmt.Sprintf("Extracted text content for session %d with some searchable keywords.", i)

		if err := pm.storageEngine.UpdateSession(session); err != nil {
			continue
		}

		// Add activity blocks
		for j := 0; j < 3; j++ {
			block := &ActivityBlock{
				BlockID:      fmt.Sprintf("%02d-%02d", 9+j, 30),
				StartTime:    time.Now().Add(-time.Duration(j) * time.Hour),
				EndTime:      time.Now().Add(-time.Duration(j) * time.Hour + 30*time.Minute),
				OCRText:      fmt.Sprintf("OCR text for block %d in session %d", j, i),
				MicroSummary: fmt.Sprintf("Summary for block %d", j),
			}

			pm.storageEngine.AddActivityBlock(date, "BenchmarkApp", block)
		}
	}

	log.Printf("Generated test data successfully")
	return nil
}

// benchmarkSessionOperations benchmarks session CRUD operations.
func (pm *PerformanceMonitor) benchmarkSessionOperations(result *BenchmarkResult) error {
	// Benchmark session lookup
	start := time.Now()
	iterations := 100

	for i := 0; i < iterations; i++ {
		date := time.Now().Add(-time.Duration(i%result.TestDataCount) * 24 * time.Hour).Format("2006-01-02")
		_, err := pm.storageEngine.GetSession(date)
		if err != nil {
			// Session might not exist, which is OK for benchmarking
		}
	}

	duration := time.Since(start)
	result.Operations["session_lookup"] = OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	return nil
}

// benchmarkSearchOperations benchmarks search operations.
func (pm *PerformanceMonitor) benchmarkSearchOperations(result *BenchmarkResult) error {
	// Benchmark full-text search
	start := time.Now()
	iterations := 50

	searchTerms := []string{"benchmark", "test", "session", "content", "text"}

	for i := 0; i < iterations; i++ {
		term := searchTerms[i%len(searchTerms)]
		_, err := pm.storageEngine.FullTextSearch(term, 1, 10)
		if err != nil {
			// Search errors are OK for benchmarking
		}
	}

	duration := time.Since(start)
	result.Operations["fulltext_search"] = OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	// Benchmark semantic search
	start = time.Now()
	iterations = 20

	for i := 0; i < iterations; i++ {
		term := searchTerms[i%len(searchTerms)]
		_, err := pm.storageEngine.SemanticSearch(term, 10, nil)
		if err != nil {
			// Search errors are OK for benchmarking
		}
	}

	duration = time.Since(start)
	result.Operations["semantic_search"] = OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	return nil
}

// benchmarkFileOperations benchmarks file operations.
func (pm *PerformanceMonitor) benchmarkFileOperations(result *BenchmarkResult) error {
	// Benchmark file save
	start := time.Now()
	iterations := 50

	testData := make([]byte, 1024) // 1KB test file
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for i := 0; i < iterations; i++ {
		date := time.Now().Add(-time.Duration(i%10) * 24 * time.Hour).Format("2006-01-02")
		filename := fmt.Sprintf("benchmark_%d.png", i)
		
		_, err := pm.storageEngine.SaveScreenshot(date, "BenchmarkApp", filename, testData)
		if err != nil {
			// File save errors are OK for benchmarking
		}
	}

	duration := time.Since(start)
	result.Operations["file_save"] = OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	return nil
}

// Close closes the performance monitor and its log file.
func (pm *PerformanceMonitor) Close() error {
	if pm.logFile != nil {
		return pm.logFile.Close()
	}
	return nil
}

// BenchmarkResult contains the results of a performance benchmark.
type BenchmarkResult struct {
	StartTime     time.Time                      `json:"startTime"`
	EndTime       time.Time                      `json:"endTime"`
	TotalDuration time.Duration                  `json:"totalDuration"`
	TestDataCount int                            `json:"testDataCount"`
	Operations    map[string]OperationBenchmark  `json:"operations"`
}

// OperationBenchmark contains benchmark results for a specific operation.
type OperationBenchmark struct {
	Iterations       int           `json:"iterations"`
	TotalDuration    time.Duration `json:"totalDuration"`
	AverageDuration  time.Duration `json:"averageDuration"`
	OperationsPerSec float64       `json:"operationsPerSec"`
}

// GetStorageMetrics returns current storage metrics.
func (pm *PerformanceMonitor) GetStorageMetrics() (*StorageMetrics, error) {
	metrics := &StorageMetrics{
		Timestamp: time.Now(),
	}

	// Get database size
	dbPath := filepath.Join(pm.config.DataDir, "waddle.db")
	if info, err := os.Stat(dbPath); err == nil {
		metrics.DatabaseSize = info.Size()
	}

	// Get vector database size
	vectorPath := filepath.Join(pm.config.DataDir, "vectors")
	if size, err := pm.calculateDirectorySize(vectorPath); err == nil {
		metrics.VectorDatabaseSize = size
	}

	// Get files directory size
	filesPath := filepath.Join(pm.config.DataDir, "files")
	if size, err := pm.calculateDirectorySize(filesPath); err == nil {
		metrics.FilesSize = size
	}

	// Get session count
	if _, count, err := pm.storageEngine.ListSessions(1, 1); err == nil {
		metrics.SessionCount = int64(count)
	}

	// Get file statistics
	if pm.storageEngine.fileMgr != nil {
		if stats, err := pm.storageEngine.fileMgr.GetStorageStats(); err == nil {
			metrics.TotalFiles = stats.TotalFiles
		}
	}

	metrics.TotalSize = metrics.DatabaseSize + metrics.VectorDatabaseSize + metrics.FilesSize

	return metrics, nil
}

// calculateDirectorySize calculates the total size of a directory.
func (pm *PerformanceMonitor) calculateDirectorySize(dirPath string) (int64, error) {
	var size int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// StorageMetrics contains current storage usage metrics.
type StorageMetrics struct {
	Timestamp           time.Time `json:"timestamp"`
	DatabaseSize        int64     `json:"databaseSize"`
	VectorDatabaseSize  int64     `json:"vectorDatabaseSize"`
	FilesSize           int64     `json:"filesSize"`
	TotalSize           int64     `json:"totalSize"`
	SessionCount        int64     `json:"sessionCount"`
	TotalFiles          int64     `json:"totalFiles"`
}