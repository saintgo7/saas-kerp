import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  ArrowLeft,
  Save,
  Package,
  AlertCircle,
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
} from "@/components/ui";
import { toast } from "@/stores/ui";
import { PRODUCT_UNITS } from "@/constants";
import type { ProductCategory } from "@/types/inventory";

// Mock categories data
const mockCategories: ProductCategory[] = [
  { id: "1", companyId: "1", code: "CAT001", name: "전자제품", level: 1, isActive: true, createdAt: "", updatedAt: "" },
  { id: "2", companyId: "1", code: "CAT002", name: "사무용품", level: 1, isActive: true, createdAt: "", updatedAt: "" },
  { id: "3", companyId: "1", code: "CAT003", name: "소모품", level: 1, isActive: true, createdAt: "", updatedAt: "" },
  { id: "4", companyId: "1", code: "CAT004", name: "원재료", level: 1, isActive: true, createdAt: "", updatedAt: "" },
];

// Convert categories to select options
const categoryOptions = mockCategories.map((cat) => ({
  value: cat.id,
  label: cat.name,
}));

// Validation schema for product form
const productSchema = z.object({
  code: z
    .string()
    .min(1, "품목코드를 입력하세요")
    .max(20, "품목코드는 20자 이하로 입력하세요")
    .regex(/^[A-Za-z0-9-_]+$/, "영문, 숫자, -, _ 만 사용 가능합니다"),
  name: z
    .string()
    .min(1, "품목명을 입력하세요")
    .max(100, "품목명은 100자 이하로 입력하세요"),
  categoryId: z.string().optional(),
  specification: z.string().max(200, "규격은 200자 이하로 입력하세요").optional(),
  unit: z.string().min(1, "단위를 선택하세요"),
  unitPrice: z.number().min(0, "0 이상이어야 합니다"),
  costPrice: z.number().min(0, "0 이상이어야 합니다"),
  minStock: z.number().min(0, "0 이상이어야 합니다").optional(),
  maxStock: z.number().min(0, "0 이상이어야 합니다").optional(),
  barcode: z.string().max(50, "바코드는 50자 이하로 입력하세요").optional(),
  description: z.string().max(500, "설명은 500자 이하로 입력하세요").optional(),
  isActive: z.boolean(),
});

type ProductFormData = z.infer<typeof productSchema>;

/**
 * Product Form Page Component
 * Handles creating and editing products
 */
export function ProductFormPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isEditMode = Boolean(id);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const {
    register,
    control,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<ProductFormData>({
    resolver: zodResolver(productSchema),
    defaultValues: {
      code: "",
      name: "",
      categoryId: "",
      specification: "",
      unit: "EA",
      unitPrice: 0,
      costPrice: 0,
      minStock: 0,
      maxStock: 0,
      barcode: "",
      description: "",
      isActive: true,
    },
  });

  // Watch prices to calculate margin
  const unitPrice = watch("unitPrice") || 0;
  const costPrice = watch("costPrice") || 0;
  const margin = unitPrice > 0 ? ((unitPrice - costPrice) / unitPrice) * 100 : 0;

  // Load product data for edit mode
  useEffect(() => {
    if (isEditMode && id) {
      setIsLoading(true);
      // TODO: Replace with actual API call
      setTimeout(() => {
        // Mock data for edit
        reset({
          code: "PRD-001",
          name: "노트북 컴퓨터 15인치",
          categoryId: "1",
          specification: "Intel i7, 16GB RAM, 512GB SSD",
          unit: "EA",
          unitPrice: 1500000,
          costPrice: 1200000,
          minStock: 5,
          maxStock: 50,
          barcode: "8801234567890",
          description: "고성능 업무용 노트북",
          isActive: true,
        });
        setIsLoading(false);
      }, 500);
    }
  }, [isEditMode, id, reset]);

  // Handle form submission
  const onSubmit = async (data: ProductFormData) => {
    setIsSubmitting(true);
    try {
      // TODO: Replace with actual API call
      console.log("Product data:", data);
      await new Promise((resolve) => setTimeout(resolve, 1000));

      toast.success(
        isEditMode ? "품목 수정 완료" : "품목 등록 완료",
        isEditMode
          ? "품목 정보가 성공적으로 수정되었습니다."
          : "새 품목이 성공적으로 등록되었습니다."
      );
      navigate("/inventory/products");
    } catch {
      toast.error(
        isEditMode ? "수정 실패" : "등록 실패",
        "품목 저장 중 오류가 발생했습니다."
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  // Generate auto product code
  const generateCode = () => {
    const timestamp = Date.now().toString(36).toUpperCase();
    const code = `PRD-${timestamp}`;
    reset((prev) => ({ ...prev, code }));
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">데이터를 불러오는 중...</p>
        </div>
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
              {isEditMode ? "품목 수정" : "품목 등록"}
            </h1>
            <p className="text-muted-foreground">
              {isEditMode
                ? "품목 정보를 수정합니다."
                : "새로운 품목을 등록합니다."}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button onClick={handleSubmit(onSubmit)} disabled={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            {isSubmitting ? "저장 중..." : "저장"}
          </Button>
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <Package className="h-5 w-5 mr-2" />
              기본 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">
                  품목코드 <span className="text-destructive">*</span>
                </label>
                <div className="flex space-x-2">
                  <Input
                    placeholder="PRD-001"
                    className="flex-1"
                    error={errors.code?.message}
                    {...register("code")}
                  />
                  {!isEditMode && (
                    <Button type="button" variant="outline" onClick={generateCode}>
                      자동생성
                    </Button>
                  )}
                </div>
                {errors.code && (
                  <p className="text-sm text-destructive">{errors.code.message}</p>
                )}
              </div>

              <Input
                label="품목명"
                placeholder="품목명을 입력하세요"
                required
                error={errors.name?.message}
                {...register("name")}
              />

              <Controller
                name="categoryId"
                control={control}
                render={({ field }) => (
                  <Select
                    label="카테고리"
                    options={categoryOptions}
                    placeholder="카테고리 선택"
                    error={errors.categoryId?.message}
                    {...field}
                  />
                )}
              />

              <Input
                label="규격"
                placeholder="규격/사양을 입력하세요"
                error={errors.specification?.message}
                {...register("specification")}
              />

              <Controller
                name="unit"
                control={control}
                render={({ field }) => (
                  <Select
                    label="단위"
                    options={PRODUCT_UNITS}
                    placeholder="단위 선택"
                    required
                    error={errors.unit?.message}
                    {...field}
                  />
                )}
              />

              <Input
                label="바코드"
                placeholder="바코드를 입력하세요"
                error={errors.barcode?.message}
                {...register("barcode")}
              />
            </div>
          </CardContent>
        </Card>

        {/* Price Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">가격 정보</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Input
                type="number"
                label="판매단가"
                placeholder="0"
                required
                error={errors.unitPrice?.message}
                {...register("unitPrice", { valueAsNumber: true })}
              />

              <Input
                type="number"
                label="매입단가"
                placeholder="0"
                required
                error={errors.costPrice?.message}
                {...register("costPrice", { valueAsNumber: true })}
              />

              <div className="space-y-1.5">
                <label className="text-sm font-medium">마진율</label>
                <div className="flex h-10 items-center px-3 rounded-md border border-input bg-muted">
                  <span
                    className={`font-mono ${
                      margin > 0 ? "text-green-600" : margin < 0 ? "text-red-600" : ""
                    }`}
                  >
                    {margin.toFixed(1)}%
                  </span>
                </div>
              </div>
            </div>

            {margin < 0 && (
              <div className="flex items-center space-x-2 mt-4 p-3 bg-destructive/10 text-destructive rounded-lg">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">
                  매입단가가 판매단가보다 높습니다. 마진율을 확인해주세요.
                </span>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Stock Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">재고 관리</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                type="number"
                label="안전재고"
                placeholder="0"
                error={errors.minStock?.message}
                {...register("minStock", { valueAsNumber: true })}
              />

              <Input
                type="number"
                label="최대재고"
                placeholder="0"
                error={errors.maxStock?.message}
                {...register("maxStock", { valueAsNumber: true })}
              />
            </div>
            <p className="text-sm text-muted-foreground mt-2">
              안전재고 이하로 떨어지면 재고 알림이 발생합니다.
            </p>
          </CardContent>
        </Card>

        {/* Additional Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">추가 정보</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <Textarea
                label="설명"
                placeholder="품목에 대한 추가 설명을 입력하세요"
                rows={3}
                error={errors.description?.message}
                {...register("description")}
              />

              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="isActive"
                  className="rounded border-input"
                  {...register("isActive")}
                />
                <label htmlFor="isActive" className="text-sm font-medium">
                  사용함
                </label>
                <span className="text-sm text-muted-foreground">
                  (체크 해제 시 품목 목록에서 숨김 처리됩니다)
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Submit Button (Mobile) */}
        <div className="flex justify-end space-x-2 lg:hidden">
          <Button variant="outline" type="button" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            {isSubmitting ? "저장 중..." : "저장"}
          </Button>
        </div>
      </form>
    </div>
  );
}
