package handler

import (
	stderrors "errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// ProductHandler serves the inventory master data: products, their categories
// and warehouses.
type ProductHandler struct {
	service service.ProductService
}

// NewProductHandler creates a ProductHandler.
func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{service: svc}
}

// productErrorStatus maps a domain error to the client-facing answer.
//
// Centralised so every endpoint in the module reports the same failure with the
// same code, and so no handler reaches for err.Error(): those strings carry
// table, column and constraint names.
func productErrorStatus(err error) (int, string, string, bool) {
	switch {
	case stderrors.Is(err, domain.ErrProductNotFound):
		return http.StatusNotFound, errors.CodeNotFound, "Product not found", true
	case stderrors.Is(err, domain.ErrCategoryNotFound):
		return http.StatusNotFound, errors.CodeNotFound, "Product category not found", true
	case stderrors.Is(err, domain.ErrWarehouseNotFound):
		return http.StatusNotFound, errors.CodeNotFound, "Warehouse not found", true
	case stderrors.Is(err, domain.ErrCategoryParentMissing):
		return http.StatusNotFound, errors.CodeNotFound, "Parent product category not found", true

	case stderrors.Is(err, domain.ErrProductCodeExists):
		return http.StatusConflict, errors.CodeAlreadyExists, "Product code already exists", true
	case stderrors.Is(err, domain.ErrProductBarcodeExists):
		return http.StatusConflict, errors.CodeAlreadyExists, "Product barcode already exists", true
	case stderrors.Is(err, domain.ErrCategoryCodeExists):
		return http.StatusConflict, errors.CodeAlreadyExists, "Product category code already exists", true
	case stderrors.Is(err, domain.ErrWarehouseCodeExists):
		return http.StatusConflict, errors.CodeAlreadyExists, "Warehouse code already exists", true

	case stderrors.Is(err, domain.ErrProductInUse):
		return http.StatusConflict, errors.CodeConflict, "Product has stock or order history and cannot be deleted", true
	case stderrors.Is(err, domain.ErrCategoryHasChildren):
		return http.StatusConflict, errors.CodeConflict, "Product category has child categories", true
	case stderrors.Is(err, domain.ErrCategoryInUse):
		return http.StatusConflict, errors.CodeConflict, "Product category is still assigned to products", true
	case stderrors.Is(err, domain.ErrWarehouseInUse):
		return http.StatusConflict, errors.CodeConflict, "Warehouse holds stock or is referenced by an order", true

	case stderrors.Is(err, domain.ErrCategoryParentCycle):
		return http.StatusBadRequest, errors.CodeInvalidInput, "A product category cannot be its own ancestor", true
	case stderrors.Is(err, domain.ErrProductInvalidUnit):
		return http.StatusBadRequest, errors.CodeInvalidInput, "Invalid product unit", true
	case stderrors.Is(err, domain.ErrProductStockRange):
		return http.StatusBadRequest, errors.CodeOutOfRange, "Max stock must not be lower than min stock", true
	case stderrors.Is(err, domain.ErrProductNegativeAmount):
		return http.StatusBadRequest, errors.CodeOutOfRange, "Prices and stock thresholds must not be negative", true
	}
	return 0, "", "", false
}

// respondProductError answers a mapped domain error, or a logged 500.
func respondProductError(c *gin.Context, err error) {
	if status, code, message, ok := productErrorStatus(err); ok {
		response.ErrorLogged(c, status, code, message, err)
		return
	}
	response.InternalErrorLogged(c, "Internal server error", err)
}

// parseOptionalUUID reads a query parameter that may be absent.
func parseOptionalUUID(c *gin.Context, key string) (*uuid.UUID, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, false
	}
	return &id, true
}

// parseOptionalBool reads a boolean query parameter that may be absent.
func parseOptionalBool(c *gin.Context, key string) *bool {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	value := raw == "true" || raw == "1"
	return &value
}

// ---------------------------------------------------------------------------
// Products
// ---------------------------------------------------------------------------

// ListProducts handles GET /products.
func (h *ProductHandler) ListProducts(c *gin.Context) {
	filter := &service.ProductFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
		IsActive:   parseOptionalBool(c, "is_active"),
	}

	categoryID, ok := parseOptionalUUID(c, "category_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid category ID")
		return
	}
	filter.CategoryID = categoryID

	// Always through the shared helper: reading page_size straight from the
	// query lets page_size=0 reach the total-pages division and page_size=10^9
	// turn one request into a full-table scan.
	filter.Page, filter.PageSize = parsePageParams(c, 20)

	products, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromProducts(products),
		listMeta(total, filter.Page, filter.PageSize),
	))
}

// GetProduct handles GET /products/:id.
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid product ID")
		return
	}

	product, err := h.service.GetByID(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProduct(product)))
}

// GetProductByCode handles GET /products/code/:code.
func (h *ProductHandler) GetProductByCode(c *gin.Context) {
	product, err := h.service.GetByCode(c.Request.Context(), appctx.GetCompanyID(c), c.Param("code"))
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProduct(product)))
}

// CreateProduct handles POST /products.
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	product := &domain.Product{
		TenantModel:   domain.TenantModel{CompanyID: appctx.GetCompanyID(c)},
		Code:          req.Code,
		Name:          req.Name,
		Specification: req.Specification,
		Unit:          req.Unit,
		UnitPrice:     req.UnitPrice,
		CostPrice:     req.CostPrice,
		MinStock:      req.MinStock,
		MaxStock:      req.MaxStock,
		IsActive:      true,
		Description:   req.Description,
		Barcode:       req.Barcode,
		ImageURL:      req.ImageURL,
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}
	if req.CategoryID != "" {
		// Already validated as a UUID by the binding tag.
		categoryID := uuid.MustParse(req.CategoryID)
		product.CategoryID = &categoryID
	}

	if err := h.service.Create(c.Request.Context(), product); err != nil {
		respondProductError(c, err)
		return
	}

	created, err := h.service.GetByID(c.Request.Context(), product.CompanyID, product.ID)
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromProduct(created)))
}

// UpdateProduct handles PUT /products/:id.
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid product ID")
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	companyID := appctx.GetCompanyID(c)
	product, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondProductError(c, err)
		return
	}

	product.Code = req.Code
	product.Name = req.Name
	product.Specification = req.Specification
	product.Unit = req.Unit
	product.UnitPrice = req.UnitPrice
	product.CostPrice = req.CostPrice
	product.MinStock = req.MinStock
	product.MaxStock = req.MaxStock
	product.Description = req.Description
	product.Barcode = req.Barcode
	product.ImageURL = req.ImageURL
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}
	if req.CategoryID != "" {
		categoryID := uuid.MustParse(req.CategoryID)
		product.CategoryID = &categoryID
	} else {
		product.CategoryID = nil
	}
	// The preloaded association must not travel back into the write: it is the
	// old category, and Save would try to persist it.
	product.Category = nil

	if err := h.service.Update(c.Request.Context(), product); err != nil {
		respondProductError(c, err)
		return
	}

	updated, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProduct(updated)))
}

// DeleteProduct handles DELETE /products/:id.
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid product ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondProductError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CanDeleteProduct handles GET /products/:id/can-delete.
func (h *ProductHandler) CanDeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid product ID")
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"can_delete": canDelete, "reason": reason}))
}

// ActivateProducts handles POST /products/activate.
func (h *ProductHandler) ActivateProducts(c *gin.Context) {
	h.setActive(c, true)
}

// DeactivateProducts handles POST /products/deactivate.
func (h *ProductHandler) DeactivateProducts(c *gin.Context) {
	h.setActive(c, false)
}

// setActive is the shared body of the two bulk status endpoints.
func (h *ProductHandler) setActive(c *gin.Context, isActive bool) {
	var req dto.BulkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	ids := make([]uuid.UUID, len(req.IDs))
	for i, raw := range req.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid product ID")
			return
		}
		ids[i] = id
	}

	if err := h.service.SetActive(c.Request.Context(), appctx.GetCompanyID(c), ids, isActive); err != nil {
		respondProductError(c, err)
		return
	}

	key := "activated"
	if !isActive {
		key = "deactivated"
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{key: len(ids)}))
}

// GetProductStats handles GET /products/stats.
func (h *ProductHandler) GetProductStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context(), appctx.GetCompanyID(c))
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.ProductStatsResponse{
		TotalCount:    stats.TotalCount,
		ActiveCount:   stats.ActiveCount,
		InactiveCount: stats.InactiveCount,
	}))
}

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

// ListCategories handles GET /product-categories.
func (h *ProductHandler) ListCategories(c *gin.Context) {
	filter := &service.CategoryFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
		IsActive:   parseOptionalBool(c, "is_active"),
	}

	parentID, ok := parseOptionalUUID(c, "parent_id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid parent category ID")
		return
	}
	filter.ParentID = parentID
	filter.Page, filter.PageSize = parsePageParams(c, 50)

	categories, total, err := h.service.ListCategories(c.Request.Context(), filter)
	if err != nil {
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromProductCategories(categories),
		listMeta(total, filter.Page, filter.PageSize),
	))
}

// GetCategory handles GET /product-categories/:id.
func (h *ProductHandler) GetCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid category ID")
		return
	}

	category, err := h.service.GetCategory(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProductCategory(category)))
}

// CreateCategory handles POST /product-categories.
func (h *ProductHandler) CreateCategory(c *gin.Context) {
	var req dto.CreateProductCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	category := &domain.ProductCategory{
		TenantModel: domain.TenantModel{CompanyID: appctx.GetCompanyID(c)},
		Code:        req.Code,
		Name:        req.Name,
		IsActive:    true,
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	if req.ParentID != "" {
		parentID := uuid.MustParse(req.ParentID)
		category.ParentID = &parentID
	}

	if err := h.service.CreateCategory(c.Request.Context(), category); err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromProductCategory(category)))
}

// UpdateCategory handles PUT /product-categories/:id.
func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid category ID")
		return
	}

	var req dto.UpdateProductCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	companyID := appctx.GetCompanyID(c)
	category, err := h.service.GetCategory(c.Request.Context(), companyID, id)
	if err != nil {
		respondProductError(c, err)
		return
	}

	category.Code = req.Code
	category.Name = req.Name
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	if req.ParentID != "" {
		parentID := uuid.MustParse(req.ParentID)
		category.ParentID = &parentID
	} else {
		category.ParentID = nil
	}

	if err := h.service.UpdateCategory(c.Request.Context(), category); err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromProductCategory(category)))
}

// DeleteCategory handles DELETE /product-categories/:id.
func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid category ID")
		return
	}

	if err := h.service.DeleteCategory(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondProductError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Warehouses
// ---------------------------------------------------------------------------

// ListWarehouses handles GET /warehouses.
func (h *ProductHandler) ListWarehouses(c *gin.Context) {
	filter := &service.WarehouseFilter{
		CompanyID:  appctx.GetCompanyID(c),
		SearchTerm: c.Query("search"),
		IsActive:   parseOptionalBool(c, "is_active"),
	}
	filter.Page, filter.PageSize = parsePageParams(c, 50)

	warehouses, total, err := h.service.ListWarehouses(c.Request.Context(), filter)
	if err != nil {
		respondProductError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromWarehouses(warehouses),
		listMeta(total, filter.Page, filter.PageSize),
	))
}

// GetWarehouse handles GET /warehouses/:id.
func (h *ProductHandler) GetWarehouse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid warehouse ID")
		return
	}

	warehouse, err := h.service.GetWarehouse(c.Request.Context(), appctx.GetCompanyID(c), id)
	if err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromWarehouse(warehouse)))
}

// CreateWarehouse handles POST /warehouses.
func (h *ProductHandler) CreateWarehouse(c *gin.Context) {
	var req dto.CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	warehouse := &domain.Warehouse{
		TenantModel: domain.TenantModel{CompanyID: appctx.GetCompanyID(c)},
		Code:        req.Code,
		Name:        req.Name,
		Address:     req.Address,
		Manager:     req.Manager,
		Phone:       req.Phone,
		IsActive:    true,
	}
	if req.IsDefault != nil {
		warehouse.IsDefault = *req.IsDefault
	}
	if req.IsActive != nil {
		warehouse.IsActive = *req.IsActive
	}

	if err := h.service.CreateWarehouse(c.Request.Context(), warehouse); err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromWarehouse(warehouse)))
}

// UpdateWarehouse handles PUT /warehouses/:id.
func (h *ProductHandler) UpdateWarehouse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid warehouse ID")
		return
	}

	var req dto.UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithDetail(c, http.StatusBadRequest, errors.CodeValidation, "Validation failed", err.Error())
		return
	}

	companyID := appctx.GetCompanyID(c)
	warehouse, err := h.service.GetWarehouse(c.Request.Context(), companyID, id)
	if err != nil {
		respondProductError(c, err)
		return
	}

	warehouse.Code = req.Code
	warehouse.Name = req.Name
	warehouse.Address = req.Address
	warehouse.Manager = req.Manager
	warehouse.Phone = req.Phone
	if req.IsDefault != nil {
		warehouse.IsDefault = *req.IsDefault
	}
	if req.IsActive != nil {
		warehouse.IsActive = *req.IsActive
	}

	if err := h.service.UpdateWarehouse(c.Request.Context(), warehouse); err != nil {
		respondProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromWarehouse(warehouse)))
}

// DeleteWarehouse handles DELETE /warehouses/:id.
func (h *ProductHandler) DeleteWarehouse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.CodeInvalidFormat, "Invalid warehouse ID")
		return
	}

	if err := h.service.DeleteWarehouse(c.Request.Context(), appctx.GetCompanyID(c), id); err != nil {
		respondProductError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
