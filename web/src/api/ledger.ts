import { apiClient } from "./client";
import type { Account, AccountType } from "@/types";

/**
 * Period parameters for ledger queries
 */
export interface PeriodParams {
  startDate: string;
  endDate: string;
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
 * Ledger API client
 */
export const ledgerApi = {
  /**
   * Get general ledger for a specific account
   */
  generalLedger: async (accountId: string, params: PeriodParams) => {
    return apiClient.get<GeneralLedgerData>(`/ledger/general/${accountId}`, params);
  },

  /**
   * Get general ledger for all accounts
   */
  generalLedgerAll: async (params: PeriodParams & { accountType?: AccountType }) => {
    return apiClient.get<GeneralLedgerData[]>("/ledger/general", params);
  },

  /**
   * Get subsidiary ledger for a specific account
   */
  subsidiaryLedger: async (
    accountId: string,
    params: PeriodParams & { partnerId?: string }
  ) => {
    return apiClient.get<SubsidiaryLedgerData>(`/ledger/subsidiary/${accountId}`, params);
  },

  /**
   * Get subsidiary ledger grouped by partner
   */
  subsidiaryLedgerByPartner: async (
    accountId: string,
    params: PeriodParams
  ) => {
    return apiClient.get<SubsidiaryLedgerData[]>(
      `/ledger/subsidiary/${accountId}/by-partner`,
      params
    );
  },

  /**
   * Get trial balance
   */
  trialBalance: async (params: PeriodParams & { includeZeroBalance?: boolean }) => {
    return apiClient.get<TrialBalanceData>("/ledger/trial-balance", params);
  },

  /**
   * Get balance sheet
   */
  balanceSheet: async (params: { asOfDate: string; comparisonDate?: string }) => {
    return apiClient.get<BalanceSheetData>("/reports/balance-sheet", params);
  },

  /**
   * Get income statement
   */
  incomeStatement: async (
    params: PeriodParams & {
      comparisonStartDate?: string;
      comparisonEndDate?: string;
      includeBudget?: boolean;
    }
  ) => {
    return apiClient.get<IncomeStatementData>("/reports/income-statement", params);
  },

  /**
   * Get account balances summary
   */
  accountBalances: async (params: PeriodParams & { accountType?: AccountType }) => {
    return apiClient.get<AccountBalanceSummary[]>("/ledger/account-balances", params);
  },

  /**
   * Export general ledger to Excel
   */
  exportGeneralLedger: async (accountId: string, params: PeriodParams) => {
    return apiClient.get<Blob>(`/ledger/general/${accountId}/export`, {
      ...params,
      responseType: "blob",
    } as Record<string, string | number | boolean | undefined>);
  },

  /**
   * Export trial balance to Excel
   */
  exportTrialBalance: async (params: PeriodParams) => {
    return apiClient.get<Blob>("/ledger/trial-balance/export", {
      ...params,
      responseType: "blob",
    } as Record<string, string | number | boolean | undefined>);
  },

  /**
   * Export balance sheet to Excel
   */
  exportBalanceSheet: async (params: { asOfDate: string; comparisonDate?: string }) => {
    return apiClient.get<Blob>("/reports/balance-sheet/export", {
      ...params,
      responseType: "blob",
    } as Record<string, string | number | boolean | undefined>);
  },

  /**
   * Export income statement to Excel
   */
  exportIncomeStatement: async (params: PeriodParams) => {
    return apiClient.get<Blob>("/reports/income-statement/export", {
      ...params,
      responseType: "blob",
    } as Record<string, string | number | boolean | undefined>);
  },
};
