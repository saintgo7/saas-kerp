import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Merge Tailwind CSS classes with proper precedence
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/**
 * Format number as Korean currency (KRW)
 */
export function formatCurrency(
  amount: number,
  options?: { showSymbol?: boolean; compact?: boolean }
): string {
  const { showSymbol = true, compact = false } = options ?? {};

  if (compact && Math.abs(amount) >= 100000000) {
    return `${showSymbol ? "" : ""}${(amount / 100000000).toFixed(1)}억`;
  }
  if (compact && Math.abs(amount) >= 10000) {
    return `${showSymbol ? "" : ""}${(amount / 10000).toFixed(0)}만`;
  }

  const formatted = new Intl.NumberFormat("ko-KR").format(amount);
  return showSymbol ? `${formatted}원` : formatted;
}

/**
 * Format date in Korean locale
 */
export function formatDate(
  date: Date | string,
  options?: { format?: "full" | "short" | "month" | "time" }
): string {
  const d = typeof date === "string" ? new Date(date) : date;
  const { format = "short" } = options ?? {};

  switch (format) {
    case "full":
      return d.toLocaleDateString("ko-KR", {
        year: "numeric",
        month: "long",
        day: "numeric",
        weekday: "long",
      });
    case "month":
      return d.toLocaleDateString("ko-KR", {
        year: "numeric",
        month: "long",
      });
    case "time":
      return d.toLocaleString("ko-KR", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      });
    case "short":
    default:
      return d.toLocaleDateString("ko-KR", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
      });
  }
}

/**
 * Format business registration number (사업자등록번호)
 */
export function formatBusinessNumber(number: string): string {
  const cleaned = number.replace(/\D/g, "");
  if (cleaned.length !== 10) return number;
  return `${cleaned.slice(0, 3)}-${cleaned.slice(3, 5)}-${cleaned.slice(5)}`;
}

/**
 * Format phone number
 */
export function formatPhoneNumber(number: string): string {
  const cleaned = number.replace(/\D/g, "");
  if (cleaned.length === 11) {
    return `${cleaned.slice(0, 3)}-${cleaned.slice(3, 7)}-${cleaned.slice(7)}`;
  }
  if (cleaned.length === 10) {
    return `${cleaned.slice(0, 3)}-${cleaned.slice(3, 6)}-${cleaned.slice(6)}`;
  }
  return number;
}

/**
 * Debounce function
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
  fn: T,
  delay: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout>;
  return (...args: Parameters<T>) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn(...args), delay);
  };
}

/**
 * Generate a random ID
 */
export function generateId(prefix = ""): string {
  const id = Math.random().toString(36).substring(2, 11);
  return prefix ? `${prefix}_${id}` : id;
}

/**
 * Deep clone an object
 */
export function deepClone<T>(obj: T): T {
  return JSON.parse(JSON.stringify(obj));
}

/**
 * Check if value is empty (null, undefined, empty string, empty array, empty object)
 */
export function isEmpty(value: unknown): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === "string") return value.trim() === "";
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return false;
}

/**
 * Sleep for a specified duration
 */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
