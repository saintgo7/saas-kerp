package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"go.uber.org/zap"

	appctx "github.com/saintgo7/saas-kerp/internal/context"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/errors"
	"github.com/saintgo7/saas-kerp/internal/handler/response"
	"github.com/saintgo7/saas-kerp/internal/middleware"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// VoucherHandler handles HTTP requests for vouchers
type VoucherHandler struct {
	service service.VoucherService
}

// NewVoucherHandler creates a new VoucherHandler
func NewVoucherHandler(service service.VoucherService) *VoucherHandler {
	return &VoucherHandler{service: service}
}

// RegisterRoutes registers voucher routes
func (h *VoucherHandler) RegisterRoutes(r *gin.RouterGroup) {
	vouchers := r.Group("/vouchers")
	{
		vouchers.GET("", h.List)
		vouchers.GET("/pending", h.GetPending)
		vouchers.GET("/no/:voucher_no", h.GetByNo)

		// Reserved collection segments. They are registered explicitly, ahead of
		// the ":id" wildcard, so that a request for an endpoint this API does not
		// implement answers 404 instead of being parsed as a voucher ID and
		// answering 400 "Invalid voucher ID".
		vouchers.GET("/next-number", notImplementedRoute)
		vouchers.GET("/export", notImplementedRoute)

		vouchers.GET("/:id", h.GetByID)

		// Writes: admin or user.
		vouchers.POST("", middleware.RequireWriter(), h.Create)
		vouchers.PUT("/:id", middleware.RequireWriter(), h.Update)
		vouchers.DELETE("/:id", middleware.RequireWriter(), h.Delete)

		// Entry operations
		vouchers.PUT("/:id/entries", middleware.RequireWriter(), h.ReplaceEntries)

		// Workflow. Submitting is a write; approving, posting, cancelling and
		// reversing change the books and are restricted to approvers.
		vouchers.POST("/:id/submit", middleware.RequireWriter(), h.Submit)
		vouchers.POST("/:id/approve", middleware.RequireApprover(), h.Approve)
		vouchers.POST("/:id/reject", middleware.RequireApprover(), h.Reject)
		vouchers.POST("/:id/post", middleware.RequireApprover(), h.Post)
		vouchers.POST("/:id/cancel", middleware.RequireApprover(), h.Cancel)
		vouchers.POST("/:id/reverse", middleware.RequireApprover(), h.Reverse)
	}
}

// getCompanyID extracts company_id from context
func (h *VoucherHandler) getCompanyID(c *gin.Context) (uuid.UUID, bool) {
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

// getUserID extracts user_id from context
func (h *VoucherHandler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(dto.ErrCodeUnauthorized, "User ID not found"))
		return uuid.Nil, false
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(dto.ErrCodeUnauthorized, "Invalid user ID"))
		return uuid.Nil, false
	}
	return userID, true
}

// drafterIs reports whether userID drafted or submitted the voucher.
func drafterIs(v *domain.Voucher, userID uuid.UUID) bool {
	if v.CreatedBy != nil && *v.CreatedBy == userID {
		return true
	}
	if v.SubmittedBy != nil && *v.SubmittedBy == userID {
		return true
	}
	return false
}

// respondWithVoucher reloads a voucher after a successful state change and
// writes it to the response.
//
// The state change has already been committed at this point. Discarding the
// reload error and calling dto.FromVoucher on a nil voucher panics
// (voucher.ID.String()); reporting the reload failure as 500 is almost as bad,
// because the client then retries an operation that in fact succeeded and the
// retry fails with a state-transition conflict. So: log the reload failure and
// answer with the identifier of the voucher that was changed.
func (h *VoucherHandler) respondWithVoucher(c *gin.Context, companyID, id uuid.UUID, status int, action string) {
	voucher, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil || voucher == nil {
		if logger := appctx.GetLogger(c); logger != nil {
			logger.Error("failed to reload voucher after "+action,
				zap.Error(err),
				zap.String("voucher_id", id.String()),
			)
		}
		c.JSON(status, dto.SuccessResponse(dto.VoucherResponse{ID: id.String()}))
		return
	}

	c.JSON(status, dto.SuccessResponse(dto.FromVoucher(voucher)))
}

// List returns a list of vouchers with filtering and pagination
// @Summary List vouchers
// @Description Get a paginated list of vouchers
// @Tags vouchers
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers [get]
func (h *VoucherHandler) List(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.VoucherListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	// Clamp the page window. The binding tags already reject page_size<1 and
	// page_size>100, but the clamp is what guarantees the division below can
	// never see a zero, whatever the tags say tomorrow.
	req.Page, req.PageSize = response.ClampPagination(req.Page, req.PageSize)

	// Build filter
	filter := repository.VoucherFilter{
		CompanyID:      companyID,
		SearchTerm:     req.Search,
		IncludeEntries: req.IncludeEntries,
		Page:           req.Page,
		PageSize:       req.PageSize,
		SortBy:         req.SortBy,
		SortDesc:       req.SortDesc,
	}

	if req.VoucherType != "" {
		voucherType := domain.VoucherType(req.VoucherType)
		filter.VoucherType = &voucherType
	}
	if req.Status != "" {
		status := domain.VoucherStatus(req.Status)
		filter.Status = &status
	}
	if req.DateFrom != "" {
		dateFrom, err := time.Parse("2006-01-02", req.DateFrom)
		if err == nil {
			filter.DateFrom = &dateFrom
		}
	}
	if req.DateTo != "" {
		dateTo, err := time.Parse("2006-01-02", req.DateTo)
		if err == nil {
			filter.DateTo = &dateTo
		}
	}
	if req.AccountID != "" {
		accountID, err := uuid.Parse(req.AccountID)
		if err == nil {
			filter.AccountID = &accountID
		}
	}
	if req.PartnerID != "" {
		partnerID, err := uuid.Parse(req.PartnerID)
		if err == nil {
			filter.PartnerID = &partnerID
		}
	}
	if req.DepartmentID != "" {
		deptID, err := uuid.Parse(req.DepartmentID)
		if err == nil {
			filter.DepartmentID = &deptID
		}
	}

	vouchers, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve vouchers"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		dto.FromVouchers(vouchers),
		listMeta(total, req.Page, req.PageSize),
	))
}

// GetPending returns vouchers pending approval
// @Summary Get pending vouchers
// @Description Get vouchers pending approval
// @Tags vouchers
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/pending [get]
func (h *VoucherHandler) GetPending(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	vouchers, err := h.service.GetPending(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve pending vouchers"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromVouchers(vouchers)))
}

// GetByID returns a voucher by ID
// @Summary Get voucher by ID
// @Description Get a single voucher by its ID
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id} [get]
func (h *VoucherHandler) GetByID(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	voucher, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrVoucherNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve voucher"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromVoucher(voucher)))
}

// GetByNo returns a voucher by voucher number
// @Summary Get voucher by number
// @Description Get a single voucher by its voucher number
// @Tags vouchers
// @Accept json
// @Produce json
// @Param voucher_no path string true "Voucher number"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/no/{voucher_no} [get]
func (h *VoucherHandler) GetByNo(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	voucherNo := c.Param("voucher_no")
	if voucherNo == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Voucher number is required"))
		return
	}

	voucher, err := h.service.GetByNo(c.Request.Context(), companyID, voucherNo)
	if err != nil {
		if err == domain.ErrVoucherNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve voucher"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromVoucher(voucher)))
}

// Create creates a new voucher
// @Summary Create voucher
// @Description Create a new voucher with entries
// @Tags vouchers
// @Accept json
// @Produce json
// @Param voucher body dto.CreateVoucherRequest true "Voucher data"
// @Success 201 {object} dto.Response
// @Router /api/v1/vouchers [post]
func (h *VoucherHandler) Create(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req dto.CreateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	voucher, err := req.ToVoucher(companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid voucher data", err.Error()))
		return
	}

	if err := h.service.Create(c.Request.Context(), voucher); err != nil {
		switch err {
		case domain.ErrVoucherUnbalanced:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Debit and credit must be equal"))
		case domain.ErrVoucherNoEntries:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Voucher must have at least one entry"))
		case domain.ErrEntryInvalidAmount:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Entry must have either debit or credit, not both"))
		case domain.ErrEntryZeroAmount:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Entry amount must be greater than zero"))
		case domain.ErrControlAccountPosting:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Cannot post to control account"))
		case domain.ErrAccountNotFound:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Account not found"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to create voucher"))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromVoucher(voucher)))
}

// Update updates an existing voucher
// @Summary Update voucher
// @Description Update an existing voucher
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Param voucher body dto.UpdateVoucherRequest true "Voucher data"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id} [put]
func (h *VoucherHandler) Update(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	var req dto.UpdateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	// Get existing voucher
	voucher, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrVoucherNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve voucher"))
		return
	}

	// Update fields
	voucherDate, err := time.Parse("2006-01-02", req.VoucherDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher date"))
		return
	}
	voucher.VoucherDate = voucherDate
	voucher.Description = req.Description
	voucher.ReferenceType = req.ReferenceType
	voucher.UpdatedBy = &userID

	if req.ReferenceID != "" {
		refID, err := uuid.Parse(req.ReferenceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid reference ID"))
			return
		}
		voucher.ReferenceID = &refID
	} else {
		voucher.ReferenceID = nil
	}

	if err := h.service.Update(c.Request.Context(), voucher); err != nil {
		if err == domain.ErrVoucherCannotEdit {
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be edited in current status"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to update voucher"))
		return
	}

	// Replace entries if provided
	if len(req.Entries) > 0 {
		var entries []domain.VoucherEntry
		for _, entryReq := range req.Entries {
			entry, err := entryReq.ToEntry(companyID)
			if err != nil {
				c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid entry data", err.Error()))
				return
			}
			entries = append(entries, *entry)
		}

		if err := h.service.ReplaceEntries(c.Request.Context(), id, entries); err != nil {
			switch err {
			case domain.ErrVoucherUnbalanced:
				c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Debit and credit must be equal"))
			case domain.ErrVoucherCannotEdit:
				c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be edited in current status"))
			default:
				c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to update entries"))
			}
			return
		}
	}

	// Reload voucher
	h.respondWithVoucher(c, companyID, id, http.StatusOK, "update")
}

// Delete removes a voucher
// @Summary Delete voucher
// @Description Delete a voucher by ID
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id} [delete]
func (h *VoucherHandler) Delete(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), companyID, id); err != nil {
		switch err {
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		case domain.ErrVoucherCannotEdit:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be deleted in current status"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to delete voucher"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Voucher deleted successfully"}))
}

// ReplaceEntries replaces all entries of a voucher
// @Summary Replace voucher entries
// @Description Replace all entries of a voucher
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Param entries body []dto.CreateVoucherEntryRequest true "Entries"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id}/entries [put]
func (h *VoucherHandler) ReplaceEntries(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	var req []dto.CreateVoucherEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	var entries []domain.VoucherEntry
	for _, entryReq := range req {
		entry, err := entryReq.ToEntry(companyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid entry data", err.Error()))
			return
		}
		entries = append(entries, *entry)
	}

	if err := h.service.ReplaceEntries(c.Request.Context(), id, entries); err != nil {
		switch err {
		case domain.ErrVoucherUnbalanced:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Debit and credit must be equal"))
		case domain.ErrVoucherCannotEdit:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be edited in current status"))
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to replace entries"))
		}
		return
	}

	h.respondWithVoucher(c, companyID, id, http.StatusOK, "state change")
}

// Submit submits a voucher for approval
// @Summary Submit voucher for approval
// @Description Submit a voucher for approval
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id}/submit [post]
func (h *VoucherHandler) Submit(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	if err := h.service.Submit(c.Request.Context(), companyID, id, userID); err != nil {
		switch err {
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		case domain.ErrVoucherCannotSubmit:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be submitted in current status"))
		case domain.ErrVoucherUnbalanced:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Voucher is not balanced"))
		case domain.ErrVoucherNoEntries:
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Voucher has no entries"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to submit voucher"))
		}
		return
	}

	h.respondWithVoucher(c, companyID, id, http.StatusOK, "state change")
}

// Approve approves a voucher
// @Summary Approve voucher
// @Description Approve a pending voucher
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id}/approve [post]
func (h *VoucherHandler) Approve(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	// Separation of duties: whoever drafted or submitted a voucher may not
	// approve it. Without this, a single account can create, approve and post an
	// entry, and the approval workflow records nothing but that account's own
	// signature twice.
	//
	// This check is best-effort at the HTTP layer: it reads the voucher and then
	// calls Approve, so a concurrent change between the two is not covered. The
	// authoritative place for the rule is VoucherService.Approve, inside the same
	// transaction as the state change.
	existing, err := h.service.GetByID(c.Request.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrVoucherNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
			return
		}
		response.InternalErrorLogged(c, "Failed to approve voucher", err)
		return
	}
	if existing != nil && drafterIs(existing, userID) {
		c.JSON(http.StatusForbidden, dto.ErrorResponse(errors.CodeForbidden,
			"A voucher cannot be approved by the user who drafted or submitted it"))
		return
	}

	if err := h.service.Approve(c.Request.Context(), companyID, id, userID); err != nil {
		switch err {
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		case domain.ErrVoucherCannotApprove:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be approved in current status"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to approve voucher"))
		}
		return
	}

	h.respondWithVoucher(c, companyID, id, http.StatusOK, "state change")
}

// Reject rejects a voucher
// @Summary Reject voucher
// @Description Reject a pending voucher
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Param body body dto.WorkflowActionRequest true "Rejection reason"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id}/reject [post]
func (h *VoucherHandler) Reject(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	var req dto.WorkflowActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	if err := h.service.Reject(c.Request.Context(), companyID, id, userID, req.Reason); err != nil {
		switch err {
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		case domain.ErrVoucherCannotReject:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be rejected in current status"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to reject voucher"))
		}
		return
	}

	h.respondWithVoucher(c, companyID, id, http.StatusOK, "state change")
}

// Post posts a voucher to the ledger
// @Summary Post voucher
// @Description Post an approved voucher to the ledger
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id}/post [post]
func (h *VoucherHandler) Post(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	if err := h.service.Post(c.Request.Context(), companyID, id, userID); err != nil {
		switch err {
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		case domain.ErrVoucherCannotPost:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher cannot be posted in current status"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to post voucher"))
		}
		return
	}

	h.respondWithVoucher(c, companyID, id, http.StatusOK, "state change")
}

// Cancel cancels a voucher
// @Summary Cancel voucher
// @Description Cancel a voucher (not posted)
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/vouchers/{id}/cancel [post]
func (h *VoucherHandler) Cancel(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	if err := h.service.Cancel(c.Request.Context(), companyID, id); err != nil {
		switch err {
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		case domain.ErrVoucherCannotCancel:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Posted vouchers cannot be cancelled, use reversal instead"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to cancel voucher"))
		}
		return
	}

	h.respondWithVoucher(c, companyID, id, http.StatusOK, "state change")
}

// Reverse creates a reversal voucher
// @Summary Reverse voucher
// @Description Create a reversal voucher for a posted voucher
// @Tags vouchers
// @Accept json
// @Produce json
// @Param id path string true "Voucher ID"
// @Param body body dto.ReverseVoucherRequest true "Reversal details"
// @Success 201 {object} dto.Response
// @Router /api/v1/vouchers/{id}/reverse [post]
func (h *VoucherHandler) Reverse(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid voucher ID"))
		return
	}

	var req dto.ReverseVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	reversalDate, err := time.Parse("2006-01-02", req.ReversalDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid reversal date"))
		return
	}

	reversal, err := h.service.Reverse(c.Request.Context(), companyID, id, userID, reversalDate, req.Description)
	if err != nil {
		switch err {
		case domain.ErrVoucherNotFound:
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Voucher not found"))
		case domain.ErrVoucherCannotReverse:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Only posted vouchers can be reversed"))
		case domain.ErrVoucherAlreadyReversed:
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Voucher has already been reversed"))
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to create reversal voucher"))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromVoucher(reversal)))
}
