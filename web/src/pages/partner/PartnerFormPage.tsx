import { useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Save, ArrowLeft, Building2, Phone } from "lucide-react";
import {
  Button,
  Input,
  Select,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
} from "@/components/ui";
import { toast } from "@/stores/ui";
import { partnersApi } from "@/api";
import { getErrorMessage } from "@/services/api";
import { PARTNER_TYPES } from "@/constants";
import type { PartnerType } from "@/types";

// Validation schema
const partnerSchema = z.object({
  // Basic info
  code: z.string().min(1, "거래처 코드를 입력하세요"),
  name: z.string().min(2, "거래처명은 최소 2자 이상이어야 합니다"),
  partnerType: z.string().min(1, "거래처 유형을 선택하세요"),

  // Business info
  businessNumber: z.string()
    .refine(
      (val) => !val || /^\d{10}$/.test(val.replace(/-/g, "")),
      "올바른 사업자등록번호를 입력하세요 (10자리)"
    )
    .optional()
    .or(z.literal("")),
  representativeName: z.string().optional(),

  // Contact info
  address: z.string().optional(),
  phone: z.string().optional(),
  fax: z.string().optional(),
  email: z.string()
    .refine(
      (val) => !val || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(val),
      "올바른 이메일 형식을 입력하세요"
    )
    .optional()
    .or(z.literal("")),

  isActive: z.boolean(),
});

type PartnerFormData = z.infer<typeof partnerSchema>;

export function PartnerFormPage() {
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
    formState: { errors },
  } = useForm<PartnerFormData>({
    resolver: zodResolver(partnerSchema),
    defaultValues: {
      code: "",
      name: "",
      partnerType: "",
      businessNumber: "",
      representativeName: "",
      address: "",
      phone: "",
      fax: "",
      email: "",
      isActive: true,
    },
  });

  const isActive = watch("isActive");
  const partnerType = watch("partnerType");

  // GET /api/v1/partners/:id
  const {
    data: partnerResponse,
    isLoading: isLoadingPartner,
    isError: isPartnerError,
    error: partnerError,
  } = useQuery({
    queryKey: ["partners", "detail", id],
    queryFn: () => partnersApi.get(id as string),
    enabled: isEditMode,
  });

  // Fill the form once the server answers. Nothing is shown until it does, so
  // the user never edits placeholder values believing they are real.
  useEffect(() => {
    const partner = partnerResponse?.data;
    if (!partner) return;
    reset({
      code: partner.code,
      name: partner.name,
      partnerType: partner.partnerType,
      businessNumber: partner.businessNumber ?? "",
      representativeName: partner.representativeName ?? "",
      address: partner.address ?? "",
      phone: partner.phone ?? "",
      fax: partner.fax ?? "",
      email: partner.email ?? "",
      isActive: partner.isActive,
    });
  }, [partnerResponse, reset]);

  const saveMutation = useMutation({
    mutationFn: (data: PartnerFormData) => {
      const payload = {
        code: data.code,
        name: data.name,
        partnerType: data.partnerType as PartnerType,
        businessNumber: data.businessNumber || undefined,
        representativeName: data.representativeName || undefined,
        address: data.address || undefined,
        phone: data.phone || undefined,
        fax: data.fax || undefined,
        email: data.email || undefined,
        isActive: data.isActive,
      };
      return isEditMode
        ? partnersApi.update(id as string, payload)
        : partnersApi.create(payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["partners"] });
      toast.success(
        isEditMode ? "거래처 수정 완료" : "거래처 등록 완료",
        isEditMode
          ? "거래처 정보가 수정되었습니다."
          : "새 거래처가 등록되었습니다."
      );
      navigate("/partners");
    },
    onError: (err: unknown) => {
      toast.error(
        "저장 실패",
        getErrorMessage(err, "거래처 정보 저장 중 오류가 발생했습니다.")
      );
    },
  });

  const onSubmit = (data: PartnerFormData) => saveMutation.mutate(data);
  const isSubmitting = saveMutation.isPending;

  // Format business number as user types
  const formatBusinessNumber = (value: string) => {
    const cleaned = value.replace(/\D/g, "").slice(0, 10);
    if (cleaned.length <= 3) return cleaned;
    if (cleaned.length <= 5) return `${cleaned.slice(0, 3)}-${cleaned.slice(3)}`;
    return `${cleaned.slice(0, 3)}-${cleaned.slice(3, 5)}-${cleaned.slice(5)}`;
  };

  if (isEditMode && isLoadingPartner) {
    return (
      <div className="py-16 text-center text-muted-foreground">
        거래처 정보를 불러오는 중...
      </div>
    );
  }

  if (isEditMode && isPartnerError) {
    return (
      <div className="space-y-4 py-16 text-center">
        <p className="text-destructive">
          {getErrorMessage(partnerError, "거래처 정보를 불러오지 못했습니다.")}
        </p>
        <Button variant="outline" onClick={() => navigate("/partners")}>
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
              {isEditMode ? "거래처 정보 수정" : "거래처 등록"}
            </h1>
            <p className="text-muted-foreground">
              {isEditMode
                ? "거래처 정보를 수정합니다."
                : "새로운 거래처를 등록합니다."}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
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
              <Building2 className="h-5 w-5 mr-2" />
              기본 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Input
                label="거래처 코드"
                required
                placeholder="P001"
                error={errors.code?.message}
                {...register("code")}
              />
              <Input
                label="거래처명"
                required
                placeholder="(주)회사명"
                error={errors.name?.message}
                {...register("name")}
              />
              <Controller
                name="partnerType"
                control={control}
                render={({ field }) => (
                  <Select
                    label="거래처 유형"
                    options={PARTNER_TYPES}
                    required
                    error={errors.partnerType?.message}
                    {...field}
                  />
                )}
              />
              {partnerType && (
                <div className="md:col-span-3">
                  <p className="text-sm text-muted-foreground">
                    {partnerType === "customer" && "매출 거래에 사용되는 고객사입니다."}
                    {partnerType === "supplier" && "매입 거래에 사용되는 공급업체입니다."}
                    {partnerType === "both" && "매출과 매입 거래 모두에 사용됩니다."}
                  </p>
                </div>
              )}
              <Controller
                name="businessNumber"
                control={control}
                render={({ field: { onChange, value, ...field } }) => (
                  <Input
                    label="사업자등록번호"
                    placeholder="123-45-67890"
                    error={errors.businessNumber?.message}
                    value={formatBusinessNumber(value || "")}
                    onChange={(e) => onChange(e.target.value.replace(/-/g, ""))}
                    {...field}
                  />
                )}
              />
              <Input
                label="대표자명"
                placeholder="홍길동"
                {...register("representativeName")}
              />
              <div className="flex items-center space-x-2 pt-6">
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
          </CardContent>
        </Card>

        {/* Contact Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <Phone className="h-5 w-5 mr-2" />
              연락처 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="md:col-span-3">
                <Input
                  label="주소"
                  placeholder="서울시 강남구 테헤란로 123"
                  {...register("address")}
                />
              </div>
              <Input
                label="전화번호"
                placeholder="02-1234-5678"
                {...register("phone")}
              />
              <Input
                label="팩스"
                placeholder="02-1234-5679"
                {...register("fax")}
              />
              <Input
                type="email"
                label="이메일"
                placeholder="contact@company.com"
                error={errors.email?.message}
                {...register("email")}
              />
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
