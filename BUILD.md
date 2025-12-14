# Building Waddle as an Installable App

This guide explains how to build Waddle as a standalone Windows application with an installer.

## Prerequisites

Before building, ensure you have:

1. **Go 1.20+** - https://go.dev/dl/
2. **Node.js 18+** - https://nodejs.org/
3. **Tesseract OCR** - https://github.com/UB-Mannheim/tesseract/wiki (for OCR features)

## Quick Build

### Using PowerShell (Recommended)
```powershell
# Full build with installer
.\build.ps1

# Portable version (no installation required)
.\build.ps1 -Portable

# Development build (no installer)
.\build.ps1 -Dev

# Skip frontend rebuild
.\build.ps1 -SkipFrontend

# Skip backend rebuild
.\build.ps1 -SkipBackend
```

### Using Batch File
```cmd
build.bat
```

## Manual Build Steps

### 1. Build the Go Backend
```bash
go mod download
go build -ldflags="-s -w" -o waddle-backend.exe .
```

### 2. Build the React Frontend
```bash
cd frontend
npm install
npm run build
cd ..
```

### 3. Setup Electron
```bash
cd electron
npm install
cd ..
```

### 4. Build the Installer
```bash
cd electron
npm run build:win      # Creates installer + portable
npm run build:portable # Creates portable only
cd ..
```

## Output Files

After building, you'll find the installers in `dist-electron/`:

| File | Description |
|------|-------------|
| `Waddle-1.0.0-Setup.exe` | Windows installer (NSIS) |
| `Waddle-1.0.0-Portable.exe` | Portable version (no install) |

## Development Mode

For development, run each component separately:

```bash
# Terminal 1: Backend
.\waddle-backend.exe

# Terminal 2: Frontend dev server
cd frontend && npm run dev

# Terminal 3: Electron (optional)
cd electron && npm start
```

## Build Configuration

### Electron Builder Config (`electron/package.json`)

The build configuration includes:
- **NSIS Installer**: Full Windows installer with Start Menu shortcuts
- **Portable**: Single executable, no installation required
- **Auto-updates**: Can be configured for future releases

### Customization

To customize the build:

1. **App Name/Version**: Edit `electron/package.json`
2. **Icon**: Replace `electron/icon.ico` (256x256 recommended)
3. **Installer Options**: Modify the `build.nsis` section

## Troubleshooting

### "Go not found"
Install Go from https://go.dev/dl/ and ensure it's in your PATH.

### "Node.js not found"
Install Node.js from https://nodejs.org/ (LTS version recommended).

### "electron-builder failed"
```bash
cd electron
rm -rf node_modules
npm install
npm run build:win
```

### "Backend won't start"
Ensure Tesseract OCR is installed at `C:\Program Files\Tesseract-OCR\`

### Large installer size
The installer includes:
- Go backend (~15-20 MB)
- Electron runtime (~150 MB)
- Frontend assets (~5 MB)

Total: ~170-180 MB (compressed to ~60-70 MB in installer)

## Distribution

### For End Users
Distribute the `Waddle-*-Setup.exe` installer. Users will need:
- Windows 10/11
- Tesseract OCR (for text extraction features)
- Ollama (optional, for AI features)

### For Portable Use
Distribute the `Waddle-*-Portable.exe`. No installation required.

## Code Signing (Optional)

For production distribution, sign your executables:

1. Obtain a code signing certificate
2. Add to `electron/package.json`:
```json
"win": {
  "certificateFile": "path/to/cert.pfx",
  "certificatePassword": "password"
}
```

## Auto-Updates (Future)

To enable auto-updates:

1. Set up a release server (GitHub Releases works)
2. Add `electron-updater` to dependencies
3. Configure update URL in main.js

---

*Last updated: December 2025*
