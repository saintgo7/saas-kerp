import { useState } from "react";
import { Link } from "react-router-dom";
import {
  Plus,
  Search,
  Filter,
  Download,
  MoreHorizontal,
  Eye,
  Edit,
  Mail,
  Phone,
  Building2,
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
} from "@/components/ui";
import { formatDate, formatPhoneNumber } from "@/lib/utils";
import { EMPLOYEE_STATUS } from "@/constants";
import type { EmployeeStatus } from "@/types";

// Mock data
const mockEmployees = [
  {
    id: "1",
    employeeNumber: "EMP001",
    name: "홍길동",
    department: "개발팀",
    position: "팀장",
    email: "hong@company.com",
    phone: "01012345678",
    joinDate: "2020-03-15",
    status: "active" as EmployeeStatus,
  },
  {
    id: "2",
    employeeNumber: "EMP002",
    name: "김철수",
    department: "영업팀",
    position: "과장",
    email: "kim@company.com",
    phone: "01098765432",
    joinDate: "2021-07-01",
    status: "active" as EmployeeStatus,
  },
  {
    id: "3",
    employeeNumber: "EMP003",
    name: "이영희",
    department: "인사팀",
    position: "대리",
    email: "lee@company.com",
    phone: "01055556666",
    joinDate: "2022-01-10",
    status: "active" as EmployeeStatus,
  },
  {
    id: "4",
    employeeNumber: "EMP004",
    name: "박민수",
    department: "회계팀",
    position: "사원",
    email: "park@company.com",
    phone: "01011112222",
    joinDate: "2023-06-01",
    status: "leave" as EmployeeStatus,
  },
  {
    id: "5",
    employeeNumber: "EMP005",
    name: "최지현",
    department: "마케팅팀",
    position: "차장",
    email: "choi@company.com",
    phone: "01033334444",
    joinDate: "2019-11-20",
    status: "resigned" as EmployeeStatus,
  },
];

const statusStyles: Record<EmployeeStatus, { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }> = {
  active: { variant: "success", label: "재직" },
  leave: { variant: "warning", label: "휴직" },
  resigned: { variant: "secondary", label: "퇴직" },
};

export function EmployeeListPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [selectedDepartment, setSelectedDepartment] = useState<string>("");

  // Get unique departments
  const departments = [...new Set(mockEmployees.map((e) => e.department))];

  const filteredEmployees = mockEmployees.filter((employee) => {
    const matchesSearch =
      employee.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      employee.employeeNumber.toLowerCase().includes(searchTerm.toLowerCase()) ||
      employee.email.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = !selectedStatus || employee.status === selectedStatus;
    const matchesDepartment = !selectedDepartment || employee.department === selectedDepartment;
    return matchesSearch && matchesStatus && matchesDepartment;
  });

  // Stats
  const activeCount = mockEmployees.filter((e) => e.status === "active").length;
  const leaveCount = mockEmployees.filter((e) => e.status === "leave").length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">직원 관리</h1>
          <p className="text-muted-foreground">직원 정보를 조회하고 관리합니다.</p>
        </div>
        <Link to="/hr/employee/new">
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            직원 등록
          </Button>
        </Link>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">전체 직원</p>
            <p className="text-2xl font-bold">{mockEmployees.length}명</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">재직 중</p>
            <p className="text-2xl font-bold text-success">{activeCount}명</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">휴직 중</p>
            <p className="text-2xl font-bold text-warning">{leaveCount}명</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">부서 수</p>
            <p className="text-2xl font-bold">{departments.length}개</p>
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
                  placeholder="이름, 사번, 이메일로 검색..."
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
              {EMPLOYEE_STATUS.map((status) => (
                <option key={status.value} value={status.value}>
                  {status.label}
                </option>
              ))}
            </select>
            <Button variant="outline">
              <Download className="h-4 w-4 mr-2" />
              내보내기
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">직원 목록</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>사번</TableHead>
                <TableHead>이름</TableHead>
                <TableHead>부서</TableHead>
                <TableHead>직급</TableHead>
                <TableHead>연락처</TableHead>
                <TableHead>입사일</TableHead>
                <TableHead>상태</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredEmployees.map((employee) => (
                <TableRow key={employee.id}>
                  <TableCell className="font-mono">{employee.employeeNumber}</TableCell>
                  <TableCell>
                    <Link
                      to={`/hr/employee/${employee.id}`}
                      className="font-medium text-primary hover:underline"
                    >
                      {employee.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center">
                      <Building2 className="h-4 w-4 mr-2 text-muted-foreground" />
                      {employee.department}
                    </div>
                  </TableCell>
                  <TableCell>{employee.position}</TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <div className="flex items-center text-sm">
                        <Mail className="h-3 w-3 mr-1 text-muted-foreground" />
                        {employee.email}
                      </div>
                      <div className="flex items-center text-sm text-muted-foreground">
                        <Phone className="h-3 w-3 mr-1" />
                        {formatPhoneNumber(employee.phone)}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>{formatDate(employee.joinDate)}</TableCell>
                  <TableCell>
                    <Badge variant={statusStyles[employee.status].variant}>
                      {statusStyles[employee.status].label}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="relative group">
                      <Button variant="ghost" size="icon">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                      <div className="absolute right-0 hidden group-hover:block z-10">
                        <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[120px]">
                          <Link
                            to={`/hr/employee/${employee.id}`}
                            className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                          >
                            <Eye className="h-4 w-4 mr-2" />
                            상세보기
                          </Link>
                          <Link
                            to={`/hr/employee/${employee.id}/edit`}
                            className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                          >
                            <Edit className="h-4 w-4 mr-2" />
                            수정
                          </Link>
                        </div>
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
