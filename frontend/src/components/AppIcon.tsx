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
  Terminal,
  AppWindow
} from 'lucide-react';

interface AppIconProps {
  app: AppType | string;
  className?: string;
}

export const AppIcon: React.FC<AppIconProps> = ({ app, className }) => {
  const iconProps = { className };

  switch (app) {
    case 'Chrome':
      return <Chrome {...iconProps} className={`${className} text-yellow-500`} />;
    case 'Slack':
      return <Slack {...iconProps} className={`${className} text-purple-500`} />;
    case 'Notes':
      return <FileText {...iconProps} className={`${className} text-yellow-600`} />;
    case 'Spotify':
      return <Music {...iconProps} className={`${className} text-green-500`} />;
    case 'Figma':
      return <Figma {...iconProps} className={`${className} text-purple-400`} />;
    case 'VS Code':
      return <Code {...iconProps} className={`${className} text-blue-500`} />;
    case 'Zoom':
      return <Video {...iconProps} className={`${className} text-blue-400`} />;
    case 'Excel':
      return <Sheet {...iconProps} className={`${className} text-green-600`} />;
    case 'Terminal':
      return <Terminal {...iconProps} className={`${className} text-gray-400`} />;
    default:
      return <AppWindow {...iconProps} className={`${className} text-gray-500`} />;
  }
};
