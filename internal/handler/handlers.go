package handler

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/grpcclient"
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

	// TaxInvoice reaches the NTS scraper over gRPC. Its transmit path is
	// fail-closed: nothing is recorded as transmitted unless the scraper
	// returns success with a 승인번호.
	TaxInvoice *TaxInvoiceHandler
}

// envDevelopment is the only environment in which handlers may return
// development-only content (such as a password reset token) in a response.
const envDevelopment = "development"

// NewHandlers creates all handlers
func NewHandlers(db *gorm.DB, redis *redis.Client, logger *zap.Logger, jwtService *auth.JWTService, version, env string) *Handlers {
	// Initialize repositories
	partnerRepo := repository.NewPartnerRepositoryGorm(db)
	voucherRepo := repository.NewVoucherRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	taxInvoiceRepo := repository.NewTaxInvoiceRepositoryGorm(db)

	// Initialize services
	partnerService := service.NewPartnerService(partnerRepo)
	accountService := service.NewAccountService(accountRepo)
	voucherService := service.NewVoucherService(voucherRepo, accountRepo, ledgerRepo)
	ledgerService := service.NewLedgerService(ledgerRepo, accountRepo)
	userService := service.NewUserService(userRepo)
	roleService := service.NewRoleService(roleRepo)
	companyService := service.NewCompanyService(companyRepo)
	projectService := service.NewProjectService(projectRepo)
	taxInvoiceClient := grpcclient.NewTaxInvoiceClient(grpcclient.NewManager(grpcclient.ConfigFromEnv(env)))
	taxInvoiceService := service.NewTaxInvoiceService(taxInvoiceRepo, taxInvoiceClient)

	return &Handlers{
		Health:     NewHealthHandler(db, redis, logger, version),
		Auth:       NewAuthHandler(db, redis, logger, jwtService, env),
		Partner:    NewPartnerHandler(partnerService),
		Voucher:    NewVoucherHandler(voucherService),
		Ledger:     NewLedgerHandler(ledgerService, accountService),
		Account:    NewAccountHandler(accountService),
		User:       NewUserHandler(userService),
		Role:       NewRoleHandler(roleService),
		Company:    NewCompanyHandler(companyService),
		Project:    NewProjectHandler(projectService),
		TaxInvoice: NewTaxInvoiceHandler(taxInvoiceService),
	}
}
