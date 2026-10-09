package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyPONumber    = errors.New("purchase order number cannot be empty")
	ErrEmptyVendorName  = errors.New("vendor name cannot be empty")
	ErrInvalidPODate    = errors.New("purchase order date is required")
	ErrNoItemsInPO      = errors.New("purchase order must contain at least one item")
	ErrInvalidItemQty   = errors.New("item quantity must be greater than zero")
	ErrInvalidItemCost  = errors.New("item cost cannot be negative")
)

// PurchaseOrderItem represents a single line item in a Purchase Order
type PurchaseOrderItem struct {
	ID          int     `json:"id"`
	POID        int     `json:"po_id"`
	ItemCode    string  `json:"item_code"`
	Description string  `json:"description"`
	UOM         string  `json:"uom"`
	Qty         float64 `json:"qty"`
	UnitCost    float64 `json:"unit_cost"`
	TotalAmount float64 `json:"total_amount"`
}

// PurchaseOrder represents a vendor procurement order aggregate
type PurchaseOrder struct {
	ID             int                 `json:"id"`
	PONo           string              `json:"po_no"`
	Date           time.Time           `json:"date"`
	DueDate        *time.Time          `json:"due_date,omitempty"`
	VendorName     string              `json:"vendor_name"`
	VendorTIN      string              `json:"vendor_tin"`
	VendorAddress  string              `json:"vendor_address"`
	VendorContact  string              `json:"vendor_contact"`
	ShipTo         string              `json:"ship_to"`
	PaymentTerms   string              `json:"payment_terms"`
	ShippingMethod string              `json:"shipping_method"`
	PreparedBy     string              `json:"prepared_by"`
	Freight        float64             `json:"freight"`
	ApplyVAT       bool                `json:"apply_vat"`
	VATAmt         float64             `json:"vat_amt"`
	Subtotal       float64             `json:"subtotal"`
	GrandTotal     float64             `json:"grand_total"`
	Notes          string              `json:"notes"`
	Items          []PurchaseOrderItem `json:"items"`
	CreatedAt      time.Time           `json:"created_at"`
}

// CalculateTotals calculates line item amounts, subtotal, VAT, and grand total
func (po *PurchaseOrder) CalculateTotals() {
	var subtotal float64
	for i := range po.Items {
		item := &po.Items[i]
		if item.Qty < 0 {
			item.Qty = 0
		}
		if item.UnitCost < 0 {
			item.UnitCost = 0
		}
		item.TotalAmount = item.Qty * item.UnitCost
		subtotal += item.TotalAmount
	}
	po.Subtotal = subtotal

	if po.Freight < 0 {
		po.Freight = 0
	}

	if po.ApplyVAT {
		po.VATAmt = (po.Subtotal + po.Freight) * 0.12
	} else {
		po.VATAmt = 0
	}

	po.GrandTotal = po.Subtotal + po.Freight + po.VATAmt
}

// Validate verifies domain invariants for the Purchase Order aggregate
func (po *PurchaseOrder) Validate() error {
	if strings.TrimSpace(po.PONo) == "" {
		return ErrEmptyPONumber
	}
	if po.Date.IsZero() {
		return ErrInvalidPODate
	}
	if strings.TrimSpace(po.VendorName) == "" {
		return ErrEmptyVendorName
	}
	if len(po.Items) == 0 {
		return ErrNoItemsInPO
	}

	hasValidItem := false
	for _, item := range po.Items {
		if strings.TrimSpace(item.Description) == "" && strings.TrimSpace(item.ItemCode) == "" {
			continue
		}
		if item.Qty <= 0 {
			return ErrInvalidItemQty
		}
		if item.UnitCost < 0 {
			return ErrInvalidItemCost
		}
		hasValidItem = true
	}

	if !hasValidItem {
		return ErrNoItemsInPO
	}

	return nil
}
