import { apiClient } from "./client";
import type { Partner, PaginatedResponse, PaginationParams } from "@/types";

export interface PartnerListParams extends PaginationParams {
  search?: string;
  partnerType?: string;
  isActive?: boolean;
  [key: string]: string | number | boolean | undefined;
}

export interface CreatePartnerData {
  code: string;
  name: string;
  businessNumber?: string;
  representativeName?: string;
  businessType?: string;
  businessCategory?: string;
  partnerType: string;
  address?: string;
  phone?: string;
  fax?: string;
  email?: string;
  bankName?: string;
  bankAccount?: string;
  accountHolder?: string;
  note?: string;
}

export interface UpdatePartnerData extends Partial<CreatePartnerData> {
  isActive?: boolean;
}

/**
 * Partner API client
 */
export const partnersApi = {
  /**
   * Get paginated list of partners
   */
  list: async (params?: PartnerListParams) => {
    return apiClient.get<PaginatedResponse<Partner>>("/partners", params);
  },

  /**
   * Get a single partner by ID
   */
  get: async (id: string) => {
    return apiClient.get<Partner>(`/partners/${id}`);
  },

  /**
   * Create a new partner
   */
  create: async (data: CreatePartnerData) => {
    return apiClient.post<Partner>("/partners", data);
  },

  /**
   * Update an existing partner
   */
  update: async (id: string, data: UpdatePartnerData) => {
    return apiClient.put<Partner>(`/partners/${id}`, data);
  },

  /**
   * Delete a partner
   */
  delete: async (id: string) => {
    return apiClient.delete<void>(`/partners/${id}`);
  },

  /**
   * Toggle partner active status
   */
  toggleActive: async (id: string, isActive: boolean) => {
    return apiClient.patch<Partner>(`/partners/${id}`, { isActive });
  },

  /**
   * Search partners by name or code
   */
  search: async (query: string, partnerType?: string) => {
    return apiClient.get<Partner[]>("/partners/search", { q: query, partnerType });
  },
};
