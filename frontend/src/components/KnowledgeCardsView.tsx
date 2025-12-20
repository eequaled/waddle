import React, { useState, useEffect } from 'react';
import { KnowledgeCard, Entity } from '../types';
import { api } from '../services/api';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Badge } from './ui/badge';
import { Clock, Hash, AtSign, ExternalLink, AlertCircle, Loader2 } from 'lucide-react';
import { cn } from './ui/utils';

interface KnowledgeCardsViewProps {
  onCardClick?: (sessionId: string) => void;
}

export const KnowledgeCardsView: React.FC<KnowledgeCardsViewProps> = ({ onCardClick }) => {
  const [cards, setCards] = useState<KnowledgeCard[]>([]);
  const [pendingCount, setPendingCount] = useState<number>(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadData = async () => {
      try {
        setLoading(true);
        const [cardsData, pendingData] = await Promise.all([
          api.getKnowledgeCards(),
          api.getPendingCount()
        ]);
        
        setCards(cardsData || []);
        setPendingCount(pendingData?.count || 0);
        setError(null);
      } catch (err) {
        console.error('Failed to load knowledge cards:', err);
        setError('Failed to load knowledge cards');
      } finally {
        setLoading(false);
      }
    };

    loadData();
    
    // Poll for updates every 30 seconds
    const interval = setInterval(loadData, 30000);
    return () => clearInterval(interval);
  }, []);

  const getEntityIcon = (type: Entity['type']) => {
    switch (type) {
      case 'hashtag':
        return <Hash size={12} />;
      case 'mention':
        return <AtSign size={12} />;
      case 'url':
        return <ExternalLink size={12} />;
      case 'jira':
        return <Badge variant="outline" className="text-[10px] px-1 h-4">JIRA</Badge>;
      default:
        return null;
    }
  };

  const formatTimestamp = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch {
      return timestamp;
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        <AlertCircle className="h-8 w-8 mr-2" />
        {error}
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header with pending count */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold">Knowledge Cards</h2>
        {pendingCount > 0 && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            {pendingCount} sessions pending synthesis
          </div>
        )}
      </div>

      {/* Responsive grid layout */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {cards.map((card) => (
          <KnowledgeCardComponent
            key={card.sessionId}
            card={card}
            onClick={() => onCardClick?.(card.sessionId)}
            getEntityIcon={getEntityIcon}
            formatTimestamp={formatTimestamp}
          />
        ))}
        
        {/* Show pending placeholders */}
        {Array.from({ length: Math.min(pendingCount, 6) }).map((_, index) => (
          <PendingCardPlaceholder key={`pending-${index}`} />
        ))}
      </div>

      {cards.length === 0 && pendingCount === 0 && (
        <div className="text-center text-muted-foreground py-12">
          No knowledge cards available yet. Start using the system to generate insights!
        </div>
      )}
    </div>
  );
};

interface KnowledgeCardComponentProps {
  card: KnowledgeCard;
  onClick: () => void;
  getEntityIcon: (type: Entity['type']) => React.ReactNode;
  formatTimestamp: (timestamp: string) => string;
}

const KnowledgeCardComponent: React.FC<KnowledgeCardComponentProps> = ({
  card,
  onClick,
  getEntityIcon,
  formatTimestamp
}) => {
  const isProcessed = card.status === 'processed';
  const isFailed = card.status === 'failed';

  return (
    <Card 
      className={cn(
        "cursor-pointer transition-all hover:shadow-md hover:scale-[1.02]",
        "border-border/50 bg-card/50",
        isFailed && "border-red-200 bg-red-50/50 dark:border-red-800 dark:bg-red-950/20"
      )}
      onClick={onClick}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <CardTitle className="text-sm font-medium line-clamp-2 flex-1">
            {card.title}
          </CardTitle>
          <div className="flex items-center gap-1 text-xs text-muted-foreground ml-2">
            <Clock size={12} />
            {formatTimestamp(card.timestamp)}
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="pt-0 space-y-3">
        {/* Status indicator */}
        {isFailed && (
          <div className="flex items-center gap-2 text-xs text-red-600 dark:text-red-400">
            <AlertCircle size={12} />
            Synthesis failed
          </div>
        )}

        {/* Bullets */}
        {isProcessed && card.bullets.length > 0 && (
          <ul className="space-y-1 text-xs text-muted-foreground">
            {card.bullets.slice(0, 3).map((bullet, index) => (
              <li key={index} className="flex items-start gap-2">
                <span className="text-primary mt-1">•</span>
                <span className="line-clamp-2">{bullet}</span>
              </li>
            ))}
          </ul>
        )}

        {/* Entities */}
        {card.entities.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {card.entities.slice(0, 6).map((entity, index) => (
              <Badge
                key={index}
                variant="secondary"
                className="text-[10px] px-1.5 h-5 font-normal flex items-center gap-1"
              >
                {getEntityIcon(entity.type)}
                <span className="truncate max-w-[80px]">{entity.value}</span>
                {entity.count > 1 && (
                  <span className="text-muted-foreground">×{entity.count}</span>
                )}
              </Badge>
            ))}
            {card.entities.length > 6 && (
              <Badge variant="secondary" className="text-[10px] px-1.5 h-5 font-normal">
                +{card.entities.length - 6}
              </Badge>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

const PendingCardPlaceholder: React.FC = () => {
  return (
    <Card className="border-border/50 bg-card/30">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          Pending synthesis...
        </CardTitle>
      </CardHeader>
      
      <CardContent className="pt-0 space-y-3">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 size={12} className="animate-spin" />
          Processing session data
        </div>
        
        {/* Placeholder content */}
        <div className="space-y-2">
          <div className="h-2 bg-muted/50 rounded animate-pulse" />
          <div className="h-2 bg-muted/50 rounded animate-pulse w-3/4" />
          <div className="h-2 bg-muted/50 rounded animate-pulse w-1/2" />
        </div>
      </CardContent>
    </Card>
  );
};