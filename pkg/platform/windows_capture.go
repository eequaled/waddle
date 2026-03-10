//go:build windows

package platform

import "waddle/pkg/capture"

// windowsScreenCapturer delegates to the capture package's screenshot functions.
type windowsScreenCapturer struct{}

func (w *windowsScreenCapturer) CaptureWindow(hwnd uintptr) ([]byte, error) {
	return capture.CaptureWindowAsPNG(hwnd)
}

func (w *windowsScreenCapturer) CaptureScreen() ([]byte, error) {
	return capture.CaptureFullScreen()
}
