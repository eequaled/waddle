import React from 'react';
import { AppType } from '../types';
import {
  Chrome,
  Slack,
  FileText,
  Music,
  Figma,
  Code,
  Video,
  Sheet,
  AppWindow
} from 'lucide-react';

interface AppIconProps {
  app: AppType | string;
  className?: string;
}

export const AppIcon: React.FC<AppIconProps> = ({ app, className }) => {
  const iconProps = { className };

  // Normalize app name: remove .exe, lowercase for matching
  const name = app.toLowerCase().replace('.exe', '');

  if (name.includes('chrome') || name.includes('edge') || name.includes('browser')) {
    return <Chrome {...iconProps} className={`${className} text-yellow-500`} />;
  }
  if (name.includes('slack')) {
    return <Slack {...iconProps} className={`${className} text-purple-500`} />;
  }
  if (name.includes('note') || name.includes('notepad') || name.includes('word')) {
    return <FileText {...iconProps} className={`${className} text-blue-600`} />;
  }
  if (name.includes('spotify') || name.includes('music')) {
    return <Music {...iconProps} className={`${className} text-green-500`} />;
  }
  if (name.includes('figma')) {
    return <Figma {...iconProps} className={`${className} text-purple-400`} />;
  }
  if (name.includes('code') || name.includes('vim') || name.includes('terminal') || name.includes('powershell')) {
    return <Code {...iconProps} className={`${className} text-blue-500`} />;
  }
  if (name.includes('zoom') || name.includes('meet')) {
    return <Video {...iconProps} className={`${className} text-blue-400`} />;
  }
  if (name.includes('excel') || name.includes('sheet') || name.includes('calc')) {
    return <Sheet {...iconProps} className={`${className} text-green-600`} />;
  }

  return <AppWindow {...iconProps} className={`${className} text-gray-500`} />;
};
