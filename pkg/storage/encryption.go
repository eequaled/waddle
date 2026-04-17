package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/argon2"

	"waddle/pkg/infra/vault"
)

const (
	// KeySize is the size of the AES-256 key in bytes.
	KeySize = 32
	// NonceSize is the size of the GCM nonce in bytes.
	NonceSize = 12
	// SaltSize is the size of the salt for key derivation.
	SaltSize = 16
	// VaultKeyName is the name used to store the master key in the vault.
	VaultKeyName = "master_key"
)

// Argon2 parameters for key derivation.
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
)

// EncryptionManager handles encryption/decryption using AES-256-GCM.
// Key material is stored via a vault.Vault (DPAPI on Windows).
type EncryptionManager struct {
	key   []byte
	salt  []byte
	aead  cipher.AEAD
	vault vault.Vault
	mutex sync.RWMutex
}

// NewEncryptionManager creates a new EncryptionManager backed by the given data directory.
func NewEncryptionManager(dataDir string) *EncryptionManager {
	return &EncryptionManager{
		vault: vault.New(filepath.Join(dataDir, "vault")),
	}
}

// InitializeKey loads or generates the encryption key.
func (em *EncryptionManager) InitializeKey() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Try to load existing key from vault
	combined, err := em.vault.Load(VaultKeyName)
	if err != nil {
		// Key doesn't exist — generate a new one
		combined, err = em.generateAndStoreKey()
		if err != nil {
			return NewStorageError(ErrEncryption, "failed to initialize encryption key", err)
		}
	}

	if len(combined) != KeySize+SaltSize {
		return NewStorageError(ErrEncryption, "invalid key data length", nil)
	}

	masterKey := combined[:KeySize]
	salt := combined[KeySize:]

	// Derive the actual encryption key using Argon2
	derivedKey := argon2.IDKey(masterKey, salt, argon2Time, argon2Memory, argon2Threads, KeySize)

	// Initialize AES-GCM cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return NewStorageError(ErrEncryption, "failed to create AES cipher", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return NewStorageError(ErrEncryption, "failed to create GCM cipher", err)
	}

	em.key = derivedKey
	em.salt = salt
	em.aead = aead

	return nil
}

// generateAndStoreKey generates a new random key+salt and stores via vault.
func (em *EncryptionManager) generateAndStoreKey() ([]byte, error) {
	// Generate random master key
	masterKey := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}

	// Generate random salt
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Combine and store via vault (DPAPI-protected on Windows)
	combined := append(masterKey, salt...)
	if err := em.vault.Save(VaultKeyName, combined); err != nil {
		return nil, fmt.Errorf("failed to store key in vault: %w", err)
	}

	return combined, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
func (em *EncryptionManager) Encrypt(plaintext []byte) ([]byte, error) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	if em.aead == nil {
		return nil, NewStorageError(ErrEncryption, "encryption not initialized", nil)
	}

	// Handle empty input
	if len(plaintext) == 0 {
		return []byte{}, nil
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, NewStorageError(ErrEncryption, "failed to generate nonce", err)
	}

	// Encrypt and prepend nonce
	ciphertext := em.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
func (em *EncryptionManager) Decrypt(ciphertext []byte) ([]byte, error) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	if em.aead == nil {
		return nil, NewStorageError(ErrEncryption, "encryption not initialized", nil)
	}

	// Handle empty input
	if len(ciphertext) == 0 {
		return []byte{}, nil
	}

	// Validate minimum length (nonce + at least auth tag)
	if len(ciphertext) < NonceSize+em.aead.Overhead() {
		return nil, NewStorageError(ErrEncryption, "ciphertext too short", nil)
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:NonceSize]
	encryptedData := ciphertext[NonceSize:]

	// Decrypt
	plaintext, err := em.aead.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, NewStorageError(ErrEncryption, "decryption failed", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (em *EncryptionManager) EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	ciphertext, err := em.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts base64-encoded ciphertext and returns the plaintext string.
func (em *EncryptionManager) DecryptString(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", NewStorageError(ErrEncryption, "invalid base64 encoding", err)
	}

	plaintext, err := em.Decrypt(data)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// RotateKey rotates the encryption key with a new passphrase.
func (em *EncryptionManager) RotateKey(newPassphrase string) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Generate new salt
	newSalt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, newSalt); err != nil {
		return NewStorageError(ErrEncryption, "failed to generate new salt", err)
	}

	// Derive new key from passphrase
	newKey := argon2.IDKey([]byte(newPassphrase), newSalt, argon2Time, argon2Memory, argon2Threads, KeySize)

	// Create new cipher
	block, err := aes.NewCipher(newKey)
	if err != nil {
		return NewStorageError(ErrEncryption, "failed to create new AES cipher", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return NewStorageError(ErrEncryption, "failed to create new GCM cipher", err)
	}

	// Store new key material in vault
	combined := append([]byte(newPassphrase), newSalt...)
	if err := em.vault.Save(VaultKeyName, combined); err != nil {
		return NewStorageError(ErrEncryption, "failed to store new key", err)
	}

	// Update internal state
	em.key = newKey
	em.salt = newSalt
	em.aead = aead

	return nil
}
