import React, { useState, useEffect, useRef } from 'react';
import {
    Carousel,
    CarouselContent,
    CarouselItem,
    CarouselNext,
    CarouselPrevious,
} from "./ui/carousel";
import { User, Plus, X } from "lucide-react";
import { api } from '../services/api';
import { toast } from 'sonner';

interface ProfileImageCarouselProps {
    onSelectImage: (imageUrl: string | null) => void;
    currentImage: string | null;
}

export function ProfileImageCarousel({ onSelectImage, currentImage }: ProfileImageCarouselProps) {
    const [images, setImages] = useState<string[]>([]);
    const [hoveredImage, setHoveredImage] = useState<string | null>(null);
    const fileInputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        loadImages();
    }, []);

    const loadImages = async () => {
        try {
            const imageList = await api.getProfileImages();
            // Sort: default_1, default_2 first, then uploads sorted by name
            const sorted = imageList.sort((a, b) => {
                const aIsDefault = a.startsWith('default_');
                const bIsDefault = b.startsWith('default_');
                if (aIsDefault && !bIsDefault) return -1;
                if (!aIsDefault && bIsDefault) return 1;
                return a.localeCompare(b);
            });
            setImages(sorted);
        } catch (error) {
            console.error('Failed to load profile images:', error);
        }
    };

    const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        try {
            const result = await api.uploadProfileImage(file);
            await loadImages();
            onSelectImage(result.url);
            toast.success('Profile picture uploaded');
            if (fileInputRef.current) {
                fileInputRef.current.value = '';
            }
        } catch (error) {
            console.error('Failed to upload image:', error);
            toast.error('Failed to upload image');
        }
    };

    const handleDelete = async (filename: string, e: React.MouseEvent) => {
        e.stopPropagation();
        
        if (filename.startsWith('default_')) {
            toast.error('Cannot delete default images');
            return;
        }

        try {
            await api.deleteProfileImage(filename);
            // If deleted image was selected, clear selection
            const deletedUrl = getImageUrl(filename);
            if (currentImage?.includes(filename)) {
                onSelectImage(null);
            }
            await loadImages();
            toast.success('Image deleted');
        } catch (error) {
            console.error('Failed to delete image:', error);
            toast.error('Failed to delete image');
        }
    };

    const getImageUrl = (filename: string) => {
        const timestamp = new Date().getTime();
        return `http://localhost:8080/images/profile/${filename}?t=${timestamp}`;
    };

    const isSelected = (url: string) => {
        if (!currentImage) return false;
        // Compare without timestamp
        const currentBase = currentImage.split('?')[0];
        const urlBase = url.split('?')[0];
        return currentBase === urlBase;
    };

    return (
        <div className="w-full px-8 py-2">
            <Carousel
                key={images.length}
                opts={{
                    align: "start",
                    loop: false,
                }}
                className="w-full max-w-xs mx-auto"
            >
                <CarouselContent className="-ml-2">
                    {/* Default "No Picture" Option */}
                    <CarouselItem className="pl-2 basis-1/3">
                        <div
                            className={`aspect-square rounded-full flex items-center justify-center border-2 cursor-pointer transition-all ${currentImage === null ? 'border-primary bg-primary/10' : 'border-border hover:border-primary/50'}`}
                            onClick={() => onSelectImage(null)}
                        >
                            <User className="h-6 w-6 text-muted-foreground" />
                        </div>
                    </CarouselItem>

                    {/* Profile Images (defaults first, then uploads) */}
                    {images.map((img) => {
                        const url = getImageUrl(img);
                        const selected = isSelected(url);
                        const isDefault = img.startsWith('default_');
                        const isHovered = hoveredImage === img;
                        
                        return (
                            <CarouselItem key={img} className="pl-2 basis-1/3">
                                <div
                                    className={`relative aspect-square rounded-full overflow-hidden border-2 cursor-pointer transition-all ${selected ? 'border-primary' : 'border-transparent hover:border-primary/50'}`}
                                    onClick={() => onSelectImage(url)}
                                    onMouseEnter={() => setHoveredImage(img)}
                                    onMouseLeave={() => setHoveredImage(null)}
                                >
                                    <img src={url} alt="Profile" className="h-full w-full object-cover" />
                                    {/* Delete button for non-default images */}
                                    {!isDefault && isHovered && (
                                        <button
                                            className="absolute top-0 right-0 bg-destructive text-destructive-foreground rounded-full p-0.5 hover:bg-destructive/90 transition-colors"
                                            onClick={(e) => handleDelete(img, e)}
                                            title="Delete image"
                                        >
                                            <X className="h-3 w-3" />
                                        </button>
                                    )}
                                </div>
                            </CarouselItem>
                        );
                    })}

                    {/* Upload Button */}
                    <CarouselItem className="pl-2 basis-1/3">
                        <div
                            className="aspect-square rounded-full flex items-center justify-center border-2 border-dashed border-muted-foreground/50 cursor-pointer hover:border-primary hover:bg-accent/50 transition-all"
                            onClick={() => fileInputRef.current?.click()}
                        >
                            <Plus className="h-6 w-6 text-muted-foreground" />
                            <input
                                type="file"
                                ref={fileInputRef}
                                className="hidden"
                                accept="image/*"
                                onChange={handleFileUpload}
                            />
                        </div>
                    </CarouselItem>
                </CarouselContent>
                <CarouselPrevious className="-left-4 h-6 w-6" />
                <CarouselNext className="-right-4 h-6 w-6" />
            </Carousel>
        </div>
    );
}
