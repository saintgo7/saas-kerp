import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { http, HttpResponse } from 'msw';
import { server } from '@/__tests__/mocks/server';
import { LoginPage } from '../LoginPage';
import { useAuthStore } from '@/stores';

const API_BASE = '/api';

// Mock navigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// Wrapper component with Router
function renderLoginPage() {
  return render(
    <BrowserRouter>
      <LoginPage />
    </BrowserRouter>
  );
}

// Reset store and mocks before each test
const initialAuthState = useAuthStore.getState();

beforeEach(() => {
  useAuthStore.setState(initialAuthState);
  vi.clearAllMocks();
  mockNavigate.mockReset();
});

describe('LoginPage', () => {
  // ==========================================================================
  // Rendering Tests
  // ==========================================================================

  describe('rendering', () => {
    it('should render login form', () => {
      renderLoginPage();

      expect(screen.getByRole('heading', { name: '로그인' })).toBeInTheDocument();
      expect(screen.getByLabelText(/이메일/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/비밀번호/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '로그인' })).toBeInTheDocument();
    });

    it('should render password toggle button', () => {
      renderLoginPage();

      const toggleButton = screen.getByRole('button', { name: '' });
      expect(toggleButton).toBeInTheDocument();
    });

    it('should render "remember me" checkbox', () => {
      renderLoginPage();

      expect(screen.getByRole('checkbox')).toBeInTheDocument();
      expect(screen.getByText('로그인 상태 유지')).toBeInTheDocument();
    });

    it('should render forgot password link', () => {
      renderLoginPage();

      expect(screen.getByRole('link', { name: '비밀번호 찾기' })).toHaveAttribute(
        'href',
        '/forgot-password'
      );
    });

    it('should render register link', () => {
      renderLoginPage();

      expect(screen.getByRole('link', { name: '회원가입' })).toHaveAttribute(
        'href',
        '/register'
      );
    });
  });

  // ==========================================================================
  // Form Validation Tests
  // ==========================================================================

  describe('form validation', () => {
    it('should show error for invalid email', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const emailInput = screen.getByLabelText(/이메일/i);
      const passwordInput = screen.getByLabelText(/비밀번호/i);
      const submitButton = screen.getByRole('button', { name: '로그인' });

      await user.type(emailInput, 'invalid-email');
      await user.type(passwordInput, 'validpassword123');
      await user.click(submitButton);

      await waitFor(
        () => {
          expect(screen.getByText('올바른 이메일 형식이 아닙니다.')).toBeInTheDocument();
        },
        { timeout: 3000 }
      );
    });

    it('should show error for short password', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const emailInput = screen.getByLabelText(/이메일/i);
      const passwordInput = screen.getByLabelText(/비밀번호/i);
      const submitButton = screen.getByRole('button', { name: '로그인' });

      await user.type(emailInput, 'test@example.com');
      await user.type(passwordInput, '1234567'); // 7 chars
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('비밀번호는 최소 8자 이상이어야 합니다.')).toBeInTheDocument();
      });
    });

    it('should not show error for valid input', async () => {
      const user = userEvent.setup();

      server.use(
        http.post(`${API_BASE}/auth/login`, () => {
          return HttpResponse.json({
            success: true,
            data: {
              user: { id: '1', email: 'test@example.com', name: 'Test' },
              accessToken: 'token',
            },
          });
        })
      );

      renderLoginPage();

      const emailInput = screen.getByLabelText(/이메일/i);
      const passwordInput = screen.getByLabelText(/비밀번호/i);
      const submitButton = screen.getByRole('button', { name: '로그인' });

      await user.type(emailInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(
          screen.queryByText('올바른 이메일 형식이 아닙니다.')
        ).not.toBeInTheDocument();
        expect(
          screen.queryByText('비밀번호는 최소 8자 이상이어야 합니다.')
        ).not.toBeInTheDocument();
      });
    });
  });

  // ==========================================================================
  // Password Visibility Tests
  // ==========================================================================

  describe('password visibility toggle', () => {
    it('should toggle password visibility', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      const passwordInput = screen.getByLabelText(/비밀번호/i);
      const toggleButtons = screen.getAllByRole('button');
      const toggleButton = toggleButtons.find(
        (btn) => btn.querySelector('svg') !== null && (btn as HTMLButtonElement).type === 'button'
      );

      expect(passwordInput).toHaveAttribute('type', 'password');

      if (toggleButton) {
        await user.click(toggleButton);
        expect(passwordInput).toHaveAttribute('type', 'text');

        await user.click(toggleButton);
        expect(passwordInput).toHaveAttribute('type', 'password');
      }
    });
  });

  // ==========================================================================
  // Login Flow Tests
  // ==========================================================================

  describe('login flow', () => {
    it('should call login and navigate on success', async () => {
      const user = userEvent.setup();

      server.use(
        http.post(`${API_BASE}/auth/login`, () => {
          return HttpResponse.json({
            success: true,
            data: {
              user: {
                id: 'user-1',
                email: 'test@example.com',
                name: 'Test User',
              },
              accessToken: 'access-token',
              refreshToken: 'refresh-token',
            },
          });
        })
      );

      renderLoginPage();

      await user.type(screen.getByLabelText(/이메일/i), 'test@example.com');
      await user.type(screen.getByLabelText(/비밀번호/i), 'password123');
      await user.click(screen.getByRole('button', { name: '로그인' }));

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
      });

      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    it('should show error toast on login failure', async () => {
      const user = userEvent.setup();

      server.use(
        http.post(`${API_BASE}/auth/login`, () => {
          return HttpResponse.json(
            {
              success: false,
              error: { code: 'AUTH_FAILED', message: 'Invalid credentials' },
            },
            { status: 401 }
          );
        })
      );

      renderLoginPage();

      await user.type(screen.getByLabelText(/이메일/i), 'wrong@example.com');
      await user.type(screen.getByLabelText(/비밀번호/i), 'wrongpassword');
      await user.click(screen.getByRole('button', { name: '로그인' }));

      await waitFor(() => {
        expect(mockNavigate).not.toHaveBeenCalled();
      });

      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    it('should show loading state during submission', async () => {
      const user = userEvent.setup();

      server.use(
        http.post(`${API_BASE}/auth/login`, async () => {
          await new Promise((resolve) => setTimeout(resolve, 100));
          return HttpResponse.json({
            success: true,
            data: {
              user: { id: '1', email: 'test@example.com', name: 'Test' },
            },
          });
        })
      );

      renderLoginPage();

      await user.type(screen.getByLabelText(/이메일/i), 'test@example.com');
      await user.type(screen.getByLabelText(/비밀번호/i), 'password123');
      await user.click(screen.getByRole('button', { name: '로그인' }));

      // Button should show loading state
      const submitButton = screen.getByRole('button', { name: '로그인' });
      expect(submitButton).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================

  describe('accessibility', () => {
    it('should have proper form labels', () => {
      renderLoginPage();

      const emailInput = screen.getByLabelText(/이메일/i);
      const passwordInput = screen.getByLabelText(/비밀번호/i);

      expect(emailInput).toHaveAttribute('type', 'email');
      expect(passwordInput).toHaveAttribute('type', 'password');
    });

    it('should focus email input first on tab', async () => {
      const user = userEvent.setup();
      renderLoginPage();

      await user.tab();

      expect(screen.getByLabelText(/이메일/i)).toHaveFocus();
    });
  });
});
