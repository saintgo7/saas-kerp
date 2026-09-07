package router

import (
	"github.com/gin-gonic/gin"

	"github.com/saintgo7/saas-kerp/internal/handler"
	"github.com/saintgo7/saas-kerp/internal/middleware"
)

// InventoryHandlers is the set of handlers the inventory routes need.
//
// It is a struct of its own rather than a set of fields on handler.Handlers so
// that this file owns its wiring end to end. RegisterInventoryRoutes is called
// from registerTenantRoutes in v1.go once handler.Handlers grows the three
// fields; see the handover note in the build report for the exact lines.
type InventoryHandlers struct {
	Product *handler.ProductHandler
	Stock   *handler.StockHandler
	Order   *handler.OrderHandler
}

// RegisterInventoryRoutes mounts the inventory module on the tenant group.
//
// The caller must already have applied Auth and Tenant middleware (and the
// tenant session, where it is enabled): every handler below reads the company
// id out of the request context and every query is filtered by it, on top of
// the RLS policies from 000022.
//
// RBAC, stated once here rather than scattered across the handlers:
//
//	read            - any authenticated member of the tenant, viewer included
//	create / edit   - RequireWriter (admin, user), never viewer
//	approve, reject,
//	place, confirm,
//	cancel          - RequireApprover (admin only)
//	receive, ship   - RequireApprover
//	stock adjust,
//	stock transfer  - RequireApprover
//
// Receiving, shipping, adjusting and transferring sit with the approvers on
// purpose. Each of them writes an irreversible line into the stock ledger and
// moves the balance a warehouse count is measured against - the same class of
// action as approving a voucher or transmitting a tax invoice, which
// RequireApprover already guards elsewhere in this API. An ordinary "user" can
// prepare all of them by drafting the order; only an approver can post one.
func RegisterInventoryRoutes(tenant *gin.RouterGroup, h *InventoryHandlers) {
	if h == nil {
		return
	}

	registerProductRoutes(tenant, h.Product)
	registerCategoryRoutes(tenant, h.Product)
	registerWarehouseRoutes(tenant, h.Product)
	registerStockRoutes(tenant, h.Stock)
	registerPurchaseOrderRoutes(tenant, h.Order)
	registerSalesOrderRoutes(tenant, h.Order)
}

// registerProductRoutes mounts /products.
func registerProductRoutes(tenant *gin.RouterGroup, h *handler.ProductHandler) {
	if h == nil {
		return
	}
	products := tenant.Group("/products")
	{
		// Literal segments are declared before the ":id" wildcard so that
		// /products/stats is matched by its own route. Without that it parses
		// as a product id, fails uuid.Parse and answers "Invalid product ID",
		// which tells the client its identifier was malformed when in fact it
		// asked for a different endpoint. Same defect notImplementedRoute was
		// added for on /vouchers.
		products.GET("", h.ListProducts)
		products.GET("/stats", h.GetProductStats)
		products.GET("/code/:code", h.GetProductByCode)
		products.GET("/:id", h.GetProduct)
		products.GET("/:id/can-delete", h.CanDeleteProduct)

		products.POST("", middleware.RequireWriter(), h.CreateProduct)
		products.POST("/activate", middleware.RequireWriter(), h.ActivateProducts)
		products.POST("/deactivate", middleware.RequireWriter(), h.DeactivateProducts)
		products.PUT("/:id", middleware.RequireWriter(), h.UpdateProduct)
		products.DELETE("/:id", middleware.RequireWriter(), h.DeleteProduct)
	}
}

// registerCategoryRoutes mounts /product-categories.
func registerCategoryRoutes(tenant *gin.RouterGroup, h *handler.ProductHandler) {
	if h == nil {
		return
	}
	categories := tenant.Group("/product-categories")
	{
		categories.GET("", h.ListCategories)
		categories.GET("/:id", h.GetCategory)

		categories.POST("", middleware.RequireWriter(), h.CreateCategory)
		categories.PUT("/:id", middleware.RequireWriter(), h.UpdateCategory)
		categories.DELETE("/:id", middleware.RequireWriter(), h.DeleteCategory)
	}
}

// registerWarehouseRoutes mounts /warehouses.
func registerWarehouseRoutes(tenant *gin.RouterGroup, h *handler.ProductHandler) {
	if h == nil {
		return
	}
	warehouses := tenant.Group("/warehouses")
	{
		warehouses.GET("", h.ListWarehouses)
		warehouses.GET("/:id", h.GetWarehouse)

		warehouses.POST("", middleware.RequireWriter(), h.CreateWarehouse)
		warehouses.PUT("/:id", middleware.RequireWriter(), h.UpdateWarehouse)
		warehouses.DELETE("/:id", middleware.RequireWriter(), h.DeleteWarehouse)
	}
}

// registerStockRoutes mounts /stocks, /stock-movements, /stock-alerts and the
// two posting endpoints.
func registerStockRoutes(tenant *gin.RouterGroup, h *handler.StockHandler) {
	if h == nil {
		return
	}

	stocks := tenant.Group("/stocks")
	{
		stocks.GET("", h.ListStocks)
		stocks.GET("/stats", h.GetStats)
	}

	tenant.GET("/stock-movements", h.ListMovements)
	tenant.GET("/stock-alerts", h.ListAlerts)

	// Both of these write the ledger directly, with no order behind them to
	// review afterwards. Approver-only.
	tenant.POST("/stock-adjustments", middleware.RequireApprover(), h.AdjustStock)
	tenant.POST("/stock-transfers", middleware.RequireApprover(), h.TransferStock)
}

// registerPurchaseOrderRoutes mounts /purchase-orders.
func registerPurchaseOrderRoutes(tenant *gin.RouterGroup, h *handler.OrderHandler) {
	if h == nil {
		return
	}
	orders := tenant.Group("/purchase-orders")
	{
		orders.GET("", h.ListPurchaseOrders)
		orders.GET("/stats", h.GetPurchaseOrderStats)
		orders.GET("/:id", h.GetPurchaseOrder)

		orders.POST("", middleware.RequireWriter(), h.CreatePurchaseOrder)
		orders.PUT("/:id", middleware.RequireWriter(), h.UpdatePurchaseOrder)
		orders.DELETE("/:id", middleware.RequireWriter(), h.DeletePurchaseOrder)

		// Submitting is a writer action: it asks for approval, it does not
		// grant it.
		orders.POST("/:id/submit", middleware.RequireWriter(), h.SubmitPurchaseOrder)

		orders.POST("/:id/approve", middleware.RequireApprover(), h.ApprovePurchaseOrder)
		orders.POST("/:id/reject", middleware.RequireApprover(), h.RejectPurchaseOrder)
		orders.POST("/:id/place", middleware.RequireApprover(), h.PlacePurchaseOrder)
		orders.POST("/:id/cancel", middleware.RequireApprover(), h.CancelPurchaseOrder)
		orders.POST("/:id/receive", middleware.RequireApprover(), h.ReceivePurchaseOrder)
	}
}

// registerSalesOrderRoutes mounts /sales-orders.
func registerSalesOrderRoutes(tenant *gin.RouterGroup, h *handler.OrderHandler) {
	if h == nil {
		return
	}
	orders := tenant.Group("/sales-orders")
	{
		orders.GET("", h.ListSalesOrders)
		orders.GET("/stats", h.GetSalesOrderStats)
		orders.GET("/:id", h.GetSalesOrder)

		orders.POST("", middleware.RequireWriter(), h.CreateSalesOrder)
		orders.PUT("/:id", middleware.RequireWriter(), h.UpdateSalesOrder)
		orders.DELETE("/:id", middleware.RequireWriter(), h.DeleteSalesOrder)

		orders.POST("/:id/submit", middleware.RequireWriter(), h.SubmitSalesOrder)

		orders.POST("/:id/approve", middleware.RequireApprover(), h.ApproveSalesOrder)
		orders.POST("/:id/reject", middleware.RequireApprover(), h.RejectSalesOrder)
		orders.POST("/:id/confirm", middleware.RequireApprover(), h.ConfirmSalesOrder)
		orders.POST("/:id/cancel", middleware.RequireApprover(), h.CancelSalesOrder)
		orders.POST("/:id/ship", middleware.RequireApprover(), h.ShipSalesOrder)
	}
}

// NewInventoryHandlers builds the inventory handlers from the services the
// caller has already constructed.
//
// It exists so that the wiring in handler.Handlers (which this agent does not
// own) is a single call rather than three constructor invocations plus the
// repository and service graph behind them.
func NewInventoryHandlers(product *handler.ProductHandler, stock *handler.StockHandler, order *handler.OrderHandler) *InventoryHandlers {
	return &InventoryHandlers{Product: product, Stock: stock, Order: order}
}
