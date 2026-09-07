import { apiClient } from "./client";
import { compactParams, mapEnvelope, mapPaginated } from "./envelope";
import type { Partner, PartnerType, PaginationParams } from "@/types";

/**
 * Partner API client.
 *
 * Routes mirror internal/handler/partner_handler.go:
 *   GET    /partners             (query: page, page_size, search, type, is_active)
 *   GET    /partners/stats
 *   GET    /partners/:id
 *   GET    /partners/code/:code
 *   GET    /partners/bizno/:bizno
 *   POST   /partners
 *   PUT    /partners/:id
 *   DELETE /partners/:id
 *   GET    /partners/:id/can-delete
 *   POST   /partners/activate | /partners/deactivate  (body: { ids: [...] })
 *
 * There is no `/partners/search` and no `PATCH /partners/:id`.
 */

// Wire type: internal/dto/partner_dto.go PartnerResponse
export interface PartnerWire {
  id: string;
  code: string;
  name: string;
  name_en?: string;
  business_number?: string;
  partner_type: PartnerType;
  representative?: string;
  phone?: string;
  fax?: string;
  email?: string;
  website?: string;
  zip_code?: string;
  address?: string;
  address_detail?: string;
  payment_term_days: number;
  credit_limit: number;
  ar_account_id?: string;
  ap_account_id?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface PartnerListParams extends PaginationParams {
  search?: string;
  partnerType?: PartnerType;
  isActive?: boolean;
}

export interface CreatePartnerData {
  code: string;
  name: string;
  nameEn?: string;
  businessNumber?: string;
  partnerType: PartnerType;
  representativeName?: string;
  phone?: string;
  fax?: string;
  email?: string;
  website?: string;
  zipCode?: string;
  address?: string;
  addressDetail?: string;
  paymentTermDays?: number;
  creditLimit?: number;
  arAccountId?: string;
  apAccountId?: string;
  isActive?: boolean;
}

export type UpdatePartnerData = CreatePartnerData;

export interface PartnerStats {
  totalCount: number;
  customerCount: number;
  vendorCount: number;
  activeCount: number;
  inactiveCount: number;
}

export function toPartner(wire: PartnerWire): Partner {
  return {
    id: wire.id,
    companyId: "",
    code: wire.code,
    name: wire.name,
    nameEn: wire.name_en,
    businessNumber: wire.business_number,
    representativeName: wire.representative,
    partnerType: wire.partner_type,
    phone: wire.phone,
    fax: wire.fax,
    email: wire.email,
    website: wire.website,
    zipCode: wire.zip_code,
    address: wire.address,
    addressDetail: wire.address_detail,
    paymentTermDays: wire.payment_term_days,
    creditLimit: wire.credit_limit,
    arAccountId: wire.ar_account_id,
    apAccountId: wire.ap_account_id,
    isActive: wire.is_active,
    createdAt: wire.created_at,
    updatedAt: wire.updated_at,
  };
}

function toBody(data: CreatePartnerData): Record<string, unknown> {
  const body: Record<string, unknown> = {
    code: data.code,
    name: data.name,
    partner_type: data.partnerType,
  };
  const optional: Record<string, unknown> = {
    name_en: data.nameEn,
    business_number: data.businessNumber,
    representative: data.representativeName,
    phone: data.phone,
    fax: data.fax,
    email: data.email,
    website: data.website,
    zip_code: data.zipCode,
    address: data.address,
    address_detail: data.addressDetail,
    payment_term_days: data.paymentTermDays,
    credit_limit: data.creditLimit,
    ar_account_id: data.arAccountId,
    ap_account_id: data.apAccountId,
    is_active: data.isActive,
  };
  Object.entries(optional).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      body[key] = value;
    }
  });
  return body;
}

export const partnersApi = {
  /** GET /partners */
  list: async (params?: PartnerListParams) => {
    const response = await apiClient.get<PartnerWire[]>(
      "/partners",
      compactParams({
        page: params?.page,
        page_size: params?.pageSize,
        search: params?.search,
        type: params?.partnerType,
        is_active: params?.isActive,
      })
    );
    return mapPaginated(response, toPartner);
  },

  /** GET /partners/:id */
  get: async (id: string) => {
    const response = await apiClient.get<PartnerWire>(`/partners/${id}`);
    return mapEnvelope(response, toPartner);
  },

  /** GET /partners/code/:code */
  getByCode: async (code: string) => {
    const response = await apiClient.get<PartnerWire>(`/partners/code/${code}`);
    return mapEnvelope(response, toPartner);
  },

  /** GET /partners/bizno/:bizno */
  getByBusinessNumber: async (businessNumber: string) => {
    const response = await apiClient.get<PartnerWire>(
      `/partners/bizno/${businessNumber}`
    );
    return mapEnvelope(response, toPartner);
  },

  /** POST /partners */
  create: async (data: CreatePartnerData) => {
    const response = await apiClient.post<PartnerWire>("/partners", toBody(data));
    return mapEnvelope(response, toPartner);
  },

  /** PUT /partners/:id (the backend has no PATCH) */
  update: async (id: string, data: UpdatePartnerData) => {
    const response = await apiClient.put<PartnerWire>(
      `/partners/${id}`,
      toBody(data)
    );
    return mapEnvelope(response, toPartner);
  },

  /** DELETE /partners/:id */
  delete: async (id: string) => {
    return apiClient.delete<{ message?: string }>(`/partners/${id}`);
  },

  /** GET /partners/:id/can-delete */
  canDelete: async (id: string) => {
    return apiClient.get<{ can_delete: boolean; reason?: string }>(
      `/partners/${id}/can-delete`
    );
  },

  /** POST /partners/activate | /partners/deactivate */
  setActive: async (ids: string[], isActive: boolean) => {
    return apiClient.post<{ message?: string }>(
      isActive ? "/partners/activate" : "/partners/deactivate",
      { ids }
    );
  },

  /** GET /partners/stats */
  stats: async () => {
    const response = await apiClient.get<{
      total_count: number;
      customer_count: number;
      vendor_count: number;
      active_count: number;
      inactive_count: number;
    }>("/partners/stats");
    return mapEnvelope(
      response,
      (wire): PartnerStats => ({
        totalCount: wire.total_count,
        customerCount: wire.customer_count,
        vendorCount: wire.vendor_count,
        activeCount: wire.active_count,
        inactiveCount: wire.inactive_count,
      })
    );
  },

  /** Search = the list endpoint's `search` filter. */
  search: async (query: string, partnerType?: PartnerType) => {
    const response = await apiClient.get<PartnerWire[]>(
      "/partners",
      compactParams({
        search: query,
        type: partnerType,
        page_size: 50,
        is_active: true,
      })
    );
    return { ...response, data: (response.data ?? []).map(toPartner) };
  },
};
