import React, { useState } from 'react';
import { ContentBlock } from '../types';
import { CheckSquare } from 'lucide-react';
import { ImageLightbox } from './ImageLightbox';

interface BlockRendererProps {
  block: ContentBlock;
}

export const BlockRenderer: React.FC<BlockRendererProps> = ({ block }) => {
  const [lightboxOpen, setLightboxOpen] = useState(false);

  if (block.type === 'summary') {
    return (
      <div className="mb-6 p-4 bg-muted/30 rounded-lg border-l-4 border-primary">
        <div className="text-xs uppercase tracking-wider text-muted-foreground font-semibold mb-2">
          Auto-Summary
        </div>
        <p className="text-foreground leading-relaxed">{block.content}</p>
      </div>
    );
  }

  if (block.type === 'heading') {
    return (
      <h2 className="text-2xl font-bold mt-8 mb-4 text-foreground">
        {block.content}
      </h2>
    );
  }

  if (block.type === 'paragraph') {
    return (
      <p className="mb-4 text-muted-foreground leading-relaxed">
        {block.content}
      </p>
    );
  }

  if (block.type === 'todo') {
    return (
      <div className="flex items-start gap-3 mb-3">
        <CheckSquare
          size={18}
          className={block.checked ? 'text-primary' : 'text-muted-foreground'}
        />
        <span className={`${block.checked ? 'line-through text-muted-foreground' : 'text-foreground'}`}>
          {block.content}
        </span>
      </div>
    );
  }

  if (block.type === 'image') {
    return (
      <>
        <div
          className="mb-6 rounded-lg overflow-hidden border border-border bg-muted/20 cursor-pointer hover:border-primary/50 transition-colors group"
          onClick={() => setLightboxOpen(true)}
        >
          <img
            src={block.content}
            alt="Session Screenshot"
            className="w-full h-auto max-h-[400px] object-contain group-hover:scale-[1.02] transition-transform"
          />
          <div className="p-2 text-xs text-muted-foreground text-center bg-muted/50">
            Click to view full size
          </div>
        </div>
        {lightboxOpen && (
          <ImageLightbox
            src={block.content}
            alt="Screenshot"
            onClose={() => setLightboxOpen(false)}
          />
        )}
      </>
    );
  }

  if (block.type === 'code') {
    return (
      <pre className="mb-4 p-4 bg-slate-950 rounded-lg overflow-x-auto">
        <code className="text-sm text-slate-300 font-mono">
          {block.content}
        </code>
      </pre>
    );
  }

  return null;
};
