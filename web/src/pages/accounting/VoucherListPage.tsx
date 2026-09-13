import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Search,
  Filter,
  Download,
  MoreHorizontal,
  Eye,
  Edit,
  Trash2,
  Check,
  X,
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
import { formatCurrency, formatDate } from "@/lib/utils";
import { VOUCHER_STATUS } from "@/constants";
import { vouchersApi } from "@/api";
import { getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";
import { notifyUnavailable } from "@/components/common";
import type { VoucherStatus } from "@/hooks/useVoucher";

const PAGE_SIZE = 20;

const statusStyles: Record<
  VoucherStatus,
  {
    variant: "default" | "secondary" | "destructive" | "success" | "warning" | "info";
    label: string;
  }
> = {
  draft: { variant: "secondary", label: "작성중" },
  pending: { variant: "warning", label: "승인대기" },
  approved: { variant: "success", label: "승인완료" },
  posted: { variant: "info", label: "전기완료" },
  rejected: { variant: "destructive", label: "반려" },
  cancelled: { variant: "secondary", label: "취소" },
};

/** Fallback for a status the backend adds later; never crash the page. */
const unknownStatus = { variant: "secondary" as const, label: "알 수 없음" };

export function VoucherListPage() {
  const queryClient = useQueryClient();

  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [selectedRows, setSelectedRows] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [voucherToDelete, setVoucherToDelete] = useState<string | null>(null);

  // Keep the search box responsive without firing a request per keystroke.
  // Resetting the page/selection happens in the same timer rather than in a
  // second effect: a synchronous setState inside an effect cascades renders.
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearch(searchTerm);
      setPage(1);
      setSelectedRows([]);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [searchTerm]);

  const {
    data: response,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["vouchers", "list", { page, debouncedSearch, selectedStatus }],
    queryFn: () =>
      vouchersApi.list({
        page,
        pageSize: PAGE_SIZE,
        search: debouncedSearch || undefined,
        status: (selectedStatus as VoucherStatus) || undefined,
      }),
  });

  // No mock fallback: a list that failed to load must not look like real data.
  const vouchers = response?.data?.items ?? [];
  const total = response?.data?.total ?? 0;
  const totalPages = response?.data?.totalPages ?? 1;

  const deleteMutation = useMutation({
    mutationFn: (id: string) => vouchersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vouchers"] });
      setDeleteModalOpen(false);
      setVoucherToDelete(null);
      toast.success("삭제 완료", "전표가 삭제되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error(
        "삭제 실패",
        getErrorMessage(err, "전표 삭제 중 오류가 발생했습니다.")
      );
    },
  });

  const pageNumbers = useMemo(() => {
    const start = Math.max(1, Math.min(page - 1, totalPages - 2));
    const end = Math.min(totalPages, start + 2);
    const numbers: number[] = [];
    for (let n = start; n <= end; n += 1) numbers.push(n);
    return numbers;
  }, [page, totalPages]);

  const toggleRowSelection = (id: string) => {
    setSelectedRows((prev) =>
      prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]
    );
  };

  const toggleAllRows = () => {
    if (vouchers.length > 0 && selectedRows.length === vouchers.length) {
      setSelectedRows([]);
    } else {
      setSelectedRows(vouchers.map((v) => v.id));
    }
  };

  const handleDelete = (id: string) => {
    setVoucherToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDelete = () => {
    if (voucherToDelete) {
      deleteMutation.mutate(voucherToDelete);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">전표 관리</h1>
          <p className="text-muted-foreground">회계 전표를 조회하고 관리합니다.</p>
        </div>
        <Link to="/accounting/voucher/new">
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            전표 작성
          </Button>
        </Link>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="전표번호 또는 적요로 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedStatus}
              onChange={(e) => {
                setSelectedStatus(e.target.value);
                setPage(1);
                setSelectedRows([]);
              }}
            >
              <option value="">전체 상태</option>
              {VOUCHER_STATUS.map((status) => (
                <option key={status.value} value={status.value}>
                  {status.label}
                </option>
              ))}
            </select>
            <Button
              variant="outline"
              size="icon"
              onClick={() => notifyUnavailable("전표 상세 필터")}
            >
              <Filter className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              onClick={() => notifyUnavailable("전표 목록 내보내기")}
            >
              <Download className="h-4 w-4 mr-2" />
              내보내기
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Bulk Actions */}
      {selectedRows.length > 0 && (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">
                {selectedRows.length}개 항목 선택됨
              </span>
              <div className="flex items-center space-x-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => notifyUnavailable("전표 일괄 승인")}
                >
                  <Check className="h-4 w-4 mr-2" />
                  일괄 승인
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="text-destructive"
                  onClick={() => notifyUnavailable("전표 일괄 반려")}
                >
                  <X className="h-4 w-4 mr-2" />
                  일괄 반려
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">전표 목록</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">
                  <input
                    type="checkbox"
                    className="rounded border-input"
                    checked={vouchers.length > 0 && selectedRows.length === vouchers.length}
                    onChange={toggleAllRows}
                  />
                </TableHead>
                <TableHead>전표번호</TableHead>
                <TableHead>전표일자</TableHead>
                <TableHead>적요</TableHead>
                <TableHead className="text-right">차변</TableHead>
                <TableHead className="text-right">대변</TableHead>
                <TableHead>상태</TableHead>
                <TableHead>유형</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center py-8 text-muted-foreground">
                    전표를 불러오는 중입니다.
                  </TableCell>
                </TableRow>
              ) : error ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center py-8 text-destructive">
                    {getErrorMessage(error, "전표 목록 조회에 실패했습니다.")}
                  </TableCell>
                </TableRow>
              ) : vouchers.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center py-8 text-muted-foreground">
                    조회된 전표가 없습니다.
                  </TableCell>
                </TableRow>
              ) : (
                vouchers.map((voucher) => {
                  const style = statusStyles[voucher.status] ?? unknownStatus;
                  return (
                    <TableRow key={voucher.id}>
                      <TableCell>
                        <input
                          type="checkbox"
                          className="rounded border-input"
                          checked={selectedRows.includes(voucher.id)}
                          onChange={() => toggleRowSelection(voucher.id)}
                        />
                      </TableCell>
                      <TableCell>
                        <Link
                          to={`/accounting/voucher/${voucher.id}`}
                          className="font-medium text-primary hover:underline"
                        >
                          {voucher.voucherNo}
                        </Link>
                      </TableCell>
                      <TableCell>{formatDate(voucher.voucherDate)}</TableCell>
                      <TableCell className="max-w-[200px] truncate">
                        {voucher.description}
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatCurrency(voucher.totalDebit, { showSymbol: false })}
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatCurrency(voucher.totalCredit, { showSymbol: false })}
                      </TableCell>
                      <TableCell>
                        <Badge variant={style.variant}>
                          {voucher.statusLabel || style.label}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {voucher.voucherTypeLabel || voucher.voucherType}
                      </TableCell>
                      <TableCell>
                        <div className="relative group">
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                          <div className="absolute right-0 hidden group-hover:block z-10">
                            <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[120px]">
                              <Link
                                to={`/accounting/voucher/${voucher.id}`}
                                className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                              >
                                <Eye className="h-4 w-4 mr-2" />
                                상세보기
                              </Link>
                              <Link
                                to={`/accounting/voucher/${voucher.id}/edit`}
                                className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                              >
                                <Edit className="h-4 w-4 mr-2" />
                                수정
                              </Link>
                              <button
                                onClick={() => handleDelete(voucher.id)}
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
                  );
                })
              )}
            </TableBody>
          </Table>

          {/* Pagination */}
          <div className="flex items-center justify-between mt-4 pt-4 border-t">
            <p className="text-sm text-muted-foreground">
              총 {total}건
            </p>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                이전
              </Button>
              {pageNumbers.map((n) => (
                <Button
                  key={n}
                  variant="outline"
                  size="sm"
                  className={n === page ? "bg-primary text-primary-foreground" : ""}
                  onClick={() => setPage(n)}
                >
                  {n}
                </Button>
              ))}
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
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
        title="전표 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            이 전표를 삭제하시겠습니까? 작성중 상태의 전표만 삭제할 수 있습니다.
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
