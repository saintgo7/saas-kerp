package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ledgerRepositoryGorm implements LedgerRepository using GORM
type ledgerRepositoryGorm struct {
	db *gorm.DB
}

// NewLedgerRepository creates a new LedgerRepository with GORM
func NewLedgerRepository(db *gorm.DB) LedgerRepository {
	return &ledgerRepositoryGorm{db: db}
}

// GetBalance retrieves a single ledger balance
func (r *ledgerRepositoryGorm) GetBalance(ctx context.Context, companyID, accountID uuid.UUID, year, month int) (*domain.LedgerBalance, error) {
	var balance domain.LedgerBalance
	err := r.db.WithContext(ctx).
		Preload("Account").
		Where("company_id = ? AND account_id = ? AND fiscal_year = ? AND fiscal_month = ?",
			companyID, accountID, year, month).
		First(&balance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrLedgerBalanceNotFound
		}
		return nil, err
	}
	return &balance, nil
}

// GetBalances retrieves all ledger balances for a period
func (r *ledgerRepositoryGorm) GetBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error) {
	var balances []domain.LedgerBalance
	err := r.db.WithContext(ctx).
		Preload("Account").
		Where("company_id = ? AND fiscal_year = ? AND fiscal_month = ?", companyID, year, month).
		Order("account_id").
		Find(&balances).Error
	return balances, err
}

// GetBalancesByType retrieves ledger balances by account type
func (r *ledgerRepositoryGorm) GetBalancesByType(ctx context.Context, companyID uuid.UUID, year, month int, accountType domain.AccountType) ([]domain.LedgerBalance, error) {
	var balances []domain.LedgerBalance
	err := r.db.WithContext(ctx).
		Preload("Account").
		Joins("JOIN accounts a ON ledger_balances.account_id = a.id").
		Where("ledger_balances.company_id = ? AND fiscal_year = ? AND fiscal_month = ? AND a.account_type = ?",
			companyID, year, month, accountType).
		Order("a.sort_order, a.code").
		Find(&balances).Error
	return balances, err
}

// UpsertBalance inserts or updates a ledger balance
func (r *ledgerRepositoryGorm) UpsertBalance(ctx context.Context, balance *domain.LedgerBalance) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "company_id"}, {Name: "account_id"}, {Name: "fiscal_year"}, {Name: "fiscal_month"}},
			DoUpdates: clause.AssignmentColumns([]string{"opening_debit", "opening_credit", "period_debit", "period_credit", "closing_debit", "closing_credit", "updated_at"}),
		}).
		Create(balance).Error
}

// UpsertBalances inserts or updates multiple ledger balances
func (r *ledgerRepositoryGorm) UpsertBalances(ctx context.Context, balances []domain.LedgerBalance) error {
	if len(balances) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "company_id"}, {Name: "account_id"}, {Name: "fiscal_year"}, {Name: "fiscal_month"}},
			DoUpdates: clause.AssignmentColumns([]string{"opening_debit", "opening_credit", "period_debit", "period_credit", "closing_debit", "closing_credit", "updated_at"}),
		}).
		CreateInBatches(balances, 100).Error
}

// CalculatePeriodBalances calculates balances from posted vouchers
func (r *ledgerRepositoryGorm) CalculatePeriodBalances(ctx context.Context, companyID uuid.UUID, year, month int) ([]domain.LedgerBalance, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	var results []struct {
		AccountID    uuid.UUID `gorm:"column:account_id"`
		PeriodDebit  float64   `gorm:"column:period_debit"`
		PeriodCredit float64   `gorm:"column:period_credit"`
	}

	err := r.db.WithContext(ctx).
		Table("voucher_entries ve").
		Select("ve.account_id, COALESCE(SUM(ve.debit_amount), 0) as period_debit, COALESCE(SUM(ve.credit_amount), 0) as period_credit").
		Joins("JOIN vouchers v ON ve.voucher_id = v.id").
		Where("ve.company_id = ? AND v.status = ? AND v.voucher_date >= ? AND v.voucher_date <= ?",
			companyID, domain.VoucherStatusPosted, startDate, endDate).
		Group("ve.account_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Get previous period balances for opening balances
	var prevYear, prevMonth int
	if month == 1 {
		prevYear = year - 1
		prevMonth = 12
	} else {
		prevYear = year
		prevMonth = month - 1
	}

	prevBalances, _ := r.GetBalances(ctx, companyID, prevYear, prevMonth)
	prevBalanceMap := make(map[uuid.UUID]*domain.LedgerBalance)
	for i := range prevBalances {
		prevBalanceMap[prevBalances[i].AccountID] = &prevBalances[i]
	}

	// Build new balances
	var balances []domain.LedgerBalance
	for _, result := range results {
		balance := domain.LedgerBalance{
			CompanyID:    companyID,
			AccountID:    result.AccountID,
			FiscalYear:   year,
			FiscalMonth:  month,
			PeriodDebit:  result.PeriodDebit,
			PeriodCredit: result.PeriodCredit,
		}

		// Set opening balance from previous period closing
		if prev, ok := prevBalanceMap[result.AccountID]; ok {
			balance.OpeningDebit = prev.ClosingDebit
			balance.OpeningCredit = prev.ClosingCredit
		}

		// Calculate closing
		balance.CalculateClosing()
		balances = append(balances, balance)
	}

	return balances, nil
}

// RecalculateBalances recalculates all balances from a starting period
func (r *ledgerRepositoryGorm) RecalculateBalances(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth int) error {
	// Get current date to determine end period
	now := time.Now()
	endYear := now.Year()
	endMonth := int(now.Month())

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &ledgerRepositoryGorm{db: tx}

		year := fromYear
		month := fromMonth

		for year < endYear || (year == endYear && month <= endMonth) {
			balances, err := txRepo.CalculatePeriodBalances(ctx, companyID, year, month)
			if err != nil {
				return err
			}

			if len(balances) > 0 {
				if err := txRepo.UpsertBalances(ctx, balances); err != nil {
					return err
				}
			}

			// Move to next month
			month++
			if month > 12 {
				month = 1
				year++
			}
		}

		return nil
	})
}

// GetAccountLedger retrieves detailed ledger entries for an account
func (r *ledgerRepositoryGorm) GetAccountLedger(ctx context.Context, companyID, accountID uuid.UUID, from, to time.Time) ([]domain.AccountLedgerEntry, error) {
	var entries []domain.AccountLedgerEntry

	query := `
		SELECT
			v.id as voucher_id,
			v.voucher_no,
			v.voucher_date,
			v.voucher_type,
			ve.id as entry_id,
			ve.line_no,
			ve.description,
			ve.debit_amount,
			ve.credit_amount,
			ve.partner_id,
			p.name as partner_name,
			ve.department_id,
			d.name as department_name
		FROM voucher_entries ve
		JOIN vouchers v ON ve.voucher_id = v.id
		LEFT JOIN partners p ON ve.partner_id = p.id
		LEFT JOIN departments d ON ve.department_id = d.id
		WHERE ve.company_id = ? AND ve.account_id = ?
			AND v.voucher_date >= ? AND v.voucher_date <= ?
			AND v.status = ?
		ORDER BY v.voucher_date, v.voucher_no, ve.line_no
	`

	if err := r.db.WithContext(ctx).Raw(query, companyID, accountID, from, to, domain.VoucherStatusPosted).Scan(&entries).Error; err != nil {
		return nil, err
	}

	// Calculate running balance
	var runningBalance float64
	for i := range entries {
		runningBalance += entries[i].DebitAmount - entries[i].CreditAmount
		entries[i].Balance = runningBalance
	}

	return entries, nil
}

// GetAccountLedgerByPeriod retrieves ledger entries for a period
func (r *ledgerRepositoryGorm) GetAccountLedgerByPeriod(ctx context.Context, companyID, accountID uuid.UUID, year, month int) ([]domain.AccountLedgerEntry, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)
	return r.GetAccountLedger(ctx, companyID, accountID, startDate, endDate)
}

// GetTrialBalance generates a trial balance report
func (r *ledgerRepositoryGorm) GetTrialBalance(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.TrialBalance, error) {
	// Get fiscal period
	period, err := r.GetFiscalPeriod(ctx, companyID, year, month)
	if err != nil && err != domain.ErrFiscalPeriodNotFound {
		return nil, err
	}

	// Get balances with account info
	var items []domain.TrialBalanceItem

	query := `
		SELECT
			lb.account_id,
			a.code as account_code,
			a.name as account_name,
			a.account_type,
			a.level as account_level,
			COALESCE(lb.opening_debit, 0) as opening_debit,
			COALESCE(lb.opening_credit, 0) as opening_credit,
			COALESCE(lb.period_debit, 0) as period_debit,
			COALESCE(lb.period_credit, 0) as period_credit,
			COALESCE(lb.closing_debit, 0) as closing_debit,
			COALESCE(lb.closing_credit, 0) as closing_credit
		FROM ledger_balances lb
		JOIN accounts a ON lb.account_id = a.id
		WHERE lb.company_id = ? AND lb.fiscal_year = ? AND lb.fiscal_month = ?
		ORDER BY a.account_type, a.sort_order, a.code
	`

	if err := r.db.WithContext(ctx).Raw(query, companyID, year, month).Scan(&items).Error; err != nil {
		return nil, err
	}

	// Calculate totals
	var totalDebit, totalCredit float64
	for _, item := range items {
		totalDebit += item.ClosingDebit
		totalCredit += item.ClosingCredit
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	periodName := ""
	if period != nil {
		periodName = period.PeriodName
	}

	tb := &domain.TrialBalance{
		CompanyID:   companyID,
		FiscalYear:  year,
		FiscalMonth: month,
		PeriodName:  periodName,
		StartDate:   startDate,
		EndDate:     endDate,
		GeneratedAt: time.Now(),
		Items:       items,
		TotalDebit:  totalDebit,
		TotalCredit: totalCredit,
	}

	tb.Validate()
	return tb, nil
}

// GetTrialBalanceRange generates a trial balance for a date range
func (r *ledgerRepositoryGorm) GetTrialBalanceRange(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) (*domain.TrialBalance, error) {
	// For range, we sum up all period movements
	var items []domain.TrialBalanceItem

	query := `
		SELECT
			lb.account_id,
			a.code as account_code,
			a.name as account_name,
			a.account_type,
			a.level as account_level,
			COALESCE(SUM(lb.period_debit), 0) as period_debit,
			COALESCE(SUM(lb.period_credit), 0) as period_credit,
			COALESCE(SUM(lb.period_debit), 0) as closing_debit,
			COALESCE(SUM(lb.period_credit), 0) as closing_credit
		FROM ledger_balances lb
		JOIN accounts a ON lb.account_id = a.id
		WHERE lb.company_id = ?
			AND (lb.fiscal_year > ? OR (lb.fiscal_year = ? AND lb.fiscal_month >= ?))
			AND (lb.fiscal_year < ? OR (lb.fiscal_year = ? AND lb.fiscal_month <= ?))
		GROUP BY lb.account_id, a.code, a.name, a.account_type, a.level
		ORDER BY a.account_type, a.sort_order, a.code
	`

	if err := r.db.WithContext(ctx).Raw(query, companyID, fromYear, fromYear, fromMonth, toYear, toYear, toMonth).Scan(&items).Error; err != nil {
		return nil, err
	}

	var totalDebit, totalCredit float64
	for _, item := range items {
		totalDebit += item.ClosingDebit
		totalCredit += item.ClosingCredit
	}

	startDate := time.Date(fromYear, time.Month(fromMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(toYear, time.Month(toMonth)+1, 0, 0, 0, 0, 0, time.UTC)

	tb := &domain.TrialBalance{
		CompanyID:   companyID,
		FiscalYear:  toYear,
		FiscalMonth: toMonth,
		StartDate:   startDate,
		EndDate:     endDate,
		GeneratedAt: time.Now(),
		Items:       items,
		TotalDebit:  totalDebit,
		TotalCredit: totalCredit,
	}

	tb.Validate()
	return tb, nil
}

// GetFiscalPeriod retrieves a fiscal period
func (r *ledgerRepositoryGorm) GetFiscalPeriod(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.FiscalPeriod, error) {
	var period domain.FiscalPeriod
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND fiscal_year = ? AND fiscal_month = ?", companyID, year, month).
		First(&period).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrFiscalPeriodNotFound
		}
		return nil, err
	}
	return &period, nil
}

// GetFiscalPeriods retrieves all fiscal periods for a year
func (r *ledgerRepositoryGorm) GetFiscalPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.FiscalPeriod, error) {
	var periods []domain.FiscalPeriod
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND fiscal_year = ?", companyID, year).
		Order("fiscal_month").
		Find(&periods).Error
	return periods, err
}

// CreateFiscalPeriod creates a new fiscal period
func (r *ledgerRepositoryGorm) CreateFiscalPeriod(ctx context.Context, period *domain.FiscalPeriod) error {
	return r.db.WithContext(ctx).Create(period).Error
}

// UpdateFiscalPeriod updates a fiscal period
func (r *ledgerRepositoryGorm) UpdateFiscalPeriod(ctx context.Context, period *domain.FiscalPeriod) error {
	return r.db.WithContext(ctx).Save(period).Error
}

// GetOpenPeriods retrieves all open fiscal periods
func (r *ledgerRepositoryGorm) GetOpenPeriods(ctx context.Context, companyID uuid.UUID) ([]domain.FiscalPeriod, error) {
	var periods []domain.FiscalPeriod
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND status = ?", companyID, domain.FiscalPeriodOpen).
		Order("fiscal_year, fiscal_month").
		Find(&periods).Error
	return periods, err
}

// CarryForwardBalances carries forward closing balances to next period opening
func (r *ledgerRepositoryGorm) CarryForwardBalances(ctx context.Context, companyID uuid.UUID, fromYear, fromMonth, toYear, toMonth int) error {
	// Get source period balances
	sourceBalances, err := r.GetBalances(ctx, companyID, fromYear, fromMonth)
	if err != nil {
		return err
	}

	// Create target period balances with opening = source closing
	var targetBalances []domain.LedgerBalance
	for _, src := range sourceBalances {
		target := domain.LedgerBalance{
			CompanyID:     companyID,
			AccountID:     src.AccountID,
			FiscalYear:    toYear,
			FiscalMonth:   toMonth,
			OpeningDebit:  src.ClosingDebit,
			OpeningCredit: src.ClosingCredit,
			PeriodDebit:   0,
			PeriodCredit:  0,
			ClosingDebit:  src.ClosingDebit,
			ClosingCredit: src.ClosingCredit,
		}
		targetBalances = append(targetBalances, target)
	}

	return r.UpsertBalances(ctx, targetBalances)
}
