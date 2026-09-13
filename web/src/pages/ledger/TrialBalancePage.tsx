import React, { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, Printer, ChevronRight, ChevronDown } from "lucide-react";
import {
  Button,
  Card,
  CardContent,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { DateRangePicker, notifyUnavailable } from "@/components/common";
import { formatCurrency } from "@/lib/utils";
import { ledgerApi } from "@/api";
import { getErrorMessage } from "@/services/api";
import type { TrialBalanceRow } from "@/api/ledger";
import type { AccountType } from "@/types";
import { ACCOUNT_TYPES } from "@/constants";

export function TrialBalancePage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [includeZeroBalance, setIncludeZeroBalance] = useState(false);
  const [expandedTypes, setExpandedTypes] = useState<Set<string>>(
    new Set(["asset", "liability", "equity", "revenue", "expense"])
  );
  const [viewMode, setViewMode] = useState<"summary" | "detail">("detail");

  // Fetch trial balance data
  const {
    data: trialBalanceResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["ledger", "trial-balance", startDate, endDate, includeZeroBalance],
    queryFn: () =>
      ledgerApi.trialBalance({ startDate, endDate, includeZeroBalance }),
    enabled: !!startDate && !!endDate,
  });

  // Never fall back to sample figures: an accounting screen must show only
  // what the server actually returned.
  const trialBalanceData = trialBalanceResponse?.data ?? null;

  // Group rows by account type
  const groupedRows = useMemo(() => {
    const groups: Record<AccountType, TrialBalanceRow[]> = {
      asset: [],
      liability: [],
      equity: [],
      revenue: [],
      expense: [],
    };

    trialBalanceData?.rows.forEach((row) => {
      if (row.level === 0 || (viewMode === "detail" && row.level > 0)) {
        groups[row.accountType].push(row);
      }
    });

    return groups;
  }, [trialBalanceData, viewMode]);

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  const toggleTypeExpand = (type: string) => {
    setExpandedTypes((prev) => {
      const next = new Set(prev);
      if (next.has(type)) {
        next.delete(type);
      } else {
        next.add(type);
      }
      return next;
    });
  };

  const getAccountTypeLabel = (type: AccountType) => {
    return ACCOUNT_TYPES.find((t) => t.value === type)?.label || type;
  };

  const getTypeColor = (type: AccountType) => {
    const colors: Record<AccountType, string> = {
      asset: "bg-blue-100 text-blue-800",
      liability: "bg-red-100 text-red-800",
      equity: "bg-purple-100 text-purple-800",
      revenue: "bg-green-100 text-green-800",
      expense: "bg-orange-100 text-orange-800",
    };
    return colors[type];
  };

  // Check if trial balance is balanced
  const totals = trialBalanceData?.totals ?? null;
  const isBalanced = totals
    ? totals.closingDebit === totals.closingCredit
    : false;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">시산표</h1>
          <p className="text-muted-foreground">
            계정과목별 잔액 현황을 조회합니다.
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
                  checked={includeZeroBalance}
                  onChange={(e) => setIncludeZeroBalance(e.target.checked)}
                  className="rounded border-input"
                />
                <span className="text-sm">잔액 없는 계정 포함</span>
              </label>
              <select
                className="h-9 rounded-md border border-input bg-background px-3 text-sm"
                value={viewMode}
                onChange={(e) => setViewMode(e.target.value as "summary" | "detail")}
              >
                <option value="detail">상세 보기</option>
                <option value="summary">요약 보기</option>
              </select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Balance Status */}
      {totals && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <Badge variant={isBalanced ? "success" : "destructive"}>
                  {isBalanced ? "균형" : "불균형"}
                </Badge>
                <span className="text-sm text-muted-foreground">
                  차변과 대변의 합계가 {isBalanced ? "일치합니다." : "일치하지 않습니다."}
                </span>
              </div>
              {!isBalanced && (
                <span className="text-sm text-destructive font-mono">
                  차이:{" "}
                  {formatCurrency(
                    Math.abs(totals.closingDebit - totals.closingCredit),
                    { showSymbol: false }
                  )}
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Trial Balance Table */}
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
              {getErrorMessage(error, "시산표 조회에 실패했습니다.")}
            </p>
          </CardContent>
        </Card>
      ) : !trialBalanceData ? (
        <Card>
          <CardContent className="py-16 text-center">
            <p className="text-muted-foreground">
              조회된 시산표 데이터가 없습니다.
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">코드</TableHead>
                  <TableHead>계정과목</TableHead>
                  <TableHead className="text-right">이월 차변</TableHead>
                  <TableHead className="text-right">이월 대변</TableHead>
                  <TableHead className="text-right">당기 차변</TableHead>
                  <TableHead className="text-right">당기 대변</TableHead>
                  <TableHead className="text-right">잔액 차변</TableHead>
                  <TableHead className="text-right">잔액 대변</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(["asset", "liability", "equity", "revenue", "expense"] as AccountType[]).map(
                  (type) => {
                    const typeRows = groupedRows[type];
                    const isExpanded = expandedTypes.has(type);
                    const headerRow = typeRows.find((r) => r.level === 0);

                    if (!headerRow && typeRows.length === 0) return null;

                    return (
                      <React.Fragment key={type}>
                        {/* Type Header */}
                        <TableRow
                          className="bg-muted/50 cursor-pointer hover:bg-muted"
                          onClick={() => toggleTypeExpand(type)}
                        >
                          <TableCell colSpan={2}>
                            <div className="flex items-center space-x-2">
                              {isExpanded ? (
                                <ChevronDown className="h-4 w-4" />
                              ) : (
                                <ChevronRight className="h-4 w-4" />
                              )}
                              <Badge className={getTypeColor(type)}>
                                {getAccountTypeLabel(type)}
                              </Badge>
                            </div>
                          </TableCell>
                          <TableCell className="text-right font-mono font-semibold">
                            {headerRow && headerRow.openingDebit > 0
                              ? formatCurrency(headerRow.openingDebit, { showSymbol: false })
                              : "-"}
                          </TableCell>
                          <TableCell className="text-right font-mono font-semibold">
                            {headerRow && headerRow.openingCredit > 0
                              ? formatCurrency(headerRow.openingCredit, { showSymbol: false })
                              : "-"}
                          </TableCell>
                          <TableCell className="text-right font-mono font-semibold">
                            {headerRow && headerRow.periodDebit > 0
                              ? formatCurrency(headerRow.periodDebit, { showSymbol: false })
                              : "-"}
                          </TableCell>
                          <TableCell className="text-right font-mono font-semibold">
                            {headerRow && headerRow.periodCredit > 0
                              ? formatCurrency(headerRow.periodCredit, { showSymbol: false })
                              : "-"}
                          </TableCell>
                          <TableCell className="text-right font-mono font-semibold">
                            {headerRow && headerRow.closingDebit > 0
                              ? formatCurrency(headerRow.closingDebit, { showSymbol: false })
                              : "-"}
                          </TableCell>
                          <TableCell className="text-right font-mono font-semibold">
                            {headerRow && headerRow.closingCredit > 0
                              ? formatCurrency(headerRow.closingCredit, { showSymbol: false })
                              : "-"}
                          </TableCell>
                        </TableRow>

                        {/* Detail Rows */}
                        {isExpanded &&
                          viewMode === "detail" &&
                          typeRows
                            .filter((r) => r.level > 0)
                            .map((row) => (
                              <TableRow key={row.accountId}>
                                <TableCell
                                  className="font-mono text-sm"
                                  style={{ paddingLeft: `${row.level * 20 + 16}px` }}
                                >
                                  {row.accountCode}
                                </TableCell>
                                <TableCell>{row.accountName}</TableCell>
                                <TableCell className="text-right font-mono">
                                  {row.openingDebit > 0
                                    ? formatCurrency(row.openingDebit, { showSymbol: false })
                                    : "-"}
                                </TableCell>
                                <TableCell className="text-right font-mono">
                                  {row.openingCredit > 0
                                    ? formatCurrency(row.openingCredit, { showSymbol: false })
                                    : "-"}
                                </TableCell>
                                <TableCell className="text-right font-mono">
                                  {row.periodDebit > 0
                                    ? formatCurrency(row.periodDebit, { showSymbol: false })
                                    : "-"}
                                </TableCell>
                                <TableCell className="text-right font-mono">
                                  {row.periodCredit > 0
                                    ? formatCurrency(row.periodCredit, { showSymbol: false })
                                    : "-"}
                                </TableCell>
                                <TableCell className="text-right font-mono">
                                  {row.closingDebit > 0
                                    ? formatCurrency(row.closingDebit, { showSymbol: false })
                                    : "-"}
                                </TableCell>
                                <TableCell className="text-right font-mono">
                                  {row.closingCredit > 0
                                    ? formatCurrency(row.closingCredit, { showSymbol: false })
                                    : "-"}
                                </TableCell>
                              </TableRow>
                            ))}
                      </React.Fragment>
                    );
                  }
                )}

                {/* Totals */}
                <TableRow className="bg-primary/10 font-bold">
                  <TableCell colSpan={2} className="text-right">
                    합계
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(trialBalanceData.totals.openingDebit, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(trialBalanceData.totals.openingCredit, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(trialBalanceData.totals.periodDebit, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(trialBalanceData.totals.periodCredit, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(trialBalanceData.totals.closingDebit, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(trialBalanceData.totals.closingCredit, { showSymbol: false })}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
