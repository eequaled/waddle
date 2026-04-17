/**
 * Unit Tests for Notification System
 * **Feature: second-brain-enhancement**
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';

interface Notification {
  id: string;
  type: 'status' | 'insight' | 'processing';
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
  sessionRef?: string;
}

// Simple component to test notification logic
const NotificationTestComponent: React.FC<{
  notifications: Notification[];
  onMarkAsRead: (ids: string[]) => void;
  onNavigate: (sessionId: string) => void;
}> = ({ notifications, onMarkAsRead, onNavigate }) => {
  const [isOpen, setIsOpen] = React.useState(false);
  
  const unreadCount = notifications.filter(n => !n.read).length;

  const handleClose = () => {
    if (unreadCount > 0) {
      const unreadIds = notifications.filter(n => !n.read).map(n => n.id);
      onMarkAsRead(unreadIds);
    }
    setIsOpen(false);
  };

  return (
    <div>
      <button data-testid="bell-btn" onClick={() => setIsOpen(!isOpen)}>
        Notifications
        {unreadCount > 0 && (
          <span data-testid="badge">{unreadCount}</span>
        )}
      </button>
      
      {isOpen && (
        <div data-testid="panel">
          <button data-testid="close-panel" onClick={handleClose}>Close</button>
          {notifications.length === 0 ? (
            <p data-testid="empty-message">No notifications</p>
          ) : (
            <ul>
              {notifications.map(n => (
                <li
                  key={n.id}
                  data-testid={`notification-${n.id}`}
                  data-read={n.read}
                  onClick={() => n.sessionRef && onNavigate(n.sessionRef)}
                >
                  <span data-testid={`title-${n.id}`}>{n.title}</span>
                  <span data-testid={`message-${n.id}`}>{n.message}</span>
                  {!n.read && <span data-testid={`unread-${n.id}`}>•</span>}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
};

describe('Notification System Unit Tests', () => {
  const mockNotifications: Notification[] = [
    {
      id: 'notif-1',
      type: 'status',
      title: 'Recording Paused',
      message: 'Screen recording has been paused',
      timestamp: '2024-01-15T10:00:00.000Z',
      read: false,
    },
    {
      id: 'notif-2',
      type: 'processing',
      title: 'AI Processing Complete',
      message: 'Memory summary ready',
      timestamp: '2024-01-15T09:00:00.000Z',
      read: true,
      sessionRef: 'session-1',
    },
    {
      id: 'notif-3',
      type: 'insight',
      title: 'Usage Insight',
      message: 'You spent 3h on VSCode',
      timestamp: '2024-01-15T08:00:00.000Z',
      read: false,
      sessionRef: 'session-2',
    },
  ];

  it('should display badge with unread count', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    
    render(
      <NotificationTestComponent
        notifications={mockNotifications}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    expect(screen.getByTestId('badge')).toHaveTextContent('2');
  });

  it('should not display badge when no unread notifications', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    const allRead = mockNotifications.map(n => ({ ...n, read: true }));
    
    render(
      <NotificationTestComponent
        notifications={allRead}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    expect(screen.queryByTestId('badge')).not.toBeInTheDocument();
  });

  it('should open panel when bell is clicked', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    
    render(
      <NotificationTestComponent
        notifications={mockNotifications}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    expect(screen.queryByTestId('panel')).not.toBeInTheDocument();
    
    fireEvent.click(screen.getByTestId('bell-btn'));
    
    expect(screen.getByTestId('panel')).toBeInTheDocument();
  });

  it('should display all notifications in panel', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    
    render(
      <NotificationTestComponent
        notifications={mockNotifications}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    fireEvent.click(screen.getByTestId('bell-btn'));
    
    expect(screen.getByTestId('notification-notif-1')).toBeInTheDocument();
    expect(screen.getByTestId('notification-notif-2')).toBeInTheDocument();
    expect(screen.getByTestId('notification-notif-3')).toBeInTheDocument();
  });

  it('should show unread indicator for unread notifications', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    
    render(
      <NotificationTestComponent
        notifications={mockNotifications}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    fireEvent.click(screen.getByTestId('bell-btn'));
    
    expect(screen.getByTestId('unread-notif-1')).toBeInTheDocument();
    expect(screen.queryByTestId('unread-notif-2')).not.toBeInTheDocument();
    expect(screen.getByTestId('unread-notif-3')).toBeInTheDocument();
  });

  it('should mark notifications as read when panel is closed', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    
    render(
      <NotificationTestComponent
        notifications={mockNotifications}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    fireEvent.click(screen.getByTestId('bell-btn'));
    fireEvent.click(screen.getByTestId('close-panel'));
    
    expect(onMarkAsRead).toHaveBeenCalledWith(['notif-1', 'notif-3']);
  });

  it('should navigate when notification with sessionRef is clicked', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    
    render(
      <NotificationTestComponent
        notifications={mockNotifications}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    fireEvent.click(screen.getByTestId('bell-btn'));
    fireEvent.click(screen.getByTestId('notification-notif-2'));
    
    expect(onNavigate).toHaveBeenCalledWith('session-1');
  });

  it('should show empty message when no notifications', () => {
    const onMarkAsRead = vi.fn();
    const onNavigate = vi.fn();
    
    render(
      <NotificationTestComponent
        notifications={[]}
        onMarkAsRead={onMarkAsRead}
        onNavigate={onNavigate}
      />
    );
    
    fireEvent.click(screen.getByTestId('bell-btn'));
    
    expect(screen.getByTestId('empty-message')).toBeInTheDocument();
  });
});
