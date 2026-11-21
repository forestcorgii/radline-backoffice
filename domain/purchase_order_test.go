package domain_test

import (
	"testing"
	"time"

	"rav/domain"
)

func TestPurchaseOrder_Validation(t *testing.T) {
	now := time.Now()

	t.Run("empty PO number fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []domain.PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != domain.ErrEmptyPONumber {
			t.Fatalf("expected ErrEmptyPONumber, got %v", err)
		}
	})

	t.Run("empty date fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       time.Time{},
			VendorName: "Acme Corp",
			Items: []domain.PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != domain.ErrInvalidPODate {
			t.Fatalf("expected ErrInvalidPODate, got %v", err)
		}
	})

	t.Run("due date before PO date fails", func(t *testing.T) {
		before := now.Add(-24 * time.Hour)
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			DueDate:    &before,
			VendorName: "Acme Corp",
			Items: []domain.PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != domain.ErrInvalidPODueDate {
			t.Fatalf("expected ErrInvalidPODueDate, got %v", err)
		}
	})

	t.Run("negative freight fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Freight:    -20,
			Items: []domain.PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != domain.ErrInvalidPOFreight {
			t.Fatalf("expected ErrInvalidPOFreight, got %v", err)
		}
	})

	t.Run("empty vendor name fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "   ",
			Items: []domain.PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != domain.ErrEmptyVendorName {
			t.Fatalf("expected ErrEmptyVendorName, got %v", err)
		}
	})

	t.Run("no items fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items:      []domain.PurchaseOrderItem{},
		}
		if err := po.Validate(); err != domain.ErrNoItemsInPO {
			t.Fatalf("expected ErrNoItemsInPO, got %v", err)
		}
	})

	t.Run("blank item row with non-zero qty or cost fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []domain.PurchaseOrderItem{
				{Description: "Valid Item", Qty: 1, UnitCost: 100},
				{Description: "  ", ItemCode: "", Qty: 2, UnitCost: 50},
			},
		}
		if err := po.Validate(); err != domain.ErrBlankPOItemRow {
			t.Fatalf("expected ErrBlankPOItemRow, got %v", err)
		}
	})

	t.Run("invalid item qty fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []domain.PurchaseOrderItem{
				{Description: "Test Item", Qty: 0, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != domain.ErrInvalidItemQty {
			t.Fatalf("expected ErrInvalidItemQty, got %v", err)
		}
	})

	t.Run("negative item cost fails", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []domain.PurchaseOrderItem{
				{Description: "Test Item", Qty: 5, UnitCost: -10},
			},
		}
		if err := po.Validate(); err != domain.ErrInvalidItemCost {
			t.Fatalf("expected ErrInvalidItemCost, got %v", err)
		}
	})

	t.Run("valid purchase order passes", func(t *testing.T) {
		po := domain.PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []domain.PurchaseOrderItem{
				{ItemCode: "ITEM-1", Description: "Test Item", Qty: 5, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})
}

func TestPurchaseOrder_CalculateTotals(t *testing.T) {
	t.Run("subtotal and grand total without vat", func(t *testing.T) {
		po := domain.PurchaseOrder{
			Freight:  50,
			ApplyVAT: false,
			Items: []domain.PurchaseOrderItem{
				{Description: "Item 1", Qty: 2, UnitCost: 100}, // 200
				{Description: "Item 2", Qty: 3, UnitCost: 50},  // 150
				{Description: "", ItemCode: "", Qty: 0, UnitCost: 0},
			},
		}
		po.CalculateTotals()

		if po.Subtotal != 350 {
			t.Errorf("expected Subtotal 350, got %f", po.Subtotal)
		}
		if po.VATAmt != 0 {
			t.Errorf("expected VATAmt 0, got %f", po.VATAmt)
		}
		if po.GrandTotal != 400 {
			t.Errorf("expected GrandTotal 400, got %f", po.GrandTotal)
		}
	})

	t.Run("subtotal and grand total with 12% vat and rounding", func(t *testing.T) {
		po := domain.PurchaseOrder{
			Freight:  100,
			ApplyVAT: true,
			Items: []domain.PurchaseOrderItem{
				{Description: "Item 1", Qty: 10, UnitCost: 100}, // 1000
			},
		}
		po.CalculateTotals()

		if po.Subtotal != 1000 {
			t.Errorf("expected Subtotal 1000, got %f", po.Subtotal)
		}
		if po.VATAmt != 132 {
			t.Errorf("expected VATAmt 132, got %f", po.VATAmt)
		}
		if po.GrandTotal != 1232 {
			t.Errorf("expected GrandTotal 1232, got %f", po.GrandTotal)
		}
	})
}
