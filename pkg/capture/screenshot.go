package capture

import (
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
