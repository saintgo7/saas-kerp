import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { STORAGE_KEYS } from "@/constants";
import { createSafeJSONStorage } from "@/lib/storage";

type Theme = "light" | "dark" | "system";

export interface Toast {
  id: string;
  type: "success" | "error" | "warning" | "info";
  title: string;
  message?: string;
  duration?: number;
}

interface UIState {
  // Sidebar
  sidebarCollapsed: boolean;
  sidebarMobileOpen: boolean;

  // Theme
  theme: Theme;

  // Loading
  globalLoading: boolean;
  loadingMessage: string;

  // Toasts
  toasts: Toast[];

  // Modal
  modalOpen: boolean;
  modalContent: React.ReactNode | null;

  // Actions
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setSidebarMobileOpen: (open: boolean) => void;
  setTheme: (theme: Theme) => void;
  setGlobalLoading: (loading: boolean, message?: string) => void;
  addToast: (toast: Omit<Toast, "id">) => void;
  removeToast: (id: string) => void;
  clearToasts: () => void;
  openModal: (content: React.ReactNode) => void;
  closeModal: () => void;
}

// Toasts are transient; never let the queue grow without bound.
const MAX_TOASTS = 5;

// Default lifetime (ms) used by the Toaster when a toast omits `duration`.
export const DEFAULT_TOAST_DURATION = 4000;

// Helper function to apply theme to document
function applyThemeToDocument(theme: Theme) {
  const root = document.documentElement;
  root.classList.remove("light", "dark");
  if (theme === "system") {
    const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
    root.classList.add(systemTheme);
  } else {
    root.classList.add(theme);
  }
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      // Initial state
      sidebarCollapsed: false,
      sidebarMobileOpen: false,
      theme: "system",
      globalLoading: false,
      loadingMessage: "",
      toasts: [],
      modalOpen: false,
      modalContent: null,

      // Actions
      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),

      setSidebarMobileOpen: (open) => set({ sidebarMobileOpen: open }),

      setTheme: (theme) => {
        set({ theme });
        applyThemeToDocument(theme);
      },

      setGlobalLoading: (loading, message = "") =>
        set({ globalLoading: loading, loadingMessage: message }),

      addToast: (toast) =>
        set((state) => {
          const next = [
            ...state.toasts,
            { ...toast, id: `toast_${Date.now()}_${Math.random()}` },
          ];
          // Hard cap so the array cannot grow without bound if no Toaster is
          // mounted or a burst of errors arrives faster than they dismiss.
          return {
            toasts: next.length > MAX_TOASTS ? next.slice(-MAX_TOASTS) : next,
          };
        }),

      removeToast: (id) =>
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        })),

      clearToasts: () => set({ toasts: [] }),

      openModal: (content) => set({ modalOpen: true, modalContent: content }),

      closeModal: () => set({ modalOpen: false, modalContent: null }),
    }),
    {
      name: STORAGE_KEYS.sidebarCollapsed,
      storage: createJSONStorage(createSafeJSONStorage),
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        theme: state.theme,
      }),
    }
  )
);

// Toast helper functions
export const toast = {
  success: (title: string, message?: string) =>
    useUIStore.getState().addToast({ type: "success", title, message }),
  error: (title: string, message?: string) =>
    useUIStore.getState().addToast({ type: "error", title, message }),
  warning: (title: string, message?: string) =>
    useUIStore.getState().addToast({ type: "warning", title, message }),
  info: (title: string, message?: string) =>
    useUIStore.getState().addToast({ type: "info", title, message }),
};

// Listen for system theme changes
if (typeof window !== "undefined") {
  const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
  mediaQuery.addEventListener("change", () => {
    const currentTheme = useUIStore.getState().theme;
    if (currentTheme === "system") {
      applyThemeToDocument("system");
    }
  });

  // Apply stored theme on initial load (after hydration)
  useUIStore.persist.onFinishHydration(() => {
    const theme = useUIStore.getState().theme;
    applyThemeToDocument(theme);
  });
}
