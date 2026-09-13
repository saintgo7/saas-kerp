import { STORAGE_KEYS } from "@/constants";
import { safeStorage } from "./storage";

/**
 * Single owner of "the session is gone" so that every HTTP client clears the
 * exact same state. Previously `services/api.ts` cleared three keys while
 * `api/client.ts` cleared two, which left the persisted `isAuthenticated: true`
 * behind and produced a /login <-> /dashboard redirect loop.
 */

type Listener = () => void;

const listeners = new Set<Listener>();

/** Register a callback invoked whenever the session is cleared. */
export function onSessionCleared(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function clearSession(): void {
  safeStorage.removeItem(STORAGE_KEYS.accessToken);
  safeStorage.removeItem(STORAGE_KEYS.refreshToken);
  safeStorage.removeItem(STORAGE_KEYS.user);
  safeStorage.removeItem(STORAGE_KEYS.authStore);

  listeners.forEach((listener) => {
    try {
      listener();
    } catch {
      // A broken listener must not stop the others.
    }
  });
}

/**
 * Navigate to the login screen, but never when we are already there.
 * Redirecting from /login is what turned a bad password into a page reload.
 */
export function redirectToLogin(): void {
  if (typeof window === "undefined") return;
  if (window.location.pathname === "/login") return;
  window.location.href = "/login";
}

/** Clear the session and bounce to /login. */
export function endSession(): void {
  clearSession();
  redirectToLogin();
}
