import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Save,
  ArrowLeft,
  Building2,
  Phone,
  CreditCard,
  FileText,
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
import { PARTNER_TYPES, BANKS } from "@/constants";

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
  businessType: z.string().optional(),
  businessCategory: z.string().optional(),

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

  // Bank info
  bankName: z.string().optional(),
  bankAccount: z.string().optional(),
  accountHolder: z.string().optional(),

  // Others
  note: z.string().optional(),
  isActive: z.boolean(),
});

type PartnerFormData = z.infer<typeof partnerSchema>;

// Mock partner data for edit mode
const mockPartner: PartnerFormData = {
  code: "P001",
  name: "(주)테크솔루션",
  partnerType: "both",
  businessNumber: "1234567890",
  representativeName: "김대표",
  businessType: "서비스",
  businessCategory: "소프트웨어 개발",
  address: "서울시 강남구 테헤란로 123",
  phone: "0212345678",
  fax: "0212345679",
  email: "contact@techsolution.co.kr",
  bankName: "KB",
  bankAccount: "123-456-789012",
  accountHolder: "김대표",
  note: "주요 고객사",
  isActive: true,
};

export function PartnerFormPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isEditMode = !!id;
  const [isSubmitting, setIsSubmitting] = useState(false);

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
      businessType: "",
      businessCategory: "",
      address: "",
      phone: "",
      fax: "",
      email: "",
      bankName: "",
      bankAccount: "",
      accountHolder: "",
      note: "",
      isActive: true,
    },
  });

  const isActive = watch("isActive");
  const partnerType = watch("partnerType");

  // Load partner data in edit mode
  useEffect(() => {
    if (isEditMode) {
      // TODO: API call to fetch partner data
      reset(mockPartner);
    }
  }, [isEditMode, reset]);

  const onSubmit = async (data: PartnerFormData) => {
    setIsSubmitting(true);
    try {
      // TODO: API call
      console.log("Partner data:", data);
      toast.success(
        isEditMode ? "거래처 수정 완료" : "거래처 등록 완료",
        isEditMode
          ? "거래처 정보가 수정되었습니다."
          : "새 거래처가 등록되었습니다."
      );
      navigate("/partners");
    } catch {
      toast.error("저장 실패", "거래처 정보 저장 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  // Format business number as user types
  const formatBusinessNumber = (value: string) => {
    const cleaned = value.replace(/\D/g, "").slice(0, 10);
    if (cleaned.length <= 3) return cleaned;
    if (cleaned.length <= 5) return `${cleaned.slice(0, 3)}-${cleaned.slice(3)}`;
    return `${cleaned.slice(0, 3)}-${cleaned.slice(3, 5)}-${cleaned.slice(5)}`;
  };

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
              <Input
                label="업태"
                placeholder="서비스, 제조업 등"
                {...register("businessType")}
              />
              <Input
                label="업종"
                placeholder="소프트웨어 개발 등"
                {...register("businessCategory")}
              />
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

        {/* Bank Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <CreditCard className="h-5 w-5 mr-2" />
              계좌 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Controller
                name="bankName"
                control={control}
                render={({ field }) => (
                  <Select
                    label="은행"
                    options={[{ value: "", label: "선택하세요" }, ...BANKS]}
                    {...field}
                  />
                )}
              />
              <Input
                label="계좌번호"
                placeholder="123-456-789012"
                {...register("bankAccount")}
              />
              <Input
                label="예금주"
                placeholder="홍길동"
                {...register("accountHolder")}
              />
            </div>
            <p className="text-sm text-muted-foreground mt-4">
              * 계좌 정보는 대금 지급 시 사용됩니다.
            </p>
          </CardContent>
        </Card>

        {/* Notes */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <FileText className="h-5 w-5 mr-2" />
              비고
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Textarea
              placeholder="거래처에 대한 메모를 입력하세요"
              rows={4}
              {...register("note")}
            />
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
