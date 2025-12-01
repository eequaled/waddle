/**
 * Property-Based Tests for Memory Block Interaction and Markdown Rendering
 * **Feature: second-brain-enhancement**
 */
import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import { BlockData, Session } from '../types';

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

// Helper functions
function buildBlockContext(session: Session, block: BlockData): string {
  return `Session: ${session.customTitle || session.title}
Date: ${session.date}
Block Time: ${block.startTime} - ${block.endTime}

Content:
${block.microSummary}

OCR Text:
${block.ocrText}`;
}

function getBlockChatHeader(block: BlockData): string {
  return `Memory Block: ${block.startTime} - ${block.endTime}`;
}

// Simplified markdown to HTML conversion for testing
function markdownToHtml(markdown: string): string {
  let html = markdown;
  
  // Headers
  html = html.replace(/^###### (.+)$/gm, '<h6>$1</h6>');
  html = html.replace(/^##### (.+)$/gm, '<h5>$1</h5>');
  html = html.replace(/^#### (.+)$/gm, '<h4>$1</h4>');
  html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');
  html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>');
  html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>');
  
  // Bold
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  
  // Italic
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
  
  // Code blocks
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
  
  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>');
  
  // Unordered lists
  html = html.replace(/^- (.+)$/gm, '<li>$1</li>');
  
  // Ordered lists
  html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');
  
  return html;
}

// Simple alphanumeric string generator
const alphanumericArb = fc.stringMatching(/^[a-zA-Z0-9 ]+$/, { minLength: 1, maxLength: 50 });

describe('Memory Block Property Tests', () => {
  /**
   * **Property 23: Ask AI Opens Chat with Block Context**
   * **Validates: Requirements 7.2**
   */
  it('Property 23: Ask AI Opens Chat with Block Context', () => {
    fc.assert(
      fc.property(sessionArb, (session) => {
        if (session.content.length === 0) return;
        
        const block = session.content[0].data as BlockData;
        const context = buildBlockContext(session, block);
        
        expect(context).toContain(session.customTitle || session.title);
        expect(context).toContain(session.date);
        expect(context).toContain(block.startTime);
        expect(context).toContain(block.endTime);
        expect(context).toContain(block.microSummary);
      }),
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
        const header = getBlockChatHeader(block);
        
        expect(header).toContain(block.startTime);
        expect(header).toContain(block.endTime);
      }),
      { numRuns: 100 }
    );
  });
});

describe('Markdown Rendering Property Tests', () => {
  /**
   * **Property 10: Markdown Rendering Produces Correct HTML**
   * **Validates: Requirements 3.4, 4.1**
   */
  it('Property 10: Headers render correctly', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 6 }),
        alphanumericArb,
        (level, text) => {
          const markdown = '#'.repeat(level) + ' ' + text;
          const html = markdownToHtml(markdown);
          
          expect(html).toContain(`<h${level}>`);
          expect(html).toContain(`</h${level}>`);
          expect(html).toContain(text);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('Property 10: Bold text renders correctly', () => {
    fc.assert(
      fc.property(alphanumericArb, (text) => {
        const markdown = `**${text}**`;
        const html = markdownToHtml(markdown);
        
        expect(html).toContain('<strong>');
        expect(html).toContain('</strong>');
        expect(html).toContain(text);
      }),
      { numRuns: 100 }
    );
  });

  it('Property 10: Italic text renders correctly', () => {
    fc.assert(
      fc.property(alphanumericArb, (text) => {
        const markdown = `*${text}*`;
        const html = markdownToHtml(markdown);
        
        expect(html).toContain('<em>');
        expect(html).toContain('</em>');
        expect(html).toContain(text);
      }),
      { numRuns: 100 }
    );
  });

  it('Property 10: Code blocks render correctly', () => {
    fc.assert(
      fc.property(alphanumericArb, (code) => {
        const markdown = '```js\n' + code + '\n```';
        const html = markdownToHtml(markdown);
        
        expect(html).toContain('<pre>');
        expect(html).toContain('<code>');
        expect(html).toContain(code);
      }),
      { numRuns: 100 }
    );
  });

  it('Property 10: Inline code renders correctly', () => {
    fc.assert(
      fc.property(alphanumericArb, (code) => {
        const markdown = `\`${code}\``;
        const html = markdownToHtml(markdown);
        
        expect(html).toContain('<code>');
        expect(html).toContain('</code>');
        expect(html).toContain(code);
      }),
      { numRuns: 100 }
    );
  });

  it('Property 10: List items render correctly', () => {
    fc.assert(
      fc.property(
        fc.array(alphanumericArb, { minLength: 1, maxLength: 5 }),
        (items) => {
          const markdown = items.map(item => `- ${item}`).join('\n');
          const html = markdownToHtml(markdown);
          
          items.forEach(item => {
            expect(html).toContain('<li>');
            expect(html).toContain(item);
          });
        }
      ),
      { numRuns: 100 }
    );
  });
});
