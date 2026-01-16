package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// AccountHandler handles HTTP requests for accounts
type AccountHandler struct {
	service service.AccountService
}

// NewAccountHandler creates a new AccountHandler
func NewAccountHandler(service service.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// RegisterRoutes registers account routes
func (h *AccountHandler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/accounts")
	{
		accounts.GET("", h.List)
		accounts.GET("/tree", h.GetTree)
		accounts.GET("/:id", h.GetByID)
		accounts.GET("/code/:code", h.GetByCode)
		accounts.POST("", h.Create)
		accounts.PUT("/:id", h.Update)
		accounts.DELETE("/:id", h.Delete)
		accounts.GET("/:id/children", h.GetChildren)
		accounts.PATCH("/:id/move", h.Move)
		accounts.PATCH("/sort-order", h.UpdateSortOrder)
		accounts.GET("/:id/can-delete", h.CanDelete)
		accounts.GET("/:id/can-post", h.CanPost)
	}
}

// getCompanyID extracts company_id from context (set by tenant middleware)
func (h *AccountHandler) getCompanyID(c *gin.Context) (uuid.UUID, bool) {
	companyIDVal, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(dto.ErrCodeUnauthorized, "Company ID not found"))
		return uuid.Nil, false
	}
	companyID, ok := companyIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(dto.ErrCodeUnauthorized, "Invalid company ID"))
		return uuid.Nil, false
	}
	return companyID, true
}

// List returns a list of accounts with filtering and pagination
// @Summary List accounts
// @Description Get a paginated list of accounts
// @Tags accounts
// @Accept json
// @Produce json
// @Param parent_id query string false "Parent account ID"
// @Param account_type query string false "Account type (asset, liability, equity, revenue, expense)"
// @Param is_active query bool false "Filter by active status"
// @Param search query string false "Search term"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param sort_by query string false "Sort by field"
// @Param sort_desc query bool false "Sort descending"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts [get]
func (h *AccountHandler) List(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.AccountListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// Build filter
	filter := repository.AccountFilter{
		CompanyID:  companyID,
		SearchTerm: req.Search,
		IsActive:   req.IsActive,
		Page:       req.Page,
		PageSize:   req.PageSize,
		SortBy:     req.SortBy,
		SortDesc:   req.SortDesc,
	}

	if req.ParentID != "" {
		parentID, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid parent_id"))
			return
		}
		filter.ParentID = &parentID
	}

	if req.AccountType != "" {
		accountType := domain.AccountType(req.AccountType)
		filter.AccountType = &accountType
	}

	accounts, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve accounts"))
		return
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromAccounts(accounts),
		&dto.MetaInfo{
			Total:      total,
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalPages: totalPages,
		},
	))
}

// GetTree returns the full account tree
// @Summary Get account tree
// @Description Get the full hierarchical account tree
// @Tags accounts
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/tree [get]
func (h *AccountHandler) GetTree(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	accounts, err := h.service.GetTree(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve account tree"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccounts(accounts)))
}

// GetByID returns an account by ID
// @Summary Get account by ID
// @Description Get a single account by its ID
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/{id} [get]
func (h *AccountHandler) GetByID(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	account, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve account"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccount(account)))
}

// GetByCode returns an account by code
// @Summary Get account by code
// @Description Get a single account by its code
// @Tags accounts
// @Accept json
// @Produce json
// @Param code path string true "Account code"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/code/{code} [get]
func (h *AccountHandler) GetByCode(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Account code is required"))
		return
	}

	account, err := h.service.GetByCode(c.Request.Context(), companyID, code)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve account"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccount(account)))
}

// Create creates a new account
// @Summary Create account
// @Description Create a new account
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body dto.CreateAccountRequest true "Account data"
// @Success 201 {object} dto.Response
// @Router /api/v1/accounts [post]
func (h *AccountHandler) Create(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	account, err := req.ToAccount(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid account data", err.Error()))
		return
	}

	if err := h.service.Create(c.Request.Context(), account); err != nil {
		switch err {
		case domain.ErrAccountCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Account code already exists"))
		case domain.ErrParentNotFound:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Parent account not found"))
		case domain.ErrInvalidAccountType:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account type"))
		case domain.ErrInvalidAccountNature:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account nature"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to create account"))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromAccount(account)))
}

// Update updates an existing account
// @Summary Update account
// @Description Update an existing account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param account body dto.UpdateAccountRequest true "Account data"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/{id} [put]
func (h *AccountHandler) Update(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	var req dto.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	// Get existing account
	account, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve account"))
		return
	}

	// Apply updates
	if err := req.ApplyTo(account); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid update data", err.Error()))
		return
	}

	if err := h.service.Update(c.Request.Context(), account); err != nil {
		switch err {
		case domain.ErrAccountCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Account code already exists"))
		case domain.ErrParentNotFound:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Parent account not found"))
		case domain.ErrCircularReference:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Circular reference detected"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to update account"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccount(account)))
}

// Delete removes an account
// @Summary Delete account
// @Description Delete an account by ID
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/{id} [delete]
func (h *AccountHandler) Delete(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		switch err {
		case domain.ErrAccountNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
		case domain.ErrAccountHasChildren:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Cannot delete account with children"))
		case domain.ErrAccountHasEntries:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Cannot delete account with voucher entries"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to delete account"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Account deleted successfully"}))
}

// GetChildren returns children of an account
// @Summary Get account children
// @Description Get direct children of an account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Parent account ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/{id}/children [get]
func (h *AccountHandler) GetChildren(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	children, err := h.service.GetChildren(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve children"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccounts(children)))
}

// Move moves an account to a new parent
// @Summary Move account
// @Description Move an account to a new parent
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param body body dto.MoveAccountRequest true "New parent ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/{id}/move [patch]
func (h *AccountHandler) Move(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	var req dto.MoveAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	var newParentID *uuid.UUID
	if req.ParentID != "" {
		parentID, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid parent ID"))
			return
		}
		newParentID = &parentID
	}

	if err := h.service.Move(c.Request.Context(), companyID, id, newParentID); err != nil {
		switch err {
		case domain.ErrAccountNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
		case domain.ErrParentNotFound:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Parent account not found"))
		case domain.ErrCircularReference:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Circular reference detected"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to move account"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Account moved successfully"}))
}

// UpdateSortOrder updates sort orders for accounts
// @Summary Update sort order
// @Description Update sort order for multiple accounts
// @Tags accounts
// @Accept json
// @Produce json
// @Param body body dto.UpdateSortOrderRequest true "Sort orders"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/sort-order [patch]
func (h *AccountHandler) UpdateSortOrder(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.UpdateSortOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	orders := make(map[uuid.UUID]int)
	for _, item := range req.Orders {
		id, err := uuid.Parse(item.ID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID: "+item.ID))
			return
		}
		orders[id] = item.SortOrder
	}

	if err := h.service.UpdateSortOrder(c.Request.Context(), companyID, orders); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to update sort order"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Sort order updated successfully"}))
}

// CanDelete checks if an account can be deleted
// @Summary Check if account can be deleted
// @Description Check if an account can be deleted
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/{id}/can-delete [get]
func (h *AccountHandler) CanDelete(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to check delete status"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"can_delete": canDelete,
		"reason":     reason,
	}))
}

// CanPost checks if direct posting is allowed on an account
// @Summary Check if account can accept postings
// @Description Check if direct posting is allowed on an account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/accounts/{id}/can-post [get]
func (h *AccountHandler) CanPost(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	canPost, reason, err := h.service.CanPost(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to check posting status"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"can_post": canPost,
		"reason":   reason,
	}))
}
