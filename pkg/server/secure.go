package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
)

// SecureServer provides a secure named pipe API server
type SecureServer struct {
	pipeName     string
	listener     net.Listener
	server       *http.Server
	apiSecret    []byte
	secretPath   string
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
}

// APIRequest represents an incoming API request
type APIRequest struct {
	Method    string                 `json:"method"`
	Path      string                 `json:"path"`
	Headers   map[string]string      `json:"headers"`
	Body      map[string]interface{} `json:"body"`
	Timestamp time.Time              `json:"timestamp"`
}

// APIResponse represents an API response
type APIResponse struct {
	Status    int                    `json:"status"`
	Headers   map[string]string      `json:"headers"`
	Body      map[string]interface{} `json:"body"`
	Timestamp time.Time              `json:"timestamp"`
	Error     string                 `json:"error,omitempty"`
}

const (
	// DefaultPipeName is the default named pipe path
	DefaultPipeName = `\\.\pipe\waddle`
	
	// SecretFileName is the name of the API secret file
	SecretFileName = "api_secret.dat"
	
	// SecretLength is the length of the API secret in bytes
	SecretLength = 32 // 256 bits
	
	// AuthHeaderName is the name of the authentication header
	AuthHeaderName = "X-Waddle-Auth"
)

// NewSecureServer creates a new secure API server
func NewSecureServer(pipeName string) (*SecureServer, error) {
	if pipeName == "" {
		pipeName = DefaultPipeName
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	s := &SecureServer{
		pipeName: pipeName,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// Initialize API secret
	err := s.initializeAPISecret()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize API secret: %w", err)
	}
	
	return s, nil
}

// initializeAPISecret initializes or loads the API secret
func (s *SecureServer) initializeAPISecret() error {
	// Determine secret file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	waddleDir := filepath.Join(homeDir, ".waddle")
	err = os.MkdirAll(waddleDir, 0700)
	if err != nil {
		return fmt.Errorf("failed to create waddle directory: %w", err)
	}
	
	s.secretPath = filepath.Join(waddleDir, SecretFileName)
	
	// Try to load existing secret
	if _, err := os.Stat(s.secretPath); err == nil {
		return s.loadAPISecret()
	}
	
	// Generate new secret
	return s.generateAPISecret()
}

// generateAPISecret generates a new API secret and saves it
func (s *SecureServer) generateAPISecret() error {
	// Generate random secret
	secret := make([]byte, SecretLength)
	_, err := rand.Read(secret)
	if err != nil {
		return fmt.Errorf("failed to generate random secret: %w", err)
	}
	
	s.apiSecret = secret
	
	// Save secret to file (in real implementation, this would use DPAPI)
	err = s.saveAPISecret()
	if err != nil {
		return fmt.Errorf("failed to save API secret: %w", err)
	}
	
	// Log success (in real implementation, would use proper logging)
	fmt.Println("Generated new API secret")
	return nil
}

// saveAPISecret saves the API secret to disk (DPAPI encrypted)
func (s *SecureServer) saveAPISecret() error {
	dpapi := NewDPAPI()
	
	// Check if DPAPI is available
	if !dpapi.IsAvailable() {
		// Fallback to base64 encoding if DPAPI unavailable
		encoded := base64.StdEncoding.EncodeToString(s.apiSecret)
		err := os.WriteFile(s.secretPath, []byte(encoded), 0600)
		if err != nil {
			return fmt.Errorf("failed to write secret file: %w", err)
		}
		fmt.Println("Warning: DPAPI unavailable, using base64 encoding")
		return nil
	}
	
	// Encrypt secret using DPAPI
	protectedData, err := dpapi.Protect(s.apiSecret, "Waddle API Secret")
	if err != nil {
		return fmt.Errorf("failed to protect secret with DPAPI: %w", err)
	}
	
	// Save encrypted secret to file
	err = os.WriteFile(s.secretPath, protectedData, 0600)
	if err != nil {
		return fmt.Errorf("failed to write protected secret file: %w", err)
	}
	
	return nil
}

// loadAPISecret loads the API secret from disk
func (s *SecureServer) loadAPISecret() error {
	// Read secret file
	data, err := os.ReadFile(s.secretPath)
	if err != nil {
		return fmt.Errorf("failed to read secret file: %w", err)
	}
	
	dpapi := NewDPAPI()
	
	// Check if DPAPI is available and try to decrypt
	if dpapi.IsAvailable() {
		// Try to decrypt using DPAPI first
		secret, description, err := dpapi.Unprotect(data)
		if err == nil {
			if len(secret) != SecretLength {
				return fmt.Errorf("invalid secret length: expected %d, got %d", SecretLength, len(secret))
			}
			s.apiSecret = secret
			fmt.Printf("Loaded DPAPI-protected API secret: %s\n", description)
			return nil
		}
		fmt.Printf("DPAPI decryption failed, trying base64 fallback: %v\n", err)
	}
	
	// Fallback to base64 decoding (for backwards compatibility)
	secret, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("failed to decode secret: %w", err)
	}
	
	if len(secret) != SecretLength {
		return fmt.Errorf("invalid secret length: expected %d, got %d", SecretLength, len(secret))
	}
	
	s.apiSecret = secret
	fmt.Println("Loaded base64-encoded API secret")
	return nil
}

// Start starts the secure server
func (s *SecureServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.running {
		return fmt.Errorf("server already running")
	}
	
	// Create named pipe listener with security descriptor
	config := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;AU)", // Allow authenticated users
		MessageMode:        false,              // Byte mode
		InputBufferSize:    65536,             // 64KB input buffer
		OutputBufferSize:   65536,             // 64KB output buffer
	}
	
	listener, err := winio.ListenPipe(s.pipeName, config)
	if err != nil {
		return fmt.Errorf("failed to create named pipe listener: %w", err)
	}
	
	s.listener = listener
	
	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)
	
	s.server = &http.Server{
		Handler:      s.authMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	// Start server in background
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := s.server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()
	
	s.running = true
	fmt.Printf("Secure server started on %s\n", s.pipeName)
	return nil
}

// authMiddleware validates API secret authentication
func (s *SecureServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get auth header
		authHeader := r.Header.Get(AuthHeaderName)
		if authHeader == "" {
			s.sendErrorResponse(w, http.StatusUnauthorized, "Missing authentication header")
			return
		}
		
		// Decode provided secret
		providedSecret, err := base64.StdEncoding.DecodeString(authHeader)
		if err != nil {
			s.sendErrorResponse(w, http.StatusUnauthorized, "Invalid authentication header format")
			return
		}
		
		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare(providedSecret, s.apiSecret) != 1 {
			s.sendErrorResponse(w, http.StatusUnauthorized, "Invalid authentication")
			return
		}
		
		// Authentication successful
		next.ServeHTTP(w, r)
	})
}

// handleRequest handles incoming API requests
func (s *SecureServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var body map[string]interface{}
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			json.Unmarshal(bodyBytes, &body)
		}
	}
	
	request := APIRequest{
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   make(map[string]string),
		Body:      body,
		Timestamp: time.Now(),
	}
	
	// Copy headers
	for name, values := range r.Header {
		if len(values) > 0 {
			request.Headers[name] = values[0]
		}
	}
	
	// Route request
	response := s.routeRequest(&request)
	
	// Send response
	s.sendJSONResponse(w, response)
}

// routeRequest routes API requests to appropriate handlers
func (s *SecureServer) routeRequest(req *APIRequest) *APIResponse {
	switch req.Path {
	case "/health":
		return s.handleHealth(req)
	case "/stats":
		return s.handleStats(req)
	case "/capture":
		return s.handleCapture(req)
	default:
		return &APIResponse{
			Status:    http.StatusNotFound,
			Headers:   make(map[string]string),
			Body:      map[string]interface{}{"error": "endpoint not found"},
			Timestamp: time.Now(),
			Error:     "Not Found",
		}
	}
}

// handleHealth handles health check requests
func (s *SecureServer) handleHealth(req *APIRequest) *APIResponse {
	return &APIResponse{
		Status:  http.StatusOK,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body: map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "1.0.0",
		},
		Timestamp: time.Now(),
	}
}

// handleStats handles statistics requests
func (s *SecureServer) handleStats(req *APIRequest) *APIResponse {
	return &APIResponse{
		Status:  http.StatusOK,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body: map[string]interface{}{
			"server_running": s.IsRunning(),
			"pipe_name":      s.pipeName,
			"uptime_seconds": time.Since(time.Now()).Seconds(), // Placeholder
		},
		Timestamp: time.Now(),
	}
}

// handleCapture handles capture-related requests
func (s *SecureServer) handleCapture(req *APIRequest) *APIResponse {
	switch req.Method {
	case "GET":
		return &APIResponse{
			Status:  http.StatusOK,
			Headers: map[string]string{"Content-Type": "application/json"},
			Body: map[string]interface{}{
				"capture_status": "active",
				"message":        "Capture system is running",
			},
			Timestamp: time.Now(),
		}
	case "POST":
		// Handle capture configuration or manual capture requests
		return &APIResponse{
			Status:  http.StatusAccepted,
			Headers: map[string]string{"Content-Type": "application/json"},
			Body: map[string]interface{}{
				"message": "Capture request accepted",
			},
			Timestamp: time.Now(),
		}
	default:
		return &APIResponse{
			Status:    http.StatusMethodNotAllowed,
			Headers:   make(map[string]string),
			Body:      map[string]interface{}{"error": "method not allowed"},
			Timestamp: time.Now(),
			Error:     "Method Not Allowed",
		}
	}
}

// sendJSONResponse sends a JSON response
func (s *SecureServer) sendJSONResponse(w http.ResponseWriter, response *APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	
	// Set custom headers
	for name, value := range response.Headers {
		w.Header().Set(name, value)
	}
	
	w.WriteHeader(response.Status)
	
	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse sends an error response
func (s *SecureServer) sendErrorResponse(w http.ResponseWriter, status int, message string) {
	response := &APIResponse{
		Status:    status,
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      map[string]interface{}{"error": message},
		Timestamp: time.Now(),
		Error:     message,
	}
	
	s.sendJSONResponse(w, response)
}

// GetAPISecret returns the base64-encoded API secret for client use
func (s *SecureServer) GetAPISecret() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return base64.StdEncoding.EncodeToString(s.apiSecret)
}

// IsRunning returns true if the server is running
func (s *SecureServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetPipeName returns the named pipe path
func (s *SecureServer) GetPipeName() string {
	return s.pipeName
}

// Stop stops the secure server
func (s *SecureServer) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()
	
	// Cancel context
	s.cancel()
	
	// Shutdown HTTP server
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
	
	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}
	
	// Wait for goroutines to finish
	s.wg.Wait()
	
	fmt.Println("Secure server stopped")
	return nil
}