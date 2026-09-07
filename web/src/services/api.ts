import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from "axios";
import { API_BASE_URL, STORAGE_KEYS } from "@/constants";
import { safeStorage } from "@/lib/storage";
import { clearSession, redirectToLogin } from "@/lib/session";
import type { ApiResponse, AuthTokens } from "@/types";

// Create axios instance
const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

/**
 * Normalized error shape every caller can rely on.
 */
export interface ApiRequestError {
  status?: number;
  /** internal/errors/codes.go vocabulary, e.g. "VAL_001". */
  code?: string;
  message: string;
  detail?: string;
  details?: { field: string; message: string }[];
}

/**
 * Endpoints whose own 401 means "these credentials are wrong", not
 * "the access token expired". Refreshing or redirecting on those turns a
 * failed login into a full page reload that wipes the form and the error.
 */
const AUTH_ENDPOINTS = [
  "/auth/login",
  "/auth/register",
  "/auth/refresh",
  "/auth/forgot-password",
];

function isAuthEndpoint(url?: string): boolean {
  if (!url) return false;
  return AUTH_ENDPOINTS.some((endpoint) => url.includes(endpoint));
}

// ---------------------------------------------------------------------------
// Token management
// ---------------------------------------------------------------------------

let isRefreshing = false;
let refreshSubscribers: {
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}[] = [];

function subscribeTokenRefresh(
  resolve: (token: string) => void,
  reject: (error: unknown) => void
): void {
  refreshSubscribers.push({ resolve, reject });
}

function onTokenRefreshed(token: string): void {
  const subscribers = refreshSubscribers;
  refreshSubscribers = [];
  subscribers.forEach(({ resolve }) => resolve(token));
}

/**
 * Every queued request must be settled when a refresh fails, otherwise the
 * promises stay pending forever and their screens spin until a reload.
 */
function onTokenRefreshFailed(error: unknown): void {
  const subscribers = refreshSubscribers;
  refreshSubscribers = [];
  subscribers.forEach(({ reject }) => reject(error));
}

function getAccessToken(): string | null {
  return safeStorage.getItem(STORAGE_KEYS.accessToken);
}

function getRefreshToken(): string | null {
  return safeStorage.getItem(STORAGE_KEYS.refreshToken);
}

function setTokens(tokens: AuthTokens): void {
  if (!tokens?.accessToken || !tokens?.refreshToken) {
    throw new Error("Invalid token payload received from the server");
  }
  safeStorage.setItem(STORAGE_KEYS.accessToken, tokens.accessToken);
  safeStorage.setItem(STORAGE_KEYS.refreshToken, tokens.refreshToken);
}

function clearTokens(): void {
  clearSession();
}

// Backend token payload (snake_case, see internal/service/auth_service.go)
interface BackendTokenPayload {
  access_token: string;
  refresh_token: string;
  token_type?: string;
  expires_in?: number;
}

/**
 * Maps the backend's snake_case token payload onto the frontend shape.
 * Returns null when the payload is not usable, so we never persist the
 * string "undefined" into localStorage.
 */
function mapTokens(payload: BackendTokenPayload | undefined): AuthTokens | null {
  if (!payload?.access_token || !payload?.refresh_token) return null;
  return {
    accessToken: payload.access_token,
    refreshToken: payload.refresh_token,
    expiresAt: Date.now() + (payload.expires_in ?? 0) * 1000,
  };
}

// ---------------------------------------------------------------------------
// Interceptors
// ---------------------------------------------------------------------------

api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = getAccessToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: AxiosError) => Promise.reject(error)
);

/**
 * The API has exactly one error shape (internal/dto/common.go):
 *   { success: false, error: { code, message, detail?, details? }, meta }
 * There is no top-level `message` — reading one is why server errors used to
 * reach the user as a generic axios string. Codes come from
 * internal/errors/codes.go ("VAL_001", "AUTH_001", "RES_001", ...).
 */
function normalizeError(error: AxiosError<ApiResponse<unknown>>): ApiRequestError {
  const body = error.response?.data;
  return {
    status: error.response?.status,
    code: body?.error?.code,
    message:
      body?.error?.message || error.message || "요청을 처리하지 못했습니다.",
    detail: body?.error?.detail,
    details: body?.error?.details,
  };
}

async function performRefresh(): Promise<string> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    throw new Error("No refresh token available");
  }

  // /auth/refresh is a public route (internal/router/v1.go): the refresh token
  // in the body is the credential, so no Authorization header is sent. Using
  // the bare `axios` (not the `api` instance) also keeps this call out of the
  // response interceptor, so a failed refresh cannot recurse into itself.
  const response = await axios.post<ApiResponse<BackendTokenPayload>>(
    `${API_BASE_URL}/auth/refresh`,
    { refresh_token: refreshToken }
  );

  const tokens = mapTokens(response.data?.data);
  if (!response.data?.success || !tokens) {
    throw new Error("Token refresh returned an unusable payload");
  }

  setTokens(tokens);
  return tokens.accessToken;
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiResponse<unknown>>) => {
    const originalRequest = error.config as
      | (AxiosRequestConfig & { _retry?: boolean })
      | undefined;

    const shouldAttemptRefresh =
      error.response?.status === 401 &&
      originalRequest !== undefined &&
      !originalRequest._retry &&
      !isAuthEndpoint(originalRequest.url);

    if (shouldAttemptRefresh && originalRequest) {
      if (isRefreshing) {
        // Queue behind the in-flight refresh; both outcomes settle the promise.
        return new Promise((resolve, reject) => {
          subscribeTokenRefresh(
            (token: string) => {
              originalRequest._retry = true;
              originalRequest.headers = {
                ...originalRequest.headers,
                Authorization: `Bearer ${token}`,
              };
              resolve(api(originalRequest));
            },
            (refreshError) => reject(refreshError)
          );
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const token = await performRefresh();
        onTokenRefreshed(token);
        originalRequest.headers = {
          ...originalRequest.headers,
          Authorization: `Bearer ${token}`,
        };
        return await api(originalRequest);
      } catch (refreshError) {
        onTokenRefreshFailed(refreshError);
        clearTokens();
        redirectToLogin();
        return Promise.reject(normalizeError(error));
      } finally {
        // Reset on every path, including the early failures above, otherwise
        // the flag stays true and every later 401 queues up forever.
        isRefreshing = false;
      }
    }

    return Promise.reject(normalizeError(error));
  }
);

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export const apiClient = {
  get: <T>(url: string, config?: AxiosRequestConfig) =>
    api.get<ApiResponse<T>>(url, config).then((res) => res.data),

  post: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) =>
    api.post<ApiResponse<T>>(url, data, config).then((res) => res.data),

  put: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) =>
    api.put<ApiResponse<T>>(url, data, config).then((res) => res.data),

  patch: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) =>
    api.patch<ApiResponse<T>>(url, data, config).then((res) => res.data),

  delete: <T>(url: string, config?: AxiosRequestConfig) =>
    api.delete<ApiResponse<T>>(url, config).then((res) => res.data),

  /** Binary download (exports). Returns the raw Blob, never JSON-parsed. */
  getBlob: (url: string, config?: AxiosRequestConfig) =>
    api
      .get<Blob>(url, { ...config, responseType: "blob" })
      .then((res) => res.data),
};

/** Human-readable message for any rejection thrown by this module. */
export function getErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === "string") return error;
  if (error && typeof error === "object" && "message" in error) {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string" && message.length > 0) return message;
  }
  return fallback;
}

export { setTokens, clearTokens, getAccessToken, mapTokens };
export type { BackendTokenPayload };
export default api;
