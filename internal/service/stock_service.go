package service

import (
	"context"
	"math"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// Filters re-exported from the repository so handlers do not import it.
type (
	StockFilter    = repository.StockFilter
	MovementFilter = repository.MovementFilter
	StockSummary   = repository.StockSummary
)

// AdjustStockInput is a manual correction of one balance.
type AdjustStockInput struct {
	CompanyID   uuid.UUID
	ProductID   uuid.UUID
	WarehouseID uuid.UUID
	UserID      uuid.UUID

	// TargetQuantity is the balance the operator wants the warehouse to show.
	// The service computes the delta from the current balance rather than
	// taking one, so the client never has to know what the balance was - and
	// cannot get it wrong by working from a stale screen.
	TargetQuantity float64
	Reason         string
}

// TransferStockInput moves stock between two warehouses of the same tenant.
type TransferStockInput struct {
	CompanyID       uuid.UUID
	ProductID       uuid.UUID
	FromWarehouseID uuid.UUID
	ToWarehouseID   uuid.UUID
	Quantity        float64
	UserID          uuid.UUID
	Note            string
}

// StockService is the business logic for stock balances and movements.
type StockService interface {
	List(ctx context.Context, filter *StockFilter) ([]domain.Stock, int64, error)
	Get(ctx context.Context, companyID, productID, warehouseID uuid.UUID) (*domain.Stock, error)
	ListMovements(ctx context.Context, filter *MovementFilter) ([]domain.StockMovement, int64, error)

	// Adjust books the difference between the current and the requested balance
	// as an adjustment movement. It never assigns the balance.
	Adjust(ctx context.Context, in *AdjustStockInput) (*domain.StockMovement, error)

	// Transfer books an outbound and an inbound movement in one transaction.
	Transfer(ctx context.Context, in *TransferStockInput) error

	// ListAlerts derives the threshold breaches. See domain.StockAlert for why
	// there is no alert table.
	ListAlerts(ctx context.Context, companyID uuid.UUID, alertType string) ([]domain.StockAlert, error)

	GetSummary(ctx context.Context, companyID uuid.UUID) (*StockSummary, error)
}

// stockService implements StockService.
type stockService struct {
	stockRepo   repository.StockRepository
	productRepo repository.ProductRepository
}

// NewStockService creates a StockService.
func NewStockService(stockRepo repository.StockRepository, productRepo repository.ProductRepository) StockService {
	return &stockService{stockRepo: stockRepo, productRepo: productRepo}
}

// List retrieves a page of balances.
func (s *stockService) List(ctx context.Context, filter *StockFilter) ([]domain.Stock, int64, error) {
	return s.stockRepo.List(ctx, filter)
}

// Get retrieves one balance.
func (s *stockService) Get(ctx context.Context, companyID, productID, warehouseID uuid.UUID) (*domain.Stock, error) {
	return s.stockRepo.Get(ctx, companyID, productID, warehouseID)
}

// ListMovements retrieves a page of the ledger.
func (s *stockService) ListMovements(ctx context.Context, filter *MovementFilter) ([]domain.StockMovement, int64, error) {
	return s.stockRepo.ListMovements(ctx, filter)
}

// quantityEpsilon is the tolerance below which a computed delta counts as zero.
//
// Quantities are NUMERIC(18,3) in the database and float64 in Go, so a target
// that equals the current balance can come back as a difference of 1e-16.
// Without this, such a request would post a movement of "0.0000000000000001"
// and be rejected by chk_stock_movements_quantity_positive with a 500. Half of
// the smallest storable unit is the natural threshold.
const quantityEpsilon = 0.0005

// Adjust books a manual correction.
func (s *stockService) Adjust(ctx context.Context, in *AdjustStockInput) (*domain.StockMovement, error) {
	if in.TargetQuantity < 0 {
		return nil, domain.ErrStockNegativeResult
	}

	// The product and the warehouse must both belong to the caller's tenant.
	// The RLS policies would refuse a cross-tenant write anyway, but the failure
	// would arrive as a foreign-key or policy error rather than a 404.
	if _, err := s.productRepo.GetByID(ctx, in.CompanyID, in.ProductID); err != nil {
		return nil, err
	}
	if _, err := s.productRepo.GetWarehouseByID(ctx, in.CompanyID, in.WarehouseID); err != nil {
		return nil, err
	}

	var movement *domain.StockMovement

	// The read of the current balance and the movement that corrects it must be
	// one unit of work: between a read outside the transaction and the write
	// inside it, another request can post a receipt, and the adjustment would
	// then silently undo it.
	err := s.stockRepo.WithTransaction(ctx, func(repo repository.StockRepository) error {
		current := 0.0
		stock, err := repo.Get(ctx, in.CompanyID, in.ProductID, in.WarehouseID)
		switch {
		case err == nil:
			current = stock.Quantity
		case err == domain.ErrStockNotFound:
			// No balance row yet: the current quantity is zero and
			// ApplyMovement will create the row.
		default:
			return err
		}

		delta := in.TargetQuantity - current
		if math.Abs(delta) < quantityEpsilon {
			return domain.ErrOrderNothingToPost
		}

		movementType := domain.MovementAdjustmentIn
		if delta < 0 {
			movementType = domain.MovementAdjustmentOut
		}

		movement = &domain.StockMovement{
			CompanyID:     in.CompanyID,
			ProductID:     in.ProductID,
			WarehouseID:   in.WarehouseID,
			MovementType:  movementType,
			Quantity:      math.Abs(delta),
			ReferenceType: domain.StockRefStockAdjustment,
			Note:          in.Reason,
			CreatedBy:     in.UserID,
		}
		return repo.ApplyMovement(ctx, movement)
	})
	if err != nil {
		return nil, err
	}
	return movement, nil
}

// Transfer moves stock between two warehouses.
func (s *stockService) Transfer(ctx context.Context, in *TransferStockInput) error {
	if in.Quantity <= 0 {
		return domain.ErrStockQuantityNotPositive
	}
	if in.FromWarehouseID == in.ToWarehouseID {
		return domain.ErrStockSameWarehouse
	}

	if _, err := s.productRepo.GetByID(ctx, in.CompanyID, in.ProductID); err != nil {
		return err
	}
	if _, err := s.productRepo.GetWarehouseByID(ctx, in.CompanyID, in.FromWarehouseID); err != nil {
		return err
	}
	if _, err := s.productRepo.GetWarehouseByID(ctx, in.CompanyID, in.ToWarehouseID); err != nil {
		return err
	}

	// Both legs commit together. A transfer that booked the issue and then
	// failed the receipt would destroy stock outright.
	return s.stockRepo.WithTransaction(ctx, func(repo repository.StockRepository) error {
		out := &domain.StockMovement{
			CompanyID:     in.CompanyID,
			ProductID:     in.ProductID,
			WarehouseID:   in.FromWarehouseID,
			MovementType:  domain.MovementTransferOut,
			Quantity:      in.Quantity,
			ReferenceType: domain.StockRefStockTransfer,
			Note:          in.Note,
			CreatedBy:     in.UserID,
		}
		if err := repo.ApplyMovement(ctx, out); err != nil {
			return err
		}

		into := &domain.StockMovement{
			CompanyID:     in.CompanyID,
			ProductID:     in.ProductID,
			WarehouseID:   in.ToWarehouseID,
			MovementType:  domain.MovementTransferIn,
			Quantity:      in.Quantity,
			ReferenceType: domain.StockRefStockTransfer,
			ReferenceID:   &out.ID,
			Note:          in.Note,
			CreatedBy:     in.UserID,
		}
		return repo.ApplyMovement(ctx, into)
	})
}

// ListAlerts derives the threshold breaches for the tenant.
func (s *stockService) ListAlerts(ctx context.Context, companyID uuid.UUID, alertType string) ([]domain.StockAlert, error) {
	candidates, err := s.stockRepo.ListAlertCandidates(ctx, companyID)
	if err != nil {
		return nil, err
	}

	alerts := make([]domain.StockAlert, 0, len(candidates))
	for i := range candidates {
		alert := domain.DeriveStockAlert(&candidates[i])
		if alert == nil {
			continue
		}
		if alertType != "" && string(alert.AlertType) != alertType {
			continue
		}
		alerts = append(alerts, *alert)
	}
	return alerts, nil
}

// GetSummary returns the stock dashboard aggregate.
func (s *stockService) GetSummary(ctx context.Context, companyID uuid.UUID) (*StockSummary, error) {
	return s.stockRepo.Summary(ctx, companyID)
}
