import { useEffect, useState } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { FeatureUnavailable, notifyUnavailable } from "@/components/common";
import { formatDate } from "@/lib/utils";
import { apiClient } from "@/api";
import { getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";
import { USER_ROLES, DEFAULT_PAGE_SIZE } from "@/constants";
import type { UserRole } from "@/types";

// Server contract: internal/dto/user_dto.go
interface ApiUser {
  id: string;
  email: string;
  name: string;
  role: string;
  status: string;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

interface UserStats {
  total_count: number;
  active_count: number;
  inactive_count: number;
  locked_count: number;
  admin_count: number;
  user_count: number;
  viewer_count: number;
}

// Validation schema for user. Roles must match internal/domain/user.go.
const userSchema = z.object({
  email: z.string().email("올바른 이메일 형식이 아닙니다"),
  name: z.string().min(1, "이름을 입력하세요"),
  role: z.enum(["admin", "user", "viewer"]),
  password: z.string().optional(),
});

type UserFormData = z.infer<typeof userSchema>;

// User status type
type UserStatus = "active" | "inactive" | "locked";

// Role options for select
const roleOptions = Object.entries(USER_ROLES).map(([value, label]) => ({
  value,
  label,
}));

const toUserRole = (value: string): UserRole =>
  value === "admin" || value === "viewer" ? value : "user";

// Status badge styles
const statusStyles: Record<UserStatus, { variant: "success" | "secondary" | "destructive"; label: string; icon: typeof CheckCircle }> = {
  active: { variant: "success", label: "활성", icon: CheckCircle },
  inactive: { variant: "secondary", label: "비활성", icon: XCircle },
  locked: { variant: "destructive", label: "잠금", icon: Lock },
};

const styleForStatus = (status: string) =>
  statusStyles[status as UserStatus] ?? statusStyles.inactive;

export function UserManagementPage() {
  const queryClient = useQueryClient();

  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [selectedRole, setSelectedRole] = useState<string>("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [page, setPage] = useState(1);
  const [userModalOpen, setUserModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<ApiUser | null>(null);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [userToDelete, setUserToDelete] = useState<ApiUser | null>(null);

  // Debounce the search box so typing does not fire a request per keystroke
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearch(searchTerm);
      setPage(1);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [searchTerm]);

  const listParams = {
    page,
    page_size: DEFAULT_PAGE_SIZE,
    search: debouncedSearch || undefined,
    role: selectedRole || undefined,
    status: selectedStatus || undefined,
  };

  const {
    data: usersResponse,
    isLoading,
    isFetching,
    error: listError,
  } = useQuery({
    queryKey: ["users", listParams],
    queryFn: () => apiClient.get<ApiUser[]>("/users", listParams),
  });

  const { data: statsResponse } = useQuery({
    queryKey: ["user-stats"],
    queryFn: () => apiClient.get<UserStats>("/users/stats"),
  });

  const users = usersResponse?.data ?? [];
  // internal/dto/common.go: meta.pagination.{page, per_page, total, total_pages}
  const pagination = usersResponse?.meta?.pagination;
  const totalCount = pagination?.total ?? users.length;
  const totalPages = pagination?.total_pages ?? 1;
  const stats = statsResponse?.data;

  // User form
  const {
    register,
    handleSubmit,
    reset,
    control,
    setError,
    formState: { errors },
  } = useForm<UserFormData>({
    resolver: zodResolver(userSchema),
    defaultValues: { email: "", name: "", role: "user", password: "" },
  });

  const invalidateUsers = () => {
    queryClient.invalidateQueries({ queryKey: ["users"] });
    queryClient.invalidateQueries({ queryKey: ["user-stats"] });
  };

  const createUserMutation = useMutation({
    mutationFn: (payload: {
      email: string;
      password: string;
      name: string;
      role: UserRole;
    }) => apiClient.post<ApiUser>("/users", payload),
    onSuccess: () => {
      invalidateUsers();
      setUserModalOpen(false);
      toast.success("등록 완료", "새 사용자가 등록되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error(
        "등록 실패",
        getErrorMessage(err, "사용자 등록 중 오류가 발생했습니다.")
      );
    },
  });

  const updateUserMutation = useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: string;
      payload: { email: string; name: string; role: UserRole };
    }) => apiClient.put<ApiUser>(`/users/${id}`, payload),
    onSuccess: () => {
      invalidateUsers();
      setUserModalOpen(false);
      toast.success("수정 완료", "사용자 정보가 수정되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error(
        "수정 실패",
        getErrorMessage(err, "사용자 수정 중 오류가 발생했습니다.")
      );
    },
  });

  const deleteUserMutation = useMutation({
    mutationFn: (id: string) => apiClient.delete<{ deleted: boolean }>(`/users/${id}`),
    onSuccess: () => {
      invalidateUsers();
      setDeleteModalOpen(false);
      setUserToDelete(null);
      toast.success("삭제 완료", "사용자가 삭제되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error(
        "삭제 실패",
        getErrorMessage(err, "사용자 삭제 중 오류가 발생했습니다.")
      );
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, action }: { id: string; action: "activate" | "deactivate" }) =>
      apiClient.post<{ activated?: boolean; deactivated?: boolean }>(
        `/users/${id}/${action}`,
        {}
      ),
    onSuccess: (_data, variables) => {
      invalidateUsers();
      toast.success(
        "상태 변경",
        variables.action === "activate"
          ? "사용자가 활성 상태로 변경되었습니다."
          : "사용자가 비활성 상태로 변경되었습니다."
      );
    },
    onError: (err: unknown) => {
      toast.error(
        "상태 변경 실패",
        getErrorMessage(err, "사용자 상태 변경 중 오류가 발생했습니다.")
      );
    },
  });

  // Open user modal for add/edit
  const openUserModal = (user?: ApiUser) => {
    if (user) {
      setEditingUser(user);
      reset({
        email: user.email,
        name: user.name,
        role: toUserRole(user.role),
        password: "",
      });
    } else {
      setEditingUser(null);
      reset({
        email: "",
        name: "",
        role: "user",
        password: "",
      });
    }
    setUserModalOpen(true);
  };

  // Submit user form
  const onSubmitUser = (data: UserFormData) => {
    if (editingUser) {
      updateUserMutation.mutate({
        id: editingUser.id,
        payload: {
          email: data.email,
          name: data.name,
          role: data.role,
        },
      });
      return;
    }

    // POST /users requires an initial password (min 8 chars)
    if (!data.password || data.password.length < 8) {
      setError("password", {
        type: "manual",
        message: "초기 비밀번호는 8자 이상 입력하세요",
      });
      return;
    }

    createUserMutation.mutate({
      email: data.email,
      password: data.password,
      name: data.name,
      role: data.role,
    });
  };

  // Delete user
  const handleDeleteUser = (user: ApiUser) => {
    if (user.role === "admin" && (stats?.admin_count ?? 0) <= 1) {
      toast.error("삭제 불가", "최소 1명의 관리자가 필요합니다.");
      return;
    }
    setUserToDelete(user);
    setDeleteModalOpen(true);
  };

  const confirmDeleteUser = () => {
    if (userToDelete) {
      deleteUserMutation.mutate(userToDelete.id);
    }
  };

  // Toggle user status
  const toggleUserStatus = (user: ApiUser) => {
    statusMutation.mutate({
      id: user.id,
      action: user.status === "active" ? "deactivate" : "activate",
    });
  };

  // Unlock user (the server's activate endpoint clears the locked status)
  const unlockUser = (user: ApiUser) => {
    statusMutation.mutate({ id: user.id, action: "activate" });
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
                <p className="text-2xl font-bold">{stats?.total_count ?? 0}명</p>
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
                <p className="text-2xl font-bold text-success">
                  {stats?.active_count ?? 0}명
                </p>
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
                <p className="text-2xl font-bold">{stats?.inactive_count ?? 0}명</p>
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
                <p className="text-2xl font-bold text-destructive">
                  {stats?.locked_count ?? 0}명
                </p>
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
              onChange={(e) => {
                setSelectedRole(e.target.value);
                setPage(1);
              }}
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
              onChange={(e) => {
                setSelectedStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">전체 상태</option>
              <option value="active">활성</option>
              <option value="inactive">비활성</option>
              <option value="locked">잠금</option>
            </select>
            <Button
              variant="outline"
              onClick={() => notifyUnavailable("사용자 목록 내보내기")}
            >
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
                <TableHead>상태</TableHead>
                <TableHead>최근 로그인</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                    불러오는 중...
                  </TableCell>
                </TableRow>
              ) : listError ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8 text-destructive">
                    {getErrorMessage(listError, "사용자 목록을 불러오지 못했습니다.")}
                  </TableCell>
                </TableRow>
              ) : users.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                    검색 결과가 없습니다.
                  </TableCell>
                </TableRow>
              ) : (
                users.map((user) => {
                  const status = styleForStatus(user.status);
                  const StatusIcon = status.icon;
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
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="flex items-center w-fit">
                          <Shield className="h-3 w-3 mr-1" />
                          {USER_ROLES[toUserRole(user.role)]}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={status.variant}>
                          <StatusIcon className="h-3 w-3 mr-1" />
                          {status.label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {user.last_login_at ? (
                          <div className="flex items-center text-sm text-muted-foreground">
                            <Clock className="h-3 w-3 mr-1" />
                            {formatDate(user.last_login_at, { format: "time" })}
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
                                type="button"
                                onClick={() => openUserModal(user)}
                                className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                              >
                                <Edit className="h-4 w-4 mr-2" />
                                수정
                              </button>
                              <button
                                type="button"
                                onClick={() => notifyUnavailable("비밀번호 초기화")}
                                className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                              >
                                <Key className="h-4 w-4 mr-2" />
                                비밀번호 초기화
                              </button>
                              {user.status === "locked" ? (
                                <button
                                  type="button"
                                  onClick={() => unlockUser(user)}
                                  className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                                >
                                  <Unlock className="h-4 w-4 mr-2" />
                                  잠금 해제
                                </button>
                              ) : (
                                <button
                                  type="button"
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
                                type="button"
                                onClick={() => handleDeleteUser(user)}
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
            <p className="text-sm text-muted-foreground">총 {totalCount}명</p>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1 || isFetching}
                onClick={() => setPage((prev) => Math.max(1, prev - 1))}
              >
                이전
              </Button>
              <Button variant="outline" size="sm" className="bg-primary text-primary-foreground">
                {page}
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages || isFetching}
                onClick={() => setPage((prev) => prev + 1)}
              >
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
          {!editingUser && (
            <Input
              label="초기 비밀번호"
              type="password"
              required
              placeholder="8자 이상"
              error={errors.password?.message}
              helperText="서버에 임시 비밀번호 발송 기능이 없어 관리자가 직접 지정한 뒤 사용자에게 전달해야 합니다."
              {...register("password")}
            />
          )}
          <FeatureUnavailable
            feature="연락처·부서 지정"
            detail="서버의 사용자 API에 연락처와 부서 항목이 없어 저장할 수 없습니다."
          />
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setUserModalOpen(false)}
            >
              취소
            </Button>
            <Button
              type="submit"
              isLoading={createUserMutation.isPending || updateUserMutation.isPending}
            >
              {editingUser ? "수정" : "등록"}
            </Button>
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
            {userToDelete?.name} 사용자를 삭제하시겠습니까? 삭제된 사용자는 더 이상 시스템에 접근할 수 없습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              취소
            </Button>
            <Button
              variant="destructive"
              isLoading={deleteUserMutation.isPending}
              onClick={confirmDeleteUser}
            >
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
