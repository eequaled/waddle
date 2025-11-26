import React, { useState, useMemo } from 'react';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle
} from './ui/dialog';
import { Input } from './ui/input';
import { Button } from './ui/button';
import {
    Search,
    Calendar,
    Filter,
    X,
    CheckCircle2,
    MinusCircle
} from 'lucide-react';
import { AppIcon } from './AppIcon';
import { AppType, Session } from '../types';
import { Badge } from './ui/badge';
import { Switch } from './ui/switch';
import { Label } from './ui/label';
import { ScrollArea } from './ui/scroll-area';

interface SearchModalProps {
    isOpen: boolean;
    onClose: () => void;
    sessions: Session[];
    onSelectSession: (id: string) => void;
}

// Removed hardcoded APPS_LIST

export const SearchModal: React.FC<SearchModalProps> = ({
    isOpen,
    onClose,
    sessions,
    onSelectSession
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

    const filteredSessions = useMemo(() => {
        return sessions.filter(session => {
            // Text Search
            const matchesText =
                session.title.toLowerCase().includes(query.toLowerCase()) ||
                session.summary.toLowerCase().includes(query.toLowerCase()) ||
                session.tags.some(t => t.toLowerCase().includes(query.toLowerCase()));

            if (!matchesText) return false;

            // Date Filter
            if (dateFilter !== 'All' && session.date !== dateFilter) return false;

            // App Filter
            if (selectedApps.length > 0) {
                const sessionHasApp = selectedApps.some(app => session.apps.includes(app));

                if (excludeMode) {
                    // If Exclude Mode is ON, we want sessions that DO NOT have the selected apps
                    // Wait, the prompt says: "Result: The search returns activities where those apps were not active."
                    // So if Slack is selected in Exclude mode, show sessions WITHOUT Slack.
                    const hasExcludedApp = selectedApps.some(app => session.apps.includes(app));
                    if (hasExcludedApp) return false;
                } else {
                    // Normal mode: Show sessions that HAVE at least one of the selected apps? 
                    // Or MUST have ALL? Usually "Included Apps" means at least one or subset.
                    // Let's go with "Contains at least one of the selected apps" for flexibility.
                    if (!sessionHasApp) return false;
                }
            }

            return true;
        });
    }, [sessions, query, excludeMode, selectedApps, dateFilter]);

    const handleSelect = (id: string) => {
        onSelectSession(id);
        onClose();
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
                        {filteredSessions.length === 0 ? (
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
                                                {session.summary}
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
