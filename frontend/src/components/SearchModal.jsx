import React, { useState, useMemo, useEffect } from 'react';
import {
    Dialog,
    DialogContent,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
    Search,
    Calendar as CalendarIcon,
    X,
    CheckCircle2,
    MinusCircle,
    History,
    Clock
} from 'lucide-react';
import { AppIcon } from './AppIcon';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { format } from 'date-fns';
import { cn } from '@/lib/utils';

const APPS_LIST = ['Chrome', 'Slack', 'Spotify', 'VS Code', 'Figma', 'Notes', 'Excel', 'Zoom'];

export const SearchModal = ({
    isOpen,
    onClose,
    sessions,
    onSelectSession
}) => {
    const [query, setQuery] = useState('');
    const [excludeMode, setExcludeMode] = useState(false);
    const [selectedApps, setSelectedApps] = useState([]);
    const [dateFilter, setDateFilter] = useState('All'); // 'All' | 'Today' | 'Yesterday' | 'Last Week' | 'Custom'
    const [dateRange, setDateRange] = useState(undefined);
    const [searchHistory, setSearchHistory] = useState([]);

    // Load history on mount
    useEffect(() => {
        const saved = localStorage.getItem('search_history');
        if (saved) {
            try {
                setSearchHistory(JSON.parse(saved));
            } catch (e) {
                console.error("Failed to parse search history", e);
            }
        }
    }, []);

    const addToHistory = (term) => {
        if (!term.trim()) return;
        const newHistory = [term, ...searchHistory.filter(h => h !== term)].slice(0, 10);
        setSearchHistory(newHistory);
        localStorage.setItem('search_history', JSON.stringify(newHistory));
    };

    const toggleApp = (app) => {
        if (selectedApps.includes(app)) {
            setSelectedApps(selectedApps.filter(a => a !== app));
        } else {
            setSelectedApps([...selectedApps, app]);
        }
    };

    const filteredSessions = useMemo(() => {
        if (!query && selectedApps.length === 0 && dateFilter === 'All') return [];

        return sessions.filter(session => {
            // Text Search
            const matchesText =
                !query ||
                session.title.toLowerCase().includes(query.toLowerCase()) ||
                (session.summary && session.summary.toLowerCase().includes(query.toLowerCase())) ||
                session.tags.some(t => t.toLowerCase().includes(query.toLowerCase()));

            if (!matchesText) return false;

            // Date Filter
            if (dateFilter !== 'All') {
                if (dateFilter === 'Custom' && dateRange?.from) {
                    // Simple string comparison for mock data (assuming YYYY-MM-DD or similar consistent format would be better in real app)
                    // For now, we skip strict date checking on mock data unless it matches exact strings like "Today"
                    // In a real app, we'd parse session.date
                } else if (session.date !== dateFilter && dateFilter !== 'Custom') {
                    return false;
                }
            }

            // App Filter
            if (selectedApps.length > 0) {
                const sessionHasApp = selectedApps.some(app => session.apps.includes(app));

                if (excludeMode) {
                    const hasExcludedApp = selectedApps.some(app => session.apps.includes(app));
                    if (hasExcludedApp) return false;
                } else {
                    if (!sessionHasApp) return false;
                }
            }

            return true;
        });
    }, [sessions, query, excludeMode, selectedApps, dateFilter, dateRange]);

    const handleSelect = (id) => {
        addToHistory(query);
        onSelectSession(id);
        onClose();
    };

    const handleHistoryClick = (term) => {
        setQuery(term);
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
                                {['All', 'Today', 'Yesterday', 'Last Week'].map(filter => (
                                    <button
                                        key={filter}
                                        onClick={() => setDateFilter(filter)}
                                        className={`px-3 py-1.5 text-xs font-medium transition-colors ${dateFilter === filter ? 'bg-primary text-primary-foreground' : 'hover:bg-muted text-muted-foreground'}`}
                                    >
                                        {filter}
                                    </button>
                                ))}
                                <Popover>
                                    <PopoverTrigger asChild>
                                        <button
                                            onClick={() => setDateFilter('Custom')}
                                            className={`px-3 py-1.5 text-xs font-medium transition-colors border-l border-border flex items-center gap-1 ${dateFilter === 'Custom' ? 'bg-primary text-primary-foreground' : 'hover:bg-muted text-muted-foreground'}`}
                                        >
                                            <CalendarIcon size={12} />
                                            {dateRange?.from ? (
                                                dateRange.to ? (
                                                    <>
                                                        {format(dateRange.from, "LLL dd")} - {format(dateRange.to, "LLL dd")}
                                                    </>
                                                ) : (
                                                    format(dateRange.from, "LLL dd")
                                                )
                                            ) : (
                                                <span>Custom</span>
                                            )}
                                        </button>
                                    </PopoverTrigger>
                                    <PopoverContent className="w-auto p-0" align="start">
                                        <Calendar
                                            initialFocus
                                            mode="range"
                                            defaultMonth={dateRange?.from}
                                            selected={dateRange}
                                            onSelect={setDateRange}
                                            numberOfMonths={2}
                                        />
                                    </PopoverContent>
                                </Popover>
                            </div>

                            {/* App Toggles */}
                            <div className="h-6 w-px bg-border mx-2"></div>

                            {APPS_LIST.map(app => {
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
                        {!query && selectedApps.length === 0 && dateFilter === 'All' ? (
                            // Empty State / History
                            <div className="p-4">
                                <h3 className="text-xs font-semibold text-muted-foreground uppercase mb-3 flex items-center gap-2">
                                    <History size={14} />
                                    Recent Searches
                                </h3>
                                {searchHistory.length > 0 ? (
                                    <div className="space-y-1">
                                        {searchHistory.map((term, i) => (
                                            <button
                                                key={i}
                                                onClick={() => handleHistoryClick(term)}
                                                className="w-full text-left px-3 py-2 text-sm text-foreground hover:bg-accent rounded-md flex items-center gap-2 group"
                                            >
                                                <Clock size={14} className="text-muted-foreground group-hover:text-primary" />
                                                {term}
                                            </button>
                                        ))}
                                    </div>
                                ) : (
                                    <p className="text-sm text-muted-foreground italic">No recent searches</p>
                                )}
                            </div>
                        ) : filteredSessions.length === 0 ? (
                            <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
                                <Search className="w-8 h-8 mb-2 opacity-20" />
                                <p>No memories found matching your criteria.</p>
                            </div>
                        ) : (
                            <div className="space-y-1">
                                {filteredSessions.map(session => (
                                    <div
                                        key={session.id}
                                        onClick={() => handleSelect(session.id)}
                                        className="group flex items-center gap-4 p-3 hover:bg-accent/50 rounded-lg cursor-pointer transition-colors"
                                    >
                                        <div className="flex flex-col items-center w-16 text-xs text-muted-foreground">
                                            <span className="font-medium text-foreground">{session.date}</span>
                                            <span>{session.startTime}</span>
                                        </div>

                                        <div className="flex-1">
                                            <div className="flex items-center gap-2 mb-1">
                                                <h4 className="font-semibold text-sm group-hover:text-primary transition-colors">
                                                    {session.title}
                                                </h4>
                                                {session.tags.slice(0, 2).map(tag => (
                                                    <Badge key={tag} variant="outline" className="text-[10px] h-4 px-1 text-muted-foreground">
                                                        {tag}
                                                    </Badge>
                                                ))}
                                            </div>
                                            <p className="text-xs text-muted-foreground line-clamp-1">
                                                {session.summary || "No summary available"}
                                            </p>
                                        </div>

                                        <div className="flex gap-1 opacity-50 group-hover:opacity-100 transition-opacity">
                                            {session.apps.slice(0, 3).map(app => (
                                                <AppIcon key={app} app={app} className="w-4 h-4" />
                                            ))}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </ScrollArea>
            </DialogContent>
        </Dialog>
    );
};
