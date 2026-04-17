import React, { useState } from 'react';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { User, Settings, Clock, MessageSquare, Archive, Sparkles, LogOut, Brain } from "lucide-react";
import { ProfileImageCarousel } from './ProfileImageCarousel';

interface ProfileMenuProps {
    activeView: 'timeline' | 'chat' | 'archives' | 'insights' | 'knowledge';
    setActiveView: (view: 'timeline' | 'chat' | 'archives' | 'insights' | 'knowledge') => void;
    setIsSettingsOpen: (isOpen: boolean) => void;
}

export function ProfileMenu({ activeView, setActiveView, setIsSettingsOpen }: ProfileMenuProps) {
    const [profileImage, setProfileImage] = useState<string | null>(() => {
        return localStorage.getItem('profileImage');
    });

    const handleImageSelect = (url: string | null) => {
        setProfileImage(url);
        if (url) {
            localStorage.setItem('profileImage', url);
        } else {
            localStorage.removeItem('profileImage');
        }
    };

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <div className="h-8 w-8 rounded-full bg-accent ml-2 flex items-center justify-center text-accent-foreground border border-border cursor-pointer overflow-hidden hover:opacity-90 transition-opacity">
                    {profileImage ? (
                        <img src={profileImage} alt="Profile" className="h-full w-full object-cover" />
                    ) : (
                        <User size={16} />
                    )}
                </div>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-64 mr-2" align="end">
                <DropdownMenuLabel className="text-center">My Profile</DropdownMenuLabel>

                {/* Profile Picture Carousel */}
                <div className="py-2">
                    <ProfileImageCarousel
                        currentImage={profileImage}
                        onSelectImage={handleImageSelect}
                    />
                </div>

                <DropdownMenuSeparator />

                {/* Navigation */}
                <div className="p-1 space-y-1">
                    <DropdownMenuItem
                        className={`cursor-pointer ${activeView === 'timeline' ? 'bg-accent' : ''}`}
                        onClick={() => setActiveView('timeline')}
                    >
                        <Clock className="mr-2 h-4 w-4" />
                        <span>Timeline</span>
                    </DropdownMenuItem>

                    <DropdownMenuItem
                        className={`cursor-pointer ${activeView === 'chat' ? 'bg-accent' : ''}`}
                        onClick={() => setActiveView('chat')}
                    >
                        <MessageSquare className="mr-2 h-4 w-4" />
                        <span>Chat</span>
                    </DropdownMenuItem>

                    <DropdownMenuItem
                        className={`cursor-pointer ${activeView === 'archives' ? 'bg-accent' : ''}`}
                        onClick={() => setActiveView('archives')}
                    >
                        <Archive className="mr-2 h-4 w-4" />
                        <span>Archives</span>
                    </DropdownMenuItem>

                    <DropdownMenuItem
                        className={`cursor-pointer ${activeView === 'insights' ? 'bg-accent' : ''}`}
                        onClick={() => setActiveView('insights')}
                    >
                        <Sparkles className="mr-2 h-4 w-4" />
                        <span>Insights</span>
                    </DropdownMenuItem>

                    <DropdownMenuItem
                        className={`cursor-pointer ${activeView === 'knowledge' ? 'bg-accent' : ''}`}
                        onClick={() => setActiveView('knowledge')}
                    >
                        <Brain className="mr-2 h-4 w-4" />
                        <span>Knowledge Cards</span>
                    </DropdownMenuItem>
                </div>

                <DropdownMenuSeparator />

                {/* Settings & Account */}
                <div className="p-1 space-y-1">
                    <DropdownMenuItem
                        className="cursor-pointer"
                        onClick={() => setIsSettingsOpen(true)}
                    >
                        <Settings className="mr-2 h-4 w-4" />
                        <span>Settings</span>
                    </DropdownMenuItem>
                </div>

                <DropdownMenuSeparator />

                {/* Login/Logout (Visual Only) */}
                <div className="p-1">
                    <DropdownMenuItem className="cursor-pointer text-red-500 focus:text-red-500">
                        <LogOut className="mr-2 h-4 w-4" />
                        <span>Log out</span>
                    </DropdownMenuItem>
                </div>
            </DropdownMenuContent>
        </DropdownMenu>
    );
}
