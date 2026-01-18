import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { useAuthStore } from "@/stores";

// Layouts
import { MainLayout, AuthLayout } from "@/components/layout";

// Auth Pages
import { LoginPage, RegisterPage, ForgotPasswordPage } from "@/pages/auth";

// Main Pages
import { DashboardPage } from "@/pages/dashboard";
import {
  VoucherListPage,
  VoucherFormPage,
  VoucherDetailPage,
  AccountListPage,
  AccountFormPage,
} from "@/pages/accounting";
import {
  GeneralLedgerPage,
  SubsidiaryLedgerPage,
  TrialBalancePage,
} from "@/pages/ledger";
import {
  FinancialStatementsPage,
  BalanceSheetPage,
  IncomeStatementPage,
  SalesReportPage,
  ExpenseReportPage,
  HRReportPage,
  CustomReportPage,
} from "@/pages/reports";
import {
  InvoiceListPage,
  InvoiceIssuePage,
  InvoiceReceivedPage,
  HometaxSyncPage,
} from "@/pages/invoice";
import {
  EmployeeListPage,
  EmployeeFormPage,
  DepartmentPage,
  PayrollPage,
  InsurancePage,
  AttendancePage,
} from "@/pages/hr";
import { PartnerListPage, PartnerFormPage } from "@/pages/partner";
import {
  ProductListPage,
  ProductFormPage,
  StockStatusPage,
  PurchaseOrderPage,
  SalesOrderPage,
} from "@/pages/inventory";
import {
  CompanySettingsPage,
  UserManagementPage,
  PermissionPage,
  IntegrationPage,
  ProfilePage,
} from "@/pages/settings";

// Create QueryClient
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

// Protected Route Component
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

// Public Route Component (redirect to dashboard if authenticated)
function PublicRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore();

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}

// Placeholder component for routes not yet implemented
function ComingSoon({ title }: { title: string }) {
  return (
    <div className="flex flex-col items-center justify-center h-[60vh] text-center">
      <h1 className="text-2xl font-bold mb-2">{title}</h1>
      <p className="text-muted-foreground">이 페이지는 준비 중입니다.</p>
    </div>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Public Routes */}
          <Route
            element={
              <PublicRoute>
                <AuthLayout />
              </PublicRoute>
            }
          >
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          </Route>

          {/* Protected Routes */}
          <Route
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            {/* Dashboard */}
            <Route path="/dashboard" element={<DashboardPage />} />

            {/* Accounting - Voucher */}
            <Route path="/accounting/voucher" element={<VoucherListPage />} />
            <Route path="/accounting/voucher/new" element={<VoucherFormPage />} />
            <Route path="/accounting/voucher/:id" element={<VoucherDetailPage />} />
            <Route path="/accounting/voucher/:id/edit" element={<VoucherFormPage />} />

            {/* Accounting - Ledger */}
            <Route path="/accounting/ledger" element={<GeneralLedgerPage />} />
            <Route path="/accounting/ledger/general" element={<GeneralLedgerPage />} />
            <Route path="/accounting/ledger/subsidiary" element={<SubsidiaryLedgerPage />} />

            {/* Accounting - Trial Balance */}
            <Route path="/accounting/trial-balance" element={<TrialBalancePage />} />

            {/* Accounting - Financial Statements */}
            <Route path="/accounting/financial-statements" element={<FinancialStatementsPage />} />
            <Route path="/accounting/financial-statements/balance-sheet" element={<BalanceSheetPage />} />
            <Route path="/accounting/financial-statements/income-statement" element={<IncomeStatementPage />} />

            {/* Accounting - Accounts */}
            <Route path="/accounting/accounts" element={<AccountListPage />} />
            <Route path="/accounting/accounts/new" element={<AccountFormPage />} />
            <Route path="/accounting/accounts/:id" element={<AccountFormPage />} />

            {/* Invoice */}
            <Route path="/invoice/list" element={<InvoiceListPage />} />
            <Route path="/invoice/issue" element={<InvoiceIssuePage />} />
            <Route path="/invoice/received" element={<InvoiceReceivedPage />} />
            <Route path="/invoice/hometax" element={<HometaxSyncPage />} />
            <Route path="/invoice/:id" element={<InvoiceIssuePage />} />

            {/* Partners */}
            <Route path="/partners" element={<PartnerListPage />} />
            <Route path="/partners/new" element={<PartnerFormPage />} />
            <Route path="/partners/:id" element={<PartnerFormPage />} />

            {/* HR */}
            <Route path="/hr/employee" element={<EmployeeListPage />} />
            <Route path="/hr/employee/new" element={<EmployeeFormPage />} />
            <Route path="/hr/employee/:id" element={<EmployeeFormPage />} />
            <Route path="/hr/department" element={<DepartmentPage />} />
            <Route path="/hr/payroll" element={<PayrollPage />} />
            <Route path="/hr/insurance" element={<InsurancePage />} />
            <Route path="/hr/attendance" element={<AttendancePage />} />

            {/* Inventory */}
            <Route path="/inventory/products" element={<ProductListPage />} />
            <Route path="/inventory/products/new" element={<ProductFormPage />} />
            <Route path="/inventory/products/:id" element={<ProductFormPage />} />
            <Route path="/inventory/stock" element={<StockStatusPage />} />
            <Route path="/inventory/purchase" element={<PurchaseOrderPage />} />
            <Route path="/inventory/sales" element={<SalesOrderPage />} />

            {/* Reports */}
            <Route path="/reports/sales" element={<SalesReportPage />} />
            <Route path="/reports/expense" element={<ExpenseReportPage />} />
            <Route path="/reports/hr" element={<HRReportPage />} />
            <Route path="/reports/custom" element={<CustomReportPage />} />

            {/* Settings */}
            <Route path="/settings" element={<Navigate to="/settings/company" replace />} />
            <Route path="/settings/company" element={<CompanySettingsPage />} />
            <Route path="/settings/users" element={<UserManagementPage />} />
            <Route path="/settings/permissions" element={<PermissionPage />} />
            <Route path="/settings/integrations" element={<IntegrationPage />} />
            <Route path="/settings/profile" element={<ProfilePage />} />
          </Route>

          {/* Redirect root to dashboard */}
          <Route path="/" element={<Navigate to="/dashboard" replace />} />

          {/* 404 */}
          <Route path="*" element={<ComingSoon title="페이지를 찾을 수 없습니다" />} />
        </Routes>
      </BrowserRouter>
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}

export default App;
