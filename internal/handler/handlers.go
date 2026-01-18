package handler

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	Health  *HealthHandler
	Auth    *AuthHandler
	Partner *PartnerHandler
	Voucher *VoucherHandler
	Ledger  *LedgerHandler
	Account *AccountHandler
	User    *UserHandler
	Role    *RoleHandler
	Company *CompanyHandler
	Project *ProjectHandler
}

// NewHandlers creates all handlers
func NewHandlers(db *gorm.DB, redis *redis.Client, logger *zap.Logger, jwtService *auth.JWTService, version string) *Handlers {
	// Initialize repositories
	partnerRepo := repository.NewPartnerRepositoryGorm(db)
	voucherRepo := repository.NewVoucherRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	projectRepo := repository.NewProjectRepository(db)

	// Initialize services
	partnerService := service.NewPartnerService(partnerRepo)
	accountService := service.NewAccountService(accountRepo)
	voucherService := service.NewVoucherService(voucherRepo, accountRepo)
	ledgerService := service.NewLedgerService(ledgerRepo, accountRepo)
	userService := service.NewUserService(userRepo)
	roleService := service.NewRoleService(roleRepo)
	companyService := service.NewCompanyService(companyRepo)
	projectService := service.NewProjectService(projectRepo)

	return &Handlers{
		Health:  NewHealthHandler(db, redis, logger, version),
		Auth:    NewAuthHandler(db, redis, logger, jwtService),
		Partner: NewPartnerHandler(partnerService),
		Voucher: NewVoucherHandler(voucherService),
		Ledger:  NewLedgerHandler(ledgerService, accountService),
		Account: NewAccountHandler(accountService),
		User:    NewUserHandler(userService),
		Role:    NewRoleHandler(roleService),
		Company: NewCompanyHandler(companyService),
		Project: NewProjectHandler(projectService),
	}
}
