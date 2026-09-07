/**
 * Safe localStorage access.
 *
 * The global `localStorage` can be missing or throw in several environments:
 * - Safari private mode / "block site data" -> SecurityError on access
 * - Node >= 22 exposes an experimental global `localStorage` that is `undefined`
 *   unless `--localstorage-file` is given, and it shadows the jsdom one in tests
 * - SSR / non-browser runtimes have no `localStorage` at all
 *
 * Every read/write here degrades to an in-memory map instead of throwing.
 */

const memoryStore = new Map<string, string>();

function resolveStorage(): Storage | null {
  try {
    const candidate =
      typeof globalThis !== "undefined" && "localStorage" in globalThis
        ? (globalThis as { localStorage?: Storage | null }).localStorage
        : null;

    if (!candidate || typeof candidate.getItem !== "function") {
      return null;
    }

    // Probe: Safari private mode throws on setItem even though the object exists.
    const probeKey = "__kerp_storage_probe__";
    candidate.setItem(probeKey, "1");
    candidate.removeItem(probeKey);

    return candidate;
  } catch {
    return null;
  }
}

export const safeStorage = {
  getItem(key: string): string | null {
    const storage = resolveStorage();
    if (!storage) {
      return memoryStore.has(key) ? (memoryStore.get(key) as string) : null;
    }
    try {
      return storage.getItem(key);
    } catch {
      return memoryStore.has(key) ? (memoryStore.get(key) as string) : null;
    }
  },

  setItem(key: string, value: string): void {
    memoryStore.set(key, value);
    const storage = resolveStorage();
    if (!storage) return;
    try {
      storage.setItem(key, value);
    } catch {
      // Quota exceeded or blocked; the in-memory copy is the fallback.
    }
  },

  removeItem(key: string): void {
    memoryStore.delete(key);
    const storage = resolveStorage();
    if (!storage) return;
    try {
      storage.removeItem(key);
    } catch {
      // ignore
    }
  },
};

/**
 * Storage adapter shaped for `zustand/persist`'s `createJSONStorage`.
 */
export const createSafeJSONStorage = () => ({
  getItem: (name: string) => safeStorage.getItem(name),
  setItem: (name: string, value: string) => safeStorage.setItem(name, value),
  removeItem: (name: string) => safeStorage.removeItem(name),
});
