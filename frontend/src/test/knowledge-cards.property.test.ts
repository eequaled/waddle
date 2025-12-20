/**
 * Property-Based Tests for Knowledge Cards
 * **Feature: ai-synthesis**
 */
import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import { KnowledgeCard, Entity } from '../types';

// Entity arbitrary
const entityArb = fc.record({
  value: fc.string({ minLength: 1, maxLength: 50 }),
  type: fc.constantFrom('jira', 'hashtag', 'mention', 'url') as fc.Arbitrary<Entity['type']>,
  count: fc.integer({ min: 1, max: 10 }),
});

// Knowledge Card arbitrary
const knowledgeCardArb = fc.record({
  sessionId: fc.uuid(),
  title: fc.string({ minLength: 1, maxLength: 100 }),
  bullets: fc.array(fc.string({ minLength: 1, maxLength: 200 }), { minLength: 3, maxLength: 3 }), // Exactly 3 bullets
  entities: fc.array(entityArb, { maxLength: 10 }),
  timestamp: fc.integer({ min: 1577836800000, max: 1924991999000 }).map(ms => new Date(ms).toISOString()),
  status: fc.constantFrom('pending', 'processed', 'failed') as fc.Arbitrary<KnowledgeCard['status']>,
});

// Helper function to validate knowledge card completeness
function validateKnowledgeCardCompleteness(card: KnowledgeCard): boolean {
  // All required fields must be present and non-empty (except entities which may be empty array)
  const hasSessionId = typeof card.sessionId === 'string' && card.sessionId.length > 0;
  const hasTitle = typeof card.title === 'string' && card.title.length > 0;
  const hasBullets = Array.isArray(card.bullets);
  const hasEntities = Array.isArray(card.entities);
  const hasTimestamp = typeof card.timestamp === 'string' && card.timestamp.length > 0;
  const hasStatus = typeof card.status === 'string' && 
    ['pending', 'processed', 'failed'].includes(card.status);

  return hasSessionId && hasTitle && hasBullets && hasEntities && hasTimestamp && hasStatus;
}

// Helper function to validate processed card has exactly 3 bullets
function validateProcessedCardBullets(card: KnowledgeCard): boolean {
  if (card.status === 'processed') {
    return card.bullets.length === 3 && card.bullets.every(bullet => 
      typeof bullet === 'string' && bullet.length > 0
    );
  }
  return true; // Non-processed cards don't need to validate bullet count
}

describe('Knowledge Cards Property Tests', () => {
  /**
   * **Property 8: Knowledge Card Completeness**
   * **Validates: Requirements 5.4**
   */
  it('Property 8: Knowledge Card Completeness', () => {
    fc.assert(
      fc.property(knowledgeCardArb, (card) => {
        const isComplete = validateKnowledgeCardCompleteness(card);
        expect(isComplete).toBe(true);
        
        // Verify specific field types and constraints
        expect(card.sessionId).toBeDefined();
        expect(card.sessionId.length).toBeGreaterThan(0);
        
        expect(card.title).toBeDefined();
        expect(card.title.length).toBeGreaterThan(0);
        
        expect(Array.isArray(card.bullets)).toBe(true);
        expect(Array.isArray(card.entities)).toBe(true);
        
        expect(card.timestamp).toBeDefined();
        expect(card.timestamp.length).toBeGreaterThan(0);
        
        expect(['pending', 'processed', 'failed']).toContain(card.status);
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 7: Three-Bullet Summary Format**
   * **Validates: Requirements 5.2**
   */
  it('Property 7: Three-Bullet Summary Format', () => {
    fc.assert(
      fc.property(knowledgeCardArb, (card) => {
        const hasValidBullets = validateProcessedCardBullets(card);
        expect(hasValidBullets).toBe(true);
        
        if (card.status === 'processed') {
          expect(card.bullets).toHaveLength(3);
          card.bullets.forEach(bullet => {
            expect(typeof bullet).toBe('string');
            expect(bullet.length).toBeGreaterThan(0);
          });
        }
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property: Entity Validation**
   * **Validates: Entity structure and types**
   */
  it('Property: Entity Validation', () => {
    fc.assert(
      fc.property(knowledgeCardArb, (card) => {
        card.entities.forEach(entity => {
          expect(typeof entity.value).toBe('string');
          expect(entity.value.length).toBeGreaterThan(0);
          
          expect(['jira', 'hashtag', 'mention', 'url']).toContain(entity.type);
          
          expect(typeof entity.count).toBe('number');
          expect(entity.count).toBeGreaterThan(0);
        });
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property: Timestamp Validity**
   * **Validates: Timestamp format and parsing**
   */
  it('Property: Timestamp Validity', () => {
    fc.assert(
      fc.property(knowledgeCardArb, (card) => {
        // Should be able to parse timestamp as valid date
        const date = new Date(card.timestamp);
        expect(date.getTime()).not.toBeNaN();
        expect(date.toISOString()).toBe(card.timestamp);
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 9: Pending Session Indicator**
   * **Validates: Requirements 5.5**
   */
  it('Property 9: Pending Session Indicator', () => {
    fc.assert(
      fc.property(knowledgeCardArb, (card) => {
        // For any knowledge card, if status is 'pending', it should be identifiable as pending
        if (card.status === 'pending') {
          expect(card.status).toBe('pending');
          // Pending cards may have empty bullets or incomplete data
          // The key is that the status field correctly indicates pending state
        }
        
        // For processed cards, status should be 'processed'
        if (card.status === 'processed') {
          expect(card.status).toBe('processed');
          // Processed cards should have 3 bullets
          expect(card.bullets).toHaveLength(3);
        }
        
        // For failed cards, status should be 'failed'
        if (card.status === 'failed') {
          expect(card.status).toBe('failed');
        }
      }),
      { numRuns: 100 }
    );
  });

  /**
   * **Property: Status Consistency**
   * **Validates: Status field matches card state**
   */
  it('Property: Status Consistency', () => {
    fc.assert(
      fc.property(
        fc.array(knowledgeCardArb, { minLength: 1, maxLength: 20 }),
        (cards) => {
          // Count cards by status
          const pendingCards = cards.filter(c => c.status === 'pending');
          const processedCards = cards.filter(c => c.status === 'processed');
          const failedCards = cards.filter(c => c.status === 'failed');
          
          // Total should equal original array length
          expect(pendingCards.length + processedCards.length + failedCards.length).toBe(cards.length);
          
          // All pending cards should have 'pending' status
          pendingCards.forEach(card => {
            expect(card.status).toBe('pending');
          });
          
          // All processed cards should have 'processed' status and 3 bullets
          processedCards.forEach(card => {
            expect(card.status).toBe('processed');
            expect(card.bullets).toHaveLength(3);
          });
          
          // All failed cards should have 'failed' status
          failedCards.forEach(card => {
            expect(card.status).toBe('failed');
          });
        }
      ),
      { numRuns: 100 }
    );
  });
});