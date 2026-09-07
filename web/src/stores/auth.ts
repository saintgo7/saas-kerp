import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { User } from "@/types";
import { authService } from "@/services/auth";
import { STORAGE_KEYS } from "@/constants";
import { createSafeJSONStorage } from "@/lib/storage";
import { onSessionCleared } from "@/lib/session";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  register: (data: {
    email: string;
    password: string;
    name: string;
    phone?: string;
    companyName: string;
    businessNumber: string;
  }) => Promise<void>;
  fetchUser: () => Promise<void>;
  clearError: () => void;
  setUser: (user: User | null) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      login: async (email: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
          const { user } = await authService.login({ email, password });
          set({ user, isAuthenticated: true, isLoading: false });
        } catch (err) {
          const error = err as { message?: string };
          set({
            error: error.message || "로그인에 실패했습니다.",
            isLoading: false,
          });
          throw err;
        }
      },

      logout: async () => {
        set({ isLoading: true });
        try {
          await authService.logout();
        } finally {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            error: null,
          });
        }
      },

      register: async (data) => {
        set({ isLoading: true, error: null });
        try {
          const { user } = await authService.register(data);
          set({ user, isAuthenticated: true, isLoading: false });
        } catch (err) {
          const error = err as { message?: string };
          set({
            error: error.message || "회원가입에 실패했습니다.",
            isLoading: false,
          });
          throw err;
        }
      },

      fetchUser: async () => {
        set({ isLoading: true });
        try {
          const user = await authService.getCurrentUser();
          set({ user, isAuthenticated: true, isLoading: false });
        } catch {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },

      clearError: () => set({ error: null }),

      setUser: (user) =>
        set({
          user,
          isAuthenticated: !!user,
        }),
    }),
    {
      // Own bucket. `STORAGE_KEYS.user` is the authService user cache; sharing
      // one key meant each writer destroyed the other's format and a refresh
      // could drop `isAuthenticated` back to false with valid tokens present.
      name: STORAGE_KEYS.authStore,
      storage: createJSONStorage(createSafeJSONStorage),
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// A 401 teardown from any HTTP client must also drop the in-memory flag,
// otherwise PublicRoute bounces the user straight back to /dashboard.
onSessionCleared(() => {
  const { user, isAuthenticated } = useAuthStore.getState();
  if (user !== null || isAuthenticated) {
    useAuthStore.setState({ user: null, isAuthenticated: false });
  }
});
