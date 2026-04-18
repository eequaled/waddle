//go:build windows

package platform

import capwin "waddle/pkg/capture/windows"

// windowsUIReader bridges the capwin.Reader to the platform.UIReader interface.
// capwin.NewReader() creates its own Marshaler internally (STA COM thread).
// Close() delegates to reader.Close() → marshaler.Close(), so the entire
// STA lifecycle is self-contained.
type windowsUIReader struct {
	reader *capwin.Reader
}

func newWindowsUIReader() (*windowsUIReader, error) {
	r, err := capwin.NewReader()
	if err != nil {
		return nil, err
	}
	return &windowsUIReader{reader: r}, nil
}

func (w *windowsUIReader) GetStructuredData(hwnd uintptr) (*WindowInfo, error) {
	info, err := w.reader.GetWindowInfo(hwnd)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return &WindowInfo{
		HWND:        info.HWND,
		ProcessID:   info.ProcessID,
		ProcessName: info.ProcessName,
		WindowTitle: info.WindowTitle,
		AppType:     info.AppType.String(), // capture.AppType → string
		Metadata:    info.Metadata,
	}, nil
}

func (w *windowsUIReader) Close() error {
	if w.reader != nil {
		return w.reader.Close()
	}
	return nil
}
