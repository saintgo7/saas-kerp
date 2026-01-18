import { useState } from "react";
import {
  Search,
  Download,
  Calendar,
  Clock,
  ChevronLeft,
  ChevronRight,
  Plus,
  Sun,
  Coffee,
  Palmtree,
  AlertTriangle,
  Check,
  X,
  Edit,
  Eye,
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
import { formatDate, cn } from "@/lib/utils";
import { toast } from "@/stores/ui";

// Types
type AttendanceStatus = "present" | "late" | "absent" | "leave" | "halfday" | "holiday";
type LeaveType = "annual" | "sick" | "family" | "official" | "unpaid" | "maternity" | "paternity";
type LeaveStatus = "pending" | "approved" | "rejected";

interface AttendanceRecord {
  id: string;
  employeeId: string;
  employeeNumber: string;
  name: string;
  department: string;
  date: string;
  checkIn?: string;
  checkOut?: string;
  status: AttendanceStatus;
  workHours?: number;
  overtimeHours?: number;
  note?: string;
}

interface LeaveRequest {
  id: string;
  employeeId: string;
  employeeNumber: string;
  name: string;
  department: string;
  leaveType: LeaveType;
  startDate: string;
  endDate: string;
  days: number;
  reason: string;
  status: LeaveStatus;
  requestedAt: string;
  processedAt?: string;
  processedBy?: string;
}

interface EmployeeSummary {
  employeeId: string;
  employeeNumber: string;
  name: string;
  department: string;
  totalWorkDays: number;
  presentDays: number;
  lateDays: number;
  absentDays: number;
  leaveDays: number;
  totalWorkHours: number;
  totalOvertimeHours: number;
  annualLeaveUsed: number;
  annualLeaveRemaining: number;
}

const ATTENDANCE_STATUS: Record<
  AttendanceStatus,
  { label: string; variant: "default" | "secondary" | "destructive" | "success" | "warning"; color: string }
> = {
  present: { label: "정상출근", variant: "success", color: "text-success" },
  late: { label: "지각", variant: "warning", color: "text-warning" },
  absent: { label: "결근", variant: "destructive", color: "text-destructive" },
  leave: { label: "휴가", variant: "default", color: "text-primary" },
  halfday: { label: "반차", variant: "secondary", color: "text-muted-foreground" },
  holiday: { label: "휴일", variant: "secondary", color: "text-muted-foreground" },
};

const LEAVE_TYPES: Record<LeaveType, { label: string; icon: React.ElementType }> = {
  annual: { label: "연차", icon: Palmtree },
  sick: { label: "병가", icon: AlertTriangle },
  family: { label: "경조사", icon: Calendar },
  official: { label: "공가", icon: Calendar },
  unpaid: { label: "무급휴가", icon: Calendar },
  maternity: { label: "출산휴가", icon: Calendar },
  paternity: { label: "배우자출산휴가", icon: Calendar },
};

const LEAVE_STATUS: Record<
  LeaveStatus,
  { label: string; variant: "default" | "secondary" | "destructive" | "success" | "warning" }
> = {
  pending: { label: "승인대기", variant: "warning" },
  approved: { label: "승인", variant: "success" },
  rejected: { label: "반려", variant: "destructive" },
};

// Mock data
const generateMockAttendance = (): AttendanceRecord[] => {
  const records: AttendanceRecord[] = [];
  const employees = [
    { id: "emp1", number: "EMP001", name: "홍길동", dept: "개발팀" },
    { id: "emp2", number: "EMP002", name: "김철수", dept: "영업팀" },
    { id: "emp3", number: "EMP003", name: "이영희", dept: "인사팀" },
    { id: "emp4", number: "EMP004", name: "박민수", dept: "회계팀" },
    { id: "emp5", number: "EMP005", name: "최지현", dept: "마케팅팀" },
  ];

  const today = new Date();
  for (let i = 0; i < 5; i++) {
    const date = new Date(today);
    date.setDate(today.getDate() - i);
    const dateStr = date.toISOString().split("T")[0];
    const dayOfWeek = date.getDay();

    for (const emp of employees) {
      if (dayOfWeek === 0 || dayOfWeek === 6) {
        records.push({
          id: `${emp.id}-${dateStr}`,
          employeeId: emp.id,
          employeeNumber: emp.number,
          name: emp.name,
          department: emp.dept,
          date: dateStr,
          status: "holiday",
        });
      } else {
        const statuses: AttendanceStatus[] = ["present", "present", "present", "late", "leave"];
        const status = statuses[Math.floor(Math.random() * statuses.length)];
        const checkIn =
          status === "present"
            ? "08:55"
            : status === "late"
            ? "09:25"
            : undefined;
        const checkOut =
          status === "present" || status === "late" ? "18:05" : undefined;

        records.push({
          id: `${emp.id}-${dateStr}`,
          employeeId: emp.id,
          employeeNumber: emp.number,
          name: emp.name,
          department: emp.dept,
          date: dateStr,
          checkIn,
          checkOut,
          status,
          workHours: checkIn && checkOut ? 8 : 0,
          overtimeHours: checkIn && checkOut ? 0.08 : 0,
        });
      }
    }
  }

  return records;
};

const mockLeaveRequests: LeaveRequest[] = [
  {
    id: "1",
    employeeId: "emp1",
    employeeNumber: "EMP001",
    name: "홍길동",
    department: "개발팀",
    leaveType: "annual",
    startDate: "2024-01-22",
    endDate: "2024-01-24",
    days: 3,
    reason: "개인 휴가",
    status: "pending",
    requestedAt: "2024-01-15",
  },
  {
    id: "2",
    employeeId: "emp2",
    employeeNumber: "EMP002",
    name: "김철수",
    department: "영업팀",
    leaveType: "sick",
    startDate: "2024-01-18",
    endDate: "2024-01-18",
    days: 1,
    reason: "병원 진료",
    status: "approved",
    requestedAt: "2024-01-17",
    processedAt: "2024-01-17",
    processedBy: "관리자",
  },
  {
    id: "3",
    employeeId: "emp3",
    employeeNumber: "EMP003",
    name: "이영희",
    department: "인사팀",
    leaveType: "family",
    startDate: "2024-01-25",
    endDate: "2024-01-26",
    days: 2,
    reason: "가족 행사",
    status: "rejected",
    requestedAt: "2024-01-10",
    processedAt: "2024-01-11",
    processedBy: "관리자",
  },
];

const mockSummary: EmployeeSummary[] = [
  {
    employeeId: "emp1",
    employeeNumber: "EMP001",
    name: "홍길동",
    department: "개발팀",
    totalWorkDays: 22,
    presentDays: 20,
    lateDays: 1,
    absentDays: 0,
    leaveDays: 1,
    totalWorkHours: 176,
    totalOvertimeHours: 12,
    annualLeaveUsed: 5,
    annualLeaveRemaining: 10,
  },
  {
    employeeId: "emp2",
    employeeNumber: "EMP002",
    name: "김철수",
    department: "영업팀",
    totalWorkDays: 22,
    presentDays: 19,
    lateDays: 2,
    absentDays: 0,
    leaveDays: 1,
    totalWorkHours: 168,
    totalOvertimeHours: 8,
    annualLeaveUsed: 3,
    annualLeaveRemaining: 12,
  },
  {
    employeeId: "emp3",
    employeeNumber: "EMP003",
    name: "이영희",
    department: "인사팀",
    totalWorkDays: 22,
    presentDays: 21,
    lateDays: 0,
    absentDays: 0,
    leaveDays: 1,
    totalWorkHours: 176,
    totalOvertimeHours: 4,
    annualLeaveUsed: 2,
    annualLeaveRemaining: 13,
  },
];

export function AttendancePage() {
  const [activeTab, setActiveTab] = useState<"daily" | "summary" | "leave">("daily");
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedDepartment, setSelectedDepartment] = useState<string>("");
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth() + 1);
  const [selectedDate, setSelectedDate] = useState(
    new Date().toISOString().split("T")[0]
  );
  const [attendanceData] = useState<AttendanceRecord[]>(generateMockAttendance);
  const [isLeaveModalOpen, setIsLeaveModalOpen] = useState(false);
  const [selectedLeave, setSelectedLeave] = useState<LeaveRequest | null>(null);
  const [isLeaveDetailModalOpen, setIsLeaveDetailModalOpen] = useState(false);

  // Get unique departments
  const departments = [...new Set(attendanceData.map((a) => a.department))];

  // Filter attendance by date
  const filteredAttendance = attendanceData.filter((record) => {
    const matchesSearch =
      record.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      record.employeeNumber.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesDepartment =
      !selectedDepartment || record.department === selectedDepartment;
    const matchesDate = record.date === selectedDate;
    return matchesSearch && matchesDepartment && matchesDate;
  });

  // Filter leave requests
  const filteredLeaveRequests = mockLeaveRequests.filter((request) => {
    const matchesSearch =
      request.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      request.employeeNumber.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesDepartment =
      !selectedDepartment || request.department === selectedDepartment;
    return matchesSearch && matchesDepartment;
  });

  // Calculate summary stats for the selected date
  const presentCount = filteredAttendance.filter(
    (a) => a.status === "present"
  ).length;
  const lateCount = filteredAttendance.filter((a) => a.status === "late").length;
  const absentCount = filteredAttendance.filter(
    (a) => a.status === "absent"
  ).length;
  const leaveCount = filteredAttendance.filter(
    (a) => a.status === "leave" || a.status === "halfday"
  ).length;
  const pendingLeaveCount = mockLeaveRequests.filter(
    (r) => r.status === "pending"
  ).length;

  // Change date
  const handlePrevDate = () => {
    const date = new Date(selectedDate);
    date.setDate(date.getDate() - 1);
    setSelectedDate(date.toISOString().split("T")[0]);
  };

  const handleNextDate = () => {
    const date = new Date(selectedDate);
    date.setDate(date.getDate() + 1);
    setSelectedDate(date.toISOString().split("T")[0]);
  };

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

  // Handle leave approval
  const handleApproveLeave = (request: LeaveRequest) => {
    // TODO: API call
    toast.success("휴가 승인 완료", `${request.name}님의 휴가가 승인되었습니다.`);
    setIsLeaveDetailModalOpen(false);
  };

  const handleRejectLeave = (request: LeaveRequest) => {
    // TODO: API call
    toast.info("휴가 반려", `${request.name}님의 휴가가 반려되었습니다.`);
    setIsLeaveDetailModalOpen(false);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">근태 관리</h1>
          <p className="text-muted-foreground">
            출퇴근 기록과 휴가를 관리합니다.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setIsLeaveModalOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            휴가 신청
          </Button>
          <Button variant="outline">
            <Download className="h-4 w-4 mr-2" />
            근태 보고서
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">정상출근</p>
                <p className="text-2xl font-bold text-success">{presentCount}명</p>
              </div>
              <Sun className="h-8 w-8 text-success/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">지각</p>
                <p className="text-2xl font-bold text-warning">{lateCount}명</p>
              </div>
              <Clock className="h-8 w-8 text-warning/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">결근</p>
                <p className="text-2xl font-bold text-destructive">{absentCount}명</p>
              </div>
              <X className="h-8 w-8 text-destructive/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">휴가/반차</p>
                <p className="text-2xl font-bold text-primary">{leaveCount}명</p>
              </div>
              <Palmtree className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">휴가 승인대기</p>
                <p className="text-2xl font-bold">{pendingLeaveCount}건</p>
              </div>
              <Coffee className="h-8 w-8 text-muted-foreground/20" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tabs */}
      <div className="flex border-b">
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 transition-colors",
            activeTab === "daily"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("daily")}
        >
          일별 근태
        </button>
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 transition-colors",
            activeTab === "summary"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("summary")}
        >
          월별 현황
        </button>
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 transition-colors relative",
            activeTab === "leave"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("leave")}
        >
          휴가 관리
          {pendingLeaveCount > 0 && (
            <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-destructive text-destructive-foreground text-xs flex items-center justify-center">
              {pendingLeaveCount}
            </span>
          )}
        </button>
      </div>

      {/* Date/Month Selector */}
      {activeTab === "daily" ? (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-center gap-4">
              <Button variant="ghost" size="icon" onClick={handlePrevDate}>
                <ChevronLeft className="h-5 w-5" />
              </Button>
              <div className="flex items-center gap-2">
                <Calendar className="h-5 w-5 text-muted-foreground" />
                <Input
                  type="date"
                  value={selectedDate}
                  onChange={(e) => setSelectedDate(e.target.value)}
                  className="w-40"
                />
              </div>
              <Button variant="ghost" size="icon" onClick={handleNextDate}>
                <ChevronRight className="h-5 w-5" />
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : activeTab === "summary" ? (
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
      ) : null}

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
          </div>
        </CardContent>
      </Card>

      {/* Content */}
      {activeTab === "daily" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">
              {formatDate(selectedDate, { format: "full" })} 근태 현황
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>사번</TableHead>
                  <TableHead>이름</TableHead>
                  <TableHead>부서</TableHead>
                  <TableHead className="text-center">출근</TableHead>
                  <TableHead className="text-center">퇴근</TableHead>
                  <TableHead className="text-center">근무시간</TableHead>
                  <TableHead className="text-center">초과근무</TableHead>
                  <TableHead>상태</TableHead>
                  <TableHead className="w-12"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredAttendance.map((record) => {
                  const statusConfig = ATTENDANCE_STATUS[record.status];
                  return (
                    <TableRow key={record.id}>
                      <TableCell className="font-mono">
                        {record.employeeNumber}
                      </TableCell>
                      <TableCell className="font-medium">{record.name}</TableCell>
                      <TableCell>{record.department}</TableCell>
                      <TableCell className="text-center font-mono">
                        {record.checkIn || "-"}
                      </TableCell>
                      <TableCell className="text-center font-mono">
                        {record.checkOut || "-"}
                      </TableCell>
                      <TableCell className="text-center font-mono">
                        {record.workHours ? `${record.workHours}h` : "-"}
                      </TableCell>
                      <TableCell className="text-center font-mono">
                        {record.overtimeHours
                          ? `${record.overtimeHours.toFixed(1)}h`
                          : "-"}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusConfig.variant}>
                          {statusConfig.label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button variant="ghost" size="icon" title="수정">
                          <Edit className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {activeTab === "summary" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">
              {selectedYear}년 {selectedMonth}월 근태 현황
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>사번</TableHead>
                  <TableHead>이름</TableHead>
                  <TableHead>부서</TableHead>
                  <TableHead className="text-center">근무일</TableHead>
                  <TableHead className="text-center">출근</TableHead>
                  <TableHead className="text-center">지각</TableHead>
                  <TableHead className="text-center">결근</TableHead>
                  <TableHead className="text-center">휴가</TableHead>
                  <TableHead className="text-center">총 근무시간</TableHead>
                  <TableHead className="text-center">초과근무</TableHead>
                  <TableHead className="text-center">잔여연차</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mockSummary.map((summary) => (
                  <TableRow key={summary.employeeId}>
                    <TableCell className="font-mono">
                      {summary.employeeNumber}
                    </TableCell>
                    <TableCell className="font-medium">{summary.name}</TableCell>
                    <TableCell>{summary.department}</TableCell>
                    <TableCell className="text-center">
                      {summary.totalWorkDays}일
                    </TableCell>
                    <TableCell className="text-center text-success">
                      {summary.presentDays}일
                    </TableCell>
                    <TableCell className="text-center text-warning">
                      {summary.lateDays}일
                    </TableCell>
                    <TableCell className="text-center text-destructive">
                      {summary.absentDays}일
                    </TableCell>
                    <TableCell className="text-center text-primary">
                      {summary.leaveDays}일
                    </TableCell>
                    <TableCell className="text-center font-mono">
                      {summary.totalWorkHours}h
                    </TableCell>
                    <TableCell className="text-center font-mono">
                      {summary.totalOvertimeHours}h
                    </TableCell>
                    <TableCell className="text-center">
                      <span className="font-medium">
                        {summary.annualLeaveRemaining}
                      </span>
                      <span className="text-muted-foreground">
                        /{summary.annualLeaveUsed + summary.annualLeaveRemaining}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {activeTab === "leave" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">휴가 신청 내역</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>사번</TableHead>
                  <TableHead>이름</TableHead>
                  <TableHead>부서</TableHead>
                  <TableHead>휴가 유형</TableHead>
                  <TableHead>시작일</TableHead>
                  <TableHead>종료일</TableHead>
                  <TableHead className="text-center">일수</TableHead>
                  <TableHead>사유</TableHead>
                  <TableHead>상태</TableHead>
                  <TableHead className="w-12"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredLeaveRequests.map((request) => {
                  const leaveType = LEAVE_TYPES[request.leaveType];
                  const statusConfig = LEAVE_STATUS[request.status];
                  const LeaveIcon = leaveType.icon;

                  return (
                    <TableRow key={request.id}>
                      <TableCell className="font-mono">
                        {request.employeeNumber}
                      </TableCell>
                      <TableCell className="font-medium">{request.name}</TableCell>
                      <TableCell>{request.department}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <LeaveIcon className="h-4 w-4 text-muted-foreground" />
                          {leaveType.label}
                        </div>
                      </TableCell>
                      <TableCell>{formatDate(request.startDate)}</TableCell>
                      <TableCell>{formatDate(request.endDate)}</TableCell>
                      <TableCell className="text-center">{request.days}일</TableCell>
                      <TableCell className="max-w-[200px] truncate">
                        {request.reason}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusConfig.variant}>
                          {statusConfig.label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => {
                            setSelectedLeave(request);
                            setIsLeaveDetailModalOpen(true);
                          }}
                          title="상세보기"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Leave Request Modal */}
      <Modal
        isOpen={isLeaveModalOpen}
        onClose={() => setIsLeaveModalOpen(false)}
        title="휴가 신청"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">휴가 유형</label>
            <select className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm">
              {Object.entries(LEAVE_TYPES).map(([key, value]) => (
                <option key={key} value={key}>
                  {value.label}
                </option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <Input type="date" label="시작일" required />
            <Input type="date" label="종료일" required />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">휴가 사유</label>
            <textarea
              className="w-full min-h-[100px] rounded-md border border-input bg-background px-3 py-2 text-sm"
              placeholder="휴가 사유를 입력하세요"
            />
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => setIsLeaveModalOpen(false)}>
              취소
            </Button>
            <Button
              onClick={() => {
                toast.success("휴가 신청 완료", "휴가 신청이 제출되었습니다.");
                setIsLeaveModalOpen(false);
              }}
            >
              신청
            </Button>
          </div>
        </div>
      </Modal>

      {/* Leave Detail Modal */}
      <Modal
        isOpen={isLeaveDetailModalOpen}
        onClose={() => {
          setIsLeaveDetailModalOpen(false);
          setSelectedLeave(null);
        }}
        title="휴가 상세"
      >
        {selectedLeave && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">신청자</p>
                <p className="font-medium">{selectedLeave.name}</p>
              </div>
              <div>
                <p className="text-muted-foreground">부서</p>
                <p className="font-medium">{selectedLeave.department}</p>
              </div>
              <div>
                <p className="text-muted-foreground">휴가 유형</p>
                <p className="font-medium">
                  {LEAVE_TYPES[selectedLeave.leaveType].label}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">일수</p>
                <p className="font-medium">{selectedLeave.days}일</p>
              </div>
              <div>
                <p className="text-muted-foreground">시작일</p>
                <p className="font-medium">{formatDate(selectedLeave.startDate)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">종료일</p>
                <p className="font-medium">{formatDate(selectedLeave.endDate)}</p>
              </div>
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">사유</p>
              <p className="text-sm p-3 bg-muted rounded-lg">
                {selectedLeave.reason}
              </p>
            </div>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">신청일</p>
                <p className="font-medium">
                  {formatDate(selectedLeave.requestedAt)}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">상태</p>
                <Badge variant={LEAVE_STATUS[selectedLeave.status].variant}>
                  {LEAVE_STATUS[selectedLeave.status].label}
                </Badge>
              </div>
            </div>
            {selectedLeave.status === "pending" && (
              <div className="flex justify-end gap-2 pt-4 border-t">
                <Button
                  variant="outline"
                  onClick={() => handleRejectLeave(selectedLeave)}
                >
                  <X className="h-4 w-4 mr-2" />
                  반려
                </Button>
                <Button onClick={() => handleApproveLeave(selectedLeave)}>
                  <Check className="h-4 w-4 mr-2" />
                  승인
                </Button>
              </div>
            )}
            {selectedLeave.status !== "pending" && (
              <div className="flex justify-end pt-4 border-t">
                <Button
                  variant="outline"
                  onClick={() => setIsLeaveDetailModalOpen(false)}
                >
                  닫기
                </Button>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
