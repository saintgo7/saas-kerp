package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// accountRepositoryGorm implements AccountRepository using GORM
type accountRepositoryGorm struct {
	db *gorm.DB
}

// NewAccountRepository creates a new AccountRepository with GORM
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepositoryGorm{db: db}
}

// Create inserts a new account
func (r *accountRepositoryGorm) Create(ctx context.Context, account *domain.Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// Update modifies an existing account
func (r *accountRepositoryGorm) Update(ctx context.Context, account *domain.Account) error {
	return r.db.WithContext(ctx).
		Model(account).
		Select("code", "name", "name_en", "parent_id", "level", "path",
			"account_type", "account_nature", "account_category",
			"is_active", "is_control_account", "allow_direct_posting", "sort_order").
		Updates(account).Error
}

// Delete removes an account by ID
func (r *accountRepositoryGorm) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		Delete(&domain.Account{}).Error
}

// FindByID retrieves an account by ID
func (r *accountRepositoryGorm) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Account, error) {
	var account domain.Account
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND id = ?", companyID, id).
		First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

// FindByCode retrieves an account by code
func (r *accountRepositoryGorm) FindByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Account, error) {
	var account domain.Account
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND code = ?", companyID, code).
		First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

// FindAll retrieves accounts with filtering and pagination
func (r *accountRepositoryGorm) FindAll(ctx context.Context, filter AccountFilter) ([]domain.Account, int64, error) {
	var accounts []domain.Account
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Account{}).
		Where("company_id = ?", filter.CompanyID)

	// Apply filters
	if filter.ParentID != nil {
		query = query.Where("parent_id = ?", *filter.ParentID)
	}
	if filter.AccountType != nil {
		query = query.Where("account_type = ?", *filter.AccountType)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.SearchTerm != "" {
		searchTerm := "%" + strings.ToLower(filter.SearchTerm) + "%"
		query = query.Where("LOWER(code) LIKE ? OR LOWER(name) LIKE ? OR LOWER(name_en) LIKE ?",
			searchTerm, searchTerm, searchTerm)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "sort_order"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	if filter.SortDesc {
		sortBy = sortBy + " DESC"
	}
	query = query.Order(sortBy)

	// Apply pagination
	if filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		if offset < 0 {
			offset = 0
		}
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Include children if tree is requested
	if filter.IncludeTree {
		query = query.Preload("Children")
	}

	if err := query.Find(&accounts).Error; err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// FindChildren retrieves direct children of an account
func (r *accountRepositoryGorm) FindChildren(ctx context.Context, companyID, parentID uuid.UUID) ([]domain.Account, error) {
	var accounts []domain.Account
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND parent_id = ?", companyID, parentID).
		Order("sort_order, code").
		Find(&accounts).Error
	return accounts, err
}

// FindByType retrieves accounts by type
func (r *accountRepositoryGorm) FindByType(ctx context.Context, companyID uuid.UUID, accountType domain.AccountType) ([]domain.Account, error) {
	var accounts []domain.Account
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND account_type = ?", companyID, accountType).
		Order("sort_order, code").
		Find(&accounts).Error
	return accounts, err
}

// GetTree retrieves the full account tree for a company
func (r *accountRepositoryGorm) GetTree(ctx context.Context, companyID uuid.UUID) ([]domain.Account, error) {
	var accounts []domain.Account
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND parent_id IS NULL", companyID).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order, code")
		}).
		Order("sort_order, code").
		Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	// Recursively load children
	for i := range accounts {
		if err := r.loadChildren(ctx, &accounts[i]); err != nil {
			return nil, err
		}
	}

	return accounts, nil
}

// loadChildren recursively loads children for an account
func (r *accountRepositoryGorm) loadChildren(ctx context.Context, account *domain.Account) error {
	if len(account.Children) == 0 {
		return nil
	}
	for i := range account.Children {
		if err := r.db.WithContext(ctx).
			Where("parent_id = ?", account.Children[i].ID).
			Order("sort_order, code").
			Find(&account.Children[i].Children).Error; err != nil {
			return err
		}
		if err := r.loadChildren(ctx, &account.Children[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetAncestors retrieves all ancestor accounts
func (r *accountRepositoryGorm) GetAncestors(ctx context.Context, companyID, id uuid.UUID) ([]domain.Account, error) {
	var ancestors []domain.Account

	// Use recursive CTE for efficient ancestor retrieval
	query := `
		WITH RECURSIVE ancestors AS (
			SELECT * FROM accounts WHERE company_id = ? AND id = ?
			UNION ALL
			SELECT a.* FROM accounts a
			INNER JOIN ancestors anc ON a.id = anc.parent_id
			WHERE a.company_id = ?
		)
		SELECT * FROM ancestors WHERE id != ? ORDER BY level ASC
	`

	err := r.db.WithContext(ctx).Raw(query, companyID, id, companyID, id).Scan(&ancestors).Error
	return ancestors, err
}

// GetDescendants retrieves all descendant accounts
func (r *accountRepositoryGorm) GetDescendants(ctx context.Context, companyID, id uuid.UUID) ([]domain.Account, error) {
	var descendants []domain.Account

	// Use recursive CTE for efficient descendant retrieval
	query := `
		WITH RECURSIVE descendants AS (
			SELECT * FROM accounts WHERE company_id = ? AND id = ?
			UNION ALL
			SELECT a.* FROM accounts a
			INNER JOIN descendants d ON a.parent_id = d.id
			WHERE a.company_id = ?
		)
		SELECT * FROM descendants WHERE id != ? ORDER BY level ASC, sort_order, code
	`

	err := r.db.WithContext(ctx).Raw(query, companyID, id, companyID, id).Scan(&descendants).Error
	return descendants, err
}

// UpdatePath updates the path for an account and its descendants
func (r *accountRepositoryGorm) UpdatePath(ctx context.Context, account *domain.Account) error {
	// Calculate new path
	var newPath string
	if account.ParentID == nil {
		newPath = account.Code
	} else {
		var parent domain.Account
		if err := r.db.WithContext(ctx).
			Select("path").
			Where("id = ?", account.ParentID).
			First(&parent).Error; err != nil {
			return err
		}
		newPath = fmt.Sprintf("%s.%s", parent.Path, account.Code)
	}

	// Update account path
	return r.db.WithContext(ctx).
		Model(&domain.Account{}).
		Where("id = ?", account.ID).
		Update("path", newPath).Error
}

// ExistsByCode checks if an account with the given code exists
func (r *accountRepositoryGorm) ExistsByCode(ctx context.Context, companyID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Account{}).
		Where("company_id = ? AND code = ?", companyID, code)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	err := query.Count(&count).Error
	return count > 0, err
}

// HasChildren checks if an account has child accounts
func (r *accountRepositoryGorm) HasChildren(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Account{}).
		Where("company_id = ? AND parent_id = ?", companyID, id).
		Count(&count).Error
	return count > 0, err
}

// HasVoucherEntries checks if an account has associated voucher entries
func (r *accountRepositoryGorm) HasVoucherEntries(ctx context.Context, companyID, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("voucher_entries").
		Where("company_id = ? AND account_id = ?", companyID, id).
		Count(&count).Error
	return count > 0, err
}

// CreateBatch inserts multiple accounts in a single transaction
func (r *accountRepositoryGorm) CreateBatch(ctx context.Context, accounts []domain.Account) error {
	return r.db.WithContext(ctx).CreateInBatches(accounts, 100).Error
}

// UpdateSortOrder updates sort orders for multiple accounts
func (r *accountRepositoryGorm) UpdateSortOrder(ctx context.Context, companyID uuid.UUID, orders map[uuid.UUID]int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for id, order := range orders {
			if err := tx.Model(&domain.Account{}).
				Where("company_id = ? AND id = ?", companyID, id).
				Update("sort_order", order).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
