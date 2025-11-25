import React from 'react';
import { Session } from '../types';
import { AppIcon } from './AppIcon';
import { Badge } from './ui/badge';
import { Card } from './ui/card';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from './ui/collapsible';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { cn } from './ui/utils';

interface SessionCardProps {
  session: Session;
  isSelected: boolean;
  onClick: () => void;
}

export const SessionCard: React.FC<SessionCardProps> = ({ session, isSelected, onClick }) => {
  const [isOpen, setIsOpen] = React.useState(false);

  // Stop propagation when clicking the expand button so we don't select the card if user just wants to peek
  const toggleOpen = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsOpen(!isOpen);
  };

  return (
    <Card 
      className={cn(
        "p-4 mb-3 cursor-pointer transition-all hover:bg-accent/50 border-none shadow-sm bg-card/50",
        isSelected ? "ring-1 ring-primary bg-accent" : ""
      )}
      onClick={onClick}
    >
      <Collapsible open={isOpen} onOpenChange={setIsOpen}>
        <div className="flex justify-between items-start">
          <div className="flex gap-2 mb-2">
            {session.apps.slice(0, 4).map((app, i) => (
              <AppIcon key={i} app={app} className="w-5 h-5" />
            ))}
          </div>
          <CollapsibleTrigger asChild onClick={toggleOpen}>
            <button className="text-muted-foreground hover:text-foreground p-1 rounded-sm hover:bg-background/20 transition-colors">
              {isOpen ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
            </button>
          </CollapsibleTrigger>
        </div>

        <h3 className="font-semibold text-sm mb-2 line-clamp-2">{session.title}</h3>
        
        <div className="flex flex-wrap gap-2 mb-3">
          {session.tags.map(tag => (
            <Badge key={tag} variant="secondary" className="text-[10px] px-1.5 h-5 font-normal text-muted-foreground">
              {tag}
            </Badge>
          ))}
        </div>

        <div className="text-xs text-muted-foreground mb-1">
          {session.startTime} - {session.endTime}
        </div>

        <CollapsibleContent className="mt-4 space-y-3 border-t border-border/50 pt-3">
           {session.activities.map(activity => (
             <div key={activity.id} className="flex items-start gap-2 text-xs text-muted-foreground">
               <AppIcon app={activity.app} className="w-3 h-3 mt-0.5 shrink-0 opacity-70" />
               <span className="line-clamp-1">{activity.description}</span>
             </div>
           ))}
        </CollapsibleContent>
      </Collapsible>
    </Card>
  );
};
