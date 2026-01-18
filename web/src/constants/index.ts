import type { MenuItem, SelectOption, UserRole } from "@/types";

// API Configuration
export const API_BASE_URL = import.meta.env.VITE_API_URL || "/api";

// Pagination Defaults
export const DEFAULT_PAGE_SIZE = 20;
export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100];

// Date Formats
export const DATE_FORMAT = "YYYY-MM-DD";
export const DATETIME_FORMAT = "YYYY-MM-DD HH:mm:ss";
export const MONTH_FORMAT = "YYYY-MM";

// User Roles
export const USER_ROLES: Record<UserRole, string> = {
  admin: "관리자",
  manager: "매니저",
  accountant: "회계담당",
  hr: "인사담당",
  user: "일반사용자",
};

// Account Types (계정 유형)
export const ACCOUNT_TYPES: SelectOption[] = [
  { value: "asset", label: "자산" },
  { value: "liability", label: "부채" },
  { value: "equity", label: "자본" },
  { value: "revenue", label: "수익" },
  { value: "expense", label: "비용" },
];

// Voucher Status (전표 상태)
export const VOUCHER_STATUS: SelectOption[] = [
  { value: "draft", label: "작성중" },
  { value: "pending", label: "승인대기" },
  { value: "approved", label: "승인완료" },
  { value: "rejected", label: "반려" },
];

// Invoice Types (세금계산서 유형)
export const INVOICE_TYPES: SelectOption[] = [
  { value: "taxInvoice", label: "세금계산서" },
  { value: "simplified", label: "간이세금계산서" },
  { value: "receipt", label: "현금영수증" },
];

// Invoice Status (세금계산서 상태)
export const INVOICE_STATUS: SelectOption[] = [
  { value: "draft", label: "작성중" },
  { value: "issued", label: "발행" },
  { value: "sent", label: "전송완료" },
  { value: "cancelled", label: "취소" },
];

// Employee Status (직원 상태)
export const EMPLOYEE_STATUS: SelectOption[] = [
  { value: "active", label: "재직" },
  { value: "leave", label: "휴직" },
  { value: "resigned", label: "퇴직" },
];

// Partner Types (거래처 유형)
export const PARTNER_TYPES: SelectOption[] = [
  { value: "customer", label: "고객" },
  { value: "supplier", label: "거래처" },
  { value: "both", label: "고객/거래처" },
];

// Payroll Status (급여 상태)
export const PAYROLL_STATUS: SelectOption[] = [
  { value: "draft", label: "작성중" },
  { value: "calculated", label: "계산완료" },
  { value: "approved", label: "승인완료" },
  { value: "paid", label: "지급완료" },
];

// Banks (은행)
export const BANKS: SelectOption[] = [
  { value: "KB", label: "국민은행" },
  { value: "SHINHAN", label: "신한은행" },
  { value: "WOORI", label: "우리은행" },
  { value: "HANA", label: "하나은행" },
  { value: "NH", label: "농협은행" },
  { value: "IBK", label: "기업은행" },
  { value: "SC", label: "SC제일은행" },
  { value: "CITI", label: "씨티은행" },
  { value: "KAKAO", label: "카카오뱅크" },
  { value: "TOSS", label: "토스뱅크" },
  { value: "KBANK", label: "케이뱅크" },
  { value: "SUHYUP", label: "수협은행" },
  { value: "BUSAN", label: "부산은행" },
  { value: "DGB", label: "대구은행" },
  { value: "GWANGJU", label: "광주은행" },
  { value: "JEJU", label: "제주은행" },
  { value: "JEONBUK", label: "전북은행" },
  { value: "KYUNGNAM", label: "경남은행" },
  { value: "SAEMAUL", label: "새마을금고" },
  { value: "POSTBANK", label: "우체국" },
];

// Main Navigation Menu
export const MAIN_MENU: MenuItem[] = [
  {
    id: "dashboard",
    label: "대시보드",
    icon: "LayoutDashboard",
    path: "/dashboard",
  },
  {
    id: "accounting",
    label: "회계관리",
    icon: "Calculator",
    children: [
      { id: "voucher", label: "전표관리", path: "/accounting/voucher" },
      { id: "ledger", label: "원장조회", path: "/accounting/ledger" },
      { id: "trial-balance", label: "시산표", path: "/accounting/trial-balance" },
      { id: "financial-statements", label: "재무제표", path: "/accounting/financial-statements" },
      { id: "accounts", label: "계정과목관리", path: "/accounting/accounts" },
    ],
  },
  {
    id: "invoice",
    label: "세금계산서",
    icon: "FileText",
    children: [
      { id: "invoice-issue", label: "매출발행", path: "/invoice/issue" },
      { id: "invoice-received", label: "매입관리", path: "/invoice/received" },
      { id: "invoice-list", label: "발행내역", path: "/invoice/list" },
      { id: "invoice-hometax", label: "홈택스연동", path: "/invoice/hometax", roles: ["admin", "accountant"] },
    ],
  },
  {
    id: "hr",
    label: "인사/급여",
    icon: "Users",
    children: [
      { id: "employee", label: "직원관리", path: "/hr/employee" },
      { id: "department", label: "부서관리", path: "/hr/department" },
      { id: "payroll", label: "급여관리", path: "/hr/payroll" },
      { id: "insurance", label: "4대보험", path: "/hr/insurance", roles: ["admin", "hr"] },
      { id: "attendance", label: "근태관리", path: "/hr/attendance" },
    ],
  },
  {
    id: "partners",
    label: "거래처관리",
    icon: "Building2",
    path: "/partners",
  },
  {
    id: "inventory",
    label: "재고관리",
    icon: "Package",
    children: [
      { id: "products", label: "품목관리", path: "/inventory/products" },
      { id: "stock", label: "재고현황", path: "/inventory/stock" },
      { id: "purchase", label: "구매관리", path: "/inventory/purchase" },
      { id: "sales", label: "판매관리", path: "/inventory/sales" },
    ],
  },
  {
    id: "reports",
    label: "보고서",
    icon: "BarChart3",
    children: [
      { id: "sales-report", label: "매출분석", path: "/reports/sales" },
      { id: "expense-report", label: "비용분석", path: "/reports/expense" },
      { id: "hr-report", label: "인사현황", path: "/reports/hr" },
      { id: "custom-report", label: "맞춤보고서", path: "/reports/custom" },
    ],
  },
  {
    id: "settings",
    label: "설정",
    icon: "Settings",
    children: [
      { id: "company", label: "회사정보", path: "/settings/company" },
      { id: "users", label: "사용자관리", path: "/settings/users", roles: ["admin"] },
      { id: "permissions", label: "권한관리", path: "/settings/permissions", roles: ["admin"] },
      { id: "integrations", label: "연동설정", path: "/settings/integrations", roles: ["admin"] },
    ],
  },
];

// Validation Messages
export const VALIDATION_MESSAGES = {
  required: "필수 입력 항목입니다.",
  email: "올바른 이메일 형식이 아닙니다.",
  phone: "올바른 전화번호 형식이 아닙니다.",
  businessNumber: "올바른 사업자등록번호 형식이 아닙니다.",
  minLength: (min: number) => `최소 ${min}자 이상 입력해주세요.`,
  maxLength: (max: number) => `최대 ${max}자까지 입력 가능합니다.`,
  min: (min: number) => `최소 ${min} 이상이어야 합니다.`,
  max: (max: number) => `최대 ${max} 이하이어야 합니다.`,
  pattern: "올바른 형식이 아닙니다.",
  passwordMatch: "비밀번호가 일치하지 않습니다.",
  balanceMismatch: "차변과 대변의 합계가 일치하지 않습니다.",
};

// Toast Duration
export const TOAST_DURATION = {
  short: 2000,
  normal: 3000,
  long: 5000,
};

// Local Storage Keys
export const STORAGE_KEYS = {
  accessToken: "kerp_access_token",
  refreshToken: "kerp_refresh_token",
  user: "kerp_user",
  theme: "kerp_theme",
  sidebarCollapsed: "kerp_sidebar_collapsed",
  recentSearches: "kerp_recent_searches",
};
