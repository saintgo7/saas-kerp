import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { PARTNER_TYPES, DEFAULT_PAGE_SIZE } from "@/constants";
import { partnersApi } from "@/api";
import { getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";
import { notifyUnavailable } from "@/components/common";
import type { PartnerType } from "@/types";

const partnerTypeStyles: Record<PartnerType, { variant: "default" | "secondary" | "success"; label: string }> = {
  customer: { variant: "success", label: "고객" },
  vendor: { variant: "secondary", label: "공급업체" },
  both: { variant: "default", label: "고객/공급업체" },
};

export function PartnerListPage() {
  const queryClient = useQueryClient();

  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [selectedType, setSelectedType] = useState<string>("");
  const [showActiveOnly, setShowActiveOnly] = useState(false);
  const [page, setPage] = useState(1);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [partnerToDelete, setPartnerToDelete] = useState<string | null>(null);

  // Debounce the search box so typing does not fire a request per keystroke.
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearch(searchTerm);
      setPage(1);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [searchTerm]);

  const listParams = useMemo(
    () => ({
      page,
      pageSize: DEFAULT_PAGE_SIZE,
      search: debouncedSearch || undefined,
      partnerType: (selectedType || undefined) as PartnerType | undefined,
      isActive: showActiveOnly ? true : undefined,
    }),
    [page, debouncedSearch, selectedType, showActiveOnly]
  );

  // GET /api/v1/partners
  const {
    data: listResponse,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ["partners", "list", listParams],
    queryFn: () => partnersApi.list(listParams),
  });

  // GET /api/v1/partners/stats
  const { data: statsResponse } = useQuery({
    queryKey: ["partners", "stats"],
    queryFn: () => partnersApi.stats(),
  });

  const partners = listResponse?.data.items ?? [];
  const total = listResponse?.data.total ?? 0;
  const totalPages = listResponse?.data.totalPages ?? 1;

  const stats = statsResponse?.data;
  const totalCount = stats?.totalCount ?? total;
  const customerCount = stats?.customerCount ?? 0;
  const vendorCount = stats?.vendorCount ?? 0;
  const activeCount = stats?.activeCount ?? 0;

  // DELETE /api/v1/partners/:id
  const deleteMutation = useMutation({
    mutationFn: (id: string) => partnersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["partners"] });
      toast.success("삭제 완료", "거래처가 삭제되었습니다.");
      setDeleteModalOpen(false);
      setPartnerToDelete(null);
    },
    onError: (err: unknown) => {
      toast.error(
        "삭제 실패",
        getErrorMessage(err, "거래처 삭제 중 오류가 발생했습니다.")
      );
    },
  });

  const handleDelete = (id: string) => {
    setPartnerToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDelete = () => {
    if (!partnerToDelete) return;
    deleteMutation.mutate(partnerToDelete);
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
            <p className="text-sm text-muted-foreground">공급업체</p>
            <p className="text-2xl font-bold text-blue-500">{vendorCount}개</p>
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
              onChange={(e) => {
                setSelectedType(e.target.value);
                setPage(1);
              }}
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
                onChange={(e) => {
                  setShowActiveOnly(e.target.checked);
                  setPage(1);
                }}
              />
              <span>활성만 보기</span>
            </label>
            <Button
              variant="outline"
              onClick={() => notifyUnavailable("거래처 목록 내보내기")}
            >
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
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    불러오는 중...
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-destructive">
                    {getErrorMessage(error, "거래처 목록 조회에 실패했습니다.")}
                  </TableCell>
                </TableRow>
              ) : partners.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    검색 결과가 없습니다.
                  </TableCell>
                </TableRow>
              ) : (
                partners.map((partner) => (
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
                      <Badge
                        variant={
                          partnerTypeStyles[partner.partnerType]?.variant ??
                          "default"
                        }
                      >
                        {partnerTypeStyles[partner.partnerType]?.label ??
                          partner.partnerType}
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
                              type="button"
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
            <p className="text-sm text-muted-foreground">총 {total}건</p>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                이전
              </Button>
              <Button variant="outline" size="sm" className="bg-primary text-primary-foreground">
                {page}
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
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
            <Button
              variant="destructive"
              onClick={confirmDelete}
              isLoading={deleteMutation.isPending}
            >
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
