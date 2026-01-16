import { useState } from "react";
import { Link } from "react-router-dom";
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
} from "@/components/ui";
import { formatCurrency, formatDate } from "@/lib/utils";
import { VOUCHER_STATUS } from "@/constants";
import type { VoucherStatus } from "@/types";

// Mock data
const mockVouchers = [
  {
    id: "1",
    voucherNumber: "2024-0125",
    voucherDate: "2024-01-15",
    description: "거래처 매입 - A상사",
    status: "approved" as VoucherStatus,
    totalDebit: 5500000,
    totalCredit: 5500000,
    createdBy: "홍길동",
    createdAt: "2024-01-15T09:30:00",
  },
  {
    id: "2",
    voucherNumber: "2024-0124",
    voucherDate: "2024-01-14",
    description: "제품 판매 - B기업",
    status: "pending" as VoucherStatus,
    totalDebit: 12000000,
    totalCredit: 12000000,
    createdBy: "김철수",
    createdAt: "2024-01-14T14:20:00",
  },
  {
    id: "3",
    voucherNumber: "2024-0123",
    voucherDate: "2024-01-13",
    description: "급여 지급",
    status: "approved" as VoucherStatus,
    totalDebit: 8500000,
    totalCredit: 8500000,
    createdBy: "이영희",
    createdAt: "2024-01-13T10:00:00",
  },
  {
    id: "4",
    voucherNumber: "2024-0122",
    voucherDate: "2024-01-12",
    description: "사무용품 구입",
    status: "draft" as VoucherStatus,
    totalDebit: 320000,
    totalCredit: 320000,
    createdBy: "박민수",
    createdAt: "2024-01-12T16:45:00",
  },
  {
    id: "5",
    voucherNumber: "2024-0121",
    voucherDate: "2024-01-11",
    description: "임대료 지급",
    status: "rejected" as VoucherStatus,
    totalDebit: 2000000,
    totalCredit: 2000000,
    createdBy: "최지현",
    createdAt: "2024-01-11T11:15:00",
  },
];

const statusStyles: Record<VoucherStatus, { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }> = {
  draft: { variant: "secondary", label: "작성중" },
  pending: { variant: "warning", label: "승인대기" },
  approved: { variant: "success", label: "승인완료" },
  rejected: { variant: "destructive", label: "반려" },
};

export function VoucherListPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [selectedRows, setSelectedRows] = useState<string[]>([]);

  const filteredVouchers = mockVouchers.filter((voucher) => {
    const matchesSearch =
      voucher.voucherNumber.toLowerCase().includes(searchTerm.toLowerCase()) ||
      voucher.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = !selectedStatus || voucher.status === selectedStatus;
    return matchesSearch && matchesStatus;
  });

  const toggleRowSelection = (id: string) => {
    setSelectedRows((prev) =>
      prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]
    );
  };

  const toggleAllRows = () => {
    if (selectedRows.length === filteredVouchers.length) {
      setSelectedRows([]);
    } else {
      setSelectedRows(filteredVouchers.map((v) => v.id));
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
              onChange={(e) => setSelectedStatus(e.target.value)}
            >
              <option value="">전체 상태</option>
              {VOUCHER_STATUS.map((status) => (
                <option key={status.value} value={status.value}>
                  {status.label}
                </option>
              ))}
            </select>
            <Button variant="outline" size="icon">
              <Filter className="h-4 w-4" />
            </Button>
            <Button variant="outline">
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
                <Button variant="outline" size="sm">
                  <Check className="h-4 w-4 mr-2" />
                  일괄 승인
                </Button>
                <Button variant="outline" size="sm" className="text-destructive">
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
                    checked={selectedRows.length === filteredVouchers.length}
                    onChange={toggleAllRows}
                  />
                </TableHead>
                <TableHead>전표번호</TableHead>
                <TableHead>전표일자</TableHead>
                <TableHead>적요</TableHead>
                <TableHead className="text-right">차변</TableHead>
                <TableHead className="text-right">대변</TableHead>
                <TableHead>상태</TableHead>
                <TableHead>작성자</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredVouchers.map((voucher) => (
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
                      {voucher.voucherNumber}
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
                    <Badge variant={statusStyles[voucher.status].variant}>
                      {statusStyles[voucher.status].label}
                    </Badge>
                  </TableCell>
                  <TableCell>{voucher.createdBy}</TableCell>
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
                          <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted text-destructive">
                            <Trash2 className="h-4 w-4 mr-2" />
                            삭제
                          </button>
                        </div>
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {/* Pagination */}
          <div className="flex items-center justify-between mt-4 pt-4 border-t">
            <p className="text-sm text-muted-foreground">
              총 {filteredVouchers.length}건
            </p>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" disabled>
                이전
              </Button>
              <Button variant="outline" size="sm" className="bg-primary text-primary-foreground">
                1
              </Button>
              <Button variant="outline" size="sm">
                2
              </Button>
              <Button variant="outline" size="sm">
                3
              </Button>
              <Button variant="outline" size="sm">
                다음
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
