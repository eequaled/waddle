export type AppType = string;

export interface Activity {
  id: string;
  app: AppType;
  description: string;
  timestamp: string;
}

export interface ContentBlock {
  id: string;
  type: 'heading' | 'paragraph' | 'todo' | 'image' | 'code' | 'summary' | 'link' | 'app-memory' | 'manual-note';
  content: string; // For app-memory, this might be JSON stringified data or we use a specific field
  data?: any; // Generic data field for complex blocks like app-memory
  checked?: boolean; // for todo
  language?: string; // for code
}

export interface BlockData {
  id: string;
  startTime: string;
  endTime: string;
  microSummary: string;
  ocrText: string;
}

export interface ManualNote {
  id: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface Session {
  id: string;
  title: string;
  customTitle?: string;        // User-edited title
  summary: string;
  customSummary?: string;      // User-edited summary
  originalSummary?: string;    // Preserved AI summary
  manualNotes?: ManualNote[];  // User-added notes
  tags: string[];
  startTime: string;
  endTime: string;
  duration: string;
  apps: AppType[];
  activities: Activity[];
  content: ContentBlock[];
  date: string;
}
