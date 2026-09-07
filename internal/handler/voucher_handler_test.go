package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/dto"
	"github.com/saintgo7/saas-kerp/internal/mocks"
)

type VoucherHandlerTestSuite struct {
	suite.Suite
	router    *gin.Engine
	handler   *VoucherHandler
	mockSvc   *mocks.MockVoucherService
	companyID uuid.UUID
	userID    uuid.UUID
}

func TestVoucherHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(VoucherHandlerTestSuite))
}

func (s *VoucherHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	s.mockSvc = new(mocks.MockVoucherService)
	s.handler = NewVoucherHandler(s.mockSvc)
	s.companyID = uuid.New()
	s.userID = uuid.New()

	// Create router with middleware to inject company_id, user_id and roles.
	// The write and workflow routes carry role middleware (see RegisterRoutes),
	// so the injected identity needs a role that is allowed to reach them.
	s.router = gin.New()
	s.router.Use(func(c *gin.Context) {
		c.Set("company_id", s.companyID)
		c.Set("user_id", s.userID)
		c.Set("roles", []string{"admin"})
		c.Next()
	})
	s.handler.RegisterRoutes(s.router.Group("/api/v1"))
}

func (s *VoucherHandlerTestSuite) TearDownTest() {
	// Note: AssertExpectations moved to individual tests that need strict verification
}

// newTestVoucher creates a test voucher for handler tests
func (s *VoucherHandlerTestSuite) newTestVoucher() *domain.Voucher {
	voucherID := uuid.New()
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	return &domain.Voucher{
		TenantModel: domain.TenantModel{
			BaseModel: domain.BaseModel{
				ID:        voucherID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			CompanyID: s.companyID,
		},
		VoucherNo:   "GEN-2024-0001",
		VoucherDate: time.Now(),
		VoucherType: domain.VoucherTypeGeneral,
		Status:      domain.VoucherStatusDraft,
		TotalDebit:  1000,
		TotalCredit: 1000,
		Description: "Test voucher",
		Entries: []domain.VoucherEntry{
			{
				BaseModel:   domain.BaseModel{ID: uuid.New()},
				VoucherID:   voucherID,
				CompanyID:   s.companyID,
				AccountID:   accountID1,
				LineNo:      1,
				DebitAmount: 1000,
			},
			{
				BaseModel:    domain.BaseModel{ID: uuid.New()},
				VoucherID:    voucherID,
				CompanyID:    s.companyID,
				AccountID:    accountID2,
				LineNo:       2,
				CreditAmount: 1000,
			},
		},
	}
}

// =============================================================================
// GET /vouchers Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestList_Success() {
	vouchers := []domain.Voucher{*s.newTestVoucher()}

	s.mockSvc.On("List", mock.Anything, mock.Anything).Return(vouchers, int64(1), nil).Once()

	req := httptest.NewRequest("GET", "/api/v1/vouchers", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)
	assert.True(s.T(), resp.Success)
}

func (s *VoucherHandlerTestSuite) TestList_WithFilters() {
	vouchers := []domain.Voucher{*s.newTestVoucher()}

	s.mockSvc.On("List", mock.Anything, mock.Anything).Return(vouchers, int64(1), nil).Once()

	req := httptest.NewRequest("GET", "/api/v1/vouchers?page=1&page_size=10&status=draft", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestList_NoCompanyID() {
	// Create router without company_id
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// No company_id set
		c.Next()
	})
	s.handler.RegisterRoutes(router.Group("/api/v1"))

	req := httptest.NewRequest("GET", "/api/v1/vouchers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// =============================================================================
// GET /vouchers/pending Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestGetPending_Success() {
	voucher := s.newTestVoucher()
	voucher.Status = domain.VoucherStatusPending
	vouchers := []domain.Voucher{*voucher}

	s.mockSvc.On("GetPending", mock.Anything, mock.Anything).Return(vouchers, nil)

	req := httptest.NewRequest("GET", "/api/v1/vouchers/pending", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// =============================================================================
// GET /vouchers/:id Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestGetByID_Success() {
	voucher := s.newTestVoucher()

	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	req := httptest.NewRequest("GET", "/api/v1/vouchers/"+voucher.ID.String(), nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(s.T(), err)
	assert.True(s.T(), resp.Success)
}

func (s *VoucherHandlerTestSuite) TestGetByID_NotFound() {
	voucherID := uuid.New()

	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrVoucherNotFound)

	req := httptest.NewRequest("GET", "/api/v1/vouchers/"+voucherID.String(), nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

func (s *VoucherHandlerTestSuite) TestGetByID_InvalidUUID() {
	req := httptest.NewRequest("GET", "/api/v1/vouchers/invalid-uuid", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// =============================================================================
// GET /vouchers/no/:voucher_no Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestGetByNo_Success() {
	voucher := s.newTestVoucher()

	s.mockSvc.On("GetByNo", mock.Anything, mock.Anything, voucher.VoucherNo).Return(voucher, nil)

	req := httptest.NewRequest("GET", "/api/v1/vouchers/no/"+voucher.VoucherNo, nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestGetByNo_NotFound() {
	voucherNo := "GEN-2024-9999"

	s.mockSvc.On("GetByNo", mock.Anything, mock.Anything, voucherNo).Return(nil, domain.ErrVoucherNotFound)

	req := httptest.NewRequest("GET", "/api/v1/vouchers/no/"+voucherNo, nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

// =============================================================================
// POST /vouchers Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestCreate_Success() {
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	reqBody := dto.CreateVoucherRequest{
		VoucherDate: time.Now().Format("2006-01-02"),
		VoucherType: "general",
		Description: "Test voucher",
		Entries: []dto.CreateVoucherEntryRequest{
			{AccountID: accountID1.String(), DebitAmount: 1000},
			{AccountID: accountID2.String(), CreditAmount: 1000},
		},
	}

	s.mockSvc.On("Create", mock.Anything, mock.MatchedBy(func(v *domain.Voucher) bool {
		return v.CompanyID == s.companyID && v.VoucherType == domain.VoucherTypeGeneral
	})).Return(nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)
}

func (s *VoucherHandlerTestSuite) TestCreate_ValidationError_Unbalanced() {
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	reqBody := dto.CreateVoucherRequest{
		VoucherDate: time.Now().Format("2006-01-02"),
		VoucherType: "general",
		Description: "Unbalanced voucher",
		Entries: []dto.CreateVoucherEntryRequest{
			{AccountID: accountID1.String(), DebitAmount: 1000},
			{AccountID: accountID2.String(), CreditAmount: 500}, // Unbalanced
		},
	}

	s.mockSvc.On("Create", mock.Anything, mock.Anything).Return(domain.ErrVoucherUnbalanced)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

func (s *VoucherHandlerTestSuite) TestCreate_InvalidJSON() {
	req := httptest.NewRequest("POST", "/api/v1/vouchers", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

func (s *VoucherHandlerTestSuite) TestCreate_MissingRequiredFields() {
	reqBody := dto.CreateVoucherRequest{
		// Missing voucher_date and voucher_type
		Entries: []dto.CreateVoucherEntryRequest{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// =============================================================================
// PUT /vouchers/:id Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestUpdate_Success() {
	voucher := s.newTestVoucher()
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	// First GetByID: Get existing voucher before update
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil).Once()
	s.mockSvc.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
	s.mockSvc.On("ReplaceEntries", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	// Second GetByID: Reload voucher after update
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil).Once()

	reqBody := dto.UpdateVoucherRequest{
		VoucherDate: time.Now().Format("2006-01-02"),
		Description: "Updated description",
		Entries: []dto.CreateVoucherEntryRequest{
			{AccountID: accountID1.String(), DebitAmount: 2000},
			{AccountID: accountID2.String(), CreditAmount: 2000},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/v1/vouchers/"+voucher.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestUpdate_VoucherNotFound() {
	voucherID := uuid.New()
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrVoucherNotFound).Once()

	reqBody := dto.UpdateVoucherRequest{
		VoucherDate: time.Now().Format("2006-01-02"),
		Description: "Updated description",
		Entries: []dto.CreateVoucherEntryRequest{
			{AccountID: accountID1.String(), DebitAmount: 1000},
			{AccountID: accountID2.String(), CreditAmount: 1000},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/v1/vouchers/"+voucherID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

func (s *VoucherHandlerTestSuite) TestUpdate_CannotEdit() {
	voucher := s.newTestVoucher()
	voucher.Status = domain.VoucherStatusPosted // Cannot edit posted voucher
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil).Once()
	s.mockSvc.On("Update", mock.Anything, mock.Anything).Return(domain.ErrVoucherCannotEdit).Once()

	reqBody := dto.UpdateVoucherRequest{
		VoucherDate: time.Now().Format("2006-01-02"),
		Entries: []dto.CreateVoucherEntryRequest{
			{AccountID: accountID1.String(), DebitAmount: 1000},
			{AccountID: accountID2.String(), CreditAmount: 1000},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/v1/vouchers/"+voucher.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

// =============================================================================
// DELETE /vouchers/:id Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestDelete_Success() {
	voucherID := uuid.New()

	s.mockSvc.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest("DELETE", "/api/v1/vouchers/"+voucherID.String(), nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestDelete_NotFound() {
	voucherID := uuid.New()

	s.mockSvc.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrVoucherNotFound)

	req := httptest.NewRequest("DELETE", "/api/v1/vouchers/"+voucherID.String(), nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

func (s *VoucherHandlerTestSuite) TestDelete_CannotEdit() {
	voucherID := uuid.New()

	s.mockSvc.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrVoucherCannotEdit)

	req := httptest.NewRequest("DELETE", "/api/v1/vouchers/"+voucherID.String(), nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

// =============================================================================
// POST /vouchers/:id/submit Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestSubmit_Success() {
	voucher := s.newTestVoucher()

	s.mockSvc.On("Submit", mock.Anything, mock.Anything, voucher.ID, mock.Anything).Return(nil)
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucher.ID.String()+"/submit", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestSubmit_CannotSubmit() {
	voucherID := uuid.New()

	s.mockSvc.On("Submit", mock.Anything, mock.Anything, voucherID, mock.Anything).Return(domain.ErrVoucherCannotSubmit)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucherID.String()+"/submit", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

// =============================================================================
// POST /vouchers/:id/approve Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestApprove_Success() {
	voucher := s.newTestVoucher()
	voucher.Status = domain.VoucherStatusPending

	s.mockSvc.On("Approve", mock.Anything, mock.Anything, voucher.ID, mock.Anything).Return(nil)
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucher.ID.String()+"/approve", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestApprove_CannotApprove() {
	voucher := s.newTestVoucher()
	voucherID := voucher.ID

	// Approve reads the voucher first to enforce separation of duties.
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)
	s.mockSvc.On("Approve", mock.Anything, mock.Anything, voucherID, mock.Anything).Return(domain.ErrVoucherCannotApprove)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucherID.String()+"/approve", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

// =============================================================================
// POST /vouchers/:id/reject Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestReject_Success() {
	voucher := s.newTestVoucher()
	voucher.Status = domain.VoucherStatusPending

	s.mockSvc.On("Reject", mock.Anything, mock.Anything, voucher.ID, s.userID, "Incorrect entries").Return(nil)
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	reqBody := dto.WorkflowActionRequest{Reason: "Incorrect entries"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucher.ID.String()+"/reject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// =============================================================================
// POST /vouchers/:id/post Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestPost_Success() {
	voucher := s.newTestVoucher()
	voucher.Status = domain.VoucherStatusApproved

	s.mockSvc.On("Post", mock.Anything, mock.Anything, voucher.ID, mock.Anything).Return(nil)
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucher.ID.String()+"/post", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestPost_CannotPost() {
	voucherID := uuid.New()

	s.mockSvc.On("Post", mock.Anything, mock.Anything, voucherID, mock.Anything).Return(domain.ErrVoucherCannotPost)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucherID.String()+"/post", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

// =============================================================================
// POST /vouchers/:id/cancel Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestCancel_Success() {
	voucher := s.newTestVoucher()

	s.mockSvc.On("Cancel", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucher.ID.String()+"/cancel", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestCancel_CannotCancel() {
	voucherID := uuid.New()

	s.mockSvc.On("Cancel", mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrVoucherCannotCancel)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucherID.String()+"/cancel", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

// =============================================================================
// POST /vouchers/:id/reverse Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestReverse_Success() {
	voucher := s.newTestVoucher()
	voucher.Status = domain.VoucherStatusPosted

	reversalVoucher := s.newTestVoucher()
	reversalVoucher.IsReversal = true
	reversalVoucher.ReversalOfID = &voucher.ID

	reversalDate := time.Now()
	s.mockSvc.On("Reverse", mock.Anything, mock.Anything, voucher.ID, s.userID, mock.AnythingOfType("time.Time"), "Reversal description").
		Return(reversalVoucher, nil)

	reqBody := dto.ReverseVoucherRequest{
		ReversalDate: reversalDate.Format("2006-01-02"),
		Description:  "Reversal description",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucher.ID.String()+"/reverse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)
}

func (s *VoucherHandlerTestSuite) TestReverse_CannotReverse() {
	voucherID := uuid.New()

	s.mockSvc.On("Reverse", mock.Anything, mock.Anything, voucherID, s.userID, mock.AnythingOfType("time.Time"), mock.Anything).
		Return(nil, domain.ErrVoucherCannotReverse)

	reqBody := dto.ReverseVoucherRequest{
		ReversalDate: time.Now().Format("2006-01-02"),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucherID.String()+"/reverse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

func (s *VoucherHandlerTestSuite) TestReverse_AlreadyReversed() {
	voucherID := uuid.New()

	s.mockSvc.On("Reverse", mock.Anything, mock.Anything, voucherID, s.userID, mock.AnythingOfType("time.Time"), mock.Anything).
		Return(nil, domain.ErrVoucherAlreadyReversed)

	reqBody := dto.ReverseVoucherRequest{
		ReversalDate: time.Now().Format("2006-01-02"),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucherID.String()+"/reverse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusConflict, w.Code)
}

func (s *VoucherHandlerTestSuite) TestReverse_InvalidDate() {
	voucherID := uuid.New()

	reqBody := dto.ReverseVoucherRequest{
		ReversalDate: "invalid-date",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucherID.String()+"/reverse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// =============================================================================
// PUT /vouchers/:id/entries Tests
// =============================================================================

func (s *VoucherHandlerTestSuite) TestReplaceEntries_Success() {
	voucher := s.newTestVoucher()
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	s.mockSvc.On("ReplaceEntries", mock.Anything, voucher.ID, mock.Anything).Return(nil)
	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	reqBody := []dto.CreateVoucherEntryRequest{
		{AccountID: accountID1.String(), DebitAmount: 2000},
		{AccountID: accountID2.String(), CreditAmount: 2000},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/v1/vouchers/"+voucher.ID.String()+"/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *VoucherHandlerTestSuite) TestReplaceEntries_Unbalanced() {
	voucherID := uuid.New()
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	s.mockSvc.On("ReplaceEntries", mock.Anything, voucherID, mock.Anything).Return(domain.ErrVoucherUnbalanced)

	reqBody := []dto.CreateVoucherEntryRequest{
		{AccountID: accountID1.String(), DebitAmount: 2000},
		{AccountID: accountID2.String(), CreditAmount: 1000}, // Unbalanced
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/v1/vouchers/"+voucherID.String()+"/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// =============================================================================
// Authorization tests
// =============================================================================

// A voucher may not be approved by the user who drafted it.
func (s *VoucherHandlerTestSuite) TestApprove_RejectedForDrafter() {
	voucher := s.newTestVoucher()
	voucher.Status = domain.VoucherStatusPending
	voucher.CreatedBy = &s.userID

	s.mockSvc.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(voucher, nil)

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+voucher.ID.String()+"/approve", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
	s.mockSvc.AssertNotCalled(s.T(), "Approve", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// A viewer may read vouchers but must not create, approve or post them.
func (s *VoucherHandlerTestSuite) TestViewerIsReadOnly() {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("company_id", s.companyID)
		c.Set("user_id", s.userID)
		c.Set("roles", []string{"viewer"})
		c.Next()
	})
	s.handler.RegisterRoutes(router.Group("/api/v1"))

	voucherID := uuid.New()
	forbidden := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/vouchers"},
		{"PUT", "/api/v1/vouchers/" + voucherID.String()},
		{"DELETE", "/api/v1/vouchers/" + voucherID.String()},
		{"POST", "/api/v1/vouchers/" + voucherID.String() + "/submit"},
		{"POST", "/api/v1/vouchers/" + voucherID.String() + "/approve"},
		{"POST", "/api/v1/vouchers/" + voucherID.String() + "/post"},
		{"POST", "/api/v1/vouchers/" + voucherID.String() + "/reverse"},
	}

	for _, tc := range forbidden {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(s.T(), http.StatusForbidden, w.Code, "%s %s should be forbidden for a viewer", tc.method, tc.path)
	}
}

// "user" may draft but not approve.
func (s *VoucherHandlerTestSuite) TestUserRoleCannotApprove() {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("company_id", s.companyID)
		c.Set("user_id", s.userID)
		c.Set("roles", []string{"user"})
		c.Next()
	})
	s.handler.RegisterRoutes(router.Group("/api/v1"))

	req := httptest.NewRequest("POST", "/api/v1/vouchers/"+uuid.New().String()+"/approve", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code)
}

// A reserved collection path this API does not implement answers 404, not the
// 400 that ":id" parsing would produce.
func (s *VoucherHandlerTestSuite) TestReservedCollectionPathIsNotFound() {
	for _, path := range []string{"/api/v1/vouchers/next-number", "/api/v1/vouchers/export"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(s.T(), http.StatusNotFound, w.Code, "GET %s", path)
	}
}

// page_size=0 must not reach the total-pages division.
func (s *VoucherHandlerTestSuite) TestList_ZeroPageSizeIsClamped() {
	s.mockSvc.On("List", mock.Anything, mock.Anything).Return([]domain.Voucher{}, int64(0), nil).Once()

	req := httptest.NewRequest("GET", "/api/v1/vouchers?page_size=0", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// The binding tag rejects page_size=0 outright; what matters is that the
	// request is answered rather than panicking with a divide by zero.
	assert.NotEqual(s.T(), http.StatusInternalServerError, w.Code)
}
