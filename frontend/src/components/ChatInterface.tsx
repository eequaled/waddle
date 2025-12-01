import React, { useState, useEffect, useRef } from 'react';
import { Send, History, Bot, User, RefreshCw, Zap } from 'lucide-react';
import { api } from '../services/api';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { ScrollArea } from './ui/scroll-area';
import { Card } from './ui/card';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

// Quick query templates
const QUICK_QUERIES = [
  { label: "What was I working on?", query: "Summarize my recent activity across all sessions. What projects or tasks have I been focused on?" },
  { label: "Today's summary", query: "Give me a summary of everything I did today, including apps used and time spent." },
  { label: "Weekly overview", query: "Provide an overview of my work this week. What were the main themes and accomplishments?" },
];

interface Message {
    role: 'user' | 'assistant';
    content: string;
    timestamp: string;
}

interface ChatSession {
    id: string;
    context: string;
    messages: Message[];
    updatedAt: string;
}

interface ChatInterfaceProps {
    allSessions?: any[];
}

// Proactive suggestions component
const ProactiveSuggestions: React.FC<{
    query: string;
    sessions: any[];
    onSelectSuggestion: (text: string) => void;
}> = ({ query, sessions, onSelectSuggestion }) => {
    const suggestions = React.useMemo(() => {
        const queryLower = query.toLowerCase();
        const matches: { session: any; reason: string }[] = [];
        
        sessions.forEach(session => {
            const title = (session.customTitle || session.title || '').toLowerCase();
            const summary = (session.customSummary || session.summary || '').toLowerCase();
            const tags = (session.tags || []).join(' ').toLowerCase();
            
            if (title.includes(queryLower) || summary.includes(queryLower) || tags.includes(queryLower)) {
                matches.push({
                    session,
                    reason: `You worked on something similar on ${session.date}`
                });
            }
        });
        
        return matches.slice(0, 3);
    }, [query, sessions]);

    if (suggestions.length === 0) return null;

    return (
        <div className="px-4 py-2 border-t border-border/50 bg-muted/30">
            <p className="text-xs text-muted-foreground mb-2">Related memories:</p>
            <div className="flex flex-wrap gap-2">
                {suggestions.map((s, i) => (
                    <button
                        key={i}
                        className="text-xs px-2 py-1 bg-background border border-border rounded hover:bg-accent transition-colors text-left"
                        onClick={() => onSelectSuggestion(`(referencing ${s.session.date})`)}
                    >
                        <span className="font-medium">{s.session.customTitle || s.session.title}</span>
                        <span className="text-muted-foreground ml-1">• {s.session.date}</span>
                    </button>
                ))}
            </div>
        </div>
    );
};

export function ChatInterface({ allSessions = [] }: ChatInterfaceProps) {
    const [messages, setMessages] = useState<Message[]>([]);
    const [input, setInput] = useState('');
    const [loading, setLoading] = useState(false);
    const [historyOpen, setHistoryOpen] = useState(false);
    const [sessions, setSessions] = useState<ChatSession[]>([]);
    const [showContextInfo, setShowContextInfo] = useState(false);
    const [usedContext, setUsedContext] = useState<string[]>([]);
    const scrollRef = useRef<HTMLDivElement>(null);

    // Build smart context from recent and relevant sessions
    const buildSmartContext = (userQuery: string): { context: string; sessionRefs: string[] } => {
        if (allSessions.length === 0) {
            return { context: 'global', sessionRefs: [] };
        }

        const queryLower = userQuery.toLowerCase();
        const sessionRefs: string[] = [];
        let contextParts: string[] = [];

        // Always include recent sessions (last 3 days)
        const threeDaysAgo = new Date();
        threeDaysAgo.setDate(threeDaysAgo.getDate() - 3);
        
        const recentSessions = allSessions.filter(s => {
            const sessionDate = new Date(s.date);
            return sessionDate >= threeDaysAgo;
        }).slice(0, 5);

        // Find sessions relevant to the query
        const relevantSessions = allSessions.filter(s => {
            const title = (s.customTitle || s.title || '').toLowerCase();
            const summary = (s.customSummary || s.summary || '').toLowerCase();
            const tags = (s.tags || []).join(' ').toLowerCase();
            const apps = (s.apps || []).join(' ').toLowerCase();
            
            return title.includes(queryLower) || 
                   summary.includes(queryLower) ||
                   tags.includes(queryLower) ||
                   apps.includes(queryLower);
        }).slice(0, 3);

        // Combine and deduplicate
        const sessionsToInclude = [...new Map(
            [...recentSessions, ...relevantSessions].map(s => [s.id, s])
        ).values()].slice(0, 5);

        sessionsToInclude.forEach(session => {
            sessionRefs.push(session.customTitle || session.title || session.date);
            contextParts.push(`
Session: ${session.customTitle || session.title}
Date: ${session.date}
Apps: ${(session.apps || []).join(', ')}
Summary: ${session.customSummary || session.summary}
            `.trim());
        });

        return {
            context: contextParts.length > 0 
                ? `Here is context from the user's recent activity:\n\n${contextParts.join('\n\n---\n\n')}`
                : 'global',
            sessionRefs
        };
    };

    useEffect(() => {
        loadHistory();
    }, []);

    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [messages]);

    const loadHistory = async () => {
        try {
            const data = await api.getChatHistory();
            setSessions(data);
            // Load the most recent session if available
            if (data.length > 0) {
                setMessages(data[0].messages);
            }
        } catch (error) {
            console.error('Failed to load history:', error);
        }
    };

    const handleSend = async () => {
        if (!input.trim()) return;

        const userMsg: Message = { role: 'user', content: input, timestamp: new Date().toISOString() };
        setMessages(prev => [...prev, userMsg]);
        const currentInput = input;
        setInput('');
        setLoading(true);

        try {
            // Build smart context based on query and recent sessions
            const { context, sessionRefs } = buildSmartContext(currentInput);
            setUsedContext(sessionRefs);
            
            const response = await api.chat(context, currentInput);
            const aiMsg: Message = { role: 'assistant', content: response.content, timestamp: response.timestamp };
            setMessages(prev => [...prev, aiMsg]);
        } catch (error) {
            console.error('Chat failed:', error);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="flex h-full bg-background text-foreground relative">
            {/* Main Chat Area */}
            <div className="flex-1 flex flex-col h-full">
                {/* Header */}
                <div className="h-14 border-b border-border flex items-center justify-between px-4 bg-card/50 backdrop-blur-sm">
                    <div className="flex items-center gap-2">
                        <Bot className="w-5 h-5 text-primary" />
                        <h2 className="font-semibold">Global Memory Chat</h2>
                        {usedContext.length > 0 && (
                            <Button
                                variant="ghost"
                                size="sm"
                                className="text-xs text-muted-foreground gap-1"
                                onClick={() => setShowContextInfo(!showContextInfo)}
                            >
                                <Zap size={12} className="text-amber-500" />
                                {usedContext.length} sessions in context
                            </Button>
                        )}
                    </div>
                    <div className="flex items-center gap-1">
                        <Button 
                            variant="ghost" 
                            size="sm" 
                            onClick={() => { setMessages([]); setUsedContext([]); }}
                            className="gap-1"
                        >
                            <RefreshCw className="w-4 h-4" />
                            New Chat
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => setHistoryOpen(!historyOpen)}>
                            <History className="w-5 h-5" />
                        </Button>
                    </div>
                </div>

                {/* Context Info Panel */}
                {showContextInfo && usedContext.length > 0 && (
                    <div className="px-4 py-2 bg-amber-500/10 border-b border-amber-500/20">
                        <p className="text-xs text-amber-600 dark:text-amber-400 font-medium mb-1">
                            AI is using context from these sessions:
                        </p>
                        <div className="flex flex-wrap gap-1">
                            {usedContext.map((ref, i) => (
                                <span key={i} className="text-xs px-2 py-0.5 bg-amber-500/20 rounded">
                                    {ref}
                                </span>
                            ))}
                        </div>
                    </div>
                )}

                {/* Messages */}
                <div className="flex-1 min-h-0 overflow-hidden">
                    <ScrollArea className="h-full">
                        <div className="space-y-4 max-w-3xl mx-auto p-4">
                            {messages.map((msg, idx) => (
                                <div key={idx} className={`flex gap-3 ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                                    {msg.role === 'assistant' && (
                                        <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                                            <Bot className="w-5 h-5 text-primary" />
                                        </div>
                                    )}
                                    <div className={`max-w-[80%] p-3 rounded-lg ${msg.role === 'user'
                                        ? 'bg-primary text-primary-foreground'
                                        : 'bg-muted/50'
                                        }`}>
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
                                        <div className="w-8 h-8 rounded-full bg-secondary flex items-center justify-center shrink-0">
                                            <User className="w-5 h-5" />
                                        </div>
                                    )}
                                </div>
                            ))}
                            {loading && (
                                <div className="flex gap-3">
                                    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                                        <Bot className="w-5 h-5 text-primary" />
                                    </div>
                                    <div className="bg-muted/50 p-3 rounded-lg">
                                        <div className="flex gap-1">
                                            <span className="w-2 h-2 bg-primary/50 rounded-full animate-bounce" />
                                            <span className="w-2 h-2 bg-primary/50 rounded-full animate-bounce delay-75" />
                                            <span className="w-2 h-2 bg-primary/50 rounded-full animate-bounce delay-150" />
                                        </div>
                                    </div>
                                </div>
                            )}
                            <div ref={scrollRef} />
                        </div>
                    </ScrollArea>
                </div>

                {/* Quick Queries */}
                {messages.length === 0 && (
                    <div className="px-4 pb-2">
                        <div className="max-w-3xl mx-auto">
                            <p className="text-xs text-muted-foreground mb-2">Quick queries:</p>
                            <div className="flex flex-wrap gap-2">
                                {QUICK_QUERIES.map((q, i) => (
                                    <Button
                                        key={i}
                                        variant="outline"
                                        size="sm"
                                        className="gap-1 text-xs"
                                        onClick={() => {
                                            setInput(q.query);
                                            // Auto-send after a brief delay
                                            setTimeout(() => {
                                                const userMsg: Message = { role: 'user', content: q.query, timestamp: new Date().toISOString() };
                                                setMessages(prev => [...prev, userMsg]);
                                                setInput('');
                                                setLoading(true);
                                                api.chat('global', q.query).then(response => {
                                                    const aiMsg: Message = { role: 'assistant', content: response.content, timestamp: response.timestamp };
                                                    setMessages(prev => [...prev, aiMsg]);
                                                }).catch(console.error).finally(() => setLoading(false));
                                            }, 100);
                                        }}
                                    >
                                        <Zap size={12} />
                                        {q.label}
                                    </Button>
                                ))}
                            </div>
                        </div>
                    </div>
                )}

                {/* Proactive Suggestions */}
                {input.length > 3 && allSessions.length > 0 && (
                    <ProactiveSuggestions 
                        query={input} 
                        sessions={allSessions} 
                        onSelectSuggestion={(text) => setInput(prev => prev + ' ' + text)}
                    />
                )}

                {/* Input */}
                <div className="p-4 border-t border-border bg-background">
                    <div className="max-w-3xl mx-auto flex gap-2">
                        <Input
                            value={input}
                            onChange={e => setInput(e.target.value)}
                            onKeyDown={e => e.key === 'Enter' && handleSend()}
                            placeholder="Ask about your memories..."
                            className="flex-1"
                        />
                        <Button onClick={handleSend} disabled={loading}>
                            <Send className="w-4 h-4" />
                        </Button>
                    </div>
                </div>
            </div>

            {/* History Sidebar (Overlay) */}
            {historyOpen && (
                <div className="absolute top-0 right-0 w-80 h-full bg-card border-l border-border shadow-2xl z-20 flex flex-col animate-in slide-in-from-right">
                    <div className="p-4 border-b border-border flex justify-between items-center">
                        <h3 className="font-semibold">Chat History</h3>
                        <Button variant="ghost" size="sm" onClick={() => setHistoryOpen(false)}>Close</Button>
                    </div>
                    <ScrollArea className="flex-1">
                        <div className="p-2 space-y-2">
                            {sessions.map(session => {
                                const firstMessage = session.messages[0]?.content || 'Empty Chat';
                                const preview = firstMessage.length > 60 
                                    ? firstMessage.substring(0, 60) + '...' 
                                    : firstMessage;
                                return (
                                    <Card
                                        key={session.id}
                                        className="p-3 cursor-pointer hover:bg-accent/50 transition-colors"
                                        onClick={() => {
                                            setMessages(session.messages);
                                            setHistoryOpen(false);
                                        }}
                                    >
                                        <p className="text-sm font-medium truncate">
                                            {preview}
                                        </p>
                                        <p className="text-xs text-muted-foreground mt-1">
                                            {new Date(session.updatedAt).toLocaleDateString()} at {new Date(session.updatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                        </p>
                                        <p className="text-xs text-muted-foreground">
                                            {session.messages.length} messages
                                        </p>
                                    </Card>
                                );
                            })}
                        </div>
                    </ScrollArea>
                </div>
            )}
        </div>
    );
}
