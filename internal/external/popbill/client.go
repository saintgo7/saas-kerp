// Package popbill provides a client for the Popbill API.
// Popbill is a Korean ASP service for tax invoice management.
package popbill

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// API endpoints
	ProductionURL = "https://popbill.linkhub.co.kr"
	SandboxURL    = "https://popbill-test.linkhub.co.kr"

	// Service types
	ServiceTypeTaxInvoice = "TAXINVOICE"
)

// Config holds Popbill API configuration.
type Config struct {
	LinkID       string
	SecretKey    string
	IsSandbox    bool
	CorpNum      string // Business registration number
	UserID       string // Popbill user ID
	Timeout      time.Duration
}

// Client provides methods for interacting with Popbill API.
type Client struct {
	config     *Config
	httpClient *http.Client
	baseURL    string
	token      *accessToken
}

// accessToken represents Popbill API access token.
type accessToken struct {
	SessionToken string    `json:"session_token"`
	ServiceID    string    `json:"serviceID"`
	LinkID       string    `json:"linkID"`
	UserCode     string    `json:"usercode"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// NewClient creates a new Popbill API client.
func NewClient(config *Config) *Client {
	baseURL := ProductionURL
	if config.IsSandbox {
		baseURL = SandboxURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// getToken retrieves or refreshes the API access token.
func (c *Client) getToken(ctx context.Context) (*accessToken, error) {
	// Check if existing token is valid
	if c.token != nil && time.Now().Before(c.token.ExpiresAt.Add(-5*time.Minute)) {
		return c.token, nil
	}

	// Generate authentication data
	authData := c.generateAuthData()

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/TAXINVOICE/Token", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("x-lh-date", authData.timestamp)
	req.Header.Set("x-lh-version", "2.0")
	req.Header.Set("Authorization", authData.signature)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var token accessToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Set expiration (typically 1 hour)
	token.ExpiresAt = time.Now().Add(50 * time.Minute)
	c.token = &token

	return &token, nil
}

// authData holds authentication data for API requests.
type authData struct {
	timestamp string
	signature string
}

// generateAuthData generates authentication data for API requests.
func (c *Client) generateAuthData() *authData {
	timestamp := time.Now().UTC().Format("20060102150405")

	// Create signature using HMAC-SHA256
	message := fmt.Sprintf("%s%s%s", c.config.LinkID, timestamp, ServiceTypeTaxInvoice)
	hash := sha256.Sum256([]byte(message + c.config.SecretKey))
	signature := base64.StdEncoding.EncodeToString(hash[:])

	return &authData{
		timestamp: timestamp,
		signature: fmt.Sprintf("LINKHUB %s %s", c.config.LinkID, signature),
	}
}

// doRequest performs an authenticated API request.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.SessionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-pb-userid", c.config.UserID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp PopbillError
		if json.Unmarshal(respBody, &errResp) == nil {
			return nil, &errResp
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// PopbillError represents a Popbill API error.
type PopbillError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *PopbillError) Error() string {
	return fmt.Sprintf("Popbill error %d: %s", e.Code, e.Message)
}

// TaxInvoice represents a tax invoice for Popbill API.
type TaxInvoice struct {
	// Basic info
	WriteDate              string `json:"writeDate"`              // 작성일자 (YYYYMMDD)
	ChargeDirection        string `json:"chargeDirection"`        // 과금방향 (정과금/역과금)
	IssueType              string `json:"issueType"`              // 발행형태 (정발행/역발행/위수탁)
	TaxType                string `json:"taxType"`                // 과세형태 (과세/면세/영세)
	PurposeType            string `json:"purposeType"`            // 영수/청구

	// Supplier info
	InvoicerCorpNum        string `json:"invoicerCorpNum"`        // 공급자 사업자번호
	InvoicerCorpName       string `json:"invoicerCorpName"`       // 공급자 상호
	InvoicerCEOName        string `json:"invoicerCEOName"`        // 공급자 대표자명
	InvoicerAddr           string `json:"invoicerAddr"`           // 공급자 주소
	InvoicerBizType        string `json:"invoicerBizType"`        // 공급자 업태
	InvoicerBizClass       string `json:"invoicerBizClass"`       // 공급자 종목
	InvoicerContactName    string `json:"invoicerContactName"`    // 담당자명
	InvoicerEmail          string `json:"invoicerEmail"`          // 담당자 이메일

	// Buyer info
	InvoiceeType           string `json:"invoiceeType"`           // 공급받는자 유형 (사업자/개인/외국인)
	InvoiceeCorpNum        string `json:"invoiceeCorpNum"`        // 공급받는자 사업자번호
	InvoiceeCorpName       string `json:"invoiceeCorpName"`       // 공급받는자 상호
	InvoiceeCEOName        string `json:"invoiceeCEOName"`        // 공급받는자 대표자명
	InvoiceeAddr           string `json:"invoiceeAddr"`           // 공급받는자 주소
	InvoiceeBizType        string `json:"invoiceeBizType"`        // 공급받는자 업태
	InvoiceeBizClass       string `json:"invoiceeBizClass"`       // 공급받는자 종목
	InvoiceeContactName1   string `json:"invoiceeContactName1"`   // 담당자명
	InvoiceeEmail1         string `json:"invoiceeEmail1"`         // 담당자 이메일

	// Amount info
	SupplyCostTotal        string `json:"supplyCostTotal"`        // 공급가액 합계
	TaxTotal               string `json:"taxTotal"`               // 세액 합계
	TotalAmount            string `json:"totalAmount"`            // 합계금액

	// Items
	DetailList             []TaxInvoiceDetail `json:"detailList"` // 품목 리스트

	// Etc
	Remark1                string `json:"remark1"`                // 비고1
	NTSConfirmNum          string `json:"ntsconfirmNum"`          // 국세청 승인번호 (응답용)
}

// TaxInvoiceDetail represents a line item in a tax invoice.
type TaxInvoiceDetail struct {
	SerialNum     int    `json:"serialNum"`     // 품목 일련번호
	PurchaseDT    string `json:"purchaseDT"`    // 거래일자
	ItemName      string `json:"itemName"`      // 품목명
	Spec          string `json:"spec"`          // 규격
	Qty           string `json:"qty"`           // 수량
	UnitCost      string `json:"unitCost"`      // 단가
	SupplyCost    string `json:"supplyCost"`    // 공급가액
	Tax           string `json:"tax"`           // 세액
	Remark        string `json:"remark"`        // 비고
}

// IssueTaxInvoice issues a tax invoice.
func (c *Client) IssueTaxInvoice(ctx context.Context, invoice *TaxInvoice) (*IssueResponse, error) {
	path := fmt.Sprintf("/TAXINVOICE/%s", c.config.CorpNum)

	respBody, err := c.doRequest(ctx, "POST", path, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to issue tax invoice: %w", err)
	}

	var resp IssueResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse issue response: %w", err)
	}

	return &resp, nil
}

// IssueResponse represents the response from issuing a tax invoice.
type IssueResponse struct {
	Code          int    `json:"code"`
	Message       string `json:"message"`
	NTSConfirmNum string `json:"ntsConfirmNum"` // 국세청 승인번호
	ItemKey       string `json:"itemKey"`       // 팝빌 문서번호
}

// GetTaxInvoice retrieves a tax invoice by item key.
func (c *Client) GetTaxInvoice(ctx context.Context, itemKey string) (*TaxInvoice, error) {
	path := fmt.Sprintf("/TAXINVOICE/%s/%s", c.config.CorpNum, itemKey)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get tax invoice: %w", err)
	}

	var invoice TaxInvoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse tax invoice: %w", err)
	}

	return &invoice, nil
}

// SearchRequest represents a search request for tax invoices.
type SearchRequest struct {
	DType      string   `json:"DType"`      // 검색일자 유형 (W/I/S)
	SDate      string   `json:"SDate"`      // 시작일자
	EDate      string   `json:"EDate"`      // 종료일자
	State      []string `json:"State"`      // 상태코드
	Type       []string `json:"Type"`       // 문서형태
	TaxType    []string `json:"TaxType"`    // 과세형태
	Page       int      `json:"Page"`       // 페이지번호
	PerPage    int      `json:"PerPage"`    // 페이지당 건수
}

// SearchResponse represents the response from searching tax invoices.
type SearchResponse struct {
	Code       int          `json:"code"`
	Message    string       `json:"message"`
	Total      int          `json:"total"`
	PerPage    int          `json:"perPage"`
	PageNum    int          `json:"pageNum"`
	PageCount  int          `json:"pageCount"`
	List       []TaxInvoice `json:"list"`
}

// SearchTaxInvoices searches for tax invoices.
func (c *Client) SearchTaxInvoices(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	path := fmt.Sprintf("/TAXINVOICE/%s/Search", c.config.CorpNum)

	respBody, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search tax invoices: %w", err)
	}

	var resp SearchResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return &resp, nil
}

// CancelTaxInvoice cancels a tax invoice.
func (c *Client) CancelTaxInvoice(ctx context.Context, itemKey, memo string) error {
	path := fmt.Sprintf("/TAXINVOICE/%s/%s/Cancel", c.config.CorpNum, itemKey)

	body := map[string]string{"memo": memo}
	_, err := c.doRequest(ctx, "POST", path, body)
	if err != nil {
		return fmt.Errorf("failed to cancel tax invoice: %w", err)
	}

	return nil
}

// GetBalance retrieves the remaining balance (API usage credits).
func (c *Client) GetBalance(ctx context.Context) (float64, error) {
	path := fmt.Sprintf("/TAXINVOICE/%s/Balance", c.config.CorpNum)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	var result struct {
		Balance float64 `json:"balance"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse balance response: %w", err)
	}

	return result.Balance, nil
}

// encryptSecretKey encrypts the secret key for storage using AES.
func encryptSecretKey(key []byte, plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// decryptSecretKey decrypts the secret key from storage.
func decryptSecretKey(key []byte, ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
