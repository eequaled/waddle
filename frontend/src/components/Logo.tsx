import React from 'react';

interface LogoProps {
  className?: string;
}

export const Logo: React.FC<LogoProps> = ({ className = "w-8 h-8" }) => {
  return (
      <svg 
        xmlns="http://www.w3.org/2000/svg" 
        viewBox="80 110 420 310" 
        className={className}
        fill="none"
        aria-label="Ideathon Logo"
      >
      <g 
        stroke="currentColor" 
        strokeWidth="24" 
        strokeLinecap="round" 
        strokeLinejoin="round"
      >
        <path 
          d="M 110 240 c 0 0 20 140 170 140 100 0 130-100 130-130 0-50-20-70-37-93-10-20 0-30 0-30 l 70 0 c 69-38 18 58-40 60 0 0-10-90-70-100-70-10-110 40-110 80 0 30 20 60 50 70 0 0-80 10-170 0 z" 
          strokeLinejoin="round" 
        />
        <path d="M190,290 C220,340 300,340 320,260" />
        <circle cx="310" cy="160" r="13" fill="currentColor" stroke="none" />
      </g>
    </svg>
  );
};

