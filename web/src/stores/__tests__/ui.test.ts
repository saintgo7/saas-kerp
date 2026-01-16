import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { useUIStore, toast } from '../ui';

// Mock document and window for theme tests
const mockClassList = {
  add: vi.fn(),
  remove: vi.fn(),
};

Object.defineProperty(document, 'documentElement', {
  value: { classList: mockClassList },
  writable: true,
});

const mockMatchMedia = vi.fn();
Object.defineProperty(window, 'matchMedia', {
  value: mockMatchMedia,
  writable: true,
});

// Reset Zustand store before each test
const initialStoreState = useUIStore.getState();

beforeEach(() => {
  useUIStore.setState(initialStoreState);
  vi.clearAllMocks();
  mockMatchMedia.mockReturnValue({ matches: false });
});

afterEach(() => {
  localStorage.clear();
});

describe('useUIStore', () => {
  // ==========================================================================
  // Initial State Tests
  // ==========================================================================

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const state = useUIStore.getState();

      expect(state.sidebarCollapsed).toBe(false);
      expect(state.sidebarMobileOpen).toBe(false);
      expect(state.theme).toBe('system');
      expect(state.globalLoading).toBe(false);
      expect(state.loadingMessage).toBe('');
      expect(state.toasts).toEqual([]);
      expect(state.modalOpen).toBe(false);
      expect(state.modalContent).toBeNull();
    });
  });

  // ==========================================================================
  // Sidebar Tests
  // ==========================================================================

  describe('sidebar actions', () => {
    it('should toggle sidebar collapsed state', () => {
      const { toggleSidebar } = useUIStore.getState();

      expect(useUIStore.getState().sidebarCollapsed).toBe(false);

      toggleSidebar();
      expect(useUIStore.getState().sidebarCollapsed).toBe(true);

      toggleSidebar();
      expect(useUIStore.getState().sidebarCollapsed).toBe(false);
    });

    it('should set sidebar collapsed state directly', () => {
      const { setSidebarCollapsed } = useUIStore.getState();

      setSidebarCollapsed(true);
      expect(useUIStore.getState().sidebarCollapsed).toBe(true);

      setSidebarCollapsed(false);
      expect(useUIStore.getState().sidebarCollapsed).toBe(false);
    });

    it('should set sidebar mobile open state', () => {
      const { setSidebarMobileOpen } = useUIStore.getState();

      setSidebarMobileOpen(true);
      expect(useUIStore.getState().sidebarMobileOpen).toBe(true);

      setSidebarMobileOpen(false);
      expect(useUIStore.getState().sidebarMobileOpen).toBe(false);
    });
  });

  // ==========================================================================
  // Theme Tests
  // ==========================================================================

  describe('theme actions', () => {
    it('should set light theme', () => {
      const { setTheme } = useUIStore.getState();

      setTheme('light');

      expect(useUIStore.getState().theme).toBe('light');
      expect(mockClassList.remove).toHaveBeenCalledWith('light', 'dark');
      expect(mockClassList.add).toHaveBeenCalledWith('light');
    });

    it('should set dark theme', () => {
      const { setTheme } = useUIStore.getState();

      setTheme('dark');

      expect(useUIStore.getState().theme).toBe('dark');
      expect(mockClassList.remove).toHaveBeenCalledWith('light', 'dark');
      expect(mockClassList.add).toHaveBeenCalledWith('dark');
    });

    it('should set system theme and apply based on preference (dark)', () => {
      mockMatchMedia.mockReturnValue({ matches: true }); // Dark preference

      const { setTheme } = useUIStore.getState();
      setTheme('system');

      expect(useUIStore.getState().theme).toBe('system');
      expect(mockClassList.remove).toHaveBeenCalledWith('light', 'dark');
      expect(mockClassList.add).toHaveBeenCalledWith('dark');
    });

    it('should set system theme and apply based on preference (light)', () => {
      mockMatchMedia.mockReturnValue({ matches: false }); // Light preference

      const { setTheme } = useUIStore.getState();
      setTheme('system');

      expect(useUIStore.getState().theme).toBe('system');
      expect(mockClassList.remove).toHaveBeenCalledWith('light', 'dark');
      expect(mockClassList.add).toHaveBeenCalledWith('light');
    });
  });

  // ==========================================================================
  // Loading State Tests
  // ==========================================================================

  describe('loading actions', () => {
    it('should set global loading state', () => {
      const { setGlobalLoading } = useUIStore.getState();

      setGlobalLoading(true);
      expect(useUIStore.getState().globalLoading).toBe(true);
      expect(useUIStore.getState().loadingMessage).toBe('');

      setGlobalLoading(false);
      expect(useUIStore.getState().globalLoading).toBe(false);
    });

    it('should set global loading with message', () => {
      const { setGlobalLoading } = useUIStore.getState();

      setGlobalLoading(true, 'Loading data...');

      expect(useUIStore.getState().globalLoading).toBe(true);
      expect(useUIStore.getState().loadingMessage).toBe('Loading data...');
    });

    it('should clear loading message when loading stops', () => {
      const { setGlobalLoading } = useUIStore.getState();

      setGlobalLoading(true, 'Loading...');
      setGlobalLoading(false);

      expect(useUIStore.getState().globalLoading).toBe(false);
      expect(useUIStore.getState().loadingMessage).toBe('');
    });
  });

  // ==========================================================================
  // Toast Tests
  // ==========================================================================

  describe('toast actions', () => {
    it('should add toast notification', () => {
      const { addToast } = useUIStore.getState();

      addToast({ type: 'success', title: 'Success!', message: 'Operation completed' });

      const toasts = useUIStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('success');
      expect(toasts[0].title).toBe('Success!');
      expect(toasts[0].message).toBe('Operation completed');
      expect(toasts[0].id).toMatch(/^toast_\d+_/);
    });

    it('should add multiple toasts', () => {
      const { addToast } = useUIStore.getState();

      addToast({ type: 'info', title: 'Info 1' });
      addToast({ type: 'error', title: 'Error 1' });
      addToast({ type: 'warning', title: 'Warning 1' });

      const toasts = useUIStore.getState().toasts;
      expect(toasts).toHaveLength(3);
      expect(toasts[0].type).toBe('info');
      expect(toasts[1].type).toBe('error');
      expect(toasts[2].type).toBe('warning');
    });

    it('should remove toast by id', () => {
      const { addToast, removeToast } = useUIStore.getState();

      addToast({ type: 'success', title: 'Toast 1' });
      addToast({ type: 'error', title: 'Toast 2' });

      const toasts = useUIStore.getState().toasts;
      expect(toasts).toHaveLength(2);

      const toastToRemove = toasts[0].id;
      removeToast(toastToRemove);

      const remainingToasts = useUIStore.getState().toasts;
      expect(remainingToasts).toHaveLength(1);
      expect(remainingToasts[0].title).toBe('Toast 2');
    });

    it('should clear all toasts', () => {
      const { addToast, clearToasts } = useUIStore.getState();

      addToast({ type: 'success', title: 'Toast 1' });
      addToast({ type: 'error', title: 'Toast 2' });
      addToast({ type: 'info', title: 'Toast 3' });

      expect(useUIStore.getState().toasts).toHaveLength(3);

      clearToasts();

      expect(useUIStore.getState().toasts).toHaveLength(0);
    });

    it('should generate unique toast ids', () => {
      const { addToast } = useUIStore.getState();

      addToast({ type: 'success', title: 'Toast 1' });
      addToast({ type: 'success', title: 'Toast 2' });
      addToast({ type: 'success', title: 'Toast 3' });

      const toasts = useUIStore.getState().toasts;
      const ids = toasts.map((t) => t.id);
      const uniqueIds = new Set(ids);

      expect(uniqueIds.size).toBe(ids.length);
    });
  });

  // ==========================================================================
  // Toast Helper Functions Tests
  // ==========================================================================

  describe('toast helper functions', () => {
    it('should add success toast', () => {
      toast.success('Success!', 'Operation completed');

      const toasts = useUIStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('success');
      expect(toasts[0].title).toBe('Success!');
      expect(toasts[0].message).toBe('Operation completed');
    });

    it('should add error toast', () => {
      toast.error('Error!', 'Something went wrong');

      const toasts = useUIStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('error');
      expect(toasts[0].title).toBe('Error!');
    });

    it('should add warning toast', () => {
      toast.warning('Warning!');

      const toasts = useUIStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('warning');
      expect(toasts[0].title).toBe('Warning!');
      expect(toasts[0].message).toBeUndefined();
    });

    it('should add info toast', () => {
      toast.info('Information', 'Here is some info');

      const toasts = useUIStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0].type).toBe('info');
      expect(toasts[0].title).toBe('Information');
    });
  });

  // ==========================================================================
  // Modal Tests
  // ==========================================================================

  describe('modal actions', () => {
    it('should open modal with content', () => {
      const { openModal } = useUIStore.getState();
      const mockContent = 'Modal Content';

      openModal(mockContent);

      const state = useUIStore.getState();
      expect(state.modalOpen).toBe(true);
      expect(state.modalContent).toBe(mockContent);
    });

    it('should close modal and clear content', () => {
      const { openModal, closeModal } = useUIStore.getState();

      openModal('Some content');
      expect(useUIStore.getState().modalOpen).toBe(true);

      closeModal();

      const state = useUIStore.getState();
      expect(state.modalOpen).toBe(false);
      expect(state.modalContent).toBeNull();
    });

    it('should replace modal content when opening new modal', () => {
      const { openModal } = useUIStore.getState();

      openModal('Content 1');
      expect(useUIStore.getState().modalContent).toBe('Content 1');

      openModal('Content 2');
      expect(useUIStore.getState().modalContent).toBe('Content 2');
    });
  });

  // ==========================================================================
  // Persistence Tests
  // ==========================================================================

  describe('persistence', () => {
    it('should persist sidebar collapsed state', () => {
      const { setSidebarCollapsed } = useUIStore.getState();

      setSidebarCollapsed(true);

      // The partialize function should only persist sidebarCollapsed and theme
      const state = useUIStore.getState();
      expect(state.sidebarCollapsed).toBe(true);
    });

    it('should persist theme setting', () => {
      mockMatchMedia.mockReturnValue({ matches: false });

      const { setTheme } = useUIStore.getState();

      setTheme('dark');

      const state = useUIStore.getState();
      expect(state.theme).toBe('dark');
    });

    it('should not persist transient state like toasts', () => {
      const { addToast } = useUIStore.getState();

      addToast({ type: 'success', title: 'Test' });

      // Toasts should not be persisted (not in partialize)
      const state = useUIStore.getState();
      expect(state.toasts).toHaveLength(1);
    });
  });

  // ==========================================================================
  // Integration Tests
  // ==========================================================================

  describe('integration', () => {
    it('should handle complete user flow', () => {
      const store = useUIStore.getState();

      // User opens app with collapsed sidebar
      store.setSidebarCollapsed(true);

      // Sets dark theme
      mockMatchMedia.mockReturnValue({ matches: false });
      store.setTheme('dark');

      // Starts loading
      store.setGlobalLoading(true, 'Fetching data...');

      // Receives success
      store.setGlobalLoading(false);
      toast.success('Data loaded', 'Your data is ready');

      // Opens modal
      store.openModal('Confirmation dialog');

      const state = useUIStore.getState();
      expect(state.sidebarCollapsed).toBe(true);
      expect(state.theme).toBe('dark');
      expect(state.globalLoading).toBe(false);
      expect(state.toasts).toHaveLength(1);
      expect(state.modalOpen).toBe(true);
    });

    it('should handle error flow with multiple toasts', () => {
      const store = useUIStore.getState();

      // Multiple errors occur
      toast.error('Error 1', 'First error');
      toast.error('Error 2', 'Second error');
      toast.warning('Warning', 'Something might be wrong');

      expect(useUIStore.getState().toasts).toHaveLength(3);

      // User dismisses first error
      const toasts = useUIStore.getState().toasts;
      store.removeToast(toasts[0].id);

      expect(useUIStore.getState().toasts).toHaveLength(2);

      // User clears all
      store.clearToasts();

      expect(useUIStore.getState().toasts).toHaveLength(0);
    });
  });
});
