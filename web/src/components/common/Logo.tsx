import { useEffect, useState } from 'react';
import { useUIStore } from '@/stores';
import { cn } from '@/lib/utils';

interface LogoProps {
  className?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  showText?: boolean;
}

const sizeClasses = {
  sm: 'h-6',
  md: 'h-8',
  lg: 'h-10',
  xl: 'h-14',
};

export function Logo({ className, size = 'md', showText: _showText = true }: LogoProps) {
  const { theme } = useUIStore();
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    // Determine if we should use dark logo
    if (theme === 'dark') {
      setIsDark(true);
    } else if (theme === 'light') {
      setIsDark(false);
    } else {
      // System theme - check media query
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      setIsDark(mediaQuery.matches);

      const handler = (e: MediaQueryListEvent) => setIsDark(e.matches);
      mediaQuery.addEventListener('change', handler);
      return () => mediaQuery.removeEventListener('change', handler);
    }
  }, [theme]);

  const logoSrc = isDark ? '/images/logo-dark.png' : '/images/logo-light.png';

  return (
    <img
      src={logoSrc}
      alt="K-ERP"
      className={cn(sizeClasses[size], 'w-auto object-contain', className)}
    />
  );
}

// Icon-only version for collapsed sidebar
export function LogoIcon({ className, size = 'md' }: Omit<LogoProps, 'showText'>) {
  const { theme } = useUIStore();
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    if (theme === 'dark') {
      setIsDark(true);
    } else if (theme === 'light') {
      setIsDark(false);
    } else {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      setIsDark(mediaQuery.matches);

      const handler = (e: MediaQueryListEvent) => setIsDark(e.matches);
      mediaQuery.addEventListener('change', handler);
      return () => mediaQuery.removeEventListener('change', handler);
    }
  }, [theme]);

  // For icon-only, we use a simple K badge since cropping the logo would be complex
  return (
    <div
      className={cn(
        'flex items-center justify-center rounded-lg',
        isDark ? 'bg-sky-500' : 'bg-blue-600',
        size === 'sm' && 'w-6 h-6',
        size === 'md' && 'w-8 h-8',
        size === 'lg' && 'w-10 h-10',
        size === 'xl' && 'w-14 h-14',
        className
      )}
    >
      <span
        className={cn(
          'font-bold text-white',
          size === 'sm' && 'text-sm',
          size === 'md' && 'text-base',
          size === 'lg' && 'text-lg',
          size === 'xl' && 'text-2xl'
        )}
      >
        K
      </span>
    </div>
  );
}
