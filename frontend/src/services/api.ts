import { Session, Activity, ContentBlock } from '../types';

const API_BASE = 'http://localhost:8080/api';

export const api = {
    getSessions: async (): Promise<string[]> => {
        const response = await fetch(`${API_BASE}/sessions`);
        if (!response.ok) throw new Error('Failed to fetch sessions');
        return response.json();
    },

    getAppsForDate: async (date: string): Promise<string[]> => {
        const response = await fetch(`${API_BASE}/sessions/${date}`);
        if (!response.ok) throw new Error('Failed to fetch apps');
        return response.json();
    },

    getAppDetails: async (date: string, app: string): Promise<any[]> => {
        const response = await fetch(`${API_BASE}/sessions/${date}/${app}`);
        if (!response.ok) throw new Error('Failed to fetch app details');
        return response.json();
    },

    getAppBlocks: async (date: string, app: string): Promise<any[]> => {
        const response = await fetch(`${API_BASE}/sessions/${date}/${app}/blocks`);
        if (!response.ok) return [];
        return response.json();
    },

    getStatus: async (): Promise<{ paused: boolean }> => {
        const response = await fetch(`${API_BASE}/status`);
        if (!response.ok) throw new Error('Failed to fetch status');
        return response.json();
    },

    setStatus: async (paused: boolean): Promise<{ paused: boolean }> => {
        const response = await fetch(`${API_BASE}/status`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ paused })
        });
        if (!response.ok) throw new Error('Failed to set status');
        return response.json();
    },

    getBlacklist: async (): Promise<string[]> => {
        const response = await fetch(`${API_BASE}/blacklist`);
        if (!response.ok) throw new Error('Failed to fetch blacklist');
        return response.json();
    },

    setBlacklist: async (apps: string[]): Promise<string[]> => {
        const response = await fetch(`${API_BASE}/blacklist`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(apps)
        });
        if (!response.ok) throw new Error('Failed to set blacklist');
        return response.json();
    },

    // Chat
    chat: async (context: string, message: string, sessionId?: string) => {
        const res = await fetch(`${API_BASE}/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ context, message, sessionId }),
        });
        if (!res.ok) throw new Error('Chat failed');
        return res.json();
    },

    getChatHistory: async () => {
        const res = await fetch(`${API_BASE}/chat`);
        if (!res.ok) throw new Error('Failed to fetch chat history');
        return res.json();
    },

    // Archives
    getArchives: async () => {
        const res = await fetch(`${API_BASE}/archives`);
        if (!res.ok) throw new Error('Failed to fetch archives');
        return res.json();
    },

    createArchive: async (name: string) => {
        const res = await fetch(`${API_BASE}/archives`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name }),
        });
        if (!res.ok) throw new Error('Failed to create archive');
        return res.json();
    },

    moveToArchive: async (sessionId: string, targetGroup: string, appName?: string) => {
        const res = await fetch(`${API_BASE}/archives/move`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ sessionId, targetGroup, appName }),
        });
        if (!res.ok) throw new Error('Failed to move to archive');
        return res.json();
    },

    // Metadata
    getSessionMetadata: async (date: string) => {
        const res = await fetch(`${API_BASE}/sessions/${date}/metadata`);
        if (!res.ok) return null;
        return res.json();
    },

    // Session Update
    updateSession: async (date: string, data: {
        customTitle?: string;
        customSummary?: string;
        originalSummary?: string;
        manualNotes?: Array<{ id: string; content: string; createdAt: string; updatedAt: string }>;
    }) => {
        const res = await fetch(`${API_BASE}/sessions/${date}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
        });
        if (!res.ok) throw new Error('Failed to update session');
        return res.json();
    },

    // Session Delete
    deleteSession: async (date: string) => {
        const res = await fetch(`${API_BASE}/sessions/${date}`, {
            method: 'DELETE',
        });
        if (!res.ok) throw new Error('Failed to delete session');
        return res.json();
    },

    // Notifications
    getNotifications: async () => {
        const res = await fetch(`${API_BASE}/notifications`);
        if (!res.ok) throw new Error('Failed to fetch notifications');
        return res.json();
    },

    createNotification: async (notification: {
        type: 'status' | 'insight' | 'processing';
        title: string;
        message: string;
        sessionRef?: string;
        metadata?: { appName?: string; timeSpent?: string };
    }) => {
        const res = await fetch(`${API_BASE}/notifications`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(notification),
        });
        if (!res.ok) throw new Error('Failed to create notification');
        return res.json();
    },

    markNotificationsRead: async (ids: string[]) => {
        const res = await fetch(`${API_BASE}/notifications/read`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ids }),
        });
        if (!res.ok) throw new Error('Failed to mark notifications as read');
        return res.json();
    },

    // Profile
    getProfileImages: async (): Promise<string[]> => {
        const res = await fetch(`${API_BASE}/profile/images`);
        if (!res.ok) throw new Error('Failed to fetch profile images');
        return res.json();
    },

    uploadProfileImage: async (file: File): Promise<{ filename: string; url: string }> => {
        const formData = new FormData();
        formData.append('file', file);
        const res = await fetch(`${API_BASE}/profile/upload`, {
            method: 'POST',
            body: formData,
        });
        if (!res.ok) throw new Error('Failed to upload profile image');
        return res.json();
    },

    deleteProfileImage: async (filename: string): Promise<void> => {
        const res = await fetch(`${API_BASE}/profile/delete`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ filename }),
        });
        if (!res.ok) throw new Error('Failed to delete profile image');
    },
};

// Helper to get lightweight session summary (for list view)
// Does NOT fetch blocks or images, just apps list
export const getSessionSummary = async (date: string): Promise<Session> => {
    const [apps, metadata] = await Promise.all([
        api.getAppsForDate(date),
        api.getSessionMetadata(date)
    ]);

    // Create minimal activity list
    const activities: Activity[] = apps.map(app => ({
        id: `act-${app}`,
        app: app,
        description: `Used ${app}`,
        timestamp: 'All Day'
    }));

    const session: Session = {
        id: date,
        title: `Session: ${date}`,
        summary: `Recorded activity for ${apps.length} applications.`,
        tags: ['recorded'],
        startTime: '9:00 AM', // Placeholder
        endTime: '5:00 PM',   // Placeholder
        duration: '8h',       // Placeholder
        apps: apps,
        date: date,
        activities: activities,
        content: [] // Empty content for summary
    };

    if (metadata) {
        if (metadata.customTitle) session.customTitle = metadata.customTitle;
        if (metadata.customSummary) session.customSummary = metadata.customSummary;
        // We don't need notes for the summary view
    }

    return session;
};

// Helper to transform backend data into the Session format expected by the UI
// Fetches EVERYTHING (expensive)
export const getFullSession = async (date: string): Promise<Session> => {
    const [apps, metadata] = await Promise.all([
        api.getAppsForDate(date),
        api.getSessionMetadata(date)
    ]);

    const activities: Activity[] = [];
    const content: ContentBlock[] = [];

    // Create a summary block
    content.push({
        id: `summary-${date}`,
        type: 'summary',
        content: `Activity log for ${date}. Detected apps: ${apps.join(', ')}.`
    });

    for (const app of apps) {
        try {
            const details = await api.getAppDetails(date, app);
            const blocks = await api.getAppBlocks(date, app);

            // Find latest screenshot
            let latestScreenshot = '';
            // Check for "latest.png" explicitly if available, or sort details
            const images = details.filter(d => d.type === 'image');
            if (images.length > 0) {
                // If we have latest.png, use it. Otherwise use the last one.
                const latest = images.find(img => img.name === 'latest.png') || images[images.length - 1];
                latestScreenshot = latest.url;
            }

            // Add activity for the app
            activities.push({
                id: `act-${app}`,
                app: app,
                description: `Used ${app}`,
                timestamp: 'All Day'
            });

            // Create App Memory Block
            // This replaces the generic heading/image list with a rich card
            content.push({
                id: `mem-${app}`,
                type: 'app-memory',
                content: app,
                data: {
                    appName: app,
                    latestScreenshot: latestScreenshot,
                    blocks: blocks,
                    timestamp: 'Session'
                }
            });

        } catch (error) {
            console.error(`[ERROR] Failed to load details for app ${app}:`, error);
        }
    }

    const session: Session = {
        id: date,
        title: `Session: ${date}`,
        summary: `Recorded activity for ${apps.length} applications.`,
        tags: ['recorded', 'auto-generated'],
        startTime: '9:00 AM', // Placeholder
        endTime: '5:00 PM',   // Placeholder
        duration: '8h',       // Placeholder
        apps: apps,
        date: date,
        activities: activities,
        content: content
    };

    if (metadata) {
        session.customTitle = metadata.customTitle;
        session.customSummary = metadata.customSummary;
        session.originalSummary = metadata.originalSummary;
        session.manualNotes = metadata.manualNotes;
    }

    return session;
};
