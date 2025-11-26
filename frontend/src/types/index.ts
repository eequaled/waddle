export type AppType = string;

export interface Activity {
  id: string;
  app: AppType;
  description: string;
  timestamp: string;
}

export interface ContentBlock {
  id: string;
  type: 'heading' | 'paragraph' | 'todo' | 'image' | 'code' | 'summary' | 'link';
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
  content: ContentBlock[]; // For the editor
  date: string;
}
