/**
 * Semantic Tagging Service
 * Auto-generates tags from OCR content and categorizes activities
 */

// Activity categories
export type ActivityCategory = 'coding' | 'research' | 'communication' | 'design' | 'writing' | 'browsing' | 'other';

// Keywords that indicate different activity types
const CATEGORY_KEYWORDS: Record<ActivityCategory, string[]> = {
  coding: [
    'function', 'const', 'let', 'var', 'class', 'import', 'export', 'return',
    'if', 'else', 'for', 'while', 'switch', 'case', 'try', 'catch',
    'async', 'await', 'promise', 'console', 'log', 'error', 'debug',
    'npm', 'yarn', 'git', 'commit', 'push', 'pull', 'merge', 'branch',
    'typescript', 'javascript', 'python', 'java', 'react', 'vue', 'angular',
    'api', 'endpoint', 'request', 'response', 'json', 'xml', 'html', 'css',
    'database', 'query', 'select', 'insert', 'update', 'delete', 'table',
    'vscode', 'intellij', 'sublime', 'atom', 'vim', 'terminal', 'shell'
  ],
  research: [
    'search', 'google', 'stackoverflow', 'documentation', 'docs', 'wiki',
    'article', 'blog', 'tutorial', 'guide', 'learn', 'study', 'read',
    'paper', 'research', 'analysis', 'compare', 'review', 'evaluate',
    'github', 'repository', 'example', 'sample', 'demo', 'reference'
  ],
  communication: [
    'email', 'gmail', 'outlook', 'mail', 'inbox', 'sent', 'reply',
    'slack', 'teams', 'discord', 'zoom', 'meet', 'call', 'video',
    'message', 'chat', 'conversation', 'meeting', 'standup', 'sync',
    'calendar', 'schedule', 'invite', 'attendee', 'agenda'
  ],
  design: [
    'figma', 'sketch', 'adobe', 'photoshop', 'illustrator', 'xd',
    'design', 'layout', 'mockup', 'wireframe', 'prototype', 'ui', 'ux',
    'color', 'font', 'typography', 'icon', 'image', 'graphic', 'visual',
    'component', 'style', 'theme', 'responsive', 'mobile', 'desktop'
  ],
  writing: [
    'document', 'word', 'docs', 'notion', 'confluence', 'readme',
    'write', 'draft', 'edit', 'review', 'comment', 'feedback',
    'paragraph', 'section', 'chapter', 'outline', 'summary', 'notes',
    'markdown', 'text', 'content', 'copy', 'headline', 'title'
  ],
  browsing: [
    'chrome', 'firefox', 'safari', 'edge', 'browser', 'tab',
    'website', 'page', 'link', 'url', 'http', 'www',
    'youtube', 'twitter', 'linkedin', 'facebook', 'reddit', 'news'
  ],
  other: []
};

// App to category mapping
const APP_CATEGORIES: Record<string, ActivityCategory> = {
  'vscode': 'coding',
  'visual studio code': 'coding',
  'intellij': 'coding',
  'webstorm': 'coding',
  'pycharm': 'coding',
  'sublime': 'coding',
  'atom': 'coding',
  'terminal': 'coding',
  'iterm': 'coding',
  'powershell': 'coding',
  'cmd': 'coding',
  'figma': 'design',
  'sketch': 'design',
  'photoshop': 'design',
  'illustrator': 'design',
  'xd': 'design',
  'slack': 'communication',
  'teams': 'communication',
  'discord': 'communication',
  'zoom': 'communication',
  'outlook': 'communication',
  'gmail': 'communication',
  'mail': 'communication',
  'notion': 'writing',
  'word': 'writing',
  'docs': 'writing',
  'chrome': 'browsing',
  'firefox': 'browsing',
  'safari': 'browsing',
  'edge': 'browsing',
};

/**
 * Extract semantic tags from OCR text
 */
export function extractSemanticTags(ocrText: string): string[] {
  const tags = new Set<string>();
  const textLower = ocrText.toLowerCase();
  
  // Extract programming languages
  const languages = ['javascript', 'typescript', 'python', 'java', 'go', 'rust', 'c++', 'ruby', 'php'];
  languages.forEach(lang => {
    if (textLower.includes(lang)) tags.add(lang);
  });
  
  // Extract frameworks/libraries
  const frameworks = ['react', 'vue', 'angular', 'next', 'express', 'django', 'flask', 'spring'];
  frameworks.forEach(fw => {
    if (textLower.includes(fw)) tags.add(fw);
  });
  
  // Extract common tech terms
  const techTerms = ['api', 'database', 'frontend', 'backend', 'fullstack', 'devops', 'cloud', 'aws', 'docker', 'kubernetes'];
  techTerms.forEach(term => {
    if (textLower.includes(term)) tags.add(term);
  });
  
  // Extract action words
  const actions = ['debugging', 'testing', 'deploying', 'reviewing', 'refactoring', 'implementing'];
  actions.forEach(action => {
    if (textLower.includes(action)) tags.add(action);
  });
  
  return Array.from(tags).slice(0, 5);
}

/**
 * Categorize activity based on app name and OCR content
 */
export function categorizeActivity(appName: string, ocrText: string): ActivityCategory {
  const appLower = appName.toLowerCase();
  
  // Check app-based category first
  for (const [app, category] of Object.entries(APP_CATEGORIES)) {
    if (appLower.includes(app)) {
      return category;
    }
  }
  
  // Fall back to content-based categorization
  const textLower = ocrText.toLowerCase();
  const categoryScores: Record<ActivityCategory, number> = {
    coding: 0,
    research: 0,
    communication: 0,
    design: 0,
    writing: 0,
    browsing: 0,
    other: 0
  };
  
  for (const [category, keywords] of Object.entries(CATEGORY_KEYWORDS)) {
    keywords.forEach(keyword => {
      if (textLower.includes(keyword)) {
        categoryScores[category as ActivityCategory] += 1;
      }
    });
  }
  
  // Find category with highest score
  let maxCategory: ActivityCategory = 'other';
  let maxScore = 0;
  
  for (const [category, score] of Object.entries(categoryScores)) {
    if (score > maxScore) {
      maxScore = score;
      maxCategory = category as ActivityCategory;
    }
  }
  
  return maxCategory;
}

/**
 * Generate a smart summary tag based on session content
 */
export function generateSessionTags(session: {
  apps: string[];
  content: Array<{ type: string; data?: { ocrText?: string } }>;
}): string[] {
  const tags = new Set<string>();
  
  // Add app-based tags
  session.apps.forEach(app => {
    const category = APP_CATEGORIES[app.toLowerCase()];
    if (category) tags.add(category);
  });
  
  // Extract tags from OCR content
  session.content.forEach(block => {
    if (block.type === 'app-memory' && block.data?.ocrText) {
      const ocrTags = extractSemanticTags(block.data.ocrText);
      ocrTags.forEach(tag => tags.add(tag));
    }
  });
  
  return Array.from(tags).slice(0, 8);
}
