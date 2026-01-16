import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Save,
  ArrowLeft,
  User,
  Briefcase,
  CreditCard,
  Shield,
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

// Options for selects
const genderOptions = [
  { value: "M", label: "남성" },
  { value: "F", label: "여성" },
];

const employmentTypeOptions = [
  { value: "regular", label: "정규직" },
  { value: "contract", label: "계약직" },
  { value: "parttime", label: "파트타임" },
  { value: "intern", label: "인턴" },
];

const departmentOptions = [
  { value: "dev", label: "개발팀" },
  { value: "design", label: "디자인팀" },
  { value: "marketing", label: "마케팅팀" },
  { value: "sales", label: "영업팀" },
  { value: "hr", label: "인사팀" },
  { value: "finance", label: "재무팀" },
  { value: "admin", label: "경영지원팀" },
];

const positionOptions = [
  { value: "intern", label: "인턴" },
  { value: "staff", label: "사원" },
  { value: "senior", label: "대리" },
  { value: "manager", label: "과장" },
  { value: "deputy", label: "차장" },
  { value: "director", label: "부장" },
  { value: "executive", label: "임원" },
];

const statusOptions = [
  { value: "active", label: "재직" },
  { value: "leave", label: "휴직" },
  { value: "resigned", label: "퇴직" },
];

const bankOptions = [
  { value: "kb", label: "국민은행" },
  { value: "shinhan", label: "신한은행" },
  { value: "woori", label: "우리은행" },
  { value: "hana", label: "하나은행" },
  { value: "nh", label: "농협은행" },
  { value: "ibk", label: "기업은행" },
  { value: "kakao", label: "카카오뱅크" },
  { value: "toss", label: "토스뱅크" },
];

// Validation schema
const employeeSchema = z.object({
  // Basic info
  name: z.string().min(2, "이름은 최소 2자 이상이어야 합니다"),
  nameEn: z.string().optional(),
  email: z.string().email("올바른 이메일을 입력하세요"),
  phone: z.string().min(10, "연락처를 입력하세요"),
  birthDate: z.string().min(1, "생년월일을 입력하세요"),
  gender: z.string().min(1, "성별을 선택하세요"),
  address: z.string().optional(),

  // Employment info
  employeeNo: z.string().min(1, "사번을 입력하세요"),
  employmentType: z.string().min(1, "고용형태를 선택하세요"),
  department: z.string().min(1, "부서를 선택하세요"),
  position: z.string().min(1, "직급을 선택하세요"),
  hireDate: z.string().min(1, "입사일을 입력하세요"),
  resignDate: z.string().optional(),
  status: z.string().min(1, "상태를 선택하세요"),

  // Salary info
  baseSalary: z.number().min(0, "기본급을 입력하세요"),
  bankName: z.string().min(1, "은행을 선택하세요"),
  accountNumber: z.string().min(1, "계좌번호를 입력하세요"),
  accountHolder: z.string().min(1, "예금주를 입력하세요"),

  // Insurance info (4대보험)
  nationalPensionNo: z.string().optional(),
  healthInsuranceNo: z.string().optional(),
  employmentInsuranceNo: z.string().optional(),
  industrialAccidentNo: z.string().optional(),

  // Notes
  note: z.string().optional(),
});

type EmployeeFormData = z.infer<typeof employeeSchema>;

// Mock employee data for edit mode
const mockEmployee: EmployeeFormData = {
  name: "홍길동",
  nameEn: "Hong Gildong",
  email: "hong@company.com",
  phone: "010-1234-5678",
  birthDate: "1990-03-15",
  gender: "M",
  address: "서울시 강남구 테헤란로 123",
  employeeNo: "EMP001",
  employmentType: "regular",
  department: "dev",
  position: "senior",
  hireDate: "2020-03-02",
  resignDate: "",
  status: "active",
  baseSalary: 4500000,
  bankName: "kb",
  accountNumber: "123-456-789012",
  accountHolder: "홍길동",
  nationalPensionNo: "12345678",
  healthInsuranceNo: "H12345678",
  employmentInsuranceNo: "E12345678",
  industrialAccidentNo: "I12345678",
  note: "개발팀 리드",
};

export function EmployeeFormPage() {
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
  } = useForm<EmployeeFormData>({
    resolver: zodResolver(employeeSchema),
    defaultValues: {
      name: "",
      nameEn: "",
      email: "",
      phone: "",
      birthDate: "",
      gender: "",
      address: "",
      employeeNo: "",
      employmentType: "",
      department: "",
      position: "",
      hireDate: new Date().toISOString().split("T")[0],
      resignDate: "",
      status: "active",
      baseSalary: 0,
      bankName: "",
      accountNumber: "",
      accountHolder: "",
      nationalPensionNo: "",
      healthInsuranceNo: "",
      employmentInsuranceNo: "",
      industrialAccidentNo: "",
      note: "",
    },
  });

  const status = watch("status");

  // Load employee data in edit mode
  useEffect(() => {
    if (isEditMode) {
      // TODO: API call to fetch employee data
      reset(mockEmployee);
    }
  }, [isEditMode, reset]);

  const onSubmit = async (data: EmployeeFormData) => {
    setIsSubmitting(true);
    try {
      // TODO: API call
      console.log("Employee data:", data);
      toast.success(
        isEditMode ? "직원 수정 완료" : "직원 등록 완료",
        isEditMode
          ? "직원 정보가 수정되었습니다."
          : "새 직원이 등록되었습니다."
      );
      navigate("/hr/employee");
    } catch {
      toast.error("저장 실패", "직원 정보 저장 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const formatSalary = (value: number) => {
    return new Intl.NumberFormat("ko-KR").format(value);
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
              {isEditMode ? "직원 정보 수정" : "직원 등록"}
            </h1>
            <p className="text-muted-foreground">
              {isEditMode
                ? "직원 정보를 수정합니다."
                : "새로운 직원을 등록합니다."}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          {isEditMode && (
            <Badge
              variant={
                status === "active"
                  ? "success"
                  : status === "leave"
                  ? "warning"
                  : "secondary"
              }
            >
              {status === "active"
                ? "재직중"
                : status === "leave"
                ? "휴직중"
                : "퇴직"}
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
              <User className="h-5 w-5 mr-2" />
              기본 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Input
                label="이름"
                required
                error={errors.name?.message}
                {...register("name")}
              />
              <Input
                label="영문이름"
                placeholder="Hong Gildong"
                {...register("nameEn")}
              />
              <Input
                label="사번"
                required
                placeholder="EMP001"
                error={errors.employeeNo?.message}
                {...register("employeeNo")}
              />
              <Input
                type="email"
                label="이메일"
                required
                error={errors.email?.message}
                {...register("email")}
              />
              <Input
                label="연락처"
                required
                placeholder="010-1234-5678"
                error={errors.phone?.message}
                {...register("phone")}
              />
              <Controller
                name="gender"
                control={control}
                render={({ field }) => (
                  <Select
                    label="성별"
                    options={genderOptions}
                    required
                    error={errors.gender?.message}
                    {...field}
                  />
                )}
              />
              <Input
                type="date"
                label="생년월일"
                required
                error={errors.birthDate?.message}
                {...register("birthDate")}
              />
              <div className="md:col-span-2">
                <Input
                  label="주소"
                  placeholder="서울시 강남구..."
                  {...register("address")}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Employment Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <Briefcase className="h-5 w-5 mr-2" />
              고용 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Controller
                name="employmentType"
                control={control}
                render={({ field }) => (
                  <Select
                    label="고용형태"
                    options={employmentTypeOptions}
                    required
                    error={errors.employmentType?.message}
                    {...field}
                  />
                )}
              />
              <Controller
                name="department"
                control={control}
                render={({ field }) => (
                  <Select
                    label="부서"
                    options={departmentOptions}
                    required
                    error={errors.department?.message}
                    {...field}
                  />
                )}
              />
              <Controller
                name="position"
                control={control}
                render={({ field }) => (
                  <Select
                    label="직급"
                    options={positionOptions}
                    required
                    error={errors.position?.message}
                    {...field}
                  />
                )}
              />
              <Input
                type="date"
                label="입사일"
                required
                error={errors.hireDate?.message}
                {...register("hireDate")}
              />
              <Input
                type="date"
                label="퇴사일"
                {...register("resignDate")}
              />
              <Controller
                name="status"
                control={control}
                render={({ field }) => (
                  <Select
                    label="상태"
                    options={statusOptions}
                    required
                    error={errors.status?.message}
                    {...field}
                  />
                )}
              />
            </div>
          </CardContent>
        </Card>

        {/* Salary & Bank Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <CreditCard className="h-5 w-5 mr-2" />
              급여 및 계좌 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <Input
                  type="number"
                  label="기본급 (월)"
                  required
                  className="font-mono"
                  error={errors.baseSalary?.message}
                  {...register("baseSalary", { valueAsNumber: true })}
                />
                <p className="text-sm text-muted-foreground mt-1">
                  {formatSalary(watch("baseSalary") || 0)}원
                </p>
              </div>
              <Controller
                name="bankName"
                control={control}
                render={({ field }) => (
                  <Select
                    label="은행"
                    options={bankOptions}
                    required
                    error={errors.bankName?.message}
                    {...field}
                  />
                )}
              />
              <Input
                label="계좌번호"
                required
                placeholder="123-456-789012"
                error={errors.accountNumber?.message}
                {...register("accountNumber")}
              />
              <Input
                label="예금주"
                required
                error={errors.accountHolder?.message}
                {...register("accountHolder")}
              />
            </div>
          </CardContent>
        </Card>

        {/* Insurance Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <Shield className="h-5 w-5 mr-2" />
              4대보험 정보
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                label="국민연금 관리번호"
                placeholder="국민연금공단 관리번호"
                {...register("nationalPensionNo")}
              />
              <Input
                label="건강보험 자격번호"
                placeholder="건강보험공단 자격번호"
                {...register("healthInsuranceNo")}
              />
              <Input
                label="고용보험 피보험자번호"
                placeholder="고용보험 피보험자번호"
                {...register("employmentInsuranceNo")}
              />
              <Input
                label="산재보험 번호"
                placeholder="산재보험 번호"
                {...register("industrialAccidentNo")}
              />
            </div>
            <p className="text-sm text-muted-foreground mt-4">
              * 4대보험 정보는 EDI 연동 시 자동으로 업데이트됩니다.
            </p>
          </CardContent>
        </Card>

        {/* Notes */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">비고</CardTitle>
          </CardHeader>
          <CardContent>
            <Textarea
              placeholder="직원에 대한 메모를 입력하세요"
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
