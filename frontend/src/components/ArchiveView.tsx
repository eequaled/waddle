import { useState, useEffect } from 'react';
import { Folder, Plus, ArrowRight } from 'lucide-react';
import { api } from '../services/api';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Card } from './ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from './ui/dialog';

interface ArchiveGroup {
    name: string;
    items: string[];
}

export function ArchiveView() {
    const [groups, setGroups] = useState<ArchiveGroup[]>([]);
    const [newGroupName, setNewGroupName] = useState('');
    const [isDialogOpen, setIsDialogOpen] = useState(false);

    useEffect(() => {
        loadArchives();
    }, []);

    const loadArchives = async () => {
        try {
            const data = await api.getArchives();
            setGroups(data || []);
        } catch (error) {
            console.error('Failed to load archives:', error);
        }
    };

    const handleCreateGroup = async () => {
        if (!newGroupName.trim()) return;
        try {
            await api.createArchive(newGroupName);
            setNewGroupName('');
            setIsDialogOpen(false);
            loadArchives();
        } catch (error) {
            console.error('Failed to create archive:', error);
        }
    };

    return (
        <div className="h-full flex flex-col bg-background text-foreground p-8 overflow-y-auto">
            <div className="flex justify-between items-center mb-8">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">Archives</h2>
                    <p className="text-muted-foreground mt-1">Organize your sessions into custom collections.</p>
                </div>

                <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
                    <DialogTrigger asChild>
                        <Button className="gap-2">
                            <Plus className="w-4 h-4" />
                            New Collection
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>Create New Collection</DialogTitle>
                        </DialogHeader>
                        <div className="flex gap-2 mt-4">
                            <Input
                                placeholder="Collection Name (e.g., Project X)"
                                value={newGroupName}
                                onChange={e => setNewGroupName(e.target.value)}
                            />
                            <Button onClick={handleCreateGroup}>Create</Button>
                        </div>
                    </DialogContent>
                </Dialog>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6">
                {groups.map((group) => (
                    <Card key={group.name} className="p-6 hover:bg-accent/50 transition-all cursor-pointer group relative overflow-hidden">
                        <div className="absolute top-0 right-0 p-4 opacity-0 group-hover:opacity-100 transition-opacity">
                            <ArrowRight className="w-5 h-5 text-muted-foreground" />
                        </div>

                        <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                            <Folder className="w-6 h-6 text-primary" />
                        </div>

                        <h3 className="font-semibold text-lg mb-1">{group.name}</h3>
                        <p className="text-sm text-muted-foreground">
                            {group.items?.length || 0} items
                        </p>
                    </Card>
                ))}

                {groups.length === 0 && (
                    <div className="col-span-full flex flex-col items-center justify-center py-20 text-muted-foreground border-2 border-dashed rounded-xl">
                        <Folder className="w-12 h-12 mb-4 opacity-20" />
                        <p>No archives yet. Create a collection to get started.</p>
                    </div>
                )}
            </div>
        </div>
    );
}
