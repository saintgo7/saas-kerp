import { useState } from "react";
import { Link } from "react-router-dom";
import {
  Search,
  Filter,
  Download,
  AlertTriangle,
  Package,
  TrendingDown,
  TrendingUp,
  Warehouse,
  ArrowRightLeft,
  History,
  RefreshCw,
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
import { STOCK_MOVEMENT_TYPES } from "@/constants";
import type {
  Stock,
  StockMovement,
  StockAlert,
  Product,
  Warehouse as WarehouseType,
} from "@/types/inventory";

// Mock warehouses
const mockWarehouses: WarehouseType[] = [
  { id: "1", companyId: "1", code: "WH001", name: "본사 창고", isDefault: true, isActive: true, createdAt: "", updatedAt: "" },
  { id: "2", companyId: "1", code: "WH002", name: "공장 창고", isDefault: false, isActive: true, createdAt: "", updatedAt: "" },
  { id: "3", companyId: "1", code: "WH003", name: "지점 창고", isDefault: false, isActive: true, createdAt: "", updatedAt: "" },
];

// Mock products
const mockProducts: Product[] = [
  { id: "1", companyId: "1", code: "PRD-001", name: "노트북 컴퓨터 15인치", unit: "EA", unitPrice: 1500000, costPrice: 1200000, minStock: 5, isActive: true, createdAt: "", updatedAt: "" },
  { id: "2", companyId: "1", code: "PRD-002", name: "무선 마우스", unit: "EA", unitPrice: 45000, costPrice: 28000, minStock: 20, isActive: true, createdAt: "", updatedAt: "" },
  { id: "3", companyId: "1", code: "PRD-003", name: "A4 복사용지", unit: "BOX", unitPrice: 28000, costPrice: 22000, minStock: 50, isActive: true, createdAt: "", updatedAt: "" },
  { id: "4", companyId: "1", code: "PRD-004", name: "볼펜 (파랑)", unit: "EA", unitPrice: 1500, costPrice: 800, minStock: 100, isActive: true, createdAt: "", updatedAt: "" },
  { id: "5", companyId: "1", code: "PRD-005", name: "토너 카트리지", unit: "EA", unitPrice: 85000, costPrice: 62000, minStock: 10, isActive: true, createdAt: "", updatedAt: "" },
];

// Mock stock data
const mockStocks: Stock[] = [
  { id: "1", companyId: "1", productId: "1", product: mockProducts[0], warehouseId: "1", warehouse: mockWarehouses[0], quantity: 25, availableQuantity: 23, reservedQuantity: 2, lastUpdatedAt: "2024-01-15T14:30:00" },
  { id: "2", companyId: "1", productId: "2", product: mockProducts[1], warehouseId: "1", warehouse: mockWarehouses[0], quantity: 150, availableQuantity: 150, reservedQuantity: 0, lastUpdatedAt: "2024-01-15T14:30:00" },
  { id: "3", companyId: "1", productId: "3", product: mockProducts[2], warehouseId: "1", warehouse: mockWarehouses[0], quantity: 35, availableQuantity: 35, reservedQuantity: 0, lastUpdatedAt: "2024-01-15T14:30:00" },
  { id: "4", companyId: "1", productId: "4", product: mockProducts[3], warehouseId: "1", warehouse: mockWarehouses[0], quantity: 85, availableQuantity: 80, reservedQuantity: 5, lastUpdatedAt: "2024-01-15T14:30:00" },
  { id: "5", companyId: "1", productId: "5", product: mockProducts[4], warehouseId: "1", warehouse: mockWarehouses[0], quantity: 8, availableQuantity: 8, reservedQuantity: 0, lastUpdatedAt: "2024-01-15T14:30:00" },
  { id: "6", companyId: "1", productId: "1", product: mockProducts[0], warehouseId: "2", warehouse: mockWarehouses[1], quantity: 15, availableQuantity: 15, reservedQuantity: 0, lastUpdatedAt: "2024-01-15T14:30:00" },
  { id: "7", companyId: "1", productId: "2", product: mockProducts[1], warehouseId: "2", warehouse: mockWarehouses[1], quantity: 80, availableQuantity: 80, reservedQuantity: 0, lastUpdatedAt: "2024-01-15T14:30:00" },
];

// Mock stock movements
const mockMovements: StockMovement[] = [
  { id: "1", companyId: "1", productId: "1", product: mockProducts[0], warehouseId: "1", warehouse: mockWarehouses[0], movementType: "purchase_in", quantity: 10, previousQuantity: 15, currentQuantity: 25, note: "정기 입고", createdBy: "홍길동", createdAt: "2024-01-15T10:30:00" },
  { id: "2", companyId: "1", productId: "2", product: mockProducts[1], warehouseId: "1", warehouse: mockWarehouses[0], movementType: "sales_out", quantity: 20, previousQuantity: 170, currentQuantity: 150, note: "B기업 판매", createdBy: "김철수", createdAt: "2024-01-14T15:20:00" },
  { id: "3", companyId: "1", productId: "3", product: mockProducts[2], warehouseId: "1", warehouse: mockWarehouses[0], movementType: "adjustment_out", quantity: 5, previousQuantity: 40, currentQuantity: 35, note: "재고 조정", createdBy: "이영희", createdAt: "2024-01-13T11:00:00" },
  { id: "4", companyId: "1", productId: "5", product: mockProducts[4], warehouseId: "1", warehouse: mockWarehouses[0], movementType: "sales_out", quantity: 2, previousQuantity: 10, currentQuantity: 8, note: "C회사 판매", createdBy: "박민수", createdAt: "2024-01-12T09:45:00" },
];

// Mock stock alerts
const mockAlerts: StockAlert[] = [
  { id: "1", companyId: "1", productId: "3", product: mockProducts[2], warehouseId: "1", warehouse: mockWarehouses[0], alertType: "low_stock", currentQuantity: 35, threshold: 50, isRead: false, createdAt: "2024-01-15T10:00:00" },
  { id: "2", companyId: "1", productId: "5", product: mockProducts[4], warehouseId: "1", warehouse: mockWarehouses[0], alertType: "low_stock", currentQuantity: 8, threshold: 10, isRead: false, createdAt: "2024-01-14T16:00:00" },
  { id: "3", companyId: "1", productId: "4", product: mockProducts[3], warehouseId: "1", warehouse: mockWarehouses[0], alertType: "low_stock", currentQuantity: 85, threshold: 100, isRead: true, createdAt: "2024-01-13T08:00:00" },
];

// Get movement type label
const getMovementTypeLabel = (type: string) => {
  const found = STOCK_MOVEMENT_TYPES.find((t) => t.value === type);
  return found ? found.label : type;
};

// Get movement type badge variant
const getMovementTypeBadge = (type: string): "default" | "success" | "destructive" | "warning" => {
  if (type.includes("in")) return "success";
  if (type.includes("out")) return "destructive";
  return "default";
};

// Get alert type badge variant
const getAlertBadge = (type: string): "destructive" | "warning" | "default" => {
  switch (type) {
    case "out_of_stock":
      return "destructive";
    case "low_stock":
      return "warning";
    default:
      return "default";
  }
};

// Get alert type label
const getAlertLabel = (type: string) => {
  switch (type) {
    case "out_of_stock":
      return "재고 소진";
    case "low_stock":
      return "안전재고 미달";
    case "overstock":
      return "과잉재고";
    default:
      return type;
  }
};

/**
 * Stock Status Page Component
 * Displays current stock levels, movements, and alerts
 */
export function StockStatusPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedWarehouse, setSelectedWarehouse] = useState<string>("");
  const [showMovementModal, setShowMovementModal] = useState(false);
  const [showAdjustmentModal, setShowAdjustmentModal] = useState(false);
  const [activeTab, setActiveTab] = useState<"stock" | "movements" | "alerts">("stock");

  // Calculate summary statistics
  const totalStockValue = mockStocks.reduce((sum, stock) => {
    return sum + (stock.quantity * (stock.product?.costPrice || 0));
  }, 0);

  const lowStockCount = mockAlerts.filter((a) => !a.isRead && a.alertType === "low_stock").length;
  const totalItems = mockStocks.length;

  // Filter stocks
  const filteredStocks = mockStocks.filter((stock) => {
    const matchesSearch =
      stock.product?.code.toLowerCase().includes(searchTerm.toLowerCase()) ||
      stock.product?.name.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesWarehouse = !selectedWarehouse || stock.warehouseId === selectedWarehouse;
    return matchesSearch && matchesWarehouse;
  });

  // Check if stock is below minimum
  const isBelowMinStock = (stock: Stock) => {
    const minStock = stock.product?.minStock || 0;
    return stock.quantity < minStock;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">재고 현황</h1>
          <p className="text-muted-foreground">현재 재고 수량과 이동 내역을 확인합니다.</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={() => setShowAdjustmentModal(true)}>
            <ArrowRightLeft className="h-4 w-4 mr-2" />
            재고 조정
          </Button>
          <Button variant="outline">
            <RefreshCw className="h-4 w-4 mr-2" />
            새로고침
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-blue-100 rounded-lg">
                <Package className="h-5 w-5 text-blue-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">총 품목수</p>
                <p className="text-2xl font-bold">{totalItems}개</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-green-100 rounded-lg">
                <TrendingUp className="h-5 w-5 text-green-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">총 재고가치</p>
                <p className="text-2xl font-bold">
                  {formatCurrency(totalStockValue, { compact: true })}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-orange-100 rounded-lg">
                <AlertTriangle className="h-5 w-5 text-orange-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">안전재고 미달</p>
                <p className="text-2xl font-bold text-orange-600">{lowStockCount}건</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-purple-100 rounded-lg">
                <Warehouse className="h-5 w-5 text-purple-600" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">창고 수</p>
                <p className="text-2xl font-bold">{mockWarehouses.length}개</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tabs */}
      <div className="border-b">
        <nav className="flex space-x-4">
          <button
            className={`pb-3 px-1 border-b-2 font-medium text-sm ${
              activeTab === "stock"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("stock")}
          >
            <Package className="h-4 w-4 inline mr-2" />
            재고 현황
          </button>
          <button
            className={`pb-3 px-1 border-b-2 font-medium text-sm ${
              activeTab === "movements"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("movements")}
          >
            <History className="h-4 w-4 inline mr-2" />
            이동 내역
          </button>
          <button
            className={`pb-3 px-1 border-b-2 font-medium text-sm relative ${
              activeTab === "alerts"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("alerts")}
          >
            <AlertTriangle className="h-4 w-4 inline mr-2" />
            재고 알림
            {lowStockCount > 0 && (
              <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-destructive text-destructive-foreground text-xs flex items-center justify-center">
                {lowStockCount}
              </span>
            )}
          </button>
        </nav>
      </div>

      {/* Stock Tab */}
      {activeTab === "stock" && (
        <>
          {/* Filters */}
          <Card>
            <CardContent className="p-4">
              <div className="flex flex-col md:flex-row gap-4">
                <div className="flex-1">
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      placeholder="품목코드, 품목명으로 검색..."
                      className="pl-10"
                      value={searchTerm}
                      onChange={(e) => setSearchTerm(e.target.value)}
                    />
                  </div>
                </div>
                <select
                  className="h-9 rounded-md border border-input bg-background px-3 text-sm"
                  value={selectedWarehouse}
                  onChange={(e) => setSelectedWarehouse(e.target.value)}
                >
                  <option value="">전체 창고</option>
                  {mockWarehouses.map((wh) => (
                    <option key={wh.id} value={wh.id}>
                      {wh.name}
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

          {/* Stock Table */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">재고 목록</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>품목코드</TableHead>
                    <TableHead>품목명</TableHead>
                    <TableHead>창고</TableHead>
                    <TableHead className="text-right">현재고</TableHead>
                    <TableHead className="text-right">가용재고</TableHead>
                    <TableHead className="text-right">예약재고</TableHead>
                    <TableHead className="text-right">안전재고</TableHead>
                    <TableHead className="text-right">재고가치</TableHead>
                    <TableHead>상태</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredStocks.map((stock) => (
                    <TableRow key={stock.id}>
                      <TableCell>
                        <Link
                          to={`/inventory/products/${stock.productId}`}
                          className="font-medium text-primary hover:underline"
                        >
                          {stock.product?.code}
                        </Link>
                      </TableCell>
                      <TableCell>{stock.product?.name}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">{stock.warehouse?.name}</Badge>
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {stock.quantity.toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right font-mono text-green-600">
                        {stock.availableQuantity.toLocaleString()}
                      </TableCell>
                      <TableCell className="text-right font-mono text-orange-600">
                        {stock.reservedQuantity > 0 ? stock.reservedQuantity.toLocaleString() : "-"}
                      </TableCell>
                      <TableCell className="text-right font-mono text-muted-foreground">
                        {stock.product?.minStock?.toLocaleString() || "-"}
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {formatCurrency(stock.quantity * (stock.product?.costPrice || 0), { showSymbol: false })}
                      </TableCell>
                      <TableCell>
                        {isBelowMinStock(stock) ? (
                          <Badge variant="warning" className="flex items-center w-fit">
                            <TrendingDown className="h-3 w-3 mr-1" />
                            부족
                          </Badge>
                        ) : (
                          <Badge variant="success">정상</Badge>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {filteredStocks.length === 0 && (
                <div className="text-center py-12">
                  <Package className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                  <h3 className="text-lg font-medium mb-2">재고 데이터가 없습니다</h3>
                  <p className="text-muted-foreground">검색 조건을 변경하거나 재고를 등록해주세요.</p>
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}

      {/* Movements Tab */}
      {activeTab === "movements" && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">재고 이동 내역</CardTitle>
            <Button variant="outline" size="sm" onClick={() => setShowMovementModal(true)}>
              전체 보기
            </Button>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>일시</TableHead>
                  <TableHead>품목</TableHead>
                  <TableHead>창고</TableHead>
                  <TableHead>유형</TableHead>
                  <TableHead className="text-right">수량</TableHead>
                  <TableHead className="text-right">이전재고</TableHead>
                  <TableHead className="text-right">현재재고</TableHead>
                  <TableHead>비고</TableHead>
                  <TableHead>담당자</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mockMovements.map((movement) => (
                  <TableRow key={movement.id}>
                    <TableCell className="text-muted-foreground">
                      {formatDate(movement.createdAt, { format: "time" })}
                    </TableCell>
                    <TableCell>
                      <div>
                        <div className="font-medium">{movement.product?.name}</div>
                        <div className="text-sm text-muted-foreground">
                          {movement.product?.code}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>{movement.warehouse?.name}</TableCell>
                    <TableCell>
                      <Badge variant={getMovementTypeBadge(movement.movementType)}>
                        {getMovementTypeLabel(movement.movementType)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {movement.movementType.includes("in") ? "+" : "-"}
                      {movement.quantity.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground">
                      {movement.previousQuantity.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {movement.currentQuantity.toLocaleString()}
                    </TableCell>
                    <TableCell className="max-w-[150px] truncate">
                      {movement.note || "-"}
                    </TableCell>
                    <TableCell>{movement.createdBy}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Alerts Tab */}
      {activeTab === "alerts" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center">
              <AlertTriangle className="h-5 w-5 mr-2 text-warning" />
              재고 알림
            </CardTitle>
          </CardHeader>
          <CardContent>
            {mockAlerts.length > 0 ? (
              <div className="space-y-3">
                {mockAlerts.map((alert) => (
                  <div
                    key={alert.id}
                    className={`flex items-center justify-between p-4 rounded-lg border ${
                      !alert.isRead ? "bg-warning/5 border-warning/20" : ""
                    }`}
                  >
                    <div className="flex items-center space-x-4">
                      <div
                        className={`p-2 rounded-lg ${
                          alert.alertType === "out_of_stock"
                            ? "bg-destructive/10"
                            : "bg-warning/10"
                        }`}
                      >
                        <AlertTriangle
                          className={`h-5 w-5 ${
                            alert.alertType === "out_of_stock"
                              ? "text-destructive"
                              : "text-warning"
                          }`}
                        />
                      </div>
                      <div>
                        <div className="flex items-center space-x-2">
                          <span className="font-medium">{alert.product?.name}</span>
                          <Badge variant={getAlertBadge(alert.alertType)}>
                            {getAlertLabel(alert.alertType)}
                          </Badge>
                          {!alert.isRead && (
                            <span className="h-2 w-2 rounded-full bg-primary"></span>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">
                          현재재고: {alert.currentQuantity}개 / 안전재고: {alert.threshold}개
                          (부족: {alert.threshold - alert.currentQuantity}개)
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {alert.warehouse?.name} - {formatDate(alert.createdAt, { format: "time" })}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <Link to="/inventory/purchase/new">
                        <Button size="sm">발주하기</Button>
                      </Link>
                      <Button variant="ghost" size="sm">
                        확인
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-12">
                <AlertTriangle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-medium mb-2">알림이 없습니다</h3>
                <p className="text-muted-foreground">모든 재고가 정상 수준입니다.</p>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Movement History Modal */}
      <Modal
        isOpen={showMovementModal}
        onClose={() => setShowMovementModal(false)}
        title="재고 이동 이력"
        size="lg"
      >
        <div className="max-h-[60vh] overflow-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>일시</TableHead>
                <TableHead>품목</TableHead>
                <TableHead>유형</TableHead>
                <TableHead className="text-right">수량</TableHead>
                <TableHead>담당자</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {mockMovements.map((movement) => (
                <TableRow key={movement.id}>
                  <TableCell>{formatDate(movement.createdAt, { format: "time" })}</TableCell>
                  <TableCell>{movement.product?.name}</TableCell>
                  <TableCell>
                    <Badge variant={getMovementTypeBadge(movement.movementType)}>
                      {getMovementTypeLabel(movement.movementType)}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {movement.movementType.includes("in") ? "+" : "-"}
                    {movement.quantity}
                  </TableCell>
                  <TableCell>{movement.createdBy}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </Modal>

      {/* Stock Adjustment Modal */}
      <Modal
        isOpen={showAdjustmentModal}
        onClose={() => setShowAdjustmentModal(false)}
        title="재고 조정"
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            재고 실사 결과를 반영하여 재고를 조정합니다.
          </p>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">품목 선택</label>
              <select className="mt-1.5 w-full h-10 rounded-md border border-input bg-background px-3 text-sm">
                <option value="">품목을 선택하세요</option>
                {mockProducts.map((product) => (
                  <option key={product.id} value={product.id}>
                    {product.code} - {product.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">창고 선택</label>
              <select className="mt-1.5 w-full h-10 rounded-md border border-input bg-background px-3 text-sm">
                <option value="">창고를 선택하세요</option>
                {mockWarehouses.map((wh) => (
                  <option key={wh.id} value={wh.id}>
                    {wh.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm font-medium">현재 재고</label>
                <Input type="number" value="0" disabled className="mt-1.5" />
              </div>
              <div>
                <label className="text-sm font-medium">조정 재고</label>
                <Input type="number" placeholder="0" className="mt-1.5" />
              </div>
            </div>
            <div>
              <label className="text-sm font-medium">조정 사유</label>
              <Input placeholder="조정 사유를 입력하세요" className="mt-1.5" />
            </div>
          </div>
          <div className="flex justify-end space-x-2 pt-4">
            <Button variant="outline" onClick={() => setShowAdjustmentModal(false)}>
              취소
            </Button>
            <Button>조정 적용</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
