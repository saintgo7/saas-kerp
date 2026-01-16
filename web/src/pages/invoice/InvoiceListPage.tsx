import { useState } from "react";
import { Link } from "react-router-dom";
import {
  Plus,
  Search,
  Filter,
  Download,
  Upload,
  Send,
  MoreHorizontal,
  Eye,
  Edit,
  Trash2,
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
import { formatCurrency, formatDate, formatBusinessNumber } from "@/lib/utils";
import { INVOICE_STATUS, INVOICE_TYPES } from "@/constants";
import type { InvoiceStatus } from "@/types";

// Mock data
const mockInvoices = [
  {
    id: "1",
    invoiceNumber: "20240115-001",
    issueDate: "2024-01-15",
    direction: "issued" as const,
    buyerName: "A상사",
    buyerBusinessNumber: "1234567890",
    supplyAmount: 10000000,
    taxAmount: 1000000,
    totalAmount: 11000000,
    status: "sent" as InvoiceStatus,
    ntsConfirmNumber: "20240115123456789",
  },
  {
    id: "2",
    invoiceNumber: "20240114-001",
    issueDate: "2024-01-14",
    direction: "issued" as const,
    buyerName: "B기업",
    buyerBusinessNumber: "9876543210",
    supplyAmount: 5000000,
    taxAmount: 500000,
    totalAmount: 5500000,
    status: "issued" as InvoiceStatus,
    ntsConfirmNumber: null,
  },
  {
    id: "3",
    invoiceNumber: "R20240113-001",
    issueDate: "2024-01-13",
    direction: "received" as const,
    buyerName: "C물류",
    buyerBusinessNumber: "5555666677",
    supplyAmount: 2000000,
    taxAmount: 200000,
    totalAmount: 2200000,
    status: "sent" as InvoiceStatus,
    ntsConfirmNumber: "20240113987654321",
  },
  {
    id: "4",
    invoiceNumber: "20240112-001",
    issueDate: "2024-01-12",
    direction: "issued" as const,
    buyerName: "D테크",
    buyerBusinessNumber: "1112223334",
    supplyAmount: 15000000,
    taxAmount: 1500000,
    totalAmount: 16500000,
    status: "draft" as InvoiceStatus,
    ntsConfirmNumber: null,
  },
];

const statusStyles: Record<InvoiceStatus, { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }> = {
  draft: { variant: "secondary", label: "작성중" },
  issued: { variant: "warning", label: "발행" },
  sent: { variant: "success", label: "전송완료" },
  cancelled: { variant: "destructive", label: "취소" },
};

export function InvoiceListPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [direction, setDirection] = useState<string>("");

  const filteredInvoices = mockInvoices.filter((invoice) => {
    const matchesSearch =
      invoice.invoiceNumber.toLowerCase().includes(searchTerm.toLowerCase()) ||
      invoice.buyerName.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = !selectedStatus || invoice.status === selectedStatus;
    const matchesDirection = !direction || invoice.direction === direction;
    return matchesSearch && matchesStatus && matchesDirection;
  });

  // Calculate totals
  const totalSupply = filteredInvoices.reduce((sum, inv) => sum + inv.supplyAmount, 0);
  const totalTax = filteredInvoices.reduce((sum, inv) => sum + inv.taxAmount, 0);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">세금계산서</h1>
          <p className="text-muted-foreground">세금계산서를 발행하고 관리합니다.</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Upload className="h-4 w-4 mr-2" />
            홈택스 연동
          </Button>
          <Link to="/invoice/issue">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              세금계산서 발행
            </Button>
          </Link>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">전체 건수</p>
            <p className="text-2xl font-bold">{filteredInvoices.length}건</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">공급가액 합계</p>
            <p className="text-2xl font-bold">{formatCurrency(totalSupply, { compact: true })}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">세액 합계</p>
            <p className="text-2xl font-bold">{formatCurrency(totalTax, { compact: true })}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">미전송</p>
            <p className="text-2xl font-bold text-warning">
              {filteredInvoices.filter((i) => i.status === "issued").length}건
            </p>
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
                  placeholder="세금계산서 번호 또는 거래처명 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={direction}
              onChange={(e) => setDirection(e.target.value)}
            >
              <option value="">매출/매입 전체</option>
              <option value="issued">매출</option>
              <option value="received">매입</option>
            </select>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedStatus}
              onChange={(e) => setSelectedStatus(e.target.value)}
            >
              <option value="">전체 상태</option>
              {INVOICE_STATUS.map((status) => (
                <option key={status.value} value={status.value}>
                  {status.label}
                </option>
              ))}
            </select>
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
          <CardTitle className="text-lg">세금계산서 목록</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>세금계산서 번호</TableHead>
                <TableHead>발행일</TableHead>
                <TableHead>구분</TableHead>
                <TableHead>거래처</TableHead>
                <TableHead>사업자번호</TableHead>
                <TableHead className="text-right">공급가액</TableHead>
                <TableHead className="text-right">세액</TableHead>
                <TableHead>상태</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredInvoices.map((invoice) => (
                <TableRow key={invoice.id}>
                  <TableCell>
                    <Link
                      to={`/invoice/${invoice.id}`}
                      className="font-medium text-primary hover:underline"
                    >
                      {invoice.invoiceNumber}
                    </Link>
                  </TableCell>
                  <TableCell>{formatDate(invoice.issueDate)}</TableCell>
                  <TableCell>
                    <Badge variant={invoice.direction === "issued" ? "default" : "secondary"}>
                      {invoice.direction === "issued" ? "매출" : "매입"}
                    </Badge>
                  </TableCell>
                  <TableCell>{invoice.buyerName}</TableCell>
                  <TableCell className="font-mono text-sm">
                    {formatBusinessNumber(invoice.buyerBusinessNumber)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(invoice.supplyAmount, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(invoice.taxAmount, { showSymbol: false })}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusStyles[invoice.status].variant}>
                      {statusStyles[invoice.status].label}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="relative group">
                      <Button variant="ghost" size="icon">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                      <div className="absolute right-0 hidden group-hover:block z-10">
                        <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[140px]">
                          <Link
                            to={`/invoice/${invoice.id}`}
                            className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                          >
                            <Eye className="h-4 w-4 mr-2" />
                            상세보기
                          </Link>
                          {invoice.status === "issued" && (
                            <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                              <Send className="h-4 w-4 mr-2" />
                              국세청 전송
                            </button>
                          )}
                          {invoice.status === "draft" && (
                            <Link
                              to={`/invoice/${invoice.id}/edit`}
                              className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                            >
                              <Edit className="h-4 w-4 mr-2" />
                              수정
                            </Link>
                          )}
                          <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                            <Download className="h-4 w-4 mr-2" />
                            PDF 다운로드
                          </button>
                        </div>
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
