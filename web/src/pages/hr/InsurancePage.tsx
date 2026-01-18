import { useState } from "react";
import {
  Search,
  Download,
  Send,
  CheckCircle,
  Clock,
  XCircle,
  FileText,
  Shield,
  Heart,
  Briefcase,
  HardHat,
  RefreshCw,
  Eye,
  ChevronLeft,
  ChevronRight,
  Plus,
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
import { formatCurrency, formatDate } from "@/lib/utils";
import { toast } from "@/stores/ui";
import { cn } from "@/lib/utils";

// Types
type InsuranceType = "national_pension" | "health" | "employment" | "industrial";
type ReportType = "acquisition" | "loss" | "change";
type ReportStatus = "draft" | "submitted" | "processing" | "completed" | "rejected";

interface InsuranceEmployee {
  id: string;
  employeeId: string;
  employeeNumber: string;
  name: string;
  department: string;
  joinDate: string;
  resignDate?: string;
  baseSalary: number;
  nationalPension: {
    number: string;
    acquired: boolean;
    monthlyAmount: number;
  };
  healthInsurance: {
    number: string;
    acquired: boolean;
    monthlyAmount: number;
    longTermCare: number;
  };
  employmentInsurance: {
    number: string;
    acquired: boolean;
    monthlyAmount: number;
  };
  industrialAccident: {
    number: string;
    acquired: boolean;
    monthlyAmount: number;
  };
}

interface InsuranceReport {
  id: string;
  type: InsuranceType;
  reportType: ReportType;
  employeeId: string;
  employeeName: string;
  employeeNumber: string;
  submittedDate?: string;
  processedDate?: string;
  status: ReportStatus;
  confirmNumber?: string;
  reason?: string;
}

// Insurance type configurations
const INSURANCE_TYPES = [
  {
    key: "national_pension" as InsuranceType,
    name: "국민연금",
    shortName: "연금",
    icon: Shield,
    color: "text-blue-600",
    bgColor: "bg-blue-50",
    rate: "9% (사업주 4.5%, 근로자 4.5%)",
    agency: "국민연금공단",
  },
  {
    key: "health" as InsuranceType,
    name: "건강보험",
    shortName: "건강",
    icon: Heart,
    color: "text-green-600",
    bgColor: "bg-green-50",
    rate: "7.09% (사업주 3.545%, 근로자 3.545%)",
    agency: "국민건강보험공단",
  },
  {
    key: "employment" as InsuranceType,
    name: "고용보험",
    shortName: "고용",
    icon: Briefcase,
    color: "text-purple-600",
    bgColor: "bg-purple-50",
    rate: "1.8% (사업주 0.9%, 근로자 0.9%)",
    agency: "근로복지공단",
  },
  {
    key: "industrial" as InsuranceType,
    name: "산재보험",
    shortName: "산재",
    icon: HardHat,
    color: "text-orange-600",
    bgColor: "bg-orange-50",
    rate: "0.7%~34% (업종별 상이, 사업주 부담)",
    agency: "근로복지공단",
  },
];

const REPORT_TYPES = [
  { value: "acquisition", label: "취득 신고" },
  { value: "loss", label: "상실 신고" },
  { value: "change", label: "변경 신고" },
];

const STATUS_CONFIG: Record<
  ReportStatus,
  { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string; icon: React.ElementType }
> = {
  draft: { variant: "secondary", label: "작성중", icon: FileText },
  submitted: { variant: "warning", label: "접수", icon: Clock },
  processing: { variant: "default", label: "처리중", icon: RefreshCw },
  completed: { variant: "success", label: "완료", icon: CheckCircle },
  rejected: { variant: "destructive", label: "반려", icon: XCircle },
};

// Mock data
const mockEmployees: InsuranceEmployee[] = [
  {
    id: "1",
    employeeId: "emp1",
    employeeNumber: "EMP001",
    name: "홍길동",
    department: "개발팀",
    joinDate: "2020-03-15",
    baseSalary: 5000000,
    nationalPension: {
      number: "NP-2020-001234",
      acquired: true,
      monthlyAmount: 225000,
    },
    healthInsurance: {
      number: "HI-2020-005678",
      acquired: true,
      monthlyAmount: 177250,
      longTermCare: 22685,
    },
    employmentInsurance: {
      number: "EI-2020-009012",
      acquired: true,
      monthlyAmount: 45000,
    },
    industrialAccident: {
      number: "IA-2020-003456",
      acquired: true,
      monthlyAmount: 0,
    },
  },
  {
    id: "2",
    employeeId: "emp2",
    employeeNumber: "EMP002",
    name: "김철수",
    department: "영업팀",
    joinDate: "2021-07-01",
    baseSalary: 4200000,
    nationalPension: {
      number: "NP-2021-002345",
      acquired: true,
      monthlyAmount: 189000,
    },
    healthInsurance: {
      number: "HI-2021-006789",
      acquired: true,
      monthlyAmount: 148890,
      longTermCare: 19054,
    },
    employmentInsurance: {
      number: "EI-2021-000123",
      acquired: true,
      monthlyAmount: 37800,
    },
    industrialAccident: {
      number: "IA-2021-004567",
      acquired: true,
      monthlyAmount: 0,
    },
  },
  {
    id: "3",
    employeeId: "emp3",
    employeeNumber: "EMP003",
    name: "이영희",
    department: "인사팀",
    joinDate: "2024-01-02",
    baseSalary: 3500000,
    nationalPension: {
      number: "",
      acquired: false,
      monthlyAmount: 157500,
    },
    healthInsurance: {
      number: "",
      acquired: false,
      monthlyAmount: 124075,
      longTermCare: 15879,
    },
    employmentInsurance: {
      number: "",
      acquired: false,
      monthlyAmount: 31500,
    },
    industrialAccident: {
      number: "",
      acquired: false,
      monthlyAmount: 0,
    },
  },
];

const mockReports: InsuranceReport[] = [
  {
    id: "1",
    type: "national_pension",
    reportType: "acquisition",
    employeeId: "emp3",
    employeeName: "이영희",
    employeeNumber: "EMP003",
    submittedDate: "2024-01-05",
    status: "processing",
    confirmNumber: "2024-NP-00123",
  },
  {
    id: "2",
    type: "health",
    reportType: "acquisition",
    employeeId: "emp3",
    employeeName: "이영희",
    employeeNumber: "EMP003",
    submittedDate: "2024-01-05",
    status: "completed",
    processedDate: "2024-01-08",
    confirmNumber: "2024-HI-00456",
  },
  {
    id: "3",
    type: "employment",
    reportType: "loss",
    employeeId: "emp4",
    employeeName: "박민수",
    employeeNumber: "EMP004",
    submittedDate: "2024-01-10",
    status: "submitted",
    confirmNumber: "2024-EI-00789",
  },
  {
    id: "4",
    type: "industrial",
    reportType: "change",
    employeeId: "emp2",
    employeeName: "김철수",
    employeeNumber: "EMP002",
    status: "draft",
  },
];

export function InsurancePage() {
  const [activeTab, setActiveTab] = useState<"overview" | "reports">("overview");
  const [selectedInsurance, setSelectedInsurance] = useState<InsuranceType | "all">(
    "all"
  );
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth() + 1);
  const [isReportModalOpen, setIsReportModalOpen] = useState(false);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_selectedEmployee, _setSelectedEmployee] = useState<InsuranceEmployee | null>(
    null
  );

  // Filter employees
  const filteredEmployees = mockEmployees.filter((emp) => {
    const matchesSearch =
      emp.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      emp.employeeNumber.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesSearch;
  });

  // Filter reports
  const filteredReports = mockReports.filter((report) => {
    const matchesSearch =
      report.employeeName.toLowerCase().includes(searchTerm.toLowerCase()) ||
      report.employeeNumber.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType =
      selectedInsurance === "all" || report.type === selectedInsurance;
    return matchesSearch && matchesType;
  });

  // Calculate totals
  const totalNationalPension = mockEmployees.reduce(
    (sum, emp) => sum + emp.nationalPension.monthlyAmount,
    0
  );
  const totalHealthInsurance = mockEmployees.reduce(
    (sum, emp) => sum + emp.healthInsurance.monthlyAmount + emp.healthInsurance.longTermCare,
    0
  );
  const totalEmploymentInsurance = mockEmployees.reduce(
    (sum, emp) => sum + emp.employmentInsurance.monthlyAmount,
    0
  );
  const acquiredCount = mockEmployees.filter(
    (emp) => emp.nationalPension.acquired
  ).length;
  const pendingCount = mockEmployees.filter(
    (emp) => !emp.nationalPension.acquired
  ).length;

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

  // Handle EDI sync
  const handleEdiSync = (insuranceType: InsuranceType) => {
    const insurance = INSURANCE_TYPES.find((i) => i.key === insuranceType);
    toast.info(
      "EDI 연동 요청",
      `${insurance?.agency}와 데이터 동기화를 시작합니다.`
    );
  };

  // Handle report submission
  const handleSubmitReport = () => {
    toast.success("신고서 제출 완료", "EDI를 통해 신고서가 전송되었습니다.");
    setIsReportModalOpen(false);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">4대보험 관리</h1>
          <p className="text-muted-foreground">
            4대보험 취득/상실 신고 및 보험료를 관리합니다.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setIsReportModalOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            신고서 작성
          </Button>
          <Button variant="outline">
            <Download className="h-4 w-4 mr-2" />
            보험료 내역
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
              {selectedYear}년 {selectedMonth}월
            </div>
            <Button variant="ghost" size="icon" onClick={handleNextMonth}>
              <ChevronRight className="h-5 w-5" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Insurance Type Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {INSURANCE_TYPES.map((insurance) => {
          const IconComponent = insurance.icon;
          let totalAmount = 0;
          switch (insurance.key) {
            case "national_pension":
              totalAmount = totalNationalPension;
              break;
            case "health":
              totalAmount = totalHealthInsurance;
              break;
            case "employment":
              totalAmount = totalEmploymentInsurance;
              break;
            case "industrial":
              totalAmount = 0; // Company pays
              break;
          }
          return (
            <Card
              key={insurance.key}
              className={cn(
                "cursor-pointer transition-all hover:shadow-md",
                selectedInsurance === insurance.key && "ring-2 ring-primary"
              )}
              onClick={() =>
                setSelectedInsurance(
                  selectedInsurance === insurance.key ? "all" : insurance.key
                )
              }
            >
              <CardContent className="p-4">
                <div className="flex items-start justify-between mb-3">
                  <div className={cn("p-2 rounded-lg", insurance.bgColor)}>
                    <IconComponent className={cn("h-5 w-5", insurance.color)} />
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleEdiSync(insurance.key);
                    }}
                    title="EDI 동기화"
                  >
                    <RefreshCw className="h-4 w-4" />
                  </Button>
                </div>
                <h3 className="font-semibold">{insurance.name}</h3>
                <p className="text-xs text-muted-foreground mb-2">
                  {insurance.agency}
                </p>
                <p className="text-lg font-bold">{formatCurrency(totalAmount)}</p>
                <p className="text-xs text-muted-foreground">{insurance.rate}</p>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Tabs */}
      <div className="flex border-b">
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 transition-colors",
            activeTab === "overview"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("overview")}
        >
          가입 현황
        </button>
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 transition-colors",
            activeTab === "reports"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("reports")}
        >
          신고 내역
        </button>
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
              value={selectedInsurance}
              onChange={(e) =>
                setSelectedInsurance(e.target.value as InsuranceType | "all")
              }
            >
              <option value="all">전체 보험</option>
              {INSURANCE_TYPES.map((insurance) => (
                <option key={insurance.key} value={insurance.key}>
                  {insurance.name}
                </option>
              ))}
            </select>
          </div>
        </CardContent>
      </Card>

      {/* Content */}
      {activeTab === "overview" ? (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">직원별 가입 현황</CardTitle>
            <div className="flex items-center gap-4 text-sm">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-success" />
                <span>가입완료 {acquiredCount}명</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-warning" />
                <span>미가입 {pendingCount}명</span>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>사번</TableHead>
                    <TableHead>이름</TableHead>
                    <TableHead>입사일</TableHead>
                    <TableHead className="text-center">국민연금</TableHead>
                    <TableHead className="text-center">건강보험</TableHead>
                    <TableHead className="text-center">고용보험</TableHead>
                    <TableHead className="text-center">산재보험</TableHead>
                    <TableHead className="text-right">월 보험료</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredEmployees.map((emp) => {
                    const totalMonthly =
                      emp.nationalPension.monthlyAmount +
                      emp.healthInsurance.monthlyAmount +
                      emp.healthInsurance.longTermCare +
                      emp.employmentInsurance.monthlyAmount;
                    return (
                      <TableRow key={emp.id}>
                        <TableCell className="font-mono">
                          {emp.employeeNumber}
                        </TableCell>
                        <TableCell className="font-medium">{emp.name}</TableCell>
                        <TableCell>{formatDate(emp.joinDate)}</TableCell>
                        <TableCell className="text-center">
                          {emp.nationalPension.acquired ? (
                            <Badge variant="success">가입</Badge>
                          ) : (
                            <Badge variant="warning">미가입</Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-center">
                          {emp.healthInsurance.acquired ? (
                            <Badge variant="success">가입</Badge>
                          ) : (
                            <Badge variant="warning">미가입</Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-center">
                          {emp.employmentInsurance.acquired ? (
                            <Badge variant="success">가입</Badge>
                          ) : (
                            <Badge variant="warning">미가입</Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-center">
                          {emp.industrialAccident.acquired ? (
                            <Badge variant="success">가입</Badge>
                          ) : (
                            <Badge variant="warning">미가입</Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {formatCurrency(totalMonthly)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">신고 내역</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>보험 종류</TableHead>
                    <TableHead>신고 유형</TableHead>
                    <TableHead>사번</TableHead>
                    <TableHead>이름</TableHead>
                    <TableHead>접수일</TableHead>
                    <TableHead>처리일</TableHead>
                    <TableHead>접수번호</TableHead>
                    <TableHead>상태</TableHead>
                    <TableHead className="w-12"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredReports.map((report) => {
                    const insurance = INSURANCE_TYPES.find(
                      (i) => i.key === report.type
                    );
                    const reportType = REPORT_TYPES.find(
                      (r) => r.value === report.reportType
                    );
                    const statusConfig = STATUS_CONFIG[report.status];
                    const IconComponent = insurance?.icon || Shield;
                    const StatusIcon = statusConfig.icon;

                    return (
                      <TableRow key={report.id}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <IconComponent
                              className={cn("h-4 w-4", insurance?.color)}
                            />
                            {insurance?.shortName}
                          </div>
                        </TableCell>
                        <TableCell>{reportType?.label}</TableCell>
                        <TableCell className="font-mono">
                          {report.employeeNumber}
                        </TableCell>
                        <TableCell className="font-medium">
                          {report.employeeName}
                        </TableCell>
                        <TableCell>
                          {report.submittedDate
                            ? formatDate(report.submittedDate)
                            : "-"}
                        </TableCell>
                        <TableCell>
                          {report.processedDate
                            ? formatDate(report.processedDate)
                            : "-"}
                        </TableCell>
                        <TableCell className="font-mono text-sm">
                          {report.confirmNumber || "-"}
                        </TableCell>
                        <TableCell>
                          <Badge variant={statusConfig.variant}>
                            <StatusIcon className="h-3 w-3 mr-1" />
                            {statusConfig.label}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Button variant="ghost" size="icon" title="상세보기">
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
      )}

      {/* Report Creation Modal */}
      <Modal
        isOpen={isReportModalOpen}
        onClose={() => setIsReportModalOpen(false)}
        title="4대보험 신고서 작성"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">보험 종류</label>
            <select className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm">
              {INSURANCE_TYPES.map((insurance) => (
                <option key={insurance.key} value={insurance.key}>
                  {insurance.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">신고 유형</label>
            <select className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm">
              {REPORT_TYPES.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">대상 직원</label>
            <select className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm">
              <option value="">직원 선택</option>
              {mockEmployees.map((emp) => (
                <option key={emp.id} value={emp.id}>
                  {emp.employeeNumber} - {emp.name} ({emp.department})
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">신고 사유</label>
            <Input placeholder="신고 사유를 입력하세요" />
          </div>
          <div className="bg-muted rounded-lg p-4 text-sm">
            <p className="font-medium mb-2">EDI 전송 안내</p>
            <ul className="list-disc list-inside space-y-1 text-muted-foreground">
              <li>신고서 제출 시 EDI를 통해 해당 공단으로 자동 전송됩니다.</li>
              <li>처리 결과는 1~3 영업일 내에 확인 가능합니다.</li>
              <li>접수번호는 신고 완료 후 자동 부여됩니다.</li>
            </ul>
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => setIsReportModalOpen(false)}>
              취소
            </Button>
            <Button onClick={handleSubmitReport}>
              <Send className="h-4 w-4 mr-2" />
              EDI 전송
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
