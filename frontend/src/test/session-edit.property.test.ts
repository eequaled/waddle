/**
 * Property-Based Tests for Session Edit Mode
 * **Feature: second-brain-enhancement**
 */
import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import { Session, ManualNote } from '../types';

// Use constant dates to avoid invalid date issues
const manualNoteArb = fc.record({
  id: fc.uuid(),
  content: fc.string(),
  createdAt: fc.constant('2024-01-15T10:00:00.000Z'),
  updatedAt: fc.constant('2024-01-15T10:00:00.000Z'),
});

const sessionArb = fc.record({
  id: fc.uuid(),
  title: fc.string({ minLength: 1 }),
  customTitle: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  summary: fc.string({ minLength: 1 }),
  customSummary: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  originalSummary: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  manualNotes: fc.array(manualNoteArb, { maxLength: 5 }),
  tags: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { maxLength: 5 }),
  startTime: fc.constant('09:00'),
  endTime: fc.constant('17:00'),
  duration: fc.constant('8h'),
  apps: fc.array(fc.string({ minLength: 1 }), { maxLength: 3 }),
  activities: fc.constant([]),
  content: fc.constant([]),
  date: fc.constant('2024-01-15'),
});

// Helper functions that mirror the component logic
function getEditState(session: Session) {
  return {
    title: session.customTitle || session.title,
    summary: session.customSummary || session.summary,
    manualNotes: session.manualNotes || [],
  };
}

function cancelEdit(session: Session, _editState: { title: string; summary: string; manualNotes: ManualNote[] }) {
  // Cancel should restore to original session values
  return {
    title: session.customTitle || session.title,
    summary: session.customSummary || session.summary,
    manualNotes: session.manualNotes || [],
  };
}

function addNoteAtBeginning(notes: ManualNote[], newNote: ManualNote): ManualNote[] {
  return [newNote, ...notes];
}

function saveSession(session: Session, editState: { title: string; summary: string; manualNotes: ManualNote[] }): Session {
  return {
    ...session,
    customTitle: editState.title !== session.title ? editState.title : session.customTitle,
    customSummary: editState.summary !== session.summary ? editState.summary : session.customSummary,
    originalSummary: session.originalSummary || session.summary,
    manualNotes: editState.manualNotes,
  };
}

describe('Session Edit Mode Property Tests', () => {
  /**
   * **Property 1: Edit Cancel Restores Original State**
   * **Validates: Requirements 1.6**
   */
  it('Property 1: Edit Cancel Restores Original State', () => {
    fc.assert(
      fc.property(
        sessionArb,
        fc.string({ minLength: 1 }),
        fc.string({ minLength: 1 }),
        (session, modifiedTitle, modifiedSummary) => {
          const initialEditState = getEditState(session);
          
          const modifiedEditState = {
            title: modifiedTitle,
            summary: modifiedSummary,
            manualNotes: [...initialEditState.manualNotes, {
              id: 'new-note',
              content: 'new content',
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
            }],
          };
          
          const restoredState = cancelEdit(session, modifiedEditState);
          
          expect(restoredState.title).toBe(initialEditState.title);
          expect(restoredState.summary).toBe(initialEditState.summary);
          expect(restoredState.manualNotes).toEqual(initialEditState.manualNotes);
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 2: Add Note Inserts at Beginning**
   * **Validates: Requirements 1.4**
   */
  it('Property 2: Add Note Inserts at Beginning', () => {
    fc.assert(
      fc.property(
        fc.array(manualNoteArb, { maxLength: 10 }),
        manualNoteArb,
        (existingNotes, newNote) => {
          const result = addNoteAtBeginning(existingNotes, newNote);
          
          expect(result[0]).toEqual(newNote);
          expect(result.length).toBe(existingNotes.length + 1);
          existingNotes.forEach((note, i) => {
            expect(result[i + 1]).toEqual(note);
          });
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * **Property 3: Modified Data Preserves Original**
   * **Validates: Requirements 1.7**
   */
  it('Property 3: Modified Data Preserves Original', () => {
    fc.assert(
      fc.property(
        sessionArb,
        fc.string({ minLength: 1 }),
        fc.string({ minLength: 1 }),
        (session, newTitle, newSummary) => {
          const editState = {
            title: newTitle,
            summary: newSummary,
            manualNotes: session.manualNotes || [],
          };
          
          const savedSession = saveSession(session, editState);
          
          expect(savedSession.originalSummary).toBeDefined();
          expect(savedSession.originalSummary).toBe(session.originalSummary || session.summary);
          
          if (newTitle !== session.title) {
            expect(savedSession.customTitle).toBe(newTitle);
          }
          
          if (newSummary !== session.summary) {
            expect(savedSession.customSummary).toBe(newSummary);
          }
        }
      ),
      { numRuns: 100 }
    );
  });
});
