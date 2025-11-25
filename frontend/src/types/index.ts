export type AppType = 'Chrome' | 'Slack' | 'Notes' | 'Spotify' | 'Figma' | 'VS Code' | 'Zoom' | 'Excel' | 'Terminal';

export interface Activity {
  id: string;
  app: AppType;
  description: string;
  timestamp: string;
}

export interface EditorBlock {
  id: string;
  type: 'heading' | 'paragraph' | 'todo' | 'image' | 'code' | 'summary';
  content: string;
  checked?: boolean; // for todo
  language?: string; // for code
}

export interface Session {
  id: string;
  title: string;
  summary: string; // The auto-summary text
  tags: string[];
  startTime: string;
  endTime: string;
  duration: string;
  apps: AppType[];
  activities: Activity[]; // For the drill-down
  content: EditorBlock[]; // For the editor
  date: 'Today' | 'Yesterday' | 'Last Week';
}
