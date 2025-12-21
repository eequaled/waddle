# Waddle — AI-Powered Second Brain

A **Windows desktop application** that silently captures your activity (focused windows, clipboard, and visible text), synthesizes daily sessions, and presents an intelligent interface to refine those into durable knowledge. Privacy-first and fully local.

![Windows](https://img.shields.io/badge/Windows-10%2F11-blue)
![License](https://img.shields.io/badge/License-MIT-green)

## ✨ Features

### Automatic Activity Capture
- **Window Tracking** — Monitors active windows and applications every 500ms
- **Text Extraction** — Captures visible text via Windows UI Automation + OCR
- **Clipboard History** — Tracks clipboard changes with timestamps
- **Screenshot Capture** — Periodic screenshots with automatic OCR processing

### Second Brain Tools
- **Session Editing** — Edit titles, summaries, and add manual notes
- **AI Chat** — Chat with AI grounded in specific session context (requires Ollama)
- **Search** — Deep linking to specific sessions and blocks with text highlighting
- **Insights** — Visualize daily/weekly app usage patterns

### Privacy & Control
- **App Blacklist** — Exclude sensitive apps from capture
- **Private Mode** — Pause all capture with one click
- **Local Storage** — All data stays on your machine
- **Data Retention** — Auto-cleanup old sessions

## 📦 Installation

### Download (Recommended)
1. Go to [Releases](../../releases)
2. Download `Waddle-x.x.x-Setup.exe`
3. Run the installer
4. Launch Waddle from Start Menu

**Portable Version:** Download `Waddle-x.x.x-Portable.exe` to run without installation.

### AI Features (Optional)
Waddle's AI chat requires [Ollama](https://ollama.ai) installed separately:

```bash
# Install Ollama from https://ollama.ai, then:
ollama serve
ollama pull gemma2:2b
```




## everything under is VERY outdated for now 


## 🖥️ System Requirements

- **OS:** Windows 10/11 (required for UI Automation APIs)
- **RAM:** 4GB minimum, 8GB recommended
- **Storage:** ~200MB for installation + session data
- **Optional:** Ollama for AI features

## 📂 Data Storage

All session data is stored locally:
```
C:\Users\<You>\Documents\Waddle\
├── sessions/          # Daily captured sessions
├── archives/          # Archived session collections
├── global_chats/      # AI chat history
└── profile/           # Profile images
```

## ⌨️ Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl + K` | Open search |
| `Esc` | Close modal/panel |

## ⚙️ Configuration

### Exclude Apps from Capture
Edit the blacklist in Settings or directly at:
`Documents\Waddle\sessions\blacklist.txt`

Add one app name per line (e.g., `KeePass.exe`).

### Command-Line Options
```bash
waddle-backend.exe -data-dir "D:\MyData\Waddle" -port 9090
```

| Flag | Description | Default |
|------|-------------|---------|
| `-data-dir` | Custom data directory | `~/Documents/Waddle` |
| `-port` | API server port | `8080` |

## 🔧 Build from Source

### Prerequisites
- Go 1.20+
- Node.js 18+
- Windows 10/11

### Build Steps
```bash
# Clone
git clone https://github.com/eequaled/waddle.git
cd waddle

# Build backend
go build -o waddle-backend.exe

# Build frontend
cd frontend && npm install && npm run build && cd ..

# Build Electron installer
cd electron && npm install && npm run build:win
```

Output: `dist-electron/Waddle-x.x.x-Setup.exe`

## 📁 Project Structure

```
waddle/
├── main.go                 # Go backend entry point
├── pkg/                    # Backend packages
│   ├── ai/                 # Ollama AI client
│   ├── capture/            # Screenshot capture
│   ├── content/            # Clipboard & UI automation
│   ├── ocr/                # Text extraction (Tesseract)
│   ├── processing/         # Batch processing
│   ├── server/             # HTTP API server
│   ├── storage/            # File system operations
│   └── tracker/            # Window focus tracking
├── frontend/               # React UI (built into Electron)
├── electron/               # Electron wrapper
│   ├── main.js             # Electron main process
│   └── package.json        # Build configuration
└── profile/                # Default profile images
```

## 🐛 Troubleshooting

### App shows blank screen
- Ensure antivirus isn't blocking the app
- Try the portable version instead of installer
- Check if another instance is running

### Sessions not appearing
- Wait 30+ seconds for first capture
- Make sure the app isn't in Private Mode (check system tray)
- Some apps don't expose text—browsers and VS Code work best

### AI chat not working
- Ollama must be installed and running: `ollama serve`
- Pull the model first: `ollama pull gemma2:2b`

### OCR not extracting text
- Tesseract is bundled—no action needed
- Very small or stylized text may not extract well

## 📜 License

MIT License - see [LICENSE](LICENSE)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

---

**Made with ❤️ for knowledge workers who want to remember everything.**
