import { useState, useMemo } from "react";
import { useForm, useFieldArray, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Plus,
  Search,
  Download,
  Upload,
  MoreHorizontal,
  Eye,
  Edit,
  Trash2,
  Save,
  X,
  Building2,
  Calculator,
  FileText,
  Calendar,
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
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  Modal,
} from "@/components/ui";
import { formatCurrency, formatDate, formatBusinessNumber, cn } from "@/lib/utils";
import { INVOICE_STATUS } from "@/constants";
import { toast } from "@/stores/ui";
import type { InvoiceStatus } from "@/types";

// Mock data for received invoices (purchases)
const mockReceivedInvoices = [
  {
    id: "1",
    invoiceNumber: "20240115-A001",
    issueDate: "2024-01-15",
    supplierName: "(주)원자재공급",
    supplierBusinessNumber: "1234567890",
    supplierCeoName: "김공급",
    supplierAddress: "서울시 강남구 테헤란로 100",
    supplierBizType: "제조업",
    supplierBizItem: "원자재",
    supplierEmail: "invoice@supplier1.com",
    supplyAmount: 5000000,
    taxAmount: 500000,
    totalAmount: 5500000,
    status: "sent" as InvoiceStatus,
    ntsConfirmNumber: "20240115123456789",
    items: [
      {
        id: "1-1",
        itemDate: "2024-01-15",
        itemName: "철강재",
        specification: "SS400",
        quantity: 100,
        unitPrice: 50000,
        supplyAmount: 5000000,
        taxAmount: 500000,
      },
    ],
    note: "",
  },
  {
    id: "2",
    invoiceNumber: "20240114-B002",
    issueDate: "2024-01-14",
    supplierName: "(주)사무용품",
    supplierBusinessNumber: "9876543210",
    supplierCeoName: "이사무",
    supplierAddress: "서울시 서초구 서초대로 200",
    supplierBizType: "도소매업",
    supplierBizItem: "사무용품",
    supplierEmail: "invoice@supplier2.com",
    supplyAmount: 1500000,
    taxAmount: 150000,
    totalAmount: 1650000,
    status: "sent" as InvoiceStatus,
    ntsConfirmNumber: "20240114987654321",
    items: [
      {
        id: "2-1",
        itemDate: "2024-01-14",
        itemName: "복합기",
        specification: "HP M428fdw",
        quantity: 1,
        unitPrice: 1000000,
        supplyAmount: 1000000,
        taxAmount: 100000,
      },
      {
        id: "2-2",
        itemDate: "2024-01-14",
        itemName: "사무용 의자",
        specification: "인체공학",
        quantity: 5,
        unitPrice: 100000,
        supplyAmount: 500000,
        taxAmount: 50000,
      },
    ],
    note: "배송비 포함",
  },
  {
    id: "3",
    invoiceNumber: "20240113-C003",
    issueDate: "2024-01-13",
    supplierName: "(주)IT서비스",
    supplierBusinessNumber: "5555666677",
    supplierCeoName: "박개발",
    supplierAddress: "경기도 성남시 분당구 판교로 300",
    supplierBizType: "서비스업",
    supplierBizItem: "소프트웨어",
    supplierEmail: "invoice@itservice.com",
    supplyAmount: 3000000,
    taxAmount: 300000,
    totalAmount: 3300000,
    status: "draft" as InvoiceStatus,
    ntsConfirmNumber: null,
    items: [
      {
        id: "3-1",
        itemDate: "2024-01-13",
        itemName: "클라우드 서비스",
        specification: "월정액",
        quantity: 1,
        unitPrice: 3000000,
        supplyAmount: 3000000,
        taxAmount: 300000,
      },
    ],
    note: "1월분 서비스 비용",
  },
  {
    id: "4",
    invoiceNumber: "20240112-D004",
    issueDate: "2024-01-12",
    supplierName: "(주)물류센터",
    supplierBusinessNumber: "1112223334",
    supplierCeoName: "최운송",
    supplierAddress: "인천시 남동구 남동대로 400",
    supplierBizType: "운수업",
    supplierBizItem: "물류",
    supplierEmail: "invoice@logistics.com",
    supplyAmount: 800000,
    taxAmount: 80000,
    totalAmount: 880000,
    status: "issued" as InvoiceStatus,
    ntsConfirmNumber: null,
    items: [
      {
        id: "4-1",
        itemDate: "2024-01-12",
        itemName: "화물 운송",
        specification: "5톤 트럭",
        quantity: 4,
        unitPrice: 200000,
        supplyAmount: 800000,
        taxAmount: 80000,
      },
    ],
    note: "",
  },
];

// Status badge styles
const statusStyles: Record<
  InvoiceStatus,
  { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }
> = {
  draft: { variant: "secondary", label: "작성중" },
  issued: { variant: "warning", label: "발행" },
  sent: { variant: "success", label: "수신완료" },
  cancelled: { variant: "destructive", label: "취소" },
};

// Invoice type options
const invoiceTypeOptions = [
  { value: "01", label: "01 일반" },
  { value: "02", label: "02 영세율" },
  { value: "03", label: "03 위수탁" },
  { value: "04", label: "04 수입" },
];

// Tax type options
const taxTypeOptions = [
  { value: "taxable", label: "과세" },
  { value: "zero", label: "영세율" },
  { value: "exempt", label: "면세" },
];

// Item schema for form validation
const invoiceItemSchema = z.object({
  itemDate: z.string().min(1, "일자를 입력하세요"),
  itemName: z.string().min(1, "품목명을 입력하세요"),
  specification: z.string().optional(),
  quantity: z.number().min(1, "수량을 입력하세요"),
  unitPrice: z.number().min(0, "단가를 입력하세요"),
  supplyAmount: z.number().min(0),
  taxAmount: z.number().min(0),
  note: z.string().optional(),
});

// Main invoice schema for form validation
const invoiceSchema = z.object({
  // Supplier info
  supplierBizNo: z.string().min(10, "사업자번호 10자리를 입력하세요"),
  supplierName: z.string().min(1, "상호를 입력하세요"),
  supplierCeoName: z.string().min(1, "대표자명을 입력하세요"),
  supplierAddress: z.string().optional(),
  supplierBizType: z.string().optional(),
  supplierBizItem: z.string().optional(),
  supplierEmail: z.string().email("올바른 이메일을 입력하세요").optional().or(z.literal("")),

  // Invoice details
  invoiceType: z.string().min(1, "종류를 선택하세요"),
  taxType: z.string().min(1, "과세유형을 선택하세요"),
  issueDate: z.string().min(1, "작성일자를 입력하세요"),
  note: z.string().optional(),

  // Items
  items: z.array(invoiceItemSchema).min(1, "최소 1개 이상의 품목이 필요합니다"),
});

type InvoiceFormData = z.infer<typeof invoiceSchema>;

// Mock suppliers for search
const mockSuppliers = [
  {
    bizNo: "1234567890",
    name: "(주)원자재공급",
    ceoName: "김공급",
    address: "서울시 강남구 테헤란로 100",
    bizType: "제조업",
    bizItem: "원자재",
  },
  {
    bizNo: "9876543210",
    name: "(주)사무용품",
    ceoName: "이사무",
    address: "서울시 서초구 서초대로 200",
    bizType: "도소매업",
    bizItem: "사무용품",
  },
  {
    bizNo: "5555666677",
    name: "(주)IT서비스",
    ceoName: "박개발",
    address: "경기도 성남시 분당구 판교로 300",
    bizType: "서비스업",
    bizItem: "소프트웨어",
  },
];

export function InvoiceReceivedPage() {
  // Filter states
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  // Modal states
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [selectedInvoice, setSelectedInvoice] = useState<(typeof mockReceivedInvoices)[0] | null>(
    null
  );
  const [showSupplierSearch, setShowSupplierSearch] = useState(false);
  const [supplierSearchQuery, setSupplierSearchQuery] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isEditing, setIsEditing] = useState(false);

  // Form setup
  const {
    register,
    control,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<InvoiceFormData>({
    resolver: zodResolver(invoiceSchema),
    defaultValues: {
      supplierBizNo: "",
      supplierName: "",
      supplierCeoName: "",
      supplierAddress: "",
      supplierBizType: "",
      supplierBizItem: "",
      supplierEmail: "",
      invoiceType: "01",
      taxType: "taxable",
      issueDate: new Date().toISOString().split("T")[0],
      note: "",
      items: [
        {
          itemDate: new Date().toISOString().split("T")[0],
          itemName: "",
          specification: "",
          quantity: 1,
          unitPrice: 0,
          supplyAmount: 0,
          taxAmount: 0,
          note: "",
        },
      ],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "items",
  });

  const items = watch("items");
  const taxType = watch("taxType");

  // Filtered invoices
  const filteredInvoices = useMemo(() => {
    return mockReceivedInvoices.filter((invoice) => {
      const matchesSearch =
        invoice.invoiceNumber.toLowerCase().includes(searchTerm.toLowerCase()) ||
        invoice.supplierName.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesStatus = !selectedStatus || invoice.status === selectedStatus;
      const matchesDateFrom = !dateFrom || invoice.issueDate >= dateFrom;
      const matchesDateTo = !dateTo || invoice.issueDate <= dateTo;
      return matchesSearch && matchesStatus && matchesDateFrom && matchesDateTo;
    });
  }, [searchTerm, selectedStatus, dateFrom, dateTo]);

  // Calculate totals
  const totalSupply = filteredInvoices.reduce((sum, inv) => sum + inv.supplyAmount, 0);
  const totalTax = filteredInvoices.reduce((sum, inv) => sum + inv.taxAmount, 0);

  // Form item amounts calculation
  const formTotalSupply =
    items?.reduce((sum, item) => sum + (Number(item.supplyAmount) || 0), 0) || 0;
  const formTotalTax = items?.reduce((sum, item) => sum + (Number(item.taxAmount) || 0), 0) || 0;
  const formTotalAmount = formTotalSupply + formTotalTax;

  // Calculate item amounts when quantity or unit price changes
  const calculateItemAmounts = (index: number) => {
    const quantity = Number(items[index]?.quantity) || 0;
    const unitPrice = Number(items[index]?.unitPrice) || 0;
    const supplyAmount = quantity * unitPrice;
    const taxAmount = taxType === "taxable" ? Math.round(supplyAmount * 0.1) : 0;

    setValue(`items.${index}.supplyAmount`, supplyAmount);
    setValue(`items.${index}.taxAmount`, taxAmount);
  };

  // Add new item row
  const addItem = () => {
    append({
      itemDate: new Date().toISOString().split("T")[0],
      itemName: "",
      specification: "",
      quantity: 1,
      unitPrice: 0,
      supplyAmount: 0,
      taxAmount: 0,
      note: "",
    });
  };

  // Select supplier from search
  const selectSupplier = (supplier: (typeof mockSuppliers)[0]) => {
    setValue("supplierBizNo", supplier.bizNo);
    setValue("supplierName", supplier.name);
    setValue("supplierCeoName", supplier.ceoName);
    setValue("supplierAddress", supplier.address);
    setValue("supplierBizType", supplier.bizType);
    setValue("supplierBizItem", supplier.bizItem);
    setShowSupplierSearch(false);
    setSupplierSearchQuery("");
  };

  // Filter suppliers for search modal
  const filteredSuppliers = mockSuppliers.filter(
    (s) =>
      s.name.includes(supplierSearchQuery) ||
      s.bizNo.includes(supplierSearchQuery) ||
      s.ceoName.includes(supplierSearchQuery)
  );

  // Open register modal for new invoice
  const openRegisterModal = () => {
    setIsEditing(false);
    reset({
      supplierBizNo: "",
      supplierName: "",
      supplierCeoName: "",
      supplierAddress: "",
      supplierBizType: "",
      supplierBizItem: "",
      supplierEmail: "",
      invoiceType: "01",
      taxType: "taxable",
      issueDate: new Date().toISOString().split("T")[0],
      note: "",
      items: [
        {
          itemDate: new Date().toISOString().split("T")[0],
          itemName: "",
          specification: "",
          quantity: 1,
          unitPrice: 0,
          supplyAmount: 0,
          taxAmount: 0,
          note: "",
        },
      ],
    });
    setShowRegisterModal(true);
  };

  // Open detail modal
  const openDetailModal = (invoice: (typeof mockReceivedInvoices)[0]) => {
    setSelectedInvoice(invoice);
    setShowDetailModal(true);
  };

  // Open edit modal
  const openEditModal = (invoice: (typeof mockReceivedInvoices)[0]) => {
    setIsEditing(true);
    setSelectedInvoice(invoice);
    reset({
      supplierBizNo: invoice.supplierBusinessNumber,
      supplierName: invoice.supplierName,
      supplierCeoName: invoice.supplierCeoName,
      supplierAddress: invoice.supplierAddress,
      supplierBizType: invoice.supplierBizType,
      supplierBizItem: invoice.supplierBizItem,
      supplierEmail: invoice.supplierEmail,
      invoiceType: "01",
      taxType: "taxable",
      issueDate: invoice.issueDate,
      note: invoice.note,
      items: invoice.items.map((item) => ({
        itemDate: item.itemDate,
        itemName: item.itemName,
        specification: item.specification || "",
        quantity: item.quantity,
        unitPrice: item.unitPrice,
        supplyAmount: item.supplyAmount,
        taxAmount: item.taxAmount,
        note: "",
      })),
    });
    setShowRegisterModal(true);
  };

  // Submit handler
  const onSubmit = async (data: InvoiceFormData) => {
    setIsSubmitting(true);
    try {
      console.log("Invoice data:", data);
      if (isEditing) {
        toast.success("수정 완료", "매입 세금계산서가 수정되었습니다.");
      } else {
        toast.success("등록 완료", "매입 세금계산서가 등록되었습니다.");
      }
      setShowRegisterModal(false);
    } catch {
      toast.error("저장 실패", "세금계산서 저장 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">매입 관리</h1>
          <p className="text-muted-foreground">매입 세금계산서를 등록하고 관리합니다.</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Upload className="h-4 w-4 mr-2" />
            Excel 업로드
          </Button>
          <Button onClick={openRegisterModal}>
            <Plus className="h-4 w-4 mr-2" />
            매입 등록
          </Button>
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
            <p className="text-sm text-muted-foreground">미확인</p>
            <p className="text-2xl font-bold text-warning">
              {filteredInvoices.filter((i) => i.status === "draft").length}건
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
            <div className="flex items-center gap-2">
              <Input
                type="date"
                className="w-[150px]"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                placeholder="시작일"
              />
              <span className="text-muted-foreground">~</span>
              <Input
                type="date"
                className="w-[150px]"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                placeholder="종료일"
              />
            </div>
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
          <CardTitle className="text-lg">매입 세금계산서 목록</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>세금계산서 번호</TableHead>
                <TableHead>발행일</TableHead>
                <TableHead>공급자</TableHead>
                <TableHead>사업자번호</TableHead>
                <TableHead className="text-right">공급가액</TableHead>
                <TableHead className="text-right">세액</TableHead>
                <TableHead className="text-right">합계</TableHead>
                <TableHead>상태</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredInvoices.map((invoice) => (
                <TableRow key={invoice.id}>
                  <TableCell>
                    <button
                      onClick={() => openDetailModal(invoice)}
                      className="font-medium text-primary hover:underline"
                    >
                      {invoice.invoiceNumber}
                    </button>
                  </TableCell>
                  <TableCell>{formatDate(invoice.issueDate)}</TableCell>
                  <TableCell>{invoice.supplierName}</TableCell>
                  <TableCell className="font-mono text-sm">
                    {formatBusinessNumber(invoice.supplierBusinessNumber)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(invoice.supplyAmount, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(invoice.taxAmount, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono font-semibold">
                    {formatCurrency(invoice.totalAmount, { showSymbol: false })}
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
                          <button
                            onClick={() => openDetailModal(invoice)}
                            className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                          >
                            <Eye className="h-4 w-4 mr-2" />
                            상세보기
                          </button>
                          {(invoice.status === "draft" || invoice.status === "issued") && (
                            <button
                              onClick={() => openEditModal(invoice)}
                              className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                            >
                              <Edit className="h-4 w-4 mr-2" />
                              수정
                            </button>
                          )}
                          <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                            <Download className="h-4 w-4 mr-2" />
                            PDF 다운로드
                          </button>
                          {invoice.status === "draft" && (
                            <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted text-destructive">
                              <Trash2 className="h-4 w-4 mr-2" />
                              삭제
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {filteredInvoices.length === 0 && (
                <TableRow>
                  <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                    검색 결과가 없습니다.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Register/Edit Modal */}
      <Modal
        isOpen={showRegisterModal}
        onClose={() => setShowRegisterModal(false)}
        title={isEditing ? "매입 세금계산서 수정" : "매입 세금계산서 등록"}
        size="xl"
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* Invoice Type & Date */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center">
                <Calendar className="h-4 w-4 mr-2" />
                발행 정보
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Controller
                  name="invoiceType"
                  control={control}
                  render={({ field }) => (
                    <Select
                      label="세금계산서 종류"
                      options={invoiceTypeOptions}
                      required
                      error={errors.invoiceType?.message}
                      {...field}
                    />
                  )}
                />
                <Controller
                  name="taxType"
                  control={control}
                  render={({ field }) => (
                    <Select
                      label="과세유형"
                      options={taxTypeOptions}
                      required
                      error={errors.taxType?.message}
                      {...field}
                    />
                  )}
                />
                <Input
                  type="date"
                  label="작성일자"
                  required
                  error={errors.issueDate?.message}
                  {...register("issueDate")}
                />
              </div>
            </CardContent>
          </Card>

          {/* Supplier Info */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-base flex items-center">
                <Building2 className="h-4 w-4 mr-2" />
                공급자 정보
              </CardTitle>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowSupplierSearch(true)}
              >
                <Search className="h-4 w-4 mr-2" />
                거래처 검색
              </Button>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="사업자번호"
                  required
                  error={errors.supplierBizNo?.message}
                  {...register("supplierBizNo")}
                />
                <Input
                  label="상호"
                  required
                  error={errors.supplierName?.message}
                  {...register("supplierName")}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="대표자"
                  required
                  error={errors.supplierCeoName?.message}
                  {...register("supplierCeoName")}
                />
                <Input
                  label="이메일"
                  type="email"
                  error={errors.supplierEmail?.message}
                  {...register("supplierEmail")}
                />
              </div>
              <Input label="사업장 주소" {...register("supplierAddress")} />
              <div className="grid grid-cols-2 gap-4">
                <Input label="업태" {...register("supplierBizType")} />
                <Input label="종목" {...register("supplierBizItem")} />
              </div>
            </CardContent>
          </Card>

          {/* Items */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-base flex items-center">
                <Calculator className="h-4 w-4 mr-2" />
                품목 내역
              </CardTitle>
              <Button type="button" variant="outline" size="sm" onClick={addItem}>
                <Plus className="h-4 w-4 mr-2" />
                품목 추가
              </Button>
            </CardHeader>
            <CardContent>
              {/* Table Header */}
              <div className="grid grid-cols-12 gap-2 px-2 py-2 bg-muted rounded-t-lg font-medium text-sm">
                <div className="col-span-2">일자</div>
                <div className="col-span-2">품목명</div>
                <div className="col-span-1">규격</div>
                <div className="col-span-1 text-right">수량</div>
                <div className="col-span-2 text-right">단가</div>
                <div className="col-span-2 text-right">공급가액</div>
                <div className="col-span-1 text-right">세액</div>
                <div className="col-span-1"></div>
              </div>

              {/* Item Rows */}
              <div className="divide-y max-h-60 overflow-y-auto">
                {fields.map((field, index) => (
                  <div key={field.id} className="grid grid-cols-12 gap-2 px-2 py-2 items-center">
                    <div className="col-span-2">
                      <Input type="date" {...register(`items.${index}.itemDate`)} />
                    </div>
                    <div className="col-span-2">
                      <Input
                        placeholder="품목명"
                        error={errors.items?.[index]?.itemName?.message}
                        {...register(`items.${index}.itemName`)}
                      />
                    </div>
                    <div className="col-span-1">
                      <Input placeholder="규격" {...register(`items.${index}.specification`)} />
                    </div>
                    <div className="col-span-1">
                      <Input
                        type="number"
                        min={1}
                        className="text-right font-mono"
                        {...register(`items.${index}.quantity`, {
                          valueAsNumber: true,
                          onChange: () => calculateItemAmounts(index),
                        })}
                      />
                    </div>
                    <div className="col-span-2">
                      <Input
                        type="number"
                        min={0}
                        className="text-right font-mono"
                        {...register(`items.${index}.unitPrice`, {
                          valueAsNumber: true,
                          onChange: () => calculateItemAmounts(index),
                        })}
                      />
                    </div>
                    <div className="col-span-2">
                      <Input
                        type="number"
                        readOnly
                        className="text-right font-mono bg-muted"
                        {...register(`items.${index}.supplyAmount`, {
                          valueAsNumber: true,
                        })}
                      />
                    </div>
                    <div className="col-span-1">
                      <Input
                        type="number"
                        readOnly
                        className="text-right font-mono bg-muted"
                        {...register(`items.${index}.taxAmount`, {
                          valueAsNumber: true,
                        })}
                      />
                    </div>
                    <div className="col-span-1 flex justify-center">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => remove(index)}
                        disabled={fields.length <= 1}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>

              {/* Totals */}
              <div className="grid grid-cols-12 gap-2 px-2 py-3 bg-muted/50 rounded-b-lg font-semibold border-t">
                <div className="col-span-6 text-right">합계</div>
                <div className="col-span-2 text-right font-mono">
                  {formatCurrency(formTotalSupply, { showSymbol: false })}
                </div>
                <div className="col-span-1 text-right font-mono">
                  {formatCurrency(formTotalTax, { showSymbol: false })}
                </div>
                <div className="col-span-3"></div>
              </div>

              {/* Grand Total */}
              <div className="mt-4 p-3 bg-primary/5 rounded-lg">
                <div className="flex justify-between items-center">
                  <span className="font-medium">총 합계금액</span>
                  <span className="text-xl font-bold text-primary">
                    {formatCurrency(formTotalAmount)}
                  </span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Note */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center">
                <FileText className="h-4 w-4 mr-2" />
                비고
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea placeholder="비고 내용을 입력하세요" {...register("note")} />
            </CardContent>
          </Card>

          {/* Submit Buttons */}
          <div className="flex justify-end space-x-2 pt-4 border-t">
            <Button type="button" variant="outline" onClick={() => setShowRegisterModal(false)}>
              <X className="h-4 w-4 mr-2" />
              취소
            </Button>
            <Button type="submit" isLoading={isSubmitting}>
              <Save className="h-4 w-4 mr-2" />
              {isEditing ? "수정" : "등록"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Supplier Search Modal */}
      <Modal
        isOpen={showSupplierSearch}
        onClose={() => setShowSupplierSearch(false)}
        title="거래처 검색"
      >
        <div className="space-y-4">
          <Input
            placeholder="상호명, 사업자번호, 대표자명으로 검색"
            value={supplierSearchQuery}
            onChange={(e) => setSupplierSearchQuery(e.target.value)}
          />
          <div className="max-h-80 overflow-y-auto space-y-2">
            {filteredSuppliers.length === 0 ? (
              <p className="text-center text-muted-foreground py-8">검색 결과가 없습니다.</p>
            ) : (
              filteredSuppliers.map((supplier) => (
                <div
                  key={supplier.bizNo}
                  className={cn(
                    "p-3 border rounded-lg cursor-pointer hover:bg-accent transition-colors"
                  )}
                  onClick={() => selectSupplier(supplier)}
                >
                  <div className="flex justify-between items-start">
                    <div>
                      <p className="font-medium">{supplier.name}</p>
                      <p className="text-sm text-muted-foreground">
                        {formatBusinessNumber(supplier.bizNo)} | {supplier.ceoName}
                      </p>
                    </div>
                    <Badge variant="outline">{supplier.bizType}</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">{supplier.address}</p>
                </div>
              ))
            )}
          </div>
        </div>
      </Modal>

      {/* Detail Modal */}
      <Modal
        isOpen={showDetailModal}
        onClose={() => setShowDetailModal(false)}
        title="매입 세금계산서 상세"
        size="lg"
      >
        {selectedInvoice && (
          <div className="space-y-6">
            {/* Basic Info */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">세금계산서 번호</p>
                <p className="font-medium">{selectedInvoice.invoiceNumber}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">발행일</p>
                <p className="font-medium">{formatDate(selectedInvoice.issueDate)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">상태</p>
                <Badge variant={statusStyles[selectedInvoice.status].variant}>
                  {statusStyles[selectedInvoice.status].label}
                </Badge>
              </div>
              {selectedInvoice.ntsConfirmNumber && (
                <div>
                  <p className="text-sm text-muted-foreground">국세청 승인번호</p>
                  <p className="font-mono text-sm">{selectedInvoice.ntsConfirmNumber}</p>
                </div>
              )}
            </div>

            {/* Supplier Info */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">공급자 정보</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">상호</p>
                    <p className="font-medium">{selectedInvoice.supplierName}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">사업자번호</p>
                    <p className="font-mono">
                      {formatBusinessNumber(selectedInvoice.supplierBusinessNumber)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">대표자</p>
                    <p>{selectedInvoice.supplierCeoName}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">업태 / 종목</p>
                    <p>
                      {selectedInvoice.supplierBizType} / {selectedInvoice.supplierBizItem}
                    </p>
                  </div>
                  <div className="col-span-2">
                    <p className="text-sm text-muted-foreground">주소</p>
                    <p>{selectedInvoice.supplierAddress}</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Items */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">품목 내역</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>일자</TableHead>
                      <TableHead>품목명</TableHead>
                      <TableHead>규격</TableHead>
                      <TableHead className="text-right">수량</TableHead>
                      <TableHead className="text-right">단가</TableHead>
                      <TableHead className="text-right">공급가액</TableHead>
                      <TableHead className="text-right">세액</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {selectedInvoice.items.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>{formatDate(item.itemDate)}</TableCell>
                        <TableCell>{item.itemName}</TableCell>
                        <TableCell>{item.specification || "-"}</TableCell>
                        <TableCell className="text-right font-mono">{item.quantity}</TableCell>
                        <TableCell className="text-right font-mono">
                          {formatCurrency(item.unitPrice, { showSymbol: false })}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {formatCurrency(item.supplyAmount, { showSymbol: false })}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {formatCurrency(item.taxAmount, { showSymbol: false })}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>

            {/* Totals */}
            <div className="p-4 bg-muted rounded-lg">
              <div className="flex justify-between items-center">
                <span className="font-medium">공급가액</span>
                <span className="font-mono">
                  {formatCurrency(selectedInvoice.supplyAmount, { showSymbol: false })}
                </span>
              </div>
              <div className="flex justify-between items-center mt-2">
                <span className="font-medium">세액</span>
                <span className="font-mono">
                  {formatCurrency(selectedInvoice.taxAmount, { showSymbol: false })}
                </span>
              </div>
              <div className="flex justify-between items-center mt-3 pt-3 border-t">
                <span className="text-lg font-semibold">합계금액</span>
                <span className="text-xl font-bold text-primary">
                  {formatCurrency(selectedInvoice.totalAmount)}
                </span>
              </div>
            </div>

            {/* Note */}
            {selectedInvoice.note && (
              <div>
                <p className="text-sm text-muted-foreground mb-1">비고</p>
                <p className="p-3 bg-muted rounded-lg">{selectedInvoice.note}</p>
              </div>
            )}

            {/* Actions */}
            <div className="flex justify-end space-x-2 pt-4 border-t">
              {(selectedInvoice.status === "draft" || selectedInvoice.status === "issued") && (
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowDetailModal(false);
                    openEditModal(selectedInvoice);
                  }}
                >
                  <Edit className="h-4 w-4 mr-2" />
                  수정
                </Button>
              )}
              <Button variant="outline">
                <Download className="h-4 w-4 mr-2" />
                PDF 다운로드
              </Button>
              <Button onClick={() => setShowDetailModal(false)}>닫기</Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
