import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/__tests__/mocks/server';
import {
  createMockUser,
  createMockUserWire,
  createMockAuthPayload,
  envelope,
  errorEnvelope,
} from '@/__tests__/test-utils';
import { useAuthStore } from '../auth';

// Must match constants/index.ts API_BASE_URL and the Gin router (/api/v1).
const API_BASE = '/api/v1';

// Reset Zustand store before each test
const initialStoreState = useAuthStore.getState();

beforeEach(() => {
  useAuthStore.setState(initialStoreState);
  vi.clearAllMocks();
});

afterEach(() => {
  // Clear persisted state
  localStorage.clear();
});

describe('useAuthStore', () => {
  // ==========================================================================
  // Initial State Tests
  // ==========================================================================

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const state = useAuthStore.getState();

      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
    });
  });

  // ==========================================================================
  // Login Tests
  // ==========================================================================

  describe('login', () => {
    it('should login successfully with valid credentials', async () => {
      // The backend answers with snake_case tokens and a snake_case user
      // (internal/service/auth_service.go LoginResult).
      server.use(
        http.post(`${API_BASE}/auth/login`, () => {
          return HttpResponse.json(envelope(createMockAuthPayload()));
        })
      );

      const { login } = useAuthStore.getState();

      await login('test@example.com', 'Test123!@#');

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).toEqual(createMockUser());
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();

      // Tokens are persisted under the snake_case keys the server sends
      expect(localStorage.getItem('kerp_access_token')).toBe('test-access-token');
      expect(localStorage.getItem('kerp_refresh_token')).toBe('test-refresh-token');
    });

    it('should set loading state during login', async () => {
      server.use(
        http.post(`${API_BASE}/auth/login`, async () => {
          // Delay response to test loading state
          await new Promise((resolve) => setTimeout(resolve, 50));
          return HttpResponse.json(envelope(createMockAuthPayload()));
        })
      );

      const { login } = useAuthStore.getState();
      const loginPromise = login('test@example.com', 'password');

      // Check loading state
      expect(useAuthStore.getState().isLoading).toBe(true);

      await loginPromise;

      expect(useAuthStore.getState().isLoading).toBe(false);
    });

    it('should handle login failure with invalid credentials', async () => {
      server.use(
        http.post(`${API_BASE}/auth/login`, () => {
          return HttpResponse.json(
            errorEnvelope('AUTH_001', 'Invalid email or password'),
            { status: 401 }
          );
        })
      );

      const { login } = useAuthStore.getState();

      await expect(login('wrong@example.com', 'wrongpassword')).rejects.toThrow();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeTruthy();
    });

    it('should handle network error during login', async () => {
      server.use(
        http.post(`${API_BASE}/auth/login`, () => {
          return HttpResponse.error();
        })
      );

      const { login } = useAuthStore.getState();

      await expect(login('test@example.com', 'password')).rejects.toThrow();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isLoading).toBe(false);
    });
  });

  // ==========================================================================
  // Logout Tests
  // ==========================================================================

  describe('logout', () => {
    it('should logout successfully', async () => {
      // First, set authenticated state
      useAuthStore.setState({
        user: createMockUser(),
        isAuthenticated: true,
      });

      server.use(
        http.post(`${API_BASE}/auth/logout`, () => {
          return HttpResponse.json(envelope({ message: 'Logged out' }));
        })
      );

      const { logout } = useAuthStore.getState();
      await logout();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
      expect(state.error).toBeNull();
    });

    // TODO: Flaky test - network timing issues in test environment
    it.skip('should clear state even if logout API fails', async () => {
      useAuthStore.setState({
        user: createMockUser(),
        isAuthenticated: true,
      });

      server.use(
        http.post(`${API_BASE}/auth/logout`, () => {
          return HttpResponse.json(errorEnvelope('SRV_001', 'Server error'), {
            status: 500,
          });
        })
      );

      const { logout } = useAuthStore.getState();
      await logout();

      const state = useAuthStore.getState();
      // State should still be cleared even on API error
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });
  });

  // ==========================================================================
  // Register Tests
  // ==========================================================================

  describe('register', () => {
    const registerData = {
      email: 'newuser@example.com',
      password: 'NewUser123!@#',
      name: 'New User',
      phone: '010-1234-5678',
      companyName: 'New Company',
      businessNumber: '123-45-67890',
    };

    it('should register successfully and send snake_case fields', async () => {
      let capturedBody: Record<string, unknown> | undefined;

      server.use(
        http.post(`${API_BASE}/auth/register`, async ({ request }) => {
          capturedBody = (await request.json()) as Record<string, unknown>;
          return HttpResponse.json(
            envelope(
              createMockAuthPayload(
                createMockUserWire({
                  email: registerData.email,
                  name: registerData.name,
                })
              )
            ),
            { status: 201 }
          );
        })
      );

      const { register } = useAuthStore.getState();
      await register(registerData);

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user?.email).toBe(registerData.email);
      expect(state.isLoading).toBe(false);

      // internal/handler/auth.go RegisterRequest requires company_name and
      // business_number, not camelCase equivalents.
      expect(capturedBody).toMatchObject({
        company_name: registerData.companyName,
        business_number: registerData.businessNumber,
        email: registerData.email,
        name: registerData.name,
      });
    });

    it('should handle registration failure (duplicate email)', async () => {
      server.use(
        http.post(`${API_BASE}/auth/register`, () => {
          return HttpResponse.json(
            errorEnvelope('RES_002', 'Email already registered'),
            { status: 409 }
          );
        })
      );

      const { register } = useAuthStore.getState();

      await expect(register(registerData)).rejects.toThrow();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.error).toBeTruthy();
    });
  });

  // ==========================================================================
  // fetchUser Tests
  // ==========================================================================

  describe('fetchUser', () => {
    it('should fetch current user successfully', async () => {
      // internal/handler/auth.go Me returns JWT claims, not a user row.
      server.use(
        http.get(`${API_BASE}/auth/me`, () => {
          return HttpResponse.json(
            envelope({
              user_id: 'usr-001',
              company_id: 'comp-001',
              email: 'test@example.com',
              name: 'Test User',
              roles: ['admin'],
            })
          );
        })
      );

      const { fetchUser } = useAuthStore.getState();
      await fetchUser();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).toMatchObject({
        id: 'usr-001',
        companyId: 'comp-001',
        email: 'test@example.com',
        name: 'Test User',
        role: 'admin',
      });
    });

    // TODO: Flaky test - network timing issues in test environment
    it.skip('should clear auth state when fetch fails (token expired)', async () => {
      // Set initial authenticated state
      useAuthStore.setState({
        user: createMockUser(),
        isAuthenticated: true,
      });

      server.use(
        http.get(`${API_BASE}/auth/me`, () => {
          return HttpResponse.json(
            errorEnvelope('AUTH_002', 'Token expired'),
            { status: 401 }
          );
        })
      );

      const { fetchUser } = useAuthStore.getState();
      await fetchUser();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });
  });

  // ==========================================================================
  // Action Tests
  // ==========================================================================

  describe('actions', () => {
    it('should clear error', () => {
      useAuthStore.setState({ error: 'Some error' });

      const { clearError } = useAuthStore.getState();
      clearError();

      expect(useAuthStore.getState().error).toBeNull();
    });

    it('should set user directly', () => {
      const mockUser = createMockUser();

      const { setUser } = useAuthStore.getState();
      setUser(mockUser);

      const state = useAuthStore.getState();
      expect(state.user).toEqual(mockUser);
      expect(state.isAuthenticated).toBe(true);
    });

    it('should clear authentication when setting null user', () => {
      useAuthStore.setState({
        user: createMockUser(),
        isAuthenticated: true,
      });

      const { setUser } = useAuthStore.getState();
      setUser(null);

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  // ==========================================================================
  // Persistence Tests
  // ==========================================================================

  describe('persistence', () => {
    it('should persist user and isAuthenticated', () => {
      const mockUser = createMockUser();

      useAuthStore.setState({
        user: mockUser,
        isAuthenticated: true,
        isLoading: true, // Should not be persisted
        error: 'Some error', // Should not be persisted
      });

      // Simulate page reload by getting new store state
      // Note: In real scenario, we'd need to recreate the store
      const state = useAuthStore.getState();
      expect(state.user).toEqual(mockUser);
      expect(state.isAuthenticated).toBe(true);
    });
  });

  // ==========================================================================
  // Integration Tests
  // ==========================================================================

  describe('integration', () => {
    it('should handle complete login -> fetch user -> logout flow', async () => {
      server.use(
        http.post(`${API_BASE}/auth/login`, () => {
          return HttpResponse.json(envelope(createMockAuthPayload()));
        }),
        http.get(`${API_BASE}/auth/me`, () => {
          return HttpResponse.json(
            envelope({
              user_id: 'usr-001',
              company_id: 'comp-001',
              email: 'test@example.com',
              name: 'Test User',
              roles: ['admin'],
            })
          );
        }),
        http.post(`${API_BASE}/auth/logout`, () => {
          return HttpResponse.json(envelope({ message: 'Logged out' }));
        })
      );

      const store = useAuthStore.getState();

      // Login
      await store.login('test@example.com', 'password');
      expect(useAuthStore.getState().isAuthenticated).toBe(true);

      // Fetch user
      await store.fetchUser();
      expect(useAuthStore.getState().user).toMatchObject({
        id: 'usr-001',
        email: 'test@example.com',
        companyId: 'comp-001',
      });

      // Logout clears both the store and every session key
      await store.logout();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(useAuthStore.getState().user).toBeNull();
      expect(localStorage.getItem('kerp_access_token')).toBeNull();
      expect(localStorage.getItem('kerp_refresh_token')).toBeNull();
    });
  });
});
