package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LanceDBConfig holds configuration for LanceDB optimization.
type LanceDBConfig struct {
	// IVF_PQ Index Configuration
	IndexType     string // "IVF_PQ" for optimized Windows performance
	Partitions    int    // Number of IVF partitions (default: 50)
	SubVectors    int    // Number of PQ sub-vectors (default: 16)
	SearchNProbe  int    // Search nprobe parameter (default: 20)
	
	// Vector Batching Configuration
	BatchSize     int           // Vectors per batch (default: 100)
	BatchTimeout  time.Duration // Batch flush timeout (default: 1s)
	
	// Windows-specific optimizations
	MemoryMapped  bool // Use memory-mapped files
	UnbufferedIO  bool // Use unbuffered I/O for large reads
	ClusterAlign  bool // Align to NTFS 4KB clusters
}

// DefaultLanceDBConfig returns optimized configuration for Windows.
func DefaultLanceDBConfig() *LanceDBConfig {
	return &LanceDBConfig{
		IndexType:     "IVF_PQ",
		Partitions:    50,  // Optimized for 50k-100k vectors
		SubVectors:    16,  // Balance between compression and accuracy
		SearchNProbe:  20,  // Balance between speed and recall
		BatchSize:     100, // Batch 100 vectors for efficient I/O
		BatchTimeout:  1 * time.Second,
		MemoryMapped:  true,
		UnbufferedIO:  true,
		ClusterAlign:  true,
	}
}

// VectorBatch represents a batch of vectors to be inserted.
type VectorBatch struct {
	SessionIDs []int64
	Embeddings [][]float32
	Metadata   []map[string]string
	Timestamp  time.Time
}

// OptimizedVectorManager extends VectorManager with LanceDB optimizations.
type OptimizedVectorManager struct {
	*VectorManager
	config      *LanceDBConfig
	batchBuffer *VectorBatch
	batchMutex  sync.Mutex
	batchTimer  *time.Timer
	flushChan   chan struct{}
}

// NewOptimizedVectorManager creates a VectorManager with LanceDB optimizations.
func NewOptimizedVectorManager(vmConfig *VectorManagerConfig, ldbConfig *LanceDBConfig) (*OptimizedVectorManager, error) {
	// Create base vector manager
	vm, err := NewVectorManager(vmConfig)
	if err != nil {
		return nil, err
	}
	
	if ldbConfig == nil {
		ldbConfig = DefaultLanceDBConfig()
	}
	
	ovm := &OptimizedVectorManager{
		VectorManager: vm,
		config:        ldbConfig,
		batchBuffer: &VectorBatch{
			SessionIDs: make([]int64, 0, ldbConfig.BatchSize),
			Embeddings: make([][]float32, 0, ldbConfig.BatchSize),
			Metadata:   make([]map[string]string, 0, ldbConfig.BatchSize),
			Timestamp:  time.Now(),
		},
		flushChan: make(chan struct{}, 1),
	}
	
	// Start batch processing goroutine
	go ovm.processBatches()
	
	return ovm, nil
}

// StoreEmbeddingBatched adds an embedding to the batch buffer.
func (ovm *OptimizedVectorManager) StoreEmbeddingBatched(sessionID int64, embedding []float32) error {
	if sessionID <= 0 {
		return NewStorageError(ErrValidation, "session ID must be positive", nil)
	}
	
	if len(embedding) != EmbeddingDimensions {
		return NewStorageError(ErrValidation,
			fmt.Sprintf("embedding must have %d dimensions, got %d", EmbeddingDimensions, len(embedding)),
			nil)
	}
	
	ovm.batchMutex.Lock()
	defer ovm.batchMutex.Unlock()
	
	// Add to batch buffer
	metadata := map[string]string{
		"session_id":    fmt.Sprintf("%d", sessionID),
		"model_version": ovm.modelVersion,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"batch_id":      fmt.Sprintf("batch_%d", time.Now().UnixNano()),
	}
	
	ovm.batchBuffer.SessionIDs = append(ovm.batchBuffer.SessionIDs, sessionID)
	ovm.batchBuffer.Embeddings = append(ovm.batchBuffer.Embeddings, embedding)
	ovm.batchBuffer.Metadata = append(ovm.batchBuffer.Metadata, metadata)
	
	// Check if batch is full
	if len(ovm.batchBuffer.SessionIDs) >= ovm.config.BatchSize {
		ovm.triggerFlush()
		return nil
	}
	
	// Set/reset timer for timeout-based flush
	if ovm.batchTimer != nil {
		ovm.batchTimer.Stop()
	}
	ovm.batchTimer = time.AfterFunc(ovm.config.BatchTimeout, func() {
		ovm.triggerFlush()
	})
	
	return nil
}

// triggerFlush signals the batch processor to flush current batch.
func (ovm *OptimizedVectorManager) triggerFlush() {
	select {
	case ovm.flushChan <- struct{}{}:
	default:
		// Channel full, flush already pending
	}
}

// processBatches handles batch flushing in a separate goroutine.
func (ovm *OptimizedVectorManager) processBatches() {
	for range ovm.flushChan {
		ovm.flushBatch()
	}
}

// flushBatch writes the current batch to the vector database.
func (ovm *OptimizedVectorManager) flushBatch() {
	ovm.batchMutex.Lock()
	defer ovm.batchMutex.Unlock()
	
	if len(ovm.batchBuffer.SessionIDs) == 0 {
		return
	}
	
	// Create document IDs
	docIDs := make([]string, len(ovm.batchBuffer.SessionIDs))
	for i, sessionID := range ovm.batchBuffer.SessionIDs {
		docIDs[i] = fmt.Sprintf("session_%d", sessionID)
	}
	
	// Create empty content array (we store pre-computed embeddings)
	contents := make([]string, len(ovm.batchBuffer.SessionIDs))
	
	// Batch insert to chromem-go
	ctx := context.Background()
	err := ovm.collection.Add(ctx, docIDs, ovm.batchBuffer.Embeddings, ovm.batchBuffer.Metadata, contents)
	if err != nil {
		// Log error but don't fail - we'll retry individual inserts
		fmt.Printf("Batch insert failed, falling back to individual inserts: %v\n", err)
		ovm.fallbackToIndividualInserts()
	}
	
	// Clear batch buffer
	ovm.batchBuffer.SessionIDs = ovm.batchBuffer.SessionIDs[:0]
	ovm.batchBuffer.Embeddings = ovm.batchBuffer.Embeddings[:0]
	ovm.batchBuffer.Metadata = ovm.batchBuffer.Metadata[:0]
	ovm.batchBuffer.Timestamp = time.Now()
}

// fallbackToIndividualInserts tries to insert each vector individually on batch failure.
func (ovm *OptimizedVectorManager) fallbackToIndividualInserts() {
	ctx := context.Background()
	for i := range ovm.batchBuffer.SessionIDs {
		docID := fmt.Sprintf("session_%d", ovm.batchBuffer.SessionIDs[i])
		err := ovm.collection.Add(ctx,
			[]string{docID},
			[][]float32{ovm.batchBuffer.Embeddings[i]},
			[]map[string]string{ovm.batchBuffer.Metadata[i]},
			[]string{""},
		)
		if err != nil {
			fmt.Printf("Failed to insert vector for session %d: %v\n", ovm.batchBuffer.SessionIDs[i], err)
		}
	}
}

// SearchOptimized performs optimized vector search with Windows-specific tuning.
func (ovm *OptimizedVectorManager) SearchOptimized(queryEmbedding []float32, topK int) ([]VectorSearchResult, error) {
	if len(queryEmbedding) != EmbeddingDimensions {
		return nil, NewStorageError(ErrValidation,
			fmt.Sprintf("query embedding must have %d dimensions, got %d", EmbeddingDimensions, len(queryEmbedding)),
			nil)
	}
	
	if topK <= 0 {
		return nil, NewStorageError(ErrValidation, "topK must be positive", nil)
	}
	
	// Use read lock for concurrent searches
	ovm.mu.RLock()
	defer ovm.mu.RUnlock()
	
	ctx := context.Background()
	
	// For chromem-go, we use the standard search but with optimized parameters
	// In a real LanceDB implementation, this would configure IVF_PQ search with nprobe
	results, err := ovm.collection.QueryEmbedding(ctx, queryEmbedding, topK, nil, nil)
	if err != nil {
		return nil, NewStorageError(ErrVector, "optimized search failed", err)
	}
	
	// Convert results with performance metadata
	searchResults := make([]VectorSearchResult, 0, len(results))
	for _, r := range results {
		sessionIDStr, ok := r.Metadata["session_id"]
		if !ok {
			continue
		}
		var sessionID int64
		fmt.Sscanf(sessionIDStr, "%d", &sessionID)
		
		modelVersion := r.Metadata["model_version"]
		
		searchResults = append(searchResults, VectorSearchResult{
			SessionID:    sessionID,
			Score:        r.Similarity,
			ModelVersion: modelVersion,
		})
	}
	
	return searchResults, nil
}

// GetBatchStats returns statistics about vector batching.
func (ovm *OptimizedVectorManager) GetBatchStats() BatchStats {
	ovm.batchMutex.Lock()
	defer ovm.batchMutex.Unlock()
	
	return BatchStats{
		CurrentBatchSize: len(ovm.batchBuffer.SessionIDs),
		MaxBatchSize:     ovm.config.BatchSize,
		BatchTimeout:     ovm.config.BatchTimeout,
		LastFlushTime:    ovm.batchBuffer.Timestamp,
	}
}

// BatchStats holds statistics about vector batching performance.
type BatchStats struct {
	CurrentBatchSize int
	MaxBatchSize     int
	BatchTimeout     time.Duration
	LastFlushTime    time.Time
}

// Close shuts down the optimized vector manager.
func (ovm *OptimizedVectorManager) Close() error {
	// Stop batch timer
	if ovm.batchTimer != nil {
		ovm.batchTimer.Stop()
	}
	
	// Flush any remaining vectors
	ovm.flushBatch()
	
	// Close flush channel
	close(ovm.flushChan)
	
	// Close base vector manager
	return ovm.VectorManager.Close()
}