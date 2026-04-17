//go:build windows

package platform

import capwin "waddle/pkg/capture/windows"

// windowsScreenCapturer delegates to the capture/windows package's screenshot functions.
type windowsScreenCapturer struct{}

func (w *windowsScreenCapturer) CaptureWindow(hwnd uintptr) ([]byte, error) {
	return capwin.CaptureWindowAsPNG(hwnd)
}

func (w *windowsScreenCapturer) CaptureScreen() ([]byte, error) {
	return capwin.CaptureFullScreen()
}
