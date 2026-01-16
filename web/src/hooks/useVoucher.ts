import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/services/api";
import { toast } from "@/stores/ui";

// Types
export interface VoucherEntry {
  id?: string;
  accountCode: string;
  accountName?: string;
  debitAmount: number;
  creditAmount: number;
  description?: string;
}

export interface Voucher {
  id: string;
  voucherNo: string;
  voucherDate: string;
  description: string;
  status: "draft" | "approved" | "posted";
  entries: VoucherEntry[];
  totalDebit: number;
  totalCredit: number;
  createdAt: string;
  createdBy: string;
}

export interface VoucherListParams {
  page?: number;
  limit?: number;
  startDate?: string;
  endDate?: string;
  status?: string;
  search?: string;
}

export interface CreateVoucherInput {
  voucherDate: string;
  description: string;
  entries: Omit<VoucherEntry, "id">[];
}

// Query keys
export const voucherKeys = {
  all: ["vouchers"] as const,
  lists: () => [...voucherKeys.all, "list"] as const,
  list: (params: VoucherListParams) => [...voucherKeys.lists(), params] as const,
  details: () => [...voucherKeys.all, "detail"] as const,
  detail: (id: string) => [...voucherKeys.details(), id] as const,
};

// Get voucher list
export function useVouchers(params: VoucherListParams = {}) {
  return useQuery({
    queryKey: voucherKeys.list(params),
    queryFn: async () => {
      const { data } = await api.get<{
        data: Voucher[];
        total: number;
        page: number;
        limit: number;
      }>("/api/v1/vouchers", { params });
      return data;
    },
  });
}

// Get single voucher
export function useVoucher(id: string) {
  return useQuery({
    queryKey: voucherKeys.detail(id),
    queryFn: async () => {
      const { data } = await api.get<Voucher>(`/api/v1/vouchers/${id}`);
      return data;
    },
    enabled: !!id,
  });
}

// Create voucher
export function useCreateVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateVoucherInput) => {
      const { data } = await api.post<Voucher>("/api/v1/vouchers", input);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      toast.success("전표 저장 완료", "전표가 성공적으로 저장되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("저장 실패", error.message || "전표 저장 중 오류가 발생했습니다.");
    },
  });
}

// Update voucher
export function useUpdateVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      ...input
    }: CreateVoucherInput & { id: string }) => {
      const { data } = await api.put<Voucher>(`/api/v1/vouchers/${id}`, input);
      return data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      queryClient.invalidateQueries({ queryKey: voucherKeys.detail(data.id) });
      toast.success("전표 수정 완료", "전표가 성공적으로 수정되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("수정 실패", error.message || "전표 수정 중 오류가 발생했습니다.");
    },
  });
}

// Delete voucher
export function useDeleteVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/v1/vouchers/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      toast.success("전표 삭제 완료", "전표가 삭제되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("삭제 실패", error.message || "전표 삭제 중 오류가 발생했습니다.");
    },
  });
}

// Approve voucher
export function useApproveVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const { data } = await api.post<Voucher>(`/api/v1/vouchers/${id}/approve`);
      return data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: voucherKeys.lists() });
      queryClient.invalidateQueries({ queryKey: voucherKeys.detail(data.id) });
      toast.success("전표 승인 완료", "전표가 승인되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("승인 실패", error.message || "전표 승인 중 오류가 발생했습니다.");
    },
  });
}
