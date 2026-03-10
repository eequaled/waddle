# Waddle v2 — AI-Powered Second Brain

A **Windows desktop application** that silently captures your activity (focused windows, clipboard, and visible text), synthesizes daily sessions, and presents an intelligent interface to refine those into durable knowledge. Privacy-first and fully local.

![Windows](https://img.shields.io/badge/Windows-10%2F11-blue)
![License](https://img.shields.io/badge/License-MIT-green)


**Waddle** is an autonomous Windows activity intelligence agent that silently captures your digital life, synthesizes it into contextual memory, and provides AI-powered tools for recall and knowledge management. Built with privacy-first principles, everything runs locally on your machine.

## Architecture Overview

Waddle implements a **four-layer autonomous agent architecture** 

```
┌─────────────────────────────────────────────────────────────────────┐
│                         WADDLE APPLICATION                           │
│  Wails + Svelte Frontend • Go Backend API • AI Reasoning Engine      │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  MR-win-activity-pipeline (SENSING LAYER) - v1.0.0                 │
│  Intelligent Capture: ETW → UIA → OCR with Entity Extraction       │
│  Performance: 1% CPU • <50ms latency • 98% accuracy                │
└─────────────────────────────────────────────────────────────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ MR-win-etw- │ │ MR-win-uia- ││   OCR (Tess)││ MR-go-entity│
│ tracker     │ │ reader      ││    Engine   ││ -extractor  │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
        │              │              │              │
        └────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  MR-go-ollama-client (PROCESSING LAYER) -                          │
│  Local LLM Integration • Streaming Chat • Embeddings               │
│  Zero dependencies • Sub-100ms response time                       │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  MR-go-lance-vector (MEMORY LAYER) - v1.0.0                        │
│  Vector Database • Semantic Search • Batch Processing              │
│  P99 <20ms on 50k+ vectors • 1,148 queries/sec                     │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  INFRASTRUCTURE LAYER                                                │
│  ├─ MR-svelte-memory-dashboard (UI Components)                    │
│  ├─ MR-go-retention-manager (Data Lifecycle)                      │
│  ├─ MR-go-sqlite-migrator (Schema Management)                     │
│  └─ MR-win-dpapi-vault (Encryption & Security)                    │
└─────────────────────────────────────────────────────────────────────┘
```

## Core Capabilities

### 🎯 Sensing Layer (MR-win-activity-pipeline)

The intelligent capture system orchestrates three data acquisition methods:

- **ETW Kernel Events** (MR-win-etw-tracker)
  - Zero-overhead window focus tracking at 47.95 ns/op
  - Sub-microsecond event processing with zero allocations
  - Graceful polling fallback when ETW unavailable
  - Process lifecycle monitoring

- **UI Automation** (MR-win-uia-reader)
  - STA thread-marshaled COM operations for safety
  - App-specific extractors for 15+ applications
  - Avoids expensive OCR when structured data available
  - Panic recovery and timeout protection

- **OCR Batch Processing**
  - 10-item batches with 500ms timeout
  - Parallel processing for efficiency
  - Fallback detection (knows when OCR is needed)

- **Entity Extraction** (MR-go-entity-extractor)
  - JIRA tickets: `PROJ-123`
  - Hashtags: `#golang`
  - Mentions: `@username`
  - URLs, emails, file paths
  - Case-insensitive deduplication

**Performance**: 98% accuracy at 1% CPU usage vs. pure OCR (85% accuracy, 15% CPU)

### 🧠 Processing Layer (MR-go-ollama-client)

Lightweight LLM integration for local AI:

- **Zero dependencies** - uses only Go stdlib
- **Functional options API** - clean, composable configuration
- **Streaming support** - real-time response processing
- **Embedding generation** - for semantic search integration
- **Built-in summarization** - optimized for activity context

```go
client := ollama.New(ollama.DefaultConfig())
summary, _ := client.Summarize("session context", capturedText)
```

### 🔍 Memory Layer (MR-go-lance-vector)

High-performance semantic search:

- **IVF_PQ indexing** - optimized for Windows
- **Batch operations** - 100 vectors in 87ms
- **Async embedding queue** - non-blocking generation
- **Retention policies** - automatic lifecycle management
- **Ollama integration** - local embedding generation

**Benchmarks**: 1,148 vector searches/second, P99 <20ms on 50k vectors

### 🎨 Interface Layer (MR-svelte-memory-dashboard)

Production-grade Svelte template:

- Svelte 5 + Vite + Vanilla CSS
- Timeline + card-based views
- Global search (Ctrl+K)
- Dark theme support (Glassmorphism)

### 🔒 Security Layer (MR-win-dpapi-vault)

Enterprise-grade encryption:

- **AES-256-GCM** encryption
- **Argon2id** KDF (64MB memory, 4 threads)
- **Windows Credential Manager** integration
- **DPAPI** key protection
- **Key rotation** without data loss
- **Zero plaintext** key storage

### 📊 Data Management

- **MR-go-retention-manager**: Automated cleanup with archive/delete policies
- **MR-go-sqlite-migrator**: State-machine validated schema migrations with rollback
- **MR-go-sqlite-migrator**: Migrated legacy JSON to SQLite (used in production)

## Installation

### Download (Recommended)
1. Go to [Releases](https://github.com/eequaled/waddle/releases)
2. Download `Waddle-x.x.x-Setup.exe` (installer) or `Waddle-x.x.x-Portable.exe`
3. Run and launch from Start Menu

### AI Features (Optional)
Waddle's AI requires [Ollama](https://ollama.ai) installed separately:
```bash
ollama serve
ollama pull gemma2:2b  # or llama3, mistral, etc.
```

### Build from Source
```bash
# Clone
git clone https://github.com/eequaled/waddle.git && cd waddle

# Development
wails dev

# Build (Native Executable)
wails build
```

## Configuration

### Data Storage
All data stored locally at `~/Documents/Waddle/`:
```
sessions/          # Daily captured sessions (SQLite)
archives/          # Archived collections
global_chats/      # AI chat history
profile/           # User profile data
```

### App Blacklist
Edit `~/Documents/Waddle/sessions/blacklist.txt` to exclude sensitive apps:
```
KeePass.exe
LastPass.exe
```

### Command-Line Options
```bash
waddle-backend.exe -data-dir "D:\Waddle" -port 9090
```

## The MR Micro-Repository Ecosystem

Each MR project is a **battle-tested, production-ready library** extracted from Waddle:

| Repository | Purpose | Key Features |
|------------|---------|--------------|
| **MR-win-activity-pipeline** | Intelligent capture orchestration | ETW/UIA/OCR hybrid, entity extraction, 98% accuracy |
| **MR-win-etw-tracker** | Kernel-level event tracking | Zero-allocation, 47.95 ns/op, graceful fallback |
| **MR-win-uia-reader** | Structured data extraction | STA thread-safe, 15+ app extractors, OCR detection |
| **MR-go-ollama-client** | Local LLM integration | Zero deps, streaming, embeddings, sub-100ms |
| **MR-go-lance-vector** | Vector search engine | P99 <20ms, batch operations, 1,148 qps |
| **MR-go-entity-extractor** | Context extraction | JIRA, hashtags, mentions, deduplication |
| **MR-go-retention-manager** | Data lifecycle | Archive/delete policies, compression |
| **MR-go-sqlite-migrator** | Schema management | State machine, rollback, checksums |
| **MR-win-dpapi-vault** | Encryption | AES-256-GCM, Argon2id, Credential Manager |
| **MR-react-memory-dashboard** | UI components | React 19, Radix UI, TipTap, Recharts |

## Use Cases

### 1. Personal Knowledge Management
Automatically capture and search everything you do:
- "What was that JIRA ticket I was working on Tuesday?"
- "Show me all my golang research sessions"
- "Summarize my week in VS Code"

### 2. Privacy-First AI Assistant
Chat with AI grounded in your actual activity:
- Context-aware responses based on real sessions
- Local processing - no data leaves your machine
- Semantic search across captured content

### 3. Productivity Analytics
Understand your work patterns:
- App usage visualization
- Time tracking by project
- Distraction detection

### 4. Automation Integration
Use with N8n, Zapier, or custom agents:
```javascript
// N8n webhook receives activity events
if (event.app === "Slack" && event.duration > 1800) {
  // Trigger automation after 30min in Slack
}
```

## Performance Characteristics

| Metric | Value | Component |
|--------|-------|-----------|
| Event latency | <50ms | MR-win-activity-pipeline |
| CPU usage | ~1% | MR-win-activity-pipeline |
| Search P99 | <20ms | MR-go-lance-vector |
| Vector QPS | 1,148 | MR-go-lance-vector |
| ETW throughput | 20M events/sec | MR-win-etw-tracker |
| Memory per op | 0 B (zero-allocation) | MR-win-etw-tracker |

## Development

### Project Structure
```
├── frontend/               # Svelte dashboard
├── build/                  # Wails build output
├── main.go                 # Application entry point
├── app.go                  # Subsystem orchestration
├── wails.json              # Wails configuration
├── pkg/
│   ├── platform/           # Platform abstraction (ETW/UIA)
└── profile/                # Default assets
```

### Testing
```bash
# Unit tests
go test ./...

# Benchmarks
go test -bench=. -benchmem

# Race detection
go test -race ./...
```

## Troubleshooting

**Sessions not appearing?**
- Wait 30 seconds for first capture cycle
- Check Private Mode isn't enabled (system tray icon)
- Verify app isn't in blacklist

**AI chat not working?**
- Ensure Ollama is running: `ollama serve`
- Pull model: `ollama pull gemma2:2b`

**High CPU usage?**
- Reduce screenshot frequency in settings
- Add more apps to blacklist
- Check OCR isn't running continuously

## Contributing

Waddle welcomes contributions! Areas of interest:
- Additional app-specific extractors for MR-win-uia-reader
- New entity types for MR-go-entity-extractor
- Performance optimizations for MR-go-lance-vector
- UI improvements for MR-react-memory-dashboard

## License

MIT License - see [LICENSE](https://github.com/eequaled/waddle/blob/main/LICENSE) for details.

---

**Built with ❤️ for knowledge workers who want to remember everything—privately.**
