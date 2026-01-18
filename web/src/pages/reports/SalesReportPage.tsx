import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Download,
  Printer,
  TrendingUp,
  Minus,
  BarChart3,
  Users,
  Package,
  DollarSign,
  ShoppingCart,
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
  Badge,
} from "@/components/ui";
import { DateRangePicker } from "@/components/common";
import { formatCurrency, cn } from "@/lib/utils";
import { reportsApi } from "@/api/reports";
import type { SalesReportData, SalesTrendItem } from "@/api/reports";

// Mock data for development
const mockSalesReportData: SalesReportData = {
  period: {
    startDate: "2024-01-01",
    endDate: "2024-01-31",
  },
  summary: {
    totalSales: 156000000,
    previousTotalSales: 142000000,
    transactionCount: 245,
    previousTransactionCount: 220,
    averageTransaction: 636735,
    previousAverageTransaction: 645455,
  },
  trend: [
    { date: "2024-01-01", amount: 4500000, count: 8 },
    { date: "2024-01-02", amount: 5200000, count: 9 },
    { date: "2024-01-03", amount: 4800000, count: 7 },
    { date: "2024-01-04", amount: 6100000, count: 11 },
    { date: "2024-01-05", amount: 5500000, count: 10 },
    { date: "2024-01-06", amount: 3200000, count: 5 },
    { date: "2024-01-07", amount: 2800000, count: 4 },
    { date: "2024-01-08", amount: 5800000, count: 9 },
    { date: "2024-01-09", amount: 6200000, count: 12 },
    { date: "2024-01-10", amount: 5900000, count: 10 },
    { date: "2024-01-11", amount: 6500000, count: 11 },
    { date: "2024-01-12", amount: 5100000, count: 8 },
    { date: "2024-01-13", amount: 3500000, count: 6 },
    { date: "2024-01-14", amount: 2900000, count: 5 },
    { date: "2024-01-15", amount: 5600000, count: 9 },
    { date: "2024-01-16", amount: 6800000, count: 13 },
    { date: "2024-01-17", amount: 5400000, count: 10 },
    { date: "2024-01-18", amount: 6000000, count: 11 },
    { date: "2024-01-19", amount: 5700000, count: 9 },
    { date: "2024-01-20", amount: 3600000, count: 6 },
    { date: "2024-01-21", amount: 3100000, count: 5 },
    { date: "2024-01-22", amount: 5900000, count: 10 },
    { date: "2024-01-23", amount: 7200000, count: 14 },
    { date: "2024-01-24", amount: 6300000, count: 12 },
    { date: "2024-01-25", amount: 5800000, count: 10 },
    { date: "2024-01-26", amount: 6100000, count: 11 },
    { date: "2024-01-27", amount: 3800000, count: 6 },
    { date: "2024-01-28", amount: 3200000, count: 5 },
    { date: "2024-01-29", amount: 6400000, count: 11 },
    { date: "2024-01-30", amount: 6900000, count: 13 },
    { date: "2024-01-31", amount: 5500000, count: 10 },
  ],
  byPartner: [
    { partnerId: "p1", partnerName: "삼성전자", businessNumber: "123-45-67890", totalAmount: 32000000, transactionCount: 45, percentage: 20.5 },
    { partnerId: "p2", partnerName: "LG전자", businessNumber: "234-56-78901", totalAmount: 28000000, transactionCount: 38, percentage: 17.9 },
    { partnerId: "p3", partnerName: "현대자동차", businessNumber: "345-67-89012", totalAmount: 24000000, transactionCount: 32, percentage: 15.4 },
    { partnerId: "p4", partnerName: "SK하이닉스", businessNumber: "456-78-90123", totalAmount: 18000000, transactionCount: 28, percentage: 11.5 },
    { partnerId: "p5", partnerName: "네이버", businessNumber: "567-89-01234", totalAmount: 15000000, transactionCount: 22, percentage: 9.6 },
    { partnerId: "p6", partnerName: "카카오", businessNumber: "678-90-12345", totalAmount: 12000000, transactionCount: 18, percentage: 7.7 },
    { partnerId: "p7", partnerName: "쿠팡", businessNumber: "789-01-23456", totalAmount: 10000000, transactionCount: 20, percentage: 6.4 },
    { partnerId: "p8", partnerName: "배달의민족", businessNumber: "890-12-34567", totalAmount: 8500000, transactionCount: 15, percentage: 5.4 },
    { partnerId: "p9", partnerName: "토스", businessNumber: "901-23-45678", totalAmount: 5000000, transactionCount: 12, percentage: 3.2 },
    { partnerId: "p10", partnerName: "기타", businessNumber: "-", totalAmount: 3500000, transactionCount: 15, percentage: 2.4 },
  ],
  byProduct: [
    { productId: "prod1", productCode: "EL-001", productName: "전자부품 A", category: "전자", quantity: 1200, totalAmount: 36000000, percentage: 23.1 },
    { productId: "prod2", productCode: "EL-002", productName: "전자부품 B", category: "전자", quantity: 950, totalAmount: 28500000, percentage: 18.3 },
    { productId: "prod3", productCode: "ME-001", productName: "기계부품 A", category: "기계", quantity: 520, totalAmount: 26000000, percentage: 16.7 },
    { productId: "prod4", productCode: "SW-001", productName: "소프트웨어 라이선스", category: "소프트웨어", quantity: 85, totalAmount: 21250000, percentage: 13.6 },
    { productId: "prod5", productCode: "SV-001", productName: "컨설팅 서비스", category: "서비스", quantity: 42, totalAmount: 16800000, percentage: 10.8 },
    { productId: "prod6", productCode: "ME-002", productName: "기계부품 B", category: "기계", quantity: 380, totalAmount: 11400000, percentage: 7.3 },
    { productId: "prod7", productCode: "EL-003", productName: "전자부품 C", category: "전자", quantity: 420, totalAmount: 8400000, percentage: 5.4 },
    { productId: "prod8", productCode: "SV-002", productName: "유지보수 서비스", category: "서비스", quantity: 120, totalAmount: 7650000, percentage: 4.8 },
  ],
};

// Simple bar chart component using CSS
function SimpleBarChart({ data, maxValue }: { data: SalesTrendItem[]; maxValue: number }) {
  return (
    <div className="flex items-end gap-0.5 h-40">
      {data.map((item, index) => {
        const height = maxValue > 0 ? (item.amount / maxValue) * 100 : 0;
        const date = new Date(item.date);
        const isWeekend = date.getDay() === 0 || date.getDay() === 6;

        return (
          <div
            key={item.date}
            className="flex-1 flex flex-col items-center group relative"
          >
            <div
              className={cn(
                "w-full rounded-t transition-all cursor-pointer",
                isWeekend ? "bg-muted-foreground/30" : "bg-primary/70",
                "hover:bg-primary"
              )}
              style={{ height: `${height}%`, minHeight: height > 0 ? "4px" : "0" }}
            />
            {/* Tooltip */}
            <div className="absolute bottom-full mb-2 hidden group-hover:block z-10">
              <div className="bg-popover border rounded-lg shadow-lg p-2 text-xs whitespace-nowrap">
                <p className="font-medium">{date.toLocaleDateString("ko-KR", { month: "short", day: "numeric" })}</p>
                <p className="text-primary font-mono">{formatCurrency(item.amount, { compact: true })}</p>
                <p className="text-muted-foreground">{item.count}건</p>
              </div>
            </div>
            {/* X-axis label for every 5th day */}
            {(index + 1) % 5 === 0 && (
              <span className="text-[10px] text-muted-foreground mt-1 rotate-0">
                {date.getDate()}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}

// Progress bar for rankings
function RankingBar({ percentage, color = "primary" }: { percentage: number; color?: string }) {
  return (
    <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
      <div
        className={cn("h-full rounded-full transition-all", `bg-${color}`)}
        style={{ width: `${Math.min(percentage, 100)}%`, backgroundColor: color === "primary" ? undefined : color }}
      />
    </div>
  );
}

export function SalesReportPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [groupBy, setGroupBy] = useState<"day" | "week" | "month">("day");
  const [activeTab, setActiveTab] = useState<"trend" | "partner" | "product">("trend");

  // Fetch sales report data
  const {
    data: salesReportResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["reports", "sales", startDate, endDate, groupBy],
    queryFn: () =>
      reportsApi.getSalesReport({
        startDate,
        endDate,
        groupBy,
      }),
    enabled: !!startDate && !!endDate,
  });

  // Use mock data if no real data
  const salesData = salesReportResponse?.data || mockSalesReportData;

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  // Calculate variance percentages
  const calculateVariance = (current: number, previous?: number) => {
    if (!previous || previous === 0) return null;
    return ((current - previous) / previous) * 100;
  };

  const renderVarianceIndicator = (current: number, previous?: number, inverse = false) => {
    const variance = calculateVariance(current, previous);
    if (variance === null) return null;

    const isPositive = inverse ? variance < 0 : variance > 0;
    const isNegative = inverse ? variance > 0 : variance < 0;

    if (isPositive) {
      return (
        <span className="flex items-center text-green-600 text-xs">
          <ArrowUpRight className="h-3 w-3 mr-0.5" />
          {Math.abs(variance).toFixed(1)}%
        </span>
      );
    } else if (isNegative) {
      return (
        <span className="flex items-center text-red-600 text-xs">
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
    return Math.max(...salesData.trend.map((item) => item.amount), 1);
  }, [salesData.trend]);

  // Colors for product categories
  const categoryColors: Record<string, string> = {
    "전자": "#3b82f6",
    "기계": "#10b981",
    "소프트웨어": "#8b5cf6",
    "서비스": "#f59e0b",
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">매출 분석</h1>
          <p className="text-muted-foreground">
            기간별 매출 현황과 거래처/품목별 분석을 확인합니다.
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
          </div>
        </CardContent>
      </Card>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">총 매출액</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {formatCurrency(salesData.summary.totalSales, { compact: true })}
                </p>
                <div className="mt-1">
                  {renderVarianceIndicator(
                    salesData.summary.totalSales,
                    salesData.summary.previousTotalSales
                  )}
                </div>
              </div>
              <div className="p-2 bg-primary/10 rounded-lg">
                <DollarSign className="h-5 w-5 text-primary" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">거래 건수</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {salesData.summary.transactionCount.toLocaleString()}건
                </p>
                <div className="mt-1">
                  {renderVarianceIndicator(
                    salesData.summary.transactionCount,
                    salesData.summary.previousTransactionCount
                  )}
                </div>
              </div>
              <div className="p-2 bg-green-100 rounded-lg">
                <ShoppingCart className="h-5 w-5 text-green-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">평균 거래금액</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {formatCurrency(salesData.summary.averageTransaction, { compact: true })}
                </p>
                <div className="mt-1">
                  {renderVarianceIndicator(
                    salesData.summary.averageTransaction,
                    salesData.summary.previousAverageTransaction
                  )}
                </div>
              </div>
              <div className="p-2 bg-blue-100 rounded-lg">
                <BarChart3 className="h-5 w-5 text-blue-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">거래처 수</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {salesData.byPartner.length}개
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  상위 3개사: {(salesData.byPartner.slice(0, 3).reduce((sum, p) => sum + p.percentage, 0)).toFixed(1)}%
                </p>
              </div>
              <div className="p-2 bg-purple-100 rounded-lg">
                <Users className="h-5 w-5 text-purple-600" />
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
              onClick={() => setActiveTab("trend")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "trend"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <TrendingUp className="h-4 w-4 inline mr-2" />
              매출 추이
            </button>
            <button
              onClick={() => setActiveTab("partner")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "partner"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <Users className="h-4 w-4 inline mr-2" />
              거래처별 매출
            </button>
            <button
              onClick={() => setActiveTab("product")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "product"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <Package className="h-4 w-4 inline mr-2" />
              품목별 매출
            </button>
          </div>

          {/* Trend Tab */}
          {activeTab === "trend" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">매출 추이 차트</CardTitle>
              </CardHeader>
              <CardContent>
                <SimpleBarChart data={salesData.trend} maxValue={maxTrendValue} />
                <div className="flex items-center justify-center gap-6 mt-4 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <span className="w-3 h-3 rounded bg-primary/70" />
                    평일
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-3 h-3 rounded bg-muted-foreground/30" />
                    주말
                  </span>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Partner Tab */}
          {activeTab === "partner" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">거래처별 매출 순위</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {salesData.byPartner.map((partner, index) => (
                    <div key={partner.partnerId} className="flex items-center gap-4">
                      <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center font-semibold text-sm">
                        {index + 1}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between mb-1">
                          <div className="flex items-center gap-2">
                            <span className="font-medium truncate">{partner.partnerName}</span>
                            {partner.businessNumber !== "-" && (
                              <span className="text-xs text-muted-foreground">
                                ({partner.businessNumber})
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-4">
                            <span className="text-sm text-muted-foreground">
                              {partner.transactionCount}건
                            </span>
                            <span className="font-mono font-medium">
                              {formatCurrency(partner.totalAmount, { showSymbol: false })}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex-1">
                            <RankingBar percentage={partner.percentage * 5} />
                          </div>
                          <span className="text-xs text-muted-foreground w-12 text-right">
                            {partner.percentage.toFixed(1)}%
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Product Tab */}
          {activeTab === "product" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">품목별 매출 분석</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {salesData.byProduct.map((product, index) => (
                    <div key={product.productId} className="flex items-center gap-4">
                      <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center font-semibold text-sm">
                        {index + 1}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between mb-1">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-xs text-muted-foreground">
                              {product.productCode}
                            </span>
                            <span className="font-medium truncate">{product.productName}</span>
                            <Badge variant="outline" className="text-xs">
                              {product.category}
                            </Badge>
                          </div>
                          <div className="flex items-center gap-4">
                            <span className="text-sm text-muted-foreground">
                              {product.quantity.toLocaleString()}개
                            </span>
                            <span className="font-mono font-medium">
                              {formatCurrency(product.totalAmount, { showSymbol: false })}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden">
                            <div
                              className="h-full rounded-full transition-all"
                              style={{
                                width: `${product.percentage * 4}%`,
                                backgroundColor: categoryColors[product.category] || "#6b7280",
                              }}
                            />
                          </div>
                          <span className="text-xs text-muted-foreground w-12 text-right">
                            {product.percentage.toFixed(1)}%
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                {/* Category Summary */}
                <div className="mt-6 pt-6 border-t">
                  <h4 className="text-sm font-medium mb-3">카테고리별 요약</h4>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    {Object.entries(
                      salesData.byProduct.reduce((acc, product) => {
                        if (!acc[product.category]) {
                          acc[product.category] = { amount: 0, count: 0 };
                        }
                        acc[product.category].amount += product.totalAmount;
                        acc[product.category].count += product.quantity;
                        return acc;
                      }, {} as Record<string, { amount: number; count: number }>)
                    ).map(([category, data]) => (
                      <div
                        key={category}
                        className="p-3 rounded-lg border"
                        style={{ borderColor: categoryColors[category] || "#e5e7eb" }}
                      >
                        <div className="flex items-center gap-2 mb-1">
                          <span
                            className="w-2 h-2 rounded-full"
                            style={{ backgroundColor: categoryColors[category] || "#6b7280" }}
                          />
                          <span className="text-sm font-medium">{category}</span>
                        </div>
                        <p className="font-mono text-lg font-semibold">
                          {formatCurrency(data.amount, { compact: true })}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {data.count.toLocaleString()}개 판매
                        </p>
                      </div>
                    ))}
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
