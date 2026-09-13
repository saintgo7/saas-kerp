import type { ApiResponse, PaginatedResponse } from "@/types";

/**
 * The Go handlers answer with snake_case wire types. These helpers keep the
 * translation in the API layer so pages and hooks only ever see the camelCase
 * domain types declared in `@/types`.
 */

/** Re-wrap an envelope with a mapped payload. */
export function mapEnvelope<W, D>(
  response: ApiResponse<W>,
  map: (wire: W) => D
): ApiResponse<D> {
  return {
    ...response,
    data: map(response.data),
  };
}

/**
 * Turn `{ data: W[], meta: { pagination: {...} } }` into the
 * `PaginatedResponse<D>` shape the UI expects.
 *
 * The backend emits one envelope (internal/dto/common.go): pagination lives at
 * `meta.pagination` and the page size key is `per_page`.
 */
export function mapPaginated<W, D>(
  response: ApiResponse<W[]>,
  map: (wire: W) => D
): ApiResponse<PaginatedResponse<D>> {
  const items = (response.data ?? []).map(map);
  const pagination = response.meta?.pagination;

  const pageSize = pagination?.per_page ?? items.length;
  const total = pagination?.total ?? items.length;
  const page = pagination?.page ?? 1;
  const totalPages =
    pagination?.total_pages ?? (pageSize > 0 ? Math.ceil(total / pageSize) : 1);

  return {
    success: response.success,
    error: response.error,
    meta: response.meta,
    data: { items, total, page, pageSize, totalPages },
  };
}

/** Drop undefined/null/empty entries so they never reach the query string. */
export function compactParams(
  params: Record<string, string | number | boolean | undefined | null>
): Record<string, string | number | boolean> {
  const out: Record<string, string | number | boolean> = {};
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      out[key] = value;
    }
  });
  return out;
}
