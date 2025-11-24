# Ideathon — Memory & Context Tracker

## 1. Project Overview

### Purpose
A local-first memory tool that silently captures activity (focused windows, clipboard, and visible text), synthesizes sessions, and presents an intelligent editor to refine those into durable knowledge. Privacy-first and aligned with developer workflows.

### Key Features
- Passive window focus tracking with app/title metadata
- Clipboard change monitoring
- Visible text extraction from the active window via Windows UI Automation
- UI with Activity Timeline, Editor scaffold, and Search overlay
- Session rotation and saving to local files for auditability

### Technologies
- Frontend: React 19, Vite 7, Tailwind v4, TipTap, lucide icons
- Backend: Go 1.21, Windows APIs (`user32.dll`, `kernel32.dll`), PowerShell UIAutomation
- Storage: Local files (daily logs + session chunks)

### System Requirements
- OS: Windows (required for UI Automation and Win32 APIs)
- Runtime: Go 1.21+, Node.js 18+, npm
- Browser: Modern Chrome/Edge/Firefox; Safari to be validated for editor behaviors

## 2. Current Implementation

### Components & Modules
- `frontend/src/components/SessionCard.jsx`: timeline items with app icons and tags
- `frontend/src/components/SearchModal.jsx`: modal overlay with filters and mock results
- `frontend/src/components/Editor.jsx`: TipTap-powered rich editor scaffold
- `pkg/tracker/window.go`: foreground window detection via Win32 API
- `pkg/content/clipboard.go`: clipboard monitoring
- `pkg/content/automation.go` + `pkg/content/scripts/get_text.ps1`: text extraction from active window
- `pkg/storage/file_manager.go`: daily append logger
- `main.go`: orchestrates channels, session lifecycle, and saving

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                             Frontend (UI)                           │
│  React + Vite + Tailwind + TipTap                                   │
│  ┌──────────────┐   ┌─────────────┐   ┌───────────────┐            │
│  │ Activity     │   │ Search      │   │ Intelligent    │            │
│  │ Timeline     │   │ Modal       │   │ Editor         │            │
│  └──────────────┘   └─────────────┘   └───────────────┘            │
└─────────────────────────────────────────────────────────────────────┘
                      ▲                           │
                      │                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                              Backend (Go)                           │
│  tracker/window.go → Focus events                                   │
│  content/clipboard.go → Clipboard events                             │
│  automation.go + get_text.ps1 → Visible text extraction               │
│  storage/file_manager.go → Daily logs + session chunks               │
│  main.go → Event loop & session lifecycle                            │
└─────────────────────────────────────────────────────────────────────┘
```

### Internal API
- Focus events: `FocusEvent{ Timestamp, AppName, PID, Title }`
- Clipboard events: `ClipboardEvent{ Timestamp, Content }`
- `ExtractContext()` returns window text; invoked periodically in-session
- Files: daily logs + `YYYY-MM-DD_HH-MM-SS_AppName.txt` per session chunk

## 3. Future Development

### Roadmap
- Persist sessions and artifacts with a searchable index
- AI summarization (local or opt-in remote)
- Artifact capture for links/files/screenshots
- Natural-language search with include/exclude app filters
- Responsive UI polish (sidebar collapse, editor toolbars)

### Known Limitations
- Windows-only backend at present
- Not all applications expose accessible text
- Browser automation requires environment preparation

### Proposed Enhancements
- macOS/Linux trackers and platform abstractions
- Code splitting to reduce initial bundle size (TipTap lazy load)
- Background indexing worker and relevance

## 4. Setup Instructions

### Clone & Install
```bash
git clone <repository-url>
cd ideathon
```

#### Frontend
```bash
cd frontend
npm install
npm run dev  # http://localhost:5173/

# Production
npm run build
npm run preview  # http://localhost:4173/
```

#### Backend
```bash
go run .
```

### First Run
- Start backend for monitoring
- Open frontend to view timeline and editor
- Logs and session files appear under `ideathon txt experiments`

## 5. Contribution Guidelines

### Code Style
- Frontend: ESLint 9, Tailwind utility-first
- Backend: idiomatic Go, clear concurrency handling

### Testing
- Frontend: lint, build, and preview must pass
- Playwright tests recommended once environment is prepared
- Backend: unit tests for storage/content; manual focus and extraction verification

### Pull Requests
- Branch per feature
- Include tests and docs
- Ensure lint/build succeed; document OS-specific behaviors

## Troubleshooting
- Tailwind v4 tokens must be declared in `frontend/src/index.css`
- Install Playwright browsers with `npx playwright install` if using automation
- Empty session files may occur for apps without accessible text

## License
MIT

A session tracking application that monitors your active windows, captures context, and provides an intelligent interface for reviewing and summarizing your activities.

## Features

- **Passive Window Tracking**: Monitors active windows and applications
- **Context Extraction**: Captures text content from active windows using Windows UI Automation
- **Session Management**: Automatically groups activities into 2-hour chunks
- **Modern UI**: React-based interface with Activity Timeline, Intelligent Editor, and Search
- **Dark Mode**: Premium dark theme with glassmorphism effects

## Requirements

### Backend (Go)
- Go 1.20 or higher
- Windows OS (for UI Automation)

### Frontend (Node.js)
- Node.js 18+ and npm
- Dependencies listed in `frontend/package.json`:
  - React 18+
  - Vite 5+
  - Tailwind CSS 4+
  - TipTap (Rich text editor)
  - Lucide React (Icons)

## Installation

### 1. Clone the Repository
\`\`\`bash
git clone <repository-url>
cd ideathon
\`\`\`

### 2. Backend Setup
\`\`\`bash
# Build the Go application
go build -o ideathon.exe main.go
\`\`\`

### 3. Frontend Setup
\`\`\`bash
cd frontend
npm install
\`\`\`

## Running the Application

### Start the Backend
\`\`\`bash
# From the root directory
./ideathon.exe
\`\`\`

The tracker will:
- Monitor active windows every 500ms
- Capture clipboard changes
- Extract window content every 10 seconds
- Save session logs to `Documents/ideathon txt experiments/`

### Start the Frontend (Development)
\`\`\`bash
cd frontend
npm run dev
\`\`\`

Access the UI at: `http://localhost:5173/`

### Build Frontend for Production
\`\`\`bash
cd frontend
npm run build
\`\`\`

## Project Structure

\`\`\`
ideathon/
├── main.go                 # Main application entry point
├── pkg/
│   ├── tracker/           # Window tracking logic
│   │   └── window.go
│   ├── content/           # Content extraction
│   │   ├── automation.go  # Go wrapper for PowerShell
│   │   ├── clipboard.go   # Clipboard monitoring
│   │   └── scripts/
│   │       └── get_text.ps1  # UI Automation script
│   └── storage/           # Data persistence
│       └── logger.go
└── frontend/              # React UI
    ├── src/
    │   ├── App.jsx        # Main application
    │   ├── components/    # UI components
    │   └── lib/           # Utilities
    └── package.json       # Node.js dependencies
\`\`\`

## Usage

### Keyboard Shortcuts
- **Ctrl+S**: Open search modal
- **Esc**: Close modal/dialog

### Session Management
- Sessions are automatically saved when you switch windows
- Sessions longer than 2 hours are split into chunks
- Each session is saved as: `YYYY-MM-DD_HH-MM-SS_AppName.txt`

## Development

### Backend Development
The Go backend uses:
- Windows API for window tracking
- PowerShell for UI Automation
- Goroutines for concurrent monitoring

### Frontend Development
The React frontend uses:
- **Vite** for fast development
- **Tailwind CSS** for styling
- **TipTap** for rich text editing
- **shadcn/ui** patterns for components

## Troubleshooting

### "Unknown at rule @tailwind" warnings
These are harmless - they're Tailwind-specific CSS directives that standard CSS linters don't recognize.

### Empty session files
Some applications (games, custom UIs) may not expose text to Windows UI Automation. Standard apps like browsers, Word, Notepad, and VS Code work best.

### Build errors
Make sure you have:
- Go 1.20+ installed
- Node.js 18+ installed
- All dependencies installed (`go mod download` and `npm install`)

## License

MIT

## Contributing

Contributions welcome! Please open an issue or pull request.
