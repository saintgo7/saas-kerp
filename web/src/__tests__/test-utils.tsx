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

// Test data factories
export const createMockVoucher = (overrides = {}) => ({
  id: `vch-${Date.now()}`,
  voucherNo: 'GJ-2026-000001',
  voucherDate: '2026-01-15',
  voucherType: 'general',
  status: 'draft',
  description: 'Test voucher',
  totalDebit: 100000,
  totalCredit: 100000,
  entries: [
    { id: 'ent-1', lineNo: 1, accountId: 'acc-1', debitAmount: 100000, creditAmount: 0 },
    { id: 'ent-2', lineNo: 2, accountId: 'acc-2', debitAmount: 0, creditAmount: 100000 },
  ],
  ...overrides,
});

export const createMockAccount = (overrides = {}) => ({
  id: `acc-${Date.now()}`,
  code: '101',
  name: 'Test Account',
  accountType: 'asset',
  accountNature: 'debit',
  isActive: true,
  allowDirectPosting: true,
  ...overrides,
});

export const createMockUser = (overrides = {}) => ({
  id: `usr-${Date.now()}`,
  email: 'test@example.com',
  name: 'Test User',
  role: 'admin' as const,
  companyId: 'comp-001',
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
  ...overrides,
});
