import { http, HttpResponse } from 'msw';

// Mirrors constants/index.ts API_BASE_URL and the Gin router (/api/v1).
const API_BASE = '/api/v1';

/** Success envelope produced by internal/dto/common.go Response. */
const ok = (data: unknown, meta?: Record<string, unknown>) => ({
  success: true,
  data,
  ...(meta ? { meta } : {}),
});

const fail = (code: string, message: string) => ({
  success: false,
  error: { code, message },
});

// Sample data
// internal/dto/voucher_dto.go VoucherResponse
const sampleVouchers = [
  {
    id: 'vch-001',
    voucher_no: 'GJ-2026-000001',
    voucher_date: '2026-01-15',
    voucher_type: 'general',
    voucher_type_label: '일반전표',
    status: 'draft',
    status_label: '작성중',
    description: 'Test voucher 1',
    total_debit: 100000,
    total_credit: 100000,
    entries: [
      { id: 'ent-001', line_no: 1, account_id: 'acc-001', account_code: '101', account_name: 'Cash', debit_amount: 100000, credit_amount: 0 },
      { id: 'ent-002', line_no: 2, account_id: 'acc-004', account_code: '401', account_name: 'Sales Revenue', debit_amount: 0, credit_amount: 100000 },
    ],
    created_at: '2026-01-15T00:00:00Z',
    updated_at: '2026-01-15T00:00:00Z',
  },
  {
    id: 'vch-002',
    voucher_no: 'GJ-2026-000002',
    voucher_date: '2026-01-16',
    voucher_type: 'general',
    voucher_type_label: '일반전표',
    status: 'pending',
    status_label: '승인대기',
    description: 'Test voucher 2',
    total_debit: 50000,
    total_credit: 50000,
    entries: [],
    created_at: '2026-01-16T00:00:00Z',
    updated_at: '2026-01-16T00:00:00Z',
  },
];

// internal/dto/account_dto.go AccountResponse
const account = (
  id: string,
  code: string,
  name: string,
  accountType: string,
  nature: string
) => ({
  id,
  code,
  name,
  level: 1,
  account_type: accountType,
  account_nature: nature,
  is_active: true,
  is_control_account: false,
  allow_direct_posting: true,
  sort_order: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
});

const sampleAccounts = [
  account('acc-001', '101', 'Cash', 'asset', 'debit'),
  account('acc-002', '102', 'Bank', 'asset', 'debit'),
  account('acc-003', '201', 'Accounts Payable', 'liability', 'credit'),
  account('acc-004', '401', 'Sales Revenue', 'revenue', 'credit'),
  account('acc-005', '501', 'COGS', 'expense', 'debit'),
];

// internal/service/auth_service.go UserResponse
const sampleUser = {
  id: 'usr-001',
  company_id: 'comp-001',
  email: 'test@example.com',
  name: 'Test User',
  role: 'admin',
  status: 'active',
};

export const handlers = [
  // Auth handlers
  http.post(`${API_BASE}/auth/login`, async ({ request }) => {
    const body = await request.json() as { email: string; password: string };

    if (body.email === 'test@example.com' && body.password === 'Test123!@#') {
      return HttpResponse.json(
        ok({
          access_token: 'test-access-token',
          refresh_token: 'test-refresh-token',
          token_type: 'Bearer',
          expires_in: 3600,
          user: sampleUser,
        })
      );
    }

    return HttpResponse.json(fail('AUTH_001', 'Invalid email or password'), {
      status: 401,
    });
  }),

  http.post(`${API_BASE}/auth/refresh`, () => {
    return HttpResponse.json({
      success: true,
      data: {
        access_token: 'new-access-token',
        refresh_token: 'new-refresh-token',
        token_type: 'Bearer',
        expires_in: 3600,
      },
    });
  }),

  http.post(`${API_BASE}/auth/logout`, () => {
    return HttpResponse.json(ok({ message: 'Logged out successfully' }));
  }),

  // internal/handler/auth.go Me returns JWT claims, not a user row
  http.get(`${API_BASE}/auth/me`, () => {
    return HttpResponse.json(
      ok({
        user_id: sampleUser.id,
        company_id: sampleUser.company_id,
        email: sampleUser.email,
        name: sampleUser.name,
        roles: [sampleUser.role],
      })
    );
  }),

  // Voucher handlers
  http.get(`${API_BASE}/vouchers`, ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const page = parseInt(url.searchParams.get('page') || '1');
    const perPage = parseInt(url.searchParams.get('page_size') || '20');

    let filtered = [...sampleVouchers];
    if (status) {
      filtered = filtered.filter((v) => v.status === status);
    }

    return HttpResponse.json(
      ok(filtered, {
        pagination: {
          page,
          per_page: perPage,
          total: filtered.length,
          total_pages: Math.max(1, Math.ceil(filtered.length / perPage)),
        },
      })
    );
  }),

  http.get(`${API_BASE}/vouchers/:id`, ({ params }) => {
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(fail('RES_001', 'Voucher not found'), {
        status: 404,
      });
    }

    return HttpResponse.json(ok(voucher));
  }),

  http.post(`${API_BASE}/vouchers`, async ({ request }) => {
    const body = (await request.json()) as {
      entries?: Array<{ debit_amount?: number; credit_amount?: number }>;
      voucher_type?: string;
      [key: string]: unknown;
    };

    // The backend requires voucher_type (oneof=general sales purchase ...)
    if (!body.voucher_type) {
      return HttpResponse.json(fail('VAL_001', 'voucher_type is required'), {
        status: 400,
      });
    }

    const totalDebit =
      body.entries?.reduce((sum, e) => sum + (e.debit_amount || 0), 0) || 0;
    const totalCredit =
      body.entries?.reduce((sum, e) => sum + (e.credit_amount || 0), 0) || 0;

    if (Math.abs(totalDebit - totalCredit) > 0.005) {
      return HttpResponse.json(
        fail('VAL_001', 'Debit and credit must be equal'),
        { status: 400 }
      );
    }

    return HttpResponse.json(
      ok({
        ...sampleVouchers[0],
        id: `vch-${Date.now()}`,
        voucher_no: `GJ-2026-${String(sampleVouchers.length + 1).padStart(6, '0')}`,
        voucher_date: body.voucher_date ?? '2026-01-15',
        voucher_type: body.voucher_type,
        description: body.description ?? '',
        status: 'draft',
        total_debit: totalDebit,
        total_credit: totalCredit,
      }),
      { status: 201 }
    );
  }),

  http.put(`${API_BASE}/vouchers/:id`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(fail('RES_001', 'Voucher not found'), {
        status: 404,
      });
    }

    return HttpResponse.json(
      ok({
        ...voucher,
        voucher_date: (body.voucher_date as string) ?? voucher.voucher_date,
        description: (body.description as string) ?? voucher.description,
        updated_at: new Date().toISOString(),
      })
    );
  }),

  http.delete(`${API_BASE}/vouchers/:id`, ({ params }) => {
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(fail('RES_001', 'Voucher not found'), {
        status: 404,
      });
    }

    if (voucher.status !== 'draft') {
      return HttpResponse.json(
        fail('BIZ_001', 'Only draft vouchers can be deleted'),
        { status: 400 }
      );
    }

    return HttpResponse.json(ok({ message: 'Voucher deleted successfully' }));
  }),

  // Voucher workflow handlers
  http.post(`${API_BASE}/vouchers/:id/submit`, ({ params }) => {
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(fail('RES_001', 'Voucher not found'), {
        status: 404,
      });
    }

    if (voucher.status !== 'draft') {
      return HttpResponse.json(
        fail('BIZ_001', 'Only draft vouchers can be submitted'),
        { status: 400 }
      );
    }

    return HttpResponse.json(
      ok({ ...voucher, status: 'pending', submitted_at: new Date().toISOString() })
    );
  }),

  http.post(`${API_BASE}/vouchers/:id/approve`, ({ params }) => {
    return HttpResponse.json(
      ok({
        ...sampleVouchers[0],
        id: params.id,
        status: 'approved',
        approved_at: new Date().toISOString(),
      })
    );
  }),

  http.post(`${API_BASE}/vouchers/:id/reject`, async ({ params, request }) => {
    const body = (await request.json()) as { reason?: string };
    return HttpResponse.json(
      ok({
        ...sampleVouchers[0],
        id: params.id,
        status: 'rejected',
        rejected_at: new Date().toISOString(),
        rejection_reason: body.reason || 'No reason provided',
      })
    );
  }),

  http.post(`${API_BASE}/vouchers/:id/post`, ({ params }) => {
    return HttpResponse.json(
      ok({
        ...sampleVouchers[0],
        id: params.id,
        status: 'posted',
        posted_at: new Date().toISOString(),
      })
    );
  }),

  // Account handlers
  http.get(`${API_BASE}/accounts`, ({ request }) => {
    const url = new URL(request.url);
    const search = url.searchParams.get('search')?.toLowerCase() ?? '';
    const filtered = search
      ? sampleAccounts.filter(
          (a) =>
            a.code.toLowerCase().includes(search) ||
            a.name.toLowerCase().includes(search)
        )
      : sampleAccounts;

    return HttpResponse.json(
      ok(filtered, {
        pagination: {
          page: 1,
          per_page: 100,
          total: filtered.length,
          total_pages: 1,
        },
      })
    );
  }),

  http.get(`${API_BASE}/accounts/tree`, () => {
    return HttpResponse.json(ok(sampleAccounts));
  }),

  http.get(`${API_BASE}/accounts/:id`, ({ params }) => {
    const found = sampleAccounts.find((a) => a.id === params.id);

    if (!found) {
      return HttpResponse.json(fail('RES_001', 'Account not found'), {
        status: 404,
      });
    }

    return HttpResponse.json(ok(found));
  }),
];
