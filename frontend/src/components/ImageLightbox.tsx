import React, { useState, useEffect } from 'react';
import { X, ZoomIn, ZoomOut, Download } from 'lucide-react';
import { Button } from './ui/button';

interface ImageLightboxProps {
    src: string;
    alt: string;
    onClose: () => void;
}

export const ImageLightbox: React.FC<ImageLightboxProps> = ({ src, alt, onClose }) => {
    const [zoom, setZoom] = useState(100);

    useEffect(() => {
        const handleEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', handleEscape);
        return () => window.removeEventListener('keydown', handleEscape);
    }, [onClose]);

    const handleZoomIn = () => setZoom(prev => Math.min(prev + 25, 200));
    const handleZoomOut = () => setZoom(prev => Math.max(prev - 25, 50));

    const handleDownload = () => {
        const link = document.createElement('a');
        link.href = src;
        link.download = alt || 'screenshot.png';
        link.click();
    };

    const handleBackdropClick = (e: React.MouseEvent) => {
        if (e.target === e.currentTarget) {
            onClose();
        }
    };

    return (
        <div
            className="fixed inset-0 z-50 bg-black/90 flex items-center justify-center p-4"
            onClick={handleBackdropClick}
        >
            {/* Controls */}
            <div className="absolute top-4 right-4 flex gap-2">
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleZoomOut}
                    className="bg-background/20 hover:bg-background/30 text-white"
                >
                    <ZoomOut size={20} />
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleZoomIn}
                    className="bg-background/20 hover:bg-background/30 text-white"
                >
                    <ZoomIn size={20} />
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleDownload}
                    className="bg-background/20 hover:bg-background/30 text-white"
                >
                    <Download size={20} />
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={onClose}
                    className="bg-background/20 hover:bg-background/30 text-white"
                >
                    <X size={20} />
                </Button>
            </div>

            {/* Zoom indicator */}
            <div className="absolute top-4 left-4 bg-background/20 text-white px-3 py-1 rounded text-sm">
                {zoom}%
            </div>

            {/* Image */}
            <div className="max-w-[90vw] max-h-[90vh] overflow-auto">
                <img
                    src={src}
                    alt={alt}
                    style={{ transform: `scale(${zoom / 100})` }}
                    className="transition-transform duration-200 origin-center"
                />
            </div>

            {/* Instructions */}
            <div className="absolute bottom-4 left-1/2 -translate-x-1/2 text-white/60 text-sm">
                Click outside to close • Esc to exit
            </div>
        </div>
    );
};
