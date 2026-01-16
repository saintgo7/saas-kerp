package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Ledger errors
var (
	ErrLedgerBalanceNotFound = errors.New("ledger balance not found")
	ErrFiscalPeriodNotFound  = errors.New("fiscal period not found")
	ErrFiscalPeriodClosed    = errors.New("fiscal period is closed")
)

// FiscalPeriodStatus represents the status of a fiscal period
type FiscalPeriodStatus string

const (
	FiscalPeriodOpen   FiscalPeriodStatus = "open"
	FiscalPeriodClosed FiscalPeriodStatus = "closed"
	FiscalPeriodLocked FiscalPeriodStatus = "locked"
)

// FiscalPeriod represents a fiscal period for accounting close
type FiscalPeriod struct {
	BaseModel
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`

	// Period info
	FiscalYear  int    `gorm:"not null" json:"fiscal_year"`
	FiscalMonth int    `gorm:"not null;check:fiscal_month >= 1 AND fiscal_month <= 12" json:"fiscal_month"`
	PeriodName  string `gorm:"type:varchar(50)" json:"period_name,omitempty"`

	// Dates
	StartDate time.Time `gorm:"type:date;not null" json:"start_date"`
	EndDate   time.Time `gorm:"type:date;not null" json:"end_date"`

	// Status
	Status   FiscalPeriodStatus `gorm:"type:varchar(20);default:open" json:"status"`
	ClosedAt *time.Time         `json:"closed_at,omitempty"`
	ClosedBy *uuid.UUID         `gorm:"type:uuid" json:"closed_by,omitempty"`
}

// TableName specifies the table name for GORM
func (FiscalPeriod) TableName() string {
	return "fiscal_periods"
}

// IsOpen returns true if the period is open for posting
func (p *FiscalPeriod) IsOpen() bool {
	return p.Status == FiscalPeriodOpen
}

// CanPost returns true if vouchers can be posted to this period
func (p *FiscalPeriod) CanPost() bool {
	return p.Status == FiscalPeriodOpen
}

// Close closes the fiscal period
func (p *FiscalPeriod) Close(userID uuid.UUID) error {
	if p.Status != FiscalPeriodOpen {
		return ErrFiscalPeriodClosed
	}
	now := time.Now()
	p.Status = FiscalPeriodClosed
	p.ClosedAt = &now
	p.ClosedBy = &userID
	return nil
}

// LedgerBalance represents pre-aggregated account balances by period
type LedgerBalance struct {
	BaseModel
	CompanyID uuid.UUID `gorm:"type:uuid;not null;index" json:"company_id"`
	AccountID uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`

	// Period
	FiscalYear  int `gorm:"not null" json:"fiscal_year"`
	FiscalMonth int `gorm:"not null;check:fiscal_month >= 1 AND fiscal_month <= 12" json:"fiscal_month"`

	// Balances
	OpeningDebit  float64 `gorm:"type:decimal(18,2);not null;default:0" json:"opening_debit"`
	OpeningCredit float64 `gorm:"type:decimal(18,2);not null;default:0" json:"opening_credit"`
	PeriodDebit   float64 `gorm:"type:decimal(18,2);not null;default:0" json:"period_debit"`
	PeriodCredit  float64 `gorm:"type:decimal(18,2);not null;default:0" json:"period_credit"`
	ClosingDebit  float64 `gorm:"type:decimal(18,2);not null;default:0" json:"closing_debit"`
	ClosingCredit float64 `gorm:"type:decimal(18,2);not null;default:0" json:"closing_credit"`

	// Computed balance (stored as generated column in DB)
	Balance float64 `gorm:"->" json:"balance"` // Read-only from DB

	// Relations
	Account *Account `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

// TableName specifies the table name for GORM
func (LedgerBalance) TableName() string {
	return "ledger_balances"
}

// CalculateClosing calculates closing balances
func (lb *LedgerBalance) CalculateClosing() {
	lb.ClosingDebit = lb.OpeningDebit + lb.PeriodDebit
	lb.ClosingCredit = lb.OpeningCredit + lb.PeriodCredit
}

// GetNetBalance returns the net balance (debit - credit)
func (lb *LedgerBalance) GetNetBalance() float64 {
	return (lb.OpeningDebit - lb.OpeningCredit) + (lb.PeriodDebit - lb.PeriodCredit)
}

// GetOpeningBalance returns the opening net balance
func (lb *LedgerBalance) GetOpeningBalance() float64 {
	return lb.OpeningDebit - lb.OpeningCredit
}

// GetPeriodMovement returns the period net movement
func (lb *LedgerBalance) GetPeriodMovement() float64 {
	return lb.PeriodDebit - lb.PeriodCredit
}

// GetClosingBalance returns the closing net balance
func (lb *LedgerBalance) GetClosingBalance() float64 {
	return lb.ClosingDebit - lb.ClosingCredit
}

// AccountLedgerEntry represents a single ledger entry for an account
type AccountLedgerEntry struct {
	VoucherID     uuid.UUID `json:"voucher_id"`
	VoucherNo     string    `json:"voucher_no"`
	VoucherDate   time.Time `json:"voucher_date"`
	VoucherType   string    `json:"voucher_type"`
	EntryID       uuid.UUID `json:"entry_id"`
	LineNo        int       `json:"line_no"`
	Description   string    `json:"description"`
	DebitAmount   float64   `json:"debit_amount"`
	CreditAmount  float64   `json:"credit_amount"`
	Balance       float64   `json:"balance"` // Running balance
	PartnerID     *uuid.UUID `json:"partner_id,omitempty"`
	PartnerName   string    `json:"partner_name,omitempty"`
	DepartmentID  *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName string   `json:"department_name,omitempty"`
}

// TrialBalanceItem represents a single item in the trial balance report
type TrialBalanceItem struct {
	AccountID       uuid.UUID `json:"account_id"`
	AccountCode     string    `json:"account_code"`
	AccountName     string    `json:"account_name"`
	AccountType     string    `json:"account_type"`
	AccountLevel    int       `json:"account_level"`
	OpeningDebit    float64   `json:"opening_debit"`
	OpeningCredit   float64   `json:"opening_credit"`
	PeriodDebit     float64   `json:"period_debit"`
	PeriodCredit    float64   `json:"period_credit"`
	ClosingDebit    float64   `json:"closing_debit"`
	ClosingCredit   float64   `json:"closing_credit"`
	IsSubTotal      bool      `json:"is_sub_total"`
	IsTotal         bool      `json:"is_total"`
}

// TrialBalance represents a trial balance report
type TrialBalance struct {
	CompanyID     uuid.UUID          `json:"company_id"`
	FiscalYear    int                `json:"fiscal_year"`
	FiscalMonth   int                `json:"fiscal_month"`
	PeriodName    string             `json:"period_name"`
	StartDate     time.Time          `json:"start_date"`
	EndDate       time.Time          `json:"end_date"`
	GeneratedAt   time.Time          `json:"generated_at"`
	Items         []TrialBalanceItem `json:"items"`
	TotalDebit    float64            `json:"total_debit"`
	TotalCredit   float64            `json:"total_credit"`
	IsBalanced    bool               `json:"is_balanced"`
}

// Validate checks if the trial balance is balanced
func (tb *TrialBalance) Validate() bool {
	tb.IsBalanced = tb.TotalDebit == tb.TotalCredit
	return tb.IsBalanced
}
