import {
  apiClient,
  setTokens,
  clearTokens,
  mapTokens,
  type BackendTokenPayload,
} from "./api";
import { STORAGE_KEYS } from "@/constants";
import { safeStorage } from "@/lib/storage";
import type {
  User,
  AuthTokens,
  LoginCredentials,
  RegisterData,
  UserRole,
} from "@/types";

interface LoginResponse {
  user: User;
  tokens: AuthTokens;
}

// Backend response format, see internal/service/auth_service.go
interface BackendUser {
  id: string;
  company_id: string;
  email: string;
  name: string;
  role: UserRole;
  status?: string;
  phone?: string;
  created_at?: string;
  updated_at?: string;
}

interface BackendAuthResponse extends BackendTokenPayload {
  user: BackendUser;
}

// GET /auth/me returns claims, not a full user row.
interface BackendMeResponse {
  user_id: string;
  company_id: string;
  email: string;
  name: string;
  roles?: string[];
}

function mapUser(user: BackendUser): User {
  return {
    id: user.id,
    email: user.email,
    name: user.name,
    phone: user.phone,
    role: user.role,
    companyId: user.company_id,
    createdAt: user.created_at ?? "",
    updatedAt: user.updated_at ?? "",
  };
}

function cacheUser(user: User): void {
  safeStorage.setItem(STORAGE_KEYS.user, JSON.stringify(user));
}

export const authService = {
  /**
   * Login with email and password
   */
  async login(credentials: LoginCredentials): Promise<LoginResponse> {
    const response = await apiClient.post<BackendAuthResponse>(
      "/auth/login",
      credentials
    );

    const data = response.data;
    const tokens = mapTokens(data);
    if (!tokens) {
      throw new Error("로그인 응답에 토큰이 없습니다.");
    }

    setTokens(tokens);

    const user = mapUser(data.user);
    cacheUser(user);

    return { user, tokens };
  },

  /**
   * Register a new user and company.
   * The backend expects snake_case (internal/handler/auth.go RegisterRequest).
   */
  async register(data: RegisterData): Promise<LoginResponse> {
    const response = await apiClient.post<BackendAuthResponse>(
      "/auth/register",
      {
        company_name: data.companyName,
        business_number: data.businessNumber,
        email: data.email,
        password: data.password,
        name: data.name,
        phone: data.phone ?? "",
      }
    );

    const payload = response.data;
    const tokens = mapTokens(payload);
    if (!tokens) {
      throw new Error("회원가입 응답에 토큰이 없습니다.");
    }

    setTokens(tokens);

    const user = mapUser(payload.user);
    cacheUser(user);

    return { user, tokens };
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
   * Get current user info. `/auth/me` returns JWT claims, so the cached
   * user from login fills in whatever the claims do not carry.
   */
  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<BackendMeResponse>("/auth/me");
    const claims = response.data;
    const cached = this.getStoredUser();

    const user: User = {
      id: claims.user_id,
      email: claims.email,
      name: claims.name,
      phone: cached?.phone,
      role: (claims.roles?.[0] as UserRole) ?? cached?.role ?? "user",
      companyId: claims.company_id,
      createdAt: cached?.createdAt ?? "",
      updatedAt: cached?.updatedAt ?? "",
    };

    cacheUser(user);
    return user;
  },

  /**
   * Check if user is authenticated
   */
  isAuthenticated(): boolean {
    return !!safeStorage.getItem(STORAGE_KEYS.accessToken);
  },

  /**
   * Get stored user from localStorage
   */
  getStoredUser(): User | null {
    const userStr = safeStorage.getItem(STORAGE_KEYS.user);
    if (!userStr) return null;
    try {
      const parsed = JSON.parse(userStr) as User;
      // Guard against a foreign shape ever landing on this key.
      return parsed && typeof parsed.email === "string" ? parsed : null;
    } catch {
      return null;
    }
  },

  /**
   * Request a password reset link.
   * POST /auth/forgot-password (internal/router/v1.go).
   */
  async requestPasswordReset(email: string): Promise<{ message: string }> {
    const response = await apiClient.post<{ message: string }>(
      "/auth/forgot-password",
      { email }
    );
    return response.data;
  },

  /**
   * Change the password of the signed-in user.
   * PUT /auth/password (internal/handler/auth.go ChangePasswordRequest).
   */
  async changePassword(
    currentPassword: string,
    newPassword: string
  ): Promise<void> {
    await apiClient.put("/auth/password", {
      current_password: currentPassword,
      new_password: newPassword,
    });
  },
};
