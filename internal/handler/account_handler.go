package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// AccountHandler handles HTTP requests for chart of accounts
type AccountHandler struct {
	service service.AccountService
}

// NewAccountHandler creates a new AccountHandler
func NewAccountHandler(svc service.AccountService) *AccountHandler {
	return &AccountHandler{service: svc}
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
		accounts.GET("/:id/can-delete", h.CanDelete)
		accounts.PUT("/:id/move", h.Move)
	}
}

// List handles GET /accounts
func (h *AccountHandler) List(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	filter := repository.AccountFilter{
		CompanyID:  companyID,
		SearchTerm: c.Query("search"),
		Page:       1,
		PageSize:   100,
	}

	if accountType := c.Query("type"); accountType != "" {
		at := domain.AccountType(accountType)
		filter.AccountType = &at
	}

	if page := c.Query("page"); page != "" {
		if p, err := parseInt(page); err == nil {
			filter.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := parseInt(pageSize); err == nil {
			filter.PageSize = ps
		}
	}
	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		filter.IsActive = &active
	}

	accounts, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromAccounts(accounts),
		&dto.MetaInfo{
			Total:      total,
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			TotalPages: int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize)),
		},
	))
}

// GetTree handles GET /accounts/tree
func (h *AccountHandler) GetTree(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	accounts, err := h.service.GetTree(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccounts(accounts)))
}

// GetByID handles GET /accounts/:id
func (h *AccountHandler) GetByID(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid account ID"))
		return
	}

	account, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Account not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccount(account)))
}

// GetByCode handles GET /accounts/code/:code
func (h *AccountHandler) GetByCode(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	code := c.Param("code")

	account, err := h.service.GetByCode(c.Request.Context(), companyID, code)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Account not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccount(account)))
}

// Create handles POST /accounts
func (h *AccountHandler) Create(c *gin.Context) {
	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	companyID := appctx.GetCompanyID(c)

	account, err := req.ToAccount(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", err.Error()))
		return
	}

	if err := h.service.Create(c.Request.Context(), account); err != nil {
		switch err {
		case domain.ErrAccountCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Account code already exists"))
		case domain.ErrParentNotFound:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_002", "Parent account not found"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromAccount(account)))
}

// Update handles PUT /accounts/:id
func (h *AccountHandler) Update(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid account ID"))
		return
	}

	var req dto.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	account, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Account not found"))
		return
	}

	// Apply updates using DTO helper
	if err := req.ApplyTo(account); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", err.Error()))
		return
	}

	if err := h.service.Update(c.Request.Context(), account); err != nil {
		switch err {
		case domain.ErrAccountCodeExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Account code already exists"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccount(account)))
}

// Delete handles DELETE /accounts/:id
func (h *AccountHandler) Delete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid account ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_004", err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// GetChildren handles GET /accounts/:id/children
func (h *AccountHandler) GetChildren(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid account ID"))
		return
	}

	children, err := h.service.GetChildren(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromAccounts(children)))
}

// CanDelete handles GET /accounts/:id/can-delete
func (h *AccountHandler) CanDelete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid account ID"))
		return
	}

	canDelete, reason, err := h.service.CanDelete(c.Request.Context(), companyID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "Account not found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"can_delete": canDelete,
		"reason":     reason,
	}))
}

// Move handles PUT /accounts/:id/move
func (h *AccountHandler) Move(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid account ID"))
		return
	}

	var req dto.MoveAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	var newParentID *uuid.UUID
	if req.ParentID != "" {
		parsed, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid parent ID"))
			return
		}
		newParentID = &parsed
	}

	if err := h.service.Move(c.Request.Context(), companyID, id, newParentID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_005", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"moved": true}))
}
