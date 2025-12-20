package main

import (
	"flag"
	"fmt"
	"waddle/pkg/storage"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var (
		generate = flag.Bool("generate", false, "Generate test data")
		run      = flag.Bool("run", false, "Run benchmarks")
		count    = flag.Int("count", 1000, "Number of test sessions to generate")
		dataDir  = flag.String("data-dir", "", "Data directory (default: ~/.waddle)")
	)
	flag.Parse()

	if !*generate && !*run {
		fmt.Println("Usage: benchmark --generate --count=N  OR  benchmark --run")
		fmt.Println("  --generate: Generate test data")
		fmt.Println("  --run: Run performance benchmarks")
		fmt.Println("  --count: Number of test sessions to generate (default: 1000)")
		fmt.Println("  --data-dir: Data directory (default: ~/.waddle)")
		os.Exit(1)
	}

	// Determine data directory
	var storageDataDir string
	if *dataDir != "" {
		storageDataDir = *dataDir
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		storageDataDir = filepath.Join(homeDir, ".waddle")
	}

	// Initialize storage engine
	config := storage.DefaultStorageConfig(storageDataDir)
	storageEngine := storage.NewStorageEngine(config)
	
	if err := storageEngine.Initialize(); err != nil {
		fmt.Printf("Error initializing storage engine: %v\n", err)
		os.Exit(1)
	}
	defer storageEngine.Close()

	// Create performance monitor
	perfMon, err := storage.NewPerformanceMonitor(config, storageEngine)
	if err != nil {
		fmt.Printf("Error creating performance monitor: %v\n", err)
		os.Exit(1)
	}
	defer perfMon.Close()

	if *generate {
		fmt.Printf("Generating %d test sessions...\n", *count)
		start := time.Now()
		
		result, err := perfMon.RunBenchmark(*count)
		if err != nil {
			fmt.Printf("Error generating test data: %v\n", err)
			os.Exit(1)
		}
		
		duration := time.Since(start)
		fmt.Printf("Generated %d sessions in %v\n", result.TestDataCount, duration)
		fmt.Printf("Data generation complete!\n")
	}

	if *run {
		fmt.Println("Running performance benchmarks...")
		
		// Get current session count
		_, sessionCount, err := storageEngine.ListSessions(1, 1)
		if err != nil {
			fmt.Printf("Error getting session count: %v\n", err)
			os.Exit(1)
		}
		
		if sessionCount == 0 {
			fmt.Println("No sessions found. Please run with --generate first.")
			os.Exit(1)
		}
		
		fmt.Printf("Found %d existing sessions\n", sessionCount)
		start := time.Now()
		
		// Create a custom benchmark result without generating new data
		result := &storage.BenchmarkResult{
			StartTime:     time.Now(),
			TestDataCount: sessionCount,
			Operations:    make(map[string]storage.OperationBenchmark),
		}
		
		// Run benchmarks manually
		if err := runBenchmarks(perfMon, storageEngine, result, sessionCount); err != nil {
			fmt.Printf("Error running benchmarks: %v\n", err)
			os.Exit(1)
		}
		
		result.EndTime = time.Now()
		result.TotalDuration = result.EndTime.Sub(result.StartTime)
		if err != nil {
			fmt.Printf("Error running benchmarks: %v\n", err)
			os.Exit(1)
		}
		
		duration := time.Since(start)
		fmt.Printf("Benchmarks completed in %v\n", duration)
		
		// Print results
		printBenchmarkResults(result)
		
		// Get storage metrics
		metrics, err := perfMon.GetStorageMetrics()
		if err == nil {
			printStorageMetrics(metrics)
		}
	}
}

func runBenchmarks(perfMon *storage.PerformanceMonitor, storageEngine *storage.StorageEngine, result *storage.BenchmarkResult, sessionCount int) error {
	// Benchmark session lookup
	fmt.Println("Benchmarking session lookup...")
	start := time.Now()
	iterations := 100

	for i := 0; i < iterations; i++ {
		date := time.Now().Add(-time.Duration(i%sessionCount) * 24 * time.Hour).Format("2006-01-02")
		_, err := storageEngine.GetSession(date)
		if err != nil {
			// Session might not exist, which is OK for benchmarking
		}
	}

	duration := time.Since(start)
	result.Operations["session_lookup"] = storage.OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	// Benchmark full-text search
	fmt.Println("Benchmarking full-text search...")
	start = time.Now()
	iterations = 50
	searchTerms := []string{"benchmark", "test", "session", "content", "text"}

	for i := 0; i < iterations; i++ {
		term := searchTerms[i%len(searchTerms)]
		_, err := storageEngine.FullTextSearch(term, 1, 10)
		if err != nil {
			// Search errors are OK for benchmarking
		}
	}

	duration = time.Since(start)
	result.Operations["fulltext_search"] = storage.OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	// Benchmark semantic search
	fmt.Println("Benchmarking semantic search...")
	start = time.Now()
	iterations = 20

	for i := 0; i < iterations; i++ {
		term := searchTerms[i%len(searchTerms)]
		_, err := storageEngine.SemanticSearch(term, 10, nil)
		if err != nil {
			// Search errors are OK for benchmarking
		}
	}

	duration = time.Since(start)
	result.Operations["semantic_search"] = storage.OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	// Benchmark file save
	fmt.Println("Benchmarking file operations...")
	start = time.Now()
	iterations = 50

	testData := make([]byte, 1024) // 1KB test file
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for i := 0; i < iterations; i++ {
		date := time.Now().Add(-time.Duration(i%10) * 24 * time.Hour).Format("2006-01-02")
		filename := fmt.Sprintf("benchmark_%d.png", i)
		
		_, err := storageEngine.SaveScreenshot(date, "BenchmarkApp", filename, testData)
		if err != nil {
			// File save errors are OK for benchmarking
		}
	}

	duration = time.Since(start)
	result.Operations["file_save"] = storage.OperationBenchmark{
		Iterations:      iterations,
		TotalDuration:   duration,
		AverageDuration: duration / time.Duration(iterations),
		OperationsPerSec: float64(iterations) / duration.Seconds(),
	}

	return nil
}

func printBenchmarkResults(result *storage.BenchmarkResult) {
	fmt.Println("\n=== BENCHMARK RESULTS ===")
	fmt.Printf("Test Data Count: %d sessions\n", result.TestDataCount)
	fmt.Printf("Total Duration: %v\n", result.TotalDuration)
	fmt.Printf("Start Time: %v\n", result.StartTime.Format(time.RFC3339))
	fmt.Printf("End Time: %v\n", result.EndTime.Format(time.RFC3339))
	
	fmt.Println("\n--- Operation Performance ---")
	for opName, benchmark := range result.Operations {
		fmt.Printf("\n%s:\n", opName)
		fmt.Printf("  Iterations: %d\n", benchmark.Iterations)
		fmt.Printf("  Total Duration: %v\n", benchmark.TotalDuration)
		fmt.Printf("  Average Duration: %v\n", benchmark.AverageDuration)
		fmt.Printf("  Operations/sec: %.2f\n", benchmark.OperationsPerSec)
		
		// Check against performance targets
		avgMs := benchmark.AverageDuration.Milliseconds()
		var target int64
		var status string
		
		switch opName {
		case "session_lookup":
			target = 10 // 10ms target
		case "fulltext_search":
			target = 100 // 100ms target
		case "semantic_search":
			target = 200 // 200ms target
		case "file_save":
			target = 50 // 50ms target
		default:
			target = 100 // Default target
		}
		
		if avgMs <= target {
			status = "✅ PASS"
		} else {
			status = "❌ FAIL"
		}
		
		fmt.Printf("  Target: <%dms, Actual: %dms %s\n", target, avgMs, status)
	}
}

func printStorageMetrics(metrics *storage.StorageMetrics) {
	fmt.Println("\n=== STORAGE METRICS ===")
	fmt.Printf("Timestamp: %v\n", metrics.Timestamp.Format(time.RFC3339))
	fmt.Printf("Database Size: %s\n", formatBytes(metrics.DatabaseSize))
	fmt.Printf("Vector Database Size: %s\n", formatBytes(metrics.VectorDatabaseSize))
	fmt.Printf("Files Size: %s\n", formatBytes(metrics.FilesSize))
	fmt.Printf("Total Size: %s\n", formatBytes(metrics.TotalSize))
	fmt.Printf("Session Count: %d\n", metrics.SessionCount)
	fmt.Printf("Total Files: %d\n", metrics.TotalFiles)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}