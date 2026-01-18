import { useState } from "react";
import {
  Download,
  Printer,
  TrendingUp,
  TrendingDown,
  Minus,
  Banknote,
} from "lucide-react";
import {
  Button,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui";
import { DateRangePicker } from "@/components/common";
import { formatCurrency } from "@/lib/utils";

// Types for Cash Flow Statement
interface CashFlowItem {
  id: string;
  name: string;
  level: number;
  currentAmount: number;
  previousAmount?: number;
  isSubtotal?: boolean;
}

interface CashFlowSection {
  title: string;
  items: CashFlowItem[];
  total: number;
  previousTotal?: number;
}

interface CashFlowStatementData {
  period: {
    startDate: string;
    endDate: string;
  };
  comparisonPeriod?: {
    startDate: string;
    endDate: string;
  };
  operatingActivities: CashFlowSection;
  investingActivities: CashFlowSection;
  financingActivities: CashFlowSection;
  netCashChange: number;
  previousNetCashChange?: number;
  beginningCash: number;
  previousBeginningCash?: number;
  endingCash: number;
  previousEndingCash?: number;
}

// Mock data for development following K-IFRS format
const mockCashFlowData: CashFlowStatementData = {
  period: {
    startDate: "2024-01-01",
    endDate: "2024-01-31",
  },
  comparisonPeriod: {
    startDate: "2023-12-01",
    endDate: "2023-12-31",
  },
  operatingActivities: {
    title: "영업활동으로 인한 현금흐름",
    items: [
      {
        id: "op-1",
        name: "당기순이익",
        level: 1,
        currentAmount: 12800000,
        previousAmount: 9880000,
      },
      {
        id: "op-2",
        name: "조정항목",
        level: 1,
        currentAmount: 0,
        previousAmount: 0,
        isSubtotal: true,
      },
      {
        id: "op-3",
        name: "감가상각비",
        level: 2,
        currentAmount: 1200000,
        previousAmount: 1200000,
      },
      {
        id: "op-4",
        name: "퇴직급여",
        level: 2,
        currentAmount: 500000,
        previousAmount: 450000,
      },
      {
        id: "op-5",
        name: "이자비용",
        level: 2,
        currentAmount: 800000,
        previousAmount: 900000,
      },
      {
        id: "op-6",
        name: "이자수익",
        level: 2,
        currentAmount: -500000,
        previousAmount: -400000,
      },
      {
        id: "op-7",
        name: "영업활동 자산부채 변동",
        level: 1,
        currentAmount: 0,
        previousAmount: 0,
        isSubtotal: true,
      },
      {
        id: "op-8",
        name: "매출채권의 감소(증가)",
        level: 2,
        currentAmount: -2500000,
        previousAmount: 1000000,
      },
      {
        id: "op-9",
        name: "재고자산의 감소(증가)",
        level: 2,
        currentAmount: 0,
        previousAmount: -500000,
      },
      {
        id: "op-10",
        name: "매입채무의 증가(감소)",
        level: 2,
        currentAmount: 1000000,
        previousAmount: -800000,
      },
      {
        id: "op-11",
        name: "미지급금의 증가(감소)",
        level: 2,
        currentAmount: 1000000,
        previousAmount: 500000,
      },
      {
        id: "op-12",
        name: "이자의 수취",
        level: 1,
        currentAmount: 450000,
        previousAmount: 380000,
      },
      {
        id: "op-13",
        name: "이자의 지급",
        level: 1,
        currentAmount: -750000,
        previousAmount: -850000,
      },
      {
        id: "op-14",
        name: "법인세의 납부",
        level: 1,
        currentAmount: -3200000,
        previousAmount: -2500000,
      },
    ],
    total: 10800000,
    previousTotal: 9260000,
  },
  investingActivities: {
    title: "투자활동으로 인한 현금흐름",
    items: [
      {
        id: "inv-1",
        name: "단기금융상품의 처분",
        level: 1,
        currentAmount: 5000000,
        previousAmount: 0,
      },
      {
        id: "inv-2",
        name: "단기금융상품의 취득",
        level: 1,
        currentAmount: -3000000,
        previousAmount: -5000000,
      },
      {
        id: "inv-3",
        name: "유형자산의 취득",
        level: 1,
        currentAmount: -2000000,
        previousAmount: -3500000,
      },
      {
        id: "inv-4",
        name: "유형자산의 처분",
        level: 1,
        currentAmount: 500000,
        previousAmount: 1000000,
      },
      {
        id: "inv-5",
        name: "무형자산의 취득",
        level: 1,
        currentAmount: -500000,
        previousAmount: -800000,
      },
    ],
    total: 0,
    previousTotal: -8300000,
  },
  financingActivities: {
    title: "재무활동으로 인한 현금흐름",
    items: [
      {
        id: "fin-1",
        name: "단기차입금의 차입",
        level: 1,
        currentAmount: 0,
        previousAmount: 5000000,
      },
      {
        id: "fin-2",
        name: "단기차입금의 상환",
        level: 1,
        currentAmount: -2000000,
        previousAmount: 0,
      },
      {
        id: "fin-3",
        name: "장기차입금의 상환",
        level: 1,
        currentAmount: -5000000,
        previousAmount: -3000000,
      },
      {
        id: "fin-4",
        name: "배당금의 지급",
        level: 1,
        currentAmount: -2000000,
        previousAmount: -1500000,
      },
      {
        id: "fin-5",
        name: "자기주식의 취득",
        level: 1,
        currentAmount: 0,
        previousAmount: -500000,
      },
    ],
    total: -9000000,
    previousTotal: 0,
  },
  netCashChange: 1800000,
  previousNetCashChange: 960000,
  beginningCash: 27700000,
  previousBeginningCash: 26740000,
  endingCash: 29500000,
  previousEndingCash: 27700000,
};

export function CashFlowStatementPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [showComparison, setShowComparison] = useState(true);

  // Use mock data (no API call in this version)
  const cashFlowData = mockCashFlowData;
  const isLoading = false;
  const error = null;

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  // Calculate variance percentages
  const calculateVariance = (current: number, previous?: number) => {
    if (previous === undefined || previous === 0) return null;
    return ((current - previous) / Math.abs(previous)) * 100;
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

  // Format amount with proper sign display for cash flow
  const formatCashFlowAmount = (amount: number, showSymbol = false) => {
    const formatted = formatCurrency(Math.abs(amount), { showSymbol });
    if (amount < 0) {
      return `(${formatted})`;
    }
    return formatted;
  };

  const renderSection = (section: CashFlowSection) => (
    <div key={section.title} className="mb-6">
      <h4 className="font-semibold text-sm text-muted-foreground mb-3 uppercase tracking-wide border-b pb-2">
        {section.title}
      </h4>
      <div className="space-y-1">
        {section.items.map((item) => (
          <div
            key={item.id}
            className={`grid grid-cols-12 gap-2 py-1.5 px-2 rounded ${
              item.isSubtotal
                ? "bg-muted/30 font-semibold text-sm mt-2"
                : "hover:bg-muted/30"
            }`}
          >
            <div className="col-span-5">
              <span
                style={{ paddingLeft: item.isSubtotal ? 0 : `${(item.level - 1) * 16}px` }}
                className={item.isSubtotal ? "text-muted-foreground" : ""}
              >
                {item.name}
              </span>
            </div>
            {!item.isSubtotal && (
              <>
                <div className="col-span-3 text-right font-mono">
                  {formatCashFlowAmount(item.currentAmount)}
                </div>
                {showComparison && (
                  <>
                    <div className="col-span-3 text-right font-mono text-muted-foreground">
                      {item.previousAmount !== undefined
                        ? formatCashFlowAmount(item.previousAmount)
                        : "-"}
                    </div>
                    <div className="col-span-1 text-right">
                      {item.previousAmount !== undefined &&
                        renderVarianceIndicator(item.currentAmount, item.previousAmount)}
                    </div>
                  </>
                )}
                {!showComparison && <div className="col-span-4" />}
              </>
            )}
            {item.isSubtotal && (
              <>
                <div className="col-span-3" />
                {showComparison && <div className="col-span-4" />}
                {!showComparison && <div className="col-span-4" />}
              </>
            )}
          </div>
        ))}
      </div>

      {/* Section Total */}
      <div className="grid grid-cols-12 gap-2 py-2 px-2 mt-3 bg-muted/50 rounded font-semibold">
        <div className="col-span-5">{section.title} 소계</div>
        <div className="col-span-3 text-right font-mono">
          {formatCashFlowAmount(section.total)}
        </div>
        {showComparison && (
          <>
            <div className="col-span-3 text-right font-mono text-muted-foreground">
              {section.previousTotal !== undefined
                ? formatCashFlowAmount(section.previousTotal)
                : "-"}
            </div>
            <div className="col-span-1 text-right">
              {section.previousTotal !== undefined &&
                renderVarianceIndicator(section.total, section.previousTotal)}
            </div>
          </>
        )}
        {!showComparison && <div className="col-span-4" />}
      </div>
    </div>
  );

  const renderSummaryRow = (
    label: string,
    current: number,
    previous?: number,
    className = ""
  ) => (
    <div
      className={`grid grid-cols-12 gap-2 py-2 px-2 rounded font-semibold ${className}`}
    >
      <div className="col-span-5">{label}</div>
      <div className="col-span-3 text-right font-mono">
        {formatCashFlowAmount(current)}
      </div>
      {showComparison && (
        <>
          <div className="col-span-3 text-right font-mono text-muted-foreground">
            {previous !== undefined ? formatCashFlowAmount(previous) : "-"}
          </div>
          <div className="col-span-1 text-right">
            {previous !== undefined && renderVarianceIndicator(current, previous)}
          </div>
        </>
      )}
      {!showComparison && <div className="col-span-4" />}
    </div>
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">현금흐름표</h1>
          <p className="text-muted-foreground">
            현금의 유입과 유출 현황을 K-IFRS 기준으로 보여줍니다.
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
            </div>
          </div>
        </CardContent>
      </Card>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">영업활동 현금흐름</p>
                <p className={`text-xl font-bold font-mono ${
                  cashFlowData.operatingActivities.total >= 0 ? "text-green-600" : "text-red-600"
                }`}>
                  {formatCashFlowAmount(cashFlowData.operatingActivities.total)}
                </p>
              </div>
              {cashFlowData.operatingActivities.previousTotal !== undefined &&
                renderVarianceIndicator(
                  cashFlowData.operatingActivities.total,
                  cashFlowData.operatingActivities.previousTotal
                )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">투자활동 현금흐름</p>
                <p className={`text-xl font-bold font-mono ${
                  cashFlowData.investingActivities.total >= 0 ? "text-green-600" : "text-red-600"
                }`}>
                  {formatCashFlowAmount(cashFlowData.investingActivities.total)}
                </p>
              </div>
              {cashFlowData.investingActivities.previousTotal !== undefined &&
                renderVarianceIndicator(
                  cashFlowData.investingActivities.total,
                  cashFlowData.investingActivities.previousTotal
                )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">재무활동 현금흐름</p>
                <p className={`text-xl font-bold font-mono ${
                  cashFlowData.financingActivities.total >= 0 ? "text-green-600" : "text-red-600"
                }`}>
                  {formatCashFlowAmount(cashFlowData.financingActivities.total)}
                </p>
              </div>
              {cashFlowData.financingActivities.previousTotal !== undefined &&
                renderVarianceIndicator(
                  cashFlowData.financingActivities.total,
                  cashFlowData.financingActivities.previousTotal
                )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">기말 현금</p>
                <p className="text-xl font-bold font-mono text-primary">
                  {formatCashFlowAmount(cashFlowData.endingCash)}
                </p>
              </div>
              {cashFlowData.previousEndingCash !== undefined &&
                renderVarianceIndicator(
                  cashFlowData.endingCash,
                  cashFlowData.previousEndingCash
                )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Cash Flow Statement Content */}
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
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center space-x-2">
              <Banknote className="h-5 w-5" />
              <span>현금흐름표 상세</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {/* Column Headers */}
            <div className="grid grid-cols-12 gap-2 py-2 px-2 bg-muted rounded-lg mb-4 text-xs font-medium text-muted-foreground">
              <div className="col-span-5">항목</div>
              <div className="col-span-3 text-right">당기</div>
              {showComparison && (
                <>
                  <div className="col-span-3 text-right">전기</div>
                  <div className="col-span-1 text-right">증감</div>
                </>
              )}
            </div>

            {/* Operating Activities */}
            {renderSection(cashFlowData.operatingActivities)}

            {/* Investing Activities */}
            <div className="my-6 border-t" />
            {renderSection(cashFlowData.investingActivities)}

            {/* Financing Activities */}
            <div className="my-6 border-t" />
            {renderSection(cashFlowData.financingActivities)}

            {/* Summary Section */}
            <div className="my-6 border-t-2 border-primary" />

            {/* Net Cash Change */}
            {renderSummaryRow(
              "현금및현금성자산의 순증가(감소)",
              cashFlowData.netCashChange,
              cashFlowData.previousNetCashChange,
              cashFlowData.netCashChange >= 0 ? "bg-green-50" : "bg-red-50"
            )}

            {/* Beginning Cash */}
            {renderSummaryRow(
              "기초 현금및현금성자산",
              cashFlowData.beginningCash,
              cashFlowData.previousBeginningCash,
              "bg-muted/30"
            )}

            {/* Ending Cash */}
            <div className="mt-2" />
            {renderSummaryRow(
              "기말 현금및현금성자산",
              cashFlowData.endingCash,
              cashFlowData.previousEndingCash,
              "bg-primary/10 font-bold text-lg"
            )}

            {/* Verification Note */}
            <div className="mt-6 p-4 bg-muted/30 rounded-lg text-sm text-muted-foreground">
              <p className="font-medium mb-1">검증</p>
              <p>
                기초 현금 ({formatCashFlowAmount(cashFlowData.beginningCash)}) +
                순증가 ({formatCashFlowAmount(cashFlowData.netCashChange)}) =
                기말 현금 ({formatCashFlowAmount(cashFlowData.endingCash)})
              </p>
              {cashFlowData.beginningCash + cashFlowData.netCashChange === cashFlowData.endingCash ? (
                <p className="text-green-600 font-medium mt-1">검증 완료</p>
              ) : (
                <p className="text-red-600 font-medium mt-1">검증 실패 - 데이터 확인 필요</p>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
