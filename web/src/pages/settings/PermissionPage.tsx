import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Shield,
  Plus,
  Edit,
  Trash2,
  Save,
  Users,
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
import { toast } from "@/stores/ui";
import { cn } from "@/lib/utils";

// Validation schema for role
const roleSchema = z.object({
  name: z.string().min(1, "역할명을 입력하세요"),
  description: z.string().optional(),
});

type RoleFormData = z.infer<typeof roleSchema>;

// Permission types
type PermissionAction = "view" | "create" | "edit" | "delete";

interface Permission {
  view: boolean;
  create: boolean;
  edit: boolean;
  delete: boolean;
}

interface MenuPermission {
  id: string;
  name: string;
  permissions: Permission;
  children?: MenuPermission[];
}

interface Role {
  id: string;
  name: string;
  description?: string;
  isSystem: boolean;
  userCount: number;
  menuPermissions: MenuPermission[];
}

// Mock roles data
const defaultPermission: Permission = {
  view: false,
  create: false,
  edit: false,
  delete: false,
};

const createMenuStructure = (overrides: Partial<Record<string, Partial<Permission>>> = {}): MenuPermission[] => [
  {
    id: "dashboard",
    name: "대시보드",
    permissions: { ...defaultPermission, view: true, ...overrides["dashboard"] },
  },
  {
    id: "accounting",
    name: "회계관리",
    permissions: { ...defaultPermission, ...overrides["accounting"] },
    children: [
      {
        id: "voucher",
        name: "전표관리",
        permissions: { ...defaultPermission, ...overrides["voucher"] },
      },
      {
        id: "ledger",
        name: "원장조회",
        permissions: { ...defaultPermission, ...overrides["ledger"] },
      },
      {
        id: "trial-balance",
        name: "시산표",
        permissions: { ...defaultPermission, ...overrides["trial-balance"] },
      },
      {
        id: "financial-statements",
        name: "재무제표",
        permissions: { ...defaultPermission, ...overrides["financial-statements"] },
      },
      {
        id: "accounts",
        name: "계정과목관리",
        permissions: { ...defaultPermission, ...overrides["accounts"] },
      },
    ],
  },
  {
    id: "invoice",
    name: "세금계산서",
    permissions: { ...defaultPermission, ...overrides["invoice"] },
    children: [
      {
        id: "invoice-issue",
        name: "매출발행",
        permissions: { ...defaultPermission, ...overrides["invoice-issue"] },
      },
      {
        id: "invoice-received",
        name: "매입관리",
        permissions: { ...defaultPermission, ...overrides["invoice-received"] },
      },
      {
        id: "invoice-list",
        name: "발행내역",
        permissions: { ...defaultPermission, ...overrides["invoice-list"] },
      },
      {
        id: "invoice-hometax",
        name: "홈택스연동",
        permissions: { ...defaultPermission, ...overrides["invoice-hometax"] },
      },
    ],
  },
  {
    id: "hr",
    name: "인사/급여",
    permissions: { ...defaultPermission, ...overrides["hr"] },
    children: [
      {
        id: "employee",
        name: "직원관리",
        permissions: { ...defaultPermission, ...overrides["employee"] },
      },
      {
        id: "department",
        name: "부서관리",
        permissions: { ...defaultPermission, ...overrides["department"] },
      },
      {
        id: "payroll",
        name: "급여관리",
        permissions: { ...defaultPermission, ...overrides["payroll"] },
      },
      {
        id: "insurance",
        name: "4대보험",
        permissions: { ...defaultPermission, ...overrides["insurance"] },
      },
    ],
  },
  {
    id: "partners",
    name: "거래처관리",
    permissions: { ...defaultPermission, ...overrides["partners"] },
  },
  {
    id: "inventory",
    name: "재고관리",
    permissions: { ...defaultPermission, ...overrides["inventory"] },
    children: [
      {
        id: "products",
        name: "품목관리",
        permissions: { ...defaultPermission, ...overrides["products"] },
      },
      {
        id: "stock",
        name: "재고현황",
        permissions: { ...defaultPermission, ...overrides["stock"] },
      },
      {
        id: "purchase",
        name: "구매관리",
        permissions: { ...defaultPermission, ...overrides["purchase"] },
      },
      {
        id: "sales",
        name: "판매관리",
        permissions: { ...defaultPermission, ...overrides["sales"] },
      },
    ],
  },
  {
    id: "reports",
    name: "보고서",
    permissions: { ...defaultPermission, ...overrides["reports"] },
  },
  {
    id: "settings",
    name: "설정",
    permissions: { ...defaultPermission, ...overrides["settings"] },
    children: [
      {
        id: "company",
        name: "회사정보",
        permissions: { ...defaultPermission, ...overrides["company"] },
      },
      {
        id: "users",
        name: "사용자관리",
        permissions: { ...defaultPermission, ...overrides["users"] },
      },
      {
        id: "permissions",
        name: "권한관리",
        permissions: { ...defaultPermission, ...overrides["permissions"] },
      },
      {
        id: "integrations",
        name: "연동설정",
        permissions: { ...defaultPermission, ...overrides["integrations"] },
      },
    ],
  },
];

// Full permissions for admin
const fullPermission: Permission = { view: true, create: true, edit: true, delete: true };

const mockRoles: Role[] = [
  {
    id: "admin",
    name: "관리자",
    description: "시스템의 모든 기능에 접근할 수 있습니다.",
    isSystem: true,
    userCount: 2,
    menuPermissions: createMenuStructure(
      Object.fromEntries(
        [
          "dashboard", "accounting", "voucher", "ledger", "trial-balance", "financial-statements",
          "accounts", "invoice", "invoice-issue", "invoice-received", "invoice-list", "invoice-hometax",
          "hr", "employee", "department", "payroll", "insurance", "partners", "inventory",
          "products", "stock", "purchase", "sales", "reports", "settings", "company",
          "users", "permissions", "integrations"
        ].map(k => [k, fullPermission])
      )
    ),
  },
  {
    id: "manager",
    name: "매니저",
    description: "대부분의 기능에 접근할 수 있지만 시스템 설정은 제한됩니다.",
    isSystem: true,
    userCount: 3,
    menuPermissions: createMenuStructure({
      dashboard: { view: true },
      accounting: { view: true, create: true, edit: true },
      voucher: { view: true, create: true, edit: true },
      ledger: { view: true },
      "trial-balance": { view: true },
      "financial-statements": { view: true },
      invoice: { view: true, create: true, edit: true },
      "invoice-issue": { view: true, create: true, edit: true },
      "invoice-received": { view: true },
      "invoice-list": { view: true },
      hr: { view: true },
      employee: { view: true },
      partners: { view: true, create: true, edit: true },
      inventory: { view: true, create: true, edit: true },
      products: { view: true, create: true, edit: true },
      stock: { view: true },
      purchase: { view: true, create: true, edit: true },
      sales: { view: true, create: true, edit: true },
      reports: { view: true },
      settings: { view: true },
      company: { view: true },
    }),
  },
  {
    id: "accountant",
    name: "회계담당",
    description: "회계 및 세금계산서 관련 기능에 접근할 수 있습니다.",
    isSystem: true,
    userCount: 4,
    menuPermissions: createMenuStructure({
      dashboard: { view: true },
      accounting: fullPermission,
      voucher: fullPermission,
      ledger: { view: true },
      "trial-balance": { view: true },
      "financial-statements": { view: true },
      accounts: fullPermission,
      invoice: fullPermission,
      "invoice-issue": fullPermission,
      "invoice-received": fullPermission,
      "invoice-list": { view: true },
      "invoice-hometax": fullPermission,
      partners: { view: true },
      reports: { view: true },
    }),
  },
  {
    id: "hr",
    name: "인사담당",
    description: "인사 및 급여 관련 기능에 접근할 수 있습니다.",
    isSystem: true,
    userCount: 2,
    menuPermissions: createMenuStructure({
      dashboard: { view: true },
      hr: fullPermission,
      employee: fullPermission,
      department: fullPermission,
      payroll: fullPermission,
      insurance: fullPermission,
      reports: { view: true },
    }),
  },
  {
    id: "user",
    name: "일반사용자",
    description: "기본적인 조회 기능만 사용할 수 있습니다.",
    isSystem: true,
    userCount: 10,
    menuPermissions: createMenuStructure({
      dashboard: { view: true },
      accounting: { view: true },
      voucher: { view: true },
      ledger: { view: true },
      invoice: { view: true },
      "invoice-list": { view: true },
      partners: { view: true },
      inventory: { view: true },
      stock: { view: true },
    }),
  },
];

// Permission action icons
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

export function PermissionPage() {
  const [roles, setRoles] = useState(mockRoles);
  const [selectedRole, setSelectedRole] = useState<Role | null>(mockRoles[0]);
  const [roleModalOpen, setRoleModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [roleToDelete, setRoleToDelete] = useState<string | null>(null);
  const [expandedMenus, setExpandedMenus] = useState<Set<string>>(new Set(["accounting", "invoice", "hr", "inventory", "settings"]));
  const [hasChanges, setHasChanges] = useState(false);

  // Role form
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<RoleFormData>({
    resolver: zodResolver(roleSchema),
  });

  // Toggle menu expansion
  const toggleMenu = (menuId: string) => {
    const newExpanded = new Set(expandedMenus);
    if (newExpanded.has(menuId)) {
      newExpanded.delete(menuId);
    } else {
      newExpanded.add(menuId);
    }
    setExpandedMenus(newExpanded);
  };

  // Open role modal for add/edit
  const openRoleModal = (role?: Role) => {
    if (role) {
      setEditingRole(role);
      reset({
        name: role.name,
        description: role.description,
      });
    } else {
      setEditingRole(null);
      reset({
        name: "",
        description: "",
      });
    }
    setRoleModalOpen(true);
  };

  // Submit role form
  const onSubmitRole = async (data: RoleFormData) => {
    try {
      if (editingRole) {
        // Update existing role
        setRoles(
          roles.map((r) =>
            r.id === editingRole.id ? { ...r, ...data } : r
          )
        );
        if (selectedRole?.id === editingRole.id) {
          setSelectedRole({ ...selectedRole, ...data });
        }
        toast.success("수정 완료", "역할이 수정되었습니다.");
      } else {
        // Add new role
        const newRole: Role = {
          id: crypto.randomUUID(),
          ...data,
          isSystem: false,
          userCount: 0,
          menuPermissions: createMenuStructure({ dashboard: { view: true } }),
        };
        setRoles((prev) => [...prev, newRole]);
        toast.success("등록 완료", "새 역할이 등록되었습니다.");
      }
      setRoleModalOpen(false);
    } catch {
      toast.error("저장 실패", "역할 저장 중 오류가 발생했습니다.");
    }
  };

  // Delete role
  const handleDeleteRole = (id: string) => {
    const role = roles.find((r) => r.id === id);
    if (role?.isSystem) {
      toast.error("삭제 불가", "시스템 역할은 삭제할 수 없습니다.");
      return;
    }
    if (role?.userCount && role.userCount > 0) {
      toast.error("삭제 불가", "이 역할에 할당된 사용자가 있습니다. 먼저 사용자의 역할을 변경해주세요.");
      return;
    }
    setRoleToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDeleteRole = () => {
    if (roleToDelete) {
      setRoles(roles.filter((r) => r.id !== roleToDelete));
      if (selectedRole?.id === roleToDelete) {
        setSelectedRole(roles[0] || null);
      }
      toast.success("삭제 완료", "역할이 삭제되었습니다.");
    }
    setDeleteModalOpen(false);
    setRoleToDelete(null);
  };

  // Toggle permission (parentId reserved for future cascade logic)
  const togglePermission = (menuId: string, action: PermissionAction, parentId?: string) => {
    void parentId; // Reserved for future cascade logic
    if (!selectedRole || selectedRole.isSystem) return;

    const updatePermissions = (menus: MenuPermission[]): MenuPermission[] => {
      return menus.map((menu) => {
        if (menu.id === menuId) {
          return {
            ...menu,
            permissions: {
              ...menu.permissions,
              [action]: !menu.permissions[action],
            },
          };
        }
        if (menu.children) {
          return {
            ...menu,
            children: updatePermissions(menu.children),
          };
        }
        return menu;
      });
    };

    const updatedMenuPermissions = updatePermissions(selectedRole.menuPermissions);
    const updatedRole = { ...selectedRole, menuPermissions: updatedMenuPermissions };

    setSelectedRole(updatedRole);
    setRoles(roles.map((r) => (r.id === selectedRole.id ? updatedRole : r)));
    setHasChanges(true);
  };

  // Save permissions
  const savePermissions = () => {
    // TODO: API call to save permissions
    toast.success("저장 완료", "권한 설정이 저장되었습니다.");
    setHasChanges(false);
  };

  // Render permission checkbox
  const renderPermissionCheckbox = (
    menu: MenuPermission,
    action: PermissionAction,
    parentId?: string
  ) => {
    const isEnabled = menu.permissions[action];
    const isDisabled = selectedRole?.isSystem;
    // ActionIcon is available if needed for future enhancement
    void actionIcons[action];

    return (
      <button
        type="button"
        onClick={() => !isDisabled && togglePermission(menu.id, action, parentId)}
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
  const renderMenuRow = (menu: MenuPermission, level: number = 0, parentId?: string) => {
    void parentId; // Reserved for cascade permission logic
    const hasChildren = menu.children && menu.children.length > 0;
    const isExpanded = expandedMenus.has(menu.id);
    // These can be used for visual indicators (partially selected, etc.)
    void Object.values(menu.permissions).every(Boolean);
    void Object.values(menu.permissions).some(Boolean);

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
          {(["view", "create", "edit", "delete"] as PermissionAction[]).map(
            (action) => (
              <div key={action} className="flex justify-center">
                {renderPermissionCheckbox(menu, action, parentId)}
              </div>
            )
          )}
        </div>

        {/* Children */}
        {hasChildren && isExpanded && (
          <div className="border-l ml-6">
            {menu.children!.map((child) => renderMenuRow(child, level + 1, menu.id))}
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
            <div className="divide-y">
              {roles.map((role) => (
                <div
                  key={role.id}
                  className={cn(
                    "p-4 cursor-pointer hover:bg-muted/50 transition-colors",
                    selectedRole?.id === role.id && "bg-muted"
                  )}
                  onClick={() => setSelectedRole(role)}
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="flex items-center">
                        <span className="font-medium">{role.name}</span>
                        {role.isSystem && (
                          <Badge variant="secondary" className="ml-2 text-xs">
                            시스템
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
                      {!role.isSystem && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="text-destructive hover:text-destructive"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteRole(role.id);
                          }}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center mt-2 text-sm text-muted-foreground">
                    <Users className="h-3 w-3 mr-1" />
                    {role.userCount}명 사용중
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Permission Matrix */}
        <Card className="lg:col-span-3">
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle className="text-lg">
                {selectedRole?.name} 권한 설정
              </CardTitle>
              {selectedRole?.isSystem && (
                <p className="text-sm text-muted-foreground mt-1">
                  시스템 역할의 권한은 수정할 수 없습니다.
                </p>
              )}
            </div>
            {hasChanges && !selectedRole?.isSystem && (
              <Button onClick={savePermissions}>
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
                  {(["view", "create", "edit", "delete"] as PermissionAction[]).map(
                    (action) => {
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
                    }
                  )}
                </div>

                {/* Permission Rows */}
                <div className="border rounded-b-lg divide-y">
                  {selectedRole.menuPermissions.map((menu) =>
                    renderMenuRow(menu)
                  )}
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
            label="역할명"
            required
            placeholder="예: 영업담당"
            error={errors.name?.message}
            disabled={editingRole?.isSystem}
            {...register("name")}
          />
          <Textarea
            label="설명"
            placeholder="이 역할에 대한 설명을 입력하세요"
            {...register("description")}
          />
          {editingRole?.isSystem && (
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
            <Button type="submit" disabled={editingRole?.isSystem}>
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
            이 역할을 삭제하시겠습니까? 삭제된 역할은 복구할 수 없습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              취소
            </Button>
            <Button variant="destructive" onClick={confirmDeleteRole}>
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
