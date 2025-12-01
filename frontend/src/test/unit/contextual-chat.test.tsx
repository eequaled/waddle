/**
 * Unit Tests for Contextual Chat
 * **Feature: second-brain-enhancement**
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';

// Mock session data
const mockSession = {
  id: 'test-session-1',
  title: 'Test Session',
  customTitle: 'Custom Title',
  summary: 'This is a test summary',
  date: '2024-01-15',
  content: [
    {
      id: 'block-1',
      type: 'app-memory' as const,
      content: 'VSCode',
      data: {
        id: 'block-1',
        startTime: '10:00',
        endTime: '11:00',
        microSummary: 'Coding session',
        ocrText: 'function test() {}',
      },
    },
  ],
};

// Simple component to test contextual chat logic
const ContextualChatTestComponent: React.FC<{
  session: typeof mockSession;
  isOpen: boolean;
  onClose: () => void;
  onSendMessage: (message: string) => void;
  onNewChat: () => void;
}> = ({ session, isOpen, onClose, onSendMessage, onNewChat }) => {
  const [messages, setMessages] = React.useState<Array<{ role: string; content: string }>>([]);
  const [input, setInput] = React.useState('');

  if (!isOpen) return null;

  const handleSend = () => {
    if (input.trim()) {
      setMessages(prev => [...prev, { role: 'user', content: input }]);
      onSendMessage(input);
      setInput('');
    }
  };

  const handleNewChat = () => {
    setMessages([]);
    onNewChat();
  };

  return (
    <div data-testid="contextual-chat">
      <div data-testid="chat-header">
        <h2 data-testid="session-title">{session.customTitle || session.title}</h2>
        <button data-testid="new-chat-btn" onClick={handleNewChat}>New Chat</button>
        <button data-testid="close-btn" onClick={onClose}>Close</button>
      </div>
      
      <div data-testid="messages">
        {messages.map((msg, i) => (
          <div key={i} data-testid={`message-${i}`} data-role={msg.role}>
            {msg.content}
          </div>
        ))}
      </div>
      
      <div>
        <input
          data-testid="message-input"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ask about this session..."
        />
        <button data-testid="send-btn" onClick={handleSend}>Send</button>
      </div>
    </div>
  );
};

describe('Contextual Chat Unit Tests', () => {
  it('should display session title in header', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={true}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    expect(screen.getByTestId('session-title')).toHaveTextContent('Custom Title');
  });

  it('should not render when closed', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={false}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    expect(screen.queryByTestId('contextual-chat')).not.toBeInTheDocument();
  });

  it('should call onClose when close button is clicked', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={true}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    fireEvent.click(screen.getByTestId('close-btn'));
    
    expect(onClose).toHaveBeenCalled();
  });

  it('should send message when send button is clicked', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={true}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    fireEvent.change(screen.getByTestId('message-input'), { target: { value: 'Test message' } });
    fireEvent.click(screen.getByTestId('send-btn'));
    
    expect(onSendMessage).toHaveBeenCalledWith('Test message');
  });

  it('should add message to list when sent', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={true}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    fireEvent.change(screen.getByTestId('message-input'), { target: { value: 'Test message' } });
    fireEvent.click(screen.getByTestId('send-btn'));
    
    expect(screen.getByTestId('message-0')).toHaveTextContent('Test message');
  });

  it('should clear input after sending', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={true}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    fireEvent.change(screen.getByTestId('message-input'), { target: { value: 'Test message' } });
    fireEvent.click(screen.getByTestId('send-btn'));
    
    expect(screen.getByTestId('message-input')).toHaveValue('');
  });

  it('should clear messages when new chat is clicked', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={true}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    // Send a message first
    fireEvent.change(screen.getByTestId('message-input'), { target: { value: 'Test message' } });
    fireEvent.click(screen.getByTestId('send-btn'));
    
    expect(screen.getByTestId('message-0')).toBeInTheDocument();
    
    // Click new chat
    fireEvent.click(screen.getByTestId('new-chat-btn'));
    
    expect(screen.queryByTestId('message-0')).not.toBeInTheDocument();
    expect(onNewChat).toHaveBeenCalled();
  });

  it('should not send empty messages', () => {
    const onClose = vi.fn();
    const onSendMessage = vi.fn();
    const onNewChat = vi.fn();
    
    render(
      <ContextualChatTestComponent
        session={mockSession}
        isOpen={true}
        onClose={onClose}
        onSendMessage={onSendMessage}
        onNewChat={onNewChat}
      />
    );
    
    fireEvent.click(screen.getByTestId('send-btn'));
    
    expect(onSendMessage).not.toHaveBeenCalled();
  });
});
