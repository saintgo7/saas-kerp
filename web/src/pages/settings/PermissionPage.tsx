import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import {
  Shield,
  Plus,
  Edit,
  Trash2,
  Save,
  Eye,
  FileEdit,
  FilePlus,
  FileX,
  ChevronDown,
  ChevronRight,
  Check,
  X,
} from "lucide-react";
import {
  Button,
  Input,
  Textarea,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Modal,
} from "@/components/ui";
import { apiClient } from "@/api";
import { getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";
import { cn } from "@/lib/utils";

// Server contract: internal/dto/role_dto.go
interface ApiPermission {
  code: string;
  name: string;
  description?: string;
  module: string;
}

interface ApiRole {
  id: string;
  code: string;
  name: string;
  description?: string;
  permissions: ApiPermission[];
  is_system: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Validation schema for role
const roleSchema = z.object({
  code: z
    .string()
    .min(1, "역할 코드를 입력하세요")
    .max(50, "역할 코드는 50자 이하입니다")
    .regex(/^[A-Za-z0-9_-]+$/, "영문, 숫자, '-', '_' 만 사용할 수 있습니다"),
  name: z.string().min(1, "역할명을 입력하세요").max(100, "역할명은 100자 이하입니다"),
  description: z.string().max(500, "설명은 500자 이하입니다").optional(),
});

type RoleFormData = z.infer<typeof roleSchema>;

// Permission types
const PERMISSION_ACTIONS = ["view", "create", "edit", "delete"] as const;
type PermissionAction = (typeof PERMISSION_ACTIONS)[number];

const actionIcons: Record<PermissionAction, typeof Eye> = {
  view: Eye,
  create: FilePlus,
  edit: FileEdit,
  delete: FileX,
};

const actionLabels: Record<PermissionAction, string> = {
  view: "조회",
  create: "등록",
  edit: "수정",
  delete: "삭제",
};

interface MenuNode {
  id: string;
  name: string;
  children?: MenuNode[];
}

/**
 * Static catalogue of the application's menus. The server stores permissions as
 * a flat list of `{code, name, description, module}` records, so each cell of
 * this matrix maps to the code `<menuId>:<action>`.
 */
const MENU_CATALOG: MenuNode[] = [
  { id: "dashboard", name: "대시보드" },
  {
    id: "accounting",
    name: "회계관리",
    children: [
      { id: "voucher", name: "전표관리" },
      { id: "ledger", name: "원장조회" },
      { id: "trial-balance", name: "시산표" },
      { id: "financial-statements", name: "재무제표" },
      { id: "accounts", name: "계정과목관리" },
    ],
  },
  {
    id: "invoice",
    name: "세금계산서",
    children: [
      { id: "invoice-issue", name: "매출발행" },
      { id: "invoice-received", name: "매입관리" },
      { id: "invoice-list", name: "발행내역" },
      { id: "invoice-hometax", name: "홈택스연동" },
    ],
  },
  {
    id: "hr",
    name: "인사/급여",
    children: [
      { id: "employee", name: "직원관리" },
      { id: "department", name: "부서관리" },
      { id: "payroll", name: "급여관리" },
      { id: "insurance", name: "4대보험" },
    ],
  },
  { id: "partners", name: "거래처관리" },
  {
    id: "inventory",
    name: "재고관리",
    children: [
      { id: "products", name: "품목관리" },
      { id: "stock", name: "재고현황" },
      { id: "purchase", name: "구매관리" },
      { id: "sales", name: "판매관리" },
    ],
  },
  { id: "reports", name: "보고서" },
  {
    id: "settings",
    name: "설정",
    children: [
      { id: "company", name: "회사정보" },
      { id: "users", name: "사용자관리" },
      { id: "permissions", name: "권한관리" },
      { id: "integrations", name: "연동설정" },
    ],
  },
];

const permissionCode = (menuId: string, action: PermissionAction): string =>
  `${menuId}:${action}`;

const buildPermissionCatalog = (
  menus: MenuNode[],
  moduleId?: string,
  acc: Map<string, ApiPermission> = new Map()
): Map<string, ApiPermission> => {
  menus.forEach((menu) => {
    const module = moduleId ?? menu.id;
    PERMISSION_ACTIONS.forEach((action) => {
      const code = permissionCode(menu.id, action);
      acc.set(code, {
        code,
        name: `${menu.name} ${actionLabels[action]}`,
        module,
      });
    });
    if (menu.children) {
      buildPermissionCatalog(menu.children, module, acc);
    }
  });
  return acc;
};

const PERMISSION_CATALOG = buildPermissionCatalog(MENU_CATALOG);

export function PermissionPage() {
  const queryClient = useQueryClient();

  const [selectedRoleId, setSelectedRoleId] = useState<string | null>(null);
  const [draftCodes, setDraftCodes] = useState<Set<string>>(new Set());
  const [roleModalOpen, setRoleModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<ApiRole | null>(null);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [roleToDelete, setRoleToDelete] = useState<ApiRole | null>(null);
  const [checkingDelete, setCheckingDelete] = useState(false);
  const [expandedMenus, setExpandedMenus] = useState<Set<string>>(
    new Set(["accounting", "invoice", "hr", "inventory", "settings"])
  );

  const {
    data: rolesResponse,
    isLoading,
    error: listError,
  } = useQuery({
    queryKey: ["roles"],
    queryFn: () => apiClient.get<ApiRole[]>("/roles"),
  });

  const roles = useMemo(() => rolesResponse?.data ?? [], [rolesResponse]);

  // Keep a valid selection whenever the role list changes
  useEffect(() => {
    if (roles.length === 0) {
      setSelectedRoleId(null);
      return;
    }
    setSelectedRoleId((current) =>
      current && roles.some((role) => role.id === current) ? current : roles[0].id
    );
  }, [roles]);

  const selectedRole = useMemo(
    () => roles.find((role) => role.id === selectedRoleId) ?? null,
    [roles, selectedRoleId]
  );

  const serverCodes = useMemo(
    () => new Set((selectedRole?.permissions ?? []).map((p) => p.code)),
    [selectedRole]
  );

  // Reset the working copy whenever the selected role (or its server state) changes
  useEffect(() => {
    setDraftCodes(new Set(serverCodes));
  }, [serverCodes]);

  const hasChanges =
    draftCodes.size !== serverCodes.size ||
    Array.from(draftCodes).some((code) => !serverCodes.has(code));

  // Role form
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<RoleFormData>({
    resolver: zodResolver(roleSchema),
    defaultValues: { code: "", name: "", description: "" },
  });

  const createRoleMutation = useMutation({
    mutationFn: (payload: RoleFormData) => apiClient.post<ApiRole>("/roles", payload),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
      setRoleModalOpen(false);
      if (response.data?.id) {
        setSelectedRoleId(response.data.id);
      }
      toast.success("등록 완료", "새 역할이 등록되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error("저장 실패", getErrorMessage(err, "역할 등록 중 오류가 발생했습니다."));
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: RoleFormData }) =>
      apiClient.put<ApiRole>(`/roles/${id}`, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
      setRoleModalOpen(false);
      toast.success("수정 완료", "역할이 수정되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error("저장 실패", getErrorMessage(err, "역할 수정 중 오류가 발생했습니다."));
    },
  });

  const deleteRoleMutation = useMutation({
    mutationFn: (id: string) => apiClient.delete<{ deleted: boolean }>(`/roles/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
      setDeleteModalOpen(false);
      setRoleToDelete(null);
      toast.success("삭제 완료", "역할이 삭제되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error("삭제 실패", getErrorMessage(err, "역할 삭제 중 오류가 발생했습니다."));
    },
  });

  const savePermissionsMutation = useMutation({
    mutationFn: ({ id, permissions }: { id: string; permissions: ApiPermission[] }) =>
      apiClient.put<ApiRole>(`/roles/${id}/permissions`, { permissions }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
      toast.success("저장 완료", "권한 설정이 저장되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error("저장 실패", getErrorMessage(err, "권한 저장 중 오류가 발생했습니다."));
    },
  });

  // Toggle menu expansion
  const toggleMenu = (menuId: string) => {
    setExpandedMenus((current) => {
      const next = new Set(current);
      if (next.has(menuId)) {
        next.delete(menuId);
      } else {
        next.add(menuId);
      }
      return next;
    });
  };

  // Open role modal for add/edit
  const openRoleModal = (role?: ApiRole) => {
    if (role) {
      setEditingRole(role);
      reset({
        code: role.code,
        name: role.name,
        description: role.description ?? "",
      });
    } else {
      setEditingRole(null);
      reset({ code: "", name: "", description: "" });
    }
    setRoleModalOpen(true);
  };

  // Submit role form
  const onSubmitRole = (data: RoleFormData) => {
    if (editingRole) {
      updateRoleMutation.mutate({ id: editingRole.id, payload: data });
      return;
    }
    createRoleMutation.mutate(data);
  };

  // Delete role - the server decides whether it is removable
  const handleDeleteRole = async (role: ApiRole) => {
    if (role.is_system) {
      toast.error("삭제 불가", "시스템 역할은 삭제할 수 없습니다.");
      return;
    }
    setCheckingDelete(true);
    try {
      const response = await apiClient.get<{ can_delete: boolean; reason: string }>(
        `/roles/${role.id}/can-delete`
      );
      if (!response.data?.can_delete) {
        toast.error("삭제 불가", response.data?.reason || "이 역할은 삭제할 수 없습니다.");
        return;
      }
      setRoleToDelete(role);
      setDeleteModalOpen(true);
    } catch (err) {
      toast.error(
        "삭제 불가",
        getErrorMessage(err, "삭제 가능 여부를 확인하지 못했습니다.")
      );
    } finally {
      setCheckingDelete(false);
    }
  };

  const confirmDeleteRole = () => {
    if (roleToDelete) {
      deleteRoleMutation.mutate(roleToDelete.id);
    }
  };

  // Toggle a single permission cell in the working copy
  const togglePermission = (menuId: string, action: PermissionAction) => {
    if (!selectedRole || selectedRole.is_system) return;
    const code = permissionCode(menuId, action);
    setDraftCodes((current) => {
      const next = new Set(current);
      if (next.has(code)) {
        next.delete(code);
      } else {
        next.add(code);
      }
      return next;
    });
  };

  // Save permissions
  const savePermissions = () => {
    if (!selectedRole) return;
    // Codes the server already holds but this screen does not model are kept as-is
    const serverByCode = new Map(
      (selectedRole.permissions ?? []).map((p) => [p.code, p])
    );
    const permissions = Array.from(draftCodes)
      .map((code) => PERMISSION_CATALOG.get(code) ?? serverByCode.get(code))
      .filter((permission): permission is ApiPermission => Boolean(permission));

    savePermissionsMutation.mutate({ id: selectedRole.id, permissions });
  };

  // Render permission checkbox
  const renderPermissionCheckbox = (menu: MenuNode, action: PermissionAction) => {
    const isEnabled = draftCodes.has(permissionCode(menu.id, action));
    const isDisabled = selectedRole?.is_system ?? true;

    return (
      <button
        type="button"
        onClick={() => !isDisabled && togglePermission(menu.id, action)}
        className={cn(
          "w-8 h-8 rounded flex items-center justify-center transition-colors",
          isEnabled
            ? "bg-primary text-primary-foreground"
            : "bg-muted text-muted-foreground hover:bg-muted/80",
          isDisabled && "cursor-not-allowed opacity-50"
        )}
        disabled={isDisabled}
        title={actionLabels[action]}
      >
        {isEnabled ? <Check className="h-4 w-4" /> : <X className="h-4 w-4" />}
      </button>
    );
  };

  // Render menu row
  const renderMenuRow = (menu: MenuNode, level: number = 0) => {
    const hasChildren = !!menu.children && menu.children.length > 0;
    const isExpanded = expandedMenus.has(menu.id);

    return (
      <div key={menu.id}>
        <div
          className={cn(
            "grid grid-cols-6 gap-2 items-center py-2 px-3 hover:bg-muted/50",
            level > 0 && "pl-8"
          )}
        >
          {/* Menu Name */}
          <div className="col-span-2 flex items-center">
            {hasChildren ? (
              <button
                type="button"
                onClick={() => toggleMenu(menu.id)}
                className="mr-2 p-1 hover:bg-muted rounded"
              >
                {isExpanded ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronRight className="h-4 w-4" />
                )}
              </button>
            ) : (
              <span className="w-6" />
            )}
            <span className={cn("text-sm", level === 0 && "font-medium")}>
              {menu.name}
            </span>
          </div>

          {/* Permission Checkboxes */}
          {PERMISSION_ACTIONS.map((action) => (
            <div key={action} className="flex justify-center">
              {renderPermissionCheckbox(menu, action)}
            </div>
          ))}
        </div>

        {/* Children */}
        {hasChildren && isExpanded && (
          <div className="border-l ml-6">
            {menu.children!.map((child) => renderMenuRow(child, level + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">권한 관리</h1>
          <p className="text-muted-foreground">역할별 메뉴 및 기능 접근 권한을 설정합니다.</p>
        </div>
        <Button onClick={() => openRoleModal()}>
          <Plus className="h-4 w-4 mr-2" />
          역할 추가
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Role List */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <Shield className="h-5 w-5 mr-2" />
              역할 목록
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {isLoading ? (
              <p className="p-4 text-sm text-muted-foreground">불러오는 중...</p>
            ) : listError ? (
              <p className="p-4 text-sm text-destructive">
                {getErrorMessage(listError, "역할 목록을 불러오지 못했습니다.")}
              </p>
            ) : roles.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">
                등록된 역할이 없습니다.
              </p>
            ) : (
              <div className="divide-y">
                {roles.map((role) => (
                  <div
                    key={role.id}
                    className={cn(
                      "p-4 cursor-pointer hover:bg-muted/50 transition-colors",
                      selectedRoleId === role.id && "bg-muted"
                    )}
                    onClick={() => setSelectedRoleId(role.id)}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="flex items-center">
                          <span className="font-medium">{role.name}</span>
                          {role.is_system && (
                            <Badge variant="secondary" className="ml-2 text-xs">
                              시스템
                            </Badge>
                          )}
                          {!role.is_active && (
                            <Badge variant="outline" className="ml-2 text-xs">
                              비활성
                            </Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground mt-1 line-clamp-1">
                          {role.description}
                        </p>
                      </div>
                      <div className="flex items-center space-x-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={(e) => {
                            e.stopPropagation();
                            openRoleModal(role);
                          }}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        {!role.is_system && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-destructive hover:text-destructive"
                            disabled={checkingDelete}
                            onClick={(e) => {
                              e.stopPropagation();
                              void handleDeleteRole(role);
                            }}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center mt-2 text-sm text-muted-foreground">
                      <Shield className="h-3 w-3 mr-1" />
                      {role.code}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Permission Matrix */}
        <Card className="lg:col-span-3">
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle className="text-lg">
                {selectedRole?.name} 권한 설정
              </CardTitle>
              {selectedRole?.is_system && (
                <p className="text-sm text-muted-foreground mt-1">
                  시스템 역할의 권한은 수정할 수 없습니다.
                </p>
              )}
            </div>
            {hasChanges && !selectedRole?.is_system && (
              <Button
                onClick={savePermissions}
                isLoading={savePermissionsMutation.isPending}
              >
                <Save className="h-4 w-4 mr-2" />
                저장
              </Button>
            )}
          </CardHeader>
          <CardContent>
            {selectedRole ? (
              <div>
                {/* Header */}
                <div className="grid grid-cols-6 gap-2 items-center py-2 px-3 bg-muted rounded-t-lg font-medium text-sm">
                  <div className="col-span-2">메뉴</div>
                  {PERMISSION_ACTIONS.map((action) => {
                    const Icon = actionIcons[action];
                    return (
                      <div
                        key={action}
                        className="flex flex-col items-center justify-center"
                      >
                        <Icon className="h-4 w-4 mb-1" />
                        <span className="text-xs">{actionLabels[action]}</span>
                      </div>
                    );
                  })}
                </div>

                {/* Permission Rows */}
                <div className="border rounded-b-lg divide-y">
                  {MENU_CATALOG.map((menu) => renderMenuRow(menu))}
                </div>
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                역할을 선택하세요.
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Role Add/Edit Modal */}
      <Modal
        isOpen={roleModalOpen}
        onClose={() => setRoleModalOpen(false)}
        title={editingRole ? "역할 수정" : "역할 추가"}
        size="md"
      >
        <form onSubmit={handleSubmit(onSubmitRole)} className="space-y-4">
          <Input
            label="역할 코드"
            required
            placeholder="예: sales_manager"
            error={errors.code?.message}
            disabled={editingRole?.is_system}
            {...register("code")}
          />
          <Input
            label="역할명"
            required
            placeholder="예: 영업담당"
            error={errors.name?.message}
            disabled={editingRole?.is_system}
            {...register("name")}
          />
          <Textarea
            label="설명"
            placeholder="이 역할에 대한 설명을 입력하세요"
            disabled={editingRole?.is_system}
            {...register("description")}
          />
          {editingRole?.is_system && (
            <p className="text-sm text-muted-foreground">
              시스템 역할의 이름은 변경할 수 없습니다.
            </p>
          )}
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setRoleModalOpen(false)}
            >
              취소
            </Button>
            <Button
              type="submit"
              disabled={editingRole?.is_system}
              isLoading={createRoleMutation.isPending || updateRoleMutation.isPending}
            >
              {editingRole ? "수정" : "추가"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="역할 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            {roleToDelete?.name} 역할을 삭제하시겠습니까? 삭제된 역할은 복구할 수 없습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              취소
            </Button>
            <Button
              variant="destructive"
              isLoading={deleteRoleMutation.isPending}
              onClick={confirmDeleteRole}
            >
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
