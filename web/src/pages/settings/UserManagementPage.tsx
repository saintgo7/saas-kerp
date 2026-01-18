import { useState } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Users,
  Plus,
  Search,
  Download,
  MoreHorizontal,
  Edit,
  Trash2,
  Key,
  Lock,
  Unlock,
  CheckCircle,
  XCircle,
  Clock,
  Mail,
  Phone,
  Shield,
} from "lucide-react";
import {
  Button,
  Input,
  Select,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Modal,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { formatDate, formatPhoneNumber } from "@/lib/utils";
import { toast } from "@/stores/ui";
import { USER_ROLES } from "@/constants";
import type { UserRole } from "@/types";

// Validation schema for user
const userSchema = z.object({
  email: z.string().email("올바른 이메일 형식이 아닙니다"),
  name: z.string().min(1, "이름을 입력하세요"),
  phone: z.string().optional(),
  role: z.enum(["admin", "manager", "accountant", "hr", "user"]),
  departmentId: z.string().optional(),
});

type UserFormData = z.infer<typeof userSchema>;

// User status type
type UserStatus = "active" | "inactive" | "locked";

// Mock user data
interface MockUser {
  id: string;
  email: string;
  name: string;
  phone?: string;
  role: UserRole;
  departmentId?: string;
  departmentName?: string;
  status: UserStatus;
  lastLoginAt?: string;
  createdAt: string;
}

const mockUsers: MockUser[] = [
  {
    id: "1",
    email: "admin@techsolution.co.kr",
    name: "김관리",
    phone: "01012345678",
    role: "admin",
    departmentName: "경영지원팀",
    status: "active",
    lastLoginAt: "2024-01-15T09:30:00",
    createdAt: "2023-01-01",
  },
  {
    id: "2",
    email: "kim.manager@techsolution.co.kr",
    name: "김매니저",
    phone: "01023456789",
    role: "manager",
    departmentName: "영업팀",
    status: "active",
    lastLoginAt: "2024-01-14T18:20:00",
    createdAt: "2023-03-15",
  },
  {
    id: "3",
    email: "lee.accountant@techsolution.co.kr",
    name: "이회계",
    phone: "01034567890",
    role: "accountant",
    departmentName: "회계팀",
    status: "active",
    lastLoginAt: "2024-01-15T08:45:00",
    createdAt: "2023-06-01",
  },
  {
    id: "4",
    email: "park.hr@techsolution.co.kr",
    name: "박인사",
    phone: "01045678901",
    role: "hr",
    departmentName: "인사팀",
    status: "inactive",
    lastLoginAt: "2024-01-10T14:30:00",
    createdAt: "2023-09-01",
  },
  {
    id: "5",
    email: "choi.user@techsolution.co.kr",
    name: "최사원",
    phone: "01056789012",
    role: "user",
    departmentName: "개발팀",
    status: "locked",
    lastLoginAt: "2024-01-05T11:00:00",
    createdAt: "2023-11-01",
  },
];

// Mock department options
const departmentOptions = [
  { value: "dept1", label: "경영지원팀" },
  { value: "dept2", label: "영업팀" },
  { value: "dept3", label: "회계팀" },
  { value: "dept4", label: "인사팀" },
  { value: "dept5", label: "개발팀" },
];

// Role options for select
const roleOptions = Object.entries(USER_ROLES).map(([value, label]) => ({
  value,
  label,
}));

// Status badge styles
const statusStyles: Record<UserStatus, { variant: "success" | "secondary" | "destructive"; label: string; icon: typeof CheckCircle }> = {
  active: { variant: "success", label: "활성", icon: CheckCircle },
  inactive: { variant: "secondary", label: "비활성", icon: XCircle },
  locked: { variant: "destructive", label: "잠금", icon: Lock },
};

export function UserManagementPage() {
  const [users, setUsers] = useState(mockUsers);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedRole, setSelectedRole] = useState<string>("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [userModalOpen, setUserModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<MockUser | null>(null);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [userToDelete, setUserToDelete] = useState<string | null>(null);
  const [resetPasswordModalOpen, setResetPasswordModalOpen] = useState(false);
  const [userToResetPassword, setUserToResetPassword] = useState<MockUser | null>(null);

  // User form
  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors },
  } = useForm<UserFormData>({
    resolver: zodResolver(userSchema),
  });

  // Filter users
  const filteredUsers = users.filter((user) => {
    const matchesSearch =
      user.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      user.email.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesRole = !selectedRole || user.role === selectedRole;
    const matchesStatus = !selectedStatus || user.status === selectedStatus;
    return matchesSearch && matchesRole && matchesStatus;
  });

  // Stats
  const totalCount = users.length;
  const activeCount = users.filter((u) => u.status === "active").length;
  const inactiveCount = users.filter((u) => u.status === "inactive").length;
  const lockedCount = users.filter((u) => u.status === "locked").length;

  // Open user modal for add/edit
  const openUserModal = (user?: MockUser) => {
    if (user) {
      setEditingUser(user);
      reset({
        email: user.email,
        name: user.name,
        phone: user.phone,
        role: user.role,
        departmentId: user.departmentId,
      });
    } else {
      setEditingUser(null);
      reset({
        email: "",
        name: "",
        phone: "",
        role: "user",
        departmentId: "",
      });
    }
    setUserModalOpen(true);
  };

  // Submit user form
  const onSubmitUser = async (data: UserFormData) => {
    try {
      if (editingUser) {
        // Update existing user
        setUsers(
          users.map((u) =>
            u.id === editingUser.id
              ? {
                  ...u,
                  ...data,
                  departmentName:
                    departmentOptions.find((d) => d.value === data.departmentId)
                      ?.label || u.departmentName,
                }
              : u
          )
        );
        toast.success("수정 완료", "사용자 정보가 수정되었습니다.");
      } else {
        // Add new user
        const newUser: MockUser = {
          id: crypto.randomUUID(),
          ...data,
          phone: data.phone || undefined,
          departmentName:
            departmentOptions.find((d) => d.value === data.departmentId)?.label ||
            undefined,
          status: "active",
          createdAt: new Date().toISOString().split("T")[0],
        };
        setUsers((prev) => [...prev, newUser]);
        toast.success("등록 완료", "새 사용자가 등록되었습니다. 임시 비밀번호가 이메일로 전송됩니다.");
      }
      setUserModalOpen(false);
    } catch {
      toast.error("저장 실패", "사용자 저장 중 오류가 발생했습니다.");
    }
  };

  // Delete user
  const handleDeleteUser = (id: string) => {
    const user = users.find((u) => u.id === id);
    if (user?.role === "admin" && users.filter((u) => u.role === "admin").length === 1) {
      toast.error("삭제 불가", "최소 1명의 관리자가 필요합니다.");
      return;
    }
    setUserToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDeleteUser = () => {
    if (userToDelete) {
      setUsers(users.filter((u) => u.id !== userToDelete));
      toast.success("삭제 완료", "사용자가 삭제되었습니다.");
    }
    setDeleteModalOpen(false);
    setUserToDelete(null);
  };

  // Toggle user status
  const toggleUserStatus = (user: MockUser) => {
    const newStatus: UserStatus =
      user.status === "active" ? "inactive" : "active";
    setUsers(
      users.map((u) => (u.id === user.id ? { ...u, status: newStatus } : u))
    );
    toast.success(
      "상태 변경",
      `사용자가 ${statusStyles[newStatus].label} 상태로 변경되었습니다.`
    );
  };

  // Unlock user
  const unlockUser = (user: MockUser) => {
    setUsers(
      users.map((u) => (u.id === user.id ? { ...u, status: "active" } : u))
    );
    toast.success("잠금 해제", "사용자 계정 잠금이 해제되었습니다.");
  };

  // Reset password
  const handleResetPassword = (user: MockUser) => {
    setUserToResetPassword(user);
    setResetPasswordModalOpen(true);
  };

  const confirmResetPassword = () => {
    if (userToResetPassword) {
      // TODO: API call to reset password
      toast.success(
        "비밀번호 초기화",
        `${userToResetPassword.email}로 임시 비밀번호가 전송되었습니다.`
      );
    }
    setResetPasswordModalOpen(false);
    setUserToResetPassword(null);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">사용자 관리</h1>
          <p className="text-muted-foreground">시스템 사용자를 등록하고 관리합니다.</p>
        </div>
        <Button onClick={() => openUserModal()}>
          <Plus className="h-4 w-4 mr-2" />
          사용자 등록
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center">
              <Users className="h-8 w-8 text-muted-foreground mr-3" />
              <div>
                <p className="text-sm text-muted-foreground">전체 사용자</p>
                <p className="text-2xl font-bold">{totalCount}명</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center">
              <CheckCircle className="h-8 w-8 text-success mr-3" />
              <div>
                <p className="text-sm text-muted-foreground">활성</p>
                <p className="text-2xl font-bold text-success">{activeCount}명</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center">
              <XCircle className="h-8 w-8 text-muted-foreground mr-3" />
              <div>
                <p className="text-sm text-muted-foreground">비활성</p>
                <p className="text-2xl font-bold">{inactiveCount}명</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center">
              <Lock className="h-8 w-8 text-destructive mr-3" />
              <div>
                <p className="text-sm text-muted-foreground">잠금</p>
                <p className="text-2xl font-bold text-destructive">{lockedCount}명</p>
              </div>
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
                  placeholder="이름, 이메일로 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedRole}
              onChange={(e) => setSelectedRole(e.target.value)}
            >
              <option value="">전체 권한</option>
              {roleOptions.map((role) => (
                <option key={role.value} value={role.value}>
                  {role.label}
                </option>
              ))}
            </select>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedStatus}
              onChange={(e) => setSelectedStatus(e.target.value)}
            >
              <option value="">전체 상태</option>
              <option value="active">활성</option>
              <option value="inactive">비활성</option>
              <option value="locked">잠금</option>
            </select>
            <Button variant="outline">
              <Download className="h-4 w-4 mr-2" />
              내보내기
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* User Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">사용자 목록</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>사용자</TableHead>
                <TableHead>권한</TableHead>
                <TableHead>부서</TableHead>
                <TableHead>상태</TableHead>
                <TableHead>최근 로그인</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredUsers.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    검색 결과가 없습니다.
                  </TableCell>
                </TableRow>
              ) : (
                filteredUsers.map((user) => {
                  const StatusIcon = statusStyles[user.status].icon;
                  return (
                    <TableRow key={user.id}>
                      <TableCell>
                        <div className="flex items-center space-x-3">
                          <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                            <span className="text-sm font-medium text-primary">
                              {user.name.charAt(0)}
                            </span>
                          </div>
                          <div>
                            <p className="font-medium">{user.name}</p>
                            <div className="flex items-center text-sm text-muted-foreground">
                              <Mail className="h-3 w-3 mr-1" />
                              {user.email}
                            </div>
                            {user.phone && (
                              <div className="flex items-center text-sm text-muted-foreground">
                                <Phone className="h-3 w-3 mr-1" />
                                {formatPhoneNumber(user.phone)}
                              </div>
                            )}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="flex items-center w-fit">
                          <Shield className="h-3 w-3 mr-1" />
                          {USER_ROLES[user.role]}
                        </Badge>
                      </TableCell>
                      <TableCell>{user.departmentName || "-"}</TableCell>
                      <TableCell>
                        <Badge variant={statusStyles[user.status].variant}>
                          <StatusIcon className="h-3 w-3 mr-1" />
                          {statusStyles[user.status].label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {user.lastLoginAt ? (
                          <div className="flex items-center text-sm text-muted-foreground">
                            <Clock className="h-3 w-3 mr-1" />
                            {formatDate(user.lastLoginAt, { format: "time" })}
                          </div>
                        ) : (
                          "-"
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="relative group">
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                          <div className="absolute right-0 hidden group-hover:block z-10">
                            <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[140px]">
                              <button
                                onClick={() => openUserModal(user)}
                                className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                              >
                                <Edit className="h-4 w-4 mr-2" />
                                수정
                              </button>
                              <button
                                onClick={() => handleResetPassword(user)}
                                className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                              >
                                <Key className="h-4 w-4 mr-2" />
                                비밀번호 초기화
                              </button>
                              {user.status === "locked" ? (
                                <button
                                  onClick={() => unlockUser(user)}
                                  className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                                >
                                  <Unlock className="h-4 w-4 mr-2" />
                                  잠금 해제
                                </button>
                              ) : (
                                <button
                                  onClick={() => toggleUserStatus(user)}
                                  className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                                >
                                  {user.status === "active" ? (
                                    <>
                                      <XCircle className="h-4 w-4 mr-2" />
                                      비활성화
                                    </>
                                  ) : (
                                    <>
                                      <CheckCircle className="h-4 w-4 mr-2" />
                                      활성화
                                    </>
                                  )}
                                </button>
                              )}
                              <button
                                onClick={() => handleDeleteUser(user.id)}
                                className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted text-destructive"
                              >
                                <Trash2 className="h-4 w-4 mr-2" />
                                삭제
                              </button>
                            </div>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>

          {/* Pagination */}
          <div className="flex items-center justify-between mt-4 pt-4 border-t">
            <p className="text-sm text-muted-foreground">총 {filteredUsers.length}명</p>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" disabled>
                이전
              </Button>
              <Button variant="outline" size="sm" className="bg-primary text-primary-foreground">
                1
              </Button>
              <Button variant="outline" size="sm">
                다음
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* User Add/Edit Modal */}
      <Modal
        isOpen={userModalOpen}
        onClose={() => setUserModalOpen(false)}
        title={editingUser ? "사용자 수정" : "사용자 등록"}
        size="lg"
      >
        <form onSubmit={handleSubmit(onSubmitUser)} className="space-y-4">
          <Input
            label="이메일"
            type="email"
            required
            placeholder="user@example.com"
            error={errors.email?.message}
            disabled={!!editingUser}
            {...register("email")}
          />
          <Input
            label="이름"
            required
            placeholder="사용자 이름"
            error={errors.name?.message}
            {...register("name")}
          />
          <Input
            label="연락처"
            placeholder="'-' 없이 입력"
            {...register("phone")}
          />
          <Controller
            name="role"
            control={control}
            render={({ field }) => (
              <Select
                label="권한"
                required
                options={roleOptions}
                error={errors.role?.message}
                value={field.value}
                onChange={field.onChange}
              />
            )}
          />
          <Controller
            name="departmentId"
            control={control}
            render={({ field }) => (
              <Select
                label="부서"
                options={[{ value: "", label: "부서 선택" }, ...departmentOptions]}
                value={field.value}
                onChange={field.onChange}
              />
            )}
          />
          {!editingUser && (
            <p className="text-sm text-muted-foreground">
              등록 완료 시 입력한 이메일로 임시 비밀번호가 전송됩니다.
            </p>
          )}
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setUserModalOpen(false)}
            >
              취소
            </Button>
            <Button type="submit">{editingUser ? "수정" : "등록"}</Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="사용자 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            이 사용자를 삭제하시겠습니까? 삭제된 사용자는 더 이상 시스템에 접근할 수 없습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              취소
            </Button>
            <Button variant="destructive" onClick={confirmDeleteUser}>
              삭제
            </Button>
          </div>
        </div>
      </Modal>

      {/* Reset Password Confirmation Modal */}
      <Modal
        isOpen={resetPasswordModalOpen}
        onClose={() => setResetPasswordModalOpen(false)}
        title="비밀번호 초기화"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            {userToResetPassword?.name} 사용자의 비밀번호를 초기화하시겠습니까?
          </p>
          <p className="text-sm text-muted-foreground">
            임시 비밀번호가 {userToResetPassword?.email}로 전송됩니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => setResetPasswordModalOpen(false)}
            >
              취소
            </Button>
            <Button onClick={confirmResetPassword}>초기화</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
