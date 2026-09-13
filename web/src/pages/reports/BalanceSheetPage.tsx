import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, Printer, Calendar, TrendingUp, TrendingDown, Minus } from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
} from "@/components/ui";
import { notifyUnavailable } from "@/components/common";
import { formatCurrency } from "@/lib/utils";
import { ledgerApi } from "@/api";
import { getErrorMessage } from "@/services/api";
import type { BalanceSheetSection } from "@/api/ledger";

export function BalanceSheetPage() {
  const today = new Date();
  const lastDayOfMonth = new Date(today.getFullYear(), today.getMonth() + 1, 0);

  const [asOfDate, setAsOfDate] = useState(lastDayOfMonth.toISOString().split("T")[0]);
  const [comparisonDate, setComparisonDate] = useState("");
  const [showComparison, setShowComparison] = useState(true);

  // Fetch balance sheet data
  const {
    data: balanceSheetResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["reports", "balance-sheet", asOfDate, comparisonDate],
    queryFn: () =>
      ledgerApi.balanceSheet({
        asOfDate,
        comparisonDate: comparisonDate || undefined,
      }),
    enabled: !!asOfDate,
  });

  // Never fall back to sample figures on a financial statement.
  const balanceSheetData = balanceSheetResponse?.data ?? null;

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

  const renderSection = (section: BalanceSheetSection) => (
    <div key={section.title} className="mb-6">
      <h4 className="font-semibold text-sm text-muted-foreground mb-2 uppercase tracking-wide">
        {section.title}
      </h4>
      <div className="space-y-1">
        {section.accounts.map((account, idx) => (
          <div
            key={`${account.accountId}-${idx}`}
            className={`grid grid-cols-12 gap-2 py-1.5 px-2 rounded ${
              account.isSubtotal ? "bg-muted/50 font-semibold" : "hover:bg-muted/30"
            }`}
          >
            <div className="col-span-5">
              <span
                style={{ paddingLeft: account.isSubtotal ? 0 : `${account.level * 12}px` }}
              >
                {!account.isSubtotal && account.accountCode && (
                  <span className="font-mono text-xs text-muted-foreground mr-2">
                    {account.accountCode}
                  </span>
                )}
                {account.accountName}
              </span>
            </div>
            <div className="col-span-3 text-right font-mono">
              {formatCurrency(account.currentAmount, { showSymbol: false })}
            </div>
            {showComparison && (
              <>
                <div className="col-span-3 text-right font-mono text-muted-foreground">
                  {account.previousAmount !== undefined
                    ? formatCurrency(account.previousAmount, { showSymbol: false })
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

  // Check if balance sheet is balanced
  const isBalanced = balanceSheetData
    ? balanceSheetData.totalAssets === balanceSheetData.totalLiabilitiesAndEquity
    : false;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">재무상태표</h1>
          <p className="text-muted-foreground">
            자산, 부채, 자본의 현황을 보여줍니다.
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
            <div>
              <label className="block text-sm font-medium mb-1.5">기준일</label>
              <div className="relative">
                <Calendar className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  type="date"
                  value={asOfDate}
                  onChange={(e) => setAsOfDate(e.target.value)}
                  className="pl-9 w-40"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">비교일 (선택)</label>
              <div className="relative">
                <Calendar className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  type="date"
                  value={comparisonDate}
                  onChange={(e) => setComparisonDate(e.target.value)}
                  className="pl-9 w-40"
                />
              </div>
            </div>
            <label className="flex items-center space-x-2 h-9">
              <input
                type="checkbox"
                checked={showComparison}
                onChange={(e) => setShowComparison(e.target.checked)}
                className="rounded border-input"
              />
              <span className="text-sm">비교 컬럼 표시</span>
            </label>
          </div>
        </CardContent>
      </Card>

      {/* Balance Status */}
      {balanceSheetData && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <Badge variant={isBalanced ? "success" : "destructive"}>
                  {isBalanced ? "균형" : "불균형"}
                </Badge>
                <span className="text-sm text-muted-foreground">
                  자산 = 부채 + 자본
                </span>
              </div>
              {!isBalanced && (
                <span className="text-sm text-destructive font-mono">
                  차이:{" "}
                  {formatCurrency(
                    Math.abs(
                      balanceSheetData.totalAssets -
                        balanceSheetData.totalLiabilitiesAndEquity
                    ),
                    { showSymbol: false }
                  )}
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Balance Sheet Content */}
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
              {getErrorMessage(error, "재무상태표 조회에 실패했습니다.")}
            </p>
          </CardContent>
        </Card>
      ) : !balanceSheetData ? (
        <Card>
          <CardContent className="py-16 text-center">
            <p className="text-muted-foreground">
              조회된 재무상태표 데이터가 없습니다.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Assets */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center justify-between">
                <span>자산</span>
                <span className="font-mono text-primary">
                  {formatCurrency(balanceSheetData.totalAssets, { showSymbol: false })}
                </span>
              </CardTitle>
              {showComparison && (
                <p className="text-sm text-muted-foreground">
                  전기:{" "}
                  <span className="font-mono">
                    {formatCurrency(balanceSheetData.previousTotalAssets || 0, {
                      showSymbol: false,
                    })}
                  </span>
                  {balanceSheetData.previousTotalAssets &&
                    renderVarianceIndicator(
                      balanceSheetData.totalAssets,
                      balanceSheetData.previousTotalAssets
                    )}
                </p>
              )}
            </CardHeader>
            <CardContent>
              {/* Column Headers */}
              <div className="grid grid-cols-12 gap-2 py-2 px-2 bg-muted rounded-lg mb-3 text-xs font-medium text-muted-foreground">
                <div className="col-span-5">계정과목</div>
                <div className="col-span-3 text-right">당기</div>
                {showComparison && (
                  <>
                    <div className="col-span-3 text-right">전기</div>
                    <div className="col-span-1 text-right">증감</div>
                  </>
                )}
              </div>

              {balanceSheetData.assets.map(renderSection)}

              {/* Total Assets */}
              <div className="grid grid-cols-12 gap-2 py-3 px-2 bg-primary/10 rounded-lg font-bold mt-4">
                <div className="col-span-5">자산 총계</div>
                <div className="col-span-3 text-right font-mono">
                  {formatCurrency(balanceSheetData.totalAssets, { showSymbol: false })}
                </div>
                {showComparison && (
                  <>
                    <div className="col-span-3 text-right font-mono text-muted-foreground">
                      {formatCurrency(balanceSheetData.previousTotalAssets || 0, {
                        showSymbol: false,
                      })}
                    </div>
                    <div className="col-span-1 text-right">
                      {renderVarianceIndicator(
                        balanceSheetData.totalAssets,
                        balanceSheetData.previousTotalAssets
                      )}
                    </div>
                  </>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Liabilities & Equity */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center justify-between">
                <span>부채 및 자본</span>
                <span className="font-mono text-primary">
                  {formatCurrency(balanceSheetData.totalLiabilitiesAndEquity, {
                    showSymbol: false,
                  })}
                </span>
              </CardTitle>
              {showComparison && (
                <p className="text-sm text-muted-foreground">
                  전기:{" "}
                  <span className="font-mono">
                    {formatCurrency(
                      balanceSheetData.previousTotalLiabilitiesAndEquity || 0,
                      { showSymbol: false }
                    )}
                  </span>
                  {balanceSheetData.previousTotalLiabilitiesAndEquity &&
                    renderVarianceIndicator(
                      balanceSheetData.totalLiabilitiesAndEquity,
                      balanceSheetData.previousTotalLiabilitiesAndEquity
                    )}
                </p>
              )}
            </CardHeader>
            <CardContent>
              {/* Column Headers */}
              <div className="grid grid-cols-12 gap-2 py-2 px-2 bg-muted rounded-lg mb-3 text-xs font-medium text-muted-foreground">
                <div className="col-span-5">계정과목</div>
                <div className="col-span-3 text-right">당기</div>
                {showComparison && (
                  <>
                    <div className="col-span-3 text-right">전기</div>
                    <div className="col-span-1 text-right">증감</div>
                  </>
                )}
              </div>

              {/* Liabilities */}
              <h3 className="font-semibold mb-3">부채</h3>
              {balanceSheetData.liabilities.map(renderSection)}

              {/* Equity */}
              <h3 className="font-semibold mb-3 mt-6">자본</h3>
              {balanceSheetData.equity.map(renderSection)}

              {/* Total Liabilities & Equity */}
              <div className="grid grid-cols-12 gap-2 py-3 px-2 bg-primary/10 rounded-lg font-bold mt-4">
                <div className="col-span-5">부채 및 자본 총계</div>
                <div className="col-span-3 text-right font-mono">
                  {formatCurrency(balanceSheetData.totalLiabilitiesAndEquity, {
                    showSymbol: false,
                  })}
                </div>
                {showComparison && (
                  <>
                    <div className="col-span-3 text-right font-mono text-muted-foreground">
                      {formatCurrency(
                        balanceSheetData.previousTotalLiabilitiesAndEquity || 0,
                        { showSymbol: false }
                      )}
                    </div>
                    <div className="col-span-1 text-right">
                      {renderVarianceIndicator(
                        balanceSheetData.totalLiabilitiesAndEquity,
                        balanceSheetData.previousTotalLiabilitiesAndEquity
                      )}
                    </div>
                  </>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
