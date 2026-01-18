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
  Package,
  Tag,
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
import { formatCurrency } from "@/lib/utils";
import { PRODUCT_UNITS } from "@/constants";
import type { Product, ProductCategory } from "@/types/inventory";

// Mock categories data
const mockCategories: ProductCategory[] = [
  { id: "1", companyId: "1", code: "CAT001", name: "전자제품", level: 1, isActive: true, createdAt: "", updatedAt: "" },
  { id: "2", companyId: "1", code: "CAT002", name: "사무용품", level: 1, isActive: true, createdAt: "", updatedAt: "" },
  { id: "3", companyId: "1", code: "CAT003", name: "소모품", level: 1, isActive: true, createdAt: "", updatedAt: "" },
  { id: "4", companyId: "1", code: "CAT004", name: "원재료", level: 1, isActive: true, createdAt: "", updatedAt: "" },
];

// Mock products data
const mockProducts: Product[] = [
  {
    id: "1",
    companyId: "1",
    code: "PRD-001",
    name: "노트북 컴퓨터 15인치",
    categoryId: "1",
    category: mockCategories[0],
    specification: "Intel i7, 16GB RAM, 512GB SSD",
    unit: "EA",
    unitPrice: 1500000,
    costPrice: 1200000,
    minStock: 5,
    maxStock: 50,
    isActive: true,
    barcode: "8801234567890",
    createdAt: "2024-01-10T09:00:00",
    updatedAt: "2024-01-15T14:30:00",
  },
  {
    id: "2",
    companyId: "1",
    code: "PRD-002",
    name: "무선 마우스",
    categoryId: "1",
    category: mockCategories[0],
    specification: "Bluetooth 5.0, 충전식",
    unit: "EA",
    unitPrice: 45000,
    costPrice: 28000,
    minStock: 20,
    maxStock: 200,
    isActive: true,
    barcode: "8801234567891",
    createdAt: "2024-01-10T09:00:00",
    updatedAt: "2024-01-15T14:30:00",
  },
  {
    id: "3",
    companyId: "1",
    code: "PRD-003",
    name: "A4 복사용지",
    categoryId: "2",
    category: mockCategories[1],
    specification: "80g/m2, 500매",
    unit: "BOX",
    unitPrice: 28000,
    costPrice: 22000,
    minStock: 50,
    maxStock: 500,
    isActive: true,
    barcode: "8801234567892",
    createdAt: "2024-01-10T09:00:00",
    updatedAt: "2024-01-15T14:30:00",
  },
  {
    id: "4",
    companyId: "1",
    code: "PRD-004",
    name: "볼펜 (파랑)",
    categoryId: "2",
    category: mockCategories[1],
    specification: "0.5mm",
    unit: "EA",
    unitPrice: 1500,
    costPrice: 800,
    minStock: 100,
    maxStock: 1000,
    isActive: true,
    barcode: "8801234567893",
    createdAt: "2024-01-10T09:00:00",
    updatedAt: "2024-01-15T14:30:00",
  },
  {
    id: "5",
    companyId: "1",
    code: "PRD-005",
    name: "토너 카트리지",
    categoryId: "3",
    category: mockCategories[2],
    specification: "HP LaserJet Pro 호환",
    unit: "EA",
    unitPrice: 85000,
    costPrice: 62000,
    minStock: 10,
    maxStock: 50,
    isActive: true,
    barcode: "8801234567894",
    createdAt: "2024-01-10T09:00:00",
    updatedAt: "2024-01-15T14:30:00",
  },
  {
    id: "6",
    companyId: "1",
    code: "PRD-006",
    name: "알루미늄 프레임",
    categoryId: "4",
    category: mockCategories[3],
    specification: "1000mm x 500mm",
    unit: "EA",
    unitPrice: 35000,
    costPrice: 25000,
    minStock: 30,
    maxStock: 200,
    isActive: false,
    barcode: "8801234567895",
    createdAt: "2024-01-10T09:00:00",
    updatedAt: "2024-01-15T14:30:00",
  },
];

/**
 * Product List Page Component
 * Displays all products with search, filter, and management capabilities
 */
export function ProductListPage() {
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [selectedRows, setSelectedRows] = useState<string[]>([]);
  const [showCategoryModal, setShowCategoryModal] = useState(false);

  // Filter products based on search and filters
  const filteredProducts = mockProducts.filter((product) => {
    const matchesSearch =
      product.code.toLowerCase().includes(searchTerm.toLowerCase()) ||
      product.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      product.specification?.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = !selectedCategory || product.categoryId === selectedCategory;
    const matchesStatus =
      !selectedStatus ||
      (selectedStatus === "active" ? product.isActive : !product.isActive);
    return matchesSearch && matchesCategory && matchesStatus;
  });

  // Get unit label from constant
  const getUnitLabel = (unit: string) => {
    const found = PRODUCT_UNITS.find((u) => u.value === unit);
    return found ? found.label : unit;
  };

  // Toggle row selection
  const toggleRowSelection = (id: string) => {
    setSelectedRows((prev) =>
      prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]
    );
  };

  // Toggle all rows selection
  const toggleAllRows = () => {
    if (selectedRows.length === filteredProducts.length) {
      setSelectedRows([]);
    } else {
      setSelectedRows(filteredProducts.map((p) => p.id));
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">품목 관리</h1>
          <p className="text-muted-foreground">상품 및 품목을 조회하고 관리합니다.</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={() => setShowCategoryModal(true)}>
            <Tag className="h-4 w-4 mr-2" />
            카테고리 관리
          </Button>
          <Link to="/inventory/products/new">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              품목 등록
            </Button>
          </Link>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="품목코드, 품목명, 규격으로 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
            >
              <option value="">전체 카테고리</option>
              {mockCategories.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.name}
                </option>
              ))}
            </select>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedStatus}
              onChange={(e) => setSelectedStatus(e.target.value)}
            >
              <option value="">전체 상태</option>
              <option value="active">사용중</option>
              <option value="inactive">미사용</option>
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
                  일괄 수정
                </Button>
                <Button variant="outline" size="sm" className="text-destructive">
                  <Trash2 className="h-4 w-4 mr-2" />
                  일괄 삭제
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Product Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center">
            <Package className="h-5 w-5 mr-2" />
            품목 목록
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">
                  <input
                    type="checkbox"
                    className="rounded border-input"
                    checked={selectedRows.length === filteredProducts.length && filteredProducts.length > 0}
                    onChange={toggleAllRows}
                  />
                </TableHead>
                <TableHead>품목코드</TableHead>
                <TableHead>품목명</TableHead>
                <TableHead>카테고리</TableHead>
                <TableHead>규격</TableHead>
                <TableHead>단위</TableHead>
                <TableHead className="text-right">판매단가</TableHead>
                <TableHead className="text-right">매입단가</TableHead>
                <TableHead>상태</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredProducts.map((product) => (
                <TableRow key={product.id}>
                  <TableCell>
                    <input
                      type="checkbox"
                      className="rounded border-input"
                      checked={selectedRows.includes(product.id)}
                      onChange={() => toggleRowSelection(product.id)}
                    />
                  </TableCell>
                  <TableCell>
                    <Link
                      to={`/inventory/products/${product.id}`}
                      className="font-medium text-primary hover:underline"
                    >
                      {product.code}
                    </Link>
                  </TableCell>
                  <TableCell className="font-medium">{product.name}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{product.category?.name}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground max-w-[150px] truncate">
                    {product.specification || "-"}
                  </TableCell>
                  <TableCell>{getUnitLabel(product.unit)}</TableCell>
                  <TableCell className="text-right font-mono">
                    {formatCurrency(product.unitPrice, { showSymbol: false })}
                  </TableCell>
                  <TableCell className="text-right font-mono text-muted-foreground">
                    {formatCurrency(product.costPrice, { showSymbol: false })}
                  </TableCell>
                  <TableCell>
                    <Badge variant={product.isActive ? "success" : "secondary"}>
                      {product.isActive ? "사용중" : "미사용"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="relative group">
                      <Button variant="ghost" size="icon">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                      <div className="absolute right-0 hidden group-hover:block z-10">
                        <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[120px]">
                          <Link
                            to={`/inventory/products/${product.id}`}
                            className="flex items-center px-3 py-2 text-sm hover:bg-muted"
                          >
                            <Eye className="h-4 w-4 mr-2" />
                            상세보기
                          </Link>
                          <Link
                            to={`/inventory/products/${product.id}/edit`}
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

          {/* Empty State */}
          {filteredProducts.length === 0 && (
            <div className="text-center py-12">
              <Package className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium mb-2">등록된 품목이 없습니다</h3>
              <p className="text-muted-foreground mb-4">
                새로운 품목을 등록하여 재고를 관리해보세요.
              </p>
              <Link to="/inventory/products/new">
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  품목 등록
                </Button>
              </Link>
            </div>
          )}

          {/* Pagination */}
          {filteredProducts.length > 0 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t">
              <p className="text-sm text-muted-foreground">
                총 {filteredProducts.length}건
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
          )}
        </CardContent>
      </Card>

      {/* Category Management Modal */}
      <Modal
        isOpen={showCategoryModal}
        onClose={() => setShowCategoryModal(false)}
        title="카테고리 관리"
      >
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <p className="text-sm text-muted-foreground">
              품목 분류를 위한 카테고리를 관리합니다.
            </p>
            <Button size="sm">
              <Plus className="h-4 w-4 mr-2" />
              추가
            </Button>
          </div>
          <div className="border rounded-lg divide-y">
            {mockCategories.map((category) => (
              <div
                key={category.id}
                className="flex items-center justify-between p-3 hover:bg-muted/50"
              >
                <div>
                  <span className="font-medium">{category.name}</span>
                  <span className="text-muted-foreground ml-2 text-sm">
                    ({category.code})
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <Button variant="ghost" size="icon">
                    <Edit className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" className="text-destructive">
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </Modal>
    </div>
  );
}
