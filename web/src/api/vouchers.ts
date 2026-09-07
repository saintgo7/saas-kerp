import { apiClient } from "./client";
import { compactParams, mapEnvelope, mapPaginated } from "./envelope";
import type { PaginationParams } from "@/types";
import {
  toVoucher,
  type CreateVoucherInput,
  type Voucher,
  type VoucherStatus,
  type VoucherType,
} from "@/hooks/useVoucher";

/**
 * Voucher API client.
 *
 * Routes mirror internal/handler/voucher_handler.go:
 *   GET    /vouchers            (voucher_type, status, date_from, date_to,
 *                                account_id, partner_id, search, page, page_size)
 *   GET    /vouchers/pending
 *   GET    /vouchers/:id
 *   GET    /vouchers/no/:voucher_no
 *   POST   /vouchers
 *   PUT    /vouchers/:id
 *   DELETE /vouchers/:id
 *   PUT    /vouchers/:id/entries
 *   POST   /vouchers/:id/{submit,approve,reject,post,cancel,reverse}
 *
 * Removed because the backend has no such route (they were 404s, and
 * `/vouchers/next-number` was being parsed as `/vouchers/:id` -> invalid UUID):
 *   /vouchers/:id/approval, /vouchers/next-number, /vouchers/export,
 *   /vouchers/:id/copy, /vouchers/validate, /vouchers/bulk-approve,
 *   /vouchers/bulk-reject
 */

// Wire type: internal/dto/voucher_dto.go VoucherResponse
interface VoucherWire {
  id: string;
  voucher_no: string;
  voucher_date: string;
  voucher_type: string;
  voucher_type_label?: string;
  status: string;
  status_label?: string;
  total_debit: number;
  total_credit: number;
  description?: string;
  entries?: {
    id: string;
    line_no: number;
    account_id: string;
    account_code?: string;
    account_name?: string;
    debit_amount: number;
    credit_amount: number;
    description?: string;
    partner_id?: string;
    partner_name?: string;
  }[];
  created_at: string;
  updated_at: string;
}

export interface VoucherListParams extends PaginationParams {
  search?: string;
  status?: VoucherStatus;
  voucherType?: VoucherType;
  startDate?: string;
  endDate?: string;
  accountId?: string;
  partnerId?: string;
  includeEntries?: boolean;
}

export type CreateVoucherData = CreateVoucherInput;

export type UpdateVoucherData = Omit<CreateVoucherInput, "voucherType">;

export type VoucherSummary = Voucher;

/** approve / reject share one UI action but are two backend routes. */
export interface ApprovalRequest {
  action: "approve" | "reject";
  comment?: string;
}

function toCreateBody(data: CreateVoucherData) {
  return {
    voucher_date: data.voucherDate,
    voucher_type: data.voucherType,
    description: data.description ?? "",
    entries: data.entries.map((entry) => ({
      account_id: entry.accountId,
      debit_amount: entry.debitAmount,
      credit_amount: entry.creditAmount,
      description: entry.description ?? "",
      ...(entry.partnerId ? { partner_id: entry.partnerId } : {}),
    })),
  };
}

export const vouchersApi = {
  /** GET /vouchers */
  list: async (params?: VoucherListParams) => {
    const response = await apiClient.get<VoucherWire[]>(
      "/vouchers",
      compactParams({
        page: params?.page,
        page_size: params?.pageSize,
        search: params?.search,
        status: params?.status,
        voucher_type: params?.voucherType,
        date_from: params?.startDate,
        date_to: params?.endDate,
        account_id: params?.accountId,
        partner_id: params?.partnerId,
        include_entries: params?.includeEntries,
      })
    );
    return mapPaginated(response, toVoucher);
  },

  /** GET /vouchers/pending */
  pending: async () => {
    const response = await apiClient.get<VoucherWire[]>("/vouchers/pending");
    return { ...response, data: (response.data ?? []).map(toVoucher) };
  },

  /** GET /vouchers/:id */
  get: async (id: string) => {
    const response = await apiClient.get<VoucherWire>(`/vouchers/${id}`);
    return mapEnvelope(response, toVoucher);
  },

  /** GET /vouchers/no/:voucher_no */
  getByNo: async (voucherNo: string) => {
    const response = await apiClient.get<VoucherWire>(
      `/vouchers/no/${voucherNo}`
    );
    return mapEnvelope(response, toVoucher);
  },

  /** POST /vouchers */
  create: async (data: CreateVoucherData) => {
    const response = await apiClient.post<VoucherWire>(
      "/vouchers",
      toCreateBody(data)
    );
    return mapEnvelope(response, toVoucher);
  },

  /** PUT /vouchers/:id — UpdateVoucherRequest carries no voucher_type */
  update: async (id: string, data: UpdateVoucherData) => {
    const { voucher_type: _voucherType, ...body } = toCreateBody({
      ...data,
      voucherType: "general",
    });
    const response = await apiClient.put<VoucherWire>(`/vouchers/${id}`, body);
    return mapEnvelope(response, toVoucher);
  },

  /** DELETE /vouchers/:id (draft only) */
  delete: async (id: string) => {
    return apiClient.delete<{ message?: string }>(`/vouchers/${id}`);
  },

  /** POST /vouchers/:id/submit */
  submit: async (id: string) => {
    const response = await apiClient.post<VoucherWire>(
      `/vouchers/${id}/submit`
    );
    return mapEnvelope(response, toVoucher);
  },

  /** POST /vouchers/:id/approve or /reject (two distinct backend routes) */
  approval: async (id: string, request: ApprovalRequest) => {
    const path =
      request.action === "approve"
        ? `/vouchers/${id}/approve`
        : `/vouchers/${id}/reject`;
    const response = await apiClient.post<VoucherWire>(path, {
      reason: request.comment ?? "",
    });
    return mapEnvelope(response, toVoucher);
  },

  /** POST /vouchers/:id/post */
  post: async (id: string) => {
    const response = await apiClient.post<VoucherWire>(`/vouchers/${id}/post`);
    return mapEnvelope(response, toVoucher);
  },

  /** POST /vouchers/:id/cancel */
  cancel: async (id: string, reason?: string) => {
    const response = await apiClient.post<VoucherWire>(
      `/vouchers/${id}/cancel`,
      { reason: reason ?? "" }
    );
    return mapEnvelope(response, toVoucher);
  },

  /** POST /vouchers/:id/reverse */
  reverse: async (id: string, reversalDate: string, description?: string) => {
    const response = await apiClient.post<VoucherWire>(
      `/vouchers/${id}/reverse`,
      { reversal_date: reversalDate, description: description ?? "" }
    );
    return mapEnvelope(response, toVoucher);
  },
};
