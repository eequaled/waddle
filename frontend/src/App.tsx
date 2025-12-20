import React, { useState, useEffect } from 'react';
import { ActivityTimeline } from './components/ActivityTimeline';
import { MainEditor } from './components/MainEditor';
import { SearchModal } from './components/SearchModal';
import { SettingsModal } from './components/SettingsModal';
import { ChatInterface } from './components/ChatInterface';
import { ArchiveView } from './components/ArchiveView';
import { ContextualChat } from './components/ContextualChat';
import { NotificationPanel, Notification } from './components/NotificationPanel';
import { RelatedMemories } from './components/RelatedMemories';
import { InsightsView } from './components/InsightsView';
import { KnowledgeCardsView } from './components/KnowledgeCardsView';
import { Toaster } from './components/ui/sonner';
import { toast } from 'sonner';
import { Settings, Circle } from 'lucide-react';
import { Button } from './components/ui/button';
import { Logo } from './components/Logo';
import { ProfileMenu } from './components/ProfileMenu';
import { api, getSessionSummary, getFullSession } from './services/api';
import { Session, BlockData } from './types';

function App() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [activeView, setActiveView] = useState<'timeline' | 'chat' | 'archives' | 'insights' | 'knowledge'>('timeline');
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isPaused, setIsPaused] = useState(false);

  // Contextual Chat State
  const [isContextualChatOpen, setIsContextualChatOpen] = useState(false);
  const [contextualChatSession, setContextualChatSession] = useState<Session | null>(null);
  const [contextualChatBlock, setContextualChatBlock] = useState<BlockData | undefined>(undefined);

  // Search highlighting state
  const [searchHighlightQuery, setSearchHighlightQuery] = useState<string>('');
  const [searchTargetBlockId, setSearchTargetBlockId] = useState<string | undefined>(undefined);

  // Notifications state
  const [notifications, setNotifications] = useState<Notification[]>([]);

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

  // Fetch Notifications on Mount and periodically
  useEffect(() => {
    const loadNotifications = async () => {
      try {
        const notifs = await api.getNotifications();
        setNotifications(notifs || []);
      } catch (error) {
        console.error('Failed to load notifications:', error);
      }
    };

    loadNotifications();
    // Poll for new notifications every 30 seconds
    const interval = setInterval(loadNotifications, 30000);
    return () => clearInterval(interval);
  }, []);

  // Create notification when recording status changes
  const prevPausedRef = React.useRef(isPaused);
  useEffect(() => {
    if (prevPausedRef.current !== isPaused) {
      // Status changed, create notification
      api.createNotification({
        type: 'status',
        title: isPaused ? 'Recording Paused' : 'Recording Resumed',
        message: isPaused
          ? 'Screen recording has been paused. Click to resume.'
          : 'Screen recording is now active.',
      }).then(() => {
        // Refresh notifications
        api.getNotifications().then(setNotifications).catch(console.error);
      }).catch(console.error);
    }
    prevPausedRef.current = isPaused;
  }, [isPaused]);

  const toggleStatus = async () => {
    try {
      const newStatus = await api.setStatus(!isPaused);
      setIsPaused(newStatus.paused);
    } catch (error) {
      console.error("Failed to toggle status:", error);
    }
  };

  // Helper function to calculate app usage time and generate insights
  const generateAppUsageInsights = async (loadedSessions: Session[]) => {
    // Calculate total time per app across all sessions for today
    const today = new Date().toISOString().split('T')[0];
    const todaySessions = loadedSessions.filter(s => s.date === today);

    const appUsageMap: Record<string, { totalSeconds: number; sessionRef: string }> = {};

    todaySessions.forEach(session => {
      session.content.forEach(block => {
        if (block.type === 'app-memory' && block.data) {
          const appName = block.data.appName || block.content;
          const blocks = block.data.blocks || [];

          // Estimate time from blocks (each block is roughly 30 seconds of activity)
          const estimatedSeconds = blocks.length * 30;

          if (!appUsageMap[appName]) {
            appUsageMap[appName] = { totalSeconds: 0, sessionRef: session.id };
          }
          appUsageMap[appName].totalSeconds += estimatedSeconds;
        }
      });
    });

    // Generate insights for apps with > 2 hours usage (7200 seconds)
    for (const [appName, usage] of Object.entries(appUsageMap)) {
      if (usage.totalSeconds > 7200) {
        const hours = Math.floor(usage.totalSeconds / 3600);
        const minutes = Math.floor((usage.totalSeconds % 3600) / 60);
        const timeSpent = `${hours}h ${minutes}m`;

        try {
          await api.createNotification({
            type: 'insight',
            title: 'Usage Insight',
            message: `You spent ${timeSpent} on ${appName} today`,
            sessionRef: usage.sessionRef,
            metadata: { appName, timeSpent },
          });
        } catch (error) {
          console.error('Failed to create insight notification:', error);
        }
      }
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

        console.log('[DEBUG] Loading session summaries (Lightweight)...');
        // Load sessions individually so one failure doesn't break all
        const sessionPromises = dates.map(async date => {
          try {
            return await getSessionSummary(date);
          } catch (e) {
            console.error(`[ERROR] Failed to load session summary ${date}:`, e);
            return null;
          }
        });

        const results = await Promise.all(sessionPromises);
        const loadedSessions = results.filter((s): s is Session => s !== null);
        console.log('[DEBUG] Loaded sessions:', loadedSessions.length);

        setSessions(loadedSessions);
        if (loadedSessions.length > 0) {
          // Only set selected if not already set
          setSelectedSessionId(prev => prev || loadedSessions[0].id);

          // Generate proactive insights based on app usage
          // NOTE: usage insights calculation might be inaccurate with lightweight sessions
          // if it relies on 'blocks'. We might skip this or accept it only works after full load.
          // generateAppUsageInsights(loadedSessions); 
        }

      } catch (error) {
        console.error("[ERROR] Failed to load sessions:", error);
      }
    };
    loadSessions();
  }, []);

  const selectedSession = sessions.find(s => s.id === selectedSessionId) || null;

  // Lazy load session content when selected
  useEffect(() => {
    const loadContent = async () => {
      if (selectedSessionId && selectedSession) {
        // If content is empty (and it's not a newly created empty manual session, which we assume has at least 1 block or we check a specific flag)
        // For now, check if content length is 0, assuming valid sessions have at least summary
        if (selectedSession.content.length === 0) {
          console.log(`[DEBUG] Lazy loading content for ${selectedSessionId}...`);
          try {
            // Store current ID to avoid race conditions
            const currentId = selectedSessionId;
            const fullSession = await getFullSession(selectedSession.date);

            // Update state only if we are still on the same session (optional, but good practice)
            // Actually we want to update the cache regardless.
            setSessions(prev => prev.map(s =>
              s.id === currentId ? fullSession : s
            ));
          } catch (e) {
            console.error(`[ERROR] Failed to lazy load session ${selectedSessionId}:`, e);
          }
        }
      }
    };
    loadContent();
  }, [selectedSessionId]);



  // Handle session update from edit mode
  const handleSessionUpdate = async (updatedSession: Session) => {
    try {
      // Update local state immediately for responsiveness
      setSessions(prev => prev.map(s =>
        s.id === updatedSession.id ? updatedSession : s
      ));

      // Persist to backend
      await api.updateSession(updatedSession.date, {
        customTitle: updatedSession.customTitle,
        customSummary: updatedSession.customSummary,
        originalSummary: updatedSession.originalSummary,
        manualNotes: updatedSession.manualNotes,
      });

      toast.success('Session saved successfully');
    } catch (error) {
      console.error('Failed to update session:', error);
      toast.error('Failed to save session');
    }
  };

  // Handle session delete
  const handleSessionDelete = async (sessionId: string) => {
    try {
      const sessionToDelete = sessions.find(s => s.id === sessionId);
      if (!sessionToDelete) return;

      await api.deleteSession(sessionToDelete.date);

      // Remove from local state
      setSessions(prev => prev.filter(s => s.id !== sessionId));

      // Select next available session
      const remainingSessions = sessions.filter(s => s.id !== sessionId);
      if (remainingSessions.length > 0) {
        setSelectedSessionId(remainingSessions[0].id);
      } else {
        setSelectedSessionId(null);
      }

      toast.success('Session deleted');
    } catch (error) {
      console.error('Failed to delete session:', error);
      toast.error('Failed to delete session');
    }
  };

  // Handle opening contextual chat
  const handleOpenContextualChat = (session: Session, block?: BlockData) => {
    setContextualChatSession(session);
    setContextualChatBlock(block);
    setIsContextualChatOpen(true);
  };

  // Handle closing contextual chat
  const handleCloseContextualChat = () => {
    setIsContextualChatOpen(false);
    setContextualChatBlock(undefined);
  };

  // Handle marking notifications as read
  const handleMarkNotificationsRead = async (ids: string[]) => {
    try {
      await api.markNotificationsRead(ids);
      setNotifications(prev =>
        prev.map(n => ids.includes(n.id) ? { ...n, read: true } : n)
      );
    } catch (error) {
      console.error('Failed to mark notifications as read:', error);
    }
  };

  // Handle notification navigation
  const handleNotificationNavigate = (sessionId: string) => {
    setActiveView('timeline');
    setSelectedSessionId(sessionId);
  };

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
            <Logo className="h-10 w-10 text-foreground" />
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

            <NotificationPanel
              notifications={notifications}
              onMarkAsRead={handleMarkNotificationsRead}
              onNavigateToSession={handleNotificationNavigate}
            />
            <Button
              variant="ghost"
              size="icon"
              className="text-muted-foreground"
              onClick={() => setIsSettingsOpen(true)}
            >
              <Settings size={18} />
            </Button>
            <ProfileMenu
              activeView={activeView}
              setActiveView={setActiveView}
              setIsSettingsOpen={setIsSettingsOpen}
            />
          </div>
        </header>

        {/* Main Content Area */}
        <div className="flex flex-1 overflow-hidden">
          <ActivityTimeline
            sessions={sessions}
            selectedSessionId={selectedSessionId}
            onSelectSession={setSelectedSessionId}
            activeView={activeView}
            onViewChange={setActiveView}
          />

          {activeView === 'timeline' && (
            <>
              <MainEditor
                session={selectedSession}
                onOpenSearch={() => setIsSearchOpen(true)}
                onSessionUpdate={handleSessionUpdate}
                onSessionDelete={handleSessionDelete}
                onOpenContextualChat={handleOpenContextualChat}
                searchHighlightQuery={searchHighlightQuery}
                searchTargetBlockId={searchTargetBlockId}
              />
              <RelatedMemories
                currentSession={selectedSession}
                allSessions={sessions}
                onSelectSession={setSelectedSessionId}
              />
            </>
          )}

          {activeView === 'chat' && (
            <ChatInterface allSessions={sessions} />
          )}

          {activeView === 'archives' && (
            <ArchiveView />
          )}

          {activeView === 'insights' && (
            <InsightsView sessions={sessions} />
          )}

          {activeView === 'knowledge' && (
            <KnowledgeCardsView 
              onCardClick={(sessionId) => {
                setActiveView('timeline');
                setSelectedSessionId(sessionId);
              }}
            />
          )}
        </div>

        {/* Search Modal Overlay */}
        <SearchModal
          isOpen={isSearchOpen}
          onClose={() => {
            setIsSearchOpen(false);
            // Clear highlight after a delay to allow viewing
            setTimeout(() => {
              setSearchHighlightQuery('');
              setSearchTargetBlockId(undefined);
            }, 5000);
          }}
          sessions={sessions}
          onSelectSession={(id, query, blockId) => {
            setSelectedSessionId(id);
            setSearchHighlightQuery(query || '');
            setSearchTargetBlockId(blockId);
          }}
          onSetActiveView={setActiveView}
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

        {/* Contextual Chat */}
        {contextualChatSession && (
          <ContextualChat
            session={contextualChatSession}
            initialBlock={contextualChatBlock}
            isOpen={isContextualChatOpen}
            onClose={handleCloseContextualChat}
          />
        )}

        <Toaster />
      </div>
    </div>
  );
}

export default App;
