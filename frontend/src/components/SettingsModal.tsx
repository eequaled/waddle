import React, { useState, useEffect } from 'react';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle
} from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { X, Plus, Ban, Shield, Trash2, HardDrive } from 'lucide-react';
import { ScrollArea } from './ui/scroll-area';

import { api } from '../services/api';
import { Switch } from './ui/switch';
import { Label } from './ui/label';
import { Moon, Sun } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

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
    const [isPrivateMode, setIsPrivateMode] = useState(false);
    const [dataRetention, setDataRetention] = useState('30');
    const [storageUsed] = useState('0 MB');

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

                    {/* Privacy Settings */}
                    <div className="space-y-4">
                        <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-2">
                            <Shield className="w-4 h-4" />
                            Privacy
                        </h3>

                        <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                                <Label className="text-base">Private Mode</Label>
                                <p className="text-xs text-muted-foreground">
                                    Temporarily pause all screen recording
                                </p>
                            </div>
                            <Switch
                                checked={isPrivateMode}
                                onCheckedChange={setIsPrivateMode}
                            />
                        </div>

                        <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                                <Label className="text-base">Data Retention</Label>
                                <p className="text-xs text-muted-foreground">
                                    Auto-delete sessions older than
                                </p>
                            </div>
                            <Select value={dataRetention} onValueChange={setDataRetention}>
                                <SelectTrigger className="w-32">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="7">7 days</SelectItem>
                                    <SelectItem value="30">30 days</SelectItem>
                                    <SelectItem value="90">90 days</SelectItem>
                                    <SelectItem value="365">1 year</SelectItem>
                                    <SelectItem value="forever">Forever</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                    </div>

                    {/* Blacklist Settings */}
                    <div className="space-y-3">
                        <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-2">
                            <Ban className="w-4 h-4" />
                            Excluded Apps
                        </h3>
                        <p className="text-xs text-muted-foreground">
                            These apps will not be recorded
                        </p>

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

                        <ScrollArea className="h-[150px] border border-border rounded-md p-2">
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

                    {/* Storage Settings */}
                    <div className="space-y-3">
                        <h3 className="text-sm font-medium text-muted-foreground uppercase tracking-wider flex items-center gap-2">
                            <HardDrive className="w-4 h-4" />
                            Storage
                        </h3>

                        <div className="flex items-center justify-between p-3 bg-muted/30 rounded-lg">
                            <div>
                                <p className="text-sm font-medium">Storage Used</p>
                                <p className="text-xs text-muted-foreground">Estimated disk usage</p>
                            </div>
                            <span className="text-lg font-bold">{storageUsed}</span>
                        </div>

                        <Button variant="outline" className="w-full gap-2 text-destructive hover:text-destructive">
                            <Trash2 size={16} />
                            Clear All Data
                        </Button>
                    </div>

                    <div className="flex justify-end">
                        <Button onClick={onClose}>Done</Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
};
