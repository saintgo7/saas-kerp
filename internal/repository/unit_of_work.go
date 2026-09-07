package repository

import (
	"context"

	"gorm.io/gorm"
)

// UnitOfWork runs a function against repositories that all share one database
// transaction.
//
// The per-aggregate WithTransaction methods (VoucherRepository,
// TaxInvoiceRepository) cover work that stays inside a single aggregate. Some
// operations do not: self-service registration has to insert a companies row
// and a users row together, because users.company_id references companies(id)
// and a user without its company violates the foreign key while a company
// without its user is an orphan tenant nobody can log into.
type UnitOfWork interface {
	// Do runs fn inside one transaction. Everything fn writes through the
	// Repositories it is handed is committed together or rolled back together.
	Do(ctx context.Context, fn func(repos Repositories) error) error
}

// Repositories is the set of repositories bound to one transaction.
//
// It is a struct rather than an interface with accessor methods so that adding
// a repository here does not force every implementation to grow a method.
type Repositories struct {
	Company CompanyRepository
	User    UserRepository
}

// unitOfWorkGorm implements UnitOfWork with GORM.
type unitOfWorkGorm struct {
	db *gorm.DB
}

// NewUnitOfWork creates a GORM-backed UnitOfWork.
func NewUnitOfWork(db *gorm.DB) UnitOfWork {
	return &unitOfWorkGorm{db: db}
}

// Do runs fn inside a single transaction.
func (u *unitOfWorkGorm) Do(ctx context.Context, fn func(repos Repositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(Repositories{
			Company: NewCompanyRepository(tx),
			User:    NewUserRepository(tx),
		})
	})
}
