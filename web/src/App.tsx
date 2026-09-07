import { Suspense, lazy } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { useAuthStore } from "@/stores";
import { ErrorBoundary } from "@/components/common";
import { Toaster } from "@/components/ui";

// Layouts
import { MainLayout, AuthLayout } from "@/components/layout";

// Auth pages stay eager: they are the first screen an unauthenticated visitor
// sees, so there is nothing to gain by deferring them.
import { LoginPage, RegisterPage, ForgotPasswordPage } from "@/pages/auth";

/**
 * Everything behind the login is code-split. Previously a single 977 kB chunk
 * meant the login screen downloaded the HR, inventory and reporting modules
 * before the user had even typed a password.
 */
const DashboardPage = lazy(() =>
  import("@/pages/dashboard").then((m) => ({ default: m.DashboardPage }))
);

const VoucherListPage = lazy(() =>
  import("@/pages/accounting").then((m) => ({ default: m.VoucherListPage }))
);
const VoucherFormPage = lazy(() =>
  import("@/pages/accounting").then((m) => ({ default: m.VoucherFormPage }))
);
const VoucherDetailPage = lazy(() =>
  import("@/pages/accounting").then((m) => ({ default: m.VoucherDetailPage }))
);
const AccountListPage = lazy(() =>
  import("@/pages/accounting").then((m) => ({ default: m.AccountListPage }))
);
const AccountFormPage = lazy(() =>
  import("@/pages/accounting").then((m) => ({ default: m.AccountFormPage }))
);

const GeneralLedgerPage = lazy(() =>
  import("@/pages/ledger").then((m) => ({ default: m.GeneralLedgerPage }))
);
const SubsidiaryLedgerPage = lazy(() =>
  import("@/pages/ledger").then((m) => ({ default: m.SubsidiaryLedgerPage }))
);
const TrialBalancePage = lazy(() =>
  import("@/pages/ledger").then((m) => ({ default: m.TrialBalancePage }))
);

const FinancialStatementsPage = lazy(() =>
  import("@/pages/reports").then((m) => ({ default: m.FinancialStatementsPage }))
);
const BalanceSheetPage = lazy(() =>
  import("@/pages/reports").then((m) => ({ default: m.BalanceSheetPage }))
);
const IncomeStatementPage = lazy(() =>
  import("@/pages/reports").then((m) => ({ default: m.IncomeStatementPage }))
);
const SalesReportPage = lazy(() =>
  import("@/pages/reports").then((m) => ({ default: m.SalesReportPage }))
);
const ExpenseReportPage = lazy(() =>
  import("@/pages/reports").then((m) => ({ default: m.ExpenseReportPage }))
);
const HRReportPage = lazy(() =>
  import("@/pages/reports").then((m) => ({ default: m.HRReportPage }))
);
const CustomReportPage = lazy(() =>
  import("@/pages/reports").then((m) => ({ default: m.CustomReportPage }))
);

const InvoiceListPage = lazy(() =>
  import("@/pages/invoice").then((m) => ({ default: m.InvoiceListPage }))
);
const InvoiceIssuePage = lazy(() =>
  import("@/pages/invoice").then((m) => ({ default: m.InvoiceIssuePage }))
);
const InvoiceReceivedPage = lazy(() =>
  import("@/pages/invoice").then((m) => ({ default: m.InvoiceReceivedPage }))
);
const HometaxSyncPage = lazy(() =>
  import("@/pages/invoice").then((m) => ({ default: m.HometaxSyncPage }))
);

const EmployeeListPage = lazy(() =>
  import("@/pages/hr").then((m) => ({ default: m.EmployeeListPage }))
);
const EmployeeFormPage = lazy(() =>
  import("@/pages/hr").then((m) => ({ default: m.EmployeeFormPage }))
);
const DepartmentPage = lazy(() =>
  import("@/pages/hr").then((m) => ({ default: m.DepartmentPage }))
);
const PayrollPage = lazy(() =>
  import("@/pages/hr").then((m) => ({ default: m.PayrollPage }))
);
const InsurancePage = lazy(() =>
  import("@/pages/hr").then((m) => ({ default: m.InsurancePage }))
);
const AttendancePage = lazy(() =>
  import("@/pages/hr").then((m) => ({ default: m.AttendancePage }))
);

const PartnerListPage = lazy(() =>
  import("@/pages/partner").then((m) => ({ default: m.PartnerListPage }))
);
const PartnerFormPage = lazy(() =>
  import("@/pages/partner").then((m) => ({ default: m.PartnerFormPage }))
);

const ProductListPage = lazy(() =>
  import("@/pages/inventory").then((m) => ({ default: m.ProductListPage }))
);
const ProductFormPage = lazy(() =>
  import("@/pages/inventory").then((m) => ({ default: m.ProductFormPage }))
);
const StockStatusPage = lazy(() =>
  import("@/pages/inventory").then((m) => ({ default: m.StockStatusPage }))
);
const PurchaseOrderPage = lazy(() =>
  import("@/pages/inventory").then((m) => ({ default: m.PurchaseOrderPage }))
);
const SalesOrderPage = lazy(() =>
  import("@/pages/inventory").then((m) => ({ default: m.SalesOrderPage }))
);

const CompanySettingsPage = lazy(() =>
  import("@/pages/settings").then((m) => ({ default: m.CompanySettingsPage }))
);
const UserManagementPage = lazy(() =>
  import("@/pages/settings").then((m) => ({ default: m.UserManagementPage }))
);
const PermissionPage = lazy(() =>
  import("@/pages/settings").then((m) => ({ default: m.PermissionPage }))
);
const IntegrationPage = lazy(() =>
  import("@/pages/settings").then((m) => ({ default: m.IntegrationPage }))
);
const ProfilePage = lazy(() =>
  import("@/pages/settings").then((m) => ({ default: m.ProfilePage }))
);

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

// Shown while a route's chunk is being fetched.
function RouteFallback() {
  return (
    <div className="flex h-[60vh] items-center justify-center text-muted-foreground">
      불러오는 중...
    </div>
  );
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
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Suspense fallback={<RouteFallback />}>
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
          </Suspense>
        </BrowserRouter>
        {/* Renders the toast queue; without it every toast.* call is silent. */}
        <Toaster />
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;
