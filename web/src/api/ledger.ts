import { apiClient } from "./client";
import type { Account, AccountType, ApiResponse } from "@/types";

/**
 * Period parameters for ledger queries
 */
export interface PeriodParams {
  startDate: string;
  endDate: string;
  [key: string]: string | number | boolean | undefined;
}

/**
 * General ledger entry
 */
export interface GeneralLedgerEntry {
  date: string;
  voucherId: string;
  voucherNumber: string;
  description: string;
  debitAmount: number;
  creditAmount: number;
  balance: number;
  entryDescription?: string;
}

/**
 * General ledger response for an account
 */
export interface GeneralLedgerData {
  account: Account;
  openingBalance: number;
  entries: GeneralLedgerEntry[];
  totalDebit: number;
  totalCredit: number;
  closingBalance: number;
}

/**
 * Subsidiary ledger entry (with partner info)
 */
export interface SubsidiaryLedgerEntry extends GeneralLedgerEntry {
  partnerId?: string;
  partnerName?: string;
  partnerCode?: string;
}

/**
 * Subsidiary ledger response
 */
export interface SubsidiaryLedgerData {
  account: Account;
  partnerId?: string;
  partnerName?: string;
  openingBalance: number;
  entries: SubsidiaryLedgerEntry[];
  totalDebit: number;
  totalCredit: number;
  closingBalance: number;
}

/**
 * Trial balance account row
 */
export interface TrialBalanceRow {
  accountId: string;
  accountCode: string;
  accountName: string;
  accountType: AccountType;
  level: number;
  parentId?: string;
  openingDebit: number;
  openingCredit: number;
  periodDebit: number;
  periodCredit: number;
  closingDebit: number;
  closingCredit: number;
  isLeaf: boolean;
}

/**
 * Trial balance response
 */
export interface TrialBalanceData {
  period: PeriodParams;
  rows: TrialBalanceRow[];
  totals: {
    openingDebit: number;
    openingCredit: number;
    periodDebit: number;
    periodCredit: number;
    closingDebit: number;
    closingCredit: number;
  };
}

/**
 * Balance sheet section
 */
export interface BalanceSheetSection {
  title: string;
  accounts: {
    accountId: string;
    accountCode: string;
    accountName: string;
    level: number;
    currentAmount: number;
    previousAmount?: number;
    isSubtotal?: boolean;
  }[];
  total: number;
  previousTotal?: number;
}

/**
 * Balance sheet data
 */
export interface BalanceSheetData {
  asOfDate: string;
  comparisonDate?: string;
  assets: BalanceSheetSection[];
  liabilities: BalanceSheetSection[];
  equity: BalanceSheetSection[];
  totalAssets: number;
  totalLiabilitiesAndEquity: number;
  previousTotalAssets?: number;
  previousTotalLiabilitiesAndEquity?: number;
}

/**
 * Income statement section
 */
export interface IncomeStatementSection {
  title: string;
  accounts: {
    accountId: string;
    accountCode: string;
    accountName: string;
    level: number;
    currentAmount: number;
    previousAmount?: number;
    budget?: number;
    variance?: number;
    isSubtotal?: boolean;
  }[];
  total: number;
  previousTotal?: number;
  budgetTotal?: number;
}

/**
 * Income statement data
 */
export interface IncomeStatementData {
  period: PeriodParams;
  comparisonPeriod?: PeriodParams;
  revenue: IncomeStatementSection;
  costOfSales?: IncomeStatementSection;
  grossProfit: number;
  previousGrossProfit?: number;
  operatingExpenses: IncomeStatementSection;
  operatingIncome: number;
  previousOperatingIncome?: number;
  nonOperatingIncome?: IncomeStatementSection;
  nonOperatingExpenses?: IncomeStatementSection;
  incomeBeforeTax: number;
  previousIncomeBeforeTax?: number;
  incomeTax?: number;
  netIncome: number;
  previousNetIncome?: number;
}

/**
 * Account balance summary
 */
export interface AccountBalanceSummary {
  accountId: string;
  accountCode: string;
  accountName: string;
  accountType: AccountType;
  balance: number;
  debitSum: number;
  creditSum: number;
}

/**
 * Raised when a screen asks for something the Go backend does not expose.
 * Better a named, catchable failure than a silent 404 rendered as empty data.
 */
export class EndpointNotImplementedError extends Error {
  constructor(what: string) {
    super(`${what} 기능은 서버에 아직 구현되지 않았습니다.`);
    this.name = "EndpointNotImplementedError";
  }
}

// ---------------------------------------------------------------------------
// Wire types (internal/dto/ledger_dto.go)
// ---------------------------------------------------------------------------

interface AccountLedgerEntryWire {
  voucher_id: string;
  voucher_no: string;
  voucher_date: string;
  voucher_type: string;
  entry_id: string;
  line_no: number;
  description?: string;
  debit_amount: number;
  credit_amount: number;
  balance: number;
  partner_id?: string;
  partner_name?: string;
}

interface AccountLedgerWire {
  account_id: string;
  account_code: string;
  account_name: string;
  from_date: string;
  to_date: string;
  opening_balance: number;
  total_debit: number;
  total_credit: number;
  closing_balance: number;
  entries: AccountLedgerEntryWire[];
}

interface TrialBalanceItemWire {
  account_id: string;
  account_code: string;
  account_name: string;
  account_type: AccountType;
  account_level: number;
  opening_debit: number;
  opening_credit: number;
  period_debit: number;
  period_credit: number;
  closing_debit: number;
  closing_credit: number;
  is_sub_total: boolean;
  is_total: boolean;
}

interface TrialBalanceWire {
  fiscal_year: number;
  fiscal_month: number;
  start_date: string;
  end_date: string;
  items: TrialBalanceItemWire[];
  total_debit: number;
  total_credit: number;
  is_balanced: boolean;
}

interface FinancialStatementItemWire {
  code: string;
  name: string;
  amount: number;
  level: number;
  is_sub_total: boolean;
  is_total: boolean;
}

interface BalanceSheetWire {
  as_of_date: string;
  assets: FinancialStatementItemWire[];
  liabilities: FinancialStatementItemWire[];
  equity: FinancialStatementItemWire[];
  total_assets: number;
  total_liabilities: number;
  total_equity: number;
  is_balanced: boolean;
}

interface IncomeStatementWire {
  from_date: string;
  to_date: string;
  revenue: FinancialStatementItemWire[];
  expenses: FinancialStatementItemWire[];
  total_revenue: number;
  total_expenses: number;
  net_income: number;
}

interface LedgerBalanceWire {
  account_id: string;
  account_code: string;
  account_name: string;
  account_type: AccountType;
  period_debit: number;
  period_credit: number;
  closing_balance: number;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * The report endpoints are keyed by fiscal year/month, not by a free date
 * range (dto.PeriodRequest requires `year` and `month`). Derive them from the
 * period end so existing screens keep their date-range pickers.
 */
function toYearMonth(date: string): { year: number; month: number } {
  const parsed = new Date(date);
  if (Number.isNaN(parsed.getTime())) {
    const now = new Date();
    return { year: now.getFullYear(), month: now.getMonth() + 1 };
  }
  return { year: parsed.getFullYear(), month: parsed.getMonth() + 1 };
}

function toStatementSection(
  title: string,
  items: FinancialStatementItemWire[],
  total: number
): BalanceSheetSection {
  return {
    title,
    accounts: (items ?? []).map((item) => ({
      accountId: item.code,
      accountCode: item.code,
      accountName: item.name,
      level: item.level,
      currentAmount: item.amount,
      isSubtotal: item.is_sub_total || item.is_total,
    })),
    total,
  };
}

function toLedgerEntry(entry: AccountLedgerEntryWire): GeneralLedgerEntry {
  return {
    date: entry.voucher_date,
    voucherId: entry.voucher_id,
    voucherNumber: entry.voucher_no,
    description: entry.description ?? "",
    debitAmount: entry.debit_amount,
    creditAmount: entry.credit_amount,
    balance: entry.balance,
    entryDescription: entry.description,
  };
}

function toGeneralLedger(wire: AccountLedgerWire): GeneralLedgerData {
  return {
    account: {
      id: wire.account_id,
      companyId: "",
      code: wire.account_code,
      name: wire.account_name,
      type: "asset",
      level: 0,
      isActive: true,
    },
    openingBalance: wire.opening_balance,
    entries: (wire.entries ?? []).map(toLedgerEntry),
    totalDebit: wire.total_debit,
    totalCredit: wire.total_credit,
    closingBalance: wire.closing_balance,
  };
}

function toTrialBalance(wire: TrialBalanceWire): TrialBalanceData {
  const rows: TrialBalanceRow[] = (wire.items ?? []).map((item) => ({
    accountId: item.account_id,
    accountCode: item.account_code,
    accountName: item.account_name,
    accountType: item.account_type,
    level: item.account_level,
    openingDebit: item.opening_debit,
    openingCredit: item.opening_credit,
    periodDebit: item.period_debit,
    periodCredit: item.period_credit,
    closingDebit: item.closing_debit,
    closingCredit: item.closing_credit,
    isLeaf: !item.is_sub_total && !item.is_total,
  }));

  const sum = (pick: (row: TrialBalanceRow) => number) =>
    rows.reduce((acc, row) => acc + pick(row), 0);

  return {
    period: { startDate: wire.start_date, endDate: wire.end_date },
    rows,
    totals: {
      openingDebit: sum((r) => r.openingDebit),
      openingCredit: sum((r) => r.openingCredit),
      periodDebit: wire.total_debit ?? sum((r) => r.periodDebit),
      periodCredit: wire.total_credit ?? sum((r) => r.periodCredit),
      closingDebit: sum((r) => r.closingDebit),
      closingCredit: sum((r) => r.closingCredit),
    },
  };
}

function toBalanceSheet(wire: BalanceSheetWire): BalanceSheetData {
  return {
    asOfDate: wire.as_of_date,
    assets: [toStatementSection("자산", wire.assets, wire.total_assets)],
    liabilities: [
      toStatementSection("부채", wire.liabilities, wire.total_liabilities),
    ],
    equity: [toStatementSection("자본", wire.equity, wire.total_equity)],
    totalAssets: wire.total_assets,
    totalLiabilitiesAndEquity: wire.total_liabilities + wire.total_equity,
  };
}

function toIncomeStatement(wire: IncomeStatementWire): IncomeStatementData {
  const revenue = toStatementSection(
    "수익",
    wire.revenue,
    wire.total_revenue
  ) as IncomeStatementSection;
  const operatingExpenses = toStatementSection(
    "비용",
    wire.expenses,
    wire.total_expenses
  ) as IncomeStatementSection;

  return {
    period: { startDate: wire.from_date, endDate: wire.to_date },
    revenue,
    grossProfit: wire.total_revenue,
    operatingExpenses,
    operatingIncome: wire.net_income,
    incomeBeforeTax: wire.net_income,
    netIncome: wire.net_income,
  };
}

/**
 * Ledger API client.
 *
 * Backed by internal/handler/ledger_handler.go:
 *   GET /ledger/balances            (year, month)
 *   GET /ledger/account             (account_id, from_date, to_date)
 *   GET /reports/trial-balance      (year, month)
 *   GET /reports/balance-sheet      (year, month)
 *   GET /reports/income-statement   (from_year, from_month, to_year, to_month)
 *
 * The previous paths (/ledger/general/:id, /ledger/subsidiary/..., 
 * /ledger/trial-balance, /ledger/account-balances and every /export route)
 * are not registered by the backend at all.
 */
export const ledgerApi = {
  /** GET /ledger/account */
  generalLedger: async (accountId: string, params: PeriodParams) => {
    const response = await apiClient.get<AccountLedgerWire>("/ledger/account", {
      account_id: accountId,
      from_date: params.startDate,
      to_date: params.endDate,
    });
    return { ...response, data: toGeneralLedger(response.data) };
  },

  /** Not available: the backend serves one account at a time. */
  generalLedgerAll: async (
    _params?: PeriodParams & { accountType?: AccountType }
  ): Promise<ApiResponse<GeneralLedgerData[]>> => {
    throw new EndpointNotImplementedError("전체 계정 총계정원장 조회");
  },

  /** Not available: no subsidiary-ledger endpoint exists. */
  subsidiaryLedger: async (
    _accountId?: string,
    _params?: PeriodParams & { partnerId?: string }
  ): Promise<ApiResponse<SubsidiaryLedgerData>> => {
    throw new EndpointNotImplementedError("거래처별 보조원장 조회");
  },

  /** Not available: no subsidiary-ledger endpoint exists. */
  subsidiaryLedgerByPartner: async (
    _accountId?: string,
    _params?: PeriodParams
  ): Promise<ApiResponse<SubsidiaryLedgerData[]>> => {
    throw new EndpointNotImplementedError("거래처별 보조원장 조회");
  },

  /** GET /reports/trial-balance */
  trialBalance: async (params: PeriodParams) => {
    const { year, month } = toYearMonth(params.endDate);
    const response = await apiClient.get<TrialBalanceWire>(
      "/reports/trial-balance",
      { year, month }
    );
    return { ...response, data: toTrialBalance(response.data) };
  },

  /** GET /reports/balance-sheet */
  balanceSheet: async (params: { asOfDate: string; comparisonDate?: string }) => {
    const { year, month } = toYearMonth(params.asOfDate);
    const response = await apiClient.get<BalanceSheetWire>(
      "/reports/balance-sheet",
      { year, month }
    );
    return { ...response, data: toBalanceSheet(response.data) };
  },

  /** GET /reports/income-statement */
  incomeStatement: async (params: PeriodParams) => {
    const from = toYearMonth(params.startDate);
    const to = toYearMonth(params.endDate);
    const response = await apiClient.get<IncomeStatementWire>(
      "/reports/income-statement",
      {
        from_year: from.year,
        from_month: from.month,
        to_year: to.year,
        to_month: to.month,
      }
    );
    return { ...response, data: toIncomeStatement(response.data) };
  },

  /** GET /ledger/balances */
  accountBalances: async (params: PeriodParams) => {
    const { year, month } = toYearMonth(params.endDate);
    const response = await apiClient.get<LedgerBalanceWire[]>(
      "/ledger/balances",
      { year, month }
    );
    const data: AccountBalanceSummary[] = (response.data ?? []).map((row) => ({
      accountId: row.account_id,
      accountCode: row.account_code,
      accountName: row.account_name,
      accountType: row.account_type,
      balance: row.closing_balance,
      debitSum: row.period_debit,
      creditSum: row.period_credit,
    }));
    return { ...response, data };
  },

  // Excel export endpoints do not exist on the backend. They previously passed
  // `responseType: "blob"` as a *query parameter* to a JSON-parsing fetch
  // client, so wiring a button to them would have failed at runtime anyway.
  exportGeneralLedger: async (
    _accountId?: string,
    _params?: PeriodParams
  ): Promise<Blob> => {
    throw new EndpointNotImplementedError("총계정원장 내보내기");
  },
  exportTrialBalance: async (_params?: PeriodParams): Promise<Blob> => {
    throw new EndpointNotImplementedError("합계잔액시산표 내보내기");
  },
  exportBalanceSheet: async (_params?: {
    asOfDate: string;
    comparisonDate?: string;
  }): Promise<Blob> => {
    throw new EndpointNotImplementedError("재무상태표 내보내기");
  },
  exportIncomeStatement: async (_params?: PeriodParams): Promise<Blob> => {
    throw new EndpointNotImplementedError("손익계산서 내보내기");
  },
};
