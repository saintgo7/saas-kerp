// Package popbill provides service layer for Popbill integration.
package popbill

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// Service provides business logic for Popbill operations.
type Service struct {
	client *Client
}

// NewService creates a new Popbill service.
func NewService(config *Config) *Service {
	return &Service{
		client: NewClient(config),
	}
}

// IssueTaxInvoice issues a tax invoice via Popbill.
func (s *Service) IssueTaxInvoice(ctx context.Context, invoice *domain.TaxInvoice) (*domain.TaxInvoice, error) {
	// Convert domain model to Popbill format
	pbInvoice := &TaxInvoice{
		WriteDate:           invoice.IssueDate.Format("20060102"),
		ChargeDirection:     "정과금",
		IssueType:           "정발행",
		TaxType:             "과세",
		PurposeType:         "영수",

		InvoicerCorpNum:     invoice.SupplierBusinessNumber,
		InvoicerCorpName:    invoice.SupplierName,
		InvoicerCEOName:     invoice.SupplierCEOName,
		InvoicerAddr:        invoice.SupplierAddress,
		InvoicerBizType:     invoice.SupplierBusinessType,
		InvoicerBizClass:    invoice.SupplierBusinessItem,
		InvoicerEmail:       invoice.SupplierEmail,

		InvoiceeType:        "사업자",
		InvoiceeCorpNum:     invoice.BuyerBusinessNumber,
		InvoiceeCorpName:    invoice.BuyerName,
		InvoiceeCEOName:     invoice.BuyerCEOName,
		InvoiceeAddr:        invoice.BuyerAddress,
		InvoiceeBizType:     invoice.BuyerBusinessType,
		InvoiceeBizClass:    invoice.BuyerBusinessItem,
		InvoiceeEmail1:      invoice.BuyerEmail,

		SupplyCostTotal:     strconv.FormatInt(invoice.SupplyAmount, 10),
		TaxTotal:            strconv.FormatInt(invoice.TaxAmount, 10),
		TotalAmount:         strconv.FormatInt(invoice.TotalAmount, 10),

		Remark1:             invoice.Remarks,
	}

	// Convert items
	for i, item := range invoice.Items {
		supplyDate := ""
		if item.SupplyDate != nil {
			supplyDate = item.SupplyDate.Format("20060102")
		}

		pbInvoice.DetailList = append(pbInvoice.DetailList, TaxInvoiceDetail{
			SerialNum:   i + 1,
			PurchaseDT:  supplyDate,
			ItemName:    item.Description,
			Spec:        item.Specification,
			Qty:         strconv.FormatFloat(item.Quantity, 'f', 2, 64),
			UnitCost:    strconv.FormatFloat(item.UnitPrice, 'f', 0, 64),
			SupplyCost:  strconv.FormatInt(item.Amount, 10),
			Tax:         strconv.FormatInt(item.TaxAmount, 10),
			Remark:      item.Remarks,
		})
	}

	// Issue via Popbill
	resp, err := s.client.IssueTaxInvoice(ctx, pbInvoice)
	if err != nil {
		return nil, fmt.Errorf("popbill issue failed: %w", err)
	}

	// Update invoice with Popbill response
	invoice.NTSConfirmNumber = resp.NTSConfirmNum
	invoice.ASPProvider = "popbill"
	invoice.ASPInvoiceID = resp.ItemKey
	invoice.Status = domain.TaxInvoiceStatusTransmitted
	now := time.Now()
	invoice.NTSTransmittedAt = &now

	return invoice, nil
}

// GetTaxInvoice retrieves a tax invoice from Popbill.
func (s *Service) GetTaxInvoice(ctx context.Context, itemKey string) (*domain.TaxInvoice, error) {
	pbInvoice, err := s.client.GetTaxInvoice(ctx, itemKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice from popbill: %w", err)
	}

	// Convert Popbill format to domain model
	issueDate, _ := time.Parse("20060102", pbInvoice.WriteDate)
	supplyAmount, _ := strconv.ParseInt(pbInvoice.SupplyCostTotal, 10, 64)
	taxAmount, _ := strconv.ParseInt(pbInvoice.TaxTotal, 10, 64)
	totalAmount, _ := strconv.ParseInt(pbInvoice.TotalAmount, 10, 64)

	invoice := &domain.TaxInvoice{
		IssueDate:              issueDate,
		SupplierBusinessNumber: pbInvoice.InvoicerCorpNum,
		SupplierName:           pbInvoice.InvoicerCorpName,
		SupplierCEOName:        pbInvoice.InvoicerCEOName,
		SupplierAddress:        pbInvoice.InvoicerAddr,
		SupplierBusinessType:   pbInvoice.InvoicerBizType,
		SupplierBusinessItem:   pbInvoice.InvoicerBizClass,
		SupplierEmail:          pbInvoice.InvoicerEmail,
		BuyerBusinessNumber:    pbInvoice.InvoiceeCorpNum,
		BuyerName:              pbInvoice.InvoiceeCorpName,
		BuyerCEOName:           pbInvoice.InvoiceeCEOName,
		BuyerAddress:           pbInvoice.InvoiceeAddr,
		BuyerBusinessType:      pbInvoice.InvoiceeBizType,
		BuyerBusinessItem:      pbInvoice.InvoiceeBizClass,
		BuyerEmail:             pbInvoice.InvoiceeEmail1,
		SupplyAmount:           supplyAmount,
		TaxAmount:              taxAmount,
		TotalAmount:            totalAmount,
		NTSConfirmNumber:       pbInvoice.NTSConfirmNum,
		Remarks:                pbInvoice.Remark1,
	}

	return invoice, nil
}

// SearchTaxInvoices searches for tax invoices in Popbill.
func (s *Service) SearchTaxInvoices(ctx context.Context, startDate, endDate time.Time, page, pageSize int) ([]*domain.TaxInvoice, int, error) {
	req := &SearchRequest{
		DType:   "W",
		SDate:   startDate.Format("20060102"),
		EDate:   endDate.Format("20060102"),
		State:   []string{"3", "4"}, // 전송완료, 국세청승인
		Type:    []string{"N", "M"}, // 일반, 수정
		TaxType: []string{"T", "N", "Z"}, // 과세, 면세, 영세
		Page:    page,
		PerPage: pageSize,
	}

	resp, err := s.client.SearchTaxInvoices(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("popbill search failed: %w", err)
	}

	var invoices []*domain.TaxInvoice
	for _, pb := range resp.List {
		issueDate, _ := time.Parse("20060102", pb.WriteDate)
		supplyAmount, _ := strconv.ParseInt(pb.SupplyCostTotal, 10, 64)
		taxAmount, _ := strconv.ParseInt(pb.TaxTotal, 10, 64)
		totalAmount, _ := strconv.ParseInt(pb.TotalAmount, 10, 64)

		invoice := &domain.TaxInvoice{
			IssueDate:              issueDate,
			InvoiceType:            domain.TaxInvoiceTypeSales,
			Status:                 domain.TaxInvoiceStatusConfirmed,
			SupplierBusinessNumber: pb.InvoicerCorpNum,
			SupplierName:           pb.InvoicerCorpName,
			BuyerBusinessNumber:    pb.InvoiceeCorpNum,
			BuyerName:              pb.InvoiceeCorpName,
			SupplyAmount:           supplyAmount,
			TaxAmount:              taxAmount,
			TotalAmount:            totalAmount,
			NTSConfirmNumber:       pb.NTSConfirmNum,
			ASPProvider:            "popbill",
		}
		invoices = append(invoices, invoice)
	}

	return invoices, resp.Total, nil
}

// CancelTaxInvoice cancels a tax invoice in Popbill.
func (s *Service) CancelTaxInvoice(ctx context.Context, itemKey, reason string) error {
	return s.client.CancelTaxInvoice(ctx, itemKey, reason)
}

// GetBalance returns the remaining API credits.
func (s *Service) GetBalance(ctx context.Context) (float64, error) {
	return s.client.GetBalance(ctx)
}
