import React from 'react';
import { Card, CardHeader, CardContent, CardTitle } from './ui/card';
import { Badge } from './ui/badge';

interface ActivityBlockProps {
  appName: string;
  startTime: string;
  endTime: string;
  microSummary: string;
  captureSource: string;
  metadata?: Record<string, unknown>;
}

export function ActivityBlock({ appName, startTime, endTime, microSummary, captureSource }: ActivityBlockProps) {
  // Simple emoji placeholder based on appName
  const getAppIcon = (name: string) => {
    const lower = name.toLowerCase();
    if (lower.includes('code') || lower.includes('cursor')) return '💻';
    if (lower.includes('browser') || lower.includes('chrome') || lower.includes('edge')) return '🌐';
    if (lower.includes('discord') || lower.includes('slack')) return '💬';
    if (lower.includes('terminal') || lower.includes('powershell')) return '⌨️';
    return '📦';
  };

  return (
    <Card className="mb-2">
      <CardHeader className="p-3 pb-0 flex flex-row items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xl">{getAppIcon(appName)}</span>
          <CardTitle className="text-sm font-medium">{appName}</CardTitle>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">{startTime} - {endTime}</span>
          <Badge variant={captureSource === 'ETW' ? 'default' : 'secondary'} className="text-[10px]">
            {captureSource}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="p-3 pt-2">
        <p className="text-xs text-muted-foreground">{microSummary || 'No summary available'}</p>
      </CardContent>
    </Card>
  );
}
