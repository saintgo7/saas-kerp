package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// Valid 사업자등록번호 used throughout these tests. Both satisfy the check
// digit rule: digits 1..9 give a check digit of 1, and 2-1-4-8-1-5-9-2-4 gives 4.
const (
	validSupplierBRN = "1234567891"
	validBuyerBRN    = "2148159244"
)

func TestValidateBusinessNumber(t *testing.T) {
	t.Run("accepts valid numbers", func(t *testing.T) {
		assert.NoError(t, domain.ValidateBusinessNumber(validSupplierBRN))
		assert.NoError(t, domain.ValidateBusinessNumber(validBuyerBRN))
	})

	t.Run("rejects a wrong check digit", func(t *testing.T) {
		// 1234567891 is valid, so 1234567890 must not be.
		assert.ErrorIs(t, domain.ValidateBusinessNumber("1234567890"), domain.ErrInvalidBusinessNumber)
	})

	t.Run("rejects non-digits", func(t *testing.T) {
		// The old check was len(...) != 10, which transmitted this to the NTS.
		assert.ErrorIs(t, domain.ValidateBusinessNumber("abcdefghij"), domain.ErrInvalidBusinessNumber)
		assert.ErrorIs(t, domain.ValidateBusinessNumber("123-45-678"), domain.ErrInvalidBusinessNumber)
	})

	t.Run("rejects wrong lengths", func(t *testing.T) {
		assert.ErrorIs(t, domain.ValidateBusinessNumber("123456789"), domain.ErrInvalidBusinessNumber)
		assert.ErrorIs(t, domain.ValidateBusinessNumber("12345678912"), domain.ErrInvalidBusinessNumber)
		assert.ErrorIs(t, domain.ValidateBusinessNumber(""), domain.ErrInvalidBusinessNumber)
	})
}

func newTestTaxInvoice() *domain.TaxInvoice {
	return &domain.TaxInvoice{
		ID:                     uuid.New(),
		CompanyID:              uuid.New(),
		InvoiceNumber:          "20260907-0001",
		InvoiceType:            domain.TaxInvoiceTypeSales,
		IssueDate:              time.Now(),
		Status:                 domain.TaxInvoiceStatusDraft,
		SupplierBusinessNumber: validSupplierBRN,
		SupplierName:           "공급자 주식회사",
		BuyerBusinessNumber:    validBuyerBRN,
		BuyerName:              "공급받는자 주식회사",
		SupplyAmount:           10_000_000,
		TaxAmount:              1_000_000,
		TotalAmount:            11_000_000,
	}
}

func TestTaxInvoice_Validate(t *testing.T) {
	t.Run("accepts a well-formed taxable invoice", func(t *testing.T) {
		assert.NoError(t, newTestTaxInvoice().Validate())
	})

	t.Run("rejects zero tax on a taxable supply", func(t *testing.T) {
		// The audit case: supply_amount 10,000,000 with tax_amount 0 used to be
		// stored verbatim and filed as zero output VAT.
		inv := newTestTaxInvoice()
		inv.TaxAmount = 0
		inv.TotalAmount = 10_000_000
		inv.VATCategory = domain.VATCategoryTaxable

		assert.ErrorIs(t, inv.Validate(), domain.ErrVATAmountMismatch)
	})

	t.Run("accepts zero tax when the supply is zero-rated", func(t *testing.T) {
		inv := newTestTaxInvoice()
		inv.TaxAmount = 0
		inv.TotalAmount = 10_000_000
		inv.VATCategory = domain.VATCategoryZeroRated

		assert.NoError(t, inv.Validate())
	})

	t.Run("accepts zero tax when no category is declared", func(t *testing.T) {
		// With no category column the treatment is inferred, and tax 0 is a
		// legitimate 영세율/면세 invoice.
		inv := newTestTaxInvoice()
		inv.TaxAmount = 0
		inv.TotalAmount = 10_000_000

		assert.NoError(t, inv.Validate())
		assert.Equal(t, domain.VATCategoryZeroRated, inv.DeclaredVATCategory())
	})

	t.Run("rejects a tax amount that is not 10% of the supply", func(t *testing.T) {
		inv := newTestTaxInvoice()
		inv.TaxAmount = 500_000
		inv.TotalAmount = 10_500_000

		assert.ErrorIs(t, inv.Validate(), domain.ErrVATAmountMismatch)
	})

	t.Run("rejects negative amounts", func(t *testing.T) {
		inv := newTestTaxInvoice()
		inv.SupplyAmount = -10_000_000
		inv.TaxAmount = -1_000_000
		inv.TotalAmount = -11_000_000

		assert.ErrorIs(t, inv.Validate(), domain.ErrTaxInvoiceNegativeAmount)
	})

	t.Run("rejects a zero supply amount", func(t *testing.T) {
		inv := newTestTaxInvoice()
		inv.SupplyAmount = 0
		inv.TaxAmount = 0
		inv.TotalAmount = 0

		assert.ErrorIs(t, inv.Validate(), domain.ErrTaxInvoiceZeroAmount)
	})

	t.Run("rejects a total that is not supply + tax", func(t *testing.T) {
		inv := newTestTaxInvoice()
		inv.TotalAmount = 12_000_000

		assert.ErrorIs(t, inv.Validate(), domain.ErrTaxInvoiceTotalMismatch)
	})

	t.Run("rejects an invalid business number", func(t *testing.T) {
		inv := newTestTaxInvoice()
		inv.BuyerBusinessNumber = "abcdefghij"

		assert.ErrorIs(t, inv.Validate(), domain.ErrInvalidBusinessNumber)
	})

	t.Run("rejects a missing invoice number", func(t *testing.T) {
		inv := newTestTaxInvoice()
		inv.InvoiceNumber = ""

		assert.ErrorIs(t, inv.Validate(), domain.ErrTaxInvoiceNumberRequired)
	})
}

func TestTaxInvoice_ValidateItems(t *testing.T) {
	withItems := func(items ...domain.TaxInvoiceItem) *domain.TaxInvoice {
		inv := newTestTaxInvoice()
		inv.Items = items
		return inv
	}

	t.Run("accepts items that reconcile with the header", func(t *testing.T) {
		inv := withItems(
			domain.TaxInvoiceItem{Description: "품목1", Amount: 6_000_000, TaxAmount: 600_000},
			domain.TaxInvoiceItem{Description: "품목2", Amount: 4_000_000, TaxAmount: 400_000},
		)
		assert.NoError(t, inv.ValidateItems())
		assert.NoError(t, inv.ValidateForIssue())
	})

	t.Run("rejects the audit case: items carry tax the header does not", func(t *testing.T) {
		// POST /tax-invoices with supply_amount 10,000,000 / tax_amount 0 while
		// the items declare 1,000,000 of tax. Nothing used to compare the two.
		inv := withItems(
			domain.TaxInvoiceItem{Description: "품목1", Amount: 10_000_000, TaxAmount: 1_000_000},
		)
		inv.TaxAmount = 0
		inv.TotalAmount = 10_000_000

		assert.ErrorIs(t, inv.ValidateItems(), domain.ErrTaxInvoiceItemSumMismatch)
	})

	t.Run("rejects an item supply sum that differs from the header", func(t *testing.T) {
		inv := withItems(
			domain.TaxInvoiceItem{Description: "품목1", Amount: 9_000_000, TaxAmount: 1_000_000},
		)
		assert.ErrorIs(t, inv.ValidateItems(), domain.ErrTaxInvoiceItemSumMismatch)
	})

	t.Run("rejects an invoice with no items", func(t *testing.T) {
		inv := newTestTaxInvoice()
		assert.ErrorIs(t, inv.ValidateItems(), domain.ErrTaxInvoiceNoItems)
	})

	t.Run("rejects negative item amounts", func(t *testing.T) {
		inv := withItems(
			domain.TaxInvoiceItem{Description: "품목1", Amount: -1, TaxAmount: 0},
		)
		assert.ErrorIs(t, inv.ValidateItems(), domain.ErrTaxInvoiceNegativeAmount)
	})

	t.Run("rejects an all-zero item", func(t *testing.T) {
		inv := withItems(
			domain.TaxInvoiceItem{Description: "품목1", Amount: 10_000_000, TaxAmount: 1_000_000},
			domain.TaxInvoiceItem{Description: "빈 품목", Amount: 0, TaxAmount: 0},
		)
		assert.ErrorIs(t, inv.ValidateItems(), domain.ErrTaxInvoiceItemInvalid)
	})
}
