import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Download,
  Printer,
  Users,
  UserPlus,
  UserMinus,
  Building2,
  Briefcase,
  Clock,
  Banknote,
  TrendingUp,
  TrendingDown,
  AlertCircle,
  CalendarDays,
} from "lucide-react";
import {
  Button,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
} from "@/components/ui";
import { DateRangePicker } from "@/components/common";
import { formatCurrency, cn } from "@/lib/utils";
import { reportsApi } from "@/api/reports";
import type { HRReportData, DepartmentHeadcount, HiringTrendItem } from "@/api/reports";

// Mock data for development
const mockHRReportData: HRReportData = {
  asOfDate: "2024-01-31",
  headcount: {
    total: 90,
    regular: 72,
    contract: 12,
    partTime: 6,
    newHires: 5,
    resignations: 2,
    turnoverRate: 2.2,
  },
  byDepartment: [
    { departmentId: "d1", departmentName: "개발부", headcount: 30, regularCount: 25, contractCount: 4, partTimeCount: 1 },
    { departmentId: "d2", departmentName: "영업부", headcount: 25, regularCount: 20, contractCount: 3, partTimeCount: 2 },
    { departmentId: "d3", departmentName: "마케팅부", headcount: 15, regularCount: 12, contractCount: 2, partTimeCount: 1 },
    { departmentId: "d4", departmentName: "경영지원부", headcount: 10, regularCount: 8, contractCount: 1, partTimeCount: 1 },
    { departmentId: "d5", departmentName: "인사부", headcount: 5, regularCount: 4, contractCount: 1, partTimeCount: 0 },
    { departmentId: "d6", departmentName: "재무부", headcount: 5, regularCount: 3, contractCount: 1, partTimeCount: 1 },
  ],
  byPosition: [
    { positionId: "pos1", positionName: "사원", headcount: 35, averageTenure: 1.5 },
    { positionId: "pos2", positionName: "대리", headcount: 25, averageTenure: 3.2 },
    { positionId: "pos3", positionName: "과장", headcount: 15, averageTenure: 5.8 },
    { positionId: "pos4", positionName: "차장", headcount: 8, averageTenure: 8.5 },
    { positionId: "pos5", positionName: "부장", headcount: 5, averageTenure: 12.3 },
    { positionId: "pos6", positionName: "임원", headcount: 2, averageTenure: 15.0 },
  ],
  hiringTrend: [
    { date: "2023-08", hired: 3, resigned: 1, netChange: 2 },
    { date: "2023-09", hired: 4, resigned: 2, netChange: 2 },
    { date: "2023-10", hired: 2, resigned: 1, netChange: 1 },
    { date: "2023-11", hired: 6, resigned: 3, netChange: 3 },
    { date: "2023-12", hired: 1, resigned: 4, netChange: -3 },
    { date: "2024-01", hired: 5, resigned: 2, netChange: 3 },
  ],
  payroll: {
    totalPayroll: 540000000,
    averageSalary: 6000000,
    medianSalary: 5500000,
    minSalary: 3200000,
    maxSalary: 15000000,
  },
  attendance: {
    workingDays: 22,
    averageAttendance: 95.2,
    lateCount: 45,
    earlyLeaveCount: 23,
    absenceCount: 12,
    overtimeHours: 450,
  },
};

// Simple bar chart for hiring trend
function HiringTrendChart({ data }: { data: HiringTrendItem[] }) {
  const maxValue = Math.max(
    ...data.map((d) => Math.max(d.hired, d.resigned)),
    1
  );

  return (
    <div className="space-y-3">
      {data.map((item) => (
        <div key={item.date} className="flex items-center gap-4">
          <div className="w-16 text-sm text-muted-foreground">
            {item.date}
          </div>
          <div className="flex-1 flex gap-2">
            {/* Hired bar */}
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <div
                  className="bg-green-500 h-6 rounded transition-all"
                  style={{ width: `${(item.hired / maxValue) * 100}%`, minWidth: item.hired > 0 ? "20px" : "0" }}
                />
                <span className="text-sm text-green-600">+{item.hired}</span>
              </div>
            </div>
            {/* Resigned bar */}
            <div className="flex-1">
              <div className="flex items-center gap-2 justify-end">
                <span className="text-sm text-red-600">-{item.resigned}</span>
                <div
                  className="bg-red-500 h-6 rounded transition-all"
                  style={{ width: `${(item.resigned / maxValue) * 100}%`, minWidth: item.resigned > 0 ? "20px" : "0" }}
                />
              </div>
            </div>
          </div>
          <div className={cn(
            "w-12 text-right text-sm font-medium",
            item.netChange > 0 ? "text-green-600" : item.netChange < 0 ? "text-red-600" : "text-muted-foreground"
          )}>
            {item.netChange > 0 ? "+" : ""}{item.netChange}
          </div>
        </div>
      ))}
    </div>
  );
}

// Donut chart for employee composition
function EmployeeCompositionChart({ regular, contract, partTime }: { regular: number; contract: number; partTime: number }) {
  const total = regular + contract + partTime;
  const regularPct = (regular / total) * 100;
  const contractPct = (contract / total) * 100;

  // Calculate stroke-dasharray values
  const circumference = 2 * Math.PI * 40;
  const regularDash = (regularPct / 100) * circumference;
  const contractDash = (contractPct / 100) * circumference;

  return (
    <div className="flex items-center gap-6">
      <div className="relative">
        <svg width="120" height="120" viewBox="0 0 100 100">
          {/* Background circle */}
          <circle
            cx="50"
            cy="50"
            r="40"
            fill="none"
            stroke="#f3f4f6"
            strokeWidth="12"
          />
          {/* Regular employees */}
          <circle
            cx="50"
            cy="50"
            r="40"
            fill="none"
            stroke="#3b82f6"
            strokeWidth="12"
            strokeDasharray={`${regularDash} ${circumference}`}
            strokeDashoffset="0"
            transform="rotate(-90 50 50)"
          />
          {/* Contract employees */}
          <circle
            cx="50"
            cy="50"
            r="40"
            fill="none"
            stroke="#10b981"
            strokeWidth="12"
            strokeDasharray={`${contractDash} ${circumference}`}
            strokeDashoffset={-regularDash}
            transform="rotate(-90 50 50)"
          />
          {/* Part-time employees (remaining) */}
          <circle
            cx="50"
            cy="50"
            r="40"
            fill="none"
            stroke="#f59e0b"
            strokeWidth="12"
            strokeDasharray={`${circumference - regularDash - contractDash} ${circumference}`}
            strokeDashoffset={-(regularDash + contractDash)}
            transform="rotate(-90 50 50)"
          />
          {/* Center text */}
          <text x="50" y="46" textAnchor="middle" className="text-2xl font-bold fill-foreground">
            {total}
          </text>
          <text x="50" y="60" textAnchor="middle" className="text-xs fill-muted-foreground">
            명
          </text>
        </svg>
      </div>
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 rounded-full bg-blue-500" />
          <span className="text-sm">정규직 {regular}명 ({regularPct.toFixed(1)}%)</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 rounded-full bg-green-500" />
          <span className="text-sm">계약직 {contract}명 ({contractPct.toFixed(1)}%)</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 rounded-full bg-amber-500" />
          <span className="text-sm">파트타임 {partTime}명 ({((partTime / total) * 100).toFixed(1)}%)</span>
        </div>
      </div>
    </div>
  );
}

// Department headcount horizontal bar
function DepartmentBar({ data, maxHeadcount }: { data: DepartmentHeadcount; maxHeadcount: number }) {
  const percentage = (data.headcount / maxHeadcount) * 100;

  return (
    <div className="flex items-center gap-4 py-2">
      <div className="w-24 text-sm font-medium truncate">{data.departmentName}</div>
      <div className="flex-1">
        <div className="relative h-8 bg-muted rounded overflow-hidden">
          <div
            className="absolute inset-y-0 left-0 bg-primary/20 rounded"
            style={{ width: `${percentage}%` }}
          />
          <div className="absolute inset-0 flex items-center">
            <div
              className="h-full flex items-center"
              style={{ width: `${(data.regularCount / maxHeadcount) * 100}%` }}
            >
              <div className="h-full bg-blue-500 rounded-l" style={{ width: '100%' }} />
            </div>
            <div
              className="h-full flex items-center"
              style={{ width: `${(data.contractCount / maxHeadcount) * 100}%` }}
            >
              <div className="h-full bg-green-500" style={{ width: '100%' }} />
            </div>
            <div
              className="h-full flex items-center"
              style={{ width: `${(data.partTimeCount / maxHeadcount) * 100}%` }}
            >
              <div className="h-full bg-amber-500 rounded-r" style={{ width: '100%' }} />
            </div>
          </div>
        </div>
      </div>
      <div className="w-16 text-right text-sm font-mono">{data.headcount}명</div>
    </div>
  );
}

export function HRReportPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [activeTab, setActiveTab] = useState<"headcount" | "payroll" | "attendance">("headcount");

  // Fetch HR report data
  const {
    data: hrReportResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["reports", "hr", startDate, endDate],
    queryFn: () =>
      reportsApi.getHRReport({
        startDate,
        endDate,
      }),
    enabled: !!startDate && !!endDate,
  });

  // Use mock data if no real data
  const hrData = hrReportResponse?.data || mockHRReportData;

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  // Calculate max headcount for scaling
  const maxDepartmentHeadcount = useMemo(() => {
    return Math.max(...hrData.byDepartment.map((d) => d.headcount), 1);
  }, [hrData.byDepartment]);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">인사 현황</h1>
          <p className="text-muted-foreground">
            인원 현황, 입퇴사, 급여, 근태 통계를 확인합니다.
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
          </div>
        </CardContent>
      </Card>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">전체 인원</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {hrData.headcount.total}명
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  정규직 {hrData.headcount.regular}명
                </p>
              </div>
              <div className="p-2 bg-primary/10 rounded-lg">
                <Users className="h-5 w-5 text-primary" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">신규 입사</p>
                <p className="text-2xl font-bold font-mono mt-1 text-green-600">
                  +{hrData.headcount.newHires}명
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  이번 기간
                </p>
              </div>
              <div className="p-2 bg-green-100 rounded-lg">
                <UserPlus className="h-5 w-5 text-green-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">퇴사</p>
                <p className="text-2xl font-bold font-mono mt-1 text-red-600">
                  -{hrData.headcount.resignations}명
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  이직률 {hrData.headcount.turnoverRate}%
                </p>
              </div>
              <div className="p-2 bg-red-100 rounded-lg">
                <UserMinus className="h-5 w-5 text-red-600" />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm text-muted-foreground">평균 급여</p>
                <p className="text-2xl font-bold font-mono mt-1">
                  {formatCurrency(hrData.payroll.averageSalary, { compact: true })}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  중간값 {formatCurrency(hrData.payroll.medianSalary, { compact: true })}
                </p>
              </div>
              <div className="p-2 bg-amber-100 rounded-lg">
                <Banknote className="h-5 w-5 text-amber-600" />
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
              onClick={() => setActiveTab("headcount")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "headcount"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <Users className="h-4 w-4 inline mr-2" />
              인원 현황
            </button>
            <button
              onClick={() => setActiveTab("payroll")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "payroll"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <Banknote className="h-4 w-4 inline mr-2" />
              급여 통계
            </button>
            <button
              onClick={() => setActiveTab("attendance")}
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                activeTab === "attendance"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              <Clock className="h-4 w-4 inline mr-2" />
              근태 통계
            </button>
          </div>

          {/* Headcount Tab */}
          {activeTab === "headcount" && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Employee Composition */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Users className="h-5 w-5" />
                    고용형태별 인원
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <EmployeeCompositionChart
                    regular={hrData.headcount.regular}
                    contract={hrData.headcount.contract}
                    partTime={hrData.headcount.partTime}
                  />
                </CardContent>
              </Card>

              {/* Hiring Trend */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <TrendingUp className="h-5 w-5" />
                    입퇴사 추이
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <HiringTrendChart data={hrData.hiringTrend} />
                  <div className="flex items-center justify-center gap-6 mt-4 pt-4 border-t text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <span className="w-3 h-3 rounded bg-green-500" />
                      입사
                    </span>
                    <span className="flex items-center gap-1">
                      <span className="w-3 h-3 rounded bg-red-500" />
                      퇴사
                    </span>
                  </div>
                </CardContent>
              </Card>

              {/* Department Headcount */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Building2 className="h-5 w-5" />
                    부서별 인원
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-1">
                    {hrData.byDepartment.map((dept) => (
                      <DepartmentBar
                        key={dept.departmentId}
                        data={dept}
                        maxHeadcount={maxDepartmentHeadcount}
                      />
                    ))}
                  </div>
                  <div className="flex items-center justify-center gap-4 mt-4 pt-4 border-t text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <span className="w-3 h-3 rounded bg-blue-500" />
                      정규직
                    </span>
                    <span className="flex items-center gap-1">
                      <span className="w-3 h-3 rounded bg-green-500" />
                      계약직
                    </span>
                    <span className="flex items-center gap-1">
                      <span className="w-3 h-3 rounded bg-amber-500" />
                      파트타임
                    </span>
                  </div>
                </CardContent>
              </Card>

              {/* Position Distribution */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Briefcase className="h-5 w-5" />
                    직급별 인원
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {hrData.byPosition.map((position) => (
                      <div
                        key={position.positionId}
                        className="flex items-center justify-between py-2 border-b last:border-0"
                      >
                        <div className="flex items-center gap-3">
                          <Badge variant="outline">{position.positionName}</Badge>
                          <span className="font-mono">{position.headcount}명</span>
                        </div>
                        <div className="text-sm text-muted-foreground">
                          평균 근속 {position.averageTenure}년
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {/* Payroll Tab */}
          {activeTab === "payroll" && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Banknote className="h-5 w-5" />
                    급여 통계
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-6">
                    <div className="grid grid-cols-2 gap-4">
                      <div className="p-4 bg-muted/50 rounded-lg">
                        <p className="text-sm text-muted-foreground">총 급여</p>
                        <p className="text-2xl font-bold font-mono mt-1">
                          {formatCurrency(hrData.payroll.totalPayroll, { compact: true })}
                        </p>
                      </div>
                      <div className="p-4 bg-muted/50 rounded-lg">
                        <p className="text-sm text-muted-foreground">평균 급여</p>
                        <p className="text-2xl font-bold font-mono mt-1">
                          {formatCurrency(hrData.payroll.averageSalary, { compact: true })}
                        </p>
                      </div>
                    </div>

                    <div className="space-y-4">
                      <div className="flex items-center justify-between py-3 border-b">
                        <span className="text-muted-foreground">중간값</span>
                        <span className="font-mono font-medium">
                          {formatCurrency(hrData.payroll.medianSalary)}
                        </span>
                      </div>
                      <div className="flex items-center justify-between py-3 border-b">
                        <span className="text-muted-foreground">최저 급여</span>
                        <span className="font-mono font-medium">
                          {formatCurrency(hrData.payroll.minSalary)}
                        </span>
                      </div>
                      <div className="flex items-center justify-between py-3 border-b">
                        <span className="text-muted-foreground">최고 급여</span>
                        <span className="font-mono font-medium">
                          {formatCurrency(hrData.payroll.maxSalary)}
                        </span>
                      </div>
                      <div className="flex items-center justify-between py-3">
                        <span className="text-muted-foreground">급여 범위</span>
                        <span className="font-mono font-medium">
                          {formatCurrency(hrData.payroll.maxSalary - hrData.payroll.minSalary)}
                        </span>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">급여 분포</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {/* Salary distribution visualization */}
                    <div className="relative h-20 bg-muted rounded-lg overflow-hidden">
                      <div className="absolute inset-0 flex items-center px-4">
                        <div className="w-full flex items-center">
                          {/* Min marker */}
                          <div className="relative" style={{ left: "0%" }}>
                            <div className="w-1 h-8 bg-blue-500 rounded" />
                            <div className="absolute top-full mt-1 text-xs text-muted-foreground whitespace-nowrap -translate-x-1/2">
                              {formatCurrency(hrData.payroll.minSalary, { compact: true })}
                            </div>
                          </div>
                          {/* Median marker */}
                          <div
                            className="relative"
                            style={{
                              left: `${((hrData.payroll.medianSalary - hrData.payroll.minSalary) /
                                (hrData.payroll.maxSalary - hrData.payroll.minSalary)) * 100}%`
                            }}
                          >
                            <div className="w-1 h-12 bg-green-500 rounded" />
                            <div className="absolute top-full mt-1 text-xs text-muted-foreground whitespace-nowrap -translate-x-1/2">
                              중간값
                            </div>
                          </div>
                          {/* Average marker */}
                          <div
                            className="relative"
                            style={{
                              left: `${((hrData.payroll.averageSalary - hrData.payroll.minSalary) /
                                (hrData.payroll.maxSalary - hrData.payroll.minSalary)) * 100}%`
                            }}
                          >
                            <div className="w-1 h-12 bg-amber-500 rounded" />
                            <div className="absolute top-full mt-1 text-xs text-muted-foreground whitespace-nowrap -translate-x-1/2">
                              평균
                            </div>
                          </div>
                          {/* Max marker */}
                          <div className="relative ml-auto">
                            <div className="w-1 h-8 bg-red-500 rounded" />
                            <div className="absolute top-full mt-1 text-xs text-muted-foreground whitespace-nowrap -translate-x-1/2">
                              {formatCurrency(hrData.payroll.maxSalary, { compact: true })}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center justify-center gap-6 pt-8 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <span className="w-3 h-3 rounded bg-green-500" />
                        중간값
                      </span>
                      <span className="flex items-center gap-1">
                        <span className="w-3 h-3 rounded bg-amber-500" />
                        평균
                      </span>
                    </div>

                    <div className="p-4 bg-muted/50 rounded-lg mt-6">
                      <p className="text-sm text-muted-foreground mb-2">인당 급여 비용 (월평균)</p>
                      <p className="text-xl font-bold font-mono">
                        {formatCurrency(hrData.payroll.totalPayroll / hrData.headcount.total)}
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {/* Attendance Tab */}
          {activeTab === "attendance" && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Clock className="h-5 w-5" />
                    근태 현황
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-4 bg-muted/50 rounded-lg text-center">
                      <CalendarDays className="h-6 w-6 mx-auto mb-2 text-muted-foreground" />
                      <p className="text-sm text-muted-foreground">근무일수</p>
                      <p className="text-2xl font-bold font-mono mt-1">
                        {hrData.attendance.workingDays}일
                      </p>
                    </div>
                    <div className="p-4 bg-green-50 rounded-lg text-center">
                      <TrendingUp className="h-6 w-6 mx-auto mb-2 text-green-600" />
                      <p className="text-sm text-muted-foreground">평균 출근율</p>
                      <p className="text-2xl font-bold font-mono mt-1 text-green-600">
                        {hrData.attendance.averageAttendance}%
                      </p>
                    </div>
                  </div>

                  <div className="mt-6 space-y-4">
                    <div className="flex items-center justify-between py-3 border-b">
                      <div className="flex items-center gap-2">
                        <AlertCircle className="h-4 w-4 text-amber-500" />
                        <span>지각</span>
                      </div>
                      <span className="font-mono font-medium">{hrData.attendance.lateCount}건</span>
                    </div>
                    <div className="flex items-center justify-between py-3 border-b">
                      <div className="flex items-center gap-2">
                        <TrendingDown className="h-4 w-4 text-orange-500" />
                        <span>조퇴</span>
                      </div>
                      <span className="font-mono font-medium">{hrData.attendance.earlyLeaveCount}건</span>
                    </div>
                    <div className="flex items-center justify-between py-3 border-b">
                      <div className="flex items-center gap-2">
                        <UserMinus className="h-4 w-4 text-red-500" />
                        <span>결근</span>
                      </div>
                      <span className="font-mono font-medium">{hrData.attendance.absenceCount}건</span>
                    </div>
                    <div className="flex items-center justify-between py-3">
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4 text-blue-500" />
                        <span>연장근무</span>
                      </div>
                      <span className="font-mono font-medium">{hrData.attendance.overtimeHours}시간</span>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">근태 요약</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-6">
                    {/* Attendance rate progress */}
                    <div>
                      <div className="flex justify-between text-sm mb-2">
                        <span className="text-muted-foreground">출근율</span>
                        <span className="font-medium">{hrData.attendance.averageAttendance}%</span>
                      </div>
                      <div className="w-full bg-muted rounded-full h-3 overflow-hidden">
                        <div
                          className={cn(
                            "h-full rounded-full transition-all",
                            hrData.attendance.averageAttendance >= 95 ? "bg-green-500" :
                            hrData.attendance.averageAttendance >= 90 ? "bg-amber-500" : "bg-red-500"
                          )}
                          style={{ width: `${hrData.attendance.averageAttendance}%` }}
                        />
                      </div>
                    </div>

                    {/* Issue breakdown */}
                    <div className="grid grid-cols-3 gap-3">
                      <div className="p-3 bg-amber-50 rounded-lg text-center">
                        <p className="text-xs text-muted-foreground">지각률</p>
                        <p className="font-mono font-semibold text-amber-600">
                          {((hrData.attendance.lateCount / (hrData.headcount.total * hrData.attendance.workingDays)) * 100).toFixed(1)}%
                        </p>
                      </div>
                      <div className="p-3 bg-orange-50 rounded-lg text-center">
                        <p className="text-xs text-muted-foreground">조퇴률</p>
                        <p className="font-mono font-semibold text-orange-600">
                          {((hrData.attendance.earlyLeaveCount / (hrData.headcount.total * hrData.attendance.workingDays)) * 100).toFixed(1)}%
                        </p>
                      </div>
                      <div className="p-3 bg-red-50 rounded-lg text-center">
                        <p className="text-xs text-muted-foreground">결근률</p>
                        <p className="font-mono font-semibold text-red-600">
                          {((hrData.attendance.absenceCount / (hrData.headcount.total * hrData.attendance.workingDays)) * 100).toFixed(1)}%
                        </p>
                      </div>
                    </div>

                    {/* Overtime summary */}
                    <div className="p-4 bg-blue-50 rounded-lg">
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="text-sm text-muted-foreground">총 연장근무</p>
                          <p className="text-xl font-bold font-mono mt-1">
                            {hrData.attendance.overtimeHours}시간
                          </p>
                        </div>
                        <div className="text-right">
                          <p className="text-sm text-muted-foreground">인당 평균</p>
                          <p className="font-mono font-medium">
                            {(hrData.attendance.overtimeHours / hrData.headcount.total).toFixed(1)}시간
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </>
      )}
    </div>
  );
}
