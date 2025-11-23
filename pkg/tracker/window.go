package tracker

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")

	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	PROCESS_QUERY_INFORMATION         = 0x0400
	MAX_PATH                          = 260
)

type FocusEvent struct {
	Timestamp time.Time
	AppName   string
	PID       uint32
	Title     string
}

type Poller struct {
	events chan FocusEvent
}

func NewPoller() *Poller {
	return &Poller{
		events: make(chan FocusEvent),
	}
}

func (p *Poller) Start() <-chan FocusEvent {
	go p.poll()
	return p.events
}

func (p *Poller) poll() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var (
		lastHwnd        syscall.Handle
		stableStartTime time.Time
		reportedHwnd    syscall.Handle
	)

	stableStartTime = time.Now()

	for range ticker.C {
		currentHwnd := getForegroundWindow()

		if currentHwnd != lastHwnd {
			lastHwnd = currentHwnd
			stableStartTime = time.Now()
			continue
		}

		if time.Since(stableStartTime) >= 1*time.Second {
			if currentHwnd != reportedHwnd {
				pid := getWindowThreadProcessId(currentHwnd)
				title := getWindowText(currentHwnd)
				exeName := getProcessExecName(pid)

				p.events <- FocusEvent{
					Timestamp: time.Now(),
					AppName:   exeName,
					PID:       pid,
					Title:     title,
				}

				reportedHwnd = currentHwnd
			}
		}
	}
}

// Windows API Helpers (Copied from original main.go)

func getForegroundWindow() syscall.Handle {
	ret, _, _ := procGetForegroundWindow.Call()
	return syscall.Handle(ret)
}

func getWindowText(hwnd syscall.Handle) string {
	buf := make([]uint16, 512)
	ret, _, _ := procGetWindowTextW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func getWindowThreadProcessId(hwnd syscall.Handle) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&pid)),
	)
	return pid
}

func getProcessExecName(pid uint32) string {
	hProcess, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_LIMITED_INFORMATION),
		uintptr(0), // bInheritHandle = FALSE
		uintptr(pid),
	)
	if hProcess == 0 {
		hProcess, _, _ = procOpenProcess.Call(
			uintptr(PROCESS_QUERY_INFORMATION),
			uintptr(0),
			uintptr(pid),
		)
	}

	if hProcess == 0 {
		return "Unknown"
	}
	defer closeHandle(syscall.Handle(hProcess))

	buf := make([]uint16, MAX_PATH*2)
	size := uint32(len(buf))

	ret, _, _ := procQueryFullProcessImageNameW.Call(
		hProcess,
		uintptr(0),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		return "Unknown"
	}

	fullPath := syscall.UTF16ToString(buf[:size])
	return getFileName(fullPath)
}

func closeHandle(handle syscall.Handle) {
	procCloseHandle.Call(uintptr(handle))
}

func getFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
