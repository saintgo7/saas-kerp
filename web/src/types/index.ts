// Common API Response Types
export interface ApiError {
  code: string;
  message: string;
  /** Free-form extra context (internal/dto/common.go ErrorInfo.detail) */
  detail?: string;
  details?: { field: string; message: string }[];
}

/**
 * internal/dto/common.go Meta. Every handler now emits this one shape; the
 * older flat `{ total, page, page_size }` meta no longer exists.
 */
export interface ApiPagination {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface ApiMeta {
  request_id?: string;
  timestamp?: string;
  pagination?: ApiPagination;
}

/**
 * The single response envelope (internal/dto/common.go Response).
 * Errors live at `error.message`; there is no top-level `message`.
 */
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: ApiError;
  meta?: ApiMeta;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface PaginationParams {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
}

// User & Auth Types
export interface User {
  id: string;
  email: string;
  name: string;
  phone?: string;
  role: UserRole;
  companyId: string;
  company?: Company;
  createdAt: string;
  updatedAt: string;
}

// Mirrors internal/domain/user.go UserRole
export type UserRole = "admin" | "user" | "viewer";

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface RegisterData {
  email: string;
  password: string;
  name: string;
  phone?: string;
  companyName: string;
  businessNumber: string;
}

// Company Types
export interface Company {
  id: string;
  name: string;
  businessNumber: string;
  representativeName: string;
  businessType?: string;
  businessCategory?: string;
  address?: string;
  phone?: string;
  fax?: string;
  email?: string;
  createdAt: string;
  updatedAt: string;
}

// Accounting Types
export interface Account {
  id: string;
  companyId: string;
  code: string;
  name: string;
  nameEn?: string;
  type: AccountType;
  parentId?: string;
  level: number;
  isActive: boolean;
  description?: string;
  // Mirrors internal/dto/account_dto.go AccountResponse
  accountNature?: "debit" | "credit";
  accountCategory?: string;
  isControlAccount?: boolean;
  allowDirectPosting?: boolean;
  sortOrder?: number;
}

export type AccountType = "asset" | "liability" | "equity" | "revenue" | "expense";

export interface Voucher {
  id: string;
  companyId: string;
  voucherNumber: string;
  voucherDate: string;
  description?: string;
  status: VoucherStatus;
  entries: VoucherEntry[];
  createdBy: string;
  approvedBy?: string;
  createdAt: string;
  updatedAt: string;
}

export type VoucherStatus = "draft" | "pending" | "approved" | "rejected";

export interface VoucherEntry {
  id: string;
  voucherId: string;
  accountId: string;
  account?: Account;
  debitAmount: number;
  creditAmount: number;
  description?: string;
  sequence: number;
}

// Invoice Types
export interface TaxInvoice {
  id: string;
  companyId: string;
  invoiceNumber: string;
  issueDate: string;
  supplyDate: string;
  invoiceType: InvoiceType;
  direction: "issued" | "received";
  supplierInfo: BusinessInfo;
  buyerInfo: BusinessInfo;
  items: TaxInvoiceItem[];
  supplyAmount: number;
  taxAmount: number;
  totalAmount: number;
  status: InvoiceStatus;
  ntsConfirmNumber?: string;
  createdAt: string;
  updatedAt: string;
}

export type InvoiceType = "taxInvoice" | "simplified" | "receipt";
export type InvoiceStatus = "draft" | "issued" | "sent" | "cancelled";

export interface BusinessInfo {
  businessNumber: string;
  companyName: string;
  representativeName: string;
  address?: string;
  businessType?: string;
  businessCategory?: string;
  email?: string;
}

export interface TaxInvoiceItem {
  id: string;
  invoiceId: string;
  sequence: number;
  itemDate: string;
  itemName: string;
  specification?: string;
  quantity: number;
  unitPrice: number;
  supplyAmount: number;
  taxAmount: number;
  note?: string;
}

// HR Types
export interface Employee {
  id: string;
  companyId: string;
  employeeNumber: string;
  name: string;
  residentNumber: string;
  joinDate: string;
  resignDate?: string;
  departmentId?: string;
  department?: Department;
  positionId?: string;
  position?: Position;
  email?: string;
  phone?: string;
  address?: string;
  bankName?: string;
  bankAccount?: string;
  status: EmployeeStatus;
  createdAt: string;
  updatedAt: string;
}

export type EmployeeStatus = "active" | "leave" | "resigned";

export interface Department {
  id: string;
  companyId: string;
  name: string;
  code: string;
  parentId?: string;
  managerId?: string;
}

export interface Position {
  id: string;
  companyId: string;
  name: string;
  level: number;
}

export interface Payroll {
  id: string;
  companyId: string;
  employeeId: string;
  employee?: Employee;
  paymentYear: number;
  paymentMonth: number;
  baseSalary: number;
  overtimePay: number;
  bonus: number;
  allowances: number;
  grossPay: number;
  nationalPension: number;
  healthInsurance: number;
  employmentInsurance: number;
  longTermCareInsurance: number;
  incomeTax: number;
  localIncomeTax: number;
  totalDeductions: number;
  netPay: number;
  paymentDate?: string;
  status: PayrollStatus;
}

export type PayrollStatus = "draft" | "calculated" | "approved" | "paid";

// Dashboard Types
export interface DashboardStats {
  revenue: {
    current: number;
    previous: number;
    change: number;
  };
  expenses: {
    current: number;
    previous: number;
    change: number;
  };
  profit: {
    current: number;
    previous: number;
    change: number;
  };
  invoicesPending: number;
  payrollPending: number;
}

export interface ChartData {
  labels: string[];
  datasets: {
    label: string;
    data: number[];
    backgroundColor?: string;
    borderColor?: string;
  }[];
}

// Partner Types
export interface Partner {
  id: string;
  companyId: string;
  code: string;
  name: string;
  nameEn?: string;
  businessNumber?: string;
  representativeName?: string;
  partnerType: PartnerType;
  address?: string;
  addressDetail?: string;
  zipCode?: string;
  phone?: string;
  fax?: string;
  email?: string;
  website?: string;
  paymentTermDays?: number;
  creditLimit?: number;
  arAccountId?: string;
  apAccountId?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// Mirrors internal/dto/partner_dto.go (oneof=customer vendor both).
// The backend has no "supplier" value.
export type PartnerType = "customer" | "vendor" | "both";

// Menu & Navigation Types
export interface MenuItem {
  id: string;
  label: string;
  icon?: string;
  path?: string;
  children?: MenuItem[];
  roles?: UserRole[];
  badge?: number | string;
}

// Form Types
export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

// Table Types
export interface TableColumn<T> {
  key: keyof T | string;
  header: string;
  width?: string;
  align?: "left" | "center" | "right";
  sortable?: boolean;
  render?: (value: unknown, row: T) => React.ReactNode;
}

// Re-export Inventory Types
export * from "./inventory";
