import React, { useState, useEffect } from 'react';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle
} from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { X, Plus, Ban } from 'lucide-react';
import { ScrollArea } from './ui/scroll-area';

import { api } from '../services/api';
import { Switch } from './ui/switch';
import { Label } from './ui/label';
import { Moon, Sun } from 'lucide-react';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
    currentTheme: 'light' | 'dark';
    onThemeChange: (theme: 'light' | 'dark') => void;
    isDuckMode: boolean;
    onDuckModeChange: (enabled: boolean) => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
    isOpen,
    onClose,
    currentTheme,
    onThemeChange,
    isDuckMode,
    onDuckModeChange
}) => {
    const [blacklist, setBlacklist] = useState<string[]>([]);
    const [newApp, setNewApp] = useState('');

    useEffect(() => {
        if (isOpen) {
            api.getBlacklist().then(setBlacklist).catch(console.error);
        }
    }, [isOpen]);

    const handleAdd = async () => {
        if (newApp && !blacklist.includes(newApp)) {
            const newList = [...blacklist, newApp];
            setBlacklist(newList);
            setNewApp('');
            await api.setBlacklist(newList);
        }
    };

    const handleRemove = async (app: string) => {
        const newList = blacklist.filter(a => a !== app);
        setBlacklist(newList);
        await api.setBlacklist(newList);
    };

    return (
        <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="max-w-md bg-background/95 backdrop-blur-xl border-border">
                <DialogHeader>
                    <DialogTitle>Settings</DialogTitle>
                </DialogHeader>

                <div className="space-y-6">
                    {/* Theme Settings */}
                    <div className="space-y-4">
                        <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">Appearance</h3>

                        <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                                <Label className="text-base">Dark Mode</Label>
                                <p className="text-xs text-muted-foreground">
                                    Switch between light and dark themes
                                </p>
                            </div>
                            <div className="flex items-center gap-2">
                                <Sun className="h-4 w-4 text-muted-foreground" />
                                <Switch
                                    checked={currentTheme === 'dark'}
                                    onCheckedChange={(checked) => onThemeChange(checked ? 'dark' : 'light')}
                                />
                                <Moon className="h-4 w-4 text-muted-foreground" />
                            </div>
                        </div>

                        <div className="flex items-center justify-between border-t border-border pt-4">
                            <div className="space-y-0.5">
                                <Label className="text-base">Duck Mode 🦆</Label>
                                <p className="text-xs text-muted-foreground">
                                    Enable enhanced duck-themed visuals
                                </p>
                            </div>
                            <Switch
                                checked={isDuckMode}
                                onCheckedChange={onDuckModeChange}
                            />
                        </div>
                    </div>

                    {/* Blacklist Settings */}
                    <div className="space-y-3">
                        <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-2">
                            <Ban className="w-4 h-4" />
                            Blacklisted Apps
                        </h3>

                        <div className="flex gap-2">
                            <Input
                                placeholder="App name (e.g., notepad.exe)"
                                value={newApp}
                                onChange={(e) => setNewApp(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
                            />
                            <Button onClick={handleAdd} size="icon">
                                <Plus size={18} />
                            </Button>
                        </div>

                        <ScrollArea className="h-[300px] border border-border rounded-md p-2">
                            <div className="space-y-2">
                                {blacklist.map(app => (
                                    <div key={app} className="flex items-center justify-between p-2 bg-muted/50 rounded-md group">
                                        <span className="text-sm font-mono">{app}</span>
                                        <button
                                            onClick={() => handleRemove(app)}
                                            className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                                        >
                                            <X size={16} />
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </ScrollArea>
                    </div>

                    <div className="flex justify-end">
                        <Button onClick={onClose}>Done</Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
};
