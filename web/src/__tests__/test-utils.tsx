import React, { type ReactElement } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';

// Create a fresh QueryClient for each test
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

interface AllProvidersProps {
  children: React.ReactNode;
}

// Provider wrapper for tests
// eslint-disable-next-line react-refresh/only-export-components
const AllProviders = ({ children }: AllProvidersProps) => {
  const queryClient = createTestQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>{children}</BrowserRouter>
    </QueryClientProvider>
  );
};

// Custom render function with all providers
const customRender = (ui: ReactElement, options?: Omit<RenderOptions, 'wrapper'>) =>
  render(ui, { wrapper: AllProviders, ...options });

// Re-export everything from testing-library
// eslint-disable-next-line react-refresh/only-export-components
export * from '@testing-library/react';
export { customRender as render };

// Helper to create a wrapper with custom QueryClient
export const createWrapper = () => {
  const queryClient = createTestQueryClient();
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>{children}</BrowserRouter>
    </QueryClientProvider>
  );
};

// Helper to wait for async operations
export const waitForLoadingToFinish = () =>
  new Promise((resolve) => setTimeout(resolve, 0));

// Mock localStorage
export const mockLocalStorage = () => {
  const store: Record<string, string> = {};

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      Object.keys(store).forEach((key) => delete store[key]);
    },
  };
};

// ---------------------------------------------------------------------------
// Test data factories
//
// The *Wire factories mirror what the Go handlers actually serialize
// (snake_case, see internal/dto/*.go). Use them for HTTP fixtures. The plain
// factories produce the camelCase domain types the UI works with, for
// assertions on hook/store output.
// ---------------------------------------------------------------------------

/** internal/dto/voucher_dto.go VoucherResponse */
export const createMockVoucherWire = (overrides: Record<string, unknown> = {}) => ({
  id: `vch-${Date.now()}`,
  voucher_no: 'GJ-2026-000001',
  voucher_date: '2026-01-15',
  voucher_type: 'general',
  voucher_type_label: '일반전표',
  status: 'draft',
  status_label: '작성중',
  description: 'Test voucher',
  total_debit: 100000,
  total_credit: 100000,
  entries: [
    {
      id: 'ent-1',
      line_no: 1,
      account_id: 'acc-1',
      account_code: '101',
      account_name: 'Cash',
      debit_amount: 100000,
      credit_amount: 0,
    },
    {
      id: 'ent-2',
      line_no: 2,
      account_id: 'acc-2',
      account_code: '201',
      account_name: 'Accounts Payable',
      debit_amount: 0,
      credit_amount: 100000,
    },
  ],
  created_at: '2026-01-15T00:00:00Z',
  updated_at: '2026-01-15T00:00:00Z',
  ...overrides,
});

/** Backwards-compatible alias: the fixture is a wire payload. */
export const createMockVoucher = createMockVoucherWire;

/** internal/dto/account_dto.go AccountResponse */
export const createMockAccountWire = (overrides: Record<string, unknown> = {}) => ({
  id: `acc-${Date.now()}`,
  code: '101',
  name: 'Test Account',
  level: 1,
  account_type: 'asset',
  account_type_label: '자산',
  account_nature: 'debit',
  account_nature_label: '차변',
  is_active: true,
  is_control_account: false,
  allow_direct_posting: true,
  sort_order: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
});

export const createMockAccount = createMockAccountWire;

/** internal/service/auth_service.go UserResponse */
export const createMockUserWire = (overrides: Record<string, unknown> = {}) => ({
  id: 'usr-001',
  company_id: 'comp-001',
  email: 'test@example.com',
  name: 'Test User',
  role: 'admin',
  status: 'active',
  ...overrides,
});

/** The camelCase `User` the auth store holds after mapping. */
export const createMockUser = (overrides = {}) => ({
  id: 'usr-001',
  email: 'test@example.com',
  name: 'Test User',
  phone: undefined,
  role: 'admin' as const,
  companyId: 'comp-001',
  createdAt: '',
  updatedAt: '',
  ...overrides,
});

/** internal/service/auth_service.go LoginResult / RegisterResult */
export const createMockAuthPayload = (
  user: Record<string, unknown> = createMockUserWire(),
  overrides: Record<string, unknown> = {}
) => ({
  access_token: 'test-access-token',
  refresh_token: 'test-refresh-token',
  token_type: 'Bearer',
  expires_in: 3600,
  user,
  ...overrides,
});

/** Standard success envelope (internal/dto/common.go Response). */
export const envelope = <T,>(data: T, meta?: Record<string, unknown>) => ({
  success: true,
  data,
  ...(meta ? { meta } : {}),
});

/** Standard error envelope. */
export const errorEnvelope = (code: string, message: string) => ({
  success: false,
  error: { code, message },
});
