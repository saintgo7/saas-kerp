import { useEffect, useState, useMemo } from 'react';
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

// Helper to get system dark mode preference
function getSystemDarkMode() {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

// Custom hook to determine if dark mode is active
function useDarkMode() {
  const { theme } = useUIStore();
  // Initialize with actual system preference to avoid initial setState in effect
  const [systemIsDark, setSystemIsDark] = useState(getSystemDarkMode);

  useEffect(() => {
    if (theme !== 'system') return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => setSystemIsDark(e.matches);
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [theme]);

  return useMemo(() => {
    if (theme === 'dark') return true;
    if (theme === 'light') return false;
    return systemIsDark;
  }, [theme, systemIsDark]);
}

export function Logo({ className, size = 'md', showText = true }: LogoProps) {
  const isDark = useDarkMode();
  // showText is available for future use when text logo variant is added
  void showText;

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
  const isDark = useDarkMode();

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
