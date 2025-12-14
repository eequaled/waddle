# Waddle v1.0.0 - First Stable Release

**Release Date:** December 14, 2025

## Overview

This is the first stable, installable release of Waddle. The focus of this release is **installation reliability** and **cross-environment compatibility**. No new features have been added—this release ensures the application works smoothly on any Windows 10/11 machine without configuration conflicts or hardcoded dependencies.

## 🎯 What's New

### Installer Package
- **One-Click Installation**: Download `Waddle-1.0.0-Setup.exe` and install
- **Portable Version**: `Waddle-1.0.0-Portable.exe` for USB/no-install usage
- **Bundled Dependencies**: Tesseract OCR included (no separate installation required)
- **System Tray Integration**: Runs in background with tray icon controls

### Installation Improvements
- **Configurable Data Directory**: No longer hardcoded to specific OneDrive paths
  - Default: `~/Documents/Waddle/sessions`
  - Override via command-line: `waddle-backend.exe -data-dir "C:\custom\path"`
- **Standard System Paths**: Uses OS-standard directories for better compatibility
- **Automatic Directory Creation**: Creates necessary folders on first run

## 🔧 Technical Improvements

### Backend Stability
- **Optimized Blacklist Checking**: In-memory caching reduces disk I/O by ~95%
- **Configurable API Port**: Backend port can be changed via `-port` flag (default: 8080)
- **Better Error Handling**: Graceful degradation when optional features (AI, OCR) are unavailable

### Frontend Performance
- **Lazy Loading**: Sessions load on-demand instead of all at once (fixes performance with 100+ sessions)
- **Auto-Refresh**: Active session updates every 30 seconds to show new captured data
- **Improved Session Metadata**: Custom titles and notes now persist correctly across restarts

### Security Fixes
- **File Upload Validation**: Profile images validated via MIME type detection (prevents malicious uploads)
- **Sanitized Filenames**: File paths cleaned to prevent directory traversal attacks

### Electron Integration
- **Improved Error Logging**: Better diagnostics for startup and loading failures
- **CORS Handling**: Frontend-backend communication works reliably in packaged app
- **Resource Path Resolution**: Correct path handling for bundled assets (frontend, Tesseract, profiles)

## 📋 Known Limitations

### AI Features Require Ollama
**Waddle does NOT bundle Ollama.** AI chat features require separate installation:

1. Download Ollama from [https://ollama.ai](https://ollama.ai)
2. Install and run: `ollama serve`
3. Pull the model: `ollama pull gemma2:2b`


### Windows Only
This release supports **Windows 10/11 only**. Linux and macOS are not supported due to Windows-specific APIs (UI Automation, Win32).

## 🚀 Installation Instructions

### Option A: Installer (Recommended)
1. Download `Waddle-1.0.0-Setup.exe`
2. Run the installer
3. Launch Waddle from the Start Menu or Desktop shortcut

### Option B: Portable
1. Download `Waddle-1.0.0-Portable.exe`
2. Run directly (no installation needed)
3. Data will be stored in `~/Documents/Waddle`

### Optional: Install Ollama for AI Features
```bash
# Download from https://ollama.ai, then:
ollama serve
ollama pull gemma2:2b
```

## 📝 What's Not Included

This is a **stability and compatibility release**. No new features were added. Focus areas:
- ✅ Cross-machine compatibility
- ✅ Installation reliability
- ✅ Performance optimization
- ✅ Security hardening
- ❌ New capture features
- ❌ UI/UX changes
- ❌ New AI capabilities

## 🐛 Bug Fixes

- Fixed: Sessions not appearing after initial install
- Fixed: Slow UI loading with large session histories
- Fixed: Session titles/notes not saving
- Fixed: Insecure profile image uploads
- Fixed: Electron app showing blank screen on some systems

## 📖 Documentation

- [README.md](README.md) - Quick start and usage guide
- [BUILD.md](BUILD.md) - Build from source instructions

## 🔄 Upgrade Notes

**First release** - No upgrade path needed.

If you were running a development build:
1. Uninstall the old version
2. Install v1.0.0 via the installer
3. Your session data in `~/Documents/Waddle` will be preserved

## 🙏 Acknowledgments

This release focused on making Waddle accessible to users beyond the development environment. Special thanks to early testers who identified installation and compatibility issues.

---

**Full Changelog**: Initial stable release
**Download**: [Releases Page](../../releases/tag/v1.0.0)
