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

// LedgerBalance represents pre-aggregated account balances by period.
//
// This model deliberately does NOT embed BaseModel. The ledger_balances table
// (db/migrations/000005_accounting_tables.up.sql) carries only updated_at as
// its audit column, so embedding BaseModel made GORM put created_at in every
// INSERT column list and every write failed with
// `42703: column "created_at" of relation "ledger_balances" does not exist` -
// taking period close, the trial balance and the year-end close with it.
//
// Declaring only the columns that exist keeps this model correct whether or
// not the accompanying migration that adds created_at has been applied: if the
// column is added with a NOT NULL DEFAULT NOW(), the database fills it in.
type LedgerBalance struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v7()" json:"id"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

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

// GetClosingBalance returns the closing net balance expressed on the DEBIT
// side: debit - credit. A credit-nature account (liability, equity, revenue)
// therefore reports a NEGATIVE value when it carries a normal balance.
//
// Callers that need the amount as it is presented on a financial statement
// must use GetClosingBalanceByNature instead; mixing the two conventions is
// what made the year-end close report a loss for a profitable year.
func (lb *LedgerBalance) GetClosingBalance() float64 {
	return lb.ClosingDebit - lb.ClosingCredit
}

// GetClosingBalanceByNature returns the closing balance signed according to
// the account's normal balance side, i.e. the amount as it appears on the
// financial statements. Revenue of 1,000,000 returns +1,000,000 (not
// -1,000,000), and an expense of 600,000 returns +600,000.
//
// A negative result means the account carries a balance on the side opposite
// to its nature (for example a revenue account with a net debit balance after
// sales returns), which is meaningful and must not be clamped.
func (lb *LedgerBalance) GetClosingBalanceByNature(nature AccountNature) float64 {
	if nature == AccountNatureCredit {
		return lb.ClosingCredit - lb.ClosingDebit
	}
	return lb.ClosingDebit - lb.ClosingCredit
}

// AccountLedgerEntry represents a single ledger entry for an account
type AccountLedgerEntry struct {
	VoucherID      uuid.UUID  `json:"voucher_id"`
	VoucherNo      string     `json:"voucher_no"`
	VoucherDate    time.Time  `json:"voucher_date"`
	VoucherType    string     `json:"voucher_type"`
	EntryID        uuid.UUID  `json:"entry_id"`
	LineNo         int        `json:"line_no"`
	Description    string     `json:"description"`
	DebitAmount    float64    `json:"debit_amount"`
	CreditAmount   float64    `json:"credit_amount"`
	Balance        float64    `json:"balance"` // Running balance
	PartnerID      *uuid.UUID `json:"partner_id,omitempty"`
	PartnerName    string     `json:"partner_name,omitempty"`
	DepartmentID   *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName string     `json:"department_name,omitempty"`
}

// TrialBalanceItem represents a single item in the trial balance report
type TrialBalanceItem struct {
	AccountID     uuid.UUID `json:"account_id"`
	AccountCode   string    `json:"account_code"`
	AccountName   string    `json:"account_name"`
	AccountType   string    `json:"account_type"`
	AccountLevel  int       `json:"account_level"`
	OpeningDebit  float64   `json:"opening_debit"`
	OpeningCredit float64   `json:"opening_credit"`
	PeriodDebit   float64   `json:"period_debit"`
	PeriodCredit  float64   `json:"period_credit"`
	ClosingDebit  float64   `json:"closing_debit"`
	ClosingCredit float64   `json:"closing_credit"`
	IsSubTotal    bool      `json:"is_sub_total"`
	IsTotal       bool      `json:"is_total"`
}

// TrialBalance represents a trial balance report
type TrialBalance struct {
	CompanyID   uuid.UUID          `json:"company_id"`
	FiscalYear  int                `json:"fiscal_year"`
	FiscalMonth int                `json:"fiscal_month"`
	PeriodName  string             `json:"period_name"`
	StartDate   time.Time          `json:"start_date"`
	EndDate     time.Time          `json:"end_date"`
	GeneratedAt time.Time          `json:"generated_at"`
	Items       []TrialBalanceItem `json:"items"`
	TotalDebit  float64            `json:"total_debit"`
	TotalCredit float64            `json:"total_credit"`
	IsBalanced  bool               `json:"is_balanced"`
}

// Validate checks if the trial balance is balanced.
//
// The totals are float64 sums over hundreds of DECIMAL(18,2) rows, so binary
// representation error accumulates. An exact == comparison reports a perfectly
// balanced ledger as unbalanced; BalanceEpsilon is the same tolerance the
// voucher layer uses.
func (tb *TrialBalance) Validate() bool {
	tb.IsBalanced = AmountsEqual(tb.TotalDebit, tb.TotalCredit)
	return tb.IsBalanced
}
