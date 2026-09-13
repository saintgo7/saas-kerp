import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import {
  Building2,
  Save,
  Upload,
  Calendar,
  MapPin,
  FileText,
  Plus,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { FeatureUnavailable, notifyUnavailable } from "@/components/common";
import { apiClient } from "@/api";
import { getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";

// Server contract: internal/dto/company_dto.go
interface CompanySettings {
  fiscal_year_start: number;
  default_currency: string;
  decimal_places: number;
  tax_rate: number;
  voucher_auto_number: boolean;
  voucher_number_format: string;
  invoice_prefix: string;
  timezone: string;
  date_format: string;
  language: string;
}

interface Company {
  id: string;
  code: string;
  name: string;
  name_en?: string;
  business_number?: string;
  representative?: string;
  phone?: string;
  fax?: string;
  email?: string;
  website?: string;
  zip_code?: string;
  address?: string;
  address_detail?: string;
  status: string;
  settings: CompanySettings;
  trial_ends_at?: string;
  logo?: string;
  created_at: string;
  updated_at: string;
}

interface UpdateCompanyPayload {
  name: string;
  name_en: string;
  business_number: string;
  representative: string;
  phone: string;
  fax: string;
  email: string;
  website: string;
  zip_code: string;
  address: string;
  address_detail: string;
  logo: string;
}

// Validation schema for company settings
const companySchema = z.object({
  name: z.string().min(1, "상호명을 입력하세요"),
  businessNumber: z
    .string()
    .regex(/^\d{10}$/, "사업자등록번호는 10자리 숫자입니다"),
  representativeName: z.string().min(1, "대표자명을 입력하세요"),
  address: z.string().optional(),
  detailAddress: z.string().optional(),
  phone: z.string().optional(),
  fax: z.string().optional(),
  email: z.string().email("올바른 이메일 형식이 아닙니다").optional().or(z.literal("")),
  website: z.string().url("올바른 URL 형식이 아닙니다").optional().or(z.literal("")),
});

const fiscalYearSchema = z.object({
  fiscalYearStart: z.string().min(1, "회계연도 시작일을 선택하세요"),
});

type CompanyFormData = z.infer<typeof companySchema>;
type FiscalYearFormData = z.infer<typeof fiscalYearSchema>;

const toDateInput = (date: Date): string => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

/**
 * The server stores only the fiscal year *start month* (1-12), so the start
 * date input is normalised to the first day of that month in the current year.
 */
const fiscalStartToDateInput = (startMonth: number): string => {
  const month = startMonth >= 1 && startMonth <= 12 ? startMonth : 1;
  return `${new Date().getFullYear()}-${String(month).padStart(2, "0")}-01`;
};

/** The end date is derived, not stored: start + 1 year - 1 day. */
const deriveFiscalYearEnd = (start: string): string => {
  if (!start) return "";
  const parsed = new Date(`${start}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) return "";
  const end = new Date(
    parsed.getFullYear() + 1,
    parsed.getMonth(),
    parsed.getDate()
  );
  end.setDate(end.getDate() - 1);
  return toDateInput(end);
};

const digitsOnly = (value?: string): string => (value ?? "").replace(/\D/g, "");

export function CompanySettingsPage() {
  const queryClient = useQueryClient();

  const {
    data: companyResponse,
    isLoading,
    error: loadError,
  } = useQuery({
    queryKey: ["company"],
    queryFn: () => apiClient.get<Company>("/company"),
  });

  const company = companyResponse?.data;

  // Company form
  const {
    register: registerCompany,
    handleSubmit: handleSubmitCompany,
    reset: resetCompany,
    formState: { errors: companyErrors },
  } = useForm<CompanyFormData>({
    resolver: zodResolver(companySchema),
    defaultValues: {
      name: "",
      businessNumber: "",
      representativeName: "",
      address: "",
      detailAddress: "",
      phone: "",
      fax: "",
      email: "",
      website: "",
    },
  });

  // Fiscal year form
  const {
    register: registerFiscal,
    handleSubmit: handleSubmitFiscal,
    reset: resetFiscal,
    control: fiscalControl,
    formState: { errors: fiscalErrors },
  } = useForm<FiscalYearFormData>({
    resolver: zodResolver(fiscalYearSchema),
    defaultValues: { fiscalYearStart: "" },
  });

  const fiscalYearStart = useWatch({
    control: fiscalControl,
    name: "fiscalYearStart",
  });
  const fiscalYearEnd = deriveFiscalYearEnd(fiscalYearStart);

  // Populate both forms once the server data arrives
  useEffect(() => {
    if (!company) return;
    resetCompany({
      name: company.name ?? "",
      businessNumber: digitsOnly(company.business_number),
      representativeName: company.representative ?? "",
      address: company.address ?? "",
      detailAddress: company.address_detail ?? "",
      phone: company.phone ?? "",
      fax: company.fax ?? "",
      email: company.email ?? "",
      website: company.website ?? "",
    });
    resetFiscal({
      fiscalYearStart: fiscalStartToDateInput(company.settings?.fiscal_year_start ?? 1),
    });
  }, [company, resetCompany, resetFiscal]);

  const updateCompanyMutation = useMutation({
    mutationFn: (payload: UpdateCompanyPayload) =>
      apiClient.put<Company>("/company", payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["company"] });
      toast.success("저장 완료", "회사 정보가 저장되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error(
        "저장 실패",
        getErrorMessage(err, "회사 정보 저장 중 오류가 발생했습니다.")
      );
    },
  });

  const updateSettingsMutation = useMutation({
    mutationFn: (payload: { fiscal_year_start: number }) =>
      apiClient.put<CompanySettings>("/company/settings", payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["company"] });
      toast.success("저장 완료", "회계 기간이 저장되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error(
        "저장 실패",
        getErrorMessage(err, "회계 기간 저장 중 오류가 발생했습니다.")
      );
    },
  });

  // Handle company save
  const onSubmitCompany = (data: CompanyFormData) => {
    updateCompanyMutation.mutate({
      name: data.name,
      // Fields the screen does not expose are echoed back so PUT does not wipe them.
      name_en: company?.name_en ?? "",
      business_number: data.businessNumber,
      representative: data.representativeName,
      phone: data.phone ?? "",
      fax: data.fax ?? "",
      email: data.email ?? "",
      website: data.website ?? "",
      zip_code: company?.zip_code ?? "",
      address: data.address ?? "",
      address_detail: data.detailAddress ?? "",
      logo: company?.logo ?? "",
    });
  };

  // Handle fiscal year save
  const onSubmitFiscal = (data: FiscalYearFormData) => {
    const month = Number(data.fiscalYearStart.slice(5, 7));
    if (!Number.isInteger(month) || month < 1 || month > 12) {
      toast.error("저장 실패", "회계연도 시작일이 올바르지 않습니다.");
      return;
    }
    updateSettingsMutation.mutate({ fiscal_year_start: month });
  };

  // No upload endpoint exists; the server stores a logo URL (max 500 chars).
  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.target.value = "";
    notifyUnavailable("회사 로고 업로드");
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (loadError || !company) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-center">
        <p className="text-destructive mb-2">회사 정보를 불러올 수 없습니다.</p>
        <p className="text-sm text-muted-foreground">
          {getErrorMessage(loadError, "잠시 후 다시 시도해주세요.")}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">회사 정보</h1>
          <p className="text-muted-foreground">회사 기본 정보 및 사업장을 관리합니다.</p>
        </div>
      </div>

      {/* Company Basic Info */}
      <form onSubmit={handleSubmitCompany(onSubmitCompany)}>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg flex items-center">
              <Building2 className="h-5 w-5 mr-2" />
              기본 정보
            </CardTitle>
            <Button type="submit" isLoading={updateCompanyMutation.isPending}>
              <Save className="h-4 w-4 mr-2" />
              저장
            </Button>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {/* Logo Upload */}
              <div className="md:col-span-2 lg:col-span-3">
                <label className="block text-sm font-medium mb-2">회사 로고</label>
                <div className="flex items-center space-x-4">
                  <div className="w-24 h-24 border-2 border-dashed rounded-lg flex items-center justify-center bg-muted/50 overflow-hidden">
                    {company.logo ? (
                      <img
                        src={company.logo}
                        alt="Company logo"
                        className="w-full h-full object-contain"
                      />
                    ) : (
                      <Building2 className="h-8 w-8 text-muted-foreground" />
                    )}
                  </div>
                  <div>
                    <label className="cursor-pointer inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium border border-input bg-background shadow-sm hover:bg-accent hover:text-accent-foreground h-9 px-4 py-2">
                      <input
                        type="file"
                        accept="image/*"
                        className="hidden"
                        onChange={handleLogoUpload}
                      />
                      <Upload className="h-4 w-4 mr-2" />
                      로고 업로드
                    </label>
                    <p className="text-xs text-muted-foreground mt-1">
                      PNG, JPG (최대 2MB)
                    </p>
                  </div>
                </div>
              </div>

              {/* Company Name */}
              <Input
                label="상호명"
                required
                placeholder="회사명을 입력하세요"
                error={companyErrors.name?.message}
                {...registerCompany("name")}
              />

              {/* Business Number */}
              <Input
                label="사업자등록번호"
                required
                placeholder="'-' 없이 입력"
                error={companyErrors.businessNumber?.message}
                {...registerCompany("businessNumber")}
              />

              {/* Representative Name */}
              <Input
                label="대표자명"
                required
                placeholder="대표자 성명"
                error={companyErrors.representativeName?.message}
                {...registerCompany("representativeName")}
              />

              <div className="md:col-span-2 lg:col-span-3">
                <FeatureUnavailable
                  feature="업태·종목 저장"
                  detail="서버의 회사 정보 API에 업태·종목 항목이 없어 저장되지 않습니다. 아래 두 항목은 입력할 수 없도록 잠갔습니다."
                />
              </div>

              {/* Business Type - no server field */}
              <Input
                label="업태"
                placeholder="예: 서비스업"
                disabled
              />

              {/* Business Category - no server field */}
              <Input
                label="종목"
                placeholder="예: 소프트웨어 개발"
                disabled
              />

              {/* Address */}
              <div className="md:col-span-2">
                <Input
                  label="주소"
                  placeholder="사업장 주소"
                  {...registerCompany("address")}
                />
              </div>

              {/* Detail Address */}
              <Input
                label="상세주소"
                placeholder="상세 주소"
                {...registerCompany("detailAddress")}
              />

              {/* Phone */}
              <Input
                label="전화번호"
                placeholder="'-' 없이 입력"
                {...registerCompany("phone")}
              />

              {/* Fax */}
              <Input
                label="팩스번호"
                placeholder="'-' 없이 입력"
                {...registerCompany("fax")}
              />

              {/* Email */}
              <Input
                label="대표 이메일"
                type="email"
                placeholder="company@example.com"
                error={companyErrors.email?.message}
                {...registerCompany("email")}
              />

              {/* Website */}
              <Input
                label="홈페이지"
                placeholder="https://example.com"
                error={companyErrors.website?.message}
                {...registerCompany("website")}
              />
            </div>
          </CardContent>
        </Card>
      </form>

      {/* Fiscal Year Settings */}
      <form onSubmit={handleSubmitFiscal(onSubmitFiscal)}>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg flex items-center">
              <Calendar className="h-5 w-5 mr-2" />
              회계 기간 설정
            </CardTitle>
            <Button type="submit" isLoading={updateSettingsMutation.isPending}>
              <Save className="h-4 w-4 mr-2" />
              저장
            </Button>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <Input
                type="date"
                label="회계연도 시작일"
                required
                error={fiscalErrors.fiscalYearStart?.message}
                {...registerFiscal("fiscalYearStart")}
              />
              <Input
                type="date"
                label="회계연도 종료일"
                readOnly
                value={fiscalYearEnd}
                helperText="시작일에서 자동 계산됩니다."
              />
            </div>
            <div className="mt-4">
              <FeatureUnavailable
                feature="회계연도 종료일 개별 지정"
                detail="서버는 회계연도 시작 '월'만 저장합니다. 시작일의 월만 반영되며 종료일은 저장되지 않고 자동 계산됩니다."
              />
            </div>
            <p className="text-sm text-muted-foreground mt-4">
              <FileText className="inline h-4 w-4 mr-1" />
              회계연도 변경 시 기존 재무 데이터에 영향을 줄 수 있습니다. 신중하게 설정해주세요.
            </p>
          </CardContent>
        </Card>
      </form>

      {/* Branch Management */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-lg flex items-center">
            <MapPin className="h-5 w-5 mr-2" />
            사업장 관리
          </CardTitle>
          <Button onClick={() => notifyUnavailable("사업장 등록")}>
            <Plus className="h-4 w-4 mr-2" />
            사업장 추가
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <FeatureUnavailable
            feature="사업장(지점) 관리"
            detail="서버에 사업장 API가 없어 목록을 불러오거나 등록·수정·삭제할 수 없습니다."
          />
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>사업장명</TableHead>
                <TableHead>사업자번호</TableHead>
                <TableHead>주소</TableHead>
                <TableHead>연락처</TableHead>
                <TableHead>구분</TableHead>
                <TableHead className="w-20"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  등록된 사업장이 없습니다.
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
