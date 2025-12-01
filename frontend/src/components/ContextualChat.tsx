import React, { useState, useEffect, useRef } from 'react';
import { Session, BlockData } from '../types';
import { api } from '../services/api';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { ScrollArea } from './ui/scroll-area';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from './ui/sheet';
import { Send, Bot, User, RefreshCw, Sparkles } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface Message {
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
}

interface ContextualChatProps {
  session: Session;
  initialBlock?: BlockData;
  isOpen: boolean;
  onClose: () => void;
}

export const ContextualChat: React.FC<ContextualChatProps> = ({
  session,
  initialBlock,
  isOpen,
  onClose,
}) => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Get display title
  const displayTitle = session.customTitle || session.title;
  
  // Build header with optional block timestamp
  const headerSubtitle = initialBlock 
    ? `Context: ${initialBlock.startTime}` 
    : `Session: ${session.date}`;


  // Scroll to bottom when messages change
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  // Build context from session memory blocks
  const buildSessionContext = (): string => {
    const blocks: string[] = [];
    
    session.content.forEach(block => {
      if (block.type === 'app-memory' && block.data?.blocks) {
        block.data.blocks.forEach((memBlock: BlockData) => {
          if (memBlock.microSummary) {
            blocks.push(`[${memBlock.startTime}] ${memBlock.microSummary}`);
          }
          if (memBlock.ocrText) {
            blocks.push(`OCR: ${memBlock.ocrText.substring(0, 500)}`);
          }
        });
      }
    });

    // If we have an initial block, prioritize it
    if (initialBlock) {
      return `Focus on this specific moment:\n[${initialBlock.startTime}] ${initialBlock.microSummary}\n\nFull session context:\n${blocks.join('\n')}`;
    }

    return blocks.join('\n');
  };

  const handleSend = async () => {
    if (!input.trim()) return;

    const userMsg: Message = { 
      role: 'user', 
      content: input, 
      timestamp: new Date().toISOString() 
    };
    setMessages(prev => [...prev, userMsg]);
    setInput('');
    setLoading(true);

    try {
      const context = buildSessionContext();
      const response = await api.chat(context, input, session.id);
      const aiMsg: Message = { 
        role: 'assistant', 
        content: response.content, 
        timestamp: response.timestamp 
      };
      setMessages(prev => [...prev, aiMsg]);
    } catch (error) {
      console.error('Chat failed:', error);
      const errorMsg: Message = {
        role: 'assistant',
        content: 'Sorry, I encountered an error. Please try again.',
        timestamp: new Date().toISOString(),
      };
      setMessages(prev => [...prev, errorMsg]);
    } finally {
      setLoading(false);
    }
  };

  const handleNewChat = () => {
    setMessages([]);
  };


  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-lg flex flex-col p-0">
        {/* Header */}
        <SheetHeader className="border-b border-border px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-primary" />
              <div>
                <SheetTitle className="text-base">{displayTitle}</SheetTitle>
                <SheetDescription className="text-xs">
                  {headerSubtitle}
                </SheetDescription>
              </div>
            </div>
            <Button 
              variant="ghost" 
              size="sm" 
              onClick={handleNewChat}
              className="gap-1"
            >
              <RefreshCw size={14} />
              New Chat
            </Button>
          </div>
        </SheetHeader>

        {/* Messages */}
        <div className="flex-1 min-h-0 overflow-hidden">
          <ScrollArea className="h-full">
            <div className="space-y-4 p-4">
              {messages.length === 0 && (
                <div className="text-center text-muted-foreground py-8">
                  <Sparkles className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Ask me anything about this session</p>
                  <p className="text-xs mt-1">I have context from all memory blocks</p>
                </div>
              )}
              
              {messages.map((msg, idx) => (
                <div 
                  key={idx} 
                  className={`flex gap-3 ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  {msg.role === 'assistant' && (
                    <div className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                      <Bot className="w-4 h-4 text-primary" />
                    </div>
                  )}
                  <div 
                    className={`max-w-[85%] p-3 rounded-lg ${
                      msg.role === 'user'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted/50'
                    }`}
                  >
                    {msg.role === 'assistant' ? (
                      <div className="prose prose-sm dark:prose-invert max-w-none">
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>
                          {msg.content}
                        </ReactMarkdown>
                      </div>
                    ) : (
                      <p className="whitespace-pre-wrap text-sm">{msg.content}</p>
                    )}
                  </div>
                  {msg.role === 'user' && (
                    <div className="w-7 h-7 rounded-full bg-secondary flex items-center justify-center shrink-0">
                      <User className="w-4 h-4" />
                    </div>
                  )}
                </div>
              ))}
              
              {loading && (
                <div className="flex gap-3">
                  <div className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                    <Bot className="w-4 h-4 text-primary" />
                  </div>
                  <div className="bg-muted/50 p-3 rounded-lg">
                    <div className="flex gap-1">
                      <span className="w-2 h-2 bg-primary/50 rounded-full animate-bounce" />
                      <span className="w-2 h-2 bg-primary/50 rounded-full animate-bounce [animation-delay:75ms]" />
                      <span className="w-2 h-2 bg-primary/50 rounded-full animate-bounce [animation-delay:150ms]" />
                    </div>
                  </div>
                </div>
              )}
              <div ref={scrollRef} />
            </div>
          </ScrollArea>
        </div>

        {/* Input */}
        <div className="p-4 border-t border-border bg-background">
          <div className="flex gap-2">
            <Input
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && !e.shiftKey && handleSend()}
              placeholder="Ask about this session..."
              className="flex-1"
            />
            <Button onClick={handleSend} disabled={loading} size="icon">
              <Send className="w-4 h-4" />
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
};
