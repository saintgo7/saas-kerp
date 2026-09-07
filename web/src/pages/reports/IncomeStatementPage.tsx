import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Download,
  Printer,
  TrendingUp,
  TrendingDown,
  Minus,
  BarChart3,
} from "lucide-react";
import {
  Button,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui";
import { DateRangePicker, notifyUnavailable } from "@/components/common";
import { formatCurrency } from "@/lib/utils";
import { ledgerApi } from "@/api";
import { getErrorMessage } from "@/services/api";
import type { IncomeStatementSection } from "@/api/ledger";

export function IncomeStatementPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [showComparison, setShowComparison] = useState(true);
  const [includeBudget, setIncludeBudget] = useState(false);

  // Fetch income statement data
  const {
    data: incomeStatementResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["reports", "income-statement", startDate, endDate, includeBudget],
    queryFn: () =>
      ledgerApi.incomeStatement({
        startDate,
        endDate,
        includeBudget,
      }),
    enabled: !!startDate && !!endDate,
  });

  // Never fall back to sample figures on a financial statement.
  const incomeStatementData = incomeStatementResponse?.data ?? null;

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  // Calculate variance percentages
  const calculateVariance = (current: number, previous?: number) => {
    if (!previous || previous === 0) return null;
    return ((current - previous) / previous) * 100;
  };

  const renderVarianceIndicator = (current: number, previous?: number) => {
    const variance = calculateVariance(current, previous);
    if (variance === null) return null;

    if (variance > 0) {
      return (
        <span className="flex items-center text-green-600 text-xs">
          <TrendingUp className="h-3 w-3 mr-0.5" />
          {variance.toFixed(1)}%
        </span>
      );
    } else if (variance < 0) {
      return (
        <span className="flex items-center text-red-600 text-xs">
          <TrendingDown className="h-3 w-3 mr-0.5" />
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

  const renderSection = (section: IncomeStatementSection, isExpense = false) => (
    <div key={section.title} className="mb-4">
      <h4 className="font-semibold text-sm mb-2 px-2">{section.title}</h4>
      <div className="space-y-1">
        {section.accounts.map((account, idx) => (
          <div
            key={`${account.accountId}-${idx}`}
            className={`grid grid-cols-12 gap-2 py-1.5 px-2 rounded hover:bg-muted/30 ${
              account.isSubtotal ? "bg-muted/50 font-semibold" : ""
            }`}
          >
            <div className="col-span-5">
              <span style={{ paddingLeft: `${account.level * 12}px` }}>
                {account.accountCode && (
                  <span className="font-mono text-xs text-muted-foreground mr-2">
                    {account.accountCode}
                  </span>
                )}
                {account.accountName}
              </span>
            </div>
            <div className="col-span-3 text-right font-mono">
              {isExpense ? "(" : ""}
              {formatCurrency(account.currentAmount, { showSymbol: false })}
              {isExpense ? ")" : ""}
            </div>
            {showComparison && (
              <>
                <div className="col-span-3 text-right font-mono text-muted-foreground">
                  {account.previousAmount !== undefined
                    ? `${isExpense ? "(" : ""}${formatCurrency(account.previousAmount, {
                        showSymbol: false,
                      })}${isExpense ? ")" : ""}`
                    : "-"}
                </div>
                <div className="col-span-1 text-right">
                  {account.previousAmount !== undefined &&
                    renderVarianceIndicator(account.currentAmount, account.previousAmount)}
                </div>
              </>
            )}
            {!showComparison && <div className="col-span-4" />}
          </div>
        ))}
      </div>
    </div>
  );

  const renderSectionTotal = (
    label: string,
    current: number,
    previous?: number,
    isSubtract = false,
    className = ""
  ) => (
    <div
      className={`grid grid-cols-12 gap-2 py-2 px-2 rounded font-semibold ${className}`}
    >
      <div className="col-span-5">{label}</div>
      <div className="col-span-3 text-right font-mono">
        {isSubtract ? "(" : ""}
        {formatCurrency(current, { showSymbol: false })}
        {isSubtract ? ")" : ""}
      </div>
      {showComparison && (
        <>
          <div className="col-span-3 text-right font-mono text-muted-foreground">
            {previous !== undefined
              ? `${isSubtract ? "(" : ""}${formatCurrency(previous, {
                  showSymbol: false,
                })}${isSubtract ? ")" : ""}`
              : "-"}
          </div>
          <div className="col-span-1 text-right">
            {previous !== undefined && renderVarianceIndicator(current, previous)}
          </div>
        </>
      )}
      {!showComparison && <div className="col-span-4" />}
    </div>
  );

  // Calculate gross margin and operating margin
  const grossMargin =
    incomeStatementData && incomeStatementData.revenue.total > 0
      ? (incomeStatementData.grossProfit / incomeStatementData.revenue.total) * 100
      : 0;
  const operatingMargin =
    incomeStatementData && incomeStatementData.revenue.total > 0
      ? (incomeStatementData.operatingIncome / incomeStatementData.revenue.total) * 100
      : 0;
  const netMargin =
    incomeStatementData && incomeStatementData.revenue.total > 0
      ? (incomeStatementData.netIncome / incomeStatementData.revenue.total) * 100
      : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">손익계산서</h1>
          <p className="text-muted-foreground">
            수익과 비용의 발생 현황을 보여줍니다.
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={() => notifyUnavailable("내보내기")}>
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
            <div className="flex items-center space-x-4">
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={showComparison}
                  onChange={(e) => setShowComparison(e.target.checked)}
                  className="rounded border-input"
                />
                <span className="text-sm">비교 컬럼 표시</span>
              </label>
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={includeBudget}
                  onChange={(e) => setIncludeBudget(e.target.checked)}
                  className="rounded border-input"
                />
                <span className="text-sm">예산 포함</span>
              </label>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* KPI Cards */}
      {incomeStatementData && (
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">매출액</p>
                <p className="text-xl font-bold font-mono">
                  {formatCurrency(incomeStatementData.revenue.total, { showSymbol: false })}
                </p>
              </div>
              {incomeStatementData.revenue.previousTotal &&
                renderVarianceIndicator(
                  incomeStatementData.revenue.total,
                  incomeStatementData.revenue.previousTotal
                )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">매출총이익</p>
                <p className="text-xl font-bold font-mono">
                  {formatCurrency(incomeStatementData.grossProfit, { showSymbol: false })}
                </p>
                <p className="text-xs text-muted-foreground">
                  마진율: {grossMargin.toFixed(1)}%
                </p>
              </div>
              {incomeStatementData.previousGrossProfit &&
                renderVarianceIndicator(
                  incomeStatementData.grossProfit,
                  incomeStatementData.previousGrossProfit
                )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">영업이익</p>
                <p className="text-xl font-bold font-mono">
                  {formatCurrency(incomeStatementData.operatingIncome, { showSymbol: false })}
                </p>
                <p className="text-xs text-muted-foreground">
                  마진율: {operatingMargin.toFixed(1)}%
                </p>
              </div>
              {incomeStatementData.previousOperatingIncome &&
                renderVarianceIndicator(
                  incomeStatementData.operatingIncome,
                  incomeStatementData.previousOperatingIncome
                )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">당기순이익</p>
                <p className="text-xl font-bold font-mono text-primary">
                  {formatCurrency(incomeStatementData.netIncome, { showSymbol: false })}
                </p>
                <p className="text-xs text-muted-foreground">
                  순이익률: {netMargin.toFixed(1)}%
                </p>
              </div>
              {incomeStatementData.previousNetIncome &&
                renderVarianceIndicator(
                  incomeStatementData.netIncome,
                  incomeStatementData.previousNetIncome
                )}
            </div>
          </CardContent>
        </Card>
      </div>
      )}

      {/* Income Statement Content */}
      {isLoading ? (
        <Card>
          <CardContent className="py-16 flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </CardContent>
        </Card>
      ) : error ? (
        <Card>
          <CardContent className="py-16 text-center">
            <p className="text-destructive">
              {getErrorMessage(error, "손익계산서 조회에 실패했습니다.")}
            </p>
          </CardContent>
        </Card>
      ) : !incomeStatementData ? (
        <Card>
          <CardContent className="py-16 text-center">
            <p className="text-muted-foreground">
              조회된 손익계산서 데이터가 없습니다.
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center space-x-2">
              <BarChart3 className="h-5 w-5" />
              <span>손익계산서 상세</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {/* Column Headers */}
            <div className="grid grid-cols-12 gap-2 py-2 px-2 bg-muted rounded-lg mb-4 text-xs font-medium text-muted-foreground">
              <div className="col-span-5">계정과목</div>
              <div className="col-span-3 text-right">당기</div>
              {showComparison && (
                <>
                  <div className="col-span-3 text-right">전기</div>
                  <div className="col-span-1 text-right">증감</div>
                </>
              )}
            </div>

            {/* Revenue */}
            {renderSection(incomeStatementData.revenue)}
            {renderSectionTotal(
              "매출액",
              incomeStatementData.revenue.total,
              incomeStatementData.revenue.previousTotal,
              false,
              "bg-green-50"
            )}

            {/* Cost of Sales */}
            {incomeStatementData.costOfSales && (
              <>
                <div className="my-4 border-t" />
                {renderSection(incomeStatementData.costOfSales, true)}
                {renderSectionTotal(
                  "매출원가",
                  incomeStatementData.costOfSales.total,
                  incomeStatementData.costOfSales.previousTotal,
                  true,
                  "bg-red-50"
                )}
              </>
            )}

            {/* Gross Profit */}
            <div className="my-4 border-t" />
            {renderSectionTotal(
              "매출총이익",
              incomeStatementData.grossProfit,
              incomeStatementData.previousGrossProfit,
              false,
              "bg-blue-50"
            )}

            {/* Operating Expenses */}
            <div className="my-4 border-t" />
            {renderSection(incomeStatementData.operatingExpenses, true)}
            {renderSectionTotal(
              "판매비와 관리비 계",
              incomeStatementData.operatingExpenses.total,
              incomeStatementData.operatingExpenses.previousTotal,
              true,
              "bg-orange-50"
            )}

            {/* Operating Income */}
            <div className="my-4 border-t" />
            {renderSectionTotal(
              "영업이익",
              incomeStatementData.operatingIncome,
              incomeStatementData.previousOperatingIncome,
              false,
              "bg-blue-100"
            )}

            {/* Non-Operating Income */}
            {incomeStatementData.nonOperatingIncome && (
              <>
                <div className="my-4 border-t" />
                {renderSection(incomeStatementData.nonOperatingIncome)}
                {renderSectionTotal(
                  "영업외수익 계",
                  incomeStatementData.nonOperatingIncome.total,
                  incomeStatementData.nonOperatingIncome.previousTotal,
                  false,
                  "bg-green-50"
                )}
              </>
            )}

            {/* Non-Operating Expenses */}
            {incomeStatementData.nonOperatingExpenses && (
              <>
                <div className="my-4 border-t" />
                {renderSection(incomeStatementData.nonOperatingExpenses, true)}
                {renderSectionTotal(
                  "영업외비용 계",
                  incomeStatementData.nonOperatingExpenses.total,
                  incomeStatementData.nonOperatingExpenses.previousTotal,
                  true,
                  "bg-red-50"
                )}
              </>
            )}

            {/* Income Before Tax */}
            <div className="my-4 border-t" />
            {renderSectionTotal(
              "법인세비용차감전순이익",
              incomeStatementData.incomeBeforeTax,
              incomeStatementData.previousIncomeBeforeTax,
              false,
              "bg-purple-50"
            )}

            {/* Income Tax */}
            {incomeStatementData.incomeTax !== undefined && (
              <div className="grid grid-cols-12 gap-2 py-2 px-2 rounded">
                <div className="col-span-5 pl-4">법인세비용</div>
                <div className="col-span-3 text-right font-mono">
                  ({formatCurrency(incomeStatementData.incomeTax, { showSymbol: false })})
                </div>
                {showComparison && <div className="col-span-4" />}
              </div>
            )}

            {/* Net Income */}
            <div className="my-4 border-t-2 border-primary" />
            {renderSectionTotal(
              "당기순이익",
              incomeStatementData.netIncome,
              incomeStatementData.previousNetIncome,
              false,
              "bg-primary/10 font-bold text-lg"
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
