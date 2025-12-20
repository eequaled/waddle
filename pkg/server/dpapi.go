package server

import (
	"fmt"
	"syscall"
	"unsafe"
	
	"golang.org/x/sys/windows"
)

// DPAPI provides Windows Data Protection API functionality
type DPAPI struct{}

// DataBlob represents a Windows DATA_BLOB structure
type DataBlob struct {
	cbData uint32
	pbData *byte
}

// NewDPAPI creates a new DPAPI instance
func NewDPAPI() *DPAPI {
	return &DPAPI{}
}

// Protect encrypts data using DPAPI for the current user
func (d *DPAPI) Protect(data []byte, description string) ([]byte, error) {
	// Load crypt32.dll
	crypt32 := syscall.NewLazyDLL("crypt32.dll")
	cryptProtectData := crypt32.NewProc("CryptProtectData")
	
	// Prepare input data blob
	var inBlob DataBlob
	if len(data) > 0 {
		inBlob.cbData = uint32(len(data))
		inBlob.pbData = &data[0]
	}
	
	// Prepare description (optional)
	var descPtr *uint16
	if description != "" {
		desc, err := syscall.UTF16PtrFromString(description)
		if err != nil {
			return nil, fmt.Errorf("failed to convert description to UTF16: %w", err)
		}
		descPtr = desc
	}
	
	// Prepare output data blob
	var outBlob DataBlob
	
	// Call CryptProtectData
	ret, _, err := cryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),  // pDataIn
		uintptr(unsafe.Pointer(descPtr)),  // szDataDescr
		0,                                 // pOptionalEntropy (NULL)
		0,                                 // pvReserved (NULL)
		0,                                 // pPromptStruct (NULL)
		0,                                 // dwFlags (0 = default)
		uintptr(unsafe.Pointer(&outBlob)), // pDataOut
	)
	
	if ret == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %w", err)
	}
	
	// Copy protected data
	if outBlob.cbData == 0 {
		return nil, fmt.Errorf("CryptProtectData returned empty data")
	}
	
	protectedData := make([]byte, outBlob.cbData)
	copy(protectedData, (*[1 << 30]byte)(unsafe.Pointer(outBlob.pbData))[:outBlob.cbData:outBlob.cbData])
	
	// Free the output buffer
	d.localFree(uintptr(unsafe.Pointer(outBlob.pbData)))
	
	return protectedData, nil
}

// Unprotect decrypts DPAPI-protected data
func (d *DPAPI) Unprotect(protectedData []byte) ([]byte, string, error) {
	// Load crypt32.dll
	crypt32 := syscall.NewLazyDLL("crypt32.dll")
	cryptUnprotectData := crypt32.NewProc("CryptUnprotectData")
	
	// Prepare input data blob
	var inBlob DataBlob
	if len(protectedData) > 0 {
		inBlob.cbData = uint32(len(protectedData))
		inBlob.pbData = &protectedData[0]
	}
	
	// Prepare output data blob
	var outBlob DataBlob
	
	// Prepare description output
	var descPtr *uint16
	
	// Call CryptUnprotectData
	ret, _, err := cryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),   // pDataIn
		uintptr(unsafe.Pointer(&descPtr)),  // ppszDataDescr
		0,                                  // pOptionalEntropy (NULL)
		0,                                  // pvReserved (NULL)
		0,                                  // pPromptStruct (NULL)
		0,                                  // dwFlags (0 = default)
		uintptr(unsafe.Pointer(&outBlob)),  // pDataOut
	)
	
	if ret == 0 {
		return nil, "", fmt.Errorf("CryptUnprotectData failed: %w", err)
	}
	
	// Copy unprotected data
	if outBlob.cbData == 0 {
		return nil, "", fmt.Errorf("CryptUnprotectData returned empty data")
	}
	
	data := make([]byte, outBlob.cbData)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(outBlob.pbData))[:outBlob.cbData:outBlob.cbData])
	
	// Get description if available
	var description string
	if descPtr != nil {
		description = windows.UTF16PtrToString(descPtr)
		d.localFree(uintptr(unsafe.Pointer(descPtr)))
	}
	
	// Free the output buffer
	d.localFree(uintptr(unsafe.Pointer(outBlob.pbData)))
	
	return data, description, nil
}

// localFree frees memory allocated by Windows APIs
func (d *DPAPI) localFree(ptr uintptr) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	localFree := kernel32.NewProc("LocalFree")
	localFree.Call(ptr)
}

// IsAvailable checks if DPAPI is available on the current system
func (d *DPAPI) IsAvailable() bool {
	// DPAPI is available on Windows 2000 and later
	// For simplicity, we'll assume it's available if we can load crypt32.dll
	crypt32 := syscall.NewLazyDLL("crypt32.dll")
	return crypt32.Load() == nil
}