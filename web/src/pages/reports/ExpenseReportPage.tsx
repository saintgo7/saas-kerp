import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Download,
  Printer,
  TrendingUp,
  Minus,
  PieChart,
  Building2,
  Wallet,
  Target,
  AlertTriangle,
  CheckCircle2,
  ArrowUpRight,
  ArrowDownRight,
} from "lucide-react";
import {
  Button,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Select,
} from "@/components/ui";
import { DateRangePicker } from "@/components/common";
import { formatCurrency, cn } from "@/lib/utils";
import { reportsApi } from "@/api/reports";
import type { ExpenseReportData, ExpenseTrendItem } from "@/api/reports";

// Mock data for development
const mockExpenseReportData: ExpenseReportData = {
  period: {
    startDate: "2024-01-01",
    endDate: "2024-01-31",
  },
  summary: {
    totalExpense: 45000000,
    previousTotalExpense: 42000000,
    totalBudget: 50000000,
    budgetUtilization: 90,
    varianceFromBudget: -5000000,
  },
  trend: [
    { date: "2024-01-01", amount: 1200000, budgetAmount: 1600000 },
    { date: "2024-01-02", amount: 1500000, budgetAmount: 1600000 },
    { date: "2024-01-03", amount: 1350000, budgetAmount: 1600000 },
    { date: "2024-01-04", amount: 1800000, budgetAmount: 1600000 },
    { date: "2024-01-05", amount: 1650000, budgetAmount: 1600000 },
    { date: "2024-01-06", amount: 800000, budgetAmount: 1600000 },
    { date: "2024-01-07", amount: 600000, budgetAmount: 1600000 },
    { date: "2024-01-08", amount: 1550000, budgetAmount: 1600000 },
    { date: "2024-01-09", amount: 1700000, budgetAmount: 1600000 },
    { date: "2024-01-10", amount: 1450000, budgetAmount: 1600000 },
    { date: "2024-01-11", amount: 1600000, budgetAmount: 1600000 },
    { date: "2024-01-12", amount: 1400000, budgetAmount: 1600000 },
    { date: "2024-01-13", amount: 750000, budgetAmount: 1600000 },
    { date: "2024-01-14", amount: 650000, budgetAmount: 1600000 },
    { date: "2024-01-15", amount: 1500000, budgetAmount: 1600000 },
    { date: "2024-01-16", amount: 1850000, budgetAmount: 1600000 },
    { date: "2024-01-17", amount: 1550000, budgetAmount: 1600000 },
    { date: "2024-01-18", amount: 1650000, budgetAmount: 1600000 },
    { date: "2024-01-19", amount: 1500000, budgetAmount: 1600000 },
    { date: "2024-01-20", amount: 850000, budgetAmount: 1600000 },
    { date: "2024-01-21", amount: 700000, budgetAmount: 1600000 },
    { date: "2024-01-22", amount: 1600000, budgetAmount: 1600000 },
    { date: "2024-01-23", amount: 1900000, budgetAmount: 1600000 },
    { date: "2024-01-24", amount: 1750000, budgetAmount: 1600000 },
    { date: "2024-01-25", amount: 1550000, budgetAmount: 1600000 },
    { date: "2024-01-26", amount: 1650000, budgetAmount: 1600000 },
    { date: "2024-01-27", amount: 900000, budgetAmount: 1600000 },
    { date: "2024-01-28", amount: 750000, budgetAmount: 1600000 },
    { date: "2024-01-29", amount: 1700000, budgetAmount: 1600000 },
    { date: "2024-01-30", amount: 1800000, budgetAmount: 1600000 },
    { date: "2024-01-31", amount: 1500000, budgetAmount: 1600000 },
  ],
  byAccount: [
    { accountId: "801", accountCode: "801", accountName: "급여", currentAmount: 18000000, budgetAmount: 18000000, previousAmount: 17500000, variance: 0, percentage: 40.0 },
    { accountId: "802", accountCode: "802", accountName: "복리후생비", currentAmount: 3500000, budgetAmount: 4000000, previousAmount: 3200000, variance: -500000, percentage: 7.8 },
    { accountId: "810", accountCode: "810", accountName: "지급임차료", currentAmount: 5000000, budgetAmount: 5000000, previousAmount: 5000000, variance: 0, percentage: 11.1 },
    { accountId: "803", accountCode: "803", accountName: "여비교통비", currentAmount: 2800000, budgetAmount: 2500000, previousAmount: 2400000, variance: 300000, percentage: 6.2 },
    { accountId: "804", accountCode: "804", accountName: "통신비", currentAmount: 1200000, budgetAmount: 1500000, previousAmount: 1100000, variance: -300000, percentage: 2.7 },
    { accountId: "805", accountCode: "805", accountName: "소모품비", currentAmount: 1500000, budgetAmount: 1800000, previousAmount: 1400000, variance: -300000, percentage: 3.3 },
    { accountId: "806", accountCode: "806", accountName: "접대비", currentAmount: 2200000, budgetAmount: 2000000, previousAmount: 1800000, variance: 200000, percentage: 4.9 },
    { accountId: "807", accountCode: "807", accountName: "광고선전비", currentAmount: 3000000, budgetAmount: 3500000, previousAmount: 2800000, variance: -500000, percentage: 6.7 },
    { accountId: "808", accountCode: "808", accountName: "교육훈련비", currentAmount: 1800000, budgetAmount: 2000000, previousAmount: 1500000, variance: -200000, percentage: 4.0 },
    { accountId: "820", accountCode: "820", accountName: "감가상각비", currentAmount: 2500000, budgetAmount: 2500000, previousAmount: 2500000, variance: 0, percentage: 5.6 },
    { accountId: "821", accountCode: "821", accountName: "보험료", currentAmount: 1500000, budgetAmount: 1500000, previousAmount: 1400000, variance: 0, percentage: 3.3 },
    { accountId: "830", accountCode: "830", accountName: "수수료", currentAmount: 2000000, budgetAmount: 2200000, previousAmount: 1900000, variance: -200000, percentage: 4.4 },
  ],
  byDepartment: [
    { departmentId: "d1", departmentName: "영업부", currentAmount: 15000000, budgetAmount: 16000000, employeeCount: 25, perCapita: 600000 },
    { departmentId: "d2", departmentName: "개발부", currentAmount: 12000000, budgetAmount: 12500000, employeeCount: 30, perCapita: 400000 },
    { departmentId: "d3", departmentName: "마케팅부", currentAmount: 8000000, budgetAmount: 9000000, employeeCount: 15, perCapita: 533333 },
    { departmentId: "d4", departmentName: "경영지원부", currentAmount: 5500000, budgetAmount: 6000000, employeeCount: 10, perCapita: 550000 },
    { departmentId: "d5", departmentName: "인사부", currentAmount: 2500000, budgetAmount: 3000000, employeeCount: 5, perCapita: 500000 },
    { departmentId: "d6", departmentName: "재무부", currentAmount: 2000000, budgetAmount: 2500000, employeeCount: 5, perCapita: 400000 },
  ],
};

// Simple line chart component using SVG
function ExpenseLineChart({ data, maxValue }: { data: ExpenseTrendItem[]; maxValue: number }) {
  const width = 100;
  const height = 40;
  const padding = 2;

  const points = data.map((item, index) => {
    const x = padding + (index / (data.length - 1)) * (width - 2 * padding);
    const y = height - padding - (item.amount / maxValue) * (height - 2 * padding);
    return `${x},${y}`;
  }).join(" ");

  const budgetPoints = data.map((item, index) => {
    const x = padding + (index / (data.length - 1)) * (width - 2 * padding);
    const y = height - padding - (item.budgetAmount / maxValue) * (height - 2 * padding);
    return `${x},${y}`;
  }).join(" ");

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-40" preserveAspectRatio="none">
      {/* Budget line (dashed) */}
      <polyline
        points={budgetPoints}
        fill="none"
        stroke="#9ca3af"
        strokeWidth="0.5"
        strokeDasharray="2,2"
      />
      {/* Expense line */}
      <polyline
        points={points}
        fill="none"
        stroke="#3b82f6"
        strokeWidth="0.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

// Budget progress bar
function BudgetProgressBar({ current, budget, showLabel = true }: { current: number; budget: number; showLabel?: boolean }) {
  const percentage = budget > 0 ? (current / budget) * 100 : 0;
  const isOverBudget = percentage > 100;
  const isWarning = percentage > 90 && percentage <= 100;

  return (
    <div className="space-y-1">
      {showLabel && (
        <div className="flex justify-between text-xs">
          <span className="text-muted-foreground">
            {formatCurrency(current, { compact: true })} / {formatCurrency(budget, { compact: true })}
          </span>
          <span className={cn(
            "font-medium",
            isOverBudget ? "text-red-600" : isWarning ? "text-amber-600" : "text-green-600"
          )}>
            {percentage.toFixed(0)}%
          </span>
        </div>
      )}
      <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
        <div
          className={cn(
            "h-full rounded-full transition-all",
            isOverBudget ? "bg-red-500" : isWarning ? "bg-amber-500" : "bg-green-500"
          )}
          style={{ width: `${Math.min(percentage, 100)}%` }}
        />
      </div>
    </div>
  );
}

export function ExpenseReportPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [groupBy, setGroupBy] = useState<"day" | "week" | "month">("day");
  const [includeBudget, setIncludeBudget] = useState(true);
  const [activeTab, setActiveTab] = useState<"account" | "department" | "trend">("account");

  // Fetch expense report data
  const {
    data: expenseReportResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["reports", "expense", startDate, endDate, groupBy, includeBudget],
    queryFn: () =>
      reportsApi.getExpenseReport({
        startDate,
        endDate,
        groupBy,
        includeBudget,
      }),
    enabled: !!startDate && !!endDate,
  });

  // Use mock data if no real data
  const expenseData = expenseReportResponse?.data || mockExpenseReportData;

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  // Calculate variance
  const calculateVariance = (current: number, previous?: number) => {
    if (!previous || previous === 0) return null;
    return ((current - previous) / previous) * 100;
  };

  // For expenses, increase is usually bad (inverse)
  const renderVarianceIndicator = (current: number, previous?: number) => {
    const variance = calculateVariance(current, previous);
    if (variance === null) return null;

    // For expenses: increase is bad (red), decrease is good (green)
    if (variance > 0) {
      return (
        <span className="flex items-center text-red-600 text-xs">
          <ArrowUpRight className="h-3 w-3 mr-0.5" />
          {variance.toFixed(1)}%
        </span>
      );
    } else if (variance < 0) {
      return (
        <span className="flex items-center text-green-600 text-xs">
          <ArrowDownRight className="h-3 w-3 mr-0.5" />
          {Math.abs(variance).toFixed(1)}%
        </span>
      );
    }
    return (
      <span className="flex items-center text-muted-foreground text-xs">
        <Minus className="h-3 w-3 mr-0.5" />
        0%
      </span>
    );
  };

  // Calculate max value for chart scaling
  const maxTrendValue = useMemo(() => {
    const maxExpense = Math.max(...expenseData.trend.map((item) => item.amount));
    const maxBudget = Math.max(...expenseData.trend.map((item) => item.budgetAmount));
    return Math.max(maxExpense, maxBudget, 1);
  }, [expenseData.trend]);

  // Count accounts over budget
  const overBudgetCount = expenseData.byAccount.filter((a) => a.variance > 0).length;
  const underBudgetCount = expenseData.byAccount.filter((a) => a.variance < 0).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">비용 분석</h1>
          <p className="text-muted-foreground">
            계정과목별, 부서별 비용 현황과 예산 대비 실적을 분석합니다.
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Download className="h-4 w-4 mr-2" />
            내보내기
          </Button>
          <Button variant="outline" onClick={() => window.print()}>
            <Printer className="h-4 w-4 mr-2" />
            인쇄
          </Button>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row md:items-end gap-4">
            <div className="flex-1">
              <DateRangePicker
                startDate={startDate}
                endDate={endDate}
                onDateChange={handleDateChange}
                label="조회기간"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">집계 단위</label>
              <Select
                value={groupBy}
                onChange={(value) => setGroupBy(value as "day" | "week" | "month")}
                options={[
                  { value: "day", label: "일별" },
                  { value: "week", label: "주별" },
                  { value: "month", label: "월별" },
                ]}
                className="w-32"
              />
            </div>
            <label className="flex items-center space-x-2 h-9">
              <input
                type="checkbox"
                checked={includeBudget}
                onChange={(e) => setIncludeBudget(e.target.checked)}
                className="rounded border-input"
              />
              <span className="text-sm">예산 포함</span>
            </label>
          </div>
        </CardContent>
      </Card>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">총 비용</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {formatCurrency(expenseData.summary.totalExpense, { compact: true })}
                </p>
                <div className="mt-1">
                  {renderVarianceIndicator(
                    expenseData.summary.totalExpense,
                    expenseData.summary.previousTotalExpense
                  )}
                </div>
              </div>
              <div className="p-2 bg-red-100 rounded-lg">
                <Wallet className="h-5 w-5 text-red-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">예산</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {formatCurrency(expenseData.summary.totalBudget, { compact: true })}
                </p>
                <div className="mt-2">
                  <BudgetProgressBar
                    current={expenseData.summary.totalExpense}
                    budget={expenseData.summary.totalBudget}
                    showLabel={false}
                  />
                </div>
              </div>
              <div className="p-2 bg-blue-100 rounded-lg">
                <Target className="h-5 w-5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">예산 잔액</p>
                <p className={cn(
                  "text-2xl font-bold font-mono mt-1",
                  expenseData.summary.varianceFromBudget >= 0 ? "text-green-600" : "text-red-600"
                )}>
                  {formatCurrency(Math.abs(expenseData.summary.varianceFromBudget), { compact: true })}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {expenseData.summary.varianceFromBudget >= 0 ? "예산 내 집행" : "예산 초과"}
                </p>
              </div>
              <div className={cn(
                "p-2 rounded-lg",
                expenseData.summary.varianceFromBudget >= 0 ? "bg-green-100" : "bg-red-100"
              )}>
                {expenseData.summary.varianceFromBudget >= 0 ? (
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                ) : (
                  <AlertTriangle className="h-5 w-5 text-red-600" />
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">계정과목 현황</p>
                <div className="flex items-center gap-3 mt-2">
                  <div className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-red-500" />
                    <span className="text-sm">초과 {overBudgetCount}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-green-500" />
                    <span className="text-sm">절감 {underBudgetCount}</span>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mt-1">
                  전체 {expenseData.byAccount.length}개 과목
                </p>
              </div>
              <div className="p-2 bg-purple-100 rounded-lg">
                <PieChart className="h-5 w-5 text-purple-600" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Content */}
      {isLoading ? (
        <Card>
          <CardContent className="py-16 flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </CardContent>
        </Card>
      ) : error ? (
        <Card>
          <CardContent className="py-16 text-center">
            <p className="text-destructive">데이터를 불러올 수 없습니다.</p>
          </CardContent>
        </Card>
      ) : (
        <>
          {/* Tab Navigation */}
          <div className="flex space-x-1 border-b">
            <button
              onClick={() => setActiveTab("account")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "account"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <Wallet className="h-4 w-4 inline mr-2" />
              계정과목별
            </button>
            <button
              onClick={() => setActiveTab("department")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "department"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <Building2 className="h-4 w-4 inline mr-2" />
              부서별
            </button>
            <button
              onClick={() => setActiveTab("trend")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "trend"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <TrendingUp className="h-4 w-4 inline mr-2" />
              비용 추이
            </button>
          </div>

          {/* Account Tab */}
          {activeTab === "account" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">계정과목별 비용 현황</CardTitle>
              </CardHeader>
              <CardContent>
                {/* Table Header */}
                <div className="grid grid-cols-12 gap-4 py-2 px-3 bg-muted rounded-lg mb-3 text-xs font-medium text-muted-foreground">
                  <div className="col-span-3">계정과목</div>
                  <div className="col-span-2 text-right">당기 실적</div>
                  <div className="col-span-2 text-right">예산</div>
                  <div className="col-span-3">예산 대비</div>
                  <div className="col-span-2 text-right">전기 대비</div>
                </div>

                <div className="space-y-2">
                  {expenseData.byAccount.map((account) => {
                    const budgetPercentage = account.budgetAmount > 0
                      ? (account.currentAmount / account.budgetAmount) * 100
                      : 0;
                    const isOverBudget = account.variance > 0;
                    const isUnderBudget = account.variance < 0;

                    return (
                      <div
                        key={account.accountId}
                        className="grid grid-cols-12 gap-4 py-3 px-3 rounded hover:bg-muted/30 items-center"
                      >
                        <div className="col-span-3">
                          <span className="font-mono text-xs text-muted-foreground mr-2">
                            {account.accountCode}
                          </span>
                          <span className="font-medium">{account.accountName}</span>
                        </div>
                        <div className="col-span-2 text-right font-mono">
                          {formatCurrency(account.currentAmount, { showSymbol: false })}
                        </div>
                        <div className="col-span-2 text-right font-mono text-muted-foreground">
                          {formatCurrency(account.budgetAmount, { showSymbol: false })}
                        </div>
                        <div className="col-span-3">
                          <div className="flex items-center gap-2">
                            <div className="flex-1">
                              <BudgetProgressBar
                                current={account.currentAmount}
                                budget={account.budgetAmount}
                                showLabel={false}
                              />
                            </div>
                            <span className={cn(
                              "text-xs font-medium w-12 text-right",
                              isOverBudget ? "text-red-600" : isUnderBudget ? "text-green-600" : "text-muted-foreground"
                            )}>
                              {budgetPercentage.toFixed(0)}%
                            </span>
                          </div>
                        </div>
                        <div className="col-span-2 text-right">
                          {renderVarianceIndicator(account.currentAmount, account.previousAmount)}
                        </div>
                      </div>
                    );
                  })}
                </div>

                {/* Summary */}
                <div className="grid grid-cols-12 gap-4 py-3 px-3 bg-primary/10 rounded-lg mt-4 font-semibold">
                  <div className="col-span-3">합계</div>
                  <div className="col-span-2 text-right font-mono">
                    {formatCurrency(expenseData.summary.totalExpense, { showSymbol: false })}
                  </div>
                  <div className="col-span-2 text-right font-mono">
                    {formatCurrency(expenseData.summary.totalBudget, { showSymbol: false })}
                  </div>
                  <div className="col-span-3">
                    <BudgetProgressBar
                      current={expenseData.summary.totalExpense}
                      budget={expenseData.summary.totalBudget}
                      showLabel={false}
                    />
                  </div>
                  <div className="col-span-2 text-right">
                    {renderVarianceIndicator(
                      expenseData.summary.totalExpense,
                      expenseData.summary.previousTotalExpense
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Department Tab */}
          {activeTab === "department" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">부서별 비용 분석</CardTitle>
              </CardHeader>
              <CardContent>
                {/* Table Header */}
                <div className="grid grid-cols-12 gap-4 py-2 px-3 bg-muted rounded-lg mb-3 text-xs font-medium text-muted-foreground">
                  <div className="col-span-3">부서</div>
                  <div className="col-span-2 text-right">비용</div>
                  <div className="col-span-2 text-right">예산</div>
                  <div className="col-span-3">예산 대비</div>
                  <div className="col-span-1 text-right">인원</div>
                  <div className="col-span-1 text-right">인당</div>
                </div>

                <div className="space-y-2">
                  {expenseData.byDepartment.map((dept) => {
                    const budgetPct = dept.budgetAmount > 0
                      ? (dept.currentAmount / dept.budgetAmount) * 100
                      : 0;
                    const isOverBudget = budgetPct > 100;

                    return (
                      <div
                        key={dept.departmentId}
                        className={cn(
                          "grid grid-cols-12 gap-4 py-3 px-3 rounded hover:bg-muted/30 items-center",
                          isOverBudget && "bg-destructive/5"
                        )}
                      >
                        <div className="col-span-3">
                          <span className="font-medium">{dept.departmentName}</span>
                        </div>
                        <div className="col-span-2 text-right font-mono">
                          {formatCurrency(dept.currentAmount, { showSymbol: false })}
                        </div>
                        <div className="col-span-2 text-right font-mono text-muted-foreground">
                          {formatCurrency(dept.budgetAmount, { showSymbol: false })}
                        </div>
                        <div className="col-span-3">
                          <BudgetProgressBar
                            current={dept.currentAmount}
                            budget={dept.budgetAmount}
                            showLabel={false}
                          />
                        </div>
                        <div className="col-span-1 text-right">
                          {dept.employeeCount}명
                        </div>
                        <div className="col-span-1 text-right font-mono text-sm">
                          {formatCurrency(dept.perCapita, { compact: true })}
                        </div>
                      </div>
                    );
                  })}
                </div>

                {/* Department Summary Cards */}
                <div className="grid grid-cols-3 gap-4 mt-6 pt-6 border-t">
                  <div className="p-4 rounded-lg bg-muted/50">
                    <p className="text-sm text-muted-foreground">최고 비용 부서</p>
                    <p className="font-semibold mt-1">
                      {expenseData.byDepartment.sort((a, b) => b.currentAmount - a.currentAmount)[0]?.departmentName}
                    </p>
                    <p className="text-sm font-mono text-muted-foreground">
                      {formatCurrency(
                        expenseData.byDepartment.sort((a, b) => b.currentAmount - a.currentAmount)[0]?.currentAmount || 0,
                        { compact: true }
                      )}
                    </p>
                  </div>
                  <div className="p-4 rounded-lg bg-muted/50">
                    <p className="text-sm text-muted-foreground">최고 인당 비용</p>
                    <p className="font-semibold mt-1">
                      {expenseData.byDepartment.sort((a, b) => b.perCapita - a.perCapita)[0]?.departmentName}
                    </p>
                    <p className="text-sm font-mono text-muted-foreground">
                      {formatCurrency(
                        expenseData.byDepartment.sort((a, b) => b.perCapita - a.perCapita)[0]?.perCapita || 0,
                        { compact: true }
                      )}/인
                    </p>
                  </div>
                  <div className="p-4 rounded-lg bg-muted/50">
                    <p className="text-sm text-muted-foreground">전체 평균 인당 비용</p>
                    <p className="font-semibold mt-1 font-mono">
                      {formatCurrency(
                        expenseData.summary.totalExpense /
                        expenseData.byDepartment.reduce((sum, d) => sum + d.employeeCount, 0),
                        { compact: true }
                      )}/인
                    </p>
                    <p className="text-sm text-muted-foreground">
                      총 {expenseData.byDepartment.reduce((sum, d) => sum + d.employeeCount, 0)}명
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Trend Tab */}
          {activeTab === "trend" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">비용 추이 차트</CardTitle>
              </CardHeader>
              <CardContent>
                <ExpenseLineChart data={expenseData.trend} maxValue={maxTrendValue} />

                <div className="flex items-center justify-center gap-6 mt-4 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <span className="w-3 h-0.5 bg-blue-500" />
                    실제 비용
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-3 h-0.5 bg-gray-400" style={{ borderStyle: "dashed" }} />
                    예산
                  </span>
                </div>

                {/* Daily Statistics */}
                <div className="grid grid-cols-4 gap-4 mt-6 pt-6 border-t">
                  <div className="text-center">
                    <p className="text-sm text-muted-foreground">일평균 비용</p>
                    <p className="font-mono font-semibold mt-1">
                      {formatCurrency(
                        expenseData.trend.reduce((sum, d) => sum + d.amount, 0) / expenseData.trend.length,
                        { compact: true }
                      )}
                    </p>
                  </div>
                  <div className="text-center">
                    <p className="text-sm text-muted-foreground">최대 비용일</p>
                    <p className="font-mono font-semibold mt-1">
                      {formatCurrency(
                        Math.max(...expenseData.trend.map((d) => d.amount)),
                        { compact: true }
                      )}
                    </p>
                  </div>
                  <div className="text-center">
                    <p className="text-sm text-muted-foreground">최소 비용일</p>
                    <p className="font-mono font-semibold mt-1">
                      {formatCurrency(
                        Math.min(...expenseData.trend.map((d) => d.amount)),
                        { compact: true }
                      )}
                    </p>
                  </div>
                  <div className="text-center">
                    <p className="text-sm text-muted-foreground">예산 초과일</p>
                    <p className="font-mono font-semibold mt-1">
                      {expenseData.trend.filter((d) => d.amount > d.budgetAmount).length}일
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
