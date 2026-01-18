import { useState } from "react";
import { Download, Printer, Calendar } from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui";
import { formatCurrency } from "@/lib/utils";

// Equity component types according to K-IFRS
interface EquityComponent {
  id: string;
  name: string;
  nameEn: string;
}

// Change row types
interface ChangeRow {
  id: string;
  label: string;
  labelEn: string;
  values: Record<string, number>;
  isSubtotal?: boolean;
  isBold?: boolean;
}

// Mock data for development
const equityComponents: EquityComponent[] = [
  { id: "capital_stock", name: "자본금", nameEn: "Capital Stock" },
  { id: "capital_surplus", name: "자본잉여금", nameEn: "Capital Surplus" },
  { id: "retained_earnings", name: "이익잉여금", nameEn: "Retained Earnings" },
  { id: "aoci", name: "기타포괄손익누계액", nameEn: "Accumulated OCI" },
  { id: "other_capital", name: "기타자본", nameEn: "Other Capital" },
  { id: "total", name: "총계", nameEn: "Total" },
];

const mockChangeRows: ChangeRow[] = [
  {
    id: "opening_balance",
    label: "기초잔액",
    labelEn: "Opening Balance",
    values: {
      capital_stock: 100000000,
      capital_surplus: 50000000,
      retained_earnings: 80000000,
      aoci: 5000000,
      other_capital: -10000000,
      total: 225000000,
    },
    isBold: true,
  },
  {
    id: "net_income",
    label: "당기순이익",
    labelEn: "Net Income",
    values: {
      capital_stock: 0,
      capital_surplus: 0,
      retained_earnings: 35000000,
      aoci: 0,
      other_capital: 0,
      total: 35000000,
    },
  },
  {
    id: "oci_changes",
    label: "기타포괄손익",
    labelEn: "Other Comprehensive Income",
    values: {
      capital_stock: 0,
      capital_surplus: 0,
      retained_earnings: 0,
      aoci: 2000000,
      other_capital: 0,
      total: 2000000,
    },
  },
  {
    id: "total_comprehensive_income",
    label: "총포괄손익",
    labelEn: "Total Comprehensive Income",
    values: {
      capital_stock: 0,
      capital_surplus: 0,
      retained_earnings: 35000000,
      aoci: 2000000,
      other_capital: 0,
      total: 37000000,
    },
    isSubtotal: true,
  },
  {
    id: "dividends",
    label: "배당금",
    labelEn: "Dividends",
    values: {
      capital_stock: 0,
      capital_surplus: 0,
      retained_earnings: -15000000,
      aoci: 0,
      other_capital: 0,
      total: -15000000,
    },
  },
  {
    id: "stock_issuance",
    label: "주식발행",
    labelEn: "Stock Issuance",
    values: {
      capital_stock: 20000000,
      capital_surplus: 10000000,
      retained_earnings: 0,
      aoci: 0,
      other_capital: 0,
      total: 30000000,
    },
  },
  {
    id: "treasury_stock",
    label: "자기주식 취득",
    labelEn: "Treasury Stock Acquisition",
    values: {
      capital_stock: 0,
      capital_surplus: 0,
      retained_earnings: 0,
      aoci: 0,
      other_capital: -5000000,
      total: -5000000,
    },
  },
  {
    id: "other_changes",
    label: "기타 증감",
    labelEn: "Other Changes",
    values: {
      capital_stock: 0,
      capital_surplus: 0,
      retained_earnings: -1000000,
      aoci: 500000,
      other_capital: 0,
      total: -500000,
    },
  },
  {
    id: "total_changes",
    label: "자본 변동 합계",
    labelEn: "Total Changes in Equity",
    values: {
      capital_stock: 20000000,
      capital_surplus: 10000000,
      retained_earnings: 19000000,
      aoci: 2500000,
      other_capital: -5000000,
      total: 46500000,
    },
    isSubtotal: true,
  },
  {
    id: "closing_balance",
    label: "기말잔액",
    labelEn: "Closing Balance",
    values: {
      capital_stock: 120000000,
      capital_surplus: 60000000,
      retained_earnings: 99000000,
      aoci: 7500000,
      other_capital: -15000000,
      total: 271500000,
    },
    isBold: true,
  },
];

export function EquityChangesStatementPage() {
  const today = new Date();
  const firstDayOfYear = new Date(today.getFullYear(), 0, 1);

  const [startDate, setStartDate] = useState(
    firstDayOfYear.toISOString().split("T")[0]
  );
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);

  // Format date for display
  const formatPeriod = () => {
    const start = new Date(startDate);
    const end = new Date(endDate);
    return `${start.getFullYear()}년 ${start.getMonth() + 1}월 ${start.getDate()}일 ~ ${end.getFullYear()}년 ${end.getMonth() + 1}월 ${end.getDate()}일`;
  };

  // Render cell value with proper formatting
  const renderCellValue = (value: number) => {
    if (value === 0) {
      return <span className="text-muted-foreground">-</span>;
    }

    const isNegative = value < 0;
    const formattedValue = formatCurrency(Math.abs(value), { showSymbol: false });

    return (
      <span className={isNegative ? "text-red-600" : ""}>
        {isNegative ? `(${formattedValue})` : formattedValue}
      </span>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">자본변동표</h1>
          <p className="text-muted-foreground">
            자본의 변동 내역을 보여줍니다.
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
            <div>
              <label className="block text-sm font-medium mb-1.5">시작일</label>
              <div className="relative">
                <Calendar className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="pl-9 w-40"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">종료일</label>
              <div className="relative">
                <Calendar className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  className="pl-9 w-40"
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Statement Content */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">
            자본변동표
            <span className="text-sm font-normal text-muted-foreground ml-2">
              ({formatPeriod()})
            </span>
          </CardTitle>
          <p className="text-sm text-muted-foreground">
            (단위: 원)
          </p>
        </CardHeader>
        <CardContent>
          {/* Matrix Table */}
          <div className="overflow-x-auto">
            <table className="w-full border-collapse min-w-[900px]">
              {/* Header Row */}
              <thead>
                <tr className="bg-muted">
                  <th className="border border-border px-4 py-3 text-left font-semibold text-sm min-w-[180px]">
                    구분
                  </th>
                  {equityComponents.map((component) => (
                    <th
                      key={component.id}
                      className={`border border-border px-4 py-3 text-right font-semibold text-sm min-w-[130px] ${
                        component.id === "total" ? "bg-primary/10" : ""
                      }`}
                    >
                      <div>{component.name}</div>
                      <div className="text-xs font-normal text-muted-foreground">
                        {component.nameEn}
                      </div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {mockChangeRows.map((row) => (
                  <tr
                    key={row.id}
                    className={`
                      ${row.isSubtotal ? "bg-muted/50" : ""}
                      ${row.isBold ? "bg-primary/5" : ""}
                      hover:bg-muted/30 transition-colors
                    `}
                  >
                    <td
                      className={`border border-border px-4 py-2.5 text-sm ${
                        row.isBold || row.isSubtotal ? "font-semibold" : ""
                      }`}
                    >
                      <div>{row.label}</div>
                      <div className="text-xs text-muted-foreground">
                        {row.labelEn}
                      </div>
                    </td>
                    {equityComponents.map((component) => (
                      <td
                        key={`${row.id}-${component.id}`}
                        className={`border border-border px-4 py-2.5 text-right font-mono text-sm ${
                          row.isBold || row.isSubtotal ? "font-semibold" : ""
                        } ${component.id === "total" ? "bg-primary/5" : ""}`}
                      >
                        {renderCellValue(row.values[component.id] || 0)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Notes */}
          <div className="mt-6 p-4 bg-muted/30 rounded-lg">
            <h4 className="font-semibold text-sm mb-2">주석</h4>
            <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside">
              <li>
                기타포괄손익누계액은 매도가능금융자산평가손익, 해외사업환산손익
                등을 포함합니다.
              </li>
              <li>
                기타자본에는 자기주식, 주식선택권 등이 포함됩니다.
              </li>
              <li>
                본 자본변동표는 K-IFRS(한국채택국제회계기준)에 따라 작성되었습니다.
              </li>
            </ul>
          </div>
        </CardContent>
      </Card>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="text-sm text-muted-foreground">기초 자본 총계</div>
            <div className="text-2xl font-bold font-mono mt-1">
              {formatCurrency(mockChangeRows[0].values.total, { showSymbol: false })}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-sm text-muted-foreground">당기 자본 변동</div>
            <div className="text-2xl font-bold font-mono mt-1 text-green-600">
              +{formatCurrency(
                mockChangeRows.find((r) => r.id === "total_changes")?.values.total || 0,
                { showSymbol: false }
              )}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-sm text-muted-foreground">기말 자본 총계</div>
            <div className="text-2xl font-bold font-mono mt-1 text-primary">
              {formatCurrency(
                mockChangeRows.find((r) => r.id === "closing_balance")?.values.total || 0,
                { showSymbol: false }
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
