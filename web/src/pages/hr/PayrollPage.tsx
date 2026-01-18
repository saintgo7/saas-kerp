import { useState } from "react";
import {
  Search,
  Download,
  Calculator,
  Eye,
  CheckCircle,
  Clock,
  AlertCircle,
  DollarSign,
  Printer,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  Modal,
} from "@/components/ui";
import { formatCurrency } from "@/lib/utils";
import { PAYROLL_STATUS } from "@/constants";
import { toast } from "@/stores/ui";
import type { PayrollStatus } from "@/types";

// Types
interface PayrollEmployee {
  id: string;
  employeeId: string;
  employeeNumber: string;
  name: string;
  department: string;
  position: string;
  baseSalary: number;
  overtimePay: number;
  bonus: number;
  mealAllowance: number;
  transportAllowance: number;
  otherAllowances: number;
  grossPay: number;
  nationalPension: number;
  healthInsurance: number;
  longTermCare: number;
  employmentInsurance: number;
  incomeTax: number;
  localIncomeTax: number;
  otherDeductions: number;
  totalDeductions: number;
  netPay: number;
  status: PayrollStatus;
}

// Mock data
const generateMockPayroll = (): PayrollEmployee[] => [
  {
    id: "1",
    employeeId: "emp1",
    employeeNumber: "EMP001",
    name: "홍길동",
    department: "개발팀",
    position: "팀장",
    baseSalary: 5000000,
    overtimePay: 300000,
    bonus: 0,
    mealAllowance: 200000,
    transportAllowance: 100000,
    otherAllowances: 0,
    grossPay: 5600000,
    nationalPension: 252000,
    healthInsurance: 196560,
    longTermCare: 25149,
    employmentInsurance: 50400,
    incomeTax: 423000,
    localIncomeTax: 42300,
    otherDeductions: 0,
    totalDeductions: 989409,
    netPay: 4610591,
    status: "calculated",
  },
  {
    id: "2",
    employeeId: "emp2",
    employeeNumber: "EMP002",
    name: "김철수",
    department: "영업팀",
    position: "과장",
    baseSalary: 4200000,
    overtimePay: 150000,
    bonus: 500000,
    mealAllowance: 200000,
    transportAllowance: 100000,
    otherAllowances: 0,
    grossPay: 5150000,
    nationalPension: 231750,
    healthInsurance: 180795,
    longTermCare: 23139,
    employmentInsurance: 46350,
    incomeTax: 352000,
    localIncomeTax: 35200,
    otherDeductions: 0,
    totalDeductions: 869234,
    netPay: 4280766,
    status: "approved",
  },
  {
    id: "3",
    employeeId: "emp3",
    employeeNumber: "EMP003",
    name: "이영희",
    department: "인사팀",
    position: "대리",
    baseSalary: 3500000,
    overtimePay: 100000,
    bonus: 0,
    mealAllowance: 200000,
    transportAllowance: 100000,
    otherAllowances: 50000,
    grossPay: 3950000,
    nationalPension: 177750,
    healthInsurance: 138705,
    longTermCare: 17753,
    employmentInsurance: 35550,
    incomeTax: 215000,
    localIncomeTax: 21500,
    otherDeductions: 0,
    totalDeductions: 606258,
    netPay: 3343742,
    status: "paid",
  },
  {
    id: "4",
    employeeId: "emp4",
    employeeNumber: "EMP004",
    name: "박민수",
    department: "회계팀",
    position: "사원",
    baseSalary: 3000000,
    overtimePay: 200000,
    bonus: 0,
    mealAllowance: 200000,
    transportAllowance: 100000,
    otherAllowances: 0,
    grossPay: 3500000,
    nationalPension: 157500,
    healthInsurance: 122850,
    longTermCare: 15724,
    employmentInsurance: 31500,
    incomeTax: 168000,
    localIncomeTax: 16800,
    otherDeductions: 0,
    totalDeductions: 512374,
    netPay: 2987626,
    status: "draft",
  },
  {
    id: "5",
    employeeId: "emp5",
    employeeNumber: "EMP005",
    name: "최지현",
    department: "마케팅팀",
    position: "차장",
    baseSalary: 4500000,
    overtimePay: 250000,
    bonus: 1000000,
    mealAllowance: 200000,
    transportAllowance: 100000,
    otherAllowances: 100000,
    grossPay: 6150000,
    nationalPension: 276750,
    healthInsurance: 215865,
    longTermCare: 27623,
    employmentInsurance: 55350,
    incomeTax: 512000,
    localIncomeTax: 51200,
    otherDeductions: 0,
    totalDeductions: 1138788,
    netPay: 5011212,
    status: "approved",
  },
];

const statusStyles: Record<
  PayrollStatus,
  { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }
> = {
  draft: { variant: "secondary", label: "작성중" },
  calculated: { variant: "warning", label: "계산완료" },
  approved: { variant: "default", label: "승인완료" },
  paid: { variant: "success", label: "지급완료" },
};

/*
 * Insurance rates (2024 Korea) - for reference in future calculations:
 * - nationalPension: 4.5% (employee portion)
 * - healthInsurance: 3.545% (employee portion)
 * - longTermCare: 12.81% of health insurance
 * - employmentInsurance: 0.9% (employee portion)
 */

export function PayrollPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [selectedDepartment, setSelectedDepartment] = useState<string>("");
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth() + 1);
  const [payrollData] = useState<PayrollEmployee[]>(generateMockPayroll);
  const [selectedPayroll, setSelectedPayroll] = useState<PayrollEmployee | null>(
    null
  );
  const [isPayslipModalOpen, setIsPayslipModalOpen] = useState(false);

  // Get unique departments
  const departments = [...new Set(payrollData.map((p) => p.department))];

  // Filter payroll data
  const filteredPayroll = payrollData.filter((payroll) => {
    const matchesSearch =
      payroll.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      payroll.employeeNumber.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = !selectedStatus || payroll.status === selectedStatus;
    const matchesDepartment =
      !selectedDepartment || payroll.department === selectedDepartment;
    return matchesSearch && matchesStatus && matchesDepartment;
  });

  // Calculate summary stats
  const totalGrossPay = filteredPayroll.reduce((sum, p) => sum + p.grossPay, 0);
  const totalDeductions = filteredPayroll.reduce(
    (sum, p) => sum + p.totalDeductions,
    0
  );
  const totalNetPay = filteredPayroll.reduce((sum, p) => sum + p.netPay, 0);
  const paidCount = filteredPayroll.filter((p) => p.status === "paid").length;

  // Change month
  const handlePrevMonth = () => {
    if (selectedMonth === 1) {
      setSelectedMonth(12);
      setSelectedYear(selectedYear - 1);
    } else {
      setSelectedMonth(selectedMonth - 1);
    }
  };

  const handleNextMonth = () => {
    if (selectedMonth === 12) {
      setSelectedMonth(1);
      setSelectedYear(selectedYear + 1);
    } else {
      setSelectedMonth(selectedMonth + 1);
    }
  };

  // Calculate payroll
  const handleCalculate = () => {
    // TODO: API call to calculate payroll
    toast.success("급여 계산 완료", "모든 직원의 급여가 계산되었습니다.");
  };

  // View payslip
  const handleViewPayslip = (payroll: PayrollEmployee) => {
    setSelectedPayroll(payroll);
    setIsPayslipModalOpen(true);
  };

  // Approve payroll
  const handleApprove = () => {
    // TODO: API call
    toast.success("급여 승인 완료", "선택된 급여가 승인되었습니다.");
  };

  // Pay out
  const handlePayout = () => {
    // TODO: API call
    toast.success("급여 지급 완료", "급여가 지급 처리되었습니다.");
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">급여 관리</h1>
          <p className="text-muted-foreground">
            월별 급여를 계산하고 관리합니다.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleCalculate}>
            <Calculator className="h-4 w-4 mr-2" />
            급여 계산
          </Button>
          <Button variant="outline">
            <Download className="h-4 w-4 mr-2" />
            급여대장
          </Button>
        </div>
      </div>

      {/* Month Selector */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-center gap-4">
            <Button variant="ghost" size="icon" onClick={handlePrevMonth}>
              <ChevronLeft className="h-5 w-5" />
            </Button>
            <div className="text-xl font-bold">
              {selectedYear}년 {selectedMonth}월 급여
            </div>
            <Button variant="ghost" size="icon" onClick={handleNextMonth}>
              <ChevronRight className="h-5 w-5" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">총 지급액</p>
                <p className="text-xl font-bold">
                  {formatCurrency(totalGrossPay)}
                </p>
              </div>
              <DollarSign className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">총 공제액</p>
                <p className="text-xl font-bold text-destructive">
                  -{formatCurrency(totalDeductions)}
                </p>
              </div>
              <AlertCircle className="h-8 w-8 text-destructive/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">실 지급액</p>
                <p className="text-xl font-bold text-success">
                  {formatCurrency(totalNetPay)}
                </p>
              </div>
              <CheckCircle className="h-8 w-8 text-success/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">지급 현황</p>
                <p className="text-xl font-bold">
                  {paidCount}/{filteredPayroll.length}명
                </p>
              </div>
              <Clock className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="이름, 사번으로 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedDepartment}
              onChange={(e) => setSelectedDepartment(e.target.value)}
            >
              <option value="">전체 부서</option>
              {departments.map((dept) => (
                <option key={dept} value={dept}>
                  {dept}
                </option>
              ))}
            </select>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedStatus}
              onChange={(e) => setSelectedStatus(e.target.value)}
            >
              <option value="">전체 상태</option>
              {PAYROLL_STATUS.map((status) => (
                <option key={status.value} value={status.value}>
                  {status.label}
                </option>
              ))}
            </select>
          </div>
        </CardContent>
      </Card>

      {/* Payroll Table */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-lg">급여 목록</CardTitle>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={handleApprove}>
              일괄 승인
            </Button>
            <Button size="sm" onClick={handlePayout}>
              급여 지급
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>사번</TableHead>
                  <TableHead>이름</TableHead>
                  <TableHead>부서</TableHead>
                  <TableHead className="text-right">기본급</TableHead>
                  <TableHead className="text-right">수당</TableHead>
                  <TableHead className="text-right">총 지급액</TableHead>
                  <TableHead className="text-right">공제액</TableHead>
                  <TableHead className="text-right">실 지급액</TableHead>
                  <TableHead>상태</TableHead>
                  <TableHead className="w-12"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredPayroll.map((payroll) => {
                  const totalAllowances =
                    payroll.overtimePay +
                    payroll.bonus +
                    payroll.mealAllowance +
                    payroll.transportAllowance +
                    payroll.otherAllowances;
                  return (
                    <TableRow key={payroll.id}>
                      <TableCell className="font-mono">
                        {payroll.employeeNumber}
                      </TableCell>
                      <TableCell className="font-medium">{payroll.name}</TableCell>
                      <TableCell>{payroll.department}</TableCell>
                      <TableCell className="text-right font-mono">
                        {formatCurrency(payroll.baseSalary, { showSymbol: false })}
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatCurrency(totalAllowances, { showSymbol: false })}
                      </TableCell>
                      <TableCell className="text-right font-mono font-medium">
                        {formatCurrency(payroll.grossPay, { showSymbol: false })}
                      </TableCell>
                      <TableCell className="text-right font-mono text-destructive">
                        -{formatCurrency(payroll.totalDeductions, { showSymbol: false })}
                      </TableCell>
                      <TableCell className="text-right font-mono font-medium text-success">
                        {formatCurrency(payroll.netPay, { showSymbol: false })}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusStyles[payroll.status].variant}>
                          {statusStyles[payroll.status].label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleViewPayslip(payroll)}
                          title="명세서 보기"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Payslip Modal */}
      <Modal
        isOpen={isPayslipModalOpen}
        onClose={() => setIsPayslipModalOpen(false)}
        title="급여 명세서"
        size="lg"
      >
        {selectedPayroll && (
          <div className="space-y-6">
            {/* Header */}
            <div className="text-center border-b pb-4">
              <h2 className="text-xl font-bold">급여 명세서</h2>
              <p className="text-muted-foreground">
                {selectedYear}년 {selectedMonth}월
              </p>
            </div>

            {/* Employee Info */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">사번</p>
                <p className="font-medium">{selectedPayroll.employeeNumber}</p>
              </div>
              <div>
                <p className="text-muted-foreground">성명</p>
                <p className="font-medium">{selectedPayroll.name}</p>
              </div>
              <div>
                <p className="text-muted-foreground">부서</p>
                <p className="font-medium">{selectedPayroll.department}</p>
              </div>
              <div>
                <p className="text-muted-foreground">직급</p>
                <p className="font-medium">{selectedPayroll.position}</p>
              </div>
            </div>

            {/* Earnings and Deductions */}
            <div className="grid md:grid-cols-2 gap-6">
              {/* Earnings */}
              <div className="border rounded-lg p-4">
                <h3 className="font-semibold mb-3 text-primary">지급 내역</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>기본급</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.baseSalary)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>시간외수당</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.overtimePay)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>상여금</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.bonus)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>식대</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.mealAllowance)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>교통비</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.transportAllowance)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>기타수당</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.otherAllowances)}
                    </span>
                  </div>
                  <div className="flex justify-between pt-2 border-t font-semibold">
                    <span>지급 총액</span>
                    <span className="font-mono text-primary">
                      {formatCurrency(selectedPayroll.grossPay)}
                    </span>
                  </div>
                </div>
              </div>

              {/* Deductions */}
              <div className="border rounded-lg p-4">
                <h3 className="font-semibold mb-3 text-destructive">공제 내역</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>국민연금</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.nationalPension)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>건강보험</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.healthInsurance)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>장기요양보험</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.longTermCare)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>고용보험</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.employmentInsurance)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>소득세</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.incomeTax)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>지방소득세</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.localIncomeTax)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>기타공제</span>
                    <span className="font-mono">
                      {formatCurrency(selectedPayroll.otherDeductions)}
                    </span>
                  </div>
                  <div className="flex justify-between pt-2 border-t font-semibold">
                    <span>공제 총액</span>
                    <span className="font-mono text-destructive">
                      {formatCurrency(selectedPayroll.totalDeductions)}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Net Pay */}
            <div className="bg-muted rounded-lg p-4">
              <div className="flex justify-between items-center">
                <span className="text-lg font-semibold">실 지급액</span>
                <span className="text-2xl font-bold text-success">
                  {formatCurrency(selectedPayroll.netPay)}
                </span>
              </div>
            </div>

            {/* Actions */}
            <div className="flex justify-end gap-2 pt-4 border-t">
              <Button
                variant="outline"
                onClick={() => setIsPayslipModalOpen(false)}
              >
                닫기
              </Button>
              <Button variant="outline">
                <Printer className="h-4 w-4 mr-2" />
                인쇄
              </Button>
              <Button>
                <Download className="h-4 w-4 mr-2" />
                PDF 다운로드
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
