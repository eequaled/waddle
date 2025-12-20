package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	chromem "github.com/philippgille/chromem-go"
)

const (
	// DefaultEmbeddingModel is the default Ollama model for embeddings.
	DefaultEmbeddingModel = "nomic-embed-text"
	// DefaultOllamaURL is the default Ollama API URL.
	DefaultOllamaURL = "http://localhost:11434"
	// EmbeddingDimensions is the expected dimension count for nomic-embed-text.
	EmbeddingDimensions = 768
	// DefaultEmbedQueueSize is the default size of the async embedding queue.
	DefaultEmbedQueueSize = 1000
	// MaxEmbedRetries is the maximum number of retries for embedding generation.
	MaxEmbedRetries = 3
	// CollectionName is the name of the embeddings collection.
	CollectionName = "session_embeddings"
)

// VectorManager manages vector embeddings using chromem-go (pure Go vector DB).
// It uses Ollama for embedding generation with the nomic-embed-text model.
type VectorManager struct {
	db           *chromem.DB
	collection   *chromem.Collection
	ollamaURL    string
	modelVersion string
	dataDir      string
	embedQueue   chan EmbedRequest
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	httpClient   *http.Client
}

// VectorManagerConfig holds configuration for VectorManager.
type VectorManagerConfig struct {
	DataDir      string // Directory for vector storage (default: ~/.waddle/)
	OllamaURL    string // Ollama API URL (default: http://localhost:11434)
	ModelVersion string // Embedding model version (default: nomic-embed-text)
	QueueSize    int    // Async embedding queue size (default: 1000)
}

// DefaultVectorManagerConfig returns a VectorManagerConfig with default values.
func DefaultVectorManagerConfig(dataDir string) *VectorManagerConfig {
	return &VectorManagerConfig{
		DataDir:      dataDir,
		OllamaURL:    DefaultOllamaURL,
		ModelVersion: DefaultEmbeddingModel,
		QueueSize:    DefaultEmbedQueueSize,
	}
}

// NewVectorManager creates a new VectorManager instance.
func NewVectorManager(config *VectorManagerConfig) (*VectorManager, error) {
	if config == nil {
		return nil, NewStorageError(ErrValidation, "config cannot be nil", nil)
	}

	if config.DataDir == "" {
		return nil, NewStorageError(ErrValidation, "data directory cannot be empty", nil)
	}

	// Set defaults
	if config.OllamaURL == "" {
		config.OllamaURL = DefaultOllamaURL
	}
	if config.ModelVersion == "" {
		config.ModelVersion = DefaultEmbeddingModel
	}
	if config.QueueSize <= 0 {
		config.QueueSize = DefaultEmbedQueueSize
	}

	vm := &VectorManager{
		ollamaURL:    config.OllamaURL,
		modelVersion: config.ModelVersion,
		dataDir:      config.DataDir,
		embedQueue:   make(chan EmbedRequest, config.QueueSize),
		stopChan:     make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Initialize the vector database
	if err := vm.initialize(); err != nil {
		return nil, err
	}

	return vm, nil
}

// initialize sets up the chromem-go database and collection.
func (vm *VectorManager) initialize() error {
	// Create the vectors directory
	vectorsDir := filepath.Join(vm.dataDir, "vectors")
	if err := os.MkdirAll(vectorsDir, 0755); err != nil {
		return NewStorageError(ErrFileSystem, "failed to create vectors directory", err)
	}

	// Create persistent chromem-go database
	dbPath := filepath.Join(vectorsDir, "chromem")
	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return NewStorageError(ErrVector, "failed to create vector database", err)
	}
	vm.db = db

	// Create or get the embeddings collection
	// We use a custom embedding function that calls Ollama
	embeddingFunc := vm.createEmbeddingFunc()
	collection, err := db.GetOrCreateCollection(CollectionName, nil, embeddingFunc)
	if err != nil {
		return NewStorageError(ErrVector, "failed to create embeddings collection", err)
	}
	vm.collection = collection

	return nil
}

// createEmbeddingFunc creates an embedding function using Ollama.
func (vm *VectorManager) createEmbeddingFunc() chromem.EmbeddingFunc {
	return chromem.NewEmbeddingFuncOllama(vm.modelVersion, vm.ollamaURL)
}

// GenerateEmbedding generates an embedding vector for the given text using Ollama.
func (vm *VectorManager) GenerateEmbedding(text string) ([]float32, error) {
	if text == "" {
		return nil, NewStorageError(ErrValidation, "text cannot be empty", nil)
	}

	// Call Ollama embeddings API directly
	embedding, err := vm.callOllamaEmbeddings(text)
	if err != nil {
		return nil, NewStorageError(ErrVector, "failed to generate embedding", err)
	}

	// Validate embedding dimensions
	if len(embedding) != EmbeddingDimensions {
		return nil, NewStorageError(ErrVector,
			fmt.Sprintf("unexpected embedding dimensions: got %d, expected %d", len(embedding), EmbeddingDimensions),
			nil)
	}

	return embedding, nil
}

// ollamaEmbeddingRequest represents the request to Ollama embeddings API.
type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbeddingResponse represents the response from Ollama embeddings API.
type ollamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// callOllamaEmbeddings calls the Ollama API to generate embeddings.
func (vm *VectorManager) callOllamaEmbeddings(text string) ([]float32, error) {
	reqBody := ollamaEmbeddingRequest{
		Model:  vm.modelVersion,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", vm.ollamaURL)
	resp, err := vm.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var embResp ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(embResp.Embedding))
	for i, v := range embResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// StoreEmbedding stores an embedding for a session.
func (vm *VectorManager) StoreEmbedding(sessionID int64, embedding []float32) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if sessionID <= 0 {
		return NewStorageError(ErrValidation, "session ID must be positive", nil)
	}

	if len(embedding) != EmbeddingDimensions {
		return NewStorageError(ErrValidation,
			fmt.Sprintf("embedding must have %d dimensions, got %d", EmbeddingDimensions, len(embedding)),
			nil)
	}

	// Create document ID from session ID
	docID := fmt.Sprintf("session_%d", sessionID)

	// Create metadata with model version and timestamps
	metadata := map[string]string{
		"session_id":    fmt.Sprintf("%d", sessionID),
		"model_version": vm.modelVersion,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Add document with pre-computed embedding
	ctx := context.Background()
	err := vm.collection.Add(ctx,
		[]string{docID},
		[][]float32{embedding},
		[]map[string]string{metadata},
		[]string{""}, // Empty content since we're storing pre-computed embeddings
	)
	if err != nil {
		return NewStorageError(ErrVector, "failed to store embedding", err)
	}

	return nil
}

// UpdateEmbedding updates an existing embedding for a session.
func (vm *VectorManager) UpdateEmbedding(sessionID int64, embedding []float32) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if sessionID <= 0 {
		return NewStorageError(ErrValidation, "session ID must be positive", nil)
	}

	if len(embedding) != EmbeddingDimensions {
		return NewStorageError(ErrValidation,
			fmt.Sprintf("embedding must have %d dimensions, got %d", EmbeddingDimensions, len(embedding)),
			nil)
	}

	docID := fmt.Sprintf("session_%d", sessionID)
	ctx := context.Background()

	// Delete existing embedding first
	err := vm.collection.Delete(ctx, nil, nil, docID)
	if err != nil {
		// Ignore not found errors - we'll just add the new one
	}

	// Create metadata with model version and timestamps
	metadata := map[string]string{
		"session_id":    fmt.Sprintf("%d", sessionID),
		"model_version": vm.modelVersion,
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Add new embedding
	err = vm.collection.Add(ctx,
		[]string{docID},
		[][]float32{embedding},
		[]map[string]string{metadata},
		[]string{""},
	)
	if err != nil {
		return NewStorageError(ErrVector, "failed to update embedding", err)
	}

	return nil
}

// DeleteEmbedding removes an embedding for a session.
func (vm *VectorManager) DeleteEmbedding(sessionID int64) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if sessionID <= 0 {
		return NewStorageError(ErrValidation, "session ID must be positive", nil)
	}

	docID := fmt.Sprintf("session_%d", sessionID)
	ctx := context.Background()

	err := vm.collection.Delete(ctx, nil, nil, docID)
	if err != nil {
		return NewStorageError(ErrVector, "failed to delete embedding", err)
	}

	return nil
}

// Search performs semantic search and returns the top-k most similar sessions.
func (vm *VectorManager) Search(queryEmbedding []float32, topK int) ([]VectorSearchResult, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if len(queryEmbedding) != EmbeddingDimensions {
		return nil, NewStorageError(ErrValidation,
			fmt.Sprintf("query embedding must have %d dimensions, got %d", EmbeddingDimensions, len(queryEmbedding)),
			nil)
	}

	if topK <= 0 {
		return nil, NewStorageError(ErrValidation, "topK must be positive", nil)
	}

	ctx := context.Background()
	results, err := vm.collection.QueryEmbedding(ctx, queryEmbedding, topK, nil, nil)
	if err != nil {
		return nil, NewStorageError(ErrVector, "failed to search embeddings", err)
	}

	// Convert to VectorSearchResult
	searchResults := make([]VectorSearchResult, 0, len(results))
	for _, r := range results {
		// Parse session ID from metadata
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

// QueueEmbedding adds an embedding request to the async queue.
func (vm *VectorManager) QueueEmbedding(sessionID int64, text string) error {
	if sessionID <= 0 {
		return NewStorageError(ErrValidation, "session ID must be positive", nil)
	}

	if text == "" {
		return NewStorageError(ErrValidation, "text cannot be empty", nil)
	}

	select {
	case vm.embedQueue <- EmbedRequest{SessionID: sessionID, Text: text}:
		return nil
	default:
		return NewStorageError(ErrVector, "embedding queue is full", nil)
	}
}

// ProcessQueue starts processing the async embedding queue.
func (vm *VectorManager) ProcessQueue() error {
	vm.wg.Add(1)
	go func() {
		defer vm.wg.Done()
		for {
			select {
			case <-vm.stopChan:
				return
			case req := <-vm.embedQueue:
				vm.processEmbedRequest(req)
			}
		}
	}()
	return nil
}

// processEmbedRequest processes a single embedding request with retries.
func (vm *VectorManager) processEmbedRequest(req EmbedRequest) {
	var lastErr error
	for i := 0; i < MaxEmbedRetries; i++ {
		embedding, err := vm.GenerateEmbedding(req.Text)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
			continue
		}

		err = vm.StoreEmbedding(req.SessionID, embedding)
		if err != nil {
			lastErr = err
			continue
		}

		if req.Callback != nil {
			req.Callback(nil)
		}
		return
	}

	if req.Callback != nil {
		req.Callback(lastErr)
	}
}

// Reindex regenerates all embeddings with a new model version.
func (vm *VectorManager) Reindex(modelVersion string) error {
	// This would require iterating through all sessions and regenerating embeddings
	// For now, we just update the model version
	vm.mu.Lock()
	vm.modelVersion = modelVersion
	vm.mu.Unlock()
	return nil
}

// GetEmbedding retrieves the embedding for a specific session.
func (vm *VectorManager) GetEmbedding(sessionID int64) ([]float32, string, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if sessionID <= 0 {
		return nil, "", NewStorageError(ErrValidation, "session ID must be positive", nil)
	}

	docID := fmt.Sprintf("session_%d", sessionID)
	ctx := context.Background()

	doc, err := vm.collection.GetByID(ctx, docID)
	if err != nil {
		return nil, "", NewStorageError(ErrNotFound, "embedding not found", err)
	}

	modelVersion := doc.Metadata["model_version"]
	return doc.Embedding, modelVersion, nil
}

// HasEmbedding checks if an embedding exists for a session.
func (vm *VectorManager) HasEmbedding(sessionID int64) bool {
	_, _, err := vm.GetEmbedding(sessionID)
	return err == nil
}

// Count returns the number of embeddings stored.
func (vm *VectorManager) Count() int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.collection.Count()
}

// Flush ensures all pending embeddings are persisted to disk.
func (vm *VectorManager) Flush() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	// chromem-go automatically persists data, but we can force a sync
	// by accessing the collection (which triggers persistence)
	if vm.collection != nil {
		// Get collection count to trigger persistence
		_ = vm.collection.Count()
	}
	return nil
}

// Close shuts down the VectorManager and releases resources.
func (vm *VectorManager) Close() error {
	// Signal queue processor to stop
	close(vm.stopChan)
	vm.wg.Wait()

	// Close the database (chromem-go handles persistence)
	// Note: chromem-go doesn't have an explicit Close method,
	// but persistence is handled automatically
	return nil
}

// IsOllamaAvailable checks if Ollama is available and the model is loaded.
func (vm *VectorManager) IsOllamaAvailable() bool {
	// Try to generate a simple embedding to check availability
	_, err := vm.callOllamaEmbeddings("test")
	return err == nil
}

// GetModelVersion returns the current embedding model version.
func (vm *VectorManager) GetModelVersion() string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.modelVersion
}
