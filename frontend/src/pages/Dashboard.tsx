import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { ScrollArea } from '../components/ui/scroll-area';
import { useSessions, useAppDetails } from '../hooks/useStorage';
import { ActivityBlock } from '../components/ActivityBlock';

export function Dashboard() {
  const { sessions, loading: sessionsLoading } = useSessions();
  const today = new Date().toISOString().split('T')[0];
  const todaySession = sessions?.find(s => s.date === today);

  const { appDetails, loading: detailsLoading } = useAppDetails(todaySession?.date || today);

  if (sessionsLoading) {
    return <div className="p-8">Loading dashboard...</div>;
  }

  return (
    <div className="flex-1 p-8 overflow-auto">
      <h1 className="text-3xl font-bold mb-6">Activity Dashboard</h1>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <Card>
          <CardHeader>
            <CardTitle>Today's Summary</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              {todaySession?.aiSummary || todaySession?.summary || 'No activity recorded yet today.'}
            </p>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <CardTitle>Quick Stats</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-sm">Total Sessions</span>
                <span className="font-bold">{sessions.length}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm">Apps Used Today</span>
                <span className="font-bold">{appDetails?.length || 0}</span>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>App Activity Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-[120px]">
              {detailsLoading ? (
                <p className="text-sm text-muted-foreground">Loading...</p>
              ) : appDetails && appDetails.length > 0 ? (
                <div className="space-y-2 pr-4">
                  {appDetails.map((app, i) => (
                    <div key={i} className="flex justify-between text-sm">
                      <span className="truncate max-w-[120px]">{app.appName}</span>
                      <span className="text-muted-foreground">{app.blockCount} blocks</span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No data</p>
              )}
            </ScrollArea>
          </CardContent>
        </Card>
      </div>

      <h2 className="text-xl font-semibold mb-4">Recent Activity (Placeholder)</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <ActivityBlock 
          appName="Cursor"
          startTime="10:00"
          endTime="10:15"
          microSummary="Editing frontend components"
          captureSource="ETW"
        />
        <ActivityBlock 
          appName="Google Chrome"
          startTime="10:15"
          endTime="10:30"
          microSummary="Reading documentation"
          captureSource="UIA"
        />
      </div>
    </div>
  );
}
