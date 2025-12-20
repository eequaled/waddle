package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"unsafe"

	"golang.org/x/crypto/argon2"
	"golang.org/x/sys/windows"
)

const (
	// KeySize is the size of the AES-256 key in bytes.
	KeySize = 32
	// NonceSize is the size of the GCM nonce in bytes.
	NonceSize = 12
	// SaltSize is the size of the salt for key derivation.
	SaltSize = 16
	// CredentialName is the name used to store the key in Windows Credential Manager.
	CredentialName = "Waddle_Encryption_Key"
)

// Argon2 parameters for key derivation.
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
)

// EncryptionManager handles encryption/decryption using Windows DPAPI and AES-256-GCM.
type EncryptionManager struct {
	key   []byte
	salt  []byte
	aead  cipher.AEAD
	mutex sync.RWMutex
}

// NewEncryptionManager creates a new EncryptionManager instance.
func NewEncryptionManager() *EncryptionManager {
	return &EncryptionManager{}
}

// InitializeKey initializes the encryption key from Windows Credential Manager.
// If no key exists, it generates a new one and stores it.
func (em *EncryptionManager) InitializeKey() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Try to load existing key from Credential Manager
	key, salt, err := em.loadKeyFromCredentialManager()
	if err != nil {
		// Key doesn't exist, generate a new one
		key, salt, err = em.generateAndStoreKey()
		if err != nil {
			return NewStorageError(ErrEncryption, "failed to initialize encryption key", err)
		}
	}

	// Derive the actual encryption key using Argon2
	derivedKey := argon2.IDKey(key, salt, argon2Time, argon2Memory, argon2Threads, KeySize)

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

// generateAndStoreKey generates a new random key and stores it in Credential Manager.
func (em *EncryptionManager) generateAndStoreKey() ([]byte, []byte, error) {
	// Generate random master key
	masterKey := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		return nil, nil, fmt.Errorf("failed to generate master key: %w", err)
	}

	// Generate random salt
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Combine key and salt for storage
	combined := append(masterKey, salt...)

	// Protect with DPAPI
	protected, err := em.dpapiEncrypt(combined)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to protect key with DPAPI: %w", err)
	}

	// Store in Credential Manager
	if err := em.storeInCredentialManager(protected); err != nil {
		return nil, nil, fmt.Errorf("failed to store key in Credential Manager: %w", err)
	}

	return masterKey, salt, nil
}

// loadKeyFromCredentialManager loads the encryption key from Windows Credential Manager.
func (em *EncryptionManager) loadKeyFromCredentialManager() ([]byte, []byte, error) {
	protected, err := em.readFromCredentialManager()
	if err != nil {
		return nil, nil, err
	}

	// Decrypt with DPAPI
	combined, err := em.dpapiDecrypt(protected)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt key with DPAPI: %w", err)
	}

	if len(combined) != KeySize+SaltSize {
		return nil, nil, fmt.Errorf("invalid key data length")
	}

	masterKey := combined[:KeySize]
	salt := combined[KeySize:]

	return masterKey, salt, nil
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

	// Validate minimum length (nonce + at least 1 byte + auth tag)
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
// This re-encrypts all data with the new key.
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

	// Store new key in Credential Manager
	combined := append([]byte(newPassphrase), newSalt...)
	protected, err := em.dpapiEncrypt(combined)
	if err != nil {
		return NewStorageError(ErrEncryption, "failed to protect new key", err)
	}

	if err := em.storeInCredentialManager(protected); err != nil {
		return NewStorageError(ErrEncryption, "failed to store new key", err)
	}

	// Update internal state
	em.key = newKey
	em.salt = newSalt
	em.aead = aead

	return nil
}

// DPAPI functions using Windows syscalls

var (
	crypt32                  = windows.NewLazySystemDLL("crypt32.dll")
	procCryptProtectData     = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData   = crypt32.NewProc("CryptUnprotectData")
	advapi32                 = windows.NewLazySystemDLL("advapi32.dll")
	procCredWriteW           = advapi32.NewProc("CredWriteW")
	procCredReadW            = advapi32.NewProc("CredReadW")
	procCredDeleteW          = advapi32.NewProc("CredDeleteW")
	procCredFree             = advapi32.NewProc("CredFree")
)

// DATA_BLOB structure for DPAPI
type dataBlob struct {
	cbData uint32
	pbData *byte
}

// CREDENTIAL structure for Credential Manager
type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

const (
	CRED_TYPE_GENERIC          = 1
	CRED_PERSIST_LOCAL_MACHINE = 2
)

// dpapiEncrypt encrypts data using Windows DPAPI.
func (em *EncryptionManager) dpapiEncrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot encrypt empty data")
	}

	input := dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
	var output dataBlob

	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&input)),
		0, // description
		0, // entropy
		0, // reserved
		0, // prompt struct
		0, // flags
		uintptr(unsafe.Pointer(&output)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %w", err)
	}

	// Copy output data
	result := make([]byte, output.cbData)
	copy(result, unsafe.Slice(output.pbData, output.cbData))

	// Free the output buffer
	windows.LocalFree(windows.Handle(unsafe.Pointer(output.pbData)))

	return result, nil
}

// dpapiDecrypt decrypts data using Windows DPAPI.
func (em *EncryptionManager) dpapiDecrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot decrypt empty data")
	}

	input := dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
	var output dataBlob

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&input)),
		0, // description
		0, // entropy
		0, // reserved
		0, // prompt struct
		0, // flags
		uintptr(unsafe.Pointer(&output)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}

	// Copy output data
	result := make([]byte, output.cbData)
	copy(result, unsafe.Slice(output.pbData, output.cbData))

	// Free the output buffer
	windows.LocalFree(windows.Handle(unsafe.Pointer(output.pbData)))

	return result, nil
}

// storeInCredentialManager stores data in Windows Credential Manager.
func (em *EncryptionManager) storeInCredentialManager(data []byte) error {
	targetName, err := windows.UTF16PtrFromString(CredentialName)
	if err != nil {
		return err
	}

	userName, err := windows.UTF16PtrFromString("WaddleUser")
	if err != nil {
		return err
	}

	cred := credential{
		Type:               CRED_TYPE_GENERIC,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(data)),
		CredentialBlob:     &data[0],
		Persist:            CRED_PERSIST_LOCAL_MACHINE,
		UserName:           userName,
	}

	ret, _, err := procCredWriteW.Call(
		uintptr(unsafe.Pointer(&cred)),
		0, // flags
	)

	if ret == 0 {
		return fmt.Errorf("CredWriteW failed: %w", err)
	}

	return nil
}

// readFromCredentialManager reads data from Windows Credential Manager.
func (em *EncryptionManager) readFromCredentialManager() ([]byte, error) {
	targetName, err := windows.UTF16PtrFromString(CredentialName)
	if err != nil {
		return nil, err
	}

	var pcred *credential

	ret, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0, // flags
		uintptr(unsafe.Pointer(&pcred)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("CredReadW failed: %w", err)
	}

	defer procCredFree.Call(uintptr(unsafe.Pointer(pcred)))

	// Copy credential blob
	result := make([]byte, pcred.CredentialBlobSize)
	copy(result, unsafe.Slice(pcred.CredentialBlob, pcred.CredentialBlobSize))

	return result, nil
}

// deleteFromCredentialManager deletes the credential from Windows Credential Manager.
func (em *EncryptionManager) deleteFromCredentialManager() error {
	targetName, err := windows.UTF16PtrFromString(CredentialName)
	if err != nil {
		return err
	}

	ret, _, err := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0, // flags
	)

	if ret == 0 {
		return fmt.Errorf("CredDeleteW failed: %w", err)
	}

	return nil
}
