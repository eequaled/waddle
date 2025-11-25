import { useState, useRef, useEffect } from 'react';
import { Block } from '../types';
import { Type, Code, Image as ImageIcon, Link as LinkIcon, CheckSquare, Heading1 } from 'lucide-react';

interface BlockEditorProps {
  blocks: Block[];
  onBlocksChange: (blocks: Block[]) => void;
}

export function BlockEditor({ blocks, onBlocksChange }: BlockEditorProps) {
  const [showCommandMenu, setShowCommandMenu] = useState(false);
  const [commandMenuPosition, setCommandMenuPosition] = useState({ top: 0, left: 0 });
  const [activeBlockId, setActiveBlockId] = useState<string | null>(null);

  const handleKeyDown = (e: React.KeyboardEvent, blockId: string, index: number) => {
    if (e.key === '/' && e.currentTarget.textContent === '') {
      e.preventDefault();
      const rect = e.currentTarget.getBoundingClientRect();
      setCommandMenuPosition({ top: rect.bottom, left: rect.left });
      setShowCommandMenu(true);
      setActiveBlockId(blockId);
    }

    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      const newBlock: Block = {
        id: Date.now().toString(),
        type: 'paragraph',
        content: ''
      };
      const newBlocks = [...blocks];
      newBlocks.splice(index + 1, 0, newBlock);
      onBlocksChange(newBlocks);
    }

    if (e.key === 'Backspace' && e.currentTarget.textContent === '') {
      e.preventDefault();
      if (blocks.length > 1) {
        const newBlocks = blocks.filter(b => b.id !== blockId);
        onBlocksChange(newBlocks);
      }
    }
  };

  const insertBlock = (type: Block['type']) => {
    if (!activeBlockId) return;
    
    const blockIndex = blocks.findIndex(b => b.id === activeBlockId);
    const newBlocks = [...blocks];
    newBlocks[blockIndex] = {
      ...newBlocks[blockIndex],
      type,
      content: type === 'heading' ? 'Heading' : type === 'code' ? '// code here' : ''
    };
    onBlocksChange(newBlocks);
    setShowCommandMenu(false);
    setActiveBlockId(null);
  };

  const updateBlockContent = (blockId: string, content: string) => {
    const newBlocks = blocks.map(b => 
      b.id === blockId ? { ...b, content } : b
    );
    onBlocksChange(newBlocks);
  };

  const toggleTodo = (blockId: string) => {
    const newBlocks = blocks.map(b => 
      b.id === blockId ? { ...b, checked: !b.checked } : b
    );
    onBlocksChange(newBlocks);
  };

  useEffect(() => {
    const handleClickOutside = () => {
      setShowCommandMenu(false);
    };
    
    if (showCommandMenu) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [showCommandMenu]);

  return (
    <div className="relative">
      {blocks.map((block, index) => (
        <BlockComponent
          key={block.id}
          block={block}
          onKeyDown={(e) => handleKeyDown(e, block.id, index)}
          onContentChange={(content) => updateBlockContent(block.id, content)}
          onToggleTodo={() => toggleTodo(block.id)}
        />
      ))}

      {showCommandMenu && (
        <CommandMenu
          position={commandMenuPosition}
          onSelect={insertBlock}
        />
      )}
    </div>
  );
}

function BlockComponent({ 
  block, 
  onKeyDown, 
  onContentChange,
  onToggleTodo 
}: { 
  block: Block; 
  onKeyDown: (e: React.KeyboardEvent) => void;
  onContentChange: (content: string) => void;
  onToggleTodo: () => void;
}) {
  const handleInput = (e: React.FormEvent<HTMLDivElement>) => {
    onContentChange(e.currentTarget.textContent || '');
  };

  if (block.type === 'heading') {
    return (
      <div
        contentEditable
        suppressContentEditableWarning
        onKeyDown={onKeyDown}
        onInput={handleInput}
        className="text-3xl text-slate-100 mb-4 outline-none"
        placeholder="Heading"
      >
        {block.content}
      </div>
    );
  }

  if (block.type === 'code') {
    return (
      <div className="bg-slate-950 rounded-lg p-4 mb-3 font-mono text-sm">
        <div className="flex items-center gap-2 mb-2 text-slate-400 text-xs">
          <Code size={14} />
          <span>code</span>
        </div>
        <pre
          contentEditable
          suppressContentEditableWarning
          onKeyDown={onKeyDown}
          onInput={handleInput}
          className="text-slate-300 outline-none whitespace-pre-wrap"
        >
          {block.content}
        </pre>
      </div>
    );
  }

  if (block.type === 'todo') {
    return (
      <div className="flex items-start gap-3 mb-2">
        <input
          type="checkbox"
          checked={block.checked || false}
          onChange={onToggleTodo}
          className="mt-1"
        />
        <div
          contentEditable
          suppressContentEditableWarning
          onKeyDown={onKeyDown}
          onInput={handleInput}
          className={`flex-1 text-slate-300 outline-none ${block.checked ? 'line-through opacity-60' : ''}`}
        >
          {block.content}
        </div>
      </div>
    );
  }

  if (block.type === 'link') {
    return (
      <div className="flex items-center gap-2 mb-3 p-3 bg-slate-800 rounded-lg hover:bg-slate-750 transition-colors">
        <LinkIcon size={16} className="text-blue-400" />
        <div
          contentEditable
          suppressContentEditableWarning
          onKeyDown={onKeyDown}
          onInput={handleInput}
          className="text-blue-400 outline-none"
        >
          {block.content}
        </div>
      </div>
    );
  }

  // Default paragraph
  return (
    <div
      contentEditable
      suppressContentEditableWarning
      onKeyDown={onKeyDown}
      onInput={handleInput}
      className="text-slate-300 mb-3 outline-none"
      placeholder="Type '/' for commands"
    >
      {block.content}
    </div>
  );
}

function CommandMenu({ 
  position, 
  onSelect 
}: { 
  position: { top: number; left: number };
  onSelect: (type: Block['type']) => void;
}) {
  const commands = [
    { type: 'heading' as const, icon: Heading1, label: 'Heading' },
    { type: 'paragraph' as const, icon: Type, label: 'Text' },
    { type: 'code' as const, icon: Code, label: 'Code' },
    { type: 'todo' as const, icon: CheckSquare, label: 'To-do List' },
    { type: 'link' as const, icon: LinkIcon, label: 'Link' },
  ];

  return (
    <div 
      className="fixed bg-slate-800 border border-slate-700 rounded-lg shadow-xl p-2 z-50 min-w-[200px]"
      style={{ top: position.top + 4, left: position.left }}
      onClick={(e) => e.stopPropagation()}
    >
      {commands.map((cmd) => (
        <button
          key={cmd.type}
          onClick={() => onSelect(cmd.type)}
          className="w-full flex items-center gap-3 px-3 py-2 hover:bg-slate-700 rounded text-left transition-colors"
        >
          <cmd.icon size={16} className="text-slate-400" />
          <span className="text-slate-200">{cmd.label}</span>
        </button>
      ))}
    </div>
  );
}
