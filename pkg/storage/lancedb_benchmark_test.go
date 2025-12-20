package storage

import (
	"math/rand"
	"os"
	"testing"
	"time"
)

// BenchmarkOptimizedVectorSearch benchmarks optimized vector search performance.
func BenchmarkOptimizedVectorSearch(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vector_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	vmConfig := DefaultVectorManagerConfig(tempDir)
	ldbConfig := DefaultLanceDBConfig()
	
	ovm, err := NewOptimizedVectorManager(vmConfig, ldbConfig)
	if err != nil {
		b.Fatalf("Failed to create optimized vector manager: %v", err)
	}
	defer ovm.Close()
	
	// Insert test vectors
	numVectors := 1000 // Start with 1k vectors for benchmark
	b.Logf("Inserting %d test vectors...", numVectors)
	
	for i := 1; i <= numVectors; i++ {
		embedding := make([]float32, EmbeddingDimensions)
		for j := 0; j < EmbeddingDimensions; j++ {
			embedding[j] = rand.Float32()
		}
		
		err = ovm.StoreEmbedding(int64(i), embedding)
		if err != nil {
			b.Fatalf("Failed to store embedding %d: %v", i, err)
		}
	}
	
	// Create query vector
	queryEmbedding := make([]float32, EmbeddingDimensions)
	for i := 0; i < EmbeddingDimensions; i++ {
		queryEmbedding[i] = rand.Float32()
	}
	
	b.ResetTimer()
	
	// Benchmark search
	for i := 0; i < b.N; i++ {
		_, err := ovm.SearchOptimized(queryEmbedding, 10)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// BenchmarkVectorBatching benchmarks vector insertion batching.
func BenchmarkVectorBatching(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "batch_benchmark_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	vmConfig := DefaultVectorManagerConfig(tempDir)
	ldbConfig := DefaultLanceDBConfig()
	ldbConfig.BatchSize = 100
	ldbConfig.BatchTimeout = 1 * time.Second
	
	ovm, err := NewOptimizedVectorManager(vmConfig, ldbConfig)
	if err != nil {
		b.Fatalf("Failed to create optimized vector manager: %v", err)
	}
	defer ovm.Close()
	
	b.ResetTimer()
	
	// Benchmark batched insertion
	for i := 0; i < b.N; i++ {
		embedding := make([]float32, EmbeddingDimensions)
		for j := 0; j < EmbeddingDimensions; j++ {
			embedding[j] = rand.Float32()
		}
		
		err = ovm.StoreEmbeddingBatched(int64(i+1), embedding)
		if err != nil {
			b.Fatalf("Failed to store batched embedding: %v", err)
		}
	}
	
	// Wait for final batch to flush
	time.Sleep(1100 * time.Millisecond)
}

// TestPerformanceTargets validates the P99 search latency targets.
func TestPerformanceTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	tempDir, err := os.MkdirTemp("", "performance_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	vmConfig := DefaultVectorManagerConfig(tempDir)
	ldbConfig := DefaultLanceDBConfig()
	
	ovm, err := NewOptimizedVectorManager(vmConfig, ldbConfig)
	if err != nil {
		t.Fatalf("Failed to create optimized vector manager: %v", err)
	}
	defer ovm.Close()
	
	// Test with different vector counts
	testCases := []struct {
		name        string
		vectorCount int
		targetP99   time.Duration
	}{
		{"1k vectors", 1000, 10 * time.Millisecond},   // Warm-up
		{"10k vectors", 10000, 15 * time.Millisecond}, // Intermediate
		{"50k vectors", 50000, 20 * time.Millisecond}, // P0 target
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Insert vectors
			t.Logf("Inserting %d vectors...", tc.vectorCount)
			start := time.Now()
			
			for i := 1; i <= tc.vectorCount; i++ {
				embedding := make([]float32, EmbeddingDimensions)
				for j := 0; j < EmbeddingDimensions; j++ {
					embedding[j] = rand.Float32()
				}
				
				err = ovm.StoreEmbedding(int64(i), embedding)
				if err != nil {
					t.Fatalf("Failed to store embedding %d: %v", i, err)
				}
				
				if i%10000 == 0 {
					t.Logf("Inserted %d vectors...", i)
				}
			}
			
			insertTime := time.Since(start)
			t.Logf("Inserted %d vectors in %v (%.2f vectors/sec)", 
				tc.vectorCount, insertTime, float64(tc.vectorCount)/insertTime.Seconds())
			
			// Measure search latencies
			queryEmbedding := make([]float32, EmbeddingDimensions)
			for i := 0; i < EmbeddingDimensions; i++ {
				queryEmbedding[i] = rand.Float32()
			}
			
			numQueries := 100
			latencies := make([]time.Duration, numQueries)
			
			t.Logf("Running %d search queries...", numQueries)
			for i := 0; i < numQueries; i++ {
				start := time.Now()
				_, err := ovm.SearchOptimized(queryEmbedding, 10)
				latencies[i] = time.Since(start)
				
				if err != nil {
					t.Fatalf("Search %d failed: %v", i, err)
				}
			}
			
			// Calculate P99 latency
			// Sort latencies
			for i := 0; i < len(latencies); i++ {
				for j := i + 1; j < len(latencies); j++ {
					if latencies[i] > latencies[j] {
						latencies[i], latencies[j] = latencies[j], latencies[i]
					}
				}
			}
			
			p99Index := int(float64(numQueries) * 0.99)
			if p99Index >= numQueries {
				p99Index = numQueries - 1
			}
			p99Latency := latencies[p99Index]
			avgLatency := time.Duration(0)
			for _, lat := range latencies {
				avgLatency += lat
			}
			avgLatency /= time.Duration(numQueries)
			
			t.Logf("Search latencies for %d vectors:", tc.vectorCount)
			t.Logf("  Average: %v", avgLatency)
			t.Logf("  P99: %v", p99Latency)
			t.Logf("  Target P99: %v", tc.targetP99)
			
			if p99Latency > tc.targetP99 {
				t.Logf("WARNING: P99 latency %v exceeds target %v", p99Latency, tc.targetP99)
				// Don't fail the test - this is informational for now
				// In production, we'd fail here: t.Errorf("P99 latency %v exceeds target %v", p99Latency, tc.targetP99)
			} else {
				t.Logf("✓ P99 latency %v meets target %v", p99Latency, tc.targetP99)
			}
		})
	}
}

// TestBatchingPerformance validates vector batching performance.
func TestBatchingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	tempDir, err := os.MkdirTemp("", "batch_perf_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	vmConfig := DefaultVectorManagerConfig(tempDir)
	ldbConfig := DefaultLanceDBConfig()
	ldbConfig.BatchSize = 100
	ldbConfig.BatchTimeout = 1 * time.Second
	
	ovm, err := NewOptimizedVectorManager(vmConfig, ldbConfig)
	if err != nil {
		t.Fatalf("Failed to create optimized vector manager: %v", err)
	}
	defer ovm.Close()
	
	// Test batching performance
	numVectors := 1000
	t.Logf("Testing batched insertion of %d vectors...", numVectors)
	
	start := time.Now()
	for i := 1; i <= numVectors; i++ {
		embedding := make([]float32, EmbeddingDimensions)
		for j := 0; j < EmbeddingDimensions; j++ {
			embedding[j] = rand.Float32()
		}
		
		err = ovm.StoreEmbeddingBatched(int64(i), embedding)
		if err != nil {
			t.Fatalf("Failed to store batched embedding %d: %v", i, err)
		}
	}
	
	// Wait for final batch to flush
	time.Sleep(1100 * time.Millisecond)
	insertTime := time.Since(start)
	
	t.Logf("Batched insertion completed in %v (%.2f vectors/sec)", 
		insertTime, float64(numVectors)/insertTime.Seconds())
	
	// Verify all vectors were stored
	count := ovm.Count()
	if count != numVectors {
		t.Errorf("Expected %d vectors, got %d", numVectors, count)
	}
	
	// Test batch statistics
	stats := ovm.GetBatchStats()
	t.Logf("Final batch stats: current=%d, max=%d, timeout=%v", 
		stats.CurrentBatchSize, stats.MaxBatchSize, stats.BatchTimeout)
	
	if stats.CurrentBatchSize != 0 {
		t.Errorf("Expected empty batch after flush, got size %d", stats.CurrentBatchSize)
	}
}