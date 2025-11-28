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
    }
};

// Helper to transform backend data into the Session format expected by the UI
export const transformToSession = async (date: string): Promise<Session> => {
    const apps = await api.getAppsForDate(date);

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

    return {
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
};
