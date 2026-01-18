import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { useAuthStore } from "@/stores";

// Layouts
import { MainLayout, AuthLayout } from "@/components/layout";

// Auth Pages
import { LoginPage, RegisterPage } from "@/pages/auth";

// Main Pages
import { DashboardPage } from "@/pages/dashboard";
import { VoucherListPage, VoucherFormPage, AccountListPage, AccountFormPage } from "@/pages/accounting";
import { InvoiceListPage, InvoiceIssuePage } from "@/pages/invoice";
import { EmployeeListPage, EmployeeFormPage } from "@/pages/hr";
import { PartnerListPage, PartnerFormPage } from "@/pages/partner";

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
            <Route path="/forgot-password" element={<ComingSoon title="비밀번호 찾기" />} />
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

            {/* Accounting */}
            <Route path="/accounting/voucher" element={<VoucherListPage />} />
            <Route path="/accounting/voucher/new" element={<VoucherFormPage />} />
            <Route path="/accounting/voucher/:id" element={<VoucherFormPage />} />
            <Route path="/accounting/ledger" element={<ComingSoon title="원장 조회" />} />
            <Route path="/accounting/trial-balance" element={<ComingSoon title="시산표" />} />
            <Route path="/accounting/financial-statements" element={<ComingSoon title="재무제표" />} />
            <Route path="/accounting/accounts" element={<AccountListPage />} />
            <Route path="/accounting/accounts/new" element={<AccountFormPage />} />
            <Route path="/accounting/accounts/:id" element={<AccountFormPage />} />

            {/* Invoice */}
            <Route path="/invoice/list" element={<InvoiceListPage />} />
            <Route path="/invoice/issue" element={<InvoiceIssuePage />} />
            <Route path="/invoice/received" element={<ComingSoon title="매입 관리" />} />
            <Route path="/invoice/hometax" element={<ComingSoon title="홈택스 연동" />} />
            <Route path="/invoice/:id" element={<InvoiceIssuePage />} />

            {/* Partners */}
            <Route path="/partners" element={<PartnerListPage />} />
            <Route path="/partners/new" element={<PartnerFormPage />} />
            <Route path="/partners/:id" element={<PartnerFormPage />} />

            {/* HR */}
            <Route path="/hr/employee" element={<EmployeeListPage />} />
            <Route path="/hr/employee/new" element={<EmployeeFormPage />} />
            <Route path="/hr/employee/:id" element={<EmployeeFormPage />} />
            <Route path="/hr/department" element={<ComingSoon title="부서 관리" />} />
            <Route path="/hr/payroll" element={<ComingSoon title="급여 관리" />} />
            <Route path="/hr/insurance" element={<ComingSoon title="4대보험" />} />
            <Route path="/hr/attendance" element={<ComingSoon title="근태 관리" />} />

            {/* Inventory */}
            <Route path="/inventory/products" element={<ComingSoon title="품목 관리" />} />
            <Route path="/inventory/stock" element={<ComingSoon title="재고 현황" />} />
            <Route path="/inventory/purchase" element={<ComingSoon title="구매 관리" />} />
            <Route path="/inventory/sales" element={<ComingSoon title="판매 관리" />} />

            {/* Reports */}
            <Route path="/reports/sales" element={<ComingSoon title="매출 분석" />} />
            <Route path="/reports/expense" element={<ComingSoon title="비용 분석" />} />
            <Route path="/reports/hr" element={<ComingSoon title="인사 현황" />} />
            <Route path="/reports/custom" element={<ComingSoon title="맞춤 보고서" />} />

            {/* Settings */}
            <Route path="/settings" element={<ComingSoon title="설정" />} />
            <Route path="/settings/company" element={<ComingSoon title="회사 정보" />} />
            <Route path="/settings/users" element={<ComingSoon title="사용자 관리" />} />
            <Route path="/settings/permissions" element={<ComingSoon title="권한 관리" />} />
            <Route path="/settings/integrations" element={<ComingSoon title="연동 설정" />} />
            <Route path="/settings/profile" element={<ComingSoon title="내 프로필" />} />
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
