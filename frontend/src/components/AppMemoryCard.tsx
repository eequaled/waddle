import React, { useState } from 'react';
import { BlockData } from '../types';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from './ui/accordion';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { AppIcon } from './AppIcon';
import { ImageLightbox } from './ImageLightbox';
import { Clock, FileText, Sparkles } from 'lucide-react';

interface AppMemoryCardProps {
    appName: string;
    latestScreenshot?: string;
    blocks: BlockData[];
    timestamp?: string;
    onAskAI?: (block: BlockData) => void;
}

export const AppMemoryCard: React.FC<AppMemoryCardProps> = ({
    appName,
    latestScreenshot,
    blocks,
    timestamp,
    onAskAI,
}) => {
    const [lightboxOpen, setLightboxOpen] = useState(false);

    // Get the latest block for the "Current Activity" summary
    const latestBlock = blocks.length > 0 ? blocks[blocks.length - 1] : null;

    return (
        <Card className="mb-8 overflow-hidden border-border/60 shadow-sm hover:shadow-md transition-shadow">
            <CardHeader className="bg-muted/30 pb-4 border-b border-border/40">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-background rounded-md shadow-sm">
                            <AppIcon app={appName} className="w-6 h-6" />
                        </div>
                        <div>
                            <CardTitle className="text-lg font-semibold">{appName}</CardTitle>
                            {timestamp && (
                                <div className="text-xs text-muted-foreground flex items-center gap-1 mt-0.5">
                                    <Clock size={12} />
                                    {timestamp}
                                </div>
                            )}
                        </div>
                    </div>
                    <Badge variant="outline" className="bg-background/50">
                        {blocks.length} Memory Blocks
                    </Badge>
                </div>
            </CardHeader>

            <CardContent className="p-0">
                <div className="grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-border/40">

                    {/* Left Column: Visual Context */}
                    <div className="p-6 flex flex-col gap-4">
                        <h4 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">Latest Context</h4>

                        {latestScreenshot ? (
                            <>
                                <div
                                    className="relative rounded-lg overflow-hidden border border-border bg-muted/20 cursor-pointer group aspect-video"
                                    onClick={() => setLightboxOpen(true)}
                                >
                                    <img
                                        src={latestScreenshot}
                                        alt={`Latest ${appName} screenshot`}
                                        className="w-full h-full object-cover object-top transition-transform group-hover:scale-105"
                                    />
                                    <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
                                        <span className="bg-background/80 backdrop-blur text-xs px-2 py-1 rounded shadow-sm">
                                            View Fullscreen
                                        </span>
                                    </div>
                                </div>
                                {lightboxOpen && (
                                    <ImageLightbox
                                        src={latestScreenshot}
                                        alt={`${appName} Screenshot`}
                                        onClose={() => setLightboxOpen(false)}
                                    />
                                )}
                            </>
                        ) : (
                            <div className="h-40 rounded-lg border border-dashed border-border flex items-center justify-center text-muted-foreground bg-muted/10">
                                No screenshot available
                            </div>
                        )}

                        {latestBlock && (
                            <div className="bg-primary/5 border border-primary/10 rounded-lg p-4">
                                <div className="flex items-center gap-2 text-primary text-xs font-semibold uppercase tracking-wider mb-2">
                                    <FileText size={14} />
                                    AI Summary
                                </div>
                                <p className="text-sm text-foreground/90 leading-relaxed">
                                    {latestBlock.microSummary || "No summary available."}
                                </p>
                            </div>
                        )}
                    </div>

                    {/* Right Column: Memory Timeline */}
                    <div className="p-6 bg-muted/5">
                        <h4 className="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-4">Activity Timeline</h4>

                        {blocks.length === 0 ? (
                            <div className="text-sm text-muted-foreground italic">No recorded activity blocks.</div>
                        ) : (
                            <Accordion type="single" collapsible className="w-full space-y-2">
                                {blocks.map((block) => (
                                    <AccordionItem key={block.id} value={block.id} className="border border-border/60 rounded-lg bg-background px-3 group/block">
                                        <AccordionTrigger className="hover:no-underline py-3">
                                            <div className="flex items-center gap-3 text-left flex-1">
                                                <div className="text-xs font-mono text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                                                    {block.startTime.split('T')[1]?.substring(0, 5) || block.startTime}
                                                </div>
                                                <span className="text-sm font-medium truncate max-w-[180px]">
                                                    {block.microSummary ? block.microSummary.substring(0, 40) + "..." : "Activity Block"}
                                                </span>
                                            </div>
                                        </AccordionTrigger>
                                        <AccordionContent className="pb-3 pt-1">
                                            <div className="space-y-3">
                                                <p className="text-sm text-muted-foreground leading-relaxed">
                                                    {block.microSummary}
                                                </p>
                                                {block.ocrText && (
                                                    <div className="text-xs text-muted-foreground/70 bg-muted/30 p-2 rounded border border-border/30 font-mono max-h-32 overflow-y-auto">
                                                        {block.ocrText}
                                                    </div>
                                                )}
                                                {onAskAI && (
                                                    <Button
                                                        variant="outline"
                                                        size="sm"
                                                        className="gap-1.5 opacity-0 group-hover/block:opacity-100 transition-opacity"
                                                        onClick={(e) => {
                                                            e.stopPropagation();
                                                            onAskAI(block);
                                                        }}
                                                    >
                                                        <Sparkles size={14} />
                                                        Ask AI about this
                                                    </Button>
                                                )}
                                            </div>
                                        </AccordionContent>
                                    </AccordionItem>
                                ))}
                            </Accordion>
                        )}
                    </div>
                </div>
            </CardContent>
        </Card>
    );
};
