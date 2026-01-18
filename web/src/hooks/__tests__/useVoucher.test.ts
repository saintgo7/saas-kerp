import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { server } from '@/__tests__/mocks/server';
import { createWrapper, createMockVoucher } from '@/__tests__/test-utils';
import {
  useVouchers,
  useVoucher,
  useCreateVoucher,
  useUpdateVoucher,
  useDeleteVoucher,
  useApproveVoucher,
  voucherKeys,
} from '../useVoucher';

const API_BASE = '/api';

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
          return HttpResponse.json({
            data: mockVouchers,
            total: 2,
            page: 1,
            limit: 20,
          });
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
          return HttpResponse.json({
            data: [createMockVoucher({ status: 'draft' })],
            total: 1,
            page: 1,
            limit: 10,
          });
        })
      );

      const { result } = renderHook(
        () => useVouchers({ status: 'draft', page: 1, limit: 10 }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(capturedParams).toBeDefined();
      expect(capturedParams!.get('status')).toBe('draft');
      expect(capturedParams!.get('page')).toBe('1');
      expect(capturedParams!.get('limit')).toBe('10');
    });

    it('should handle fetch error', async () => {
      server.use(
        http.get(`${API_BASE}/vouchers`, () => {
          return HttpResponse.json(
            { error: { message: 'Server error' } },
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
          return HttpResponse.json(mockVoucher);
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
            { error: { code: 'NOT_FOUND', message: 'Voucher not found' } },
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
        description: 'New test voucher',
        entries: [
          { accountCode: '101', debitAmount: 10000, creditAmount: 0 },
          { accountCode: '201', debitAmount: 0, creditAmount: 10000 },
        ],
      };

      server.use(
        http.post(`${API_BASE}/vouchers`, async ({ request }) => {
          const body = await request.json() as Record<string, unknown>;
          return HttpResponse.json(
            {
              id: 'vch-new-001',
              voucherNo: 'GJ-2026-000001',
              ...body,
              status: 'draft',
              totalDebit: 10000,
              totalCredit: 10000,
            },
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
    });

    it('should handle validation error for unbalanced voucher', async () => {
      const unbalancedVoucher = {
        voucherDate: '2026-01-15',
        description: 'Unbalanced voucher',
        entries: [
          { accountCode: '101', debitAmount: 10000, creditAmount: 0 },
          { accountCode: '201', debitAmount: 0, creditAmount: 5000 }, // Not balanced
        ],
      };

      server.use(
        http.post(`${API_BASE}/vouchers`, () => {
          return HttpResponse.json(
            {
              error: {
                code: 'VALIDATION_ERROR',
                message: 'Debit and credit must be equal',
              },
            },
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
        description: 'Updated description',
        entries: [
          { accountCode: '101', debitAmount: 20000, creditAmount: 0 },
          { accountCode: '201', debitAmount: 0, creditAmount: 20000 },
        ],
      };

      server.use(
        http.put(`${API_BASE}/vouchers/vch-001`, async ({ request }) => {
          const body = await request.json() as Record<string, unknown>;
          return HttpResponse.json({
            id: 'vch-001',
            ...body,
            updatedAt: new Date().toISOString(),
          });
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
    });

    it('should handle not found error', async () => {
      server.use(
        http.put(`${API_BASE}/vouchers/nonexistent`, () => {
          return HttpResponse.json(
            { error: { code: 'NOT_FOUND', message: 'Voucher not found' } },
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
          return HttpResponse.json({ success: true });
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
            {
              error: {
                code: 'INVALID_STATUS',
                message: 'Only draft vouchers can be deleted',
              },
            },
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
          return HttpResponse.json({
            id: 'vch-001',
            status: 'approved',
            approvedAt: new Date().toISOString(),
          });
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
            {
              error: {
                code: 'INVALID_STATUS',
                message: 'Voucher is not in pending status',
              },
            },
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
      expect(voucherKeys.list({ page: 1, limit: 10 })).toEqual([
        'vouchers',
        'list',
        { page: 1, limit: 10 },
      ]);
      expect(voucherKeys.details()).toEqual(['vouchers', 'detail']);
      expect(voucherKeys.detail('vch-001')).toEqual(['vouchers', 'detail', 'vch-001']);
    });
  });
});
