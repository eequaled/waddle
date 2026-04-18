import React, { useState } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/card';
import { Input } from '../components/ui/input';
import { useSessions } from '../hooks/useStorage';
import { Search } from 'lucide-react';

export function Sessions({ onSelectSession }: { onSelectSession: (id: string) => void }) {
  const { sessions, loading } = useSessions();
  const [searchQuery, setSearchQuery] = useState('');

  const filteredSessions = sessions.filter(session => {
    const q = searchQuery.toLowerCase();
    return session.date.includes(q) || 
           (session.title || '').toLowerCase().includes(q) || 
           (session.summary || '').toLowerCase().includes(q) ||
           (session.aiSummary || '').toLowerCase().includes(q);
  });

  if (loading) {
    return <div className="p-8">Loading sessions...</div>;
  }

  return (
    <div className="flex-1 p-8 overflow-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold">All Sessions</h1>
        <div className="relative w-72">
          <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input 
            placeholder="Search sessions..." 
            className="pl-8" 
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredSessions.map(session => (
          <Card 
            key={session.id} 
            className="cursor-pointer hover:border-primary/50 transition-colors"
            onClick={() => onSelectSession(session.id)}
          >
            <CardHeader className="pb-2">
              <CardTitle className="text-lg">{session.customTitle || session.title || session.date}</CardTitle>
              <div className="text-xs text-muted-foreground">{session.date}</div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground line-clamp-3">
                {session.customSummary || session.aiSummary || session.summary || 'No summary available'}
              </p>
            </CardContent>
          </Card>
        ))}
        {filteredSessions.length === 0 && (
          <div className="col-span-full text-center p-8 text-muted-foreground">
            No sessions found matching your search.
          </div>
        )}
      </div>
    </div>
  );
}
