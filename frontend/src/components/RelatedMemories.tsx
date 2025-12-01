import React, { useMemo } from 'react';
import { Session } from '../types';
import { Button } from './ui/button';
import { ScrollArea } from './ui/scroll-area';
import { 
  ChevronRight, 
  ChevronDown, 
  Sparkles,
  Calendar,
  AppWindow
} from 'lucide-react';
import { AppIcon } from './AppIcon';

interface RelatedMemoriesProps {
  currentSession: Session | null;
  allSessions: Session[];
  onSelectSession: (sessionId: string) => void;
}

interface RelatedSession {
  session: Session;
  relevanceScore: number;
  matchReasons: string[];
}

export const RelatedMemories: React.FC<RelatedMemoriesProps> = ({
  currentSession,
  allSessions,
  onSelectSession,
}) => {
  const [isExpanded, setIsExpanded] = React.useState(true);

  const relatedSessions = useMemo(() => {
    if (!currentSession) return [];

    const related: RelatedSession[] = [];

    allSessions.forEach(session => {
      if (session.id === currentSession.id) return;

      let score = 0;
      const reasons: string[] = [];

      // Check for app overlap
      const currentApps = new Set(currentSession.apps);
      const overlappingApps = session.apps.filter(app => currentApps.has(app));
      if (overlappingApps.length > 0) {
        score += overlappingApps.length * 10;
        reasons.push(`Same apps: ${overlappingApps.join(', ')}`);
      }

      // Check for tag overlap
      const currentTags = new Set(currentSession.tags);
      const overlappingTags = session.tags.filter(tag => currentTags.has(tag));
      if (overlappingTags.length > 0) {
        score += overlappingTags.length * 5;
        reasons.push(`Same tags: ${overlappingTags.join(', ')}`);
      }

      // Check for keyword overlap in title/summary
      const currentKeywords = extractKeywords(
        (currentSession.customTitle || currentSession.title) + ' ' + 
        (currentSession.customSummary || currentSession.summary)
      );
      const sessionKeywords = extractKeywords(
        (session.customTitle || session.title) + ' ' + 
        (session.customSummary || session.summary)
      );
      const keywordOverlap = currentKeywords.filter(k => sessionKeywords.includes(k));
      if (keywordOverlap.length > 0) {
        score += keywordOverlap.length * 3;
        reasons.push(`Similar content`);
      }

      // Recency bonus (sessions closer in time are more relevant)
      const daysDiff = Math.abs(
        new Date(currentSession.date).getTime() - new Date(session.date).getTime()
      ) / (1000 * 60 * 60 * 24);
      if (daysDiff <= 7) {
        score += Math.max(0, 7 - daysDiff);
        if (daysDiff <= 1) reasons.push('Recent');
      }

      if (score > 0) {
        related.push({ session, relevanceScore: score, matchReasons: reasons });
      }
    });

    // Sort by relevance score and take top 5
    return related
      .sort((a, b) => b.relevanceScore - a.relevanceScore)
      .slice(0, 5);
  }, [currentSession, allSessions]);

  if (!currentSession || relatedSessions.length === 0) {
    return null;
  }

  return (
    <div className="border-l border-border bg-background/50 w-64 shrink-0">
      <div 
        className="p-3 border-b border-border flex items-center justify-between cursor-pointer hover:bg-accent/50"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-2">
          <Sparkles size={14} className="text-amber-500" />
          <span className="text-sm font-medium">Related Memories</span>
        </div>
        {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
      </div>

      {isExpanded && (
        <ScrollArea className="h-[300px]">
          <div className="p-2 space-y-2">
            {relatedSessions.map(({ session, matchReasons }) => (
              <div
                key={session.id}
                className="p-2 rounded-lg border border-border/50 hover:bg-accent/50 cursor-pointer transition-colors"
                onClick={() => onSelectSession(session.id)}
              >
                <div className="flex items-start gap-2">
                  <div className="flex -space-x-1 shrink-0 mt-0.5">
                    {session.apps.slice(0, 2).map((app, i) => (
                      <div key={i} className="w-5 h-5 rounded-full bg-muted flex items-center justify-center">
                        <AppIcon app={app} className="w-3 h-3" />
                      </div>
                    ))}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-medium truncate">
                      {session.customTitle || session.title}
                    </p>
                    <div className="flex items-center gap-1 mt-0.5">
                      <Calendar size={10} className="text-muted-foreground" />
                      <span className="text-[10px] text-muted-foreground">
                        {session.date}
                      </span>
                    </div>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {matchReasons.slice(0, 2).map((reason, i) => (
                        <span 
                          key={i}
                          className="text-[9px] px-1.5 py-0.5 bg-primary/10 text-primary rounded"
                        >
                          {reason}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  );
};

// Helper function to extract keywords from text
function extractKeywords(text: string): string[] {
  const stopWords = new Set([
    'the', 'a', 'an', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for',
    'of', 'with', 'by', 'from', 'as', 'is', 'was', 'are', 'were', 'been',
    'be', 'have', 'has', 'had', 'do', 'does', 'did', 'will', 'would', 'could',
    'should', 'may', 'might', 'must', 'shall', 'can', 'need', 'dare', 'ought',
    'used', 'this', 'that', 'these', 'those', 'i', 'you', 'he', 'she', 'it',
    'we', 'they', 'what', 'which', 'who', 'whom', 'whose', 'where', 'when',
    'why', 'how', 'all', 'each', 'every', 'both', 'few', 'more', 'most',
    'other', 'some', 'such', 'no', 'nor', 'not', 'only', 'own', 'same', 'so',
    'than', 'too', 'very', 'just', 'session', 'activity', 'recorded'
  ]);

  return text
    .toLowerCase()
    .replace(/[^a-z0-9\s]/g, '')
    .split(/\s+/)
    .filter(word => word.length > 2 && !stopWords.has(word));
}
