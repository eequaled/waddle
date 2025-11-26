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

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
}

// Mock blacklist for now since we don't have a write endpoint yet
// In a real app, we'd fetch this from the backend
const DEFAULT_BLACKLIST = [
    'PickerHost.exe',
    'SearchHost.exe',
    'StartMenuExperienceHost.exe',
    'LockApp.exe',
    'backgroundTaskHost.exe',
    'ApplicationFrameHost.exe',
    'ShellExperienceHost.exe',
    'TextInputHost.exe',
    'SystemSettings.exe',
    'Taskmgr.exe',
    'explorer.exe'
];

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose }) => {
    const [blacklist, setBlacklist] = useState<string[]>(DEFAULT_BLACKLIST);
    const [newApp, setNewApp] = useState('');

    const handleAdd = () => {
        if (newApp && !blacklist.includes(newApp)) {
            setBlacklist([...blacklist, newApp]);
            setNewApp('');
        }
    };

    const handleRemove = (app: string) => {
        setBlacklist(blacklist.filter(a => a !== app));
    };

    return (
        <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="max-w-md bg-background border-border">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Ban className="w-5 h-5 text-destructive" />
                        Blacklisted Apps
                    </DialogTitle>
                </DialogHeader>

                <div className="space-y-4">
                    <p className="text-sm text-muted-foreground">
                        These applications will be ignored by the activity tracker.
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

                    <div className="flex justify-end">
                        <Button onClick={onClose}>Done</Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
};
