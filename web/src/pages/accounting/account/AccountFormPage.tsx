import { useEffect, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Save,
  ArrowLeft,
  BookOpen,
  Info,
} from "lucide-react";
import {
  Button,
  Input,
  Textarea,
  Select,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
} from "@/components/ui";
import { toast } from "@/stores/ui";
import { ACCOUNT_TYPES } from "@/constants";
import { accountsApi } from "@/api";
import type { AccountTreeNode } from "@/api/accounts";
import { getErrorMessage } from "@/services/api";
import type { AccountType } from "@/types";

/** Shape used by the parent-account picker. */
interface ParentAccount {
  id: string;
  code: string;
  name: string;
  type: AccountType;
  level: number;
}

// Validation schema
const accountSchema = z.object({
  code: z.string().min(1, "계정과목 코드를 입력하세요").max(20, "코드는 20자 이내로 입력하세요"),
  name: z.string().min(1, "계정과목명을 입력하세요").max(100, "계정과목명은 100자 이내로 입력하세요"),
  type: z.string().min(1, "계정 유형을 선택하세요"),
  parentId: z.string().optional(),
  description: z.string().optional(),
  isActive: z.boolean(),
});

type AccountFormData = z.infer<typeof accountSchema>;

const accountTypeStyles: Record<AccountType, { variant: "default" | "secondary" | "success" | "warning" | "destructive"; label: string }> = {
  asset: { variant: "success", label: "자산" },
  liability: { variant: "warning", label: "부채" },
  equity: { variant: "secondary", label: "자본" },
  revenue: { variant: "default", label: "수익" },
  expense: { variant: "destructive", label: "비용" },
};

export function AccountFormPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id } = useParams<{ id: string }>();
  const isEditMode = !!id;

  const {
    register,
    control,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = useForm<AccountFormData>({
    resolver: zodResolver(accountSchema),
    defaultValues: {
      code: "",
      name: "",
      type: "",
      parentId: "",
      description: "",
      isActive: true,
    },
  });

  const selectedType = watch("type");
  const selectedParentId = watch("parentId");
  const isActive = watch("isActive");

  // GET /api/v1/accounts/tree — real parent candidates, not a hardcoded list.
  const { data: treeResponse } = useQuery({
    queryKey: ["accounts", "tree"],
    queryFn: () => accountsApi.tree(),
  });

  const allParents = useMemo<ParentAccount[]>(() => {
    const flat: ParentAccount[] = [];
    const walk = (nodes: AccountTreeNode[]) => {
      nodes.forEach((node) => {
        flat.push({
          id: node.id,
          code: node.code,
          name: node.name,
          type: node.type,
          level: node.level,
        });
        if (node.children?.length) walk(node.children);
      });
    };
    if (treeResponse?.data) walk(treeResponse.data);
    // An account cannot be its own parent.
    return isEditMode ? flat.filter((p) => p.id !== id) : flat;
  }, [treeResponse, isEditMode, id]);

  const filteredParents = useMemo(
    () =>
      selectedType
        ? allParents.filter((p) => p.type === selectedType)
        : allParents,
    [allParents, selectedType]
  );

  // Clear parent selection when the chosen type no longer matches it
  useEffect(() => {
    if (selectedType && selectedParentId) {
      const parent = allParents.find((p) => p.id === selectedParentId);
      if (parent && parent.type !== selectedType) {
        setValue("parentId", "");
      }
    }
  }, [selectedType, selectedParentId, setValue, allParents]);

  // GET /api/v1/accounts/:id
  const {
    data: accountResponse,
    isLoading: isLoadingAccount,
    isError: isAccountError,
    error: accountError,
  } = useQuery({
    queryKey: ["accounts", "detail", id],
    queryFn: () => accountsApi.get(id as string),
    enabled: isEditMode,
  });

  useEffect(() => {
    const account = accountResponse?.data;
    if (!account) return;
    reset({
      code: account.code,
      name: account.name,
      type: account.type,
      parentId: account.parentId ?? "",
      description: account.accountCategory ?? "",
      isActive: account.isActive,
    });
  }, [accountResponse, reset]);

  const saveMutation = useMutation({
    mutationFn: (data: AccountFormData) => {
      const payload = {
        code: data.code,
        name: data.name,
        type: data.type as AccountType,
        parentId: data.parentId || undefined,
        // The backend has no free-text description; `account_category` is the
        // closest field it stores (max 50 chars).
        accountCategory: data.description?.slice(0, 50) || undefined,
        isActive: data.isActive,
      };
      return isEditMode
        ? accountsApi.update(id as string, { ...payload, isActive: data.isActive })
        : accountsApi.create(payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["accounts"] });
      toast.success(
        isEditMode ? "계정과목 수정 완료" : "계정과목 등록 완료",
        isEditMode
          ? "계정과목이 수정되었습니다."
          : "새 계정과목이 등록되었습니다."
      );
      navigate("/accounting/accounts");
    },
    onError: (err: unknown) => {
      toast.error(
        "저장 실패",
        getErrorMessage(err, "계정과목 저장 중 오류가 발생했습니다.")
      );
    },
  });

  const onSubmit = (data: AccountFormData) => saveMutation.mutate(data);
  const isSubmitting = saveMutation.isPending;

  // Get parent account name
  const getParentName = () => {
    if (!selectedParentId) return null;
    const parent = allParents.find((p) => p.id === selectedParentId);
    return parent ? `${parent.code} - ${parent.name}` : null;
  };

  // Generate next code suggestion
  const suggestedCode = () => {
    if (!selectedParentId) return "";
    const parent = allParents.find((p) => p.id === selectedParentId);
    return parent ? `${parent.code}X` : "";
  };

  if (isEditMode && isLoadingAccount) {
    return (
      <div className="py-16 text-center text-muted-foreground">
        계정과목 정보를 불러오는 중...
      </div>
    );
  }

  if (isEditMode && isAccountError) {
    return (
      <div className="space-y-4 py-16 text-center">
        <p className="text-destructive">
          {getErrorMessage(accountError, "계정과목 정보를 불러오지 못했습니다.")}
        </p>
        <Button variant="outline" onClick={() => navigate("/accounting/accounts")}>
          목록으로
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold">
              {isEditMode ? "계정과목 수정" : "계정과목 등록"}
            </h1>
            <p className="text-muted-foreground">
              {isEditMode
                ? "계정과목 정보를 수정합니다."
                : "새로운 계정과목을 등록합니다."}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          {isEditMode && selectedType && (
            <Badge variant={accountTypeStyles[selectedType as AccountType]?.variant || "default"}>
              {accountTypeStyles[selectedType as AccountType]?.label || selectedType}
            </Badge>
          )}
          {isEditMode && (
            <Badge variant={isActive ? "success" : "secondary"}>
              {isActive ? "활성" : "비활성"}
            </Badge>
          )}
          <Button variant="outline" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button onClick={handleSubmit(onSubmit)} isLoading={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            저장
          </Button>
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <BookOpen className="h-5 w-5 mr-2" />
              계정과목 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Controller
                name="type"
                control={control}
                render={({ field }) => (
                  <Select
                    label="계정 유형"
                    options={ACCOUNT_TYPES}
                    required
                    error={errors.type?.message}
                    {...field}
                  />
                )}
              />
              <Controller
                name="parentId"
                control={control}
                render={({ field }) => (
                  <Select
                    label="상위 계정"
                    options={[
                      { value: "", label: "없음 (최상위 계정)" },
                      ...filteredParents.map((p) => ({
                        value: p.id,
                        label: `${"  ".repeat(p.level - 1)}${p.code} - ${p.name}`,
                      })),
                    ]}
                    error={errors.parentId?.message}
                    disabled={!selectedType}
                    {...field}
                  />
                )}
              />
              <Input
                label="계정과목 코드"
                required
                placeholder={suggestedCode() || "예: 111"}
                error={errors.code?.message}
                className="font-mono"
                {...register("code")}
              />
              <Input
                label="계정과목명"
                required
                placeholder="예: 현금및현금성자산"
                error={errors.name?.message}
                {...register("name")}
              />
              <div className="md:col-span-2">
                <Textarea
                  label="설명"
                  placeholder="계정과목에 대한 설명을 입력하세요"
                  rows={3}
                  {...register("description")}
                />
              </div>
              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="isActive"
                  className="rounded border-input"
                  {...register("isActive")}
                />
                <label htmlFor="isActive" className="text-sm font-medium">
                  활성 상태
                </label>
              </div>
            </div>

            {/* Parent Account Info */}
            {selectedParentId && (
              <div className="mt-4 p-4 bg-muted/50 rounded-lg">
                <p className="text-sm font-medium mb-1">상위 계정 정보</p>
                <p className="text-sm text-muted-foreground">{getParentName()}</p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Account Type Guide */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <Info className="h-5 w-5 mr-2" />
              계정 유형 안내
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
              <div className="p-4 rounded-lg bg-success/10 border border-success/20">
                <Badge variant="success" className="mb-2">자산</Badge>
                <p className="text-sm text-muted-foreground">
                  회사가 소유한 경제적 자원. 현금, 재고, 설비 등
                </p>
              </div>
              <div className="p-4 rounded-lg bg-warning/10 border border-warning/20">
                <Badge variant="warning" className="mb-2">부채</Badge>
                <p className="text-sm text-muted-foreground">
                  회사가 갚아야 할 의무. 차입금, 미지급금 등
                </p>
              </div>
              <div className="p-4 rounded-lg bg-secondary/10 border border-secondary/20">
                <Badge variant="secondary" className="mb-2">자본</Badge>
                <p className="text-sm text-muted-foreground">
                  자산에서 부채를 뺀 순자산. 자본금, 이익잉여금 등
                </p>
              </div>
              <div className="p-4 rounded-lg bg-primary/10 border border-primary/20">
                <Badge variant="default" className="mb-2">수익</Badge>
                <p className="text-sm text-muted-foreground">
                  영업활동으로 발생한 수입. 매출, 이자수익 등
                </p>
              </div>
              <div className="p-4 rounded-lg bg-destructive/10 border border-destructive/20">
                <Badge variant="destructive" className="mb-2">비용</Badge>
                <p className="text-sm text-muted-foreground">
                  영업활동에 사용된 지출. 급여, 임차료 등
                </p>
              </div>
            </div>
            <div className="mt-4 p-4 border rounded-lg">
              <p className="text-sm font-medium mb-2">복식부기 원리</p>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <p className="font-medium text-muted-foreground mb-1">차변 증가</p>
                  <p>자산 / 비용</p>
                </div>
                <div>
                  <p className="font-medium text-muted-foreground mb-1">대변 증가</p>
                  <p>부채 / 자본 / 수익</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Submit Buttons (Mobile) */}
        <div className="flex justify-end space-x-2 lg:hidden">
          <Button variant="outline" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button type="submit" isLoading={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            저장
          </Button>
        </div>
      </form>
    </div>
  );
}
