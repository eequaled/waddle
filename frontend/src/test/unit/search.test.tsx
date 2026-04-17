/**
 * Unit Tests for Search Functionality
 * **Feature: second-brain-enhancement**
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';

interface Session {
  id: string;
  title: string;
  summary: string;
  tags: string[];
  date: string;
}

interface SearchResult {
  session: Session;
  matchField: string;
  matchSnippet: string;
}

// Simple component to test search logic
const SearchTestComponent: React.FC<{
  sessions: Session[];
  onSelectResult: (sessionId: string, query: string) => void;
  onClose: () => void;
}> = ({ sessions, onSelectResult, onClose }) => {
  const [query, setQuery] = React.useState('');
  const [results, setResults] = React.useState<SearchResult[]>([]);

  const handleSearch = (searchQuery: string) => {
    setQuery(searchQuery);
    
    if (!searchQuery.trim()) {
      setResults([]);
      return;
    }
    
    const lowerQuery = searchQuery.toLowerCase();
    const searchResults: SearchResult[] = [];
    
    sessions.forEach(session => {
      if (session.title.toLowerCase().includes(lowerQuery)) {
        searchResults.push({
          session,
          matchField: 'title',
          matchSnippet: session.title,
        });
      } else if (session.summary.toLowerCase().includes(lowerQuery)) {
        searchResults.push({
          session,
          matchField: 'summary',
          matchSnippet: session.summary,
        });
      } else if (session.tags.some(t => t.toLowerCase().includes(lowerQuery))) {
        searchResults.push({
          session,
          matchField: 'tags',
          matchSnippet: session.tags.join(', '),
        });
      }
    });
    
    setResults(searchResults);
  };

  return (
    <div data-testid="search-modal">
      <input
        data-testid="search-input"
        value={query}
        onChange={(e) => handleSearch(e.target.value)}
        placeholder="Search memories..."
      />
      <button data-testid="close-btn" onClick={onClose}>Close</button>
      
      <div data-testid="results">
        {results.length === 0 && query && (
          <p data-testid="no-results">No results found</p>
        )}
        {results.map((result, i) => (
          <div
            key={result.session.id}
            data-testid={`result-${i}`}
            onClick={() => onSelectResult(result.session.id, query)}
          >
            <span data-testid={`result-title-${i}`}>{result.session.title}</span>
            <span data-testid={`result-field-${i}`}>{result.matchField}</span>
            <span data-testid={`result-snippet-${i}`}>{result.matchSnippet}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

describe('Search Unit Tests', () => {
  const mockSessions: Session[] = [
    {
      id: 'session-1',
      title: 'Coding Session',
      summary: 'Worked on React components',
      tags: ['coding', 'react'],
      date: '2024-01-15',
    },
    {
      id: 'session-2',
      title: 'Research Session',
      summary: 'Researched AI features',
      tags: ['research', 'ai'],
      date: '2024-01-14',
    },
    {
      id: 'session-3',
      title: 'Meeting Notes',
      summary: 'Daily discussion',
      tags: ['meeting', 'sync'],
      date: '2024-01-13',
    },
  ];

  it('should display search input', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    expect(screen.getByTestId('search-input')).toBeInTheDocument();
  });

  it('should show results when searching by title', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'Coding' } });
    
    expect(screen.getByTestId('result-0')).toBeInTheDocument();
    expect(screen.getByTestId('result-title-0')).toHaveTextContent('Coding Session');
    expect(screen.getByTestId('result-field-0')).toHaveTextContent('title');
  });

  it('should show results when searching by summary', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'React' } });
    
    expect(screen.getByTestId('result-0')).toBeInTheDocument();
    expect(screen.getByTestId('result-field-0')).toHaveTextContent('summary');
  });

  it('should show results when searching by tags', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    // Search for a tag that doesn't appear in title or summary
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'sync' } });
    
    expect(screen.getByTestId('result-0')).toBeInTheDocument();
    expect(screen.getByTestId('result-field-0')).toHaveTextContent('tags');
  });

  it('should show no results message when no matches', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'xyz123' } });
    
    expect(screen.getByTestId('no-results')).toBeInTheDocument();
  });

  it('should call onSelectResult when result is clicked', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'Coding' } });
    fireEvent.click(screen.getByTestId('result-0'));
    
    expect(onSelectResult).toHaveBeenCalledWith('session-1', 'Coding');
  });

  it('should call onClose when close button is clicked', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    fireEvent.click(screen.getByTestId('close-btn'));
    
    expect(onClose).toHaveBeenCalled();
  });

  it('should clear results when query is cleared', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'Coding' } });
    expect(screen.getByTestId('result-0')).toBeInTheDocument();
    
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: '' } });
    expect(screen.queryByTestId('result-0')).not.toBeInTheDocument();
  });

  it('should be case insensitive', () => {
    const onSelectResult = vi.fn();
    const onClose = vi.fn();
    
    render(
      <SearchTestComponent
        sessions={mockSessions}
        onSelectResult={onSelectResult}
        onClose={onClose}
      />
    );
    
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'CODING' } });
    
    expect(screen.getByTestId('result-0')).toBeInTheDocument();
  });
});
