package dto

import (
	"github.com/saintgo7/saas-kerp/internal/domain"
)

// TaxInvoiceResponse is the wire representation of a tax invoice.
//
// It exists so the handler stops serialising *domain.TaxInvoice directly. The
// domain struct carries json tags on every field, including company_id,
// created_by/updated_by and the ASP provider identifiers, so any field added to
// it in the future would appear in the API without anyone deciding that it
// should.
type TaxInvoiceResponse struct {
	ID            string `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	InvoiceType   string `json:"invoice_type"`
	IssueDate     string `json:"issue_date"`
	Status        string `json:"status"`

	SupplierBusinessNumber string `json:"supplier_business_number"`
	SupplierName           string `json:"supplier_name"`
	SupplierCEOName        string `json:"supplier_ceo_name,omitempty"`
	SupplierAddress        string `json:"supplier_address,omitempty"`
	SupplierBusinessType   string `json:"supplier_business_type,omitempty"`
	SupplierBusinessItem   string `json:"supplier_business_item,omitempty"`
	SupplierEmail          string `json:"supplier_email,omitempty"`

	BuyerBusinessNumber string `json:"buyer_business_number"`
	BuyerName           string `json:"buyer_name"`
	BuyerCEOName        string `json:"buyer_ceo_name,omitempty"`
	BuyerAddress        string `json:"buyer_address,omitempty"`
	BuyerBusinessType   string `json:"buyer_business_type,omitempty"`
	BuyerBusinessItem   string `json:"buyer_business_item,omitempty"`
	BuyerEmail          string `json:"buyer_email,omitempty"`

	SupplyAmount int64 `json:"supply_amount"`
	TaxAmount    int64 `json:"tax_amount"`
	TotalAmount  int64 `json:"total_amount"`

	NTSConfirmNumber string `json:"nts_confirm_number,omitempty"`
	NTSTransmittedAt string `json:"nts_transmitted_at,omitempty"`
	NTSConfirmedAt   string `json:"nts_confirmed_at,omitempty"`

	VoucherID string `json:"voucher_id,omitempty"`

	Items []TaxInvoiceItemResponse `json:"items,omitempty"`

	Remarks   string `json:"remarks,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// TaxInvoiceItemResponse is one line of a tax invoice.
type TaxInvoiceItemResponse struct {
	ID             string  `json:"id"`
	SequenceNumber int     `json:"sequence_number"`
	SupplyDate     string  `json:"supply_date,omitempty"`
	Description    string  `json:"description"`
	Specification  string  `json:"specification,omitempty"`
	Quantity       float64 `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	Amount         int64   `json:"amount"`
	TaxAmount      int64   `json:"tax_amount"`
	Remarks        string  `json:"remarks,omitempty"`
}

// FromTaxInvoice converts a domain tax invoice to its wire representation.
// A nil invoice yields the zero response rather than a nil-pointer panic.
func FromTaxInvoice(invoice *domain.TaxInvoice) TaxInvoiceResponse {
	if invoice == nil {
		return TaxInvoiceResponse{}
	}

	resp := TaxInvoiceResponse{
		ID:            invoice.ID.String(),
		InvoiceNumber: invoice.InvoiceNumber,
		InvoiceType:   string(invoice.InvoiceType),
		IssueDate:     invoice.IssueDate.Format(dateLayout),
		Status:        string(invoice.Status),

		SupplierBusinessNumber: invoice.SupplierBusinessNumber,
		SupplierName:           invoice.SupplierName,
		SupplierCEOName:        invoice.SupplierCEOName,
		SupplierAddress:        invoice.SupplierAddress,
		SupplierBusinessType:   invoice.SupplierBusinessType,
		SupplierBusinessItem:   invoice.SupplierBusinessItem,
		SupplierEmail:          invoice.SupplierEmail,

		BuyerBusinessNumber: invoice.BuyerBusinessNumber,
		BuyerName:           invoice.BuyerName,
		BuyerCEOName:        invoice.BuyerCEOName,
		BuyerAddress:        invoice.BuyerAddress,
		BuyerBusinessType:   invoice.BuyerBusinessType,
		BuyerBusinessItem:   invoice.BuyerBusinessItem,
		BuyerEmail:          invoice.BuyerEmail,

		SupplyAmount: invoice.SupplyAmount,
		TaxAmount:    invoice.TaxAmount,
		TotalAmount:  invoice.TotalAmount,

		NTSConfirmNumber: invoice.NTSConfirmNumber,

		Remarks:   invoice.Remarks,
		CreatedAt: invoice.CreatedAt.Format(timestampLayout),
		UpdatedAt: invoice.UpdatedAt.Format(timestampLayout),
	}

	if invoice.NTSTransmittedAt != nil {
		resp.NTSTransmittedAt = invoice.NTSTransmittedAt.Format(timestampLayout)
	}
	if invoice.NTSConfirmedAt != nil {
		resp.NTSConfirmedAt = invoice.NTSConfirmedAt.Format(timestampLayout)
	}
	if invoice.VoucherID != nil {
		resp.VoucherID = invoice.VoucherID.String()
	}

	if len(invoice.Items) > 0 {
		resp.Items = make([]TaxInvoiceItemResponse, 0, len(invoice.Items))
		for i := range invoice.Items {
			resp.Items = append(resp.Items, fromTaxInvoiceItem(&invoice.Items[i]))
		}
	}

	return resp
}

// FromTaxInvoices converts a slice of domain tax invoices.
func FromTaxInvoices(invoices []*domain.TaxInvoice) []TaxInvoiceResponse {
	out := make([]TaxInvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, FromTaxInvoice(inv))
	}
	return out
}

func fromTaxInvoiceItem(item *domain.TaxInvoiceItem) TaxInvoiceItemResponse {
	resp := TaxInvoiceItemResponse{
		ID:             item.ID.String(),
		SequenceNumber: item.SequenceNumber,
		Description:    item.Description,
		Specification:  item.Specification,
		Quantity:       item.Quantity,
		UnitPrice:      item.UnitPrice,
		Amount:         item.Amount,
		TaxAmount:      item.TaxAmount,
		Remarks:        item.Remarks,
	}
	if item.SupplyDate != nil {
		resp.SupplyDate = item.SupplyDate.Format(dateLayout)
	}
	return resp
}

const (
	dateLayout      = "2006-01-02"
	timestampLayout = "2006-01-02T15:04:05Z07:00"
)
