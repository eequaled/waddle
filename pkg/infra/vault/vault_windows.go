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
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("vault load %s: %w", key, err)
	}
	return dpapiDecrypt(encrypted)
}

func (v *windowsVault) Delete(key string) error {
	path := filepath.Join(v.storePath, key+".dat")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil // idempotent
	}
	return err
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
