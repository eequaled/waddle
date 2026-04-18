# Waddle Build Guide

This document outlines the required toolchain and steps to build the Waddle project reproducibly.

## Required Toolchain
- **Go Version:** 1.25.0
- **Node.js Version:** 20+ (with npm)
- **Wails CLI:** v2.11.0 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`)
- **Windows SDK:** Required for CGo and Wails WebView2 compilation
- **WebView2 Runtime:** Required on the target Windows machine

## Multi-Module Workspace
The project currently uses a single Go module (`waddle`). A `go.work` file is deferred until a multi-module need arises (e.g., separating the capture or types packages).

## Build Instructions

### 1. Install Dependencies
```bash
go mod download
go mod verify
```

### 2. Frontend Build
The frontend uses React 19 + Vite.
```bash
cd frontend
npm install
npm run build
```

### 3. Application Build
To build the application for production:
```bash
wails build
```
This will produce `waddle.exe` in the `build/bin/` directory.

### 4. Development Mode
To run the application with hot-reloading:
```bash
wails dev
```

## Troubleshooting
- **ETW Admin Requirements:** The capture pipeline may require running Waddle as an Administrator for ETW tracing.
- **WebView2 Runtime:** If the app launches to a blank screen, ensure the Microsoft Edge WebView2 Runtime is installed.

## Known Issues

### Vite Dev Server Crash on First Run
- **Symptom:** `wails dev` exits immediately with exit status `0xc0000409` (STATUS_STACK_BUFFER_OVERRUN).
- **Workaround:** Simply run `wails dev` again — it works on the second attempt.
- **Root cause:** DevWatcher process stack overflow during initial Vite startup.
- **Status:** Non-blocking, does not affect production builds.

### Go 1.25.6 Compiler ICE

`go build ./...` may hit an internal compiler error (race condition) on Go 1.25.6.
Workaround: use `go build -p 1 ./...` to serialize compilation.
This is a known Go toolchain bug and should be fixed in a future release.
