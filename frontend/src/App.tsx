import React, { useState, useEffect } from 'react';
import { ActivityTimeline } from './components/ActivityTimeline';
import { MainEditor } from './components/MainEditor';
import { SearchModal } from './components/SearchModal';
import { SettingsModal } from './components/SettingsModal';
import { Toaster } from './components/ui/sonner';
import { User, Settings, Bell, Play, Pause, Circle } from 'lucide-react';
import { Button } from './components/ui/button';
import { api, transformToSession } from './services/api';
import { Session } from './types';

function App() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isPaused, setIsPaused] = useState(false);

  // Theme State
  const [theme, setTheme] = useState<'light' | 'dark'>('dark');
  const [isDuckMode, setIsDuckMode] = useState(false);
  const [isSunset, setIsSunset] = useState(false);
  const [themeSwitchCount, setThemeSwitchCount] = useState(0);
  const [lastSwitchTime, setLastSwitchTime] = useState(0);

  // Apply Theme
  useEffect(() => {
    const root = window.document.documentElement;
    root.classList.remove('light', 'dark', 'duck');
    root.classList.add(theme);
    if (isDuckMode) {
      root.classList.add('duck');
    }
    if (isSunset) {
      root.classList.add('sunset');
    } else {
      root.classList.remove('sunset');
    }
  }, [theme, isDuckMode, isSunset]);

  const handleThemeChange = (newTheme: 'light' | 'dark') => {
    setTheme(newTheme);

    // Easter Egg Logic (Only in Duck Mode)
    if (isDuckMode) {
      const now = Date.now();
      if (now - lastSwitchTime < 1000) {
        const newCount = themeSwitchCount + 1;
        setThemeSwitchCount(newCount);
        if (newCount >= 5 && !isSunset) {
          setIsSunset(true);
          // Play Quack Sound
          const audio = new Audio('/assets/sounds/quack.mp3');
          audio.play().catch(console.error);
        }
      } else {
        setThemeSwitchCount(0);
      }
      setLastSwitchTime(now);
    }
  };

  // Fetch Status on Mount
  useEffect(() => {
    api.getStatus().then(status => setIsPaused(status.paused)).catch(console.error);
  }, []);

  const toggleStatus = async () => {
    try {
      const newStatus = await api.setStatus(!isPaused);
      setIsPaused(newStatus.paused);
    } catch (error) {
      console.error("Failed to toggle status:", error);
    }
  };

  // Fetch Sessions on Mount
  useEffect(() => {
    console.log('[DEBUG] useEffect triggered - starting to load sessions');
    const loadSessions = async () => {
      try {
        console.log('[DEBUG] Fetching session dates...');
        const dates = await api.getSessions();
        console.log('[DEBUG] Received dates:', dates);

        console.log('[DEBUG] Transforming sessions...');
        const loadedSessions = await Promise.all(dates.map(date => transformToSession(date)));
        console.log('[DEBUG] Loaded sessions:', loadedSessions.length, loadedSessions);

        setSessions(loadedSessions);
        if (loadedSessions.length > 0) {
          setSelectedSessionId(loadedSessions[0].id);
          console.log('[DEBUG] Set initial session ID:', loadedSessions[0].id);
        }
      } catch (error) {
        console.error("[ERROR] Failed to load sessions:", error);
      }
    };
    loadSessions();
  }, []);

  const selectedSession = sessions.find(s => s.id === selectedSessionId) || null;

  // Global Hotkey for Search (Cmd+K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setIsSearchOpen(prev => !prev);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Force dark mode for the prototype to match the design
  return (
    <div className="h-screen w-full">
      <div className="flex flex-col h-screen bg-background text-foreground overflow-hidden font-sans antialiased">

        {/* Top Navigation Bar */}
        <header className="h-14 border-b border-border flex items-center justify-between px-4 bg-background shrink-0 z-10">
          <div className="flex items-center gap-4 w-1/3">
            <div className="w-8 h-8 bg-primary rounded-md flex items-center justify-center text-primary-foreground font-bold text-lg">
              M
            </div>
            <div className="relative max-w-md w-full hidden md:block">
              <div
                className="flex items-center px-3 py-1.5 rounded-md border border-input bg-muted/30 text-sm text-muted-foreground cursor-pointer hover:bg-accent/50 transition-colors"
                onClick={() => setIsSearchOpen(true)}
              >
                <span>Search memories...</span>
                <span className="ml-auto text-xs opacity-50">Cmd+K</span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant={isPaused ? "outline" : "default"}
              size="sm"
              className={`gap-2 ${isPaused ? "border-red-500 text-red-500 hover:bg-red-500/10" : "bg-green-600 hover:bg-green-700"}`}
              onClick={toggleStatus}
            >
              {isPaused ? (
                <>
                  <Circle size={14} className="fill-current" />
                  Stopped
                </>
              ) : (
                <>
                  <Circle size={14} className="fill-current animate-pulse" />
                  Recording
                </>
              )}
            </Button>

            <Button variant="ghost" size="icon" className="text-muted-foreground">
              <Bell size={18} />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="text-muted-foreground"
              onClick={() => setIsSettingsOpen(true)}
            >
              <Settings size={18} />
            </Button>
            <div className="h-8 w-8 rounded-full bg-accent ml-2 flex items-center justify-center text-accent-foreground border border-border">
              <User size={16} />
            </div>
          </div>
        </header>

        {/* Main Content Area */}
        <div className="flex flex-1 overflow-hidden">
          <ActivityTimeline
            sessions={sessions}
            selectedSessionId={selectedSessionId}
            onSelectSession={setSelectedSessionId}
          />

          <MainEditor
            session={selectedSession}
            onOpenSearch={() => setIsSearchOpen(true)}
          />
        </div>

        {/* Search Modal Overlay */}
        <SearchModal
          isOpen={isSearchOpen}
          onClose={() => setIsSearchOpen(false)}
          sessions={sessions}
          onSelectSession={setSelectedSessionId}
        />

        {/* Settings Modal Overlay */}
        <SettingsModal
          isOpen={isSettingsOpen}
          onClose={() => setIsSettingsOpen(false)}
          currentTheme={theme}
          onThemeChange={handleThemeChange}
          isDuckMode={isDuckMode}
          onDuckModeChange={setIsDuckMode}
        />

        <Toaster />
      </div>
    </div>
  );
}

export default App;
