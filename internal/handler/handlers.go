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

	// HR groups employees, positions, departments and leave. Payroll is kept
	// separate because its routes sit at a different permission level.
	HR *HRHandlers

	// Payroll covers 급여 and 4대보험. Reads are RequireWriter, not merely
	// authenticated: salary figures are personnel data.
	Payroll *PayrollHandlers

	// Inventory handlers are held individually; router.NewInventoryHandlers
	// assembles them, so this package does not depend on the router.
	Product *ProductHandler
	Stock   *StockHandler
	Order   *OrderHandler
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
	productRepo := repository.NewProductRepositoryGorm(db)
	stockRepo := repository.NewStockRepositoryGorm(db)
	orderRepo := repository.NewOrderRepositoryGorm(db)

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
	productService := service.NewProductService(productRepo)
	stockService := service.NewStockService(stockRepo, productRepo)
	orderService := service.NewOrderService(orderRepo, stockRepo, productRepo)

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
		HR:         NewHRHandlers(db, logger),
		Payroll:    NewPayrollHandlers(db),
		Product:    NewProductHandler(productService),
		Stock:      NewStockHandler(stockService),
		Order:      NewOrderHandler(orderService),
	}
}
