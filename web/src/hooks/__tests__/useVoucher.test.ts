import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { server } from '@/__tests__/mocks/server';
import {
  createWrapper,
  createMockVoucher,
  envelope,
  errorEnvelope,
} from '@/__tests__/test-utils';
import {
  useVouchers,
  useVoucher,
  useCreateVoucher,
  useUpdateVoucher,
  useDeleteVoucher,
  useApproveVoucher,
  voucherKeys,
} from '../useVoucher';

// Must match constants/index.ts API_BASE_URL and the Gin router (/api/v1).
const API_BASE = '/api/v1';

describe('useVoucher Hooks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ==========================================================================
  // useVouchers Tests
  // ==========================================================================

  describe('useVouchers', () => {
    it('should fetch voucher list successfully', async () => {
      const mockVouchers = [
        createMockVoucher({ id: 'vch-001' }),
        createMockVoucher({ id: 'vch-002' }),
      ];

      server.use(
        http.get(`${API_BASE}/vouchers`, () => {
          return HttpResponse.json(
            envelope(mockVouchers, {
              pagination: { page: 1, per_page: 20, total: 2, total_pages: 1 },
            })
          );
        })
      );

      const { result } = renderHook(() => useVouchers(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data?.data).toHaveLength(2);
      expect(result.current.data?.total).toBe(2);
    });

    it('should fetch vouchers with filters', async () => {
      let capturedParams: URLSearchParams | undefined;

      server.use(
        http.get(`${API_BASE}/vouchers`, ({ request }) => {
          capturedParams = new URL(request.url).searchParams;
          return HttpResponse.json(
            envelope([createMockVoucher({ status: 'draft' })], {
              pagination: { page: 1, per_page: 10, total: 1, total_pages: 1 },
            })
          );
        })
      );

      const { result } = renderHook(
        () => useVouchers({ status: 'draft', page: 1, pageSize: 10 }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(capturedParams).toBeDefined();
      expect(capturedParams!.get('status')).toBe('draft');
      expect(capturedParams!.get('page')).toBe('1');
      // The Go handler reads `page_size`, not `limit`
      expect(capturedParams!.get('page_size')).toBe('10');
    });

    it('should handle fetch error', async () => {
      server.use(
        http.get(`${API_BASE}/vouchers`, () => {
          return HttpResponse.json(
            errorEnvelope('SRV_001', 'Server error'),
            { status: 500 }
          );
        })
      );

      const { result } = renderHook(() => useVouchers(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  // ==========================================================================
  // useVoucher (single) Tests
  // ==========================================================================

  describe('useVoucher', () => {
    it('should fetch single voucher by ID', async () => {
      const mockVoucher = createMockVoucher({ id: 'vch-test-001' });

      server.use(
        http.get(`${API_BASE}/vouchers/vch-test-001`, () => {
          return HttpResponse.json(envelope(mockVoucher));
        })
      );

      const { result } = renderHook(() => useVoucher('vch-test-001'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data?.id).toBe('vch-test-001');
    });

    it('should not fetch when ID is empty', async () => {
      const { result } = renderHook(() => useVoucher(''), {
        wrapper: createWrapper(),
      });

      // Query should not be enabled
      expect(result.current.isFetching).toBe(false);
    });

    it('should handle 404 error', async () => {
      server.use(
        http.get(`${API_BASE}/vouchers/nonexistent`, () => {
          return HttpResponse.json(
            errorEnvelope('RES_001', 'Voucher not found'),
            { status: 404 }
          );
        })
      );

      const { result } = renderHook(() => useVoucher('nonexistent'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  // ==========================================================================
  // useCreateVoucher Tests
  // ==========================================================================

  describe('useCreateVoucher', () => {
    it('should create voucher successfully', async () => {
      const newVoucher = {
        voucherDate: '2026-01-15',
        voucherType: 'general' as const,
        description: 'New test voucher',
        entries: [
          { accountId: 'acc-001', debitAmount: 10000, creditAmount: 0 },
          { accountId: 'acc-003', debitAmount: 0, creditAmount: 10000 },
        ],
      };

      let capturedBody: Record<string, unknown> | undefined;

      server.use(
        http.post(`${API_BASE}/vouchers`, async ({ request }) => {
          capturedBody = (await request.json()) as Record<string, unknown>;
          return HttpResponse.json(
            envelope(
              createMockVoucher({
                id: 'vch-new-001',
                status: 'draft',
                total_debit: 10000,
                total_credit: 10000,
              })
            ),
            { status: 201 }
          );
        })
      );

      const { result } = renderHook(() => useCreateVoucher(), {
        wrapper: createWrapper(),
      });

      await result.current.mutateAsync(newVoucher);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data?.id).toBe('vch-new-001');
      expect(result.current.data?.status).toBe('draft');

      // The backend requires snake_case and a voucher_type
      expect(capturedBody).toMatchObject({
        voucher_date: '2026-01-15',
        voucher_type: 'general',
        entries: [
          { account_id: 'acc-001', debit_amount: 10000, credit_amount: 0 },
          { account_id: 'acc-003', debit_amount: 0, credit_amount: 10000 },
        ],
      });
    });

    it('should handle validation error for unbalanced voucher', async () => {
      const unbalancedVoucher = {
        voucherDate: '2026-01-15',
        voucherType: 'general' as const,
        description: 'Unbalanced voucher',
        entries: [
          { accountId: 'acc-001', debitAmount: 10000, creditAmount: 0 },
          { accountId: 'acc-003', debitAmount: 0, creditAmount: 5000 }, // Not balanced
        ],
      };

      server.use(
        http.post(`${API_BASE}/vouchers`, () => {
          return HttpResponse.json(
            errorEnvelope('VAL_001', 'Debit and credit must be equal'),
            { status: 400 }
          );
        })
      );

      const { result } = renderHook(() => useCreateVoucher(), {
        wrapper: createWrapper(),
      });

      await expect(result.current.mutateAsync(unbalancedVoucher)).rejects.toThrow();

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  // ==========================================================================
  // useUpdateVoucher Tests
  // ==========================================================================

  describe('useUpdateVoucher', () => {
    it('should update voucher successfully', async () => {
      const updatedVoucher = {
        id: 'vch-001',
        voucherDate: '2026-01-16',
        voucherType: 'general' as const,
        description: 'Updated description',
        entries: [
          { accountId: 'acc-001', debitAmount: 20000, creditAmount: 0 },
          { accountId: 'acc-003', debitAmount: 0, creditAmount: 20000 },
        ],
      };

      let capturedBody: Record<string, unknown> | undefined;

      server.use(
        http.put(`${API_BASE}/vouchers/vch-001`, async ({ request }) => {
          capturedBody = (await request.json()) as Record<string, unknown>;
          return HttpResponse.json(
            envelope(
              createMockVoucher({
                id: 'vch-001',
                description: 'Updated description',
              })
            )
          );
        })
      );

      const { result } = renderHook(() => useUpdateVoucher(), {
        wrapper: createWrapper(),
      });

      await result.current.mutateAsync(updatedVoucher);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data?.description).toBe('Updated description');
      // UpdateVoucherRequest carries no voucher_type
      expect(capturedBody).not.toHaveProperty('voucher_type');
      expect(capturedBody).toMatchObject({ voucher_date: '2026-01-16' });
    });

    it('should handle not found error', async () => {
      server.use(
        http.put(`${API_BASE}/vouchers/nonexistent`, () => {
          return HttpResponse.json(
            errorEnvelope('RES_001', 'Voucher not found'),
            { status: 404 }
          );
        })
      );

      const { result } = renderHook(() => useUpdateVoucher(), {
        wrapper: createWrapper(),
      });

      await expect(
        result.current.mutateAsync({
          id: 'nonexistent',
          voucherDate: '2026-01-15',
          voucherType: 'general' as const,
          description: 'Test',
          entries: [],
        })
      ).rejects.toThrow();
    });
  });

  // ==========================================================================
  // useDeleteVoucher Tests
  // ==========================================================================

  describe('useDeleteVoucher', () => {
    it('should delete voucher successfully', async () => {
      server.use(
        http.delete(`${API_BASE}/vouchers/vch-001`, () => {
          return HttpResponse.json(envelope({ message: 'deleted' }));
        })
      );

      const { result } = renderHook(() => useDeleteVoucher(), {
        wrapper: createWrapper(),
      });

      await result.current.mutateAsync('vch-001');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });
    });

    it('should handle delete error for non-draft voucher', async () => {
      server.use(
        http.delete(`${API_BASE}/vouchers/vch-posted`, () => {
          return HttpResponse.json(
            errorEnvelope('BIZ_001', 'Only draft vouchers can be deleted'),
            { status: 400 }
          );
        })
      );

      const { result } = renderHook(() => useDeleteVoucher(), {
        wrapper: createWrapper(),
      });

      await expect(result.current.mutateAsync('vch-posted')).rejects.toThrow();

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  // ==========================================================================
  // useApproveVoucher Tests
  // ==========================================================================

  describe('useApproveVoucher', () => {
    it('should approve voucher successfully', async () => {
      server.use(
        http.post(`${API_BASE}/vouchers/vch-001/approve`, () => {
          return HttpResponse.json(
            envelope(createMockVoucher({ id: 'vch-001', status: 'approved' }))
          );
        })
      );

      const { result } = renderHook(() => useApproveVoucher(), {
        wrapper: createWrapper(),
      });

      await result.current.mutateAsync('vch-001');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data?.status).toBe('approved');
    });

    it('should handle approval error', async () => {
      server.use(
        http.post(`${API_BASE}/vouchers/vch-draft/approve`, () => {
          return HttpResponse.json(
            errorEnvelope('BIZ_002', 'Voucher is not in pending status'),
            { status: 400 }
          );
        })
      );

      const { result } = renderHook(() => useApproveVoucher(), {
        wrapper: createWrapper(),
      });

      await expect(result.current.mutateAsync('vch-draft')).rejects.toThrow();

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  // ==========================================================================
  // Query Keys Tests
  // ==========================================================================

  describe('voucherKeys', () => {
    it('should generate correct query keys', () => {
      expect(voucherKeys.all).toEqual(['vouchers']);
      expect(voucherKeys.lists()).toEqual(['vouchers', 'list']);
      expect(voucherKeys.list({ page: 1, pageSize: 10 })).toEqual([
        'vouchers',
        'list',
        { page: 1, pageSize: 10 },
      ]);
      expect(voucherKeys.details()).toEqual(['vouchers', 'detail']);
      expect(voucherKeys.detail('vch-001')).toEqual(['vouchers', 'detail', 'vch-001']);
    });
  });
});
