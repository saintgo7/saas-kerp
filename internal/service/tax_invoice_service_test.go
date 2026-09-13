package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// fakeTaxInvoiceRepo is a hand-written stand-in for TaxInvoiceRepository. It
// records whether the invoice was written, which is what the fail-closed tests
// below actually assert: nothing may be persisted when the transmission did
// not happen.
type fakeTaxInvoiceRepo struct {
	invoice *domain.TaxInvoice
	items   []*domain.TaxInvoiceItem

	updateCalls  int
	historyCalls int
	createCalls  int
}

func (f *fakeTaxInvoiceRepo) Create(ctx context.Context, invoice *domain.TaxInvoice) error {
	f.createCalls++
	f.invoice = invoice
	return nil
}

func (f *fakeTaxInvoiceRepo) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.TaxInvoice, error) {
	if f.invoice == nil {
		return nil, domain.ErrTaxInvoiceNotFound
	}
	clone := *f.invoice
	return &clone, nil
}

func (f *fakeTaxInvoiceRepo) GetByNumber(ctx context.Context, companyID uuid.UUID, number string, invoiceType domain.TaxInvoiceType) (*domain.TaxInvoice, error) {
	return nil, domain.ErrTaxInvoiceNotFound
}

func (f *fakeTaxInvoiceRepo) List(ctx context.Context, filter *repository.TaxInvoiceFilter) ([]*domain.TaxInvoice, int64, error) {
	return nil, 0, nil
}

func (f *fakeTaxInvoiceRepo) Update(ctx context.Context, invoice *domain.TaxInvoice) error {
	f.updateCalls++
	clone := *invoice
	f.invoice = &clone
	return nil
}

func (f *fakeTaxInvoiceRepo) UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.TaxInvoiceStatus, userID *uuid.UUID) error {
	return nil
}

func (f *fakeTaxInvoiceRepo) Delete(ctx context.Context, companyID, id uuid.UUID) error { return nil }

func (f *fakeTaxInvoiceRepo) CreateItem(ctx context.Context, item *domain.TaxInvoiceItem) error {
	f.items = append(f.items, item)
	return nil
}

func (f *fakeTaxInvoiceRepo) ListItems(ctx context.Context, companyID, invoiceID uuid.UUID) ([]*domain.TaxInvoiceItem, error) {
	return f.items, nil
}

func (f *fakeTaxInvoiceRepo) DeleteItems(ctx context.Context, companyID, invoiceID uuid.UUID) error {
	f.items = nil
	return nil
}

func (f *fakeTaxInvoiceRepo) CreateHistory(ctx context.Context, history *domain.TaxInvoiceHistory) error {
	f.historyCalls++
	return nil
}

func (f *fakeTaxInvoiceRepo) ListHistory(ctx context.Context, companyID, invoiceID uuid.UUID) ([]*domain.TaxInvoiceHistory, error) {
	return nil, nil
}

func (f *fakeTaxInvoiceRepo) GetSummary(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.TaxInvoiceSummary, error) {
	return &domain.TaxInvoiceSummary{}, nil
}

func (f *fakeTaxInvoiceRepo) WithTransaction(ctx context.Context, fn func(repo repository.TaxInvoiceRepository) error) error {
	return fn(f)
}

var _ repository.TaxInvoiceRepository = (*fakeTaxInvoiceRepo)(nil)

// Valid 사업자등록번호 (check digit verified in domain tests).
const (
	svcSupplierBRN = "1234567891"
	svcBuyerBRN    = "2148159244"
)

func issuedInvoice(companyID uuid.UUID) *domain.TaxInvoice {
	return &domain.TaxInvoice{
		ID:                     uuid.New(),
		CompanyID:              companyID,
		InvoiceNumber:          "20260907-0001",
		InvoiceType:            domain.TaxInvoiceTypeSales,
		IssueDate:              time.Now(),
		Status:                 domain.TaxInvoiceStatusIssued,
		SupplierBusinessNumber: svcSupplierBRN,
		SupplierName:           "공급자 주식회사",
		BuyerBusinessNumber:    svcBuyerBRN,
		BuyerName:              "공급받는자 주식회사",
		SupplyAmount:           10_000_000,
		TaxAmount:              1_000_000,
		TotalAmount:            11_000_000,
	}
}

// ============================================================================
// TransmitToNTS must fail closed
// ============================================================================

func TestTaxInvoiceService_TransmitToNTS_FailsClosed(t *testing.T) {
	companyID := newTestCompanyID()

	t.Run("gRPC 클라이언트가 없으면 전송됨으로 기록하지 않는다", func(t *testing.T) {
		// This is the fail-open the HTTP layer refused to expose a route for:
		// with no client the invoice used to be marked transmitted although
		// nothing was ever sent to the NTS, so the operator believed the
		// filing was done.
		repo := &fakeTaxInvoiceRepo{invoice: issuedInvoice(companyID)}
		svc := service.NewTaxInvoiceService(repo, nil)

		_, err := svc.TransmitToNTS(context.Background(), companyID, repo.invoice.ID, "session", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
		assert.Equal(t, domain.TaxInvoiceStatusIssued, repo.invoice.Status,
			"상태가 transmitted 로 바뀌면 안 된다")
		assert.Empty(t, repo.invoice.NTSConfirmNumber)
		assert.Nil(t, repo.invoice.NTSTransmittedAt)
		assert.Zero(t, repo.updateCalls, "아무것도 저장되면 안 된다")
		assert.Zero(t, repo.historyCalls)
	})

	t.Run("이미 승인번호가 있으면 재전송하지 않는다", func(t *testing.T) {
		inv := issuedInvoice(companyID)
		inv.NTSConfirmNumber = "NTS-2026-0001"
		repo := &fakeTaxInvoiceRepo{invoice: inv}
		svc := service.NewTaxInvoiceService(repo, nil)

		_, err := svc.TransmitToNTS(context.Background(), companyID, inv.ID, "session", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already been transmitted")
		assert.Zero(t, repo.updateCalls)
	})

	t.Run("발급되지 않은 계산서는 전송하지 않는다", func(t *testing.T) {
		inv := issuedInvoice(companyID)
		inv.Status = domain.TaxInvoiceStatusDraft
		repo := &fakeTaxInvoiceRepo{invoice: inv}
		svc := service.NewTaxInvoiceService(repo, nil)

		_, err := svc.TransmitToNTS(context.Background(), companyID, inv.ID, "session", nil)

		require.Error(t, err)
		assert.Zero(t, repo.updateCalls)
	})
}

// ============================================================================
// Issue re-validates the stored document
// ============================================================================

func TestTaxInvoiceService_Issue_ValidatesStoredItems(t *testing.T) {
	companyID := newTestCompanyID()

	t.Run("품목 합계가 헤더와 다르면 발급을 차단한다", func(t *testing.T) {
		inv := issuedInvoice(companyID)
		inv.Status = domain.TaxInvoiceStatusDraft

		repo := &fakeTaxInvoiceRepo{
			invoice: inv,
			items: []*domain.TaxInvoiceItem{
				// 헤더는 공급가액 10,000,000 / 세액 1,000,000 인데
				// 품목은 9,000,000 밖에 없다.
				{Description: "품목1", Amount: 9_000_000, TaxAmount: 1_000_000},
			},
		}
		svc := service.NewTaxInvoiceService(repo, nil)

		_, err := svc.Issue(context.Background(), companyID, inv.ID, nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrTaxInvoiceItemSumMismatch)
		assert.Equal(t, domain.TaxInvoiceStatusDraft, repo.invoice.Status)
		assert.Zero(t, repo.updateCalls)
	})

	t.Run("품목이 헤더와 맞으면 발급된다", func(t *testing.T) {
		inv := issuedInvoice(companyID)
		inv.Status = domain.TaxInvoiceStatusDraft

		repo := &fakeTaxInvoiceRepo{
			invoice: inv,
			items: []*domain.TaxInvoiceItem{
				{Description: "품목1", Amount: 6_000_000, TaxAmount: 600_000},
				{Description: "품목2", Amount: 4_000_000, TaxAmount: 400_000},
			},
		}
		svc := service.NewTaxInvoiceService(repo, nil)

		out, err := svc.Issue(context.Background(), companyID, inv.ID, nil)

		require.NoError(t, err)
		assert.Equal(t, domain.TaxInvoiceStatusIssued, out.Status)
		assert.Equal(t, 1, repo.updateCalls)
		assert.Equal(t, 1, repo.historyCalls, "이력 기록 실패는 더 이상 무시되지 않는다")
	})
}

// ============================================================================
// Create reconciles header and items before anything is written
// ============================================================================

func TestTaxInvoiceService_Create_ValidatesAmounts(t *testing.T) {
	companyID := newTestCompanyID()

	baseInput := func() *service.CreateInput {
		return &service.CreateInput{
			InvoiceNumber:          "20260907-0002",
			InvoiceType:            domain.TaxInvoiceTypeSales,
			IssueDate:              time.Now(),
			SupplierBusinessNumber: svcSupplierBRN,
			SupplierName:           "공급자 주식회사",
			BuyerBusinessNumber:    svcBuyerBRN,
			BuyerName:              "공급받는자 주식회사",
			SupplyAmount:           10_000_000,
			TaxAmount:              1_000_000,
			Items: []service.CreateItemInput{
				{Description: "품목1", Amount: 10_000_000, TaxAmount: 1_000_000},
			},
		}
	}

	t.Run("정상 건은 헤더·품목·이력이 함께 저장된다", func(t *testing.T) {
		repo := &fakeTaxInvoiceRepo{}
		svc := service.NewTaxInvoiceService(repo, nil)

		out, err := svc.Create(context.Background(), companyID, baseInput(), nil)

		require.NoError(t, err)
		assert.Equal(t, domain.TaxInvoiceStatusDraft, out.Status)
		assert.Equal(t, 1, repo.createCalls)
		assert.Len(t, repo.items, 1)
		assert.Equal(t, 1, repo.historyCalls)
	})

	t.Run("과세인데 세액 0 이면 거부한다", func(t *testing.T) {
		// 감사 보고서의 과소신고 사례: 헤더는 세액 0, 품목은 1,000,000.
		repo := &fakeTaxInvoiceRepo{}
		svc := service.NewTaxInvoiceService(repo, nil)

		input := baseInput()
		input.TaxAmount = 0

		_, err := svc.Create(context.Background(), companyID, input, nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrTaxInvoiceItemSumMismatch)
		assert.Zero(t, repo.createCalls, "검증 실패 시 아무것도 저장되면 안 된다")
		assert.Empty(t, repo.items)
		assert.Zero(t, repo.historyCalls)
	})

	t.Run("음수 금액을 거부한다", func(t *testing.T) {
		repo := &fakeTaxInvoiceRepo{}
		svc := service.NewTaxInvoiceService(repo, nil)

		input := baseInput()
		input.SupplyAmount = -10_000_000
		input.TaxAmount = -1_000_000

		_, err := svc.Create(context.Background(), companyID, input, nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrTaxInvoiceNegativeAmount)
		assert.Zero(t, repo.createCalls)
	})

	t.Run("사업자등록번호 체크디지트가 틀리면 거부한다", func(t *testing.T) {
		repo := &fakeTaxInvoiceRepo{}
		svc := service.NewTaxInvoiceService(repo, nil)

		input := baseInput()
		input.BuyerBusinessNumber = "abcdefghij"

		_, err := svc.Create(context.Background(), companyID, input, nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidBusinessNumber)
		assert.Zero(t, repo.createCalls)
	})
}
