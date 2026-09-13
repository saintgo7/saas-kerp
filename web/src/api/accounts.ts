import { apiClient } from "./client";
import { compactParams, mapEnvelope, mapPaginated } from "./envelope";
import type { Account, AccountType, PaginationParams } from "@/types";

/**
 * Account API client.
 *
 * Routes mirror internal/handler/account_handler.go exactly:
 *   GET    /accounts            (query: page, page_size, search, type, is_active)
 *   GET    /accounts/tree
 *   GET    /accounts/:id
 *   GET    /accounts/code/:code
 *   POST   /accounts
 *   PUT    /accounts/:id
 *   DELETE /accounts/:id
 *   GET    /accounts/:id/children
 *   GET    /accounts/:id/can-delete
 *   PUT    /accounts/:id/move
 *
 * There is no `/accounts/search` and no `PATCH /accounts/:id`; the former was
 * being swallowed by the `/accounts/:id` route and failing UUID parsing.
 */

// Wire type: internal/dto/account_dto.go AccountResponse
export interface AccountWire {
  id: string;
  code: string;
  name: string;
  name_en?: string;
  parent_id?: string;
  level: number;
  path?: string;
  account_type: AccountType;
  account_type_label?: string;
  account_nature?: "debit" | "credit";
  account_nature_label?: string;
  account_category?: string;
  is_active: boolean;
  is_control_account: boolean;
  allow_direct_posting: boolean;
  sort_order: number;
  children?: AccountWire[];
  created_at: string;
  updated_at: string;
}

export interface AccountListParams extends PaginationParams {
  search?: string;
  type?: AccountType;
  isActive?: boolean;
}

export interface CreateAccountData {
  code: string;
  name: string;
  nameEn?: string;
  type: AccountType;
  parentId?: string;
  accountNature?: "debit" | "credit";
  accountCategory?: string;
  isActive?: boolean;
  isControlAccount?: boolean;
  allowDirectPosting?: boolean;
  sortOrder?: number;
}

export interface UpdateAccountData extends CreateAccountData {
  isActive: boolean;
}

export interface AccountTreeNode extends Account {
  children?: AccountTreeNode[];
}

/** Default debit/credit side for a type, used when the caller omits it. */
export function defaultNature(type: AccountType): "debit" | "credit" {
  return type === "asset" || type === "expense" ? "debit" : "credit";
}

export function toAccount(wire: AccountWire): AccountTreeNode {
  return {
    id: wire.id,
    companyId: "",
    code: wire.code,
    name: wire.name,
    nameEn: wire.name_en,
    type: wire.account_type,
    parentId: wire.parent_id || undefined,
    level: wire.level,
    isActive: wire.is_active,
    accountNature: wire.account_nature,
    accountCategory: wire.account_category,
    isControlAccount: wire.is_control_account,
    allowDirectPosting: wire.allow_direct_posting,
    sortOrder: wire.sort_order,
    description: wire.account_category,
    children: wire.children?.map(toAccount),
  };
}

function toCreateBody(data: CreateAccountData) {
  return compactParamsObject({
    code: data.code,
    name: data.name,
    name_en: data.nameEn,
    parent_id: data.parentId,
    account_type: data.type,
    account_nature: data.accountNature ?? defaultNature(data.type),
    account_category: data.accountCategory,
    is_active: data.isActive,
    is_control_account: data.isControlAccount,
    allow_direct_posting: data.allowDirectPosting,
    sort_order: data.sortOrder,
  });
}

/** Like compactParams but keeps booleans/zero and allows nested values. */
function compactParamsObject(obj: Record<string, unknown>) {
  const out: Record<string, unknown> = {};
  Object.entries(obj).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      out[key] = value;
    }
  });
  return out;
}

export const accountsApi = {
  /** GET /accounts */
  list: async (params?: AccountListParams) => {
    const response = await apiClient.get<AccountWire[]>(
      "/accounts",
      compactParams({
        page: params?.page,
        page_size: params?.pageSize,
        search: params?.search,
        type: params?.type,
        is_active: params?.isActive,
      })
    );
    return mapPaginated(response, toAccount);
  },

  /** GET /accounts/tree */
  tree: async (type?: AccountType) => {
    const response = await apiClient.get<AccountWire[]>(
      "/accounts/tree",
      compactParams({ type })
    );
    return {
      ...response,
      data: (response.data ?? []).map(toAccount),
    };
  },

  /** GET /accounts/:id */
  get: async (id: string) => {
    const response = await apiClient.get<AccountWire>(`/accounts/${id}`);
    return mapEnvelope(response, toAccount);
  },

  /** GET /accounts/code/:code */
  getByCode: async (code: string) => {
    const response = await apiClient.get<AccountWire>(`/accounts/code/${code}`);
    return mapEnvelope(response, toAccount);
  },

  /** POST /accounts */
  create: async (data: CreateAccountData) => {
    const response = await apiClient.post<AccountWire>(
      "/accounts",
      toCreateBody(data)
    );
    return mapEnvelope(response, toAccount);
  },

  /** PUT /accounts/:id (the backend has no PATCH) */
  update: async (id: string, data: UpdateAccountData) => {
    const response = await apiClient.put<AccountWire>(`/accounts/${id}`, {
      ...toCreateBody(data),
      // UpdateAccountRequest requires account_nature and is_active explicitly.
      account_nature: data.accountNature ?? defaultNature(data.type),
      is_active: data.isActive,
    });
    return mapEnvelope(response, toAccount);
  },

  /** DELETE /accounts/:id */
  delete: async (id: string) => {
    return apiClient.delete<{ message?: string }>(`/accounts/${id}`);
  },

  /** GET /accounts/:id/can-delete */
  canDelete: async (id: string) => {
    return apiClient.get<{ can_delete: boolean; reason?: string }>(
      `/accounts/${id}/can-delete`
    );
  },

  /** GET /accounts/:id/children */
  children: async (parentId: string) => {
    const response = await apiClient.get<AccountWire[]>(
      `/accounts/${parentId}/children`
    );
    return { ...response, data: (response.data ?? []).map(toAccount) };
  },

  /** PUT /accounts/:id/move */
  move: async (id: string, parentId?: string) => {
    const response = await apiClient.put<AccountWire>(`/accounts/${id}/move`, {
      parent_id: parentId ?? "",
    });
    return mapEnvelope(response, toAccount);
  },

  /**
   * Search. The backend has no dedicated search route, so this is the list
   * endpoint with its `search` filter.
   */
  search: async (query: string, type?: AccountType) => {
    const response = await apiClient.get<AccountWire[]>(
      "/accounts",
      compactParams({ search: query, type, page_size: 50, is_active: true })
    );
    return { ...response, data: (response.data ?? []).map(toAccount) };
  },
};
