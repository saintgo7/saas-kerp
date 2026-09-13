import { apiClient as httpClient } from "@/services/api";
import type { ApiResponse } from "@/types";

export type QueryParams = Record<
  string,
  string | number | boolean | undefined | null
>;

/**
 * Thin wrapper over the shared axios instance in `services/api.ts`.
 *
 * It used to be a standalone `fetch` client. That had two fatal problems:
 *  1. `new URL("/api/v1/...")` throws for a relative base, so every GET failed.
 *  2. It had no token-refresh logic and cleared only two of the four session
 *     keys on 401, which produced a /login <-> /dashboard redirect loop.
 *
 * Delegating means every page-level call now goes through the same request
 * interceptor, refresh queue and session teardown.
 */
class ApiClient {
  private toConfig(params?: QueryParams) {
    if (!params) return undefined;
    const cleaned: Record<string, string | number | boolean> = {};
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== "") {
        cleaned[key] = value;
      }
    });
    return { params: cleaned };
  }

  async get<T>(endpoint: string, params?: QueryParams): Promise<ApiResponse<T>> {
    return httpClient.get<T>(endpoint, this.toConfig(params));
  }

  async post<T>(endpoint: string, data?: unknown): Promise<ApiResponse<T>> {
    return httpClient.post<T>(endpoint, data);
  }

  async put<T>(endpoint: string, data?: unknown): Promise<ApiResponse<T>> {
    return httpClient.put<T>(endpoint, data);
  }

  async patch<T>(endpoint: string, data?: unknown): Promise<ApiResponse<T>> {
    return httpClient.patch<T>(endpoint, data);
  }

  async delete<T>(endpoint: string): Promise<ApiResponse<T>> {
    return httpClient.delete<T>(endpoint);
  }

  /** Binary download. Returns the Blob itself, not an envelope. */
  async getBlob(endpoint: string, params?: QueryParams): Promise<Blob> {
    return httpClient.getBlob(endpoint, this.toConfig(params));
  }
}

export const apiClient = new ApiClient();
