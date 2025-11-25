import React from 'react';
import { Session } from '../types';
import { SessionCard } from './SessionCard';
import { ScrollArea } from './ui/scroll-area';
import { Clock, MessageSquare, Layers } from 'lucide-react';

interface ActivityTimelineProps {
  sessions: Session[];
  selectedSessionId: string | null;
  onSelectSession: (id: string) => void;
}

export const ActivityTimeline: React.FC<ActivityTimelineProps> = ({ 
  sessions, 
  selectedSessionId, 
  onSelectSession 
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
      <div className="p-4 border-b border-border">
        <h2 className="font-semibold text-lg flex items-center gap-2">
          <Clock className="w-5 h-5 text-primary" />
          Activity Timeline
        </h2>
      </div>
      
      <div className="flex gap-2 p-2 px-4 border-b border-border/50">
         {/* Sidebar Navigation / Icons as shown in the reference image top left */}
         <div className="p-2 bg-primary/10 rounded-md text-primary">
            <Clock size={20} />
         </div>
         <div className="p-2 hover:bg-accent rounded-md text-muted-foreground cursor-pointer">
            <MessageSquare size={20} />
         </div>
         <div className="p-2 hover:bg-accent rounded-md text-muted-foreground cursor-pointer">
            <Layers size={20} />
         </div>
      </div>

      <ScrollArea className="flex-1">
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
  );
};
