package dto

import (
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// StockResponse is a stock balance in API responses.
type StockResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`

	ProductID   string             `json:"product_id"`
	Product     *ProductResponse   `json:"product,omitempty"`
	WarehouseID string             `json:"warehouse_id"`
	Warehouse   *WarehouseResponse `json:"warehouse,omitempty"`

	Quantity          float64 `json:"quantity"`
	AvailableQuantity float64 `json:"available_quantity"`
	ReservedQuantity  float64 `json:"reserved_quantity"`

	// StockValue and IsBelowMinStock are not in the frontend's Stock interface:
	// StockStatusPage computes both per row, from product.costPrice and
	// product.minStock. They are supplied here because the page's own summary
	// card needs the same figures totalled over every row, and a value computed
	// two different ways in two places is a value that will eventually disagree
	// with itself.
	StockValue      float64 `json:"stock_value"`
	IsBelowMinStock bool    `json:"is_below_min_stock"`

	LastUpdatedAt string `json:"last_updated_at"`
}

// FromStock converts a domain balance to its response form.
func FromStock(s *domain.Stock) StockResponse {
	resp := StockResponse{
		ID:                s.ID.String(),
		CompanyID:         s.CompanyID.String(),
		ProductID:         s.ProductID.String(),
		WarehouseID:       s.WarehouseID.String(),
		Quantity:          s.Quantity,
		AvailableQuantity: s.AvailableQuantity,
		ReservedQuantity:  s.ReservedQuantity,
		StockValue:        s.StockValue(),
		IsBelowMinStock:   s.IsBelowMinStock(),
		LastUpdatedAt:     formatTime(s.LastUpdatedAt),
	}
	if s.Product != nil {
		product := FromProduct(s.Product)
		resp.Product = &product
	}
	if s.Warehouse != nil {
		warehouse := FromWarehouse(s.Warehouse)
		resp.Warehouse = &warehouse
	}
	return resp
}

// FromStocks converts a slice of balances.
func FromStocks(stocks []domain.Stock) []StockResponse {
	out := make([]StockResponse, len(stocks))
	for i := range stocks {
		out[i] = FromStock(&stocks[i])
	}
	return out
}

// StockMovementResponse is one ledger line in API responses.
type StockMovementResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`

	ProductID   string             `json:"product_id"`
	Product     *ProductResponse   `json:"product,omitempty"`
	WarehouseID string             `json:"warehouse_id"`
	Warehouse   *WarehouseResponse `json:"warehouse,omitempty"`

	MovementType     string  `json:"movement_type"`
	Quantity         float64 `json:"quantity"`
	PreviousQuantity float64 `json:"previous_quantity"`
	CurrentQuantity  float64 `json:"current_quantity"`

	ReferenceType string `json:"reference_type,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`
	Note          string `json:"note,omitempty"`

	// CreatedBy is the user's UUID, and CreatedByName is what to display.
	//
	// The frontend's StockMovement.createdBy is typed string and rendered
	// directly into the table, and the mock data puts a person's name in it. A
	// server that answers with the id alone would print a UUID in that column,
	// so both are sent and the client picks created_by_name.
	CreatedBy     string `json:"created_by"`
	CreatedByName string `json:"created_by_name,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// FromStockMovement converts a domain movement to its response form.
func FromStockMovement(m *domain.StockMovement) StockMovementResponse {
	resp := StockMovementResponse{
		ID:               m.ID.String(),
		CompanyID:        m.CompanyID.String(),
		ProductID:        m.ProductID.String(),
		WarehouseID:      m.WarehouseID.String(),
		MovementType:     string(m.MovementType),
		Quantity:         m.Quantity,
		PreviousQuantity: m.PreviousQuantity,
		CurrentQuantity:  m.CurrentQuantity,
		ReferenceType:    m.ReferenceType,
		Note:             m.Note,
		CreatedBy:        m.CreatedBy.String(),
		CreatedAt:        formatTime(m.CreatedAt),
	}
	if m.ReferenceID != nil {
		resp.ReferenceID = m.ReferenceID.String()
	}
	if m.Product != nil {
		product := FromProduct(m.Product)
		resp.Product = &product
	}
	if m.Warehouse != nil {
		warehouse := FromWarehouse(m.Warehouse)
		resp.Warehouse = &warehouse
	}
	resp.CreatedByName = m.CreatedByName
	return resp
}

// FromStockMovements converts a slice of movements.
func FromStockMovements(movements []domain.StockMovement) []StockMovementResponse {
	out := make([]StockMovementResponse, len(movements))
	for i := range movements {
		out[i] = FromStockMovement(&movements[i])
	}
	return out
}

// StockAlertResponse is a derived threshold breach in API responses.
type StockAlertResponse struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`

	ProductID   string             `json:"product_id"`
	Product     *ProductResponse   `json:"product,omitempty"`
	WarehouseID string             `json:"warehouse_id"`
	Warehouse   *WarehouseResponse `json:"warehouse,omitempty"`

	AlertType       string  `json:"alert_type"`
	CurrentQuantity float64 `json:"current_quantity"`
	Threshold       float64 `json:"threshold"`

	// IsRead is always false; see domain.StockAlert.
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

// FromStockAlert converts a derived alert to its response form.
func FromStockAlert(a *domain.StockAlert) StockAlertResponse {
	resp := StockAlertResponse{
		ID:              a.ID.String(),
		CompanyID:       a.CompanyID.String(),
		ProductID:       a.ProductID.String(),
		WarehouseID:     a.WarehouseID.String(),
		AlertType:       string(a.AlertType),
		CurrentQuantity: a.CurrentQuantity,
		Threshold:       a.Threshold,
		IsRead:          a.IsRead,
		CreatedAt:       formatTime(a.CreatedAt),
	}
	if a.Product != nil {
		product := FromProduct(a.Product)
		resp.Product = &product
	}
	if a.Warehouse != nil {
		warehouse := FromWarehouse(a.Warehouse)
		resp.Warehouse = &warehouse
	}
	return resp
}

// FromStockAlerts converts a slice of alerts.
func FromStockAlerts(alerts []domain.StockAlert) []StockAlertResponse {
	out := make([]StockAlertResponse, len(alerts))
	for i := range alerts {
		out[i] = FromStockAlert(&alerts[i])
	}
	return out
}

// AdjustStockRequest is the body of a manual stock correction.
//
// It carries the quantity the operator wants the warehouse to show, not a
// delta: the screen shows a current balance and an intended one, and asking for
// the difference would make the request depend on how stale that screen is.
type AdjustStockRequest struct {
	ProductID      string  `json:"product_id" binding:"required,uuid"`
	WarehouseID    string  `json:"warehouse_id" binding:"required,uuid"`
	TargetQuantity float64 `json:"target_quantity" binding:"gte=0"`
	Reason         string  `json:"reason,omitempty" binding:"max=500"`
}

// TransferStockRequest is the body of a warehouse-to-warehouse transfer.
type TransferStockRequest struct {
	ProductID       string  `json:"product_id" binding:"required,uuid"`
	FromWarehouseID string  `json:"from_warehouse_id" binding:"required,uuid"`
	ToWarehouseID   string  `json:"to_warehouse_id" binding:"required,uuid"`
	Quantity        float64 `json:"quantity" binding:"required,gt=0"`
	Note            string  `json:"note,omitempty" binding:"max=500"`
}

// StockStatsResponse is the stock dashboard card.
type StockStatsResponse struct {
	// StockRecords counts (product, warehouse) rows; ProductCount counts
	// distinct products. The frontend's "총 품목수" card uses the length of the
	// stock array, which is the first of these - both are sent so the label can
	// be made to match whichever the business actually means.
	StockRecords    int64   `json:"stock_records"`
	ProductCount    int64   `json:"product_count"`
	WarehouseCount  int64   `json:"warehouse_count"`
	TotalStockValue float64 `json:"total_stock_value"`
	LowStockCount   int64   `json:"low_stock_count"`
	OutOfStockCount int64   `json:"out_of_stock_count"`
}
