package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// UserHandler handles HTTP requests for users
type UserHandler struct {
	service service.UserService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// RegisterRoutes registers user routes
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", h.List)
		users.POST("", h.Create)
		users.GET("/stats", h.GetStats)
		users.GET("/:id", h.GetByID)
		users.PUT("/:id", h.Update)
		users.DELETE("/:id", h.Delete)
		users.PUT("/:id/password", h.ChangePassword)
		users.POST("/:id/activate", h.Activate)
		users.POST("/:id/deactivate", h.Deactivate)
	}
}

// Create handles POST /users
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	companyID := appctx.GetCompanyID(c)

	user, err := req.ToUser(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_002", err.Error()))
		return
	}

	if err := h.service.Create(c.Request.Context(), user); err != nil {
		switch err {
		case service.ErrUserEmailExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Email already exists"))
		case domain.ErrPasswordTooShort:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_003", "Password must be at least 8 characters"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromUser(user)))
}

// List handles GET /users
func (h *UserHandler) List(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	filter := repository.UserFilter{
		CompanyID:  companyID,
		SearchTerm: c.Query("search"),
		Page:       1,
		PageSize:   20,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
			filter.PageSize = ps
		}
	}
	if status := c.Query("status"); status != "" {
		s := domain.UserStatus(status)
		if s.IsValid() {
			filter.Status = &s
		}
	}
	if role := c.Query("role"); role != "" {
		r := domain.UserRole(role)
		if r.IsValid() {
			filter.Role = &r
		}
	}

	users, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromUsers(users),
		&dto.MetaInfo{
			Total:      total,
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			TotalPages: int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize)),
		},
	))
}

// GetByID handles GET /users/:id
func (h *UserHandler) GetByID(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid user ID"))
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromUser(user)))
}

// Update handles PUT /users/:id
func (h *UserHandler) Update(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid user ID"))
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	// Get existing user
	user, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	// Apply updates
	req.ApplyTo(user)

	if err := h.service.Update(c.Request.Context(), user); err != nil {
		switch err {
		case service.ErrUserEmailExists:
			c.JSON(http.StatusConflict, dto.ErrorResponse("BIZ_001", "Email already exists"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromUser(user)))
}

// Delete handles DELETE /users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	currentUserID := appctx.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid user ID"))
		return
	}

	// Prevent self-deletion
	if id == currentUserID {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_002", "Cannot delete your own account"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// ChangePassword handles PUT /users/:id/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	currentUserID := appctx.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid user ID"))
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_001", err.Error()))
		return
	}

	// Check if user is changing their own password or is an admin
	isAdmin := appctx.HasRole(c, "admin")
	if id != currentUserID && !isAdmin {
		c.JSON(http.StatusForbidden, dto.ErrorResponse("AUTH_001", "Not authorized to change this user's password"))
		return
	}

	// If admin is changing someone else's password, skip current password check
	if id != currentUserID && isAdmin {
		if err := h.service.ResetPassword(c.Request.Context(), companyID, id, req.NewPassword); err != nil {
			if err == domain.ErrUserNotFound {
				c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "User not found"))
				return
			}
			if err == domain.ErrPasswordTooShort {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_003", "Password must be at least 8 characters"))
				return
			}
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
			return
		}
	} else {
		if err := h.service.ChangePassword(c.Request.Context(), companyID, id, req.CurrentPassword, req.NewPassword); err != nil {
			if err == domain.ErrUserNotFound {
				c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "User not found"))
				return
			}
			if err == service.ErrInvalidCurrentPassword {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_005", "Invalid current password"))
				return
			}
			if err == domain.ErrPasswordTooShort {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_003", "Password must be at least 8 characters"))
				return
			}
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
			return
		}
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Password changed successfully"}))
}

// Activate handles POST /users/:id/activate
func (h *UserHandler) Activate(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid user ID"))
		return
	}

	if err := h.service.Activate(c.Request.Context(), companyID, id); err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"activated": true}))
}

// Deactivate handles POST /users/:id/deactivate
func (h *UserHandler) Deactivate(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)
	currentUserID := appctx.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VAL_004", "Invalid user ID"))
		return
	}

	// Prevent self-deactivation
	if id == currentUserID {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("BIZ_003", "Cannot deactivate your own account"))
		return
	}

	if err := h.service.Deactivate(c.Request.Context(), companyID, id); err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("RES_001", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"deactivated": true}))
}

// GetStats handles GET /users/stats
func (h *UserHandler) GetStats(c *gin.Context) {
	companyID := appctx.GetCompanyID(c)

	stats, err := h.service.GetStats(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SRV_001", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.UserStatsResponse{
		TotalCount:    stats.TotalCount,
		ActiveCount:   stats.ActiveCount,
		InactiveCount: stats.InactiveCount,
		LockedCount:   stats.LockedCount,
		AdminCount:    stats.AdminCount,
		UserCount:     stats.UserCount,
		ViewerCount:   stats.ViewerCount,
	}))
}
