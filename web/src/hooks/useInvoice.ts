import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/services/api";
import { toast } from "@/stores/ui";

// Types
export interface InvoiceItem {
  id?: string;
  itemDate: string;
  itemName: string;
  specification?: string;
  quantity: number;
  unitPrice: number;
  supplyAmount: number;
  taxAmount: number;
  note?: string;
}

export interface Invoice {
  id: string;
  invoiceNo: string;
  invoiceType: "01" | "02" | "03" | "04";
  taxType: "taxable" | "zero" | "exempt";
  issueDate: string;
  status: "draft" | "issued" | "sent" | "cancelled";

  // Supplier
  supplierBizNo: string;
  supplierName: string;
  supplierCeoName: string;
  supplierAddress: string;
  supplierBizType?: string;
  supplierBizItem?: string;
  supplierEmail: string;

  // Buyer
  buyerBizNo: string;
  buyerName: string;
  buyerCeoName: string;
  buyerAddress?: string;
  buyerBizType?: string;
  buyerBizItem?: string;
  buyerEmail: string;
  buyerEmail2?: string;

  // Amounts
  totalSupply: number;
  totalTax: number;
  totalAmount: number;

  // Items
  items: InvoiceItem[];

  note?: string;
  createdAt: string;
  issuedAt?: string;
}

export interface InvoiceListParams {
  page?: number;
  limit?: number;
  startDate?: string;
  endDate?: string;
  status?: string;
  type?: "sales" | "purchase";
  search?: string;
}

export interface CreateInvoiceInput {
  invoiceType: string;
  taxType: string;
  issueDate: string;
  supplierBizNo: string;
  supplierName: string;
  supplierCeoName: string;
  supplierAddress: string;
  supplierBizType?: string;
  supplierBizItem?: string;
  supplierEmail: string;
  buyerBizNo: string;
  buyerName: string;
  buyerCeoName: string;
  buyerAddress?: string;
  buyerBizType?: string;
  buyerBizItem?: string;
  buyerEmail: string;
  buyerEmail2?: string;
  items: Omit<InvoiceItem, "id">[];
  note?: string;
}

// Query keys
export const invoiceKeys = {
  all: ["invoices"] as const,
  lists: () => [...invoiceKeys.all, "list"] as const,
  list: (params: InvoiceListParams) => [...invoiceKeys.lists(), params] as const,
  details: () => [...invoiceKeys.all, "detail"] as const,
  detail: (id: string) => [...invoiceKeys.details(), id] as const,
};

// Get invoice list
export function useInvoices(params: InvoiceListParams = {}) {
  return useQuery({
    queryKey: invoiceKeys.list(params),
    queryFn: async () => {
      const { data } = await api.get<{
        data: Invoice[];
        total: number;
        page: number;
        limit: number;
      }>("/api/v1/invoices", { params });
      return data;
    },
  });
}

// Get single invoice
export function useInvoice(id: string) {
  return useQuery({
    queryKey: invoiceKeys.detail(id),
    queryFn: async () => {
      const { data } = await api.get<Invoice>(`/api/v1/invoices/${id}`);
      return data;
    },
    enabled: !!id,
  });
}

// Create invoice (draft)
export function useCreateInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateInvoiceInput) => {
      const { data } = await api.post<Invoice>("/api/v1/invoices", input);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: invoiceKeys.lists() });
      toast.success("세금계산서 저장 완료", "세금계산서가 저장되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("저장 실패", error.message || "세금계산서 저장 중 오류가 발생했습니다.");
    },
  });
}

// Update invoice
export function useUpdateInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      ...input
    }: CreateInvoiceInput & { id: string }) => {
      const { data } = await api.put<Invoice>(`/api/v1/invoices/${id}`, input);
      return data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: invoiceKeys.lists() });
      queryClient.invalidateQueries({ queryKey: invoiceKeys.detail(data.id) });
      toast.success("세금계산서 수정 완료", "세금계산서가 수정되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("수정 실패", error.message || "세금계산서 수정 중 오류가 발생했습니다.");
    },
  });
}

// Issue invoice (send to NTS)
export function useIssueInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const { data } = await api.post<Invoice>(`/api/v1/invoices/${id}/issue`);
      return data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: invoiceKeys.lists() });
      queryClient.invalidateQueries({ queryKey: invoiceKeys.detail(data.id) });
      toast.success("세금계산서 발행 완료", "세금계산서가 국세청에 전송되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("발행 실패", error.message || "세금계산서 발행 중 오류가 발생했습니다.");
    },
  });
}

// Cancel invoice
export function useCancelInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const { data } = await api.post<Invoice>(`/api/v1/invoices/${id}/cancel`);
      return data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: invoiceKeys.lists() });
      queryClient.invalidateQueries({ queryKey: invoiceKeys.detail(data.id) });
      toast.success("세금계산서 취소 완료", "세금계산서가 취소되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("취소 실패", error.message || "세금계산서 취소 중 오류가 발생했습니다.");
    },
  });
}

// Delete invoice (only draft)
export function useDeleteInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/v1/invoices/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: invoiceKeys.lists() });
      toast.success("세금계산서 삭제 완료", "세금계산서가 삭제되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("삭제 실패", error.message || "세금계산서 삭제 중 오류가 발생했습니다.");
    },
  });
}
