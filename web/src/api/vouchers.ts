import { apiClient } from "./client";
import type {
  Voucher,
  VoucherEntry,
  VoucherStatus,
  PaginatedResponse,
  PaginationParams,
} from "@/types";

/**
 * Voucher list query parameters
 */
export interface VoucherListParams extends PaginationParams {
  search?: string;
  status?: VoucherStatus;
  startDate?: string;
  endDate?: string;
  accountId?: string;
  createdBy?: string;
  [key: string]: string | number | boolean | undefined;
}

/**
 * Create voucher request data
 */
export interface CreateVoucherData {
  voucherDate: string;
  description: string;
  entries: Omit<VoucherEntry, "id" | "voucherId" | "account" | "sequence">[];
}

/**
 * Update voucher request data
 */
export interface UpdateVoucherData extends Partial<CreateVoucherData> {
  status?: VoucherStatus;
}

/**
 * Voucher summary for list display
 */
export interface VoucherSummary {
  id: string;
  voucherNumber: string;
  voucherDate: string;
  description: string;
  status: VoucherStatus;
  totalDebit: number;
  totalCredit: number;
  entryCount: number;
  createdBy: string;
  createdByName?: string;
  createdAt: string;
  approvedBy?: string;
  approvedByName?: string;
  approvedAt?: string;
}

/**
 * Voucher approval request
 */
export interface ApprovalRequest {
  action: "approve" | "reject";
  comment?: string;
}

/**
 * Voucher API client
 */
export const vouchersApi = {
  /**
   * Get paginated list of vouchers
   */
  list: async (params?: VoucherListParams) => {
    return apiClient.get<PaginatedResponse<VoucherSummary>>("/vouchers", params);
  },

  /**
   * Get a single voucher by ID with full details
   */
  get: async (id: string) => {
    return apiClient.get<Voucher>(`/vouchers/${id}`);
  },

  /**
   * Create a new voucher
   */
  create: async (data: CreateVoucherData) => {
    return apiClient.post<Voucher>("/vouchers", data);
  },

  /**
   * Update an existing voucher
   */
  update: async (id: string, data: UpdateVoucherData) => {
    return apiClient.put<Voucher>(`/vouchers/${id}`, data);
  },

  /**
   * Delete a voucher (only draft status allowed)
   */
  delete: async (id: string) => {
    return apiClient.delete<void>(`/vouchers/${id}`);
  },

  /**
   * Submit voucher for approval
   */
  submit: async (id: string) => {
    return apiClient.post<Voucher>(`/vouchers/${id}/submit`);
  },

  /**
   * Approve or reject a voucher
   */
  approval: async (id: string, request: ApprovalRequest) => {
    return apiClient.post<Voucher>(`/vouchers/${id}/approval`, request);
  },

  /**
   * Bulk approve vouchers
   */
  bulkApprove: async (ids: string[], comment?: string) => {
    return apiClient.post<{ success: number; failed: number }>(
      "/vouchers/bulk-approve",
      { ids, comment }
    );
  },

  /**
   * Bulk reject vouchers
   */
  bulkReject: async (ids: string[], comment?: string) => {
    return apiClient.post<{ success: number; failed: number }>(
      "/vouchers/bulk-reject",
      { ids, comment }
    );
  },

  /**
   * Copy a voucher to create a new draft
   */
  copy: async (id: string) => {
    return apiClient.post<Voucher>(`/vouchers/${id}/copy`);
  },

  /**
   * Reverse a voucher (create a reversing entry)
   */
  reverse: async (id: string, reverseDate?: string) => {
    return apiClient.post<Voucher>(`/vouchers/${id}/reverse`, { reverseDate });
  },

  /**
   * Get next voucher number
   */
  getNextNumber: async (date?: string) => {
    return apiClient.get<{ voucherNumber: string }>("/vouchers/next-number", {
      date,
    });
  },

  /**
   * Export vouchers to Excel
   */
  export: async (params?: VoucherListParams) => {
    return apiClient.get<Blob>("/vouchers/export", {
      ...params,
      responseType: "blob",
    } as Record<string, string | number | boolean | undefined>);
  },

  /**
   * Validate voucher entries (check balance)
   */
  validate: async (entries: Omit<VoucherEntry, "id" | "voucherId" | "account">[]) => {
    return apiClient.post<{
      valid: boolean;
      totalDebit: number;
      totalCredit: number;
      difference: number;
      errors?: string[];
    }>("/vouchers/validate", { entries });
  },
};
