import { apiClient } from "./client";
import type { Account, AccountType, PaginatedResponse, PaginationParams } from "@/types";

export interface AccountListParams extends PaginationParams {
  search?: string;
  type?: AccountType;
  isActive?: boolean;
  parentId?: string;
  [key: string]: string | number | boolean | undefined;
}

export interface CreateAccountData {
  code: string;
  name: string;
  type: AccountType;
  parentId?: string;
  description?: string;
}

export interface UpdateAccountData extends Partial<CreateAccountData> {
  isActive?: boolean;
}

export interface AccountTreeNode extends Account {
  children?: AccountTreeNode[];
}

/**
 * Account API client
 */
export const accountsApi = {
  /**
   * Get paginated list of accounts
   */
  list: async (params?: AccountListParams) => {
    return apiClient.get<PaginatedResponse<Account>>("/accounts", params);
  },

  /**
   * Get accounts as tree structure
   */
  tree: async (type?: AccountType) => {
    return apiClient.get<AccountTreeNode[]>("/accounts/tree", { type });
  },

  /**
   * Get a single account by ID
   */
  get: async (id: string) => {
    return apiClient.get<Account>(`/accounts/${id}`);
  },

  /**
   * Create a new account
   */
  create: async (data: CreateAccountData) => {
    return apiClient.post<Account>("/accounts", data);
  },

  /**
   * Update an existing account
   */
  update: async (id: string, data: UpdateAccountData) => {
    return apiClient.put<Account>(`/accounts/${id}`, data);
  },

  /**
   * Delete an account
   */
  delete: async (id: string) => {
    return apiClient.delete<void>(`/accounts/${id}`);
  },

  /**
   * Toggle account active status
   */
  toggleActive: async (id: string, isActive: boolean) => {
    return apiClient.patch<Account>(`/accounts/${id}`, { isActive });
  },

  /**
   * Search accounts by name or code
   */
  search: async (query: string, type?: AccountType) => {
    return apiClient.get<Account[]>("/accounts/search", { q: query, type });
  },

  /**
   * Get child accounts of a parent
   */
  children: async (parentId: string) => {
    return apiClient.get<Account[]>(`/accounts/${parentId}/children`);
  },
};
