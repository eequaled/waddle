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
            console.log(`[DEBUG] Got details for ${app}:`, details);

            // Validate that details is an array
            if (!Array.isArray(details)) {
                console.warn(`[WARN] Details for ${app} is not an array:`, details);
                continue;
            }

            // Add activity for the app
            activities.push({
                id: `act-${app}`,
                app: app,
                description: `Used ${app}`,
                timestamp: 'All Day' // We could refine this if backend provided timestamps in list
            });

            // Add heading for app
            content.push({
                id: `head-${app}`,
                type: 'heading',
                content: app
            });

            // Add images and text
            for (const file of details) {
                if (file.type === 'image') {
                    content.push({
                        id: `img-${file.name}`,
                        type: 'image',
                        content: file.url
                    });
                } else if (file.type === 'text') {
                    // Fetch text content (optional, or just show link)
                    // For now, let's just show a note that text exists
                    content.push({
                        id: `txt-${file.name}`,
                        type: 'paragraph',
                        content: `Captured text available: ${file.name}`
                    });
                }
            }
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
