import React, { useState, useEffect } from 'react';
import { Button } from './ui/button';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from './ui/popover';
import { ScrollArea } from './ui/scroll-area';
import { Badge } from './ui/badge';
import { 
  Bell, 
  Circle, 
  Sparkles, 
  Clock, 
  CheckCircle,
  ExternalLink 
} from 'lucide-react';

export interface Notification {
  id: string;
  type: 'status' | 'insight' | 'processing';
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
  sessionRef?: string;
  metadata?: {
    appName?: string;
    timeSpent?: string;
  };
}

interface NotificationPanelProps {
  notifications: Notification[];
  onMarkAsRead: (ids: string[]) => void;
  onNavigateToSession?: (sessionId: string) => void;
}

const typeIcons = {
  status: Circle,
  insight: Sparkles,
  processing: Clock,
};

const typeColors = {
  status: 'text-blue-500',
  insight: 'text-amber-500',
  processing: 'text-green-500',
};

export const NotificationPanel: React.FC<NotificationPanelProps> = ({
  notifications,
  onMarkAsRead,
  onNavigateToSession,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  
  const unreadCount = notifications.filter(n => !n.read).length;


  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    // Mark all as read when closing
    if (!open && unreadCount > 0) {
      const unreadIds = notifications.filter(n => !n.read).map(n => n.id);
      onMarkAsRead(unreadIds);
    }
  };

  const handleNotificationClick = (notification: Notification) => {
    if (notification.sessionRef && onNavigateToSession) {
      onNavigateToSession(notification.sessionRef);
      setIsOpen(false);
    }
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <Popover open={isOpen} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="relative text-muted-foreground">
          <Bell size={18} />
          {unreadCount > 0 && (
            <Badge 
              variant="destructive" 
              className="absolute -top-1 -right-1 h-5 w-5 p-0 flex items-center justify-center text-[10px]"
            >
              {unreadCount > 9 ? '9+' : unreadCount}
            </Badge>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 p-0">
        <div className="p-3 border-b border-border flex items-center justify-between">
          <h3 className="font-semibold text-sm">Notifications</h3>
          {unreadCount > 0 && (
            <span className="text-xs text-muted-foreground">
              {unreadCount} unread
            </span>
          )}
        </div>
        
        <ScrollArea className="h-[300px]">
          {notifications.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-[200px] text-muted-foreground">
              <Bell className="w-8 h-8 mb-2 opacity-20" />
              <p className="text-sm">No notifications yet</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {notifications.map(notification => {
                const Icon = typeIcons[notification.type];
                const iconColor = typeColors[notification.type];
                
                return (
                  <div
                    key={notification.id}
                    onClick={() => handleNotificationClick(notification)}
                    className={`p-3 hover:bg-accent/50 transition-colors ${
                      notification.sessionRef ? 'cursor-pointer' : ''
                    } ${!notification.read ? 'bg-primary/5' : ''}`}
                  >
                    <div className="flex gap-3">
                      <div className={`shrink-0 ${iconColor}`}>
                        <Icon size={16} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-start justify-between gap-2">
                          <p className="text-sm font-medium truncate">
                            {notification.title}
                          </p>
                          {!notification.read && (
                            <div className="w-2 h-2 rounded-full bg-primary shrink-0 mt-1" />
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                          {notification.message}
                        </p>
                        {notification.metadata?.appName && (
                          <p className="text-xs text-muted-foreground mt-1">
                            {notification.metadata.appName} • {notification.metadata.timeSpent}
                          </p>
                        )}
                        <div className="flex items-center justify-between mt-1">
                          <span className="text-[10px] text-muted-foreground">
                            {formatTimestamp(notification.timestamp)}
                          </span>
                          {notification.sessionRef && (
                            <span className="text-[10px] text-primary flex items-center gap-0.5">
                              View <ExternalLink size={10} />
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </ScrollArea>
      </PopoverContent>
    </Popover>
  );
};
