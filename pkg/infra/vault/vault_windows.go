//go:build windows

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DATA_BLOB structure for DPAPI — mirrors the CRYPTOAPI_BLOB struct.
type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32                = windows.NewLazySystemDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

// windowsVault stores DPAPI-protected blobs as files under storePath.
type windowsVault struct {
	storePath string
}

// New creates a Vault that stores DPAPI-protected data as files.
func New(storePath string) Vault {
	os.MkdirAll(storePath, 0700)
	return &windowsVault{storePath: storePath}
}

func (v *windowsVault) Save(key string, data []byte) error {
	encrypted, err := dpapiEncrypt(data)
	if err != nil {
		return fmt.Errorf("vault save %s: %w", key, err)
	}
	path := filepath.Join(v.storePath, key+".dat")
	return os.WriteFile(path, encrypted, 0600)
}

func (v *windowsVault) Load(key string) ([]byte, error) {
	path := filepath.Join(v.storePath, key+".dat")
	encrypted, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && key == "master" {
			// Legacy fallback for v1 migration: try Credential Manager
			legacyData, err := legacyReadFromCredentialManager()
			if err == nil && len(legacyData) > 0 {
				// We found legacy data! Save it in the new format right away for migration
				// The legacy data is already DPAPI protected, but our Save function will DPAPI protect it *again*.
				// Wait, legacy data is already DPAPI encrypted. 
				// The Vault interface expects byte arrays that it will DPAPI encrypt. 
				// So we should first decrypt the legacy DPAPI blob, then return it.
				plaintext, decErr := dpapiDecrypt(legacyData)
				if decErr == nil {
					// Save the plaintext via the new Vault interface to migrate it
					_ = v.Save(key, plaintext)
					_ = legacyDeleteFromCredentialManager() // <-- Clean up old key!
					return plaintext, nil
				}
			}
			return nil, ErrKeyNotFound
		}
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("vault load %s: %w", key, err)
	}
	return dpapiDecrypt(encrypted)
}

func (v *windowsVault) Delete(key string) error {
	if key == "master" {
		// Attempt to delete from legacy credential manager just in case
		_ = legacyDeleteFromCredentialManager()
	}
	path := filepath.Join(v.storePath, key+".dat")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil // idempotent
	}
	return err
}

// ── Legacy V1 Credential Manager Support ───────────────────────

const legacyCredentialName = "Waddle_Encryption_Key"

var (
	advapi32        = windows.NewLazySystemDLL("advapi32.dll")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
)

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

func legacyReadFromCredentialManager() ([]byte, error) {
	targetName, err := windows.UTF16PtrFromString(legacyCredentialName)
	if err != nil {
		return nil, err
	}
	var pcred *credential
	ret, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		1, // CRED_TYPE_GENERIC
		0,
		uintptr(unsafe.Pointer(&pcred)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("CredReadW failed: %w", err)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(pcred)))

	result := make([]byte, pcred.CredentialBlobSize)
	copy(result, unsafe.Slice(pcred.CredentialBlob, pcred.CredentialBlobSize))
	return result, nil
}

func legacyDeleteFromCredentialManager() error {
	targetName, err := windows.UTF16PtrFromString(legacyCredentialName)
	if err != nil {
		return err
	}
	ret, _, err := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetName)),
		1, // CRED_TYPE_GENERIC
		0,
	)
	if ret == 0 {
		return fmt.Errorf("CredDeleteW failed: %w", err)
	}
	return nil
}

// dpapiEncrypt encrypts data using Windows DPAPI (CryptProtectData).
func dpapiEncrypt(data []byte) ([]byte, error) {
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

// dpapiDecrypt decrypts data using Windows DPAPI (CryptUnprotectData).
func dpapiDecrypt(data []byte) ([]byte, error) {
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
