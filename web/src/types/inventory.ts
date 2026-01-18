// Inventory Module Types

// Product / Item (품목)
export interface Product {
  id: string;
  companyId: string;
  code: string;
  name: string;
  categoryId?: string;
  category?: ProductCategory;
  specification?: string;
  unit: string;
  unitPrice: number;
  costPrice: number;
  minStock?: number;
  maxStock?: number;
  isActive: boolean;
  description?: string;
  barcode?: string;
  imageUrl?: string;
  createdAt: string;
  updatedAt: string;
}

// Product Category (품목 카테고리)
export interface ProductCategory {
  id: string;
  companyId: string;
  code: string;
  name: string;
  parentId?: string;
  level: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// Warehouse (창고)
export interface Warehouse {
  id: string;
  companyId: string;
  code: string;
  name: string;
  address?: string;
  manager?: string;
  phone?: string;
  isDefault: boolean;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// Stock (재고)
export interface Stock {
  id: string;
  companyId: string;
  productId: string;
  product?: Product;
  warehouseId: string;
  warehouse?: Warehouse;
  quantity: number;
  availableQuantity: number;
  reservedQuantity: number;
  lastUpdatedAt: string;
}

// Stock Movement (재고 이동)
export interface StockMovement {
  id: string;
  companyId: string;
  productId: string;
  product?: Product;
  warehouseId: string;
  warehouse?: Warehouse;
  movementType: StockMovementType;
  quantity: number;
  previousQuantity: number;
  currentQuantity: number;
  referenceType?: string;
  referenceId?: string;
  note?: string;
  createdBy: string;
  createdAt: string;
}

export type StockMovementType =
  | "purchase_in"
  | "sales_out"
  | "adjustment_in"
  | "adjustment_out"
  | "transfer_in"
  | "transfer_out"
  | "return_in"
  | "return_out";

// Purchase Order (발주서)
export interface PurchaseOrder {
  id: string;
  companyId: string;
  orderNumber: string;
  orderDate: string;
  expectedDate?: string;
  supplierId: string;
  supplier?: import("./index").Partner;
  warehouseId: string;
  warehouse?: Warehouse;
  items: PurchaseOrderItem[];
  totalAmount: number;
  taxAmount: number;
  grandTotal: number;
  status: PurchaseOrderStatus;
  note?: string;
  createdBy: string;
  approvedBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PurchaseOrderItem {
  id: string;
  purchaseOrderId: string;
  productId: string;
  product?: Product;
  quantity: number;
  unitPrice: number;
  amount: number;
  taxAmount: number;
  receivedQuantity: number;
  note?: string;
}

export type PurchaseOrderStatus =
  | "draft"
  | "pending"
  | "approved"
  | "ordered"
  | "partial"
  | "completed"
  | "cancelled";

// Sales Order (수주)
export interface SalesOrder {
  id: string;
  companyId: string;
  orderNumber: string;
  orderDate: string;
  expectedDate?: string;
  customerId: string;
  customer?: import("./index").Partner;
  warehouseId: string;
  warehouse?: Warehouse;
  items: SalesOrderItem[];
  totalAmount: number;
  taxAmount: number;
  grandTotal: number;
  status: SalesOrderStatus;
  note?: string;
  createdBy: string;
  approvedBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SalesOrderItem {
  id: string;
  salesOrderId: string;
  productId: string;
  product?: Product;
  quantity: number;
  unitPrice: number;
  amount: number;
  taxAmount: number;
  shippedQuantity: number;
  note?: string;
}

export type SalesOrderStatus =
  | "draft"
  | "pending"
  | "approved"
  | "confirmed"
  | "partial"
  | "completed"
  | "cancelled";

// Stock Alert (재고 알림)
export interface StockAlert {
  id: string;
  companyId: string;
  productId: string;
  product?: Product;
  warehouseId: string;
  warehouse?: Warehouse;
  alertType: StockAlertType;
  currentQuantity: number;
  threshold: number;
  isRead: boolean;
  createdAt: string;
}

export type StockAlertType = "low_stock" | "out_of_stock" | "overstock";
