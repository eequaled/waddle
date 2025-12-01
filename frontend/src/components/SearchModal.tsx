import React, { useState, useMemo } from 'react';
import {
    Dialog,
    DialogContent,
} from './ui/dialog';
import { Input } from './ui/input';
import {
    Search,
    X,
    CheckCircle2,
    MinusCircle,
    FileText,
    Tag,
    AlignLeft,
    Eye
} from 'lucide-react';
import { AppIcon } from './AppIcon';
import { AppType, Session } from '../types';
import { Badge } from './ui/badge';
import { Switch } from './ui/switch';
import { Label } from './ui/label';
import { ScrollArea } from './ui/scroll-area';

export interface SearchResult {
    session: Session;
    matchField: 'title' | 'summary' | 'tags' | 'ocrText';
    matchSnippet: string;
    blockId?: string;
}

interface SearchModalProps {
    isOpen: boolean;
    onClose: () => void;
    sessions: Session[];
    onSelectSession: (id: string, searchQuery?: string, blockId?: string) => void;
    onSetActiveView?: (view: 'timeline' | 'chat' | 'archives') => void;
}

// Helper to get snippet around match
function getMatchSnippet(text: string, query: string, contextLength: number = 40): string {
    const lowerText = text.toLowerCase();
    const lowerQuery = query.toLowerCase();
    const index = lowerText.indexOf(lowerQuery);
    
    if (index === -1) return text.substring(0, contextLength * 2) + '...';
    
    const start = Math.max(0, index - contextLength);
    const end = Math.min(text.length, index + query.length + contextLength);
    
    let snippet = '';
    if (start > 0) snippet += '...';
    snippet += text.substring(start, end);
    if (end < text.length) snippet += '...';
    
    return snippet;
}

// Helper to find which field matched
function findMatchField(session: Session, query: string): SearchResult | null {
    const lowerQuery = query.toLowerCase();
    
    // Check title
    const displayTitle = session.customTitle || session.title;
    if (displayTitle.toLowerCase().includes(lowerQuery)) {
        return {
            session,
            matchField: 'title',
            matchSnippet: getMatchSnippet(displayTitle, query),
        };
    }

    
    // Check summary
    const displaySummary = session.customSummary || session.summary;
    if (displaySummary.toLowerCase().includes(lowerQuery)) {
        return {
            session,
            matchField: 'summary',
            matchSnippet: getMatchSnippet(displaySummary, query),
        };
    }
    
    // Check tags
    const matchingTag = session.tags.find(t => t.toLowerCase().includes(lowerQuery));
    if (matchingTag) {
        return {
            session,
            matchField: 'tags',
            matchSnippet: `#${matchingTag}`,
        };
    }
    
    // Check OCR text in memory blocks
    for (const block of session.content) {
        if (block.type === 'app-memory' && block.data?.blocks) {
            for (const memBlock of block.data.blocks) {
                if (memBlock.ocrText && memBlock.ocrText.toLowerCase().includes(lowerQuery)) {
                    return {
                        session,
                        matchField: 'ocrText',
                        matchSnippet: getMatchSnippet(memBlock.ocrText, query),
                        blockId: memBlock.id,
                    };
                }
                if (memBlock.microSummary && memBlock.microSummary.toLowerCase().includes(lowerQuery)) {
                    return {
                        session,
                        matchField: 'ocrText',
                        matchSnippet: getMatchSnippet(memBlock.microSummary, query),
                        blockId: memBlock.id,
                    };
                }
            }
        }
    }
    
    return null;
}

// Field icon mapping
const fieldIcons = {
    title: FileText,
    summary: AlignLeft,
    tags: Tag,
    ocrText: Eye,
};

const fieldLabels = {
    title: 'Title',
    summary: 'Summary',
    tags: 'Tag',
    ocrText: 'Content',
};

export const SearchModal: React.FC<SearchModalProps> = ({
    isOpen,
    onClose,
    sessions,
    onSelectSession,
    onSetActiveView,
}) => {
    const [query, setQuery] = useState('');
    const [excludeMode, setExcludeMode] = useState(false);
    const [selectedApps, setSelectedApps] = useState<AppType[]>([]);
    const [dateFilter, setDateFilter] = useState<'All' | 'Today' | 'Yesterday' | 'Last Week'>('All');

    // Derive unique apps from sessions
    const availableApps = useMemo(() => {
        const apps = new Set<string>();
        sessions.forEach(session => {
            session.apps.forEach(app => apps.add(app));
        });
        return Array.from(apps).sort();
    }, [sessions]);

    const toggleApp = (app: AppType) => {
        if (selectedApps.includes(app)) {
            setSelectedApps(selectedApps.filter(a => a !== app));
        } else {
            setSelectedApps([...selectedApps, app]);
        }
    };


    const searchResults = useMemo((): SearchResult[] => {
        if (!query.trim()) {
            // No query - return all sessions with default match info
            return sessions
                .filter(session => {
                    // Date Filter
                    if (dateFilter !== 'All' && session.date !== dateFilter) return false;

                    // App Filter
                    if (selectedApps.length > 0) {
                        const hasExcludedApp = selectedApps.some(app => session.apps.includes(app));
                        if (excludeMode && hasExcludedApp) return false;
                        if (!excludeMode && !hasExcludedApp) return false;
                    }

                    return true;
                })
                .map(session => ({
                    session,
                    matchField: 'title' as const,
                    matchSnippet: session.customTitle || session.title,
                }));
        }

        const results: SearchResult[] = [];
        
        for (const session of sessions) {
            // Date Filter
            if (dateFilter !== 'All' && session.date !== dateFilter) continue;

            // App Filter
            if (selectedApps.length > 0) {
                const hasExcludedApp = selectedApps.some(app => session.apps.includes(app));
                if (excludeMode && hasExcludedApp) continue;
                if (!excludeMode && !hasExcludedApp) continue;
            }

            const match = findMatchField(session, query);
            if (match) {
                results.push(match);
            }
        }
        
        return results;
    }, [sessions, query, excludeMode, selectedApps, dateFilter]);

    const handleSelect = (result: SearchResult) => {
        // Set view to timeline for deep linking
        onSetActiveView?.('timeline');
        // Pass the search query for highlighting and blockId for scrolling
        onSelectSession(result.session.id, query, result.blockId);
        onClose();
    };

    // Highlight matching text in snippet
    const highlightMatch = (text: string, searchQuery: string) => {
        if (!searchQuery.trim()) return text;
        
        const regex = new RegExp(`(${searchQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
        const parts = text.split(regex);
        
        return parts.map((part, i) => 
            regex.test(part) ? (
                <mark key={i} className="bg-yellow-500/30 text-foreground rounded px-0.5">
                    {part}
                </mark>
            ) : part
        );
    };


    return (
        <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="max-w-3xl p-0 gap-0 bg-background border-border shadow-2xl overflow-hidden">
                <div className="p-6 border-b border-border space-y-4 bg-muted/5">
                    <div className="relative">
                        <Search className="absolute left-3 top-3 h-5 w-5 text-muted-foreground" />
                        <Input
                            placeholder="Search memories (e.g., 'Japan flights last week')"
                            className="pl-10 text-lg h-12 bg-background border-muted-foreground/20 focus-visible:ring-primary/30"
                            value={query}
                            onChange={(e) => setQuery(e.target.value)}
                            autoFocus
                        />
                        {query && (
                            <button onClick={() => setQuery('')} className="absolute right-3 top-3 text-muted-foreground hover:text-foreground">
                                <X size={16} />
                            </button>
                        )}
                    </div>

                    <div className="flex flex-col gap-4 pt-2">
                        <div className="flex items-center justify-between">
                            <div className="text-xs font-semibold text-muted-foreground tracking-wider uppercase">Filters</div>

                            <div className="flex items-center gap-2">
                                <Label htmlFor="exclude-mode" className={`text-xs font-medium cursor-pointer ${excludeMode ? 'text-destructive' : 'text-muted-foreground'}`}>
                                    {excludeMode ? 'Excluding Selected Apps' : 'Including Selected Apps'}
                                </Label>
                                <Switch
                                    id="exclude-mode"
                                    checked={excludeMode}
                                    onCheckedChange={setExcludeMode}
                                    className="data-[state=checked]:bg-destructive"
                                />
                            </div>
                        </div>

                        <div className="flex flex-wrap items-center gap-2">
                            {/* Date Presets */}
                            <div className="flex items-center border border-border rounded-md bg-background overflow-hidden mr-2">
                                {(['All', 'Today', 'Yesterday', 'Last Week'] as const).map(filter => (
                                    <button
                                        key={filter}
                                        onClick={() => setDateFilter(filter)}
                                        className={`px-3 py-1.5 text-xs font-medium transition-colors ${dateFilter === filter ? 'bg-primary text-primary-foreground' : 'hover:bg-muted text-muted-foreground'}`}
                                    >
                                        {filter}
                                    </button>
                                ))}
                            </div>

                            {/* App Toggles */}
                            <div className="h-6 w-px bg-border mx-2"></div>

                            {availableApps.map(app => {
                                const isSelected = selectedApps.includes(app);
                                return (
                                    <button
                                        key={app}
                                        onClick={() => toggleApp(app)}
                                        className={`
                                            flex items-center gap-1.5 px-2 py-1 rounded-md text-xs border transition-all
                                            ${isSelected
                                                ? excludeMode
                                                    ? 'border-destructive/50 bg-destructive/10 text-destructive'
                                                    : 'border-primary/50 bg-primary/10 text-primary'
                                                : 'border-transparent hover:bg-muted text-muted-foreground'
                                            }
                                        `}
                                    >
                                        <AppIcon app={app} className="w-3.5 h-3.5" />
                                        <span>{app}</span>
                                        {isSelected && (
                                            excludeMode ? <MinusCircle size={10} /> : <CheckCircle2 size={10} />
                                        )}
                                    </button>
                                );
                            })}
                        </div>
                    </div>
                </div>


                <ScrollArea className="h-[400px] bg-background">
                    <div className="p-2">
                        {searchResults.length === 0 ? (
                            <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
                                <Search className="w-8 h-8 mb-2 opacity-20" />
                                <p>No memories found matching your criteria.</p>
                            </div>
                        ) : (
                            <div className="space-y-1">
                                {searchResults.map(result => {
                                    const FieldIcon = fieldIcons[result.matchField];
                                    return (
                                        <div
                                            key={result.session.id}
                                            onClick={() => handleSelect(result)}
                                            className="group flex items-center gap-4 p-3 hover:bg-accent/50 rounded-lg cursor-pointer transition-colors"
                                        >
                                            <div className="flex flex-col items-center w-16 text-xs text-muted-foreground">
                                                <span className="font-medium text-foreground">{result.session.date}</span>
                                                <span>{result.session.startTime}</span>
                                            </div>

                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center gap-2 mb-1">
                                                    <h4 className="font-semibold text-sm group-hover:text-primary transition-colors truncate">
                                                        {result.session.customTitle || result.session.title}
                                                    </h4>
                                                    {query && (
                                                        <Badge variant="secondary" className="text-[10px] h-4 px-1.5 gap-1 shrink-0">
                                                            <FieldIcon size={10} />
                                                            {fieldLabels[result.matchField]}
                                                        </Badge>
                                                    )}
                                                    {result.session.tags.slice(0, 2).map(tag => (
                                                        <Badge key={tag} variant="outline" className="text-[10px] h-4 px-1 text-muted-foreground">
                                                            {tag}
                                                        </Badge>
                                                    ))}
                                                </div>
                                                <p className="text-xs text-muted-foreground line-clamp-1">
                                                    {query ? highlightMatch(result.matchSnippet, query) : result.matchSnippet}
                                                </p>
                                            </div>

                                            <div className="flex gap-1 opacity-50 group-hover:opacity-100 transition-opacity shrink-0">
                                                {result.session.apps.slice(0, 3).map(app => (
                                                    <AppIcon key={app} app={app} className="w-4 h-4" />
                                                ))}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        )}
                    </div>
                </ScrollArea>
            </DialogContent>
        </Dialog>
    );
};
