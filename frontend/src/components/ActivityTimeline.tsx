import React from 'react';
import { Session } from '../types';
import { SessionCard } from './SessionCard';
import { ScrollArea } from './ui/scroll-area';
import { Clock, MessageSquare, Layers, BarChart3 } from 'lucide-react';

interface ActivityTimelineProps {
  sessions: Session[];
  selectedSessionId: string | null;
  onSelectSession: (id: string) => void;
  activeView: 'timeline' | 'chat' | 'archives' | 'insights';
  onViewChange: (view: 'timeline' | 'chat' | 'archives' | 'insights') => void;
}

export const ActivityTimeline: React.FC<ActivityTimelineProps> = ({
  sessions,
  selectedSessionId,
  onSelectSession,
  activeView,
  onViewChange
}) => {
  // Group sessions by date
  const groupedSessions = sessions.reduce((acc, session) => {
    const date = session.date;
    if (!acc[date]) acc[date] = [];
    acc[date].push(session);
    return acc;
  }, {} as Record<string, Session[]>);

  return (
    <div className="w-80 border-r border-border bg-muted/10 flex flex-col h-full">
      <div className="p-4 border-b border-border shrink-0">
        <h2 className="font-semibold text-lg flex items-center gap-2">
          <Clock className="w-5 h-5 text-primary" />
          Activity Timeline
        </h2>
      </div>

      <div className="flex gap-2 p-2 px-4 border-b border-border/50 shrink-0">
        {/* Sidebar Navigation */}
        <div
          className={`p-2 rounded-md cursor-pointer transition-colors ${activeView === 'timeline' ? 'bg-primary/10 text-primary' : 'hover:bg-accent text-muted-foreground'}`}
          onClick={() => onViewChange('timeline')}
        >
          <Clock size={20} />
        </div>
        <div
          className={`p-2 rounded-md cursor-pointer transition-colors ${activeView === 'chat' ? 'bg-primary/10 text-primary' : 'hover:bg-accent text-muted-foreground'}`}
          onClick={() => onViewChange('chat')}
        >
          <MessageSquare size={20} />
        </div>
        <div
          className={`p-2 rounded-md cursor-pointer transition-colors ${activeView === 'archives' ? 'bg-primary/10 text-primary' : 'hover:bg-accent text-muted-foreground'}`}
          onClick={() => onViewChange('archives')}
        >
          <Layers size={20} />
        </div>
        <div
          className={`p-2 rounded-md cursor-pointer transition-colors ${activeView === 'insights' ? 'bg-primary/10 text-primary' : 'hover:bg-accent text-muted-foreground'}`}
          onClick={() => onViewChange('insights')}
        >
          <BarChart3 size={20} />
        </div>
      </div>

      {/* Wrap ScrollArea in a flex container with min-h-0 to enable scrolling */}
      <div className="flex-1 min-h-0">
        <ScrollArea className="h-full">
          <div className="p-4 space-y-6">
            {Object.entries(groupedSessions).map(([date, dateSessions]) => (
              <div key={date}>
                <h3 className="text-sm font-medium text-muted-foreground mb-3 pl-1">{date}</h3>
                <div className="space-y-3">
                  {dateSessions.map(session => (
                    <SessionCard
                      key={session.id}
                      session={session}
                      isSelected={selectedSessionId === session.id}
                      onClick={() => onSelectSession(session.id)}
                    />
                  ))}
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
};
