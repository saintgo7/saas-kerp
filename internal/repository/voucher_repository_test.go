//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// VoucherRepositoryTestSuite is an integration test suite for VoucherRepository
type VoucherRepositoryTestSuite struct {
	suite.Suite
	container   testcontainers.Container
	db          *gorm.DB
	repo        repository.VoucherRepository
	accountRepo repository.AccountRepository
	companyID   uuid.UUID
	accountID1  uuid.UUID
	accountID2  uuid.UUID
}

// SetupSuite runs once before all tests
func (s *VoucherRepositoryTestSuite) SetupSuite() {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(s.T(), err)
	s.container = pgContainer

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(s.T(), err)

	// Connect to database
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	require.NoError(s.T(), err)
	s.db = db

	// Run migrations
	err = s.db.AutoMigrate(
		&domain.Account{},
		&domain.Voucher{},
		&domain.VoucherEntry{},
	)
	require.NoError(s.T(), err)

	// Create required extensions
	s.db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	// Create test company ID
	s.companyID = uuid.New()

	// Create test accounts
	s.accountID1 = uuid.New()
	s.accountID2 = uuid.New()

	account1 := &domain.Account{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: s.accountID1},
			CompanyID: s.companyID,
		},
		Code:               "101",
		Name:               "Cash",
		AccountType:        domain.AccountTypeAsset,
		AccountNature:      domain.AccountNatureDebit,
		IsActive:           true,
		AllowDirectPosting: true,
	}
	account2 := &domain.Account{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: s.accountID2},
			CompanyID: s.companyID,
		},
		Code:               "401",
		Name:               "Sales Revenue",
		AccountType:        domain.AccountTypeRevenue,
		AccountNature:      domain.AccountNatureCredit,
		IsActive:           true,
		AllowDirectPosting: true,
	}

	err = s.db.Create(account1).Error
	require.NoError(s.T(), err)
	err = s.db.Create(account2).Error
	require.NoError(s.T(), err)

	// Create repository
	s.repo = repository.NewVoucherRepository(s.db)
}

// TearDownSuite runs once after all tests
func (s *VoucherRepositoryTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Terminate(context.Background())
	}
}

// SetupTest runs before each test
func (s *VoucherRepositoryTestSuite) SetupTest() {
	// Clean voucher tables before each test
	s.db.Exec("DELETE FROM voucher_entries")
	s.db.Exec("DELETE FROM vouchers")
}

// Helper to create a test voucher
func (s *VoucherRepositoryTestSuite) newTestVoucher() *domain.Voucher {
	return &domain.Voucher{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: uuid.New()},
			CompanyID: s.companyID,
		},
		VoucherNo:   "TEST-2024-000001",
		VoucherDate: time.Now().Truncate(24 * time.Hour),
		VoucherType: domain.VoucherTypeGeneral,
		Status:      domain.VoucherStatusDraft,
		Description: "Test voucher",
		TotalDebit:  1000,
		TotalCredit: 1000,
		Entries: []domain.VoucherEntry{
			{
				TenantModel: domain.TenantModel{
					BaseModel: domain.BaseModel{ID: uuid.New()},
					CompanyID: s.companyID,
				},
				AccountID:    s.accountID1,
				LineNo:       1,
				DebitAmount:  1000,
				CreditAmount: 0,
				Description:  "Debit entry",
			},
			{
				TenantModel: domain.TenantModel{
					BaseModel: domain.BaseModel{ID: uuid.New()},
					CompanyID: s.companyID,
				},
				AccountID:    s.accountID2,
				LineNo:       2,
				DebitAmount:  0,
				CreditAmount: 1000,
				Description:  "Credit entry",
			},
		},
	}
}

// ============================================================================
// Create Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestCreate_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()

	err := s.repo.Create(ctx, voucher)

	s.NoError(err)
	s.NotEqual(uuid.Nil, voucher.ID)

	// Verify in database
	var count int64
	s.db.Model(&domain.Voucher{}).Where("id = ?", voucher.ID).Count(&count)
	s.Equal(int64(1), count)

	// Verify entries created
	var entryCount int64
	s.db.Model(&domain.VoucherEntry{}).Where("voucher_id = ?", voucher.ID).Count(&entryCount)
	s.Equal(int64(2), entryCount)
}

func (s *VoucherRepositoryTestSuite) TestCreate_WithEmptyEntries() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	voucher.Entries = []domain.VoucherEntry{}

	err := s.repo.Create(ctx, voucher)

	s.NoError(err)

	// Verify no entries created
	var entryCount int64
	s.db.Model(&domain.VoucherEntry{}).Where("voucher_id = ?", voucher.ID).Count(&entryCount)
	s.Equal(int64(0), entryCount)
}

// ============================================================================
// FindByID Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestFindByID_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	s.repo.Create(ctx, voucher)

	result, err := s.repo.FindByID(ctx, s.companyID, voucher.ID)

	s.NoError(err)
	s.NotNil(result)
	s.Equal(voucher.ID, result.ID)
	s.Equal(voucher.VoucherNo, result.VoucherNo)
	s.Len(result.Entries, 2)
}

func (s *VoucherRepositoryTestSuite) TestFindByID_NotFound() {
	ctx := context.Background()
	randomID := uuid.New()

	result, err := s.repo.FindByID(ctx, s.companyID, randomID)

	s.Error(err)
	s.Nil(result)
	s.Equal(domain.ErrVoucherNotFound, err)
}

func (s *VoucherRepositoryTestSuite) TestFindByID_WrongCompanyID() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	s.repo.Create(ctx, voucher)

	// Try to find with different company ID
	differentCompanyID := uuid.New()
	result, err := s.repo.FindByID(ctx, differentCompanyID, voucher.ID)

	s.Error(err)
	s.Nil(result)
	s.Equal(domain.ErrVoucherNotFound, err)
}

// ============================================================================
// FindByNo Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestFindByNo_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	s.repo.Create(ctx, voucher)

	result, err := s.repo.FindByNo(ctx, s.companyID, voucher.VoucherNo)

	s.NoError(err)
	s.NotNil(result)
	s.Equal(voucher.ID, result.ID)
}

func (s *VoucherRepositoryTestSuite) TestFindByNo_NotFound() {
	ctx := context.Background()

	result, err := s.repo.FindByNo(ctx, s.companyID, "NONEXISTENT-001")

	s.Error(err)
	s.Nil(result)
	s.Equal(domain.ErrVoucherNotFound, err)
}

// ============================================================================
// Update Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestUpdate_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	s.repo.Create(ctx, voucher)

	// Update voucher
	voucher.Description = "Updated description"
	err := s.repo.Update(ctx, voucher)

	s.NoError(err)

	// Verify update
	result, _ := s.repo.FindByID(ctx, s.companyID, voucher.ID)
	s.Equal("Updated description", result.Description)
}

// ============================================================================
// Delete Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestDelete_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	s.repo.Create(ctx, voucher)

	err := s.repo.Delete(ctx, s.companyID, voucher.ID)

	s.NoError(err)

	// Verify deletion
	_, err = s.repo.FindByID(ctx, s.companyID, voucher.ID)
	s.Equal(domain.ErrVoucherNotFound, err)

	// Verify entries also deleted
	var entryCount int64
	s.db.Model(&domain.VoucherEntry{}).Where("voucher_id = ?", voucher.ID).Count(&entryCount)
	s.Equal(int64(0), entryCount)
}

// ============================================================================
// FindAll Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestFindAll_WithPagination() {
	ctx := context.Background()

	// Create 5 vouchers
	for i := 0; i < 5; i++ {
		voucher := s.newTestVoucher()
		voucher.VoucherNo = fmt.Sprintf("TEST-2024-%06d", i+1)
		s.repo.Create(ctx, voucher)
	}

	filter := repository.VoucherFilter{
		CompanyID: s.companyID,
		Page:      1,
		PageSize:  2,
	}

	results, total, err := s.repo.FindAll(ctx, filter)

	s.NoError(err)
	s.Len(results, 2)
	s.Equal(int64(5), total)
}

func (s *VoucherRepositoryTestSuite) TestFindAll_FilterByStatus() {
	ctx := context.Background()

	// Create vouchers with different statuses
	draftVoucher := s.newTestVoucher()
	draftVoucher.VoucherNo = "DRAFT-001"
	draftVoucher.Status = domain.VoucherStatusDraft
	s.repo.Create(ctx, draftVoucher)

	pendingVoucher := s.newTestVoucher()
	pendingVoucher.VoucherNo = "PENDING-001"
	pendingVoucher.Status = domain.VoucherStatusPending
	s.repo.Create(ctx, pendingVoucher)

	statusFilter := domain.VoucherStatusDraft
	filter := repository.VoucherFilter{
		CompanyID: s.companyID,
		Status:    &statusFilter,
	}

	results, total, err := s.repo.FindAll(ctx, filter)

	s.NoError(err)
	s.Len(results, 1)
	s.Equal(int64(1), total)
	s.Equal(domain.VoucherStatusDraft, results[0].Status)
}

func (s *VoucherRepositoryTestSuite) TestFindAll_FilterByDateRange() {
	ctx := context.Background()

	// Create voucher with specific date
	voucher := s.newTestVoucher()
	voucher.VoucherDate = time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	s.repo.Create(ctx, voucher)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

	filter := repository.VoucherFilter{
		CompanyID: s.companyID,
		DateFrom:  &from,
		DateTo:    &to,
	}

	results, total, err := s.repo.FindAll(ctx, filter)

	s.NoError(err)
	s.Len(results, 1)
	s.Equal(int64(1), total)
}

// ============================================================================
// FindByDateRange Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestFindByDateRange_Success() {
	ctx := context.Background()

	// Create vouchers on different dates
	voucher1 := s.newTestVoucher()
	voucher1.VoucherNo = "DATE-001"
	voucher1.VoucherDate = time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	s.repo.Create(ctx, voucher1)

	voucher2 := s.newTestVoucher()
	voucher2.VoucherNo = "DATE-002"
	voucher2.VoucherDate = time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)
	s.repo.Create(ctx, voucher2)

	voucher3 := s.newTestVoucher()
	voucher3.VoucherNo = "DATE-003"
	voucher3.VoucherDate = time.Date(2024, 7, 5, 0, 0, 0, 0, time.UTC)
	s.repo.Create(ctx, voucher3)

	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

	results, err := s.repo.FindByDateRange(ctx, s.companyID, from, to)

	s.NoError(err)
	s.Len(results, 2)
}

// ============================================================================
// FindByStatus Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestFindByStatus_Success() {
	ctx := context.Background()

	// Create voucher with pending status
	pendingVoucher := s.newTestVoucher()
	pendingVoucher.VoucherNo = "STATUS-001"
	pendingVoucher.Status = domain.VoucherStatusPending
	s.repo.Create(ctx, pendingVoucher)

	results, err := s.repo.FindByStatus(ctx, s.companyID, domain.VoucherStatusPending)

	s.NoError(err)
	s.Len(results, 1)
	s.Equal(domain.VoucherStatusPending, results[0].Status)
}

// ============================================================================
// Entry Operations Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestCreateEntry_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	voucher.Entries = []domain.VoucherEntry{} // Empty entries
	s.repo.Create(ctx, voucher)

	entry := &domain.VoucherEntry{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: uuid.New()},
			CompanyID: s.companyID,
		},
		VoucherID:    voucher.ID,
		AccountID:    s.accountID1,
		LineNo:       1,
		DebitAmount:  500,
		CreditAmount: 0,
		Description:  "New entry",
	}

	err := s.repo.CreateEntry(ctx, entry)

	s.NoError(err)

	// Verify entry created
	entries, _ := s.repo.FindEntriesByVoucher(ctx, voucher.ID)
	s.Len(entries, 1)
}

func (s *VoucherRepositoryTestSuite) TestDeleteEntriesByVoucher_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	s.repo.Create(ctx, voucher)

	err := s.repo.DeleteEntriesByVoucher(ctx, voucher.ID)

	s.NoError(err)

	// Verify entries deleted
	entries, _ := s.repo.FindEntriesByVoucher(ctx, voucher.ID)
	s.Len(entries, 0)
}

// ============================================================================
// UpdateStatus Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestUpdateStatus_Success() {
	ctx := context.Background()
	voucher := s.newTestVoucher()
	s.repo.Create(ctx, voucher)

	// Update status to pending
	userID := uuid.New()
	now := time.Now()
	voucher.Status = domain.VoucherStatusPending
	voucher.SubmittedAt = &now
	voucher.SubmittedBy = &userID

	err := s.repo.UpdateStatus(ctx, voucher)

	s.NoError(err)

	// Verify status updated
	result, _ := s.repo.FindByID(ctx, s.companyID, voucher.ID)
	s.Equal(domain.VoucherStatusPending, result.Status)
	s.NotNil(result.SubmittedAt)
	s.NotNil(result.SubmittedBy)
}

// ============================================================================
// WithTransaction Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestWithTransaction_CommitOnSuccess() {
	ctx := context.Background()

	var createdVoucherID uuid.UUID

	err := s.repo.WithTransaction(ctx, func(txRepo repository.VoucherRepository) error {
		voucher := s.newTestVoucher()
		err := txRepo.Create(ctx, voucher)
		if err != nil {
			return err
		}
		createdVoucherID = voucher.ID
		return nil
	})

	s.NoError(err)

	// Voucher should exist
	_, err = s.repo.FindByID(ctx, s.companyID, createdVoucherID)
	s.NoError(err)
}

func (s *VoucherRepositoryTestSuite) TestWithTransaction_RollbackOnError() {
	ctx := context.Background()

	var createdVoucherID uuid.UUID

	err := s.repo.WithTransaction(ctx, func(txRepo repository.VoucherRepository) error {
		voucher := s.newTestVoucher()
		err := txRepo.Create(ctx, voucher)
		if err != nil {
			return err
		}
		createdVoucherID = voucher.ID

		// Return error to trigger rollback
		return fmt.Errorf("intentional error")
	})

	s.Error(err)

	// Voucher should NOT exist due to rollback
	_, err = s.repo.FindByID(ctx, s.companyID, createdVoucherID)
	s.Equal(domain.ErrVoucherNotFound, err)
}

// ============================================================================
// Multi-Tenancy Isolation Tests
// ============================================================================

func (s *VoucherRepositoryTestSuite) TestMultiTenancy_IsolatesData() {
	ctx := context.Background()

	// Create voucher for company 1
	voucher1 := s.newTestVoucher()
	voucher1.VoucherNo = "COMPANY1-001"
	s.repo.Create(ctx, voucher1)

	// Create voucher for different company
	company2ID := uuid.New()
	voucher2 := &domain.Voucher{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{ID: uuid.New()},
			CompanyID: company2ID,
		},
		VoucherNo:   "COMPANY2-001",
		VoucherDate: time.Now(),
		VoucherType: domain.VoucherTypeGeneral,
		Status:      domain.VoucherStatusDraft,
	}
	s.db.Create(voucher2)

	// Query for company 1 should only return company 1's vouchers
	filter := repository.VoucherFilter{CompanyID: s.companyID}
	results, total, err := s.repo.FindAll(ctx, filter)

	s.NoError(err)
	s.Len(results, 1)
	s.Equal(int64(1), total)
	s.Equal("COMPANY1-001", results[0].VoucherNo)
}

// Run the test suite
func TestVoucherRepositorySuite(t *testing.T) {
	suite.Run(t, new(VoucherRepositoryTestSuite))
}
