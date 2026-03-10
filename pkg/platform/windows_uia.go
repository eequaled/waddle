//go:build windows

package platform

import "waddle/pkg/capture/uia"

// windowsUIReader bridges the uia.Reader to the platform.UIReader interface.
// uia.NewReader() creates its own Marshaler internally (STA COM thread).
// Close() delegates to reader.Close() → marshaler.Close(), so the entire
// STA lifecycle is self-contained.
type windowsUIReader struct {
	reader *uia.Reader
}

func newWindowsUIReader() (*windowsUIReader, error) {
	r, err := uia.NewReader()
	if err != nil {
		return nil, err
	}
	return &windowsUIReader{reader: r}, nil
}

func (w *windowsUIReader) GetStructuredData(hwnd uintptr) (*UIResult, error) {
	info, err := w.reader.GetWindowInfo(hwnd)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return &UIResult{
		HWND:        info.HWND,
		ProcessID:   info.ProcessID,
		ProcessName: info.ProcessName,
		WindowTitle: info.WindowTitle,
		AppType:     info.AppType.String(), // uia.AppType → string
		Metadata:    info.Metadata,
	}, nil
}

func (w *windowsUIReader) Close() error {
	if w.reader != nil {
		return w.reader.Close()
	}
	return nil
}
