import React, { useMemo } from 'react';
import { Session } from '../types';
import { ScrollArea } from './ui/scroll-area';
import { Card } from './ui/card';
import { 
  BarChart3, 
  Clock, 
  Calendar,
  TrendingUp,
  AppWindow
} from 'lucide-react';
import { AppIcon } from './AppIcon';

interface InsightsViewProps {
  sessions: Session[];
}

interface AppUsageData {
  app: string;
  totalBlocks: number;
  sessions: number;
  lastUsed: string;
}

interface DailyActivity {
  date: string;
  dayOfWeek: string;
  sessionCount: number;
  totalBlocks: number;
  apps: string[];
}

export const InsightsView: React.FC<InsightsViewProps> = ({ sessions }) => {
  // Calculate app usage statistics
  const appUsageStats = useMemo(() => {
    const appMap: Record<string, AppUsageData> = {};
    
    sessions.forEach(session => {
      session.apps.forEach(app => {
        if (!appMap[app]) {
          appMap[app] = { app, totalBlocks: 0, sessions: 0, lastUsed: session.date };
        }
        appMap[app].sessions += 1;
        
        // Count blocks for this app
        const appBlocks = session.content.filter(
          c => c.type === 'app-memory' && (c.content === app || c.data?.appName === app)
        );
        appMap[app].totalBlocks += appBlocks.length;
        
        // Update last used
        if (session.date > appMap[app].lastUsed) {
          appMap[app].lastUsed = session.date;
        }
      });
    });
    
    return Object.values(appMap).sort((a, b) => b.totalBlocks - a.totalBlocks);
  }, [sessions]);

  // Calculate daily activity for the last 7 days
  const dailyActivity = useMemo(() => {
    const days: DailyActivity[] = [];
    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    
    for (let i = 6; i >= 0; i--) {
      const date = new Date();
      date.setDate(date.getDate() - i);
      const dateStr = date.toISOString().split('T')[0];
      
      const daySessions = sessions.filter(s => s.date === dateStr);
      const apps = new Set<string>();
      let totalBlocks = 0;
      
      daySessions.forEach(s => {
        s.apps.forEach(app => apps.add(app));
        totalBlocks += s.content.filter(c => c.type === 'app-memory').length;
      });
      
      days.push({
        date: dateStr,
        dayOfWeek: dayNames[date.getDay()],
        sessionCount: daySessions.length,
        totalBlocks,
        apps: Array.from(apps),
      });
    }
    
    return days;
  }, [sessions]);

  // Calculate productivity metrics
  const productivityMetrics = useMemo(() => {
    const totalSessions = sessions.length;
    const totalApps = new Set(sessions.flatMap(s => s.apps)).size;
    const avgAppsPerSession = totalSessions > 0 
      ? (sessions.reduce((sum, s) => sum + s.apps.length, 0) / totalSessions).toFixed(1)
      : '0';
    
    // Find most productive day
    const dayActivity: Record<string, number> = {};
    sessions.forEach(s => {
      const day = new Date(s.date).toLocaleDateString('en-US', { weekday: 'long' });
      dayActivity[day] = (dayActivity[day] || 0) + 1;
    });
    const mostProductiveDay = Object.entries(dayActivity)
      .sort((a, b) => b[1] - a[1])[0]?.[0] || 'N/A';
    
    return {
      totalSessions,
      totalApps,
      avgAppsPerSession,
      mostProductiveDay,
    };
  }, [sessions]);

  const maxBlocks = Math.max(...dailyActivity.map(d => d.totalBlocks), 1);

  return (
    <div className="flex-1 flex flex-col h-full bg-background">
      <div className="h-14 border-b border-border flex items-center px-6">
        <BarChart3 className="w-5 h-5 text-primary mr-2" />
        <h2 className="font-semibold">Activity Insights</h2>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-6 space-y-6 max-w-4xl mx-auto">
          {/* Quick Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card className="p-4">
              <div className="flex items-center gap-2 text-muted-foreground mb-1">
                <Calendar size={14} />
                <span className="text-xs">Total Sessions</span>
              </div>
              <p className="text-2xl font-bold">{productivityMetrics.totalSessions}</p>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-2 text-muted-foreground mb-1">
                <AppWindow size={14} />
                <span className="text-xs">Apps Tracked</span>
              </div>
              <p className="text-2xl font-bold">{productivityMetrics.totalApps}</p>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-2 text-muted-foreground mb-1">
                <TrendingUp size={14} />
                <span className="text-xs">Avg Apps/Session</span>
              </div>
              <p className="text-2xl font-bold">{productivityMetrics.avgAppsPerSession}</p>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-2 text-muted-foreground mb-1">
                <Clock size={14} />
                <span className="text-xs">Most Active Day</span>
              </div>
              <p className="text-2xl font-bold">{productivityMetrics.mostProductiveDay}</p>
            </Card>
          </div>

          {/* Weekly Activity Chart */}
          <Card className="p-4">
            <h3 className="font-medium mb-4">Last 7 Days Activity</h3>
            <div className="flex items-end justify-between gap-2 h-32">
              {dailyActivity.map((day, i) => (
                <div key={i} className="flex-1 flex flex-col items-center gap-1">
                  <div 
                    className="w-full bg-primary/20 rounded-t relative group cursor-pointer hover:bg-primary/30 transition-colors"
                    style={{ 
                      height: `${Math.max((day.totalBlocks / maxBlocks) * 100, 5)}%`,
                      minHeight: '4px'
                    }}
                  >
                    <div 
                      className="absolute bottom-0 left-0 right-0 bg-primary rounded-t transition-all"
                      style={{ height: `${day.sessionCount > 0 ? 100 : 0}%` }}
                    />
                    {/* Tooltip */}
                    <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 bg-popover border border-border rounded text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-10">
                      <p className="font-medium">{day.date}</p>
                      <p>{day.sessionCount} sessions</p>
                      <p>{day.totalBlocks} memory blocks</p>
                    </div>
                  </div>
                  <span className="text-xs text-muted-foreground">{day.dayOfWeek}</span>
                </div>
              ))}
            </div>
          </Card>

          {/* Top Apps */}
          <Card className="p-4">
            <h3 className="font-medium mb-4">Most Used Apps</h3>
            <div className="space-y-3">
              {appUsageStats.slice(0, 8).map((app, i) => (
                <div key={i} className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-muted flex items-center justify-center">
                    <AppIcon app={app.app} className="w-5 h-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium truncate">{app.app}</span>
                      <span className="text-xs text-muted-foreground">
                        {app.sessions} sessions
                      </span>
                    </div>
                    <div className="mt-1 h-1.5 bg-muted rounded-full overflow-hidden">
                      <div 
                        className="h-full bg-primary rounded-full"
                        style={{ 
                          width: `${(app.totalBlocks / (appUsageStats[0]?.totalBlocks || 1)) * 100}%` 
                        }}
                      />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Recent Activity Timeline */}
          <Card className="p-4">
            <h3 className="font-medium mb-4">Recent Sessions</h3>
            <div className="space-y-2">
              {sessions.slice(0, 5).map((session, i) => (
                <div key={i} className="flex items-center gap-3 p-2 rounded-lg hover:bg-accent/50">
                  <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center text-xs font-medium">
                    {new Date(session.date).getDate()}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">
                      {session.customTitle || session.title}
                    </p>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className="text-xs text-muted-foreground">
                        {session.apps.length} apps
                      </span>
                      <span className="text-xs text-muted-foreground">•</span>
                      <span className="text-xs text-muted-foreground">
                        {session.content.filter(c => c.type === 'app-memory').length} blocks
                      </span>
                    </div>
                  </div>
                  <div className="flex -space-x-1">
                    {session.apps.slice(0, 3).map((app, j) => (
                      <div key={j} className="w-6 h-6 rounded-full bg-muted flex items-center justify-center border-2 border-background">
                        <AppIcon app={app} className="w-3 h-3" />
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </ScrollArea>
    </div>
  );
};
