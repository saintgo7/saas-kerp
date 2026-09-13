package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// Filters and summaries re-exported from the repository.
type (
	PurchaseOrderFilter  = repository.PurchaseOrderFilter
	SalesOrderFilter     = repository.SalesOrderFilter
	PurchaseOrderSummary = repository.PurchaseOrderSummary
	SalesOrderSummary    = repository.SalesOrderSummary
	PostLine             = repository.PostLine
)

// OrderLineInput is one line of an order as the client submits it.
//
// It carries no amounts. The frontend sends {productId, quantity, unitPrice}
// and the server prices the line, so a client cannot book an amount that does
// not follow from the quantity and the price it also sent.
type OrderLineInput struct {
	ProductID uuid.UUID
	Quantity  float64
	UnitPrice float64
	Note      string
}

// PurchaseOrderInput is a purchase order as the client submits it.
type PurchaseOrderInput struct {
	CompanyID    uuid.UUID
	UserID       uuid.UUID
	OrderDate    time.Time
	ExpectedDate *time.Time
	SupplierID   uuid.UUID
	WarehouseID  uuid.UUID
	Note         string
	Items        []OrderLineInput
}

// SalesOrderInput is a sales order as the client submits it.
type SalesOrderInput struct {
	CompanyID    uuid.UUID
	UserID       uuid.UUID
	OrderDate    time.Time
	ExpectedDate *time.Time
	CustomerID   uuid.UUID
	WarehouseID  uuid.UUID
	Note         string
	Items        []OrderLineInput
}

// OrderService is the business logic for purchase and sales orders.
//
// Every transition method is idempotent in the sense that matters: running it
// twice does not apply it twice. The second call finds the order already past
// the state it expected and returns a "cannot in current status" error rather
// than approving, cancelling or receiving a second time.
type OrderService interface {
	// --- purchase orders ---
	CreatePurchaseOrder(ctx context.Context, in *PurchaseOrderInput) (*domain.PurchaseOrder, error)
	UpdatePurchaseOrder(ctx context.Context, id uuid.UUID, in *PurchaseOrderInput) (*domain.PurchaseOrder, error)
	GetPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error)
	ListPurchaseOrders(ctx context.Context, filter *PurchaseOrderFilter) ([]domain.PurchaseOrder, int64, error)
	DeletePurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error

	SubmitPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error
	ApprovePurchaseOrder(ctx context.Context, companyID, id, userID uuid.UUID) error
	RejectPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error
	PlacePurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error
	CancelPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error

	// ReceivePurchaseOrder books goods in against the order, posting one stock
	// movement per line and moving the order to partial or completed.
	ReceivePurchaseOrder(ctx context.Context, companyID, id, userID uuid.UUID, lines []PostLine) (*domain.PurchaseOrder, error)

	GetPurchaseOrderSummary(ctx context.Context, companyID uuid.UUID) (*PurchaseOrderSummary, error)

	// --- sales orders ---
	CreateSalesOrder(ctx context.Context, in *SalesOrderInput) (*domain.SalesOrder, error)
	UpdateSalesOrder(ctx context.Context, id uuid.UUID, in *SalesOrderInput) (*domain.SalesOrder, error)
	GetSalesOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.SalesOrder, error)
	ListSalesOrders(ctx context.Context, filter *SalesOrderFilter) ([]domain.SalesOrder, int64, error)
	DeleteSalesOrder(ctx context.Context, companyID, id uuid.UUID) error

	SubmitSalesOrder(ctx context.Context, companyID, id uuid.UUID) error
	ApproveSalesOrder(ctx context.Context, companyID, id, userID uuid.UUID) error
	RejectSalesOrder(ctx context.Context, companyID, id uuid.UUID) error
	ConfirmSalesOrder(ctx context.Context, companyID, id uuid.UUID) error
	CancelSalesOrder(ctx context.Context, companyID, id uuid.UUID) error

	// ShipSalesOrder books goods out against the order.
	ShipSalesOrder(ctx context.Context, companyID, id, userID uuid.UUID, lines []PostLine) (*domain.SalesOrder, error)

	GetSalesOrderSummary(ctx context.Context, companyID uuid.UUID) (*SalesOrderSummary, error)
}

// orderService implements OrderService.
type orderService struct {
	orderRepo   repository.OrderRepository
	stockRepo   repository.StockRepository
	productRepo repository.ProductRepository
}

// NewOrderService creates an OrderService.
func NewOrderService(
	orderRepo repository.OrderRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		stockRepo:   stockRepo,
		productRepo: productRepo,
	}
}

// ---------------------------------------------------------------------------
// Shared line pricing
// ---------------------------------------------------------------------------

// pricedLines is the result of pricing a submitted line set.
type pricedLines struct {
	amounts     []int64
	taxAmounts  []int64
	totalAmount int64
	taxAmount   int64
	grandTotal  int64
}

// priceLines computes every amount on an order from the quantities and unit
// prices the client sent.
//
// The order's tax is the VAT on the order total, not the sum of the per-line
// VATs. See domain.OrderTaxAmount for why the two differ and which one the
// screens show.
func priceLines(items []OrderLineInput) (*pricedLines, error) {
	if len(items) == 0 {
		return nil, domain.ErrOrderNoItems
	}

	out := &pricedLines{
		amounts:    make([]int64, len(items)),
		taxAmounts: make([]int64, len(items)),
	}

	for i, item := range items {
		if item.Quantity <= 0 {
			return nil, domain.ErrOrderNothingToPost
		}
		if item.UnitPrice < 0 {
			return nil, domain.ErrOrderNegativeAmount
		}
		amount := domain.LineAmount(item.Quantity, item.UnitPrice)
		out.amounts[i] = amount
		out.taxAmounts[i] = domain.LineTaxAmount(amount)
		out.totalAmount += amount
	}

	out.taxAmount = domain.OrderTaxAmount(out.totalAmount)
	out.grandTotal = out.totalAmount + out.taxAmount
	return out, nil
}

// validateOrderHeader checks the parts of an order that do not depend on the
// order type.
func (s *orderService) validateOrderHeader(ctx context.Context, companyID, warehouseID uuid.UUID, orderDate time.Time, expectedDate *time.Time, items []OrderLineInput) error {
	if expectedDate != nil && expectedDate.Before(orderDate) {
		return domain.ErrOrderDateRange
	}
	if _, err := s.productRepo.GetWarehouseByID(ctx, companyID, warehouseID); err != nil {
		return err
	}
	// Every product must belong to the tenant and be active. Checked before the
	// insert so an unknown product is a 404, not a foreign-key 500.
	for _, item := range items {
		product, err := s.productRepo.GetByID(ctx, companyID, item.ProductID)
		if err != nil {
			return err
		}
		if !product.IsActive {
			return domain.ErrProductInactive
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Purchase orders
// ---------------------------------------------------------------------------

// CreatePurchaseOrder prices and inserts a draft purchase order.
func (s *orderService) CreatePurchaseOrder(ctx context.Context, in *PurchaseOrderInput) (*domain.PurchaseOrder, error) {
	if err := s.validateOrderHeader(ctx, in.CompanyID, in.WarehouseID, in.OrderDate, in.ExpectedDate, in.Items); err != nil {
		return nil, err
	}

	priced, err := priceLines(in.Items)
	if err != nil {
		return nil, err
	}

	var created *domain.PurchaseOrder
	err = s.orderRepo.WithTransaction(ctx, func(orderRepo repository.OrderRepository, _ repository.StockRepository) error {
		// The number is reserved inside the same transaction as the insert, so
		// a failed insert releases nothing but also strands nothing that a
		// later order would collide with.
		number, err := orderRepo.NextOrderNumber(ctx, in.CompanyID, "purchase", in.OrderDate.Year())
		if err != nil {
			return err
		}

		order := &domain.PurchaseOrder{
			TenantModel:  domain.TenantModel{CompanyID: in.CompanyID},
			OrderNumber:  number,
			OrderDate:    in.OrderDate,
			ExpectedDate: in.ExpectedDate,
			SupplierID:   in.SupplierID,
			WarehouseID:  in.WarehouseID,
			TotalAmount:  priced.totalAmount,
			TaxAmount:    priced.taxAmount,
			GrandTotal:   priced.grandTotal,
			Status:       domain.POStatusDraft,
			Note:         in.Note,
			CreatedBy:    in.UserID,
		}

		order.Items = make([]domain.PurchaseOrderItem, len(in.Items))
		for i, item := range in.Items {
			order.Items[i] = domain.PurchaseOrderItem{
				TenantModel: domain.TenantModel{CompanyID: in.CompanyID},
				LineNo:      i + 1,
				ProductID:   item.ProductID,
				Quantity:    item.Quantity,
				UnitPrice:   item.UnitPrice,
				Amount:      priced.amounts[i],
				TaxAmount:   priced.taxAmounts[i],
				Note:        item.Note,
			}
		}

		if err := orderRepo.CreatePurchaseOrder(ctx, order); err != nil {
			return err
		}
		created = order
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.orderRepo.GetPurchaseOrder(ctx, in.CompanyID, created.ID)
}

// UpdatePurchaseOrder reprices and saves a draft purchase order.
func (s *orderService) UpdatePurchaseOrder(ctx context.Context, id uuid.UUID, in *PurchaseOrderInput) (*domain.PurchaseOrder, error) {
	existing, err := s.orderRepo.GetPurchaseOrder(ctx, in.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if !existing.Status.CanEdit() {
		return nil, domain.ErrOrderCannotEdit
	}
	if err := s.validateOrderHeader(ctx, in.CompanyID, in.WarehouseID, in.OrderDate, in.ExpectedDate, in.Items); err != nil {
		return nil, err
	}

	priced, err := priceLines(in.Items)
	if err != nil {
		return nil, err
	}

	err = s.orderRepo.WithTransaction(ctx, func(orderRepo repository.OrderRepository, _ repository.StockRepository) error {
		// Re-read under the row lock. Without it, an approval that lands between
		// the check above and the write below would be overwritten by a draft
		// edit - the order would go back to being editable after approval.
		locked, err := orderRepo.LockPurchaseOrder(ctx, in.CompanyID, id)
		if err != nil {
			return err
		}
		if !locked.Status.CanEdit() {
			return domain.ErrOrderCannotEdit
		}

		locked.OrderDate = in.OrderDate
		locked.ExpectedDate = in.ExpectedDate
		locked.SupplierID = in.SupplierID
		locked.WarehouseID = in.WarehouseID
		locked.Note = in.Note
		locked.TotalAmount = priced.totalAmount
		locked.TaxAmount = priced.taxAmount
		locked.GrandTotal = priced.grandTotal
		locked.Items = nil

		if err := orderRepo.UpdatePurchaseOrder(ctx, locked); err != nil {
			return err
		}

		items := make([]domain.PurchaseOrderItem, len(in.Items))
		for i, item := range in.Items {
			items[i] = domain.PurchaseOrderItem{
				TenantModel:     domain.TenantModel{CompanyID: in.CompanyID},
				PurchaseOrderID: id,
				LineNo:          i + 1,
				ProductID:       item.ProductID,
				Quantity:        item.Quantity,
				UnitPrice:       item.UnitPrice,
				Amount:          priced.amounts[i],
				TaxAmount:       priced.taxAmounts[i],
				Note:            item.Note,
			}
		}
		return orderRepo.ReplacePurchaseOrderItems(ctx, in.CompanyID, id, items)
	})
	if err != nil {
		return nil, err
	}

	return s.orderRepo.GetPurchaseOrder(ctx, in.CompanyID, id)
}

// GetPurchaseOrder retrieves one order.
func (s *orderService) GetPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error) {
	return s.orderRepo.GetPurchaseOrder(ctx, companyID, id)
}

// ListPurchaseOrders retrieves a page of orders.
func (s *orderService) ListPurchaseOrders(ctx context.Context, filter *PurchaseOrderFilter) ([]domain.PurchaseOrder, int64, error) {
	return s.orderRepo.ListPurchaseOrders(ctx, filter)
}

// DeletePurchaseOrder removes a draft or pending order.
func (s *orderService) DeletePurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error {
	if _, err := s.orderRepo.GetPurchaseOrder(ctx, companyID, id); err != nil {
		return err
	}
	deleted, err := s.orderRepo.DeletePurchaseOrder(ctx, companyID, id,
		[]domain.PurchaseOrderStatus{domain.POStatusDraft, domain.POStatusPending})
	if err != nil {
		return err
	}
	if !deleted {
		return domain.ErrOrderCannotDelete
	}
	return nil
}

// SubmitPurchaseOrder sends a draft for approval.
func (s *orderService) SubmitPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionPurchase(ctx, companyID, id, nil,
		[]domain.PurchaseOrderStatus{domain.POStatusDraft}, domain.POStatusPending,
		domain.ErrOrderCannotSubmit)
}

// ApprovePurchaseOrder approves a pending order.
func (s *orderService) ApprovePurchaseOrder(ctx context.Context, companyID, id, userID uuid.UUID) error {
	return s.transitionPurchase(ctx, companyID, id, &userID,
		[]domain.PurchaseOrderStatus{domain.POStatusPending}, domain.POStatusApproved,
		domain.ErrOrderCannotApprove)
}

// RejectPurchaseOrder sends a pending order back to draft.
func (s *orderService) RejectPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionPurchase(ctx, companyID, id, nil,
		[]domain.PurchaseOrderStatus{domain.POStatusPending}, domain.POStatusDraft,
		domain.ErrOrderCannotReject)
}

// PlacePurchaseOrder marks an approved order as placed with the supplier.
func (s *orderService) PlacePurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionPurchase(ctx, companyID, id, nil,
		[]domain.PurchaseOrderStatus{domain.POStatusApproved}, domain.POStatusOrdered,
		domain.ErrOrderCannotPlace)
}

// CancelPurchaseOrder cancels an order that has received nothing.
func (s *orderService) CancelPurchaseOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionPurchase(ctx, companyID, id, nil,
		[]domain.PurchaseOrderStatus{
			domain.POStatusDraft, domain.POStatusPending,
			domain.POStatusApproved, domain.POStatusOrdered,
		}, domain.POStatusCancelled, domain.ErrOrderCannotCancel)
}

// transitionPurchase is the shared shape of every purchase-order transition.
//
// The compare-and-swap in the repository is what makes a repeated request
// harmless: the second one matches no row and is reported as an illegal
// transition instead of applying the change twice.
func (s *orderService) transitionPurchase(ctx context.Context, companyID, id uuid.UUID, approvedBy *uuid.UUID, from []domain.PurchaseOrderStatus, to domain.PurchaseOrderStatus, onRefused error) error {
	if _, err := s.orderRepo.GetPurchaseOrder(ctx, companyID, id); err != nil {
		return err
	}
	ok, err := s.orderRepo.TransitionPurchaseOrder(ctx, companyID, id, from, to, approvedBy)
	if err != nil {
		return err
	}
	if !ok {
		return onRefused
	}
	return nil
}

// ReceivePurchaseOrder books goods in against the order.
func (s *orderService) ReceivePurchaseOrder(ctx context.Context, companyID, id, userID uuid.UUID, lines []PostLine) (*domain.PurchaseOrder, error) {
	if len(lines) == 0 {
		return nil, domain.ErrOrderNothingToPost
	}

	err := s.orderRepo.WithTransaction(ctx, func(orderRepo repository.OrderRepository, stockRepo repository.StockRepository) error {
		// The header lock is the serialisation point. Two receipts of the same
		// order queue here, so the second one sees the first one's quantities
		// and cannot over-receive on a stale reading.
		order, err := orderRepo.LockPurchaseOrder(ctx, companyID, id)
		if err != nil {
			return err
		}
		if !order.Status.CanReceive() {
			return domain.ErrOrderCannotReceive
		}

		byID := make(map[uuid.UUID]*domain.PurchaseOrderItem, len(order.Items))
		for i := range order.Items {
			byID[order.Items[i].ID] = &order.Items[i]
		}

		posted := 0
		for _, line := range lines {
			if line.Quantity <= 0 {
				continue
			}
			item, ok := byID[line.ItemID]
			if !ok {
				return domain.ErrOrderLineNotFound
			}

			// Raise the line first. The predicate inside the UPDATE is what
			// refuses an over-receipt, including the replayed case where the
			// quantity looked available when the request was built.
			ok, err := orderRepo.AddReceivedQuantity(ctx, companyID, item.ID, line.Quantity)
			if err != nil {
				return err
			}
			if !ok {
				return domain.ErrOrderOverReceipt
			}

			movement := &domain.StockMovement{
				CompanyID:     companyID,
				ProductID:     item.ProductID,
				WarehouseID:   order.WarehouseID,
				MovementType:  domain.MovementPurchaseIn,
				Quantity:      line.Quantity,
				ReferenceType: domain.StockRefPurchaseOrder,
				ReferenceID:   &order.ID,
				Note:          order.OrderNumber,
				CreatedBy:     userID,
			}
			if err := stockRepo.ApplyMovement(ctx, movement); err != nil {
				return err
			}

			item.ReceivedQuantity += line.Quantity
			posted++
		}

		if posted == 0 {
			return domain.ErrOrderNothingToPost
		}

		// The status follows from the lines, which the loop above has kept in
		// step with the database.
		next := domain.POStatusCompleted
		for i := range order.Items {
			if order.Items[i].OutstandingQuantity() > quantityEpsilon {
				next = domain.POStatusPartial
				break
			}
		}

		ok, err := orderRepo.TransitionPurchaseOrder(ctx, companyID, id,
			[]domain.PurchaseOrderStatus{domain.POStatusOrdered, domain.POStatusPartial}, next, nil)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrOrderCannotReceive
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.orderRepo.GetPurchaseOrder(ctx, companyID, id)
}

// GetPurchaseOrderSummary returns the purchase dashboard aggregate.
func (s *orderService) GetPurchaseOrderSummary(ctx context.Context, companyID uuid.UUID) (*PurchaseOrderSummary, error) {
	return s.orderRepo.PurchaseOrderSummary(ctx, companyID)
}

// ---------------------------------------------------------------------------
// Sales orders
// ---------------------------------------------------------------------------

// CreateSalesOrder prices and inserts a draft sales order.
func (s *orderService) CreateSalesOrder(ctx context.Context, in *SalesOrderInput) (*domain.SalesOrder, error) {
	if err := s.validateOrderHeader(ctx, in.CompanyID, in.WarehouseID, in.OrderDate, in.ExpectedDate, in.Items); err != nil {
		return nil, err
	}

	priced, err := priceLines(in.Items)
	if err != nil {
		return nil, err
	}

	var created *domain.SalesOrder
	err = s.orderRepo.WithTransaction(ctx, func(orderRepo repository.OrderRepository, _ repository.StockRepository) error {
		number, err := orderRepo.NextOrderNumber(ctx, in.CompanyID, "sales", in.OrderDate.Year())
		if err != nil {
			return err
		}

		order := &domain.SalesOrder{
			TenantModel:  domain.TenantModel{CompanyID: in.CompanyID},
			OrderNumber:  number,
			OrderDate:    in.OrderDate,
			ExpectedDate: in.ExpectedDate,
			CustomerID:   in.CustomerID,
			WarehouseID:  in.WarehouseID,
			TotalAmount:  priced.totalAmount,
			TaxAmount:    priced.taxAmount,
			GrandTotal:   priced.grandTotal,
			Status:       domain.SOStatusDraft,
			Note:         in.Note,
			CreatedBy:    in.UserID,
		}

		order.Items = make([]domain.SalesOrderItem, len(in.Items))
		for i, item := range in.Items {
			order.Items[i] = domain.SalesOrderItem{
				TenantModel: domain.TenantModel{CompanyID: in.CompanyID},
				LineNo:      i + 1,
				ProductID:   item.ProductID,
				Quantity:    item.Quantity,
				UnitPrice:   item.UnitPrice,
				Amount:      priced.amounts[i],
				TaxAmount:   priced.taxAmounts[i],
				Note:        item.Note,
			}
		}

		if err := orderRepo.CreateSalesOrder(ctx, order); err != nil {
			return err
		}
		created = order
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.orderRepo.GetSalesOrder(ctx, in.CompanyID, created.ID)
}

// UpdateSalesOrder reprices and saves a draft sales order.
func (s *orderService) UpdateSalesOrder(ctx context.Context, id uuid.UUID, in *SalesOrderInput) (*domain.SalesOrder, error) {
	existing, err := s.orderRepo.GetSalesOrder(ctx, in.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if !existing.Status.CanEdit() {
		return nil, domain.ErrOrderCannotEdit
	}
	if err := s.validateOrderHeader(ctx, in.CompanyID, in.WarehouseID, in.OrderDate, in.ExpectedDate, in.Items); err != nil {
		return nil, err
	}

	priced, err := priceLines(in.Items)
	if err != nil {
		return nil, err
	}

	err = s.orderRepo.WithTransaction(ctx, func(orderRepo repository.OrderRepository, _ repository.StockRepository) error {
		locked, err := orderRepo.LockSalesOrder(ctx, in.CompanyID, id)
		if err != nil {
			return err
		}
		if !locked.Status.CanEdit() {
			return domain.ErrOrderCannotEdit
		}

		locked.OrderDate = in.OrderDate
		locked.ExpectedDate = in.ExpectedDate
		locked.CustomerID = in.CustomerID
		locked.WarehouseID = in.WarehouseID
		locked.Note = in.Note
		locked.TotalAmount = priced.totalAmount
		locked.TaxAmount = priced.taxAmount
		locked.GrandTotal = priced.grandTotal
		locked.Items = nil

		if err := orderRepo.UpdateSalesOrder(ctx, locked); err != nil {
			return err
		}

		items := make([]domain.SalesOrderItem, len(in.Items))
		for i, item := range in.Items {
			items[i] = domain.SalesOrderItem{
				TenantModel:  domain.TenantModel{CompanyID: in.CompanyID},
				SalesOrderID: id,
				LineNo:       i + 1,
				ProductID:    item.ProductID,
				Quantity:     item.Quantity,
				UnitPrice:    item.UnitPrice,
				Amount:       priced.amounts[i],
				TaxAmount:    priced.taxAmounts[i],
				Note:         item.Note,
			}
		}
		return orderRepo.ReplaceSalesOrderItems(ctx, in.CompanyID, id, items)
	})
	if err != nil {
		return nil, err
	}

	return s.orderRepo.GetSalesOrder(ctx, in.CompanyID, id)
}

// GetSalesOrder retrieves one order.
func (s *orderService) GetSalesOrder(ctx context.Context, companyID, id uuid.UUID) (*domain.SalesOrder, error) {
	return s.orderRepo.GetSalesOrder(ctx, companyID, id)
}

// ListSalesOrders retrieves a page of orders.
func (s *orderService) ListSalesOrders(ctx context.Context, filter *SalesOrderFilter) ([]domain.SalesOrder, int64, error) {
	return s.orderRepo.ListSalesOrders(ctx, filter)
}

// DeleteSalesOrder removes a draft or pending order.
func (s *orderService) DeleteSalesOrder(ctx context.Context, companyID, id uuid.UUID) error {
	if _, err := s.orderRepo.GetSalesOrder(ctx, companyID, id); err != nil {
		return err
	}
	deleted, err := s.orderRepo.DeleteSalesOrder(ctx, companyID, id,
		[]domain.SalesOrderStatus{domain.SOStatusDraft, domain.SOStatusPending})
	if err != nil {
		return err
	}
	if !deleted {
		return domain.ErrOrderCannotDelete
	}
	return nil
}

// SubmitSalesOrder sends a draft for approval.
func (s *orderService) SubmitSalesOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionSales(ctx, companyID, id, nil,
		[]domain.SalesOrderStatus{domain.SOStatusDraft}, domain.SOStatusPending,
		domain.ErrOrderCannotSubmit)
}

// ApproveSalesOrder approves a pending order.
func (s *orderService) ApproveSalesOrder(ctx context.Context, companyID, id, userID uuid.UUID) error {
	return s.transitionSales(ctx, companyID, id, &userID,
		[]domain.SalesOrderStatus{domain.SOStatusPending}, domain.SOStatusApproved,
		domain.ErrOrderCannotApprove)
}

// RejectSalesOrder sends a pending order back to draft.
func (s *orderService) RejectSalesOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionSales(ctx, companyID, id, nil,
		[]domain.SalesOrderStatus{domain.SOStatusPending}, domain.SOStatusDraft,
		domain.ErrOrderCannotReject)
}

// ConfirmSalesOrder confirms an approved order with the customer.
func (s *orderService) ConfirmSalesOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionSales(ctx, companyID, id, nil,
		[]domain.SalesOrderStatus{domain.SOStatusApproved}, domain.SOStatusConfirmed,
		domain.ErrOrderCannotConfirm)
}

// CancelSalesOrder cancels an order that has shipped nothing.
func (s *orderService) CancelSalesOrder(ctx context.Context, companyID, id uuid.UUID) error {
	return s.transitionSales(ctx, companyID, id, nil,
		[]domain.SalesOrderStatus{
			domain.SOStatusDraft, domain.SOStatusPending,
			domain.SOStatusApproved, domain.SOStatusConfirmed,
		}, domain.SOStatusCancelled, domain.ErrOrderCannotCancel)
}

// transitionSales is the shared shape of every sales-order transition.
func (s *orderService) transitionSales(ctx context.Context, companyID, id uuid.UUID, approvedBy *uuid.UUID, from []domain.SalesOrderStatus, to domain.SalesOrderStatus, onRefused error) error {
	if _, err := s.orderRepo.GetSalesOrder(ctx, companyID, id); err != nil {
		return err
	}
	ok, err := s.orderRepo.TransitionSalesOrder(ctx, companyID, id, from, to, approvedBy)
	if err != nil {
		return err
	}
	if !ok {
		return onRefused
	}
	return nil
}

// ShipSalesOrder books goods out against the order.
func (s *orderService) ShipSalesOrder(ctx context.Context, companyID, id, userID uuid.UUID, lines []PostLine) (*domain.SalesOrder, error) {
	if len(lines) == 0 {
		return nil, domain.ErrOrderNothingToPost
	}

	err := s.orderRepo.WithTransaction(ctx, func(orderRepo repository.OrderRepository, stockRepo repository.StockRepository) error {
		order, err := orderRepo.LockSalesOrder(ctx, companyID, id)
		if err != nil {
			return err
		}
		if !order.Status.CanShip() {
			return domain.ErrOrderCannotShip
		}

		byID := make(map[uuid.UUID]*domain.SalesOrderItem, len(order.Items))
		for i := range order.Items {
			byID[order.Items[i].ID] = &order.Items[i]
		}

		posted := 0
		for _, line := range lines {
			if line.Quantity <= 0 {
				continue
			}
			item, ok := byID[line.ItemID]
			if !ok {
				return domain.ErrOrderLineNotFound
			}

			ok, err := orderRepo.AddShippedQuantity(ctx, companyID, item.ID, line.Quantity)
			if err != nil {
				return err
			}
			if !ok {
				return domain.ErrOrderOverShipment
			}

			// The issue can still fail on chk_stocks_quantity_non_negative if
			// the warehouse does not hold enough. That rolls the whole
			// transaction back, including the shipped quantity raised above, so
			// the order never claims to have shipped stock that did not exist.
			movement := &domain.StockMovement{
				CompanyID:     companyID,
				ProductID:     item.ProductID,
				WarehouseID:   order.WarehouseID,
				MovementType:  domain.MovementSalesOut,
				Quantity:      line.Quantity,
				ReferenceType: domain.StockRefSalesOrder,
				ReferenceID:   &order.ID,
				Note:          order.OrderNumber,
				CreatedBy:     userID,
			}
			if err := stockRepo.ApplyMovement(ctx, movement); err != nil {
				return err
			}

			item.ShippedQuantity += line.Quantity
			posted++
		}

		if posted == 0 {
			return domain.ErrOrderNothingToPost
		}

		next := domain.SOStatusCompleted
		for i := range order.Items {
			if order.Items[i].OutstandingQuantity() > quantityEpsilon {
				next = domain.SOStatusPartial
				break
			}
		}

		ok, err := orderRepo.TransitionSalesOrder(ctx, companyID, id,
			[]domain.SalesOrderStatus{domain.SOStatusConfirmed, domain.SOStatusPartial}, next, nil)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrOrderCannotShip
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.orderRepo.GetSalesOrder(ctx, companyID, id)
}

// GetSalesOrderSummary returns the sales dashboard aggregate.
func (s *orderService) GetSalesOrderSummary(ctx context.Context, companyID uuid.UUID) (*SalesOrderSummary, error) {
	return s.orderRepo.SalesOrderSummary(ctx, companyID)
}
