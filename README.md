# Waddle — AI-Powered Second Brain for Developers

A local-first memory tool that silently captures your activity (focused windows, clipboard, and visible text), synthesizes sessions, and presents an intelligent interface to refine those into durable knowledge. Privacy-first and aligned with developer workflows.

## Features

### Core Capture
- **Passive Window Tracking**: Monitors active windows and applications every 500ms
- **Context Extraction**: Captures text content via Windows UI Automation + OCR
- **Clipboard Monitoring**: Tracks clipboard changes with timestamps
- **Screenshot Capture**: Periodic screenshots with automatic OCR processing

### Second Brain Enhancement
- **Session Editing**: Edit titles, summaries, and add manual notes to curate memories
- **Contextual AI Chat**: Chat with AI grounded in specific session context
- **Enhanced Search**: Deep linking to specific sessions and memory blocks with text highlighting
- **Proactive Notifications**: AI-generated insights about usage patterns
- **Related Memories**: Automatic surfacing of similar past sessions
- **Activity Insights**: Visualize daily/weekly app usage patterns

### Smart Features
- **Semantic Tagging**: Auto-categorize activities (coding, research, communication)
- **Memory Linking**: Connect related sessions manually or via AI suggestions
- **Privacy Controls**: Exclude apps, private mode, data retention settings
- **Export**: Download sessions as Markdown files

## Tech Stack

### Frontend
- **React 19** - UI framework with TypeScript for type safety
- **Vite 7** - Lightning-fast build tool & dev server
- **Tailwind CSS v4** - Utility-first styling
- **Radix UI** - Accessible component primitives
- **shadcn/ui** - Beautiful component library
- **TipTap** - Rich text editor for session notes
- **Vitest** - Unit testing framework
- **fast-check** - Property-based testing for invariant verification

### Backend
- **Go 1.24** - High-performance backend
- **Windows APIs** (user32.dll, kernel32.dll) - Native system integration
- **PowerShell UI Automation** - Text extraction from UI elements
- **Ollama** - Local AI integration (gemma2:2b model)

### Storage
- Local file system (daily logs + session chunks)
- JSON-based notification storage

## Requirements

- **OS**: Windows 10/11 (required for UI Automation and Win32 APIs)
- **Go**: 1.20+
- **Node.js**: 18+
- **Ollama**: For local AI features (optional)

## Quick Start

### 1. Clone & Install
```bash
git clone <repository-url>
cd ideathon
```

### 2. Backend Setup
```bash
# Install Go dependencies
go mod download

# Build the application
go build -o ideathon.exe

# Run the tracker
./ideathon.exe
```

### 3. Frontend Setup
```bash
cd frontend

# Install dependencies
npm install

# Start dev server (http://localhost:5173)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

The frontend expects the Go backend running on `http://localhost:8080`.

### 4. Optional: Ollama Setup
```bash
# Install Ollama from https://ollama.ai
ollama pull gemma2:2b
```

## Project Structure
(old and inaccurate not gonna remake this)
```
ideathon/
├── main.go                    # Application entry point
├── pkg/
│   ├── ai/                    # Ollama AI client
│   ├── capture/               # Screenshot capture
│   ├── content/               # Clipboard & UI automation
│   ├── ocr/                   # Text extraction from images
│   ├── processing/            # Batch processing & memory management
│   ├── server/                # HTTP API server
│   ├── storage/               # File system operations
│   └── tracker/               # Window focus tracking
├── frontend/
│   ├── src/
│   │   ├── components/        # React components
│   │   │   ├── ui/           # shadcn/ui primitives
│   │   │   └── figma/        # Design system components
│   │   ├── services/          # API client & utilities
│   │   ├── hooks/             # Custom React hooks
│   │   ├── types/             # TypeScript definitions
│   │   ├── styles/            # Global CSS
│   │   ├── data/              # Mock data for development
│   │   └── test/              # Test files
│   │       ├── unit/         # Unit tests
│   │       └── *.property.test.ts  # Property-based tests
│   ├── public/                # Static assets
│   ├── index.html             # Entry HTML
│   ├── package.json           # Frontend dependencies
│   └── vite.config.js         # Vite configuration
├── profile/                   # Default profile images
└── sessions/                  # Session data storage
```

### Key Frontend Components

| Component | Description |
|-----------|-------------|
| `App.tsx` | Main application shell |
| `MainEditor` | Session viewer with edit mode |
| `ChatInterface` | Global AI chat |
| `ContextualChat` | Session-specific AI chat |
| `SearchModal` | Search with deep linking |
| `NotificationPanel` | Notifications dropdown |
| `ActivityTimeline` | Session list sidebar |
| `InsightsView` | Usage analytics |
| `RelatedMemories` | Similar sessions panel |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/sessions` | List all session dates |
| GET | `/api/sessions/{date}` | Get session details |
| PUT | `/api/sessions/{date}` | Update session |
| DELETE | `/api/sessions/{date}` | Delete session |
| GET | `/api/notifications` | Get notifications |
| POST | `/api/notifications` | Create notification |
| POST | `/api/notifications/read` | Mark as read |
| POST | `/api/chat` | Global AI chat |
| POST | `/api/chat/contextual` | Session-specific AI chat |
| GET | `/api/status` | Get recording status |
| POST | `/api/status` | Toggle recording |

## Testing

### Frontend Tests
```bash
cd frontend

# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run linting
npm run lint
```

Frontend tests are organized into:
- **Unit tests** (`test/unit/`) - Component behavior testing
- **Property tests** (`test/*.property.test.ts`) - Invariant verification using fast-check

Each property test validates a correctness property from the design specification.

### Backend Tests
```bash
# Run Go tests
go test ./...
```

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl/Cmd + K` | Open search |
| `Esc` | Close modal/panel |

## Configuration

### App Blacklist
Edit `sessions/blacklist.txt` to exclude apps from capture (one app name per line).

### Privacy Mode
Toggle "Private Mode" in Settings to pause all capture.

### Data Retention
Configure retention period in Settings to auto-cleanup old sessions.

## Troubleshooting

### Empty session files
Some applications don't expose text to Windows UI Automation. Standard apps like browsers, VS Code, and Office work best.

### Ollama not responding
Ensure Ollama is running: `ollama serve`

### Build errors
```bash
# Go dependencies
go mod download

# Frontend dependencies
cd frontend && npm install
```

## License

MIT

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request
