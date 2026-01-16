package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// LedgerHandler handles HTTP requests for ledger and reports
type LedgerHandler struct {
	ledgerService  service.LedgerService
	accountService service.AccountService
}

// NewLedgerHandler creates a new LedgerHandler
func NewLedgerHandler(ledgerService service.LedgerService, accountService service.AccountService) *LedgerHandler {
	return &LedgerHandler{
		ledgerService:  ledgerService,
		accountService: accountService,
	}
}

// RegisterRoutes registers ledger routes
func (h *LedgerHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Ledger routes
	ledger := r.Group("/ledger")
	{
		ledger.GET("/balances", h.GetPeriodBalances)
		ledger.GET("/account", h.GetAccountLedger)
		ledger.POST("/recalculate", h.RecalculateBalances)
	}

	// Report routes
	reports := r.Group("/reports")
	{
		reports.GET("/trial-balance", h.GetTrialBalance)
		reports.GET("/trial-balance/range", h.GetTrialBalanceRange)
		reports.GET("/balance-sheet", h.GetBalanceSheet)
		reports.GET("/income-statement", h.GetIncomeStatement)
	}

	// Fiscal period routes
	periods := r.Group("/fiscal-periods")
	{
		periods.GET("", h.GetFiscalPeriods)
		periods.GET("/:year/:month", h.GetFiscalPeriod)
		periods.POST("/create/:year", h.CreateFiscalPeriods)
		periods.POST("/close", h.ClosePeriod)
		periods.POST("/reopen", h.ReopenPeriod)
		periods.POST("/year-end-close", h.YearEndClose)
	}
}

// getCompanyID extracts company_id from context
func (h *LedgerHandler) getCompanyID(c *gin.Context) (uuid.UUID, bool) {
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
func (h *LedgerHandler) getUserID(c *gin.Context) (uuid.UUID, bool) {
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

// GetPeriodBalances returns all ledger balances for a period
// @Summary Get period balances
// @Description Get all ledger balances for a fiscal period
// @Tags ledger
// @Accept json
// @Produce json
// @Param year query int true "Fiscal year"
// @Param month query int true "Fiscal month"
// @Success 200 {object} dto.Response
// @Router /api/v1/ledger/balances [get]
func (h *LedgerHandler) GetPeriodBalances(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.PeriodRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	balances, err := h.ledgerService.GetPeriodBalances(c.Request.Context(), companyID, req.Year, req.Month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve balances"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromLedgerBalances(balances)))
}

// GetAccountLedger returns detailed ledger entries for an account
// @Summary Get account ledger
// @Description Get detailed ledger entries for a specific account
// @Tags ledger
// @Accept json
// @Produce json
// @Param account_id query string true "Account ID"
// @Param from_date query string true "From date (YYYY-MM-DD)"
// @Param to_date query string true "To date (YYYY-MM-DD)"
// @Success 200 {object} dto.Response
// @Router /api/v1/ledger/account [get]
func (h *LedgerHandler) GetAccountLedger(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.AccountLedgerRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid account ID"))
		return
	}

	fromDate, err := time.Parse("2006-01-02", req.FromDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid from_date format"))
		return
	}

	toDate, err := time.Parse("2006-01-02", req.ToDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid to_date format"))
		return
	}

	// Get account info
	account, err := h.accountService.GetByID(c.Request.Context(), companyID, accountID)
	if err != nil {
		if err == domain.ErrAccountNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Account not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve account"))
		return
	}

	// Get ledger entries
	entries, openingBalance, err := h.ledgerService.GetAccountLedger(c.Request.Context(), companyID, accountID, fromDate, toDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve ledger"))
		return
	}

	// Calculate totals
	var totalDebit, totalCredit float64
	entryResponses := make([]dto.AccountLedgerEntryResponse, len(entries))
	for i, entry := range entries {
		entryResponses[i] = dto.FromAccountLedgerEntry(&entry)
		totalDebit += entry.DebitAmount
		totalCredit += entry.CreditAmount
	}

	closingBalance := openingBalance + totalDebit - totalCredit

	response := dto.AccountLedgerResponse{
		AccountID:      accountID.String(),
		AccountCode:    account.Code,
		AccountName:    account.Name,
		FromDate:       fromDate.Format("2006-01-02"),
		ToDate:         toDate.Format("2006-01-02"),
		OpeningBalance: openingBalance,
		TotalDebit:     totalDebit,
		TotalCredit:    totalCredit,
		ClosingBalance: closingBalance,
		Entries:        entryResponses,
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response))
}

// RecalculateBalances recalculates ledger balances from posted vouchers
// @Summary Recalculate balances
// @Description Recalculate ledger balances from posted vouchers
// @Tags ledger
// @Accept json
// @Produce json
// @Param body body dto.PeriodRequest true "Period"
// @Success 200 {object} dto.Response
// @Router /api/v1/ledger/recalculate [post]
func (h *LedgerHandler) RecalculateBalances(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.PeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	if err := h.ledgerService.RecalculateBalances(c.Request.Context(), companyID, req.Year, req.Month); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to recalculate balances"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Balances recalculated successfully"}))
}

// GetTrialBalance generates a trial balance report
// @Summary Get trial balance
// @Description Generate a trial balance report for a fiscal period
// @Tags reports
// @Accept json
// @Produce json
// @Param year query int true "Fiscal year"
// @Param month query int true "Fiscal month"
// @Success 200 {object} dto.Response
// @Router /api/v1/reports/trial-balance [get]
func (h *LedgerHandler) GetTrialBalance(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.PeriodRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	tb, err := h.ledgerService.GetTrialBalance(c.Request.Context(), companyID, req.Year, req.Month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to generate trial balance"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromTrialBalance(tb)))
}

// GetTrialBalanceRange generates a trial balance for a date range
// @Summary Get trial balance range
// @Description Generate a trial balance report for a date range
// @Tags reports
// @Accept json
// @Produce json
// @Param from_year query int true "From year"
// @Param from_month query int true "From month"
// @Param to_year query int true "To year"
// @Param to_month query int true "To month"
// @Success 200 {object} dto.Response
// @Router /api/v1/reports/trial-balance/range [get]
func (h *LedgerHandler) GetTrialBalanceRange(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	tb, err := h.ledgerService.GetTrialBalanceRange(c.Request.Context(), companyID, req.FromYear, req.FromMonth, req.ToYear, req.ToMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to generate trial balance"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromTrialBalance(tb)))
}

// GetBalanceSheet generates a balance sheet report
// @Summary Get balance sheet
// @Description Generate a balance sheet report
// @Tags reports
// @Accept json
// @Produce json
// @Param year query int true "Fiscal year"
// @Param month query int true "Fiscal month"
// @Success 200 {object} dto.Response
// @Router /api/v1/reports/balance-sheet [get]
func (h *LedgerHandler) GetBalanceSheet(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.PeriodRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	// Get trial balance
	tb, err := h.ledgerService.GetTrialBalance(c.Request.Context(), companyID, req.Year, req.Month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to generate balance sheet"))
		return
	}

	// Build balance sheet from trial balance
	var assets, liabilities, equity []dto.FinancialStatementItem
	var totalAssets, totalLiabilities, totalEquity float64

	for _, item := range tb.Items {
		fsItem := dto.FinancialStatementItem{
			Code:   item.AccountCode,
			Name:   item.AccountName,
			Amount: item.ClosingDebit - item.ClosingCredit,
			Level:  item.AccountLevel,
		}

		switch item.AccountType {
		case "asset":
			assets = append(assets, fsItem)
			totalAssets += fsItem.Amount
		case "liability":
			fsItem.Amount = item.ClosingCredit - item.ClosingDebit // Liability is credit balance
			liabilities = append(liabilities, fsItem)
			totalLiabilities += fsItem.Amount
		case "equity":
			fsItem.Amount = item.ClosingCredit - item.ClosingDebit // Equity is credit balance
			equity = append(equity, fsItem)
			totalEquity += fsItem.Amount
		}
	}

	response := dto.BalanceSheetResponse{
		CompanyID:        companyID.String(),
		AsOfDate:         tb.EndDate.Format("2006-01-02"),
		GeneratedAt:      dto.ReportGeneratedAt(),
		Assets:           assets,
		Liabilities:      liabilities,
		Equity:           equity,
		TotalAssets:      totalAssets,
		TotalLiabilities: totalLiabilities,
		TotalEquity:      totalEquity,
		IsBalanced:       totalAssets == (totalLiabilities + totalEquity),
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response))
}

// GetIncomeStatement generates an income statement report
// @Summary Get income statement
// @Description Generate an income statement report
// @Tags reports
// @Accept json
// @Produce json
// @Param from_year query int true "From year"
// @Param from_month query int true "From month"
// @Param to_year query int true "To year"
// @Param to_month query int true "To month"
// @Success 200 {object} dto.Response
// @Router /api/v1/reports/income-statement [get]
func (h *LedgerHandler) GetIncomeStatement(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.DateRangeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid query parameters", err.Error()))
		return
	}

	// Get trial balance for the range
	tb, err := h.ledgerService.GetTrialBalanceRange(c.Request.Context(), companyID, req.FromYear, req.FromMonth, req.ToYear, req.ToMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to generate income statement"))
		return
	}

	// Build income statement from trial balance
	var revenue, expenses []dto.FinancialStatementItem
	var totalRevenue, totalExpenses float64

	for _, item := range tb.Items {
		fsItem := dto.FinancialStatementItem{
			Code:   item.AccountCode,
			Name:   item.AccountName,
			Level:  item.AccountLevel,
		}

		switch item.AccountType {
		case "revenue":
			fsItem.Amount = item.ClosingCredit - item.ClosingDebit // Revenue is credit balance
			revenue = append(revenue, fsItem)
			totalRevenue += fsItem.Amount
		case "expense":
			fsItem.Amount = item.ClosingDebit - item.ClosingCredit // Expense is debit balance
			expenses = append(expenses, fsItem)
			totalExpenses += fsItem.Amount
		}
	}

	response := dto.IncomeStatementResponse{
		CompanyID:     companyID.String(),
		FromDate:      tb.StartDate.Format("2006-01-02"),
		ToDate:        tb.EndDate.Format("2006-01-02"),
		GeneratedAt:   dto.ReportGeneratedAt(),
		Revenue:       revenue,
		Expenses:      expenses,
		TotalRevenue:  totalRevenue,
		TotalExpenses: totalExpenses,
		NetIncome:     totalRevenue - totalExpenses,
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response))
}

// GetFiscalPeriods returns all fiscal periods for a year
// @Summary Get fiscal periods
// @Description Get all fiscal periods for a year
// @Tags fiscal-periods
// @Accept json
// @Produce json
// @Param year query int true "Fiscal year"
// @Success 200 {object} dto.Response
// @Router /api/v1/fiscal-periods [get]
func (h *LedgerHandler) GetFiscalPeriods(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	year := c.Query("year")
	if year == "" {
		year = time.Now().Format("2006")
	}

	var yearInt int
	if _, err := time.Parse("2006", year); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid year"))
		return
	}
	yearInt = time.Now().Year() // Default to current year if parsing issue

	periods, err := h.ledgerService.GetFiscalPeriods(c.Request.Context(), companyID, yearInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve fiscal periods"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromFiscalPeriods(periods)))
}

// GetFiscalPeriod returns a specific fiscal period
// @Summary Get fiscal period
// @Description Get a specific fiscal period
// @Tags fiscal-periods
// @Accept json
// @Produce json
// @Param year path int true "Fiscal year"
// @Param month path int true "Fiscal month"
// @Success 200 {object} dto.Response
// @Router /api/v1/fiscal-periods/{year}/{month} [get]
func (h *LedgerHandler) GetFiscalPeriod(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	// Parse year and month from path
	var year, month int
	if _, err := c.Params.Get("year"); err {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid year"))
		return
	}

	period, err := h.ledgerService.GetFiscalPeriod(c.Request.Context(), companyID, year, month)
	if err != nil {
		if err == domain.ErrFiscalPeriodNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Fiscal period not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to retrieve fiscal period"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(dto.FromFiscalPeriod(period)))
}

// CreateFiscalPeriods creates all 12 fiscal periods for a year
// @Summary Create fiscal periods
// @Description Create all 12 fiscal periods for a year
// @Tags fiscal-periods
// @Accept json
// @Produce json
// @Param year path int true "Fiscal year"
// @Success 201 {object} dto.Response
// @Router /api/v1/fiscal-periods/create/{year} [post]
func (h *LedgerHandler) CreateFiscalPeriods(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	// Parse year from path - simplified
	year := time.Now().Year()

	periods, err := h.ledgerService.CreateFiscalPeriods(c.Request.Context(), companyID, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to create fiscal periods"))
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(dto.FromFiscalPeriods(periods)))
}

// ClosePeriod closes a fiscal period
// @Summary Close fiscal period
// @Description Close a fiscal period
// @Tags fiscal-periods
// @Accept json
// @Produce json
// @Param body body dto.ClosePeriodRequest true "Period to close"
// @Success 200 {object} dto.Response
// @Router /api/v1/fiscal-periods/close [post]
func (h *LedgerHandler) ClosePeriod(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req dto.ClosePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	if err := h.ledgerService.ClosePeriod(c.Request.Context(), companyID, req.Year, req.Month, userID); err != nil {
		if err == domain.ErrFiscalPeriodNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Fiscal period not found"))
			return
		}
		if err == domain.ErrFiscalPeriodClosed {
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Fiscal period is already closed"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to close fiscal period"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Fiscal period closed successfully"}))
}

// ReopenPeriod reopens a closed fiscal period
// @Summary Reopen fiscal period
// @Description Reopen a closed fiscal period
// @Tags fiscal-periods
// @Accept json
// @Produce json
// @Param body body dto.ClosePeriodRequest true "Period to reopen"
// @Success 200 {object} dto.Response
// @Router /api/v1/fiscal-periods/reopen [post]
func (h *LedgerHandler) ReopenPeriod(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}

	var req dto.ClosePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	if err := h.ledgerService.ReopenPeriod(c.Request.Context(), companyID, req.Year, req.Month); err != nil {
		if err == domain.ErrFiscalPeriodNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(dto.ErrCodeNotFound, "Fiscal period not found"))
			return
		}
		if err == domain.ErrFiscalPeriodClosed {
			c.JSON(http.StatusConflict, dto.ErrorResponse(dto.ErrCodeConflict, "Fiscal period is locked and cannot be reopened"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to reopen fiscal period"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Fiscal period reopened successfully"}))
}

// YearEndClose performs year-end closing
// @Summary Year-end close
// @Description Perform year-end closing
// @Tags fiscal-periods
// @Accept json
// @Produce json
// @Param body body dto.YearEndCloseRequest true "Year-end close request"
// @Success 200 {object} dto.Response
// @Router /api/v1/fiscal-periods/year-end-close [post]
func (h *LedgerHandler) YearEndClose(c *gin.Context) {
	companyID, ok := h.getCompanyID(c)
	if !ok {
		return
	}
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req dto.YearEndCloseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(dto.ErrCodeValidation, "Invalid request body", err.Error()))
		return
	}

	retainedEarningsAccountID, err := uuid.Parse(req.RetainedEarningsAccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(dto.ErrCodeValidation, "Invalid retained earnings account ID"))
		return
	}

	if err := h.ledgerService.PerformYearEndClose(c.Request.Context(), companyID, req.Year, retainedEarningsAccountID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(dto.ErrCodeInternalServerError, "Failed to perform year-end close"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"message": "Year-end close completed successfully"}))
}
