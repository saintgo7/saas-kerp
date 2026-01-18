import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Building2,
  Save,
  Upload,
  Calendar,
  MapPin,
  FileText,
  Plus,
  Trash2,
  Edit,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Modal,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { formatBusinessNumber, formatPhoneNumber } from "@/lib/utils";
import { toast } from "@/stores/ui";

// Validation schema for company settings
const companySchema = z.object({
  name: z.string().min(1, "상호명을 입력하세요"),
  businessNumber: z
    .string()
    .regex(/^\d{10}$/, "사업자등록번호는 10자리 숫자입니다"),
  representativeName: z.string().min(1, "대표자명을 입력하세요"),
  businessType: z.string().optional(),
  businessCategory: z.string().optional(),
  address: z.string().optional(),
  detailAddress: z.string().optional(),
  phone: z.string().optional(),
  fax: z.string().optional(),
  email: z.string().email("올바른 이메일 형식이 아닙니다").optional().or(z.literal("")),
  website: z.string().url("올바른 URL 형식이 아닙니다").optional().or(z.literal("")),
});

const fiscalYearSchema = z.object({
  fiscalYearStart: z.string().min(1, "회계연도 시작일을 선택하세요"),
  fiscalYearEnd: z.string().min(1, "회계연도 종료일을 선택하세요"),
});

const branchSchema = z.object({
  name: z.string().min(1, "사업장명을 입력하세요"),
  businessNumber: z.string().optional(),
  address: z.string().min(1, "주소를 입력하세요"),
  phone: z.string().optional(),
  isHeadquarters: z.boolean().optional(),
});

type CompanyFormData = z.infer<typeof companySchema>;
type FiscalYearFormData = z.infer<typeof fiscalYearSchema>;
type BranchFormData = z.infer<typeof branchSchema>;

// Mock company data
const mockCompany: CompanyFormData = {
  name: "(주)테크솔루션",
  businessNumber: "1234567890",
  representativeName: "김대표",
  businessType: "서비스업",
  businessCategory: "소프트웨어 개발",
  address: "서울특별시 강남구 테헤란로 123",
  detailAddress: "테크빌딩 10층",
  phone: "0212345678",
  fax: "0212345679",
  email: "contact@techsolution.co.kr",
  website: "https://techsolution.co.kr",
};

// Mock fiscal year data
const mockFiscalYear: FiscalYearFormData = {
  fiscalYearStart: "2024-01-01",
  fiscalYearEnd: "2024-12-31",
};

// Mock branches data
const mockBranches = [
  {
    id: "1",
    name: "본사",
    businessNumber: "1234567890",
    address: "서울특별시 강남구 테헤란로 123",
    phone: "0212345678",
    isHeadquarters: true,
  },
  {
    id: "2",
    name: "부산지사",
    businessNumber: "1234567891",
    address: "부산광역시 해운대구 마린시티로 456",
    phone: "0519876543",
    isHeadquarters: false,
  },
  {
    id: "3",
    name: "대전지사",
    businessNumber: "1234567892",
    address: "대전광역시 유성구 대학로 789",
    phone: "0421112233",
    isHeadquarters: false,
  },
];

export function CompanySettingsPage() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [logoPreview, setLogoPreview] = useState<string | null>(null);
  const [branches, setBranches] = useState(mockBranches);
  const [branchModalOpen, setBranchModalOpen] = useState(false);
  const [editingBranch, setEditingBranch] = useState<typeof mockBranches[0] | null>(null);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [branchToDelete, setBranchToDelete] = useState<string | null>(null);

  // Company form
  const {
    register: registerCompany,
    handleSubmit: handleSubmitCompany,
    formState: { errors: companyErrors },
  } = useForm<CompanyFormData>({
    resolver: zodResolver(companySchema),
    defaultValues: mockCompany,
  });

  // Fiscal year form
  const {
    register: registerFiscal,
    handleSubmit: handleSubmitFiscal,
    formState: { errors: fiscalErrors },
  } = useForm<FiscalYearFormData>({
    resolver: zodResolver(fiscalYearSchema),
    defaultValues: mockFiscalYear,
  });

  // Branch form
  const {
    register: registerBranch,
    handleSubmit: handleSubmitBranch,
    reset: resetBranch,
    formState: { errors: branchErrors },
  } = useForm<BranchFormData>({
    resolver: zodResolver(branchSchema),
  });

  // Handle company save
  const onSubmitCompany = async (data: CompanyFormData) => {
    setIsSubmitting(true);
    try {
      // TODO: API call
      console.log("Company data:", data);
      toast.success("저장 완료", "회사 정보가 저장되었습니다.");
    } catch {
      toast.error("저장 실패", "회사 정보 저장 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle fiscal year save
  const onSubmitFiscal = async (data: FiscalYearFormData) => {
    setIsSubmitting(true);
    try {
      // TODO: API call
      console.log("Fiscal year data:", data);
      toast.success("저장 완료", "회계 기간이 저장되었습니다.");
    } catch {
      toast.error("저장 실패", "회계 기간 저장 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle logo upload
  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (file.size > 2 * 1024 * 1024) {
        toast.error("파일 크기 초과", "로고 파일은 2MB 이하여야 합니다.");
        return;
      }
      const reader = new FileReader();
      reader.onload = () => {
        setLogoPreview(reader.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  // Handle branch add/edit
  const openBranchModal = (branch?: typeof mockBranches[0]) => {
    if (branch) {
      setEditingBranch(branch);
      resetBranch({
        name: branch.name,
        businessNumber: branch.businessNumber,
        address: branch.address,
        phone: branch.phone,
        isHeadquarters: branch.isHeadquarters,
      });
    } else {
      setEditingBranch(null);
      resetBranch({
        name: "",
        businessNumber: "",
        address: "",
        phone: "",
        isHeadquarters: false,
      });
    }
    setBranchModalOpen(true);
  };

  const onSubmitBranch = async (data: BranchFormData) => {
    try {
      const branchData = {
        name: data.name,
        address: data.address,
        businessNumber: data.businessNumber || "",
        phone: data.phone || "",
        isHeadquarters: data.isHeadquarters ?? false,
      };

      if (editingBranch) {
        // Update existing branch
        setBranches(
          branches.map((b) =>
            b.id === editingBranch.id ? { ...b, ...branchData } : b
          )
        );
        toast.success("수정 완료", "사업장 정보가 수정되었습니다.");
      } else {
        // Add new branch
        setBranches([
          ...branches,
          {
            id: Date.now().toString(),
            ...branchData,
          },
        ]);
        toast.success("등록 완료", "사업장이 등록되었습니다.");
      }
      setBranchModalOpen(false);
    } catch {
      toast.error("저장 실패", "사업장 저장 중 오류가 발생했습니다.");
    }
  };

  // Handle branch delete
  const handleDeleteBranch = (id: string) => {
    const branch = branches.find((b) => b.id === id);
    if (branch?.isHeadquarters) {
      toast.error("삭제 불가", "본사는 삭제할 수 없습니다.");
      return;
    }
    setBranchToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDeleteBranch = () => {
    if (branchToDelete) {
      setBranches(branches.filter((b) => b.id !== branchToDelete));
      toast.success("삭제 완료", "사업장이 삭제되었습니다.");
    }
    setDeleteModalOpen(false);
    setBranchToDelete(null);
  };

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
            <Button type="submit" isLoading={isSubmitting}>
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
                    {logoPreview ? (
                      <img
                        src={logoPreview}
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

              {/* Business Type */}
              <Input
                label="업태"
                placeholder="예: 서비스업"
                {...registerCompany("businessType")}
              />

              {/* Business Category */}
              <Input
                label="종목"
                placeholder="예: 소프트웨어 개발"
                {...registerCompany("businessCategory")}
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
            <Button type="submit" isLoading={isSubmitting}>
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
                required
                error={fiscalErrors.fiscalYearEnd?.message}
                {...registerFiscal("fiscalYearEnd")}
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
          <Button onClick={() => openBranchModal()}>
            <Plus className="h-4 w-4 mr-2" />
            사업장 추가
          </Button>
        </CardHeader>
        <CardContent>
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
              {branches.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    등록된 사업장이 없습니다.
                  </TableCell>
                </TableRow>
              ) : (
                branches.map((branch) => (
                  <TableRow key={branch.id}>
                    <TableCell className="font-medium">{branch.name}</TableCell>
                    <TableCell className="font-mono">
                      {branch.businessNumber
                        ? formatBusinessNumber(branch.businessNumber)
                        : "-"}
                    </TableCell>
                    <TableCell>{branch.address}</TableCell>
                    <TableCell>
                      {branch.phone ? formatPhoneNumber(branch.phone) : "-"}
                    </TableCell>
                    <TableCell>
                      {branch.isHeadquarters ? (
                        <Badge variant="default">본사</Badge>
                      ) : (
                        <Badge variant="secondary">지사</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center space-x-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => openBranchModal(branch)}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleDeleteBranch(branch.id)}
                          disabled={branch.isHeadquarters}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Branch Add/Edit Modal */}
      <Modal
        isOpen={branchModalOpen}
        onClose={() => setBranchModalOpen(false)}
        title={editingBranch ? "사업장 수정" : "사업장 등록"}
        size="lg"
      >
        <form onSubmit={handleSubmitBranch(onSubmitBranch)} className="space-y-4">
          <Input
            label="사업장명"
            required
            placeholder="사업장 이름"
            error={branchErrors.name?.message}
            {...registerBranch("name")}
          />
          <Input
            label="사업자등록번호"
            placeholder="별도 사업자번호가 있는 경우"
            {...registerBranch("businessNumber")}
          />
          <Input
            label="주소"
            required
            placeholder="사업장 주소"
            error={branchErrors.address?.message}
            {...registerBranch("address")}
          />
          <Input
            label="연락처"
            placeholder="전화번호"
            {...registerBranch("phone")}
          />
          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              className="rounded border-input"
              {...registerBranch("isHeadquarters")}
            />
            <span className="text-sm">본사로 지정</span>
          </label>
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setBranchModalOpen(false)}
            >
              취소
            </Button>
            <Button type="submit">
              {editingBranch ? "수정" : "등록"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="사업장 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            이 사업장을 삭제하시겠습니까? 해당 사업장과 관련된 데이터는 유지됩니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              취소
            </Button>
            <Button variant="destructive" onClick={confirmDeleteBranch}>
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
