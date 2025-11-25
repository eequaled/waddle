import React from 'react';
import { EditorBlock } from '../types';
import { Checkbox } from './ui/checkbox';
import { Bot, GripVertical, Image as ImageIcon, Link } from 'lucide-react';

interface BlockProps {
  block: EditorBlock;
}

export const BlockRenderer: React.FC<BlockProps> = ({ block }) => {
  switch (block.type) {
    case 'summary':
      return (
        <div className="bg-accent/30 p-6 rounded-lg border border-border/50 mb-6 group relative">
           <div className="flex items-center gap-2 mb-2 text-primary font-medium text-sm">
             <Bot size={16} />
             Auto-Summary
           </div>
           <p className="text-muted-foreground leading-relaxed text-sm">
             {block.content}
           </p>
        </div>
      );
    
    case 'heading':
      return (
        <div className="flex items-center gap-2 group mt-6 mb-3">
          <div className="opacity-0 group-hover:opacity-30 cursor-grab">
            <GripVertical size={14} />
          </div>
          <h2 className="text-xl font-semibold text-foreground w-full outline-none" contentEditable suppressContentEditableWarning>
            {block.content}
          </h2>
        </div>
      );

    case 'paragraph':
      return (
        <div className="flex items-start gap-2 group mb-2">
           <div className="opacity-0 group-hover:opacity-30 cursor-grab mt-1">
            <GripVertical size={14} />
          </div>
          <p className="text-foreground/90 leading-7 w-full outline-none" contentEditable suppressContentEditableWarning>
            {block.content}
          </p>
        </div>
      );

    case 'todo':
      return (
        <div className="flex items-center gap-3 group mb-2">
           <div className="opacity-0 group-hover:opacity-30 cursor-grab">
            <GripVertical size={14} />
          </div>
          <Checkbox checked={block.checked} className="rounded-sm" />
          <span className={`w-full outline-none ${block.checked ? 'text-muted-foreground line-through' : 'text-foreground'}`} contentEditable suppressContentEditableWarning>
            {block.content}
          </span>
        </div>
      );

    case 'code':
      return (
        <div className="flex items-start gap-2 group mb-4 mt-2 relative">
          <div className="opacity-0 group-hover:opacity-30 cursor-grab mt-4">
             <GripVertical size={14} />
          </div>
          <div className="bg-[#1e1e1e] rounded-md border border-border w-full overflow-hidden relative">
             <div className="flex items-center justify-between px-4 py-2 bg-[#252526] border-b border-[#333] text-xs text-muted-foreground">
                <span>{block.language || 'text'}</span>
                <button className="hover:text-white">Copy</button>
             </div>
             <pre className="p-4 font-mono text-sm text-[#d4d4d4] overflow-x-auto">
               <code>{block.content}</code>
             </pre>
             
             {/* Overlay button from reference image */}
             <div className="absolute bottom-4 right-4 opacity-0 group-hover:opacity-100 transition-opacity">
                <button className="bg-primary text-primary-foreground hover:bg-primary/90 text-xs px-3 py-1.5 rounded-md shadow-lg font-medium flex items-center gap-2">
                   Restore Session Windows
                </button>
             </div>
          </div>
        </div>
      );

     case 'image':
        return (
            <div className="flex items-start gap-2 group mb-4 mt-2">
                <div className="opacity-0 group-hover:opacity-30 cursor-grab mt-4">
                    <GripVertical size={14} />
                </div>
                 <div className="rounded-lg overflow-hidden border border-border bg-accent/20 w-full max-w-md relative aspect-video flex items-center justify-center">
                    <ImageIcon className="text-muted-foreground/50 w-12 h-12" />
                    <span className="absolute bottom-2 right-2 text-xs text-muted-foreground bg-black/50 px-2 py-1 rounded">Image Placeholder</span>
                 </div>
            </div>
        );

    default:
      return null;
  }
};
