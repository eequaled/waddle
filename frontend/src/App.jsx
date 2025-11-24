
import { useState, useEffect } from 'react'
import { Search, Menu, Bell, Settings } from 'lucide-react'
import { SessionCard } from './components/SessionCard'
import { Editor } from './components/Editor'
import { SearchModal } from './components/SearchModal'


const MOCK_SESSIONS = [
  {
    id: 1,
    title: "Researching WebSockets & Real-time Data",
    tags: ["research", "dev"],
    time: "2:30 PM - 4:15 PM",
    apps: ["chrome", "code", "slack"],
    date: "Today"
  },
  {
    id: 2,
    title: "Budget Planning",
    tags: ["finance", "planning"],
    time: "11:00 AM - 12:30 PM",
    apps: ["chrome", "notes", "slack"],
    date: "Today"
  },
  {
    id: 3,
    title: "Designing Landing Page",
    tags: ["design", "figma"],
    time: "Yesterday",
    apps: ["spotify", "chrome"],
    date: "Yesterday"
  }
]

function App() {
  const [activeSession, setActiveSession] = useState(MOCK_SESSIONS[0].id)
  const [searchOpen, setSearchOpen] = useState(false)

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.ctrlKey && e.key === 's') {
        e.preventDefault()
        setSearchOpen(true)
      }
      if (e.key === 'Escape') {
        setSearchOpen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  return (
    <div className="flex h-screen bg-background text-foreground overflow-hidden font-sans">
      <SearchModal isOpen={searchOpen} onClose={() => setSearchOpen(false)} />

      {/* Sidebar - Activity Timeline */}
      <aside className="w-80 border-r border-border bg-card/50 backdrop-blur-xl flex flex-col">
        <div className="p-4 border-b border-border flex items-center justify-between">
          <h2 className="font-semibold text-lg tracking-tight">Activity Timeline</h2>
          <button className="p-2 hover:bg-accent rounded-md transition-colors">
            <Settings className="w-4 h-4 text-muted-foreground" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-6">
          {/* Today */}
          <div className="space-y-3">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider pl-1">Today</h3>
            {MOCK_SESSIONS.filter(s => s.date === "Today").map(session => (
              <SessionCard
                key={session.id}
                {...session}
                isActive={activeSession === session.id}
                onClick={() => setActiveSession(session.id)}
              />
            ))}
          </div>

          {/* Yesterday */}
          <div className="space-y-3">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider pl-1">Yesterday</h3>
            {MOCK_SESSIONS.filter(s => s.date === "Yesterday").map(session => (
              <SessionCard
                key={session.id}
                {...session}
                isActive={activeSession === session.id}
                onClick={() => setActiveSession(session.id)}
              />
            ))}
          </div>
        </div>
      </aside>

      {/* Main Content - Intelligent Editor */}
      <main className="flex-1 flex flex-col min-w-0 bg-background/95">
        {/* Top Navigation */}
        <header className="h-14 border-b border-border flex items-center justify-between px-6 bg-card/50 backdrop-blur-sm">
          <div className="flex items-center gap-4 flex-1 max-w-xl">
            <div className="relative w-full">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search memories (Ctrl+S)..."
                className="w-full bg-accent/50 border border-transparent focus:border-primary rounded-lg pl-10 pr-4 py-1.5 text-sm outline-none transition-all"
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button className="p-2 hover:bg-accent rounded-full">
              <Bell className="w-5 h-5 text-muted-foreground" />
            </button>
            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-indigo-500 to-purple-500 ml-2" />
          </div>
        </header>

        {/* Editor Area */}
        <div className="flex-1 overflow-y-auto p-8 w-full">
          <Editor />
        </div>
      </main>
    </div>
  )
}

export default App
