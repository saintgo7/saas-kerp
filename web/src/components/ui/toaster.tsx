import { useEffect } from "react";
import { AlertCircle, AlertTriangle, CheckCircle2, Info, X } from "lucide-react";
import {
  DEFAULT_TOAST_DURATION,
  useUIStore,
  type Toast,
} from "@/stores/ui";
import { cn } from "@/lib/utils";

const toastStyles: Record<Toast["type"], string> = {
  success: "border-green-500/40 bg-green-50 text-green-900",
  error: "border-red-500/40 bg-red-50 text-red-900",
  warning: "border-amber-500/40 bg-amber-50 text-amber-900",
  info: "border-blue-500/40 bg-blue-50 text-blue-900",
};

const toastIcons: Record<Toast["type"], typeof Info> = {
  success: CheckCircle2,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

function ToastItem({ toast }: { toast: Toast }) {
  const removeToast = useUIStore((state) => state.removeToast);
  const Icon = toastIcons[toast.type];

  useEffect(() => {
    const duration = toast.duration ?? DEFAULT_TOAST_DURATION;
    if (duration <= 0) return;
    const timer = window.setTimeout(() => removeToast(toast.id), duration);
    return () => window.clearTimeout(timer);
  }, [toast.id, toast.duration, removeToast]);

  return (
    <div
      role={toast.type === "error" ? "alert" : "status"}
      aria-live={toast.type === "error" ? "assertive" : "polite"}
      className={cn(
        "pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-lg border p-4 shadow-lg",
        toastStyles[toast.type]
      )}
    >
      <Icon className="mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold break-words">{toast.title}</p>
        {toast.message && (
          <p className="mt-1 text-sm opacity-90 break-words">{toast.message}</p>
        )}
      </div>
      <button
        type="button"
        onClick={() => removeToast(toast.id)}
        aria-label="알림 닫기"
        className="shrink-0 rounded p-1 opacity-70 transition-opacity hover:opacity-100"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

/**
 * Renders the toast queue held in the UI store.
 *
 * Without this mounted, every `toast.*()` call in the app was silent: messages
 * piled up in the store and never reached the DOM.
 */
export function Toaster() {
  const toasts = useUIStore((state) => state.toasts);

  if (toasts.length === 0) return null;

  return (
    <div
      className="pointer-events-none fixed top-4 right-4 z-[100] flex flex-col items-end gap-2"
      aria-label="알림"
    >
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} />
      ))}
    </div>
  );
}

export default Toaster;
