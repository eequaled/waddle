import React, { useState, useEffect } from 'react';
import { Session, ManualNote, ContentBlock } from '../types';
import { AppIcon } from './AppIcon';
import { BlockRenderer } from './BlockRenderer';
import { SessionActionsMenu } from './SessionActionsMenu';
import { ScrollArea } from './ui/scroll-area';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Textarea } from './ui/textarea';
import { Edit3, Sparkles, Save, X, Plus, StickyNote } from 'lucide-react';

interface MainEditorProps {
  session: Session | null;
  onOpenSearch: () => void;
  onOpenContextualChat?: (session: Session, block?: any) => void;
  onSessionUpdate?: (session: Session) => void;
  onSessionDelete?: (sessionId: string) => void;
  searchHighlightQuery?: string;
  searchTargetBlockId?: string;
}

interface EditState {
  title: string;
  summary: string;
  manualNotes: ManualNote[];
}

// Helper component to highlight search matches
const HighlightedText: React.FC<{ text: string; query: string }> = ({ text, query }) => {
  if (!query.trim()) return <>{text}</>;
  
  const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
  const parts = text.split(regex);
  
  return (
    <>
      {parts.map((part, i) => 
        regex.test(part) ? (
          <mark key={i} className="bg-yellow-500/40 text-foreground rounded px-0.5">
            {part}
          </mark>
        ) : part
      )}
    </>
  );
};

export const MainEditor: React.FC<MainEditorProps> = ({ 
  session, 
  onOpenSearch,
  onOpenContextualChat,
  onSessionUpdate,
  onSessionDelete,
  searchHighlightQuery = '',
  searchTargetBlockId,
}) => {
  const [isEditMode, setIsEditMode] = useState(false);
  const [editState, setEditState] = useState<EditState>({
    title: '',
    summary: '',
    manualNotes: [],
  });

  // Scroll to target block when searchTargetBlockId changes
  useEffect(() => {
    if (searchTargetBlockId) {
      // Small delay to ensure DOM is rendered
      setTimeout(() => {
        const element = document.getElementById(`block-${searchTargetBlockId}`);
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'center' });
          // Add a highlight effect
          element.classList.add('ring-2', 'ring-yellow-500', 'ring-offset-2');
          setTimeout(() => {
            element.classList.remove('ring-2', 'ring-yellow-500', 'ring-offset-2');
          }, 3000);
        }
      }, 100);
    }
  }, [searchTargetBlockId, session]);

  // Reset edit state when session changes or edit mode is entered
  useEffect(() => {
    if (session) {
      setEditState({
        title: session.customTitle || session.title,
        summary: session.customSummary || session.summary,
        manualNotes: session.manualNotes || [],
      });
    }
  }, [session, isEditMode]);

  const handleEnterEditMode = () => {
    if (session) {
      setEditState({
        title: session.customTitle || session.title,
        summary: session.customSummary || session.summary,
        manualNotes: session.manualNotes || [],
      });
      setIsEditMode(true);
    }
  };

  const handleCancelEdit = () => {
    // Restore original values - discard all changes
    if (session) {
      setEditState({
        title: session.customTitle || session.title,
        summary: session.customSummary || session.summary,
        manualNotes: session.manualNotes || [],
      });
    }
    setIsEditMode(false);
  };

  const handleSaveEdit = () => {
    if (session && onSessionUpdate) {
      const updatedSession: Session = {
        ...session,
        customTitle: editState.title !== session.title ? editState.title : session.customTitle,
        customSummary: editState.summary !== session.summary ? editState.summary : session.customSummary,
        originalSummary: session.originalSummary || session.summary,
        manualNotes: editState.manualNotes,
      };
      onSessionUpdate(updatedSession);
    }
    setIsEditMode(false);
  };

  const handleAddNote = () => {
    const newNote: ManualNote = {
      id: `note-${Date.now()}`,
      content: '',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    // Insert at beginning (index 0)
    setEditState(prev => ({
      ...prev,
      manualNotes: [newNote, ...prev.manualNotes],
    }));
  };

  const handleUpdateNote = (noteId: string, content: string) => {
    setEditState(prev => ({
      ...prev,
      manualNotes: prev.manualNotes.map(note =>
        note.id === noteId
          ? { ...note, content, updatedAt: new Date().toISOString() }
          : note
      ),
    }));
  };

  const handleDeleteNote = (noteId: string) => {
    setEditState(prev => ({
      ...prev,
      manualNotes: prev.manualNotes.filter(note => note.id !== noteId),
    }));
  };

  const handleOpenContextualChat = () => {
    if (session && onOpenContextualChat) {
      onOpenContextualChat(session);
    }
  };

  if (!session) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground bg-background">
        <div className="max-w-md text-center space-y-4">
          <h2 className="text-2xl font-semibold text-foreground">Select a Session</h2>
          <p>Choose an activity from the timeline to view its auto-summary and details.</p>
          <Button variant="outline" onClick={onOpenSearch}>
            Search Memories (Cmd+K)
          </Button>
        </div>
      </div>
    );
  }

  // Get display title and summary (prefer custom if set)
  const displayTitle = session.customTitle || session.title;
  const displaySummary = session.customSummary || session.summary;

  // Build content with manual notes at the top
  const contentWithNotes: ContentBlock[] = [
    // Convert manual notes to content blocks
    ...(session.manualNotes || []).map(note => ({
      id: note.id,
      type: 'manual-note' as const,
      content: note.content,
      data: { createdAt: note.createdAt, updatedAt: note.updatedAt },
    })),
    // Then the regular content
    ...session.content,
  ];


  return (
    <div className="flex-1 flex flex-col h-full bg-background relative">
      {/* Top Bar / Header Area */}
      <div className="px-8 py-6 border-b border-border/40">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span className="bg-accent/50 px-2 py-1 rounded text-xs font-medium">{session.date}</span>
            <span>•</span>
            <span>{session.startTime} - {session.endTime}</span>
            <span>•</span>
            <span>{session.duration}</span>
          </div>
          <div className="flex items-center gap-2">
            {isEditMode ? (
              <>
                <Button 
                  variant="ghost" 
                  size="sm" 
                  onClick={handleCancelEdit}
                  className="gap-1 text-muted-foreground"
                >
                  <X size={16} />
                  Cancel
                </Button>
                <Button 
                  variant="default" 
                  size="sm" 
                  onClick={handleSaveEdit}
                  className="gap-1"
                >
                  <Save size={16} />
                  Save
                </Button>
              </>
            ) : (
              <>
                <Button 
                  variant="ghost" 
                  size="icon" 
                  className="h-8 w-8"
                  onClick={handleEnterEditMode}
                >
                  <Edit3 size={16} />
                </Button>
                <SessionActionsMenu 
                  session={session}
                  onSessionDelete={onSessionDelete}
                />
              </>
            )}
          </div>
        </div>

        {/* Title - Editable in edit mode */}
        {isEditMode ? (
          <Input
            value={editState.title}
            onChange={(e) => setEditState(prev => ({ ...prev, title: e.target.value }))}
            className="text-3xl font-bold mb-4 h-auto py-2 border-dashed"
            placeholder="Session title..."
          />
        ) : (
          <h1 className="text-3xl font-bold mb-4 text-foreground leading-tight">
            <HighlightedText text={displayTitle} query={searchHighlightQuery} />
          </h1>
        )}

        <div className="flex items-center gap-3">
          <div className="flex -space-x-2 overflow-hidden">
            {session.apps.map((app, i) => (
              <div key={i} className="rounded-full bg-background p-0.5 ring-2 ring-background">
                <div className="bg-muted/30 rounded-full p-1.5">
                  <AppIcon app={app} className="w-4 h-4" />
                </div>
              </div>
            ))}
          </div>
          <div className="h-4 w-px bg-border mx-2"></div>
          <div className="flex gap-2">
            {session.tags.map(tag => (
              <span key={tag} className="text-xs text-muted-foreground bg-muted/50 px-2 py-0.5 rounded-full">
                #{tag}
              </span>
            ))}
          </div>
        </div>
      </div>


      {/* Editor Content - Wrapped in container with min-h-0 to enable scrolling */}
      <div className="flex-1 min-h-0">
        <ScrollArea className="h-full">
          <div className="max-w-3xl mx-auto px-8 py-8 pb-24">
            {/* Summary Section - Editable in edit mode */}
            {isEditMode ? (
              <div className="mb-6">
                <label className="text-sm font-medium text-muted-foreground mb-2 block">
                  AI Summary (editable)
                </label>
                <Textarea
                  value={editState.summary}
                  onChange={(e) => setEditState(prev => ({ ...prev, summary: e.target.value }))}
                  className="min-h-[100px] border-dashed"
                  placeholder="Session summary..."
                />
              </div>
            ) : (
              <div className="mb-6 p-4 bg-muted/30 rounded-lg border border-border/50">
                <p className="text-muted-foreground">
                  <HighlightedText text={displaySummary} query={searchHighlightQuery} />
                </p>
              </div>
            )}

            {/* Add Note Button - Only in edit mode */}
            {isEditMode && (
              <div className="mb-6">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleAddNote}
                  className="gap-2 border-dashed"
                >
                  <Plus size={16} />
                  Add Note
                </Button>
              </div>
            )}

            {/* Manual Notes in Edit Mode */}
            {isEditMode && editState.manualNotes.map((note) => (
              <div key={note.id} className="mb-4 p-4 bg-yellow-500/10 rounded-lg border border-yellow-500/30">
                <div className="flex items-start gap-2">
                  <StickyNote size={16} className="text-yellow-500 mt-1 shrink-0" />
                  <div className="flex-1">
                    <Textarea
                      value={note.content}
                      onChange={(e) => handleUpdateNote(note.id, e.target.value)}
                      className="min-h-[60px] bg-transparent border-none p-0 resize-none focus-visible:ring-0"
                      placeholder="Write your note..."
                    />
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 text-muted-foreground hover:text-destructive"
                    onClick={() => handleDeleteNote(note.id)}
                  >
                    <X size={14} />
                  </Button>
                </div>
              </div>
            ))}

            {/* Content Blocks */}
            {isEditMode ? (
              // In edit mode, show regular content (notes are shown above)
              session.content.map(block => (
                <BlockRenderer key={block.id} block={block} />
              ))
            ) : (
              // In view mode, show content with notes integrated
              contentWithNotes.map(block => (
                <BlockRenderer 
                  key={block.id} 
                  block={block} 
                  onAskAI={(memBlock) => {
                    if (onOpenContextualChat) {
                      onOpenContextualChat(session, memBlock);
                    }
                  }}
                />
              ))
            )}

            {/* Placeholder for adding new content */}
            {!isEditMode && (
              <div className="mt-8 group cursor-text opacity-50 hover:opacity-100 transition-opacity">
                <div className="flex items-center gap-2 text-muted-foreground text-sm">
                  <span>Type '/' for commands</span>
                </div>
              </div>
            )}
          </div>
        </ScrollArea>
      </div>

      {/* Floating Action Button - Contextual Chat */}
      {!isEditMode && (
        <div className="absolute bottom-8 right-8">
          <Button 
            className="shadow-lg rounded-full h-12 w-12 p-0 bg-primary text-primary-foreground hover:bg-primary/90"
            onClick={handleOpenContextualChat}
          >
            <Sparkles size={20} />
          </Button>
        </div>
      )}
    </div>
  );
};
