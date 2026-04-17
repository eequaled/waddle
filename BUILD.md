# Building Waddle v2

Waddle has been migrated from Electron to Wails v2 + Svelte for better performance and smaller binary size.

## Prerequisites

1. **Go** (v1.24 or later)
2. **Node.js** (v18 or later) + **npm**
3. **Wails CLI**:
   ```powershell
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

## Development

To run in development mode with hot reload:
```powershell
wails dev
```

## Building for Production

To create a production build:
```powershell
wails build
```
The resulting executable will be in the `build/bin` directory.

## Architecture Notes

- **Backend**: Go (Wails v2)
- **Frontend**: Svelte (Vanilla CSS)
- **Platform Layer**: Abstraction in `pkg/platform` handles Windows ETW/UIA natively.
- **Storage**: Custom engine in `pkg/storage` (SQLite).
