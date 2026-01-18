import { apiClient } from "./client";

// ========== Sales Report Types ==========

export interface SalesTrendItem {
  date: string;
  amount: number;
  count: number;
}

export interface PartnerSalesItem {
  partnerId: string;
  partnerName: string;
  businessNumber: string;
  totalAmount: number;
  transactionCount: number;
  percentage: number;
}

export interface ProductSalesItem {
  productId: string;
  productCode: string;
  productName: string;
  category: string;
  quantity: number;
  totalAmount: number;
  percentage: number;
}

export interface SalesReportData {
  period: {
    startDate: string;
    endDate: string;
  };
  summary: {
    totalSales: number;
    previousTotalSales: number;
    transactionCount: number;
    previousTransactionCount: number;
    averageTransaction: number;
    previousAverageTransaction: number;
  };
  trend: SalesTrendItem[];
  byPartner: PartnerSalesItem[];
  byProduct: ProductSalesItem[];
}

export interface SalesReportParams {
  startDate: string;
  endDate: string;
  groupBy?: "day" | "week" | "month" | "quarter" | "year";
  partnerId?: string;
  productId?: string;
  [key: string]: string | number | boolean | undefined;
}

// ========== Expense Report Types ==========

export interface ExpenseByAccount {
  accountId: string;
  accountCode: string;
  accountName: string;
  currentAmount: number;
  budgetAmount: number;
  previousAmount: number;
  variance: number;
  percentage: number;
}

export interface ExpenseByDepartment {
  departmentId: string;
  departmentName: string;
  currentAmount: number;
  budgetAmount: number;
  employeeCount: number;
  perCapita: number;
}

export interface ExpenseTrendItem {
  date: string;
  amount: number;
  budgetAmount: number;
}

export interface ExpenseReportData {
  period: {
    startDate: string;
    endDate: string;
  };
  summary: {
    totalExpense: number;
    previousTotalExpense: number;
    totalBudget: number;
    budgetUtilization: number;
    varianceFromBudget: number;
  };
  trend: ExpenseTrendItem[];
  byAccount: ExpenseByAccount[];
  byDepartment: ExpenseByDepartment[];
}

export interface ExpenseReportParams {
  startDate: string;
  endDate: string;
  groupBy?: "day" | "week" | "month" | "quarter" | "year";
  departmentId?: string;
  accountId?: string;
  includeBudget?: boolean;
  [key: string]: string | number | boolean | undefined;
}

// ========== HR Report Types ==========

export interface DepartmentHeadcount {
  departmentId: string;
  departmentName: string;
  headcount: number;
  regularCount: number;
  contractCount: number;
  partTimeCount: number;
}

export interface PositionHeadcount {
  positionId: string;
  positionName: string;
  headcount: number;
  averageTenure: number;
}

export interface HiringTrendItem {
  date: string;
  hired: number;
  resigned: number;
  netChange: number;
}

export interface PayrollStatistics {
  totalPayroll: number;
  averageSalary: number;
  medianSalary: number;
  minSalary: number;
  maxSalary: number;
}

export interface AttendanceStatistics {
  workingDays: number;
  averageAttendance: number;
  lateCount: number;
  earlyLeaveCount: number;
  absenceCount: number;
  overtimeHours: number;
}

export interface HRReportData {
  asOfDate: string;
  headcount: {
    total: number;
    regular: number;
    contract: number;
    partTime: number;
    newHires: number;
    resignations: number;
    turnoverRate: number;
  };
  byDepartment: DepartmentHeadcount[];
  byPosition: PositionHeadcount[];
  hiringTrend: HiringTrendItem[];
  payroll: PayrollStatistics;
  attendance: AttendanceStatistics;
}

export interface HRReportParams {
  startDate: string;
  endDate: string;
  departmentId?: string;
  [key: string]: string | number | boolean | undefined;
}

// ========== Custom Report Types ==========

export interface ReportTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  columns: ReportColumn[];
  filters: ReportFilter[];
  defaultSort?: string;
  defaultGroupBy?: string;
}

export interface ReportColumn {
  field: string;
  header: string;
  type: "string" | "number" | "currency" | "date" | "percentage";
  width?: number;
  sortable?: boolean;
  aggregatable?: boolean;
}

export interface ReportFilter {
  field: string;
  label: string;
  type: "text" | "date" | "select" | "multiselect" | "daterange";
  options?: { value: string; label: string }[];
  required?: boolean;
}

export interface CustomReportParams {
  templateId: string;
  filters: Record<string, unknown>;
  sort?: {
    field: string;
    direction: "asc" | "desc";
  };
  groupBy?: string;
  page?: number;
  pageSize?: number;
}

export interface CustomReportData {
  templateId: string;
  templateName: string;
  generatedAt: string;
  columns: ReportColumn[];
  rows: Record<string, unknown>[];
  totals?: Record<string, number>;
  pagination: {
    page: number;
    pageSize: number;
    totalCount: number;
    totalPages: number;
  };
}

// ========== API Functions ==========

export const reportsApi = {
  // Sales Report
  getSalesReport: (params: SalesReportParams) =>
    apiClient.get<SalesReportData>("/reports/sales", params),

  exportSalesReport: (params: SalesReportParams & { format: "pdf" | "xlsx" }) =>
    apiClient.get<{ downloadUrl: string }>("/reports/sales/export", params),

  // Expense Report
  getExpenseReport: (params: ExpenseReportParams) =>
    apiClient.get<ExpenseReportData>("/reports/expense", params),

  exportExpenseReport: (params: ExpenseReportParams & { format: "pdf" | "xlsx" }) =>
    apiClient.get<{ downloadUrl: string }>("/reports/expense/export", params),

  // HR Report
  getHRReport: (params: HRReportParams) =>
    apiClient.get<HRReportData>("/reports/hr", params),

  exportHRReport: (params: HRReportParams & { format: "pdf" | "xlsx" }) =>
    apiClient.get<{ downloadUrl: string }>("/reports/hr/export", params),

  // Custom Report
  getTemplates: () =>
    apiClient.get<ReportTemplate[]>("/reports/templates"),

  getTemplate: (templateId: string) =>
    apiClient.get<ReportTemplate>(`/reports/templates/${templateId}`),

  generateCustomReport: (params: CustomReportParams) =>
    apiClient.post<CustomReportData>("/reports/custom", params),

  exportCustomReport: (params: CustomReportParams & { format: "pdf" | "xlsx" }) =>
    apiClient.post<{ downloadUrl: string }>("/reports/custom/export", params),
};
