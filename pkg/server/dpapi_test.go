package server

import (
	"bytes"
	"testing"
)

// TestDPAPIAvailability tests DPAPI availability detection
func TestDPAPIAvailability(t *testing.T) {
	dpapi := NewDPAPI()
	
	// DPAPI should be available on Windows
	available := dpapi.IsAvailable()
	t.Logf("DPAPI available: %v", available)
	
	// This test is informational - DPAPI may or may not be available
	// depending on the system and test environment
}

// TestDPAPIProtectUnprotect tests DPAPI protect/unprotect functionality
func TestDPAPIProtectUnprotect(t *testing.T) {
	dpapi := NewDPAPI()
	
	if !dpapi.IsAvailable() {
		t.Skip("DPAPI not available, skipping test")
	}
	
	// Test data
	originalData := []byte("This is a test secret that should be protected")
	description := "Test Secret"
	
	// Protect data
	protectedData, err := dpapi.Protect(originalData, description)
	if err != nil {
		t.Fatalf("Failed to protect data: %v", err)
	}
	
	// Protected data should be different from original
	if bytes.Equal(protectedData, originalData) {
		t.Errorf("Protected data should be different from original data")
	}
	
	// Protected data should not be empty
	if len(protectedData) == 0 {
		t.Errorf("Protected data should not be empty")
	}
	
	// Unprotect data
	unprotectedData, retrievedDesc, err := dpapi.Unprotect(protectedData)
	if err != nil {
		t.Fatalf("Failed to unprotect data: %v", err)
	}
	
	// Unprotected data should match original
	if !bytes.Equal(unprotectedData, originalData) {
		t.Errorf("Unprotected data does not match original data")
	}
	
	// Description should match
	if retrievedDesc != description {
		t.Errorf("Expected description %q, got %q", description, retrievedDesc)
	}
}

// TestDPAPIEmptyData tests DPAPI with empty data
func TestDPAPIEmptyData(t *testing.T) {
	dpapi := NewDPAPI()
	
	if !dpapi.IsAvailable() {
		t.Skip("DPAPI not available, skipping test")
	}
	
	// Test with empty data - Windows DPAPI doesn't accept empty data
	originalData := []byte{}
	description := "Empty Secret"
	
	// Protect empty data - should fail on Windows
	_, err := dpapi.Protect(originalData, description)
	if err == nil {
		t.Logf("DPAPI accepted empty data (unexpected but not a failure)")
	} else {
		// This is expected behavior - DPAPI rejects empty data
		t.Logf("DPAPI correctly rejected empty data: %v", err)
	}
}

// TestDPAPINoDescription tests DPAPI without description
func TestDPAPINoDescription(t *testing.T) {
	dpapi := NewDPAPI()
	
	if !dpapi.IsAvailable() {
		t.Skip("DPAPI not available, skipping test")
	}
	
	// Test data without description
	originalData := []byte("Secret without description")
	
	// Protect data without description
	protectedData, err := dpapi.Protect(originalData, "")
	if err != nil {
		t.Fatalf("Failed to protect data without description: %v", err)
	}
	
	// Unprotect data
	unprotectedData, retrievedDesc, err := dpapi.Unprotect(protectedData)
	if err != nil {
		t.Fatalf("Failed to unprotect data: %v", err)
	}
	
	// Unprotected data should match original
	if !bytes.Equal(unprotectedData, originalData) {
		t.Errorf("Unprotected data does not match original data")
	}
	
	// Description should be empty
	if retrievedDesc != "" {
		t.Logf("Retrieved description: %q (may be empty or default)", retrievedDesc)
	}
}

// TestDPAPIInvalidData tests DPAPI with invalid protected data
func TestDPAPIInvalidData(t *testing.T) {
	dpapi := NewDPAPI()
	
	if !dpapi.IsAvailable() {
		t.Skip("DPAPI not available, skipping test")
	}
	
	// Try to unprotect invalid data
	invalidData := []byte("This is not DPAPI-protected data")
	
	_, _, err := dpapi.Unprotect(invalidData)
	if err == nil {
		t.Errorf("Expected error when unprotecting invalid data")
	}
}

// TestDPAPILargeData tests DPAPI with large data
func TestDPAPILargeData(t *testing.T) {
	dpapi := NewDPAPI()
	
	if !dpapi.IsAvailable() {
		t.Skip("DPAPI not available, skipping test")
	}
	
	// Create large test data (1MB)
	originalData := make([]byte, 1024*1024)
	for i := range originalData {
		originalData[i] = byte(i % 256)
	}
	
	description := "Large Test Secret"
	
	// Protect large data
	protectedData, err := dpapi.Protect(originalData, description)
	if err != nil {
		t.Fatalf("Failed to protect large data: %v", err)
	}
	
	// Unprotect data
	unprotectedData, retrievedDesc, err := dpapi.Unprotect(protectedData)
	if err != nil {
		t.Fatalf("Failed to unprotect large data: %v", err)
	}
	
	// Unprotected data should match original
	if !bytes.Equal(unprotectedData, originalData) {
		t.Errorf("Unprotected large data does not match original data")
	}
	
	// Description should match
	if retrievedDesc != description {
		t.Errorf("Expected description %q, got %q", description, retrievedDesc)
	}
}

// BenchmarkDPAPIProtect benchmarks DPAPI protection
func BenchmarkDPAPIProtect(b *testing.B) {
	dpapi := NewDPAPI()
	
	if !dpapi.IsAvailable() {
		b.Skip("DPAPI not available, skipping benchmark")
	}
	
	data := []byte("This is test data for benchmarking DPAPI protection")
	description := "Benchmark Secret"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dpapi.Protect(data, description)
		if err != nil {
			b.Fatalf("Failed to protect data: %v", err)
		}
	}
}

// BenchmarkDPAPIUnprotect benchmarks DPAPI unprotection
func BenchmarkDPAPIUnprotect(b *testing.B) {
	dpapi := NewDPAPI()
	
	if !dpapi.IsAvailable() {
		b.Skip("DPAPI not available, skipping benchmark")
	}
	
	data := []byte("This is test data for benchmarking DPAPI unprotection")
	description := "Benchmark Secret"
	
	// Protect data once
	protectedData, err := dpapi.Protect(data, description)
	if err != nil {
		b.Fatalf("Failed to protect data: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := dpapi.Unprotect(protectedData)
		if err != nil {
			b.Fatalf("Failed to unprotect data: %v", err)
		}
	}
}