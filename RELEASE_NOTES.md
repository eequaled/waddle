# Waddle v1.0.0 - First Release 🎉

**AI-Powered Second Brain for Developers**

Waddle is a local-first memory tool that silently captures your activity, synthesizes sessions, and presents an intelligent interface to refine those into durable knowledge.

## ✨ Features

### Core Capture
- 📸 **Passive Window Tracking** - Monitors active windows every 500ms
- 📝 **OCR Text Extraction** - Captures visible text from screenshots
- 📋 **Clipboard Monitoring** - Tracks clipboard changes with timestamps
- 🖼️ **Screenshot Capture** - Periodic screenshots with automatic processing

### Second Brain Enhancement
- ✏️ **Session Editing** - Edit titles, summaries, and add manual notes
- 💬 **Contextual AI Chat** - Chat with AI grounded in specific session context
- 🔍 **Enhanced Search** - Deep linking to sessions with text highlighting
- 🔔 **Proactive Notifications** - AI-generated insights about usage patterns
- 🔗 **Related Memories** - Automatic surfacing of similar past sessions
- 📊 **Activity Insights** - Visualize daily/weekly app usage patterns

### Smart Features
- 🏷️ **Semantic Tagging** - Auto-categorize activities (coding, research, communication)
- 🔗 **Memory Linking** - Connect related sessions manually or via AI
- 🔒 **Privacy Controls** - Exclude apps, private mode, data retention settings
- 📤 **Export** - Download sessions as Markdown files
- 👤 **Profile Customization** - Custom profile pictures with carousel

## 📥 Downloads

| File | Description |
|------|-------------|
| `Waddle-1.0.0-Setup.exe` | Windows installer (recommended) |
| `Waddle-1.0.0-Portable.exe` | Portable version (no installation) |

## 📋 Requirements

- **OS**: Windows 10/11
- **Tesseract OCR**: Required for text extraction
  - Download: https://github.com/UB-Mannheim/tesseract/wiki
- **Ollama** (Optional): For AI features
  - Download: https://ollama.ai
  - Model: `ollama pull gemma2:2b`

## 🚀 Quick Start

1. Download and run `Waddle-1.0.0-Setup.exe`
2. Install Tesseract OCR if not already installed
3. Launch Waddle from Start Menu or Desktop
4. The app will start capturing your activity automatically

## ⚙️ Configuration

- **Blacklist Apps**: Settings → Add apps to exclude from capture
- **Private Mode**: Toggle recording on/off from system tray
- **Data Location**: `%USERPROFILE%\OneDrive\Documents\ideathon\sessions\`

## 🐛 Known Issues

- Some applications don't expose text to Windows UI Automation
- Works best with: browsers, VS Code, Office apps, and standard Windows applications

## 📝 Changelog

### v1.0.0 (Initial Release)
- Core capture functionality (screenshots, OCR, clipboard)
- Session management (edit, delete, archive, export)
- AI chat integration (global and contextual)
- Search with deep linking and highlighting
- Notification system with proactive insights
- Activity insights and usage analytics
- Profile customization
- Privacy controls and app blacklisting

---

**Full Documentation**: See README.md in the repository

**Report Issues**: https://github.com/YOUR_USERNAME/ideathon/issues
