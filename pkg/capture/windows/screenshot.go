//go:build windows

package windows

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"syscall"
	"unsafe"

	"github.com/kbinani/screenshot"
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procGetWindowRect = user32.NewProc("GetWindowRect")
)

// RECT represents a Win32 RECT structure.
type RECT struct {
	Left, Top, Right, Bottom int32
}

// SaveActiveWindow captures the window and saves it to the specified path.
func SaveActiveWindow(hwnd syscall.Handle, filepath string) error {
	var rect RECT
	ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return fmt.Errorf("failed to get window rect")
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)

	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid window dimensions: %dx%d", width, height)
	}

	// Handle multi-monitor coordinates correctly by using screenshot.CaptureRect
	// screenshot.CaptureRect takes global coordinates, which GetWindowRect returns.
	img, err := screenshot.CaptureRect(image.Rect(int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom)))
	if err != nil {
		return fmt.Errorf("failed to capture screen rect: %v", err)
	}

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("failed to encode png: %v", err)
	}

	return nil
}

// CaptureWindowAsPNG captures a specific window by handle and returns PNG bytes.
// Uses GetWindowRect to convert hwnd → global coordinates, then captures that region.
func CaptureWindowAsPNG(hwnd uintptr) ([]byte, error) {
	var rect RECT
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return nil, fmt.Errorf("failed to get window rect")
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid window dimensions: %dx%d", width, height)
	}

	img, err := screenshot.CaptureRect(image.Rect(int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom)))
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen rect: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// CaptureFullScreen captures the entire primary display and returns PNG bytes.
func CaptureFullScreen() ([]byte, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture full screen: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}
