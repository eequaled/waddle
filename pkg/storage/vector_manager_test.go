package storage

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// createNormalizedEmbedding creates a normalized embedding vector of the given dimensions.
func createNormalizedEmbedding(dimensions int) []float32 {
	embedding := make([]float32, dimensions)
	// Initialize with small values
	for i := range embedding {
		embedding[i] = float32(i%10) / 100.0
	}
	normalizeEmbedding(embedding)
	return embedding
}

// normalizeEmbedding normalizes an embedding vector to unit length.
func normalizeEmbedding(embedding []float32) {
	var sum float64
	for _, v := range embedding {
		sum += float64(v * v)
	}
	norm := float32(math.Sqrt(sum))
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}
}

// TestVectorManagerInitialization tests basic VectorManager creation.
func TestVectorManagerInitialization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("successful initialization", func(t *testing.T) {
		config := DefaultVectorManagerConfig(tempDir)
		vm, err := NewVectorManager(config)
		if err != nil {
			t.Fatalf("Failed to create VectorManager: %v", err)
		}
		defer vm.Close()

		vectorsDir := filepath.Join(tempDir, "vectors")
		if _, err := os.Stat(vectorsDir); os.IsNotExist(err) {
			t.Error("Vectors directory was not created")
		}

		if vm.GetModelVersion() != DefaultEmbeddingModel {
			t.Errorf("Expected model version %s, got %s", DefaultEmbeddingModel, vm.GetModelVersion())
		}
	})

	t.Run("nil config", func(t *testing.T) {
		_, err := NewVectorManager(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("empty data directory", func(t *testing.T) {
		config := &VectorManagerConfig{DataDir: ""}
		_, err := NewVectorManager(config)
		if err == nil {
			t.Error("Expected error for empty data directory")
		}
	})
}

// TestVectorManagerValidation tests input validation.
func TestVectorManagerValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_validation_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	t.Run("StoreEmbedding with invalid session ID", func(t *testing.T) {
		embedding := make([]float32, EmbeddingDimensions)
		err := vm.StoreEmbedding(0, embedding)
		if err == nil {
			t.Error("Expected error for zero session ID")
		}

		err = vm.StoreEmbedding(-1, embedding)
		if err == nil {
			t.Error("Expected error for negative session ID")
		}
	})

	t.Run("StoreEmbedding with wrong dimensions", func(t *testing.T) {
		embedding := make([]float32, 100)
		err := vm.StoreEmbedding(1, embedding)
		if err == nil {
			t.Error("Expected error for wrong embedding dimensions")
		}
	})

	t.Run("Search with wrong dimensions", func(t *testing.T) {
		embedding := make([]float32, 100)
		_, err := vm.Search(embedding, 10)
		if err == nil {
			t.Error("Expected error for wrong query embedding dimensions")
		}
	})

	t.Run("Search with invalid topK", func(t *testing.T) {
		embedding := make([]float32, EmbeddingDimensions)
		_, err := vm.Search(embedding, 0)
		if err == nil {
			t.Error("Expected error for zero topK")
		}

		_, err = vm.Search(embedding, -1)
		if err == nil {
			t.Error("Expected error for negative topK")
		}
	})

	t.Run("DeleteEmbedding with invalid session ID", func(t *testing.T) {
		err := vm.DeleteEmbedding(0)
		if err == nil {
			t.Error("Expected error for zero session ID")
		}
	})

	t.Run("QueueEmbedding with invalid inputs", func(t *testing.T) {
		err := vm.QueueEmbedding(0, "test")
		if err == nil {
			t.Error("Expected error for zero session ID")
		}

		err = vm.QueueEmbedding(1, "")
		if err == nil {
			t.Error("Expected error for empty text")
		}
	})
}


// TestVectorManagerCRUD tests basic CRUD operations with mock embeddings.
func TestVectorManagerCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_crud_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	embedding := createNormalizedEmbedding(EmbeddingDimensions)

	t.Run("Store and retrieve embedding", func(t *testing.T) {
		sessionID := int64(1)

		err := vm.StoreEmbedding(sessionID, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		if !vm.HasEmbedding(sessionID) {
			t.Error("Embedding should exist after storing")
		}

		retrieved, modelVersion, err := vm.GetEmbedding(sessionID)
		if err != nil {
			t.Fatalf("Failed to get embedding: %v", err)
		}

		if len(retrieved) != EmbeddingDimensions {
			t.Errorf("Expected %d dimensions, got %d", EmbeddingDimensions, len(retrieved))
		}

		if modelVersion != DefaultEmbeddingModel {
			t.Errorf("Expected model version %s, got %s", DefaultEmbeddingModel, modelVersion)
		}

		if vm.Count() != 1 {
			t.Errorf("Expected count 1, got %d", vm.Count())
		}
	})

	t.Run("Update embedding", func(t *testing.T) {
		sessionID := int64(2)

		err := vm.StoreEmbedding(sessionID, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		newEmbedding := createNormalizedEmbedding(EmbeddingDimensions)
		newEmbedding[0] = 0.5

		err = vm.UpdateEmbedding(sessionID, newEmbedding)
		if err != nil {
			t.Fatalf("Failed to update embedding: %v", err)
		}

		if !vm.HasEmbedding(sessionID) {
			t.Error("Embedding should exist after update")
		}
	})

	t.Run("Delete embedding", func(t *testing.T) {
		sessionID := int64(3)

		err := vm.StoreEmbedding(sessionID, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		err = vm.DeleteEmbedding(sessionID)
		if err != nil {
			t.Fatalf("Failed to delete embedding: %v", err)
		}

		if vm.HasEmbedding(sessionID) {
			t.Error("Embedding should not exist after deletion")
		}
	})
}

// TestVectorManagerSearch tests semantic search functionality.
func TestVectorManagerSearch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_search_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	for i := int64(1); i <= 5; i++ {
		embedding := createNormalizedEmbedding(EmbeddingDimensions)
		embedding[0] = float32(i) / 10.0
		normalizeEmbedding(embedding)

		err := vm.StoreEmbedding(i, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding %d: %v", i, err)
		}
	}

	t.Run("Search returns results", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding(EmbeddingDimensions)
		results, err := vm.Search(queryEmbedding, 3)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected search results")
		}

		if len(results) > 3 {
			t.Errorf("Expected at most 3 results, got %d", len(results))
		}
	})

	t.Run("Search results have valid fields", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding(EmbeddingDimensions)
		results, err := vm.Search(queryEmbedding, 5) // Request at most 5 (we have 5 docs)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		for _, r := range results {
			if r.SessionID <= 0 {
				t.Errorf("Invalid session ID: %d", r.SessionID)
			}
			if r.Score < 0 || r.Score > 1 {
				t.Errorf("Invalid similarity score: %f", r.Score)
			}
			if r.ModelVersion == "" {
				t.Error("Model version should not be empty")
			}
		}
	})

	t.Run("Search results ordered by similarity", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding(EmbeddingDimensions)
		results, err := vm.Search(queryEmbedding, 5) // Request at most 5 (we have 5 docs)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		for i := 1; i < len(results); i++ {
			if results[i].Score > results[i-1].Score {
				t.Errorf("Results not ordered by similarity: %f > %f at index %d",
					results[i].Score, results[i-1].Score, i)
			}
		}
	})
}


// TestPropertySessionEmbeddingInvariant is Property Test 2: Session Embedding Invariant
// For any session with non-empty text content, after saving:
// - An embedding vector of exactly 768 dimensions SHALL exist
// - The embedding SHALL reference the correct session_id
// - The embedding SHALL include a valid model_version string
func TestPropertySessionEmbeddingInvariant(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_property_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genSessionID := gen.Int64Range(1, 1000000)

	genEmbedding := gen.SliceOfN(EmbeddingDimensions, gen.Float32Range(-1.0, 1.0)).
		Map(func(v []float32) []float32 {
			normalizeEmbedding(v)
			return v
		})

	properties.Property("Session embedding invariant", prop.ForAll(
		func(sessionID int64, embedding []float32) bool {
			err := vm.StoreEmbedding(sessionID, embedding)
			if err != nil {
				t.Logf("Failed to store embedding: %v", err)
				return false
			}

			retrieved, modelVersion, err := vm.GetEmbedding(sessionID)
			if err != nil {
				t.Logf("Failed to get embedding: %v", err)
				return false
			}

			// Property 1: Embedding has exactly 768 dimensions
			if len(retrieved) != EmbeddingDimensions {
				t.Logf("Wrong dimensions: got %d, expected %d", len(retrieved), EmbeddingDimensions)
				return false
			}

			// Property 2: Model version is valid (non-empty)
			if modelVersion == "" {
				t.Logf("Model version is empty")
				return false
			}

			// Property 3: Model version matches expected
			if modelVersion != DefaultEmbeddingModel {
				t.Logf("Wrong model version: got %s, expected %s", modelVersion, DefaultEmbeddingModel)
				return false
			}

			vm.DeleteEmbedding(sessionID)
			return true
		},
		genSessionID,
		genEmbedding,
	))

	properties.TestingRun(t)
}

// TestVectorManagerPersistence tests that embeddings persist across restarts.
func TestVectorManagerPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_persist_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sessionID := int64(42)
	embedding := createNormalizedEmbedding(EmbeddingDimensions)

	{
		config := DefaultVectorManagerConfig(tempDir)
		vm, err := NewVectorManager(config)
		if err != nil {
			t.Fatalf("Failed to create VectorManager: %v", err)
		}

		err = vm.StoreEmbedding(sessionID, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}

		vm.Close()
	}

	{
		config := DefaultVectorManagerConfig(tempDir)
		vm, err := NewVectorManager(config)
		if err != nil {
			t.Fatalf("Failed to create VectorManager: %v", err)
		}
		defer vm.Close()

		if !vm.HasEmbedding(sessionID) {
			t.Error("Embedding should persist across restarts")
		}

		retrieved, _, err := vm.GetEmbedding(sessionID)
		if err != nil {
			t.Fatalf("Failed to get embedding: %v", err)
		}

		if len(retrieved) != EmbeddingDimensions {
			t.Errorf("Expected %d dimensions, got %d", EmbeddingDimensions, len(retrieved))
		}
	}
}

// TestVectorManagerQueueing tests the async embedding queue.
func TestVectorManagerQueueing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_queue_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	config.QueueSize = 10
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	t.Run("Queue accepts valid requests", func(t *testing.T) {
		err := vm.QueueEmbedding(1, "test text")
		if err != nil {
			t.Errorf("Failed to queue embedding: %v", err)
		}
	})

	t.Run("Queue rejects invalid requests", func(t *testing.T) {
		err := vm.QueueEmbedding(0, "test")
		if err == nil {
			t.Error("Expected error for invalid session ID")
		}

		err = vm.QueueEmbedding(1, "")
		if err == nil {
			t.Error("Expected error for empty text")
		}
	})
}

// TestGenerateEmbeddingValidation tests GenerateEmbedding input validation.
func TestGenerateEmbeddingValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_gen_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	t.Run("Empty text returns error", func(t *testing.T) {
		_, err := vm.GenerateEmbedding("")
		if err == nil {
			t.Error("Expected error for empty text")
		}
	})
}

// TestDefaultVectorManagerConfig tests default configuration.
func TestDefaultVectorManagerConfig(t *testing.T) {
	config := DefaultVectorManagerConfig("/test/path")

	if config.DataDir != "/test/path" {
		t.Errorf("Expected DataDir /test/path, got %s", config.DataDir)
	}

	if config.OllamaURL != DefaultOllamaURL {
		t.Errorf("Expected OllamaURL %s, got %s", DefaultOllamaURL, config.OllamaURL)
	}

	if config.ModelVersion != DefaultEmbeddingModel {
		t.Errorf("Expected ModelVersion %s, got %s", DefaultEmbeddingModel, config.ModelVersion)
	}

	if config.QueueSize != DefaultEmbedQueueSize {
		t.Errorf("Expected QueueSize %d, got %d", DefaultEmbedQueueSize, config.QueueSize)
	}
}


// TestVectorManagerErrorCodes tests that errors have correct error codes.
func TestVectorManagerErrorCodes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_errors_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	t.Run("Validation errors have correct code", func(t *testing.T) {
		err := vm.StoreEmbedding(0, make([]float32, EmbeddingDimensions))
		if storageErr, ok := err.(*StorageError); ok {
			if storageErr.Code != ErrValidation {
				t.Errorf("Expected ErrValidation, got %v", storageErr.Code)
			}
		} else {
			t.Error("Expected StorageError type")
		}
	})

	t.Run("Not found errors have correct code", func(t *testing.T) {
		_, _, err := vm.GetEmbedding(999999)
		if storageErr, ok := err.(*StorageError); ok {
			if storageErr.Code != ErrNotFound {
				t.Errorf("Expected ErrNotFound, got %v", storageErr.Code)
			}
		} else {
			t.Error("Expected StorageError type")
		}
	})
}

// TestVectorManagerMultipleEmbeddings tests storing multiple embeddings.
func TestVectorManagerMultipleEmbeddings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_multi_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	numEmbeddings := 100
	for i := int64(1); i <= int64(numEmbeddings); i++ {
		embedding := createNormalizedEmbedding(EmbeddingDimensions)
		embedding[0] = float32(i) / float32(numEmbeddings)
		normalizeEmbedding(embedding)

		err := vm.StoreEmbedding(i, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding %d: %v", i, err)
		}
	}

	if vm.Count() != numEmbeddings {
		t.Errorf("Expected count %d, got %d", numEmbeddings, vm.Count())
	}

	for i := int64(1); i <= int64(numEmbeddings); i++ {
		if !vm.HasEmbedding(i) {
			t.Errorf("Embedding %d should exist", i)
		}
	}

	queryEmbedding := createNormalizedEmbedding(EmbeddingDimensions)
	results, err := vm.Search(queryEmbedding, 50)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 50 {
		t.Errorf("Expected 50 results, got %d", len(results))
	}
}

// TestVectorManagerReindex tests the reindex functionality.
func TestVectorManagerReindex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_reindex_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	if vm.GetModelVersion() != DefaultEmbeddingModel {
		t.Errorf("Expected initial model version %s", DefaultEmbeddingModel)
	}

	newModel := "new-model-v2"
	err = vm.Reindex(newModel)
	if err != nil {
		t.Fatalf("Reindex failed: %v", err)
	}

	if vm.GetModelVersion() != newModel {
		t.Errorf("Expected model version %s, got %s", newModel, vm.GetModelVersion())
	}
}

// TestVectorManagerConcurrentAccess tests thread safety.
func TestVectorManagerConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_concurrent_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	done := make(chan bool)
	numGoroutines := 10
	numOps := 10

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < numOps; i++ {
				sessionID := int64(goroutineID*numOps + i + 1)
				embedding := createNormalizedEmbedding(EmbeddingDimensions)

				_ = vm.StoreEmbedding(sessionID, embedding)
				_, _, _ = vm.GetEmbedding(sessionID)
				_, _ = vm.Search(embedding, 5)
			}
			done <- true
		}(g)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	if vm.Count() == 0 {
		t.Error("Expected some embeddings to be stored")
	}
}

// BenchmarkStoreEmbedding benchmarks embedding storage.
func BenchmarkStoreEmbedding(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vector_manager_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		b.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	embedding := createNormalizedEmbedding(EmbeddingDimensions)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := int64(i + 1)
		_ = vm.StoreEmbedding(sessionID, embedding)
	}
}

// BenchmarkVectorSearch benchmarks semantic search.
func BenchmarkVectorSearch(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vector_manager_bench_search_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		b.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	for i := int64(1); i <= 1000; i++ {
		embedding := createNormalizedEmbedding(EmbeddingDimensions)
		embedding[0] = float32(i) / 1000.0
		normalizeEmbedding(embedding)
		_ = vm.StoreEmbedding(i, embedding)
	}

	queryEmbedding := createNormalizedEmbedding(EmbeddingDimensions)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = vm.Search(queryEmbedding, 10)
	}
}

func init() {
	fmt.Println("VectorManager tests initialized")
}


// TestPropertySemanticSearchOrdering is Property Test 4: Semantic Search Ordering
// For any semantic search query with top-k parameter:
// - Results SHALL be ordered by descending similarity score
// - Result count SHALL be at most k
// - Each result SHALL include valid session metadata
func TestPropertySemanticSearchOrdering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_search_property_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	// Pre-populate with embeddings
	numDocs := 50
	for i := int64(1); i <= int64(numDocs); i++ {
		embedding := createNormalizedEmbedding(EmbeddingDimensions)
		embedding[0] = float32(i) / float32(numDocs)
		normalizeEmbedding(embedding)
		err := vm.StoreEmbedding(i, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding %d: %v", i, err)
		}
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for topK values (1 to numDocs)
	genTopK := gen.IntRange(1, numDocs)

	// Generator for query embeddings
	genQueryEmbedding := gen.SliceOfN(EmbeddingDimensions, gen.Float32Range(-1.0, 1.0)).
		Map(func(v []float32) []float32 {
			normalizeEmbedding(v)
			return v
		})

	properties.Property("Semantic search ordering", prop.ForAll(
		func(topK int, queryEmbedding []float32) bool {
			results, err := vm.Search(queryEmbedding, topK)
			if err != nil {
				t.Logf("Search failed: %v", err)
				return false
			}

			// Property 1: Result count <= k
			if len(results) > topK {
				t.Logf("Too many results: got %d, expected at most %d", len(results), topK)
				return false
			}

			// Property 2: Results ordered by descending similarity
			for i := 1; i < len(results); i++ {
				if results[i].Score > results[i-1].Score {
					t.Logf("Results not ordered: score[%d]=%f > score[%d]=%f",
						i, results[i].Score, i-1, results[i-1].Score)
					return false
				}
			}

			// Property 3: Each result has valid metadata
			for _, r := range results {
				if r.SessionID <= 0 {
					t.Logf("Invalid session ID: %d", r.SessionID)
					return false
				}
				if r.ModelVersion == "" {
					t.Logf("Empty model version")
					return false
				}
				// Cosine similarity ranges from -1 to 1 for normalized vectors
				if r.Score < -1.01 || r.Score > 1.01 { // Allow small floating point error
					t.Logf("Invalid similarity score: %f", r.Score)
					return false
				}
			}

			return true
		},
		genTopK,
		genQueryEmbedding,
	))

	properties.TestingRun(t)
}

// TestPropertyEmbeddingUpdateOnTextChange is Property Test 3: Embedding Update on Text Change
// For any session where the text is modified, the embedding SHALL be different
func TestPropertyEmbeddingUpdateOnTextChange(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vector_manager_update_property_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultVectorManagerConfig(tempDir)
	vm, err := NewVectorManager(config)
	if err != nil {
		t.Fatalf("Failed to create VectorManager: %v", err)
	}
	defer vm.Close()

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genSessionID := gen.Int64Range(1, 1000000)

	// Generator for two different embeddings
	genEmbeddingPair := gen.SliceOfN(EmbeddingDimensions, gen.Float32Range(-1.0, 1.0)).
		FlatMap(func(v interface{}) gopter.Gen {
			first := v.([]float32)
			normalizeEmbedding(first)
			return gen.SliceOfN(EmbeddingDimensions, gen.Float32Range(-1.0, 1.0)).
				Map(func(second []float32) [2][]float32 {
					normalizeEmbedding(second)
					return [2][]float32{first, second}
				})
		}, reflect.TypeOf([2][]float32{}))

	properties.Property("Embedding update changes stored embedding", prop.ForAll(
		func(sessionID int64, embeddings [2][]float32) bool {
			firstEmbedding := embeddings[0]
			secondEmbedding := embeddings[1]

			// Store first embedding
			err := vm.StoreEmbedding(sessionID, firstEmbedding)
			if err != nil {
				t.Logf("Failed to store first embedding: %v", err)
				return false
			}

			// Get first embedding
			retrieved1, _, err := vm.GetEmbedding(sessionID)
			if err != nil {
				t.Logf("Failed to get first embedding: %v", err)
				return false
			}

			// Update with second embedding
			err = vm.UpdateEmbedding(sessionID, secondEmbedding)
			if err != nil {
				t.Logf("Failed to update embedding: %v", err)
				return false
			}

			// Get updated embedding
			retrieved2, _, err := vm.GetEmbedding(sessionID)
			if err != nil {
				t.Logf("Failed to get updated embedding: %v", err)
				return false
			}

			// Verify dimensions are correct
			if len(retrieved1) != EmbeddingDimensions || len(retrieved2) != EmbeddingDimensions {
				t.Logf("Wrong dimensions")
				return false
			}

			// Verify the embedding was actually updated (at least one value should differ)
			// Note: Due to floating point precision, we check if they're significantly different
			different := false
			for i := 0; i < len(retrieved1); i++ {
				diff := retrieved1[i] - retrieved2[i]
				if diff > 0.001 || diff < -0.001 {
					different = true
					break
				}
			}

			// If the input embeddings were different, the stored ones should be different
			inputDifferent := false
			for i := 0; i < len(firstEmbedding); i++ {
				diff := firstEmbedding[i] - secondEmbedding[i]
				if diff > 0.001 || diff < -0.001 {
					inputDifferent = true
					break
				}
			}

			if inputDifferent && !different {
				t.Logf("Embedding was not updated despite different input")
				return false
			}

			// Clean up
			vm.DeleteEmbedding(sessionID)

			return true
		},
		genSessionID,
		genEmbeddingPair,
	))

	properties.TestingRun(t)
}
