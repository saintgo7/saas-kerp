import { useState } from "react";
import { Link } from "react-router-dom";
import {
  Plus,
  Search,
  Download,
  MoreHorizontal,
  Eye,
  Edit,
  Trash2,
  Building2,
  Phone,
  Mail,
  CheckCircle,
  XCircle,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  Modal,
} from "@/components/ui";
import { formatBusinessNumber, formatPhoneNumber } from "@/lib/utils";
import { PARTNER_TYPES } from "@/constants";
import type { PartnerType } from "@/types";

// Mock data
const mockPartners = [
  {
    id: "1",
    code: "P001",
    name: "(주)테크솔루션",
    businessNumber: "1234567890",
    representativeName: "김대표",
    partnerType: "both" as PartnerType,
    address: "서울시 강남구 테헤란로 123",
    phone: "0212345678",
    email: "contact@techsolution.co.kr",
    isActive: true,
    createdAt: "2024-01-15",
  },
  {
    id: "2",
    code: "P002",
    name: "에이플러스 전자",
    businessNumber: "2345678901",
    representativeName: "이사장",
    partnerType: "supplier" as PartnerType,
    address: "경기도 성남시 분당구 판교로 456",
    phone: "0311234567",
    email: "sales@aplus.co.kr",
    isActive: true,
    createdAt: "2024-01-10",
  },
  {
    id: "3",
    code: "P003",
    name: "글로벌무역상사",
    businessNumber: "3456789012",
    representativeName: "박무역",
    partnerType: "customer" as PartnerType,
    address: "부산시 해운대구 마린시티로 789",
    phone: "0519876543",
    email: "trade@global.co.kr",
    isActive: true,
    createdAt: "2024-01-05",
  },
  {
    id: "4",
    code: "P004",
    name: "스마트오피스",
    businessNumber: "4567890123",
    representativeName: "최스마트",
    partnerType: "supplier" as PartnerType,
    address: "대전시 유성구 대학로 101",
    phone: "0421112233",
    email: "info@smartoffice.kr",
    isActive: false,
    createdAt: "2023-12-20",
  },
  {
    id: "5",
    code: "P005",
    name: "우리물류",
    businessNumber: "5678901234",
    representativeName: "강물류",
    partnerType: "supplier" as PartnerType,
    address: "인천시 남동구 논현로 202",
    phone: "0324445566",
    email: "delivery@woolilogis.com",
    isActive: true,
    createdAt: "2023-12-15",
  },
];

const partnerTypeStyles: Record<PartnerType, { variant: "default" | "secondary" | "success"; label: string }> = {
  customer: { variant: "success", label: "고객" },
  supplier: { variant: "secondary", label: "거래처" },
  both: { variant: "default", label: "고객/거래처" },
};

export function PartnerListPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedType, setSelectedType] = useState<string>("");
  const [showActiveOnly, setShowActiveOnly] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [partnerToDelete, setPartnerToDelete] = useState<string | null>(null);

  const filteredPartners = mockPartners.filter((partner) => {
    const matchesSearch =
      partner.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      partner.code.toLowerCase().includes(searchTerm.toLowerCase()) ||
      partner.businessNumber?.includes(searchTerm) ||
      partner.representativeName?.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = !selectedType || partner.partnerType === selectedType;
    const matchesActive = !showActiveOnly || partner.isActive;
    return matchesSearch && matchesType && matchesActive;
  });

  // Stats
  const totalCount = mockPartners.length;
  const customerCount = mockPartners.filter((p) => p.partnerType === "customer" || p.partnerType === "both").length;
  const supplierCount = mockPartners.filter((p) => p.partnerType === "supplier" || p.partnerType === "both").length;
  const activeCount = mockPartners.filter((p) => p.isActive).length;

  const handleDelete = (id: string) => {
    setPartnerToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDelete = () => {
    // TODO: API call to delete partner
    console.log("Deleting partner:", partnerToDelete);
    setDeleteModalOpen(false);
    setPartnerToDelete(null);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">거래처 관리</h1>
          <p className="text-muted-foreground">거래처 정보를 조회하고 관리합니다.</p>
        </div>
        <Link to="/partners/new">
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            거래처 등록
          </Button>
        </Link>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">전체 거래처</p>
            <p className="text-2xl font-bold">{totalCount}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">고객</p>
            <p className="text-2xl font-bold text-success">{customerCount}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">거래처(매입)</p>
            <p className="text-2xl font-bold text-blue-500">{supplierCount}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">활성 거래처</p>
            <p className="text-2xl font-bold">{activeCount}개</p>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="거래처명, 코드, 사업자번호로 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedType}
              onChange={(e) => setSelectedType(e.target.value)}
            >
              <option value="">전체 유형</option>
              {PARTNER_TYPES.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
            <label className="flex items-center space-x-2 text-sm">
              <input
                type="checkbox"
                className="rounded border-input"
                checked={showActiveOnly}
                onChange={(e) => setShowActiveOnly(e.target.checked)}
              />
              <span>활성만 보기</span>
            </label>
            <Button variant="outline">
              <Download className="h-4 w-4 mr-2" />
              내보내기
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">거래처 목록</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[100px]">코드</TableHead>
                <TableHead>거래처명</TableHead>
                <TableHead>사업자번호</TableHead>
                <TableHead>대표자</TableHead>
                <TableHead>유형</TableHead>
                <TableHead>연락처</TableHead>
                <TableHead>상태</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredPartners.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    검색 결과가 없습니다.
                  </TableCell>
                </TableRow>
              ) : (
                filteredPartners.map((partner) => (
                  <TableRow key={partner.id}>
                    <TableCell className="font-mono">{partner.code}</TableCell>
                    <TableCell>
                      <Link
                        to={`/partners/${partner.id}`}
                        className="font-medium text-primary hover:underline flex items-center"
                      >
                        <Building2 className="h-4 w-4 mr-2 text-muted-foreground" />
                        {partner.name}
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono">
                      {partner.businessNumber ? formatBusinessNumber(partner.businessNumber) : "-"}
                    </TableCell>
                    <TableCell>{partner.representativeName || "-"}</TableCell>
                    <TableCell>
                      <Badge variant={partnerTypeStyles[partner.partnerType].variant}>
                        {partnerTypeStyles[partner.partnerType].label}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="space-y-1">
                        {partner.phone && (
                          <div className="flex items-center text-sm">
                            <Phone className="h-3 w-3 mr-1 text-muted-foreground" />
                            {formatPhoneNumber(partner.phone)}
                          </div>
                        )}
                        {partner.email && (
                          <div className="flex items-center text-sm text-muted-foreground">
                            <Mail className="h-3 w-3 mr-1" />
                            {partner.email}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {partner.isActive ? (
                        <span className="flex items-center text-success text-sm">
                          <CheckCircle className="h-4 w-4 mr-1" />
                          활성
                        </span>
                      ) : (
                        <span className="flex items-center text-muted-foreground text-sm">
                          <XCircle className="h-4 w-4 mr-1" />
                          비활성
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="relative group">
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                        <div className="absolute right-0 hidden group-hover:block z-10">
                          <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[120px]">
                            <Link
                              to={`/partners/${partner.id}`}
                              className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                            >
                              <Eye className="h-4 w-4 mr-2" />
                              상세보기
                            </Link>
                            <Link
                              to={`/partners/${partner.id}/edit`}
                              className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                            >
                              <Edit className="h-4 w-4 mr-2" />
                              수정
                            </Link>
                            <button
                              onClick={() => handleDelete(partner.id)}
                              className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted text-destructive"
                            >
                              <Trash2 className="h-4 w-4 mr-2" />
                              삭제
                            </button>
                          </div>
                        </div>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>

          {/* Pagination */}
          <div className="flex items-center justify-between mt-4 pt-4 border-t">
            <p className="text-sm text-muted-foreground">
              총 {filteredPartners.length}건
            </p>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" disabled>
                이전
              </Button>
              <Button variant="outline" size="sm" className="bg-primary text-primary-foreground">
                1
              </Button>
              <Button variant="outline" size="sm">
                다음
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="거래처 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            이 거래처를 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              취소
            </Button>
            <Button variant="destructive" onClick={confirmDelete}>
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
