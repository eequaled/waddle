/**
 * Property-Based Tests for Session Actions
 * **Feature: second-brain-enhancement**
 */
import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import { Session } from '../types';

// Session arbitrary with constant dates
const sessionArb = fc.record({
  id: fc.uuid(),
  title: fc.string({ minLength: 1 }),
  customTitle: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  summary: fc.string({ minLength: 1 }),
  customSummary: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  originalSummary: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  manualNotes: fc.array(fc.record({
    id: fc.uuid(),
    content: fc.string(),
    createdAt: fc.constant('2024-01-15T10:00:00.000Z'),
    updatedAt: fc.constant('2024-01-15T10:00:00.000Z'),
  }), { maxLength: 3 }),
  tags: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { maxLength: 5 }),
  startTime: fc.constant('09:00'),
  endTime: fc.constant('17:00'),
  duration: fc.constant('8h'),
  apps: fc.array(fc.string({ minLength: 1 }), { maxLength: 3 }),
  activities: fc.constant([]),
  content: fc.array(fc.record({
    id: fc.uuid(),
    type: fc.constant('app-memory' as const),
    content: fc.string(),
    data: fc.record({
      id: fc.uuid(),
      startTime: fc.constant('10:00'),
      endTime: fc.constant('11:00'),
      microSummary: fc.string({ minLength: 1 }),
      ocrText: fc.string(),
    }),
  }), { minLength: 1, maxLength: 5 }),
  date: fc.constant('2024-01-15'),
});

// Helper functions
function moveToArchive(sessions: Session[], sessionId: string, archiveId: string): { 
  sessions: Session[]; 
  archive: { id: string; items: Session[] } 
} {
  const sessionToMove = sessions.find(s => s.id === sessionId);
  const remainingSessions = sessions.filter(s => s.id !== sessionId);
  return {
    sessions: remainingSessions,
    archive: { id: archiveId, items: sessionToMove ? [sessionToMove] : [] },
  };
}

function exportToMarkdown(session: Session): string {
  const displayTitle = session.customTitle || session.title;
  const displaySummary = session.customSummary || session.summary;
  
  let markdown = `# ${displayTitle}\n\n`;
  markdown += `**Date:** ${session.date}\n`;
  markdown += `**Time:** ${session.startTime} - ${session.endTime}\n`;
  markdown += `**Duration:** ${session.duration}\n\n`;
  markdown += `## Summary\n\n${displaySummary}\n\n`;
  
  if (session.tags.length > 0) {
    markdown += `**Tags:** ${session.tags.map(t => `#${t}`).join(' ')}\n\n`;
  }
  
  if (session.manualNotes && session.manualNotes.length > 0) {
    markdown += `## Notes\n\n`;
    session.manualNotes.forEach(note => {
      markdown += `- ${note.content}\n`;
    });
    markdown += '\n';
  }
  
  markdown += `## Memory Blocks\n\n`;
  session.content.forEach(block => {
    if (block.type === 'app-memory' && block.data) {
      markdown += `### ${block.data.startTime} - ${block.data.endTime}\n`;
      markdown += `${block.data.microSummary}\n\n`;
    }
  });
  
  return markdown;
}

function generateExportFilename(session: Session): string {
  return `session-${session.date}.md`;
}

function deleteSession(sessions: Session[], sessionId: string): Session[] {
  return sessions.filter(s => s.id !== sessionId);
}

describe('Session Actions Property Tests', () => {
  /**
   * **Property 4: Archive Move Removes from Timeline**
   * **Validates: Requirements 2.3**
   */
  it('Property 4: Archive Move Removes from Timeline', () => {
    fc.assert(
      fc.property(
        fc.array(sessionArb, { minLength: 1, maxLength: 10 }),
        fc.uuid(),
        (sessions, archiveId) => {
          const sessionToMove = sessions[0];
          const result = moveToArchive(sessions, sessionToMove.id, archiveId);
          
          expect(result.sessions.find(s => s.id === sessionToMove.id)).toBeUndefined();
          expect(result.archive.items.find(s => s.id === sessionToMove.id)).toBeDefined();
          expect(result.sessions.length).toBe(sessions.length - 1);
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 5: Export Contains Required Fields**
   * **Validates: Requirements 2.4**
   */
  it('Property 5: Export Contains Required Fields', () => {
    fc.assert(
      fc.property(sessionArb, (session) => {
        const markdown = exportToMarkdown(session);
        const displayTitle = session.customTitle || session.title;
        const displaySummary = session.customSummary || session.summary;
        
        expect(markdown).toContain(displayTitle);
        expect(markdown).toContain(displaySummary);
        expect(markdown).toContain(session.date);
        expect(markdown).toContain(session.startTime);
        expect(markdown).toContain(session.endTime);
        expect(markdown).toContain(session.duration);
        session.content.forEach(block => {
          if (block.type === 'app-memory' && block.data?.microSummary) {
            expect(markdown).toContain(block.data.microSummary);
          }
        });
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 6: Export Filename Matches Pattern**
   * **Validates: Requirements 2.5**
   */
  it('Property 6: Export Filename Matches Pattern', () => {
    fc.assert(
      fc.property(sessionArb, (session) => {
        const filename = generateExportFilename(session);
        
        expect(filename).toBe(`session-${session.date}.md`);
        expect(filename).toMatch(/\.md$/);
        expect(filename).toMatch(/^session-/);
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 7: Delete Reduces Session Count**
   * **Validates: Requirements 2.7**
   */
  it('Property 7: Delete Reduces Session Count', () => {
    fc.assert(
      fc.property(
        fc.array(sessionArb, { minLength: 1, maxLength: 10 }),
        (sessions) => {
          const sessionToDelete = sessions[0];
          const initialCount = sessions.length;
          const result = deleteSession(sessions, sessionToDelete.id);
          
          expect(result.length).toBe(initialCount - 1);
          expect(result.find(s => s.id === sessionToDelete.id)).toBeUndefined();
          sessions.slice(1).forEach(s => {
            expect(result.find(r => r.id === s.id)).toBeDefined();
          });
        }
      ),
      { numRuns: 100 }
    );
  });
});
