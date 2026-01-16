import { apiClient, setTokens, clearTokens } from "./api";
import { STORAGE_KEYS } from "@/constants";
import type {
  User,
  AuthTokens,
  LoginCredentials,
  RegisterData,
} from "@/types";

interface LoginResponse {
  user: User;
  tokens: AuthTokens;
}

// Backend response format (snake_case)
interface BackendLoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user: User;
}

interface RegisterResponse {
  user: User;
  tokens: AuthTokens;
}

export const authService = {
  /**
   * Login with email and password
   */
  async login(credentials: LoginCredentials): Promise<LoginResponse> {
    const response = await apiClient.post<BackendLoginResponse>(
      "/auth/login",
      credentials
    );

    if (response.success) {
      // Map snake_case backend response to camelCase frontend format
      const tokens: AuthTokens = {
        accessToken: response.data.access_token,
        refreshToken: response.data.refresh_token,
        expiresAt: Date.now() + response.data.expires_in * 1000,
      };
      setTokens(tokens);
      localStorage.setItem(
        STORAGE_KEYS.user,
        JSON.stringify(response.data.user)
      );
      return { user: response.data.user, tokens };
    }

    throw new Error("Login failed");
  },

  /**
   * Register a new user and company
   */
  async register(data: RegisterData): Promise<RegisterResponse> {
    const response = await apiClient.post<RegisterResponse>(
      "/auth/register",
      data
    );

    if (response.success) {
      setTokens(response.data.tokens);
      localStorage.setItem(
        STORAGE_KEYS.user,
        JSON.stringify(response.data.user)
      );
    }

    return response.data;
  },

  /**
   * Logout current user
   */
  async logout(): Promise<void> {
    try {
      await apiClient.post("/auth/logout");
    } finally {
      clearTokens();
    }
  },

  /**
   * Get current user info
   */
  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<User>("/auth/me");
    if (response.success) {
      localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(response.data));
    }
    return response.data;
  },

  /**
   * Check if user is authenticated
   */
  isAuthenticated(): boolean {
    return !!localStorage.getItem(STORAGE_KEYS.accessToken);
  },

  /**
   * Get stored user from localStorage
   */
  getStoredUser(): User | null {
    const userStr = localStorage.getItem(STORAGE_KEYS.user);
    if (!userStr) return null;
    try {
      return JSON.parse(userStr) as User;
    } catch {
      return null;
    }
  },

  /**
   * Request password reset
   */
  async requestPasswordReset(email: string): Promise<void> {
    await apiClient.post("/auth/password-reset/request", { email });
  },

  /**
   * Reset password with token
   */
  async resetPassword(token: string, password: string): Promise<void> {
    await apiClient.post("/auth/password-reset/confirm", { token, password });
  },

  /**
   * Change password
   */
  async changePassword(
    currentPassword: string,
    newPassword: string
  ): Promise<void> {
    await apiClient.post("/auth/change-password", {
      currentPassword,
      newPassword,
    });
  },

  /**
   * Verify email
   */
  async verifyEmail(token: string): Promise<void> {
    await apiClient.post("/auth/verify-email", { token });
  },

  /**
   * Resend verification email
   */
  async resendVerificationEmail(): Promise<void> {
    await apiClient.post("/auth/verify-email/resend");
  },
};
