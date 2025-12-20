package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSecureServerCreation tests basic server creation and cleanup
func TestSecureServerCreation(t *testing.T) {
	server, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create secure server: %v", err)
	}
	defer server.Stop()

	// Verify server is not running initially
	if server.IsRunning() {
		t.Errorf("Server should not be running after creation")
	}

	// Verify pipe name is set
	if server.GetPipeName() != DefaultPipeName {
		t.Errorf("Expected pipe name %s, got %s", DefaultPipeName, server.GetPipeName())
	}

	// Verify API secret is generated
	secret := server.GetAPISecret()
	if secret == "" {
		t.Errorf("API secret should not be empty")
	}

	// Verify secret is base64 encoded
	_, err = base64.StdEncoding.DecodeString(secret)
	if err != nil {
		t.Errorf("API secret should be valid base64: %v", err)
	}
}

// TestSecureServerCustomPipeName tests custom pipe name
func TestSecureServerCustomPipeName(t *testing.T) {
	customPipe := `\\.\pipe\waddle_test`
	server, err := NewSecureServer(customPipe)
	if err != nil {
		t.Fatalf("Failed to create secure server with custom pipe: %v", err)
	}
	defer server.Stop()

	if server.GetPipeName() != customPipe {
		t.Errorf("Expected pipe name %s, got %s", customPipe, server.GetPipeName())
	}
}

// TestAPISecretGeneration tests API secret generation and persistence
func TestAPISecretGeneration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "waddle_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
			os.Setenv("USERPROFILE", originalHome)
		}
	}()
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	// Create first server
	server1, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create first server: %v", err)
	}
	secret1 := server1.GetAPISecret()
	server1.Stop()

	// Create second server - should load same secret
	server2, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create second server: %v", err)
	}
	secret2 := server2.GetAPISecret()
	server2.Stop()

	// Secrets should be the same
	if secret1 != secret2 {
		t.Errorf("API secrets should be the same across server instances")
	}

	// Verify secret file exists
	secretPath := filepath.Join(tempDir, ".waddle", SecretFileName)
	if _, err := os.Stat(secretPath); os.IsNotExist(err) {
		t.Errorf("Secret file should exist at %s", secretPath)
	}
}

// TestSecureServerStartStop tests server start/stop functionality
func TestSecureServerStartStop(t *testing.T) {
	server, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create secure server: %v", err)
	}
	defer server.Stop()

	// Test start
	err = server.Start()
	if err != nil {
		t.Errorf("Failed to start server: %v", err)
	}

	// Verify server is running
	if !server.IsRunning() {
		t.Errorf("Server should be running after start")
	}

	// Test double start
	err = server.Start()
	if err == nil {
		t.Errorf("Double start should return error")
	}

	// Test stop
	err = server.Stop()
	if err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Verify server is stopped
	if server.IsRunning() {
		t.Errorf("Server should be stopped after stop")
	}
}

// TestAPIRequestResponse tests API request/response structures
func TestAPIRequestResponse(t *testing.T) {
	// Test APIRequest
	req := APIRequest{
		Method:    "GET",
		Path:      "/test",
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
	}

	if req.Method != "GET" {
		t.Errorf("Expected method GET, got %s", req.Method)
	}

	// Test APIResponse
	resp := APIResponse{
		Status:    200,
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      map[string]interface{}{"result": "success"},
		Timestamp: time.Now(),
	}

	if resp.Status != 200 {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}
}

// TestRouteRequest tests request routing
func TestRouteRequest(t *testing.T) {
	server, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create secure server: %v", err)
	}
	defer server.Stop()

	tests := []struct {
		path           string
		method         string
		expectedStatus int
	}{
		{"/health", "GET", http.StatusOK},
		{"/stats", "GET", http.StatusOK},
		{"/capture", "GET", http.StatusOK},
		{"/capture", "POST", http.StatusAccepted},
		{"/capture", "DELETE", http.StatusMethodNotAllowed},
		{"/nonexistent", "GET", http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s_%s", test.method, test.path), func(t *testing.T) {
			req := &APIRequest{
				Method:    test.method,
				Path:      test.path,
				Headers:   make(map[string]string),
				Body:      make(map[string]interface{}),
				Timestamp: time.Now(),
			}

			resp := server.routeRequest(req)

			if resp.Status != test.expectedStatus {
				t.Errorf("Expected status %d, got %d", test.expectedStatus, resp.Status)
			}

			if resp.Timestamp.IsZero() {
				t.Errorf("Response timestamp should not be zero")
			}
		})
	}
}

// TestHealthEndpoint tests the health endpoint specifically
func TestHealthEndpoint(t *testing.T) {
	server, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create secure server: %v", err)
	}
	defer server.Stop()

	req := &APIRequest{
		Method:    "GET",
		Path:      "/health",
		Headers:   make(map[string]string),
		Body:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	resp := server.handleHealth(req)

	if resp.Status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}

	if status, exists := resp.Body["status"]; !exists || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", status)
	}

	if version, exists := resp.Body["version"]; !exists || version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", version)
	}
}

// TestStatsEndpoint tests the stats endpoint
func TestStatsEndpoint(t *testing.T) {
	server, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create secure server: %v", err)
	}
	defer server.Stop()

	req := &APIRequest{
		Method:    "GET",
		Path:      "/stats",
		Headers:   make(map[string]string),
		Body:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	resp := server.handleStats(req)

	if resp.Status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Status)
	}

	if pipeName, exists := resp.Body["pipe_name"]; !exists || pipeName != server.GetPipeName() {
		t.Errorf("Expected pipe_name %s, got %v", server.GetPipeName(), pipeName)
	}

	if running, exists := resp.Body["server_running"]; !exists {
		t.Errorf("Expected server_running field to exist, got %v", running)
	}
}

// TestJSONSerialization tests JSON serialization of request/response
func TestJSONSerialization(t *testing.T) {
	// Test APIRequest serialization
	req := APIRequest{
		Method:    "POST",
		Path:      "/test",
		Headers:   map[string]string{"Authorization": "Bearer token"},
		Body:      map[string]interface{}{"data": "test"},
		Timestamp: time.Now(),
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Errorf("Failed to marshal APIRequest: %v", err)
	}

	var reqParsed APIRequest
	err = json.Unmarshal(reqJSON, &reqParsed)
	if err != nil {
		t.Errorf("Failed to unmarshal APIRequest: %v", err)
	}

	if reqParsed.Method != req.Method {
		t.Errorf("Method mismatch after JSON round-trip")
	}

	// Test APIResponse serialization
	resp := APIResponse{
		Status:    200,
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      map[string]interface{}{"result": "success"},
		Timestamp: time.Now(),
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal APIResponse: %v", err)
	}

	var respParsed APIResponse
	err = json.Unmarshal(respJSON, &respParsed)
	if err != nil {
		t.Errorf("Failed to unmarshal APIResponse: %v", err)
	}

	if respParsed.Status != resp.Status {
		t.Errorf("Status mismatch after JSON round-trip")
	}
}

// TestSecretLength tests that generated secrets have correct length
func TestSecretLength(t *testing.T) {
	server, err := NewSecureServer("")
	if err != nil {
		t.Fatalf("Failed to create secure server: %v", err)
	}
	defer server.Stop()

	secret := server.GetAPISecret()
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		t.Fatalf("Failed to decode secret: %v", err)
	}

	if len(decoded) != SecretLength {
		t.Errorf("Expected secret length %d, got %d", SecretLength, len(decoded))
	}
}

// BenchmarkSecretGeneration benchmarks API secret generation
func BenchmarkSecretGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		server, err := NewSecureServer("")
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}
		server.Stop()
	}
}

// BenchmarkRequestRouting benchmarks request routing
func BenchmarkRequestRouting(b *testing.B) {
	server, err := NewSecureServer("")
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	defer server.Stop()

	req := &APIRequest{
		Method:    "GET",
		Path:      "/health",
		Headers:   make(map[string]string),
		Body:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.routeRequest(req)
	}
}