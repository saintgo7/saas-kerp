import { useState } from "react";
import {
  Plus,
  Search,
  ChevronRight,
  ChevronDown,
  Edit,
  Trash2,
  Users,
  Building2,
  FolderTree,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Modal,
} from "@/components/ui";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "@/stores/ui";
import { cn } from "@/lib/utils";

// Types
interface Department {
  id: string;
  code: string;
  name: string;
  parentId: string | null;
  managerId?: string;
  managerName?: string;
  employeeCount: number;
  children?: Department[];
  level: number;
}

// Mock data
const mockDepartments: Department[] = [
  {
    id: "1",
    code: "CEO",
    name: "대표이사",
    parentId: null,
    managerId: "emp1",
    managerName: "김대표",
    employeeCount: 1,
    level: 0,
    children: [
      {
        id: "2",
        code: "DEV",
        name: "개발본부",
        parentId: "1",
        managerId: "emp2",
        managerName: "이개발",
        employeeCount: 15,
        level: 1,
        children: [
          {
            id: "5",
            code: "DEV-FE",
            name: "프론트엔드팀",
            parentId: "2",
            managerId: "emp5",
            managerName: "박프론트",
            employeeCount: 6,
            level: 2,
          },
          {
            id: "6",
            code: "DEV-BE",
            name: "백엔드팀",
            parentId: "2",
            managerId: "emp6",
            managerName: "최백엔드",
            employeeCount: 5,
            level: 2,
          },
          {
            id: "7",
            code: "DEV-QA",
            name: "QA팀",
            parentId: "2",
            managerId: "emp7",
            managerName: "정품질",
            employeeCount: 4,
            level: 2,
          },
        ],
      },
      {
        id: "3",
        code: "SALES",
        name: "영업본부",
        parentId: "1",
        managerId: "emp3",
        managerName: "홍영업",
        employeeCount: 12,
        level: 1,
        children: [
          {
            id: "8",
            code: "SALES-1",
            name: "영업1팀",
            parentId: "3",
            managerId: "emp8",
            managerName: "강영업",
            employeeCount: 6,
            level: 2,
          },
          {
            id: "9",
            code: "SALES-2",
            name: "영업2팀",
            parentId: "3",
            managerId: "emp9",
            managerName: "윤영업",
            employeeCount: 6,
            level: 2,
          },
        ],
      },
      {
        id: "4",
        code: "ADMIN",
        name: "경영지원본부",
        parentId: "1",
        managerId: "emp4",
        managerName: "서경영",
        employeeCount: 8,
        level: 1,
        children: [
          {
            id: "10",
            code: "HR",
            name: "인사팀",
            parentId: "4",
            managerId: "emp10",
            managerName: "조인사",
            employeeCount: 4,
            level: 2,
          },
          {
            id: "11",
            code: "FIN",
            name: "재무팀",
            parentId: "4",
            managerId: "emp11",
            managerName: "한재무",
            employeeCount: 4,
            level: 2,
          },
        ],
      },
    ],
  },
];

// Flatten departments for select options
function flattenDepartments(
  departments: Department[],
  result: Department[] = []
): Department[] {
  for (const dept of departments) {
    result.push(dept);
    if (dept.children) {
      flattenDepartments(dept.children, result);
    }
  }
  return result;
}

// Validation schema
const departmentSchema = z.object({
  code: z.string().min(1, "부서 코드를 입력하세요"),
  name: z.string().min(1, "부서명을 입력하세요"),
  parentId: z.string().optional(),
  managerId: z.string().optional(),
});

type DepartmentFormData = z.infer<typeof departmentSchema>;

// Department Tree Item Component
interface DepartmentTreeItemProps {
  department: Department;
  expandedIds: Set<string>;
  onToggle: (id: string) => void;
  onEdit: (dept: Department) => void;
  onDelete: (dept: Department) => void;
  onAddChild: (dept: Department) => void;
  selectedId?: string;
  onSelect: (dept: Department) => void;
}

function DepartmentTreeItem({
  department,
  expandedIds,
  onToggle,
  onEdit,
  onDelete,
  onAddChild,
  selectedId,
  onSelect,
}: DepartmentTreeItemProps) {
  const hasChildren = department.children && department.children.length > 0;
  const isExpanded = expandedIds.has(department.id);
  const isSelected = selectedId === department.id;

  return (
    <div>
      <div
        className={cn(
          "flex items-center justify-between p-3 rounded-lg cursor-pointer transition-colors group",
          isSelected ? "bg-primary/10 border border-primary/30" : "hover:bg-muted"
        )}
        style={{ paddingLeft: `${department.level * 24 + 12}px` }}
        onClick={() => onSelect(department)}
      >
        <div className="flex items-center flex-1 min-w-0">
          <button
            className="p-1 mr-1 rounded hover:bg-muted-foreground/10"
            onClick={(e) => {
              e.stopPropagation();
              if (hasChildren) onToggle(department.id);
            }}
          >
            {hasChildren ? (
              isExpanded ? (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              )
            ) : (
              <div className="w-4 h-4" />
            )}
          </button>
          <Building2 className="h-4 w-4 mr-2 text-primary" />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium truncate">{department.name}</span>
              <Badge variant="secondary" className="text-xs">
                {department.code}
              </Badge>
            </div>
            {department.managerName && (
              <p className="text-xs text-muted-foreground">
                {department.managerName} (팀장)
              </p>
            )}
          </div>
          <div className="flex items-center gap-2 mr-4">
            <Users className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">
              {department.employeeCount}명
            </span>
          </div>
        </div>
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={(e) => {
              e.stopPropagation();
              onAddChild(department);
            }}
            title="하위 부서 추가"
          >
            <Plus className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={(e) => {
              e.stopPropagation();
              onEdit(department);
            }}
            title="수정"
          >
            <Edit className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-destructive hover:text-destructive"
            onClick={(e) => {
              e.stopPropagation();
              onDelete(department);
            }}
            title="삭제"
            disabled={hasChildren}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>
      {hasChildren && isExpanded && (
        <div>
          {department.children!.map((child) => (
            <DepartmentTreeItem
              key={child.id}
              department={child}
              expandedIds={expandedIds}
              onToggle={onToggle}
              onEdit={onEdit}
              onDelete={onDelete}
              onAddChild={onAddChild}
              selectedId={selectedId}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// Mock employees for manager selection
const mockEmployees = [
  { id: "emp1", name: "김대표" },
  { id: "emp2", name: "이개발" },
  { id: "emp3", name: "홍영업" },
  { id: "emp4", name: "서경영" },
  { id: "emp5", name: "박프론트" },
  { id: "emp6", name: "최백엔드" },
  { id: "emp7", name: "정품질" },
  { id: "emp8", name: "강영업" },
  { id: "emp9", name: "윤영업" },
  { id: "emp10", name: "조인사" },
  { id: "emp11", name: "한재무" },
];

export function DepartmentPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [expandedIds, setExpandedIds] = useState<Set<string>>(
    new Set(["1", "2", "3", "4"])
  );
  const [selectedDepartment, setSelectedDepartment] = useState<Department | null>(
    null
  );
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalMode, setModalMode] = useState<"create" | "edit">("create");
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_parentForNewDept, setParentForNewDept] = useState<Department | null>(null);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [departmentToDelete, setDepartmentToDelete] = useState<Department | null>(
    null
  );

  const {
    register,
    control,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<DepartmentFormData>({
    resolver: zodResolver(departmentSchema),
    defaultValues: {
      code: "",
      name: "",
      parentId: "",
      managerId: "",
    },
  });

  // Get all departments flattened
  const allDepartments = flattenDepartments(mockDepartments);

  // Calculate stats
  const totalDepartments = allDepartments.length;
  const totalEmployees = allDepartments.reduce(
    (sum, dept) => sum + dept.employeeCount,
    0
  );

  // Toggle expand/collapse
  const handleToggle = (id: string) => {
    setExpandedIds((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
  };

  // Expand all
  const expandAll = () => {
    setExpandedIds(new Set(allDepartments.map((d) => d.id)));
  };

  // Collapse all
  const collapseAll = () => {
    setExpandedIds(new Set());
  };

  // Open modal for creating new department
  const handleOpenCreateModal = (parentDept?: Department) => {
    setModalMode("create");
    setParentForNewDept(parentDept || null);
    reset({
      code: "",
      name: "",
      parentId: parentDept?.id || "",
      managerId: "",
    });
    setIsModalOpen(true);
  };

  // Open modal for editing department
  const handleOpenEditModal = (dept: Department) => {
    setModalMode("edit");
    setSelectedDepartment(dept);
    reset({
      code: dept.code,
      name: dept.name,
      parentId: dept.parentId || "",
      managerId: dept.managerId || "",
    });
    setIsModalOpen(true);
  };

  // Submit handler
  const onSubmit = (data: DepartmentFormData) => {
    // TODO: API call
    console.log("Department data:", data);
    if (modalMode === "create") {
      toast.success("부서 등록 완료", "새 부서가 등록되었습니다.");
    } else {
      toast.success("부서 수정 완료", "부서 정보가 수정되었습니다.");
    }
    setIsModalOpen(false);
    reset();
  };

  // Delete confirmation
  const handleDeleteConfirm = () => {
    if (departmentToDelete) {
      // TODO: API call
      console.log("Delete department:", departmentToDelete.id);
      toast.success("부서 삭제 완료", "부서가 삭제되었습니다.");
      setIsDeleteModalOpen(false);
      setDepartmentToDelete(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">부서 관리</h1>
          <p className="text-muted-foreground">조직 구조를 관리합니다.</p>
        </div>
        <Button onClick={() => handleOpenCreateModal()}>
          <Plus className="h-4 w-4 mr-2" />
          부서 등록
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">전체 부서</p>
                <p className="text-2xl font-bold">{totalDepartments}개</p>
              </div>
              <Building2 className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">전체 인원</p>
                <p className="text-2xl font-bold">{totalEmployees}명</p>
              </div>
              <Users className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">조직 레벨</p>
                <p className="text-2xl font-bold">
                  {Math.max(...allDepartments.map((d) => d.level)) + 1}단계
                </p>
              </div>
              <FolderTree className="h-8 w-8 text-primary/20" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters & Actions */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="부서명, 부서코드로 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={expandAll}>
                전체 펼치기
              </Button>
              <Button variant="outline" onClick={collapseAll}>
                전체 접기
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Department Tree */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center">
            <FolderTree className="h-5 w-5 mr-2" />
            조직도
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-1">
            {mockDepartments.map((dept) => (
              <DepartmentTreeItem
                key={dept.id}
                department={dept}
                expandedIds={expandedIds}
                onToggle={handleToggle}
                onEdit={handleOpenEditModal}
                onDelete={(d) => {
                  setDepartmentToDelete(d);
                  setIsDeleteModalOpen(true);
                }}
                onAddChild={handleOpenCreateModal}
                selectedId={selectedDepartment?.id}
                onSelect={setSelectedDepartment}
              />
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Selected Department Detail */}
      {selectedDepartment && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center justify-between">
              <span>{selectedDepartment.name} 상세 정보</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleOpenEditModal(selectedDepartment)}
              >
                <Edit className="h-4 w-4 mr-2" />
                수정
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">부서 코드</p>
                <p className="font-medium">{selectedDepartment.code}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">부서명</p>
                <p className="font-medium">{selectedDepartment.name}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">팀장</p>
                <p className="font-medium">
                  {selectedDepartment.managerName || "-"}
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">소속 인원</p>
                <p className="font-medium">{selectedDepartment.employeeCount}명</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Create/Edit Modal */}
      <Modal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          reset();
        }}
        title={modalMode === "create" ? "부서 등록" : "부서 수정"}
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Input
            label="부서 코드"
            placeholder="DEV-01"
            required
            error={errors.code?.message}
            {...register("code")}
          />
          <Input
            label="부서명"
            placeholder="개발팀"
            required
            error={errors.name?.message}
            {...register("name")}
          />
          <Controller
            name="parentId"
            control={control}
            render={({ field }) => (
              <div>
                <label className="block text-sm font-medium mb-1">상위 부서</label>
                <select
                  className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                  {...field}
                >
                  <option value="">최상위 부서</option>
                  {allDepartments.map((dept) => (
                    <option key={dept.id} value={dept.id}>
                      {"  ".repeat(dept.level)}
                      {dept.name}
                    </option>
                  ))}
                </select>
              </div>
            )}
          />
          <Controller
            name="managerId"
            control={control}
            render={({ field }) => (
              <div>
                <label className="block text-sm font-medium mb-1">부서장</label>
                <select
                  className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                  {...field}
                >
                  <option value="">선택</option>
                  {mockEmployees.map((emp) => (
                    <option key={emp.id} value={emp.id}>
                      {emp.name}
                    </option>
                  ))}
                </select>
              </div>
            )}
          />
          <div className="flex justify-end gap-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setIsModalOpen(false);
                reset();
              }}
            >
              취소
            </Button>
            <Button type="submit">
              {modalMode === "create" ? "등록" : "수정"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={isDeleteModalOpen}
        onClose={() => {
          setIsDeleteModalOpen(false);
          setDepartmentToDelete(null);
        }}
        title="부서 삭제"
      >
        <div className="space-y-4">
          <p>
            <span className="font-medium">{departmentToDelete?.name}</span> 부서를
            삭제하시겠습니까?
          </p>
          <p className="text-sm text-muted-foreground">
            이 작업은 되돌릴 수 없습니다. 하위 부서가 있는 경우 삭제할 수 없습니다.
          </p>
          <div className="flex justify-end gap-2 pt-4">
            <Button
              variant="outline"
              onClick={() => {
                setIsDeleteModalOpen(false);
                setDepartmentToDelete(null);
              }}
            >
              취소
            </Button>
            <Button variant="destructive" onClick={handleDeleteConfirm}>
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
