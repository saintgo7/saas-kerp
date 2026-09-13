import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient, getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";

// ---------------------------------------------------------------------------
// Backend wire types (snake_case) — see internal/dto/voucher_dto.go
// ---------------------------------------------------------------------------

interface VoucherEntryWire {
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
}

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
  entries?: VoucherEntryWire[];
  created_at: string;
  updated_at: string;
}

// ---------------------------------------------------------------------------
// UI types (camelCase)
// ---------------------------------------------------------------------------

export interface VoucherEntry {
  id?: string;
  lineNo?: number;
  accountId: string;
  accountCode?: string;
  accountName?: string;
  debitAmount: number;
  creditAmount: number;
  description?: string;
  partnerId?: string;
  partnerName?: string;
}

export type VoucherStatus =
  | "draft"
  | "pending"
  | "approved"
  | "posted"
  | "rejected"
  | "cancelled";

export type VoucherType =
  | "general"
  | "sales"
  | "purchase"
  | "payment"
  | "receipt"
  | "adjustment"
  | "closing";

export interface Voucher {
  id: string;
  voucherNo: string;
  voucherDate: string;
  voucherType: VoucherType;
  voucherTypeLabel?: string;
  description: string;
  status: VoucherStatus;
  statusLabel?: string;
  entries: VoucherEntry[];
  totalDebit: number;
  totalCredit: number;
  createdAt: string;
  updatedAt: string;
}

export interface VoucherListParams {
  page?: number;
  pageSize?: number;
  voucherType?: VoucherType;
  status?: VoucherStatus;
  dateFrom?: string;
  dateTo?: string;
  accountId?: string;
  partnerId?: string;
  search?: string;
  includeEntries?: boolean;
}

export interface VoucherListResult {
  data: Voucher[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CreateVoucherInput {
  voucherDate: string;
  voucherType: VoucherType;
  description?: string;
  entries: {
    accountId: string;
    debitAmount: number;
    creditAmount: number;
    description?: string;
    partnerId?: string;
  }[];
}

// ---------------------------------------------------------------------------
// Mapping
// ---------------------------------------------------------------------------

function toEntry(entry: VoucherEntryWire): VoucherEntry {
  return {
    id: entry.id,
    lineNo: entry.line_no,
    accountId: entry.account_id,
    accountCode: entry.account_code,
    accountName: entry.account_name,
    debitAmount: entry.debit_amount,
    creditAmount: entry.credit_amount,
    description: entry.description,
    partnerId: entry.partner_id,
    partnerName: entry.partner_name,
  };
}

export function toVoucher(wire: VoucherWire): Voucher {
  return {
    id: wire.id,
    voucherNo: wire.voucher_no,
    voucherDate: wire.voucher_date,
    voucherType: wire.voucher_type as VoucherType,
    voucherTypeLabel: wire.voucher_type_label,
    description: wire.description ?? "",
    status: wire.status as VoucherStatus,
    statusLabel: wire.status_label,
    entries: (wire.entries ?? []).map(toEntry),
    totalDebit: wire.total_debit,
    totalCredit: wire.total_credit,
    createdAt: wire.created_at,
    updatedAt: wire.updated_at,
  };
}

/** Frontend params -> the query string the Go handler actually reads. */
function toListQuery(params: VoucherListParams): Record<string, string | number | boolean> {
  const query: Record<string, string | number | boolean> = {};
  if (params.page !== undefined) query.page = params.page;
  if (params.pageSize !== undefined) query.page_size = params.pageSize;
  if (params.voucherType) query.voucher_type = params.voucherType;
  if (params.status) query.status = params.status;
  if (params.dateFrom) query.date_from = params.dateFrom;
  if (params.dateTo) query.date_to = params.dateTo;
  if (params.accountId) query.account_id = params.accountId;
  if (params.partnerId) query.partner_id = params.partnerId;
  if (params.search) query.search = params.search;
  if (params.includeEntries) query.include_entries = params.includeEntries;
  return query;
}

function toCreateBody(input: CreateVoucherInput) {
  return {
    voucher_date: input.voucherDate,
    // Required by the backend (oneof=general sales purchase payment receipt
    // adjustment closing); omitting it made every create fail with 400.
    voucher_type: input.voucherType,
    description: input.description ?? "",
    entries: input.entries.map((entry) => ({
      account_id: entry.accountId,
      debit_amount: entry.debitAmount,
      credit_amount: entry.creditAmount,
      description: entry.description ?? "",
      ...(entry.partnerId ? { partner_id: entry.partnerId } : {}),
    })),
  };
}

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const voucherKeys = {
  all: ["vouchers"] as const,
  lists: () => [...voucherKeys.all, "list"] as const,
  list: (params: VoucherListParams) => [...voucherKeys.lists(), params] as const,
  details: () => [...voucherKeys.all, "detail"] as const,
  detail: (id: string) => [...voucherKeys.details(), id] as const,
};

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------

export function useVouchers(params: VoucherListParams = {}) {
  return useQuery<VoucherListResult>({
    queryKey: voucherKeys.list(params),
    queryFn: async () => {
      const response = await apiClient.get<VoucherWire[]>(
        "/vouchers",
        { params: toListQuery(params) }
      );
      // internal/dto/common.go: pagination lives at meta.pagination and the
      // page-size key is `per_page`.
      const pagination = response.meta?.pagination;
      const items = (response.data ?? []).map(toVoucher);
      const pageSize = pagination?.per_page ?? params.pageSize ?? items.length;
      const total = pagination?.total ?? items.length;
      return {
        data: items,
        total,
        page: pagination?.page ?? params.page ?? 1,
        pageSize,
        totalPages:
          pagination?.total_pages ??
          (pageSize > 0 ? Math.ceil(total / pageSize) : 1),
      };
    },
  });
}

export function useVoucher(id: string) {
  return useQuery<Voucher>({
    queryKey: voucherKeys.detail(id),
    queryFn: async () => {
      const response = await apiClient.get<VoucherWire>(`/vouchers/${id}`);
      return toVoucher(response.data);
    },
    enabled: !!id,
  });
}

export function useCreateVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateVoucherInput) => {
      const response = await apiClient.post<VoucherWire>(
        "/vouchers",
        toCreateBody(input)
      );
      return toVoucher(response.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      toast.success("전표 저장 완료", "전표가 성공적으로 저장되었습니다.");
    },
    onError: (error: unknown) => {
      toast.error(
        "저장 실패",
        getErrorMessage(error, "전표 저장 중 오류가 발생했습니다.")
      );
    },
  });
}

export function useUpdateVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      ...input
    }: CreateVoucherInput & { id: string }) => {
      // PUT /vouchers/:id takes UpdateVoucherRequest, which has no voucher_type.
      const { voucher_type: _voucherType, ...body } = toCreateBody(input);
      const response = await apiClient.put<VoucherWire>(
        `/vouchers/${id}`,
        body
      );
      return toVoucher(response.data);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      queryClient.invalidateQueries({ queryKey: voucherKeys.detail(data.id) });
      toast.success("전표 수정 완료", "전표가 성공적으로 수정되었습니다.");
    },
    onError: (error: unknown) => {
      toast.error(
        "수정 실패",
        getErrorMessage(error, "전표 수정 중 오류가 발생했습니다.")
      );
    },
  });
}

export function useDeleteVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await apiClient.delete(`/vouchers/${id}`);
      return id;
    },
    onSuccess: (id) => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      queryClient.removeQueries({ queryKey: voucherKeys.detail(id) });
      toast.success("전표 삭제 완료", "전표가 삭제되었습니다.");
    },
    onError: (error: unknown) => {
      toast.error(
        "삭제 실패",
        getErrorMessage(error, "전표 삭제 중 오류가 발생했습니다.")
      );
    },
  });
}

export function useApproveVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      // The backend route is /approve (and /reject), not /approval.
      const response = await apiClient.post<VoucherWire>(
        `/vouchers/${id}/approve`
      );
      return toVoucher(response.data);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      queryClient.invalidateQueries({ queryKey: voucherKeys.detail(data.id) });
      toast.success("전표 승인 완료", "전표가 승인되었습니다.");
    },
    onError: (error: unknown) => {
      toast.error(
        "승인 실패",
        getErrorMessage(error, "전표 승인 중 오류가 발생했습니다.")
      );
    },
  });
}

export function useRejectVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, reason }: { id: string; reason?: string }) => {
      const response = await apiClient.post<VoucherWire>(
        `/vouchers/${id}/reject`,
        { reason: reason ?? "" }
      );
      return toVoucher(response.data);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      queryClient.invalidateQueries({ queryKey: voucherKeys.detail(data.id) });
      toast.success("전표 반려 완료", "전표가 반려되었습니다.");
    },
    onError: (error: unknown) => {
      toast.error(
        "반려 실패",
        getErrorMessage(error, "전표 반려 중 오류가 발생했습니다.")
      );
    },
  });
}
