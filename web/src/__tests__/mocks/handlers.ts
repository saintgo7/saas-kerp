import { http, HttpResponse } from 'msw';

const API_BASE = '/api/v1';

// Sample data
const sampleVouchers = [
  {
    id: 'vch-001',
    voucherNo: 'GJ-2026-000001',
    voucherDate: '2026-01-15',
    voucherType: 'general',
    status: 'draft',
    description: 'Test voucher 1',
    totalDebit: 100000,
    totalCredit: 100000,
    entries: [
      { id: 'ent-001', lineNo: 1, accountId: 'acc-001', accountCode: '101', accountName: 'Cash', debitAmount: 100000, creditAmount: 0 },
      { id: 'ent-002', lineNo: 2, accountId: 'acc-002', accountCode: '401', accountName: 'Sales Revenue', debitAmount: 0, creditAmount: 100000 },
    ],
  },
  {
    id: 'vch-002',
    voucherNo: 'GJ-2026-000002',
    voucherDate: '2026-01-16',
    voucherType: 'general',
    status: 'pending',
    description: 'Test voucher 2',
    totalDebit: 50000,
    totalCredit: 50000,
    entries: [],
  },
];

const sampleAccounts = [
  { id: 'acc-001', code: '101', name: 'Cash', accountType: 'asset', accountNature: 'debit', isActive: true },
  { id: 'acc-002', code: '102', name: 'Bank', accountType: 'asset', accountNature: 'debit', isActive: true },
  { id: 'acc-003', code: '201', name: 'Accounts Payable', accountType: 'liability', accountNature: 'credit', isActive: true },
  { id: 'acc-004', code: '401', name: 'Sales Revenue', accountType: 'revenue', accountNature: 'credit', isActive: true },
  { id: 'acc-005', code: '501', name: 'COGS', accountType: 'expense', accountNature: 'debit', isActive: true },
];

const sampleUser = {
  id: 'usr-001',
  email: 'test@example.com',
  name: 'Test User',
  role: 'admin',
  companyId: 'comp-001',
  companyName: 'Test Company',
};

export const handlers = [
  // Auth handlers
  http.post(`${API_BASE}/auth/login`, async ({ request }) => {
    const body = await request.json() as { email: string; password: string };

    if (body.email === 'test@example.com' && body.password === 'Test123!@#') {
      return HttpResponse.json({
        success: true,
        data: {
          user: sampleUser,
          accessToken: 'test-access-token',
          refreshToken: 'test-refresh-token',
        },
      });
    }

    return HttpResponse.json(
      { success: false, error: { code: 'AUTH_FAILED', message: 'Invalid credentials' } },
      { status: 401 }
    );
  }),

  http.post(`${API_BASE}/auth/refresh`, () => {
    return HttpResponse.json({
      success: true,
      data: {
        accessToken: 'new-access-token',
        refreshToken: 'new-refresh-token',
      },
    });
  }),

  http.post(`${API_BASE}/auth/logout`, () => {
    return HttpResponse.json({ success: true });
  }),

  http.get(`${API_BASE}/auth/me`, () => {
    return HttpResponse.json({
      success: true,
      data: sampleUser,
    });
  }),

  // Voucher handlers
  http.get(`${API_BASE}/vouchers`, ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const page = parseInt(url.searchParams.get('page') || '1');
    const limit = parseInt(url.searchParams.get('limit') || '20');

    let filtered = [...sampleVouchers];
    if (status) {
      filtered = filtered.filter((v) => v.status === status);
    }

    return HttpResponse.json({
      success: true,
      data: filtered,
      meta: {
        total: filtered.length,
        page,
        limit,
        totalPages: Math.ceil(filtered.length / limit),
      },
    });
  }),

  http.get(`${API_BASE}/vouchers/:id`, ({ params }) => {
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(
        { success: false, error: { code: 'NOT_FOUND', message: 'Voucher not found' } },
        { status: 404 }
      );
    }

    return HttpResponse.json({ success: true, data: voucher });
  }),

  http.post(`${API_BASE}/vouchers`, async ({ request }) => {
    const body = await request.json() as {
      entries?: Array<{ debitAmount?: number; creditAmount?: number }>;
      [key: string]: unknown;
    };

    // Validate balance
    const totalDebit = body.entries?.reduce((sum: number, e) => sum + (e.debitAmount || 0), 0) || 0;
    const totalCredit = body.entries?.reduce((sum: number, e) => sum + (e.creditAmount || 0), 0) || 0;

    if (totalDebit !== totalCredit) {
      return HttpResponse.json(
        {
          success: false,
          error: {
            code: 'VALIDATION_ERROR',
            message: 'Debit and credit must be equal',
            details: { totalDebit, totalCredit },
          },
        },
        { status: 400 }
      );
    }

    const newVoucher = {
      id: `vch-${Date.now()}`,
      voucherNo: `GJ-2026-${String(sampleVouchers.length + 1).padStart(6, '0')}`,
      ...body,
      status: 'draft',
      totalDebit,
      totalCredit,
      createdAt: new Date().toISOString(),
    };

    return HttpResponse.json({ success: true, data: newVoucher }, { status: 201 });
  }),

  http.put(`${API_BASE}/vouchers/:id`, async ({ params, request }) => {
    const body = await request.json() as Record<string, unknown>;
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(
        { success: false, error: { code: 'NOT_FOUND', message: 'Voucher not found' } },
        { status: 404 }
      );
    }

    return HttpResponse.json({
      success: true,
      data: { ...voucher, ...body, updatedAt: new Date().toISOString() },
    });
  }),

  http.delete(`${API_BASE}/vouchers/:id`, ({ params }) => {
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(
        { success: false, error: { code: 'NOT_FOUND', message: 'Voucher not found' } },
        { status: 404 }
      );
    }

    if (voucher.status !== 'draft') {
      return HttpResponse.json(
        { success: false, error: { code: 'INVALID_STATUS', message: 'Only draft vouchers can be deleted' } },
        { status: 400 }
      );
    }

    return HttpResponse.json({ success: true });
  }),

  // Voucher workflow handlers
  http.post(`${API_BASE}/vouchers/:id/submit`, ({ params }) => {
    const voucher = sampleVouchers.find((v) => v.id === params.id);

    if (!voucher) {
      return HttpResponse.json(
        { success: false, error: { code: 'NOT_FOUND', message: 'Voucher not found' } },
        { status: 404 }
      );
    }

    if (voucher.status !== 'draft') {
      return HttpResponse.json(
        { success: false, error: { code: 'INVALID_STATUS', message: 'Only draft vouchers can be submitted' } },
        { status: 400 }
      );
    }

    return HttpResponse.json({
      success: true,
      data: { ...voucher, status: 'pending', submittedAt: new Date().toISOString() },
    });
  }),

  http.post(`${API_BASE}/vouchers/:id/approve`, ({ params }) => {
    return HttpResponse.json({
      success: true,
      data: { id: params.id, status: 'approved', approvedAt: new Date().toISOString() },
    });
  }),

  http.post(`${API_BASE}/vouchers/:id/reject`, async ({ params, request }) => {
    const body = await request.json() as { reason?: string };
    return HttpResponse.json({
      success: true,
      data: {
        id: params.id,
        status: 'rejected',
        rejectedAt: new Date().toISOString(),
        rejectionReason: body.reason || 'No reason provided',
      },
    });
  }),

  http.post(`${API_BASE}/vouchers/:id/post`, ({ params }) => {
    return HttpResponse.json({
      success: true,
      data: { id: params.id, status: 'posted', postedAt: new Date().toISOString() },
    });
  }),

  // Account handlers
  http.get(`${API_BASE}/accounts`, () => {
    return HttpResponse.json({
      success: true,
      data: sampleAccounts,
      meta: { total: sampleAccounts.length },
    });
  }),

  http.get(`${API_BASE}/accounts/:id`, ({ params }) => {
    const account = sampleAccounts.find((a) => a.id === params.id);

    if (!account) {
      return HttpResponse.json(
        { success: false, error: { code: 'NOT_FOUND', message: 'Account not found' } },
        { status: 404 }
      );
    }

    return HttpResponse.json({ success: true, data: account });
  }),
];
