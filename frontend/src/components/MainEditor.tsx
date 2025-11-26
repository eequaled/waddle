import React from 'react';
import { Session } from '../types';
import { AppIcon } from './AppIcon';
import { BlockRenderer } from './BlockRenderer';
import { ScrollArea } from './ui/scroll-area';
import { Button } from './ui/button';
import { MoreHorizontal, Edit3, Link2 } from 'lucide-react';

interface MainEditorProps {
  session: Session | null;
  onOpenSearch: () => void;
}

export const MainEditor: React.FC<MainEditorProps> = ({ session, onOpenSearch }) => {
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
            <Button variant="ghost" size="icon" className="h-8 w-8">
              <Edit3 size={16} />
            </Button>
            <Button variant="ghost" size="icon" className="h-8 w-8">
              <MoreHorizontal size={16} />
            </Button>
          </div>
        </div>

        <h1 className="text-3xl font-bold mb-4 text-foreground leading-tight">{session.title}</h1>

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
            {session.content.map(block => (
              <BlockRenderer key={block.id} block={block} />
            ))}

            {/* Placeholder for adding new content */}
            <div className="mt-8 group cursor-text opacity-50 hover:opacity-100 transition-opacity">
              <div className="flex items-center gap-2 text-muted-foreground text-sm">
                <span>Type '/' for commands</span>
              </div>
            </div>
          </div>
        </ScrollArea>
      </div>

      {/* Floating Action Button */}
      <div className="absolute bottom-8 right-8">
        <Button className="shadow-lg rounded-full h-12 w-12 p-0 bg-primary text-primary-foreground hover:bg-primary/90">
          <Link2 size={20} />
        </Button>
      </div>
    </div>
  );
};
