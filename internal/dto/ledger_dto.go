package dto

import (
	"time"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// LedgerBalanceResponse represents a ledger balance
type LedgerBalanceResponse struct {
	AccountID     string  `json:"account_id"`
	AccountCode   string  `json:"account_code"`
	AccountName   string  `json:"account_name"`
	AccountType   string  `json:"account_type"`
	FiscalYear    int     `json:"fiscal_year"`
	FiscalMonth   int     `json:"fiscal_month"`
	OpeningDebit  float64 `json:"opening_debit"`
	OpeningCredit float64 `json:"opening_credit"`
	OpeningBalance float64 `json:"opening_balance"`
	PeriodDebit   float64 `json:"period_debit"`
	PeriodCredit  float64 `json:"period_credit"`
	PeriodMovement float64 `json:"period_movement"`
	ClosingDebit  float64 `json:"closing_debit"`
	ClosingCredit float64 `json:"closing_credit"`
	ClosingBalance float64 `json:"closing_balance"`
}

// FromLedgerBalance converts domain.LedgerBalance to LedgerBalanceResponse
func FromLedgerBalance(balance *domain.LedgerBalance) LedgerBalanceResponse {
	resp := LedgerBalanceResponse{
		AccountID:     balance.AccountID.String(),
		FiscalYear:    balance.FiscalYear,
		FiscalMonth:   balance.FiscalMonth,
		OpeningDebit:  balance.OpeningDebit,
		OpeningCredit: balance.OpeningCredit,
		OpeningBalance: balance.GetOpeningBalance(),
		PeriodDebit:   balance.PeriodDebit,
		PeriodCredit:  balance.PeriodCredit,
		PeriodMovement: balance.GetPeriodMovement(),
		ClosingDebit:  balance.ClosingDebit,
		ClosingCredit: balance.ClosingCredit,
		ClosingBalance: balance.GetClosingBalance(),
	}

	if balance.Account != nil {
		resp.AccountCode = balance.Account.Code
		resp.AccountName = balance.Account.Name
		resp.AccountType = string(balance.Account.AccountType)
	}

	return resp
}

// FromLedgerBalances converts []domain.LedgerBalance to []LedgerBalanceResponse
func FromLedgerBalances(balances []domain.LedgerBalance) []LedgerBalanceResponse {
	responses := make([]LedgerBalanceResponse, len(balances))
	for i := range balances {
		responses[i] = FromLedgerBalance(&balances[i])
	}
	return responses
}

// AccountLedgerEntryResponse represents a ledger entry
type AccountLedgerEntryResponse struct {
	VoucherID      string  `json:"voucher_id"`
	VoucherNo      string  `json:"voucher_no"`
	VoucherDate    string  `json:"voucher_date"`
	VoucherType    string  `json:"voucher_type"`
	EntryID        string  `json:"entry_id"`
	LineNo         int     `json:"line_no"`
	Description    string  `json:"description,omitempty"`
	DebitAmount    float64 `json:"debit_amount"`
	CreditAmount   float64 `json:"credit_amount"`
	Balance        float64 `json:"balance"`
	PartnerID      string  `json:"partner_id,omitempty"`
	PartnerName    string  `json:"partner_name,omitempty"`
	DepartmentID   string  `json:"department_id,omitempty"`
	DepartmentName string  `json:"department_name,omitempty"`
}

// FromAccountLedgerEntry converts domain.AccountLedgerEntry to AccountLedgerEntryResponse
func FromAccountLedgerEntry(entry *domain.AccountLedgerEntry) AccountLedgerEntryResponse {
	resp := AccountLedgerEntryResponse{
		VoucherID:    entry.VoucherID.String(),
		VoucherNo:    entry.VoucherNo,
		VoucherDate:  entry.VoucherDate.Format("2006-01-02"),
		VoucherType:  entry.VoucherType,
		EntryID:      entry.EntryID.String(),
		LineNo:       entry.LineNo,
		Description:  entry.Description,
		DebitAmount:  entry.DebitAmount,
		CreditAmount: entry.CreditAmount,
		Balance:      entry.Balance,
		PartnerName:  entry.PartnerName,
		DepartmentName: entry.DepartmentName,
	}

	if entry.PartnerID != nil {
		resp.PartnerID = entry.PartnerID.String()
	}
	if entry.DepartmentID != nil {
		resp.DepartmentID = entry.DepartmentID.String()
	}

	return resp
}

// AccountLedgerResponse represents an account ledger report
type AccountLedgerResponse struct {
	AccountID      string                       `json:"account_id"`
	AccountCode    string                       `json:"account_code"`
	AccountName    string                       `json:"account_name"`
	FromDate       string                       `json:"from_date"`
	ToDate         string                       `json:"to_date"`
	OpeningBalance float64                      `json:"opening_balance"`
	TotalDebit     float64                      `json:"total_debit"`
	TotalCredit    float64                      `json:"total_credit"`
	ClosingBalance float64                      `json:"closing_balance"`
	Entries        []AccountLedgerEntryResponse `json:"entries"`
}

// TrialBalanceItemResponse represents a trial balance line item
type TrialBalanceItemResponse struct {
	AccountID      string  `json:"account_id"`
	AccountCode    string  `json:"account_code"`
	AccountName    string  `json:"account_name"`
	AccountType    string  `json:"account_type"`
	AccountLevel   int     `json:"account_level"`
	OpeningDebit   float64 `json:"opening_debit"`
	OpeningCredit  float64 `json:"opening_credit"`
	PeriodDebit    float64 `json:"period_debit"`
	PeriodCredit   float64 `json:"period_credit"`
	ClosingDebit   float64 `json:"closing_debit"`
	ClosingCredit  float64 `json:"closing_credit"`
	IsSubTotal     bool    `json:"is_sub_total"`
	IsTotal        bool    `json:"is_total"`
}

// TrialBalanceResponse represents a trial balance report
type TrialBalanceResponse struct {
	CompanyID     string                     `json:"company_id"`
	FiscalYear    int                        `json:"fiscal_year"`
	FiscalMonth   int                        `json:"fiscal_month"`
	PeriodName    string                     `json:"period_name"`
	StartDate     string                     `json:"start_date"`
	EndDate       string                     `json:"end_date"`
	GeneratedAt   string                     `json:"generated_at"`
	Items         []TrialBalanceItemResponse `json:"items"`
	TotalDebit    float64                    `json:"total_debit"`
	TotalCredit   float64                    `json:"total_credit"`
	IsBalanced    bool                       `json:"is_balanced"`
}

// FromTrialBalance converts domain.TrialBalance to TrialBalanceResponse
func FromTrialBalance(tb *domain.TrialBalance) TrialBalanceResponse {
	items := make([]TrialBalanceItemResponse, len(tb.Items))
	for i, item := range tb.Items {
		items[i] = TrialBalanceItemResponse{
			AccountID:     item.AccountID.String(),
			AccountCode:   item.AccountCode,
			AccountName:   item.AccountName,
			AccountType:   item.AccountType,
			AccountLevel:  item.AccountLevel,
			OpeningDebit:  item.OpeningDebit,
			OpeningCredit: item.OpeningCredit,
			PeriodDebit:   item.PeriodDebit,
			PeriodCredit:  item.PeriodCredit,
			ClosingDebit:  item.ClosingDebit,
			ClosingCredit: item.ClosingCredit,
			IsSubTotal:    item.IsSubTotal,
			IsTotal:       item.IsTotal,
		}
	}

	return TrialBalanceResponse{
		CompanyID:   tb.CompanyID.String(),
		FiscalYear:  tb.FiscalYear,
		FiscalMonth: tb.FiscalMonth,
		PeriodName:  tb.PeriodName,
		StartDate:   tb.StartDate.Format("2006-01-02"),
		EndDate:     tb.EndDate.Format("2006-01-02"),
		GeneratedAt: tb.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
		Items:       items,
		TotalDebit:  tb.TotalDebit,
		TotalCredit: tb.TotalCredit,
		IsBalanced:  tb.IsBalanced,
	}
}

// FiscalPeriodResponse represents a fiscal period
type FiscalPeriodResponse struct {
	ID          string  `json:"id"`
	FiscalYear  int     `json:"fiscal_year"`
	FiscalMonth int     `json:"fiscal_month"`
	PeriodName  string  `json:"period_name"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	Status      string  `json:"status"`
	ClosedAt    string  `json:"closed_at,omitempty"`
}

// FromFiscalPeriod converts domain.FiscalPeriod to FiscalPeriodResponse
func FromFiscalPeriod(period *domain.FiscalPeriod) FiscalPeriodResponse {
	resp := FiscalPeriodResponse{
		ID:          period.ID.String(),
		FiscalYear:  period.FiscalYear,
		FiscalMonth: period.FiscalMonth,
		PeriodName:  period.PeriodName,
		StartDate:   period.StartDate.Format("2006-01-02"),
		EndDate:     period.EndDate.Format("2006-01-02"),
		Status:      string(period.Status),
	}

	if period.ClosedAt != nil {
		resp.ClosedAt = period.ClosedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return resp
}

// FromFiscalPeriods converts []domain.FiscalPeriod to []FiscalPeriodResponse
func FromFiscalPeriods(periods []domain.FiscalPeriod) []FiscalPeriodResponse {
	responses := make([]FiscalPeriodResponse, len(periods))
	for i := range periods {
		responses[i] = FromFiscalPeriod(&periods[i])
	}
	return responses
}

// FinancialStatementItem represents a line in financial statement
type FinancialStatementItem struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Amount      float64 `json:"amount"`
	Level       int     `json:"level"`
	IsSubTotal  bool    `json:"is_sub_total"`
	IsTotal     bool    `json:"is_total"`
}

// BalanceSheetResponse represents a balance sheet report
type BalanceSheetResponse struct {
	CompanyID      string                   `json:"company_id"`
	AsOfDate       string                   `json:"as_of_date"`
	GeneratedAt    string                   `json:"generated_at"`
	Assets         []FinancialStatementItem `json:"assets"`
	Liabilities    []FinancialStatementItem `json:"liabilities"`
	Equity         []FinancialStatementItem `json:"equity"`
	TotalAssets    float64                  `json:"total_assets"`
	TotalLiabilities float64                `json:"total_liabilities"`
	TotalEquity    float64                  `json:"total_equity"`
	IsBalanced     bool                     `json:"is_balanced"`
}

// IncomeStatementResponse represents an income statement report
type IncomeStatementResponse struct {
	CompanyID       string                   `json:"company_id"`
	FromDate        string                   `json:"from_date"`
	ToDate          string                   `json:"to_date"`
	GeneratedAt     string                   `json:"generated_at"`
	Revenue         []FinancialStatementItem `json:"revenue"`
	Expenses        []FinancialStatementItem `json:"expenses"`
	TotalRevenue    float64                  `json:"total_revenue"`
	TotalExpenses   float64                  `json:"total_expenses"`
	NetIncome       float64                  `json:"net_income"`
}

// AccountLedgerRequest represents query parameters for account ledger
type AccountLedgerRequest struct {
	AccountID string `form:"account_id" binding:"required,uuid"`
	FromDate  string `form:"from_date" binding:"required"`
	ToDate    string `form:"to_date" binding:"required"`
}

// PeriodRequest represents query parameters for period-based reports
type PeriodRequest struct {
	Year  int `form:"year" binding:"required,min=2000,max=2100"`
	Month int `form:"month" binding:"required,min=1,max=12"`
}

// DateRangeRequest represents query parameters for date range reports
type DateRangeRequest struct {
	FromYear  int `form:"from_year" binding:"required,min=2000,max=2100"`
	FromMonth int `form:"from_month" binding:"required,min=1,max=12"`
	ToYear    int `form:"to_year" binding:"required,min=2000,max=2100"`
	ToMonth   int `form:"to_month" binding:"required,min=1,max=12"`
}

// ClosePeriodRequest represents the request to close a period
type ClosePeriodRequest struct {
	Year  int `json:"year" binding:"required,min=2000,max=2100"`
	Month int `json:"month" binding:"required,min=1,max=12"`
}

// YearEndCloseRequest represents the request for year-end closing
type YearEndCloseRequest struct {
	Year                      int    `json:"year" binding:"required,min=2000,max=2100"`
	RetainedEarningsAccountID string `json:"retained_earnings_account_id" binding:"required,uuid"`
}

// ReportGeneratedAt is a helper to get current time for reports
func ReportGeneratedAt() string {
	return time.Now().Format("2006-01-02T15:04:05Z07:00")
}
