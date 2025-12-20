package storage

import (
	"bytes"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// TestEncryptionRoundTrip is a property-based test that verifies:
// For any plaintext string, encrypting then decrypting produces the original plaintext.
// **Property 1: Encryption Round-Trip**
// **Validates: Requirements 4.2, 4.3**
func TestEncryptionRoundTrip(t *testing.T) {
	// Initialize encryption manager for testing
	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	parameters := DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	// Property 1: Encryption Round-Trip for bytes
	// For any byte slice, Decrypt(Encrypt(x)) == x
	properties.Property("Decrypt(Encrypt(plaintext)) == plaintext for bytes", prop.ForAll(
		func(plaintext []byte) bool {
			ciphertext, err := em.Encrypt(plaintext)
			if err != nil {
				t.Logf("Encryption failed: %v", err)
				return false
			}

			decrypted, err := em.Decrypt(ciphertext)
			if err != nil {
				t.Logf("Decryption failed: %v", err)
				return false
			}

			return bytes.Equal(plaintext, decrypted)
		},
		GenPlaintext(),
	))

	// Property 1: Encryption Round-Trip for strings
	// For any string, DecryptString(EncryptString(x)) == x
	properties.Property("DecryptString(EncryptString(plaintext)) == plaintext for strings", prop.ForAll(
		func(plaintext string) bool {
			ciphertext, err := em.EncryptString(plaintext)
			if err != nil {
				t.Logf("String encryption failed: %v", err)
				return false
			}

			decrypted, err := em.DecryptString(ciphertext)
			if err != nil {
				t.Logf("String decryption failed: %v", err)
				return false
			}

			return plaintext == decrypted
		},
		GenUnicodeString(),
	))

	// Property: Empty input handling
	properties.Property("Empty input returns empty output", prop.ForAll(
		func(_ bool) bool {
			// Test empty byte slice
			ciphertext, err := em.Encrypt([]byte{})
			if err != nil {
				return false
			}
			decrypted, err := em.Decrypt(ciphertext)
			if err != nil {
				return false
			}
			if len(decrypted) != 0 {
				return false
			}

			// Test empty string
			encStr, err := em.EncryptString("")
			if err != nil {
				return false
			}
			decStr, err := em.DecryptString(encStr)
			if err != nil {
				return false
			}
			return decStr == ""
		},
		gopter.Gen(func(*gopter.GenParameters) *gopter.GenResult {
			return gopter.NewGenResult(true, gopter.NoShrinker)
		}),
	))

	// Property: Ciphertext is different from plaintext (for non-empty input)
	properties.Property("Ciphertext differs from plaintext for non-empty input", prop.ForAll(
		func(plaintext []byte) bool {
			if len(plaintext) == 0 {
				return true // Skip empty inputs
			}

			ciphertext, err := em.Encrypt(plaintext)
			if err != nil {
				return false
			}

			// Ciphertext should be longer (includes nonce and auth tag)
			if len(ciphertext) <= len(plaintext) {
				return false
			}

			// Ciphertext should not equal plaintext
			return !bytes.Equal(ciphertext, plaintext)
		},
		GenPlaintext(),
	))

	// Property: Each encryption produces different ciphertext (due to random nonce)
	properties.Property("Same plaintext produces different ciphertext each time", prop.ForAll(
		func(plaintext []byte) bool {
			if len(plaintext) == 0 {
				return true // Skip empty inputs
			}

			ciphertext1, err := em.Encrypt(plaintext)
			if err != nil {
				return false
			}

			ciphertext2, err := em.Encrypt(plaintext)
			if err != nil {
				return false
			}

			// Due to random nonce, ciphertexts should differ
			return !bytes.Equal(ciphertext1, ciphertext2)
		},
		GenNonEmptyPlaintext(),
	))

	properties.TestingRun(t)
}

// TestEncryptionEdgeCases tests specific edge cases for encryption.
func TestEncryptionEdgeCases(t *testing.T) {
	em := NewEncryptionManager()
	if err := em.InitializeKey(); err != nil {
		t.Fatalf("Failed to initialize encryption: %v", err)
	}

	t.Run("Invalid ciphertext returns error", func(t *testing.T) {
		// Too short ciphertext
		_, err := em.Decrypt([]byte{1, 2, 3})
		if err == nil {
			t.Error("Expected error for short ciphertext")
		}
	})

	t.Run("Corrupted ciphertext returns error", func(t *testing.T) {
		plaintext := []byte("test data")
		ciphertext, err := em.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		// Corrupt the ciphertext
		ciphertext[len(ciphertext)-1] ^= 0xFF

		_, err = em.Decrypt(ciphertext)
		if err == nil {
			t.Error("Expected error for corrupted ciphertext")
		}
	})

	t.Run("Invalid base64 returns error", func(t *testing.T) {
		_, err := em.DecryptString("not-valid-base64!!!")
		if err == nil {
			t.Error("Expected error for invalid base64")
		}
	})

	t.Run("Unicode strings round-trip correctly", func(t *testing.T) {
		testStrings := []string{
			"Hello, 世界!",
			"Привет мир",
			"🎉🎊🎈",
			"مرحبا بالعالم",
			"שלום עולם",
			"こんにちは世界",
		}

		for _, s := range testStrings {
			encrypted, err := em.EncryptString(s)
			if err != nil {
				t.Errorf("Failed to encrypt %q: %v", s, err)
				continue
			}

			decrypted, err := em.DecryptString(encrypted)
			if err != nil {
				t.Errorf("Failed to decrypt %q: %v", s, err)
				continue
			}

			if decrypted != s {
				t.Errorf("Round-trip failed for %q: got %q", s, decrypted)
			}
		}
	})

	t.Run("Large data round-trip correctly", func(t *testing.T) {
		// Test with 10KB of data
		largeData := make([]byte, 10*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		encrypted, err := em.Encrypt(largeData)
		if err != nil {
			t.Fatalf("Failed to encrypt large data: %v", err)
		}

		decrypted, err := em.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt large data: %v", err)
		}

		if !bytes.Equal(largeData, decrypted) {
			t.Error("Large data round-trip failed")
		}
	})
}

// GenPlaintext generates random byte slices for encryption testing.
func GenPlaintext() gopter.Gen {
	return gopter.Gen(func(params *gopter.GenParameters) *gopter.GenResult {
		size := params.Rng.Intn(10001) // 0 to 10000 bytes
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(params.Rng.Intn(256))
		}
		return gopter.NewGenResult(data, gopter.NoShrinker)
	})
}

// GenNonEmptyPlaintext generates non-empty byte slices for encryption testing.
func GenNonEmptyPlaintext() gopter.Gen {
	return gopter.Gen(func(params *gopter.GenParameters) *gopter.GenResult {
		size := params.Rng.Intn(10000) + 1 // 1 to 10000 bytes
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(params.Rng.Intn(256))
		}
		return gopter.NewGenResult(data, gopter.NoShrinker)
	})
}
