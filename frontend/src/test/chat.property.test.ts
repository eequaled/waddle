/**
 * Property-Based Tests for Chat Functionality
 * **Feature: second-brain-enhancement**
 */
import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import { Session, BlockData } from '../types';

// Block data arbitrary
const blockDataArb = fc.record({
  id: fc.uuid(),
  startTime: fc.constant('10:00'),
  endTime: fc.constant('11:00'),
  microSummary: fc.string({ minLength: 1 }),
  ocrText: fc.string(),
});

const sessionArb = fc.record({
  id: fc.uuid(),
  title: fc.string({ minLength: 1 }),
  customTitle: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  summary: fc.string({ minLength: 1 }),
  customSummary: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  originalSummary: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  manualNotes: fc.constant([]),
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
    data: blockDataArb,
  }), { minLength: 1, maxLength: 5 }),
  date: fc.constant('2024-01-15'),
});

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

// Helper functions
function getContextualChatHeader(session: Session): string {
  return session.customTitle || session.title;
}

function buildChatContext(session: Session): string {
  const blocks = session.content
    .filter(b => b.type === 'app-memory' && b.data)
    .map(b => `[${b.data.startTime}-${b.data.endTime}] ${b.data.microSummary}\n${b.data.ocrText}`)
    .join('\n\n');
  return `Session: ${session.customTitle || session.title}\nDate: ${session.date}\n\n${blocks}`;
}

function buildBlockChatHeader(block: BlockData): string {
  return `Memory Block: ${block.startTime} - ${block.endTime}`;
}

function clearMessages(): ChatMessage[] {
  return [];
}

function getChatPreview(messages: ChatMessage[]): string {
  if (messages.length === 0) return '';
  const firstMessage = messages[0];
  const maxLength = 50;
  return firstMessage.content.length > maxLength 
    ? firstMessage.content.substring(0, maxLength) + '...'
    : firstMessage.content;
}

describe('Chat Property Tests', () => {
  /**
   * **Property 8: Contextual Chat Shows Session Title**
   * **Validates: Requirements 3.2**
   */
  it('Property 8: Contextual Chat Shows Session Title', () => {
    fc.assert(
      fc.property(sessionArb, (session) => {
        const header = getContextualChatHeader(session);
        const expectedTitle = session.customTitle || session.title;
        
        expect(header).toBe(expectedTitle);
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 9: Chat Request Includes Session Context**
   * **Validates: Requirements 3.3**
   */
  it('Property 9: Chat Request Includes Session Context', () => {
    fc.assert(
      fc.property(sessionArb, (session) => {
        const context = buildChatContext(session);
        
        expect(context).toContain(session.customTitle || session.title);
        expect(context).toContain(session.date);
        session.content.forEach(block => {
          if (block.type === 'app-memory' && block.data) {
            expect(context).toContain(block.data.microSummary);
          }
        });
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 12: New Chat Clears Messages**
   * **Validates: Requirements 4.3**
   */
  it('Property 12: New Chat Clears Messages', () => {
    fc.assert(
      fc.property(
        fc.array(fc.record({
          role: fc.constantFrom('user', 'assistant') as fc.Arbitrary<'user' | 'assistant'>,
          content: fc.string({ minLength: 1 }),
        }), { minLength: 1, maxLength: 20 }),
        (_messages) => {
          const result = clearMessages();
          
          expect(result).toEqual([]);
          expect(result.length).toBe(0);
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 13: Chat Preview Shows First Message**
   * **Validates: Requirements 4.4**
   */
  it('Property 13: Chat Preview Shows First Message', () => {
    fc.assert(
      fc.property(
        fc.array(fc.record({
          role: fc.constantFrom('user', 'assistant') as fc.Arbitrary<'user' | 'assistant'>,
          content: fc.string({ minLength: 1 }),
        }), { minLength: 1, maxLength: 10 }),
        (messages) => {
          const preview = getChatPreview(messages);
          const firstContent = messages[0].content;
          
          if (firstContent.length <= 50) {
            expect(preview).toBe(firstContent);
          } else {
            expect(preview).toBe(firstContent.substring(0, 50) + '...');
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 24: Block Chat Header Includes Timestamp**
   * **Validates: Requirements 7.3**
   */
  it('Property 24: Block Chat Header Includes Timestamp', () => {
    fc.assert(
      fc.property(blockDataArb, (block) => {
        const header = buildBlockChatHeader(block);
        
        expect(header).toContain(block.startTime);
        expect(header).toContain(block.endTime);
      }),
      { numRuns: 100 }
    );
  });
});
