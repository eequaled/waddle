import React, { useState } from 'react';
import { Session } from '../types';
import { Button } from './ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './ui/dropdown-menu';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from './ui/alert-dialog';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Input } from './ui/input';
import { ScrollArea } from './ui/scroll-area';
import { MoreHorizontal, Archive, FileDown, Trash2, Plus, FolderOpen } from 'lucide-react';
import { api } from '../services/api';
import { toast } from 'sonner';

interface SessionActionsMenuProps {
  session: Session;
  onSessionDelete?: (sessionId: string) => void;
  onSessionArchived?: (sessionId: string) => void;
}

// Utility function to export session to Markdown
export function exportSessionToMarkdown(session: Session): string {
  const displayTitle = session.customTitle || session.title;
  const displaySummary = session.customSummary || session.summary;
  
  let markdown = `# ${displayTitle}\n\n`;
  markdown += `**Date:** ${session.date}\n`;
  markdown += `**Time:** ${session.startTime} - ${session.endTime}\n`;
  markdown += `**Duration:** ${session.duration}\n\n`;
  
  if (session.tags.length > 0) {
    markdown += `**Tags:** ${session.tags.map(t => `#${t}`).join(' ')}\n\n`;
  }
  
  markdown += `## Summary\n\n${displaySummary}\n\n`;

  // Add manual notes if present
  if (session.manualNotes && session.manualNotes.length > 0) {
    markdown += `## Personal Notes\n\n`;
    session.manualNotes.forEach(note => {
      markdown += `- ${note.content}\n`;
    });
    markdown += '\n';
  }
  
  // Add memory blocks
  markdown += `## Activity Log\n\n`;
  session.content.forEach(block => {
    if (block.type === 'app-memory' && block.data) {
      markdown += `### ${block.data.appName || block.content}\n\n`;
      if (block.data.blocks && Array.isArray(block.data.blocks)) {
        block.data.blocks.forEach((memBlock: any) => {
          if (memBlock.microSummary) {
            markdown += `- **${memBlock.startTime || 'Unknown time'}**: ${memBlock.microSummary}\n`;
          }
        });
      }
      markdown += '\n';
    } else if (block.type === 'summary') {
      markdown += `> ${block.content}\n\n`;
    }
  });
  
  return markdown;
}

// Generate filename for export
export function generateExportFilename(session: Session): string {
  return `session-${session.date}.md`;
}

// Trigger file download
function downloadMarkdown(content: string, filename: string) {
  const blob = new Blob([content], { type: 'text/markdown' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export const SessionActionsMenu: React.FC<SessionActionsMenuProps> = ({
  session,
  onSessionDelete,
  onSessionArchived,
}) => {
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [isArchiveDialogOpen, setIsArchiveDialogOpen] = useState(false);
  const [archives, setArchives] = useState<Array<{ name: string; items: any[] }>>([]);
  const [newArchiveName, setNewArchiveName] = useState('');
  const [isCreatingArchive, setIsCreatingArchive] = useState(false);

  const handleExportMarkdown = () => {
    try {
      const markdown = exportSessionToMarkdown(session);
      const filename = generateExportFilename(session);
      downloadMarkdown(markdown, filename);
      toast.success('Session exported to Markdown');
    } catch (error) {
      toast.error('Failed to export session');
    }
  };

  const handleOpenArchiveDialog = async () => {
    try {
      const archiveList = await api.getArchives();
      setArchives(archiveList || []);
    } catch (error) {
      console.error('Failed to fetch archives:', error);
      setArchives([]);
    }
    setIsArchiveDialogOpen(true);
  };


  const handleMoveToArchive = async (archiveName: string) => {
    try {
      await api.moveToArchive(session.id, archiveName);
      setIsArchiveDialogOpen(false);
      onSessionArchived?.(session.id);
      toast.success(`Session moved to "${archiveName}"`);
    } catch (error) {
      console.error('Failed to move to archive:', error);
      toast.error('Failed to move session to archive');
    }
  };

  const handleCreateArchive = async () => {
    if (!newArchiveName.trim()) return;
    
    try {
      setIsCreatingArchive(true);
      await api.createArchive(newArchiveName.trim());
      await handleMoveToArchive(newArchiveName.trim());
      setNewArchiveName('');
    } catch (error) {
      console.error('Failed to create archive:', error);
    } finally {
      setIsCreatingArchive(false);
    }
  };

  const handleDelete = () => {
    onSessionDelete?.(session.id);
    setIsDeleteDialogOpen(false);
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="h-8 w-8">
            <MoreHorizontal size={16} />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuItem onClick={handleOpenArchiveDialog}>
            <Archive size={16} className="mr-2" />
            Move to Archive
          </DropdownMenuItem>
          <DropdownMenuItem onClick={handleExportMarkdown}>
            <FileDown size={16} className="mr-2" />
            Export to Markdown
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem 
            variant="destructive"
            onClick={() => setIsDeleteDialogOpen(true)}
          >
            <Trash2 size={16} className="mr-2" />
            Delete Session
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Session?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the session "{session.customTitle || session.title}" 
              and all its associated data. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction 
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>


      {/* Archive Selection Dialog */}
      <Dialog open={isArchiveDialogOpen} onOpenChange={setIsArchiveDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Move to Archive</DialogTitle>
            <DialogDescription>
              Select an existing archive or create a new one.
            </DialogDescription>
          </DialogHeader>
          
          <div className="py-4">
            {archives.length > 0 && (
              <ScrollArea className="h-[200px] mb-4">
                <div className="space-y-2">
                  {archives.map((archive) => (
                    <Button
                      key={archive.name}
                      variant="outline"
                      className="w-full justify-start gap-2"
                      onClick={() => handleMoveToArchive(archive.name)}
                    >
                      <FolderOpen size={16} />
                      {archive.name}
                      <span className="ml-auto text-muted-foreground text-xs">
                        {archive.items?.length || 0} items
                      </span>
                    </Button>
                  ))}
                </div>
              </ScrollArea>
            )}
            
            <div className="flex gap-2">
              <Input
                placeholder="New archive name..."
                value={newArchiveName}
                onChange={(e) => setNewArchiveName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleCreateArchive()}
              />
              <Button 
                onClick={handleCreateArchive}
                disabled={!newArchiveName.trim() || isCreatingArchive}
              >
                <Plus size={16} className="mr-1" />
                Create
              </Button>
            </div>
          </div>
          
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsArchiveDialogOpen(false)}>
              Cancel
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
