import { useState } from "react";
import { useForm, useFieldArray, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Plus,
  Search,
  Filter,
  Download,
  MoreHorizontal,
  Eye,
  Edit,
  Trash2,
  ShoppingBag,
  Truck,
  CheckCircle,
  XCircle,
  Save,
  Package,
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
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  Modal,
} from "@/components/ui";
import { formatCurrency, formatDate } from "@/lib/utils";
import { SALES_ORDER_STATUS } from "@/constants";
import type { SalesOrder, SalesOrderStatus, Product, Warehouse } from "@/types/inventory";
import type { Partner } from "@/types";

// Mock data
const mockCustomers: Partner[] = [
  { id: "1", companyId: "1", code: "CUS001", name: "(주)가나다 기업", partnerType: "customer", isActive: true, createdAt: "", updatedAt: "" },
  { id: "2", companyId: "1", code: "CUS002", name: "마바사 상사", partnerType: "customer", isActive: true, createdAt: "", updatedAt: "" },
  { id: "3", companyId: "1", code: "CUS003", name: "아자차 유통", partnerType: "customer", isActive: true, createdAt: "", updatedAt: "" },
];

const mockWarehouses: Warehouse[] = [
  { id: "1", companyId: "1", code: "WH001", name: "본사 창고", isDefault: true, isActive: true, createdAt: "", updatedAt: "" },
  { id: "2", companyId: "1", code: "WH002", name: "공장 창고", isDefault: false, isActive: true, createdAt: "", updatedAt: "" },
];

const mockProducts: Product[] = [
  { id: "1", companyId: "1", code: "PRD-001", name: "노트북 컴퓨터 15인치", unit: "EA", unitPrice: 1500000, costPrice: 1200000, isActive: true, createdAt: "", updatedAt: "" },
  { id: "2", companyId: "1", code: "PRD-002", name: "무선 마우스", unit: "EA", unitPrice: 45000, costPrice: 28000, isActive: true, createdAt: "", updatedAt: "" },
  { id: "3", companyId: "1", code: "PRD-003", name: "A4 복사용지", unit: "BOX", unitPrice: 28000, costPrice: 22000, isActive: true, createdAt: "", updatedAt: "" },
  { id: "4", companyId: "1", code: "PRD-004", name: "볼펜 (파랑)", unit: "EA", unitPrice: 1500, costPrice: 800, isActive: true, createdAt: "", updatedAt: "" },
  { id: "5", companyId: "1", code: "PRD-005", name: "토너 카트리지", unit: "EA", unitPrice: 85000, costPrice: 62000, isActive: true, createdAt: "", updatedAt: "" },
];

const mockSalesOrders: SalesOrder[] = [
  {
    id: "1",
    companyId: "1",
    orderNumber: "SO-2024-0020",
    orderDate: "2024-01-15",
    expectedDate: "2024-01-18",
    customerId: "1",
    customer: mockCustomers[0],
    warehouseId: "1",
    warehouse: mockWarehouses[0],
    items: [
      { id: "1", salesOrderId: "1", productId: "1", product: mockProducts[0], quantity: 5, unitPrice: 1500000, amount: 7500000, taxAmount: 750000, shippedQuantity: 5, note: "" },
      { id: "2", salesOrderId: "1", productId: "2", product: mockProducts[1], quantity: 20, unitPrice: 45000, amount: 900000, taxAmount: 90000, shippedQuantity: 20, note: "" },
    ],
    totalAmount: 8400000,
    taxAmount: 840000,
    grandTotal: 9240000,
    status: "completed",
    createdBy: "홍길동",
    createdAt: "2024-01-15T09:00:00",
    updatedAt: "2024-01-18T14:30:00",
  },
  {
    id: "2",
    companyId: "1",
    orderNumber: "SO-2024-0019",
    orderDate: "2024-01-14",
    expectedDate: "2024-01-17",
    customerId: "2",
    customer: mockCustomers[1],
    warehouseId: "1",
    warehouse: mockWarehouses[0],
    items: [
      { id: "3", salesOrderId: "2", productId: "3", product: mockProducts[2], quantity: 50, unitPrice: 28000, amount: 1400000, taxAmount: 140000, shippedQuantity: 30, note: "" },
      { id: "4", salesOrderId: "2", productId: "4", product: mockProducts[3], quantity: 200, unitPrice: 1500, amount: 300000, taxAmount: 30000, shippedQuantity: 100, note: "" },
    ],
    totalAmount: 1700000,
    taxAmount: 170000,
    grandTotal: 1870000,
    status: "partial",
    createdBy: "김철수",
    createdAt: "2024-01-14T10:00:00",
    updatedAt: "2024-01-16T11:00:00",
  },
  {
    id: "3",
    companyId: "1",
    orderNumber: "SO-2024-0018",
    orderDate: "2024-01-13",
    expectedDate: "2024-01-16",
    customerId: "3",
    customer: mockCustomers[2],
    warehouseId: "1",
    warehouse: mockWarehouses[0],
    items: [
      { id: "5", salesOrderId: "3", productId: "5", product: mockProducts[4], quantity: 10, unitPrice: 85000, amount: 850000, taxAmount: 85000, shippedQuantity: 0, note: "" },
    ],
    totalAmount: 850000,
    taxAmount: 85000,
    grandTotal: 935000,
    status: "confirmed",
    createdBy: "이영희",
    createdAt: "2024-01-13T11:00:00",
    updatedAt: "2024-01-13T15:00:00",
  },
  {
    id: "4",
    companyId: "1",
    orderNumber: "SO-2024-0017",
    orderDate: "2024-01-12",
    customerId: "1",
    customer: mockCustomers[0],
    warehouseId: "2",
    warehouse: mockWarehouses[1],
    items: [
      { id: "6", salesOrderId: "4", productId: "1", product: mockProducts[0], quantity: 3, unitPrice: 1500000, amount: 4500000, taxAmount: 450000, shippedQuantity: 0, note: "" },
    ],
    totalAmount: 4500000,
    taxAmount: 450000,
    grandTotal: 4950000,
    status: "pending",
    createdBy: "박민수",
    createdAt: "2024-01-12T14:00:00",
    updatedAt: "2024-01-12T14:00:00",
  },
  {
    id: "5",
    companyId: "1",
    orderNumber: "SO-2024-0016",
    orderDate: "2024-01-11",
    customerId: "2",
    customer: mockCustomers[1],
    warehouseId: "1",
    warehouse: mockWarehouses[0],
    items: [
      { id: "7", salesOrderId: "5", productId: "2", product: mockProducts[1], quantity: 50, unitPrice: 45000, amount: 2250000, taxAmount: 225000, shippedQuantity: 0, note: "" },
    ],
    totalAmount: 2250000,
    taxAmount: 225000,
    grandTotal: 2475000,
    status: "cancelled",
    createdBy: "최지현",
    createdAt: "2024-01-11T09:00:00",
    updatedAt: "2024-01-12T10:00:00",
  },
];

// Status styles
const statusStyles: Record<SalesOrderStatus, { variant: "default" | "secondary" | "destructive" | "success" | "warning"; icon: React.ReactNode }> = {
  draft: { variant: "secondary", icon: <Edit className="h-3 w-3" /> },
  pending: { variant: "warning", icon: <ShoppingBag className="h-3 w-3" /> },
  approved: { variant: "default", icon: <CheckCircle className="h-3 w-3" /> },
  confirmed: { variant: "default", icon: <FileText className="h-3 w-3" /> },
  partial: { variant: "warning", icon: <Package className="h-3 w-3" /> },
  completed: { variant: "success", icon: <CheckCircle className="h-3 w-3" /> },
  cancelled: { variant: "destructive", icon: <XCircle className="h-3 w-3" /> },
};

// Get status label
const getStatusLabel = (status: SalesOrderStatus) => {
  const found = SALES_ORDER_STATUS.find((s) => s.value === status);
  return found ? found.label : status;
};

// Validation schema for new sales order
const salesOrderSchema = z.object({
  customerId: z.string().min(1, "고객을 선택하세요"),
  warehouseId: z.string().min(1, "창고를 선택하세요"),
  orderDate: z.string().min(1, "수주일자를 선택하세요"),
  expectedDate: z.string().optional(),
  note: z.string().optional(),
  items: z.array(
    z.object({
      productId: z.string().min(1, "품목을 선택하세요"),
      quantity: z.number().min(1, "수량은 1 이상이어야 합니다"),
      unitPrice: z.number().min(0, "단가는 0 이상이어야 합니다"),
    })
  ).min(1, "최소 1개 이상의 품목을 추가하세요"),
});

type SalesOrderFormData = z.infer<typeof salesOrderSchema>;

/**
 * Sales Order Page Component
 * Manages sales orders with list and form views
 */
export function SalesOrderPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [selectedCustomer, setSelectedCustomer] = useState<string>("");
  const [showFormModal, setShowFormModal] = useState(false);
  const [showShipModal, setShowShipModal] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<SalesOrder | null>(null);

  // Filter sales orders
  const filteredOrders = mockSalesOrders.filter((order) => {
    const matchesSearch =
      order.orderNumber.toLowerCase().includes(searchTerm.toLowerCase()) ||
      order.customer?.name.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = !selectedStatus || order.status === selectedStatus;
    const matchesCustomer = !selectedCustomer || order.customerId === selectedCustomer;
    return matchesSearch && matchesStatus && matchesCustomer;
  });

  // Calculate summary
  const pendingShipCount = mockSalesOrders.filter((o) => o.status === "confirmed" || o.status === "partial").length;
  const completedAmount = mockSalesOrders
    .filter((o) => o.status === "completed")
    .reduce((sum, o) => sum + o.grandTotal, 0);
  const totalAmount = mockSalesOrders
    .filter((o) => o.status !== "cancelled")
    .reduce((sum, o) => sum + o.grandTotal, 0);

  // Form setup
  const {
    register,
    control,
    handleSubmit,
    watch,
    reset,
    formState: { errors },
  } = useForm<SalesOrderFormData>({
    resolver: zodResolver(salesOrderSchema),
    defaultValues: {
      customerId: "",
      warehouseId: "1",
      orderDate: new Date().toISOString().split("T")[0],
      expectedDate: "",
      note: "",
      items: [{ productId: "", quantity: 1, unitPrice: 0 }],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "items",
  });

  const watchItems = watch("items");

  // Calculate totals
  const calculateTotals = () => {
    const totalAmount = watchItems.reduce((sum, item) => {
      return sum + (item.quantity || 0) * (item.unitPrice || 0);
    }, 0);
    const taxAmount = totalAmount * 0.1;
    return { totalAmount, taxAmount, grandTotal: totalAmount + taxAmount };
  };

  const { totalAmount: formTotalAmount, taxAmount: formTaxAmount, grandTotal: formGrandTotal } = calculateTotals();

  // Handle form submission
  const onSubmit = (data: SalesOrderFormData) => {
    console.log("Sales order data:", data);
    setShowFormModal(false);
    reset();
  };

  // Open ship modal
  const openShipModal = (order: SalesOrder) => {
    setSelectedOrder(order);
    setShowShipModal(true);
  };

  // Convert data to select options
  const customerOptions = mockCustomers.map((c) => ({ value: c.id, label: c.name }));
  const warehouseOptions = mockWarehouses.map((w) => ({ value: w.id, label: w.name }));
  const productOptions = mockProducts.map((p) => ({ value: p.id, label: `${p.code} - ${p.name}` }));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">판매 관리</h1>
          <p className="text-muted-foreground">수주를 등록하고 출고를 처리합니다.</p>
        </div>
        <Button onClick={() => setShowFormModal(true)}>
          <Plus className="h-4 w-4 mr-2" />
          수주 등록
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-blue-100 rounded-lg">
                <ShoppingBag className="h-5 w-5 text-blue-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">전체 수주</p>
                <p className="text-2xl font-bold">{mockSalesOrders.length}건</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-orange-100 rounded-lg">
                <Truck className="h-5 w-5 text-orange-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">출고 대기</p>
                <p className="text-2xl font-bold text-orange-600">{pendingShipCount}건</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-green-100 rounded-lg">
                <CheckCircle className="h-5 w-5 text-green-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">완료 금액</p>
                <p className="text-2xl font-bold text-green-600">
                  {formatCurrency(completedAmount, { compact: true })}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-purple-100 rounded-lg">
                <FileText className="h-5 w-5 text-purple-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">수주 총액</p>
                <p className="text-2xl font-bold">
                  {formatCurrency(totalAmount, { compact: true })}
                </p>
              </div>
            </div>
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
                  placeholder="수주번호, 고객명으로 검색..."
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
              {SALES_ORDER_STATUS.map((status) => (
                <option key={status.value} value={status.value}>
                  {status.label}
                </option>
              ))}
            </select>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedCustomer}
              onChange={(e) => setSelectedCustomer(e.target.value)}
            >
              <option value="">전체 고객</option>
              {mockCustomers.map((customer) => (
                <option key={customer.id} value={customer.id}>
                  {customer.name}
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

      {/* Sales Orders Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">수주 목록</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>수주번호</TableHead>
                <TableHead>수주일자</TableHead>
                <TableHead>고객</TableHead>
                <TableHead>출고창고</TableHead>
                <TableHead className="text-right">공급가액</TableHead>
                <TableHead className="text-right">세액</TableHead>
                <TableHead className="text-right">합계</TableHead>
                <TableHead>상태</TableHead>
                <TableHead>담당자</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredOrders.map((order) => (
                <TableRow key={order.id} className={order.status === "cancelled" ? "opacity-50" : ""}>
                  <TableCell>
                    <span className="font-medium text-primary">{order.orderNumber}</span>
                  </TableCell>
                  <TableCell>{formatDate(order.orderDate)}</TableCell>
                  <TableCell>{order.customer?.name}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{order.warehouse?.name}</Badge>
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(order.totalAmount, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono text-muted-foreground">
                    {formatCurrency(order.taxAmount, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono font-semibold">
                    {formatCurrency(order.grandTotal, { showSymbol: false })}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={statusStyles[order.status].variant}
                      className="flex items-center w-fit space-x-1"
                    >
                      {statusStyles[order.status].icon}
                      <span>{getStatusLabel(order.status)}</span>
                    </Badge>
                  </TableCell>
                  <TableCell>{order.createdBy}</TableCell>
                  <TableCell>
                    <div className="relative group">
                      <Button variant="ghost" size="icon">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                      <div className="absolute right-0 hidden group-hover:block z-10">
                        <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[130px]">
                          <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                            <Eye className="h-4 w-4 mr-2" />
                            상세보기
                          </button>
                          {(order.status === "confirmed" || order.status === "partial") && (
                            <button
                              className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted text-primary"
                              onClick={() => openShipModal(order)}
                            >
                              <Truck className="h-4 w-4 mr-2" />
                              출고처리
                            </button>
                          )}
                          {order.status === "completed" && (
                            <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                              <FileText className="h-4 w-4 mr-2" />
                              세금계산서
                            </button>
                          )}
                          {order.status === "draft" && (
                            <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                              <Edit className="h-4 w-4 mr-2" />
                              수정
                            </button>
                          )}
                          {(order.status === "draft" || order.status === "pending") && (
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
            </TableBody>
          </Table>

          {filteredOrders.length === 0 && (
            <div className="text-center py-12">
              <ShoppingBag className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium mb-2">수주가 없습니다</h3>
              <p className="text-muted-foreground mb-4">새로운 수주를 등록해보세요.</p>
              <Button onClick={() => setShowFormModal(true)}>
                <Plus className="h-4 w-4 mr-2" />
                수주 등록
              </Button>
            </div>
          )}

          {filteredOrders.length > 0 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t">
              <p className="text-sm text-muted-foreground">총 {filteredOrders.length}건</p>
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
          )}
        </CardContent>
      </Card>

      {/* New Sales Order Modal */}
      <Modal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          reset();
        }}
        title="수주 등록"
        size="lg"
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Controller
              name="customerId"
              control={control}
              render={({ field }) => (
                <Select
                  label="고객"
                  options={customerOptions}
                  placeholder="고객 선택"
                  required
                  error={errors.customerId?.message}
                  {...field}
                />
              )}
            />
            <Controller
              name="warehouseId"
              control={control}
              render={({ field }) => (
                <Select
                  label="출고 창고"
                  options={warehouseOptions}
                  placeholder="창고 선택"
                  required
                  error={errors.warehouseId?.message}
                  {...field}
                />
              )}
            />
            <Input
              type="date"
              label="수주일자"
              required
              error={errors.orderDate?.message}
              {...register("orderDate")}
            />
            <Input
              type="date"
              label="출고예정일"
              {...register("expectedDate")}
            />
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">품목 목록</label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => append({ productId: "", quantity: 1, unitPrice: 0 })}
              >
                <Plus className="h-4 w-4 mr-1" />
                품목 추가
              </Button>
            </div>

            <div className="border rounded-lg divide-y max-h-[300px] overflow-auto">
              {fields.map((field, index) => (
                <div key={field.id} className="p-3 space-y-2">
                  <div className="flex items-end gap-2">
                    <div className="flex-1">
                      <Controller
                        name={`items.${index}.productId`}
                        control={control}
                        render={({ field }) => (
                          <Select
                            options={productOptions}
                            placeholder="품목 선택"
                            error={errors.items?.[index]?.productId?.message}
                            {...field}
                            onChange={(value) => {
                              field.onChange(value);
                              const product = mockProducts.find((p) => p.id === value);
                              if (product) {
                                // Auto-fill unit price from product
                              }
                            }}
                          />
                        )}
                      />
                    </div>
                    <div className="w-24">
                      <Input
                        type="number"
                        placeholder="수량"
                        min={1}
                        error={errors.items?.[index]?.quantity?.message}
                        {...register(`items.${index}.quantity`, { valueAsNumber: true })}
                      />
                    </div>
                    <div className="w-32">
                      <Input
                        type="number"
                        placeholder="단가"
                        min={0}
                        error={errors.items?.[index]?.unitPrice?.message}
                        {...register(`items.${index}.unitPrice`, { valueAsNumber: true })}
                      />
                    </div>
                    <div className="w-32 text-right font-mono text-sm pt-2">
                      {formatCurrency((watchItems[index]?.quantity || 0) * (watchItems[index]?.unitPrice || 0), { showSymbol: false })}
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="text-destructive"
                      onClick={() => remove(index)}
                      disabled={fields.length === 1}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>

            {errors.items?.root && (
              <p className="text-sm text-destructive">{errors.items.root.message}</p>
            )}
          </div>

          <div className="bg-muted/50 rounded-lg p-4 space-y-2">
            <div className="flex justify-between text-sm">
              <span>공급가액</span>
              <span className="font-mono">{formatCurrency(formTotalAmount, { showSymbol: false })}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span>부가세 (10%)</span>
              <span className="font-mono">{formatCurrency(formTaxAmount, { showSymbol: false })}</span>
            </div>
            <div className="flex justify-between font-semibold pt-2 border-t">
              <span>합계</span>
              <span className="font-mono">{formatCurrency(formGrandTotal)}</span>
            </div>
          </div>

          <Textarea
            label="비고"
            placeholder="추가 메모사항을 입력하세요"
            rows={2}
            {...register("note")}
          />

          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setShowFormModal(false);
                reset();
              }}
            >
              취소
            </Button>
            <Button type="submit">
              <Save className="h-4 w-4 mr-2" />
              수주 저장
            </Button>
          </div>
        </form>
      </Modal>

      {/* Ship Modal */}
      <Modal
        isOpen={showShipModal}
        onClose={() => {
          setShowShipModal(false);
          setSelectedOrder(null);
        }}
        title="출고 처리"
        size="lg"
      >
        {selectedOrder && (
          <div className="space-y-4">
            <div className="bg-muted/50 rounded-lg p-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">수주번호:</span>
                  <span className="ml-2 font-medium">{selectedOrder.orderNumber}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">고객:</span>
                  <span className="ml-2 font-medium">{selectedOrder.customer?.name}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">수주일:</span>
                  <span className="ml-2">{formatDate(selectedOrder.orderDate)}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">출고예정일:</span>
                  <span className="ml-2">{selectedOrder.expectedDate ? formatDate(selectedOrder.expectedDate) : "-"}</span>
                </div>
              </div>
            </div>

            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>품목</TableHead>
                  <TableHead className="text-right">수주수량</TableHead>
                  <TableHead className="text-right">기출고</TableHead>
                  <TableHead className="text-right">미출고</TableHead>
                  <TableHead className="text-right">출고수량</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {selectedOrder.items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <div>
                        <div className="font-medium">{item.product?.name}</div>
                        <div className="text-sm text-muted-foreground">{item.product?.code}</div>
                      </div>
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {item.quantity.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right font-mono text-green-600">
                      {item.shippedQuantity.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right font-mono text-orange-600">
                      {(item.quantity - item.shippedQuantity).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right w-32">
                      <Input
                        type="number"
                        min={0}
                        max={item.quantity - item.shippedQuantity}
                        defaultValue={item.quantity - item.shippedQuantity}
                        className="text-right"
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>

            <div className="flex justify-end space-x-2 pt-4">
              <Button
                variant="outline"
                onClick={() => {
                  setShowShipModal(false);
                  setSelectedOrder(null);
                }}
              >
                취소
              </Button>
              <Button>
                <Truck className="h-4 w-4 mr-2" />
                출고 처리
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
