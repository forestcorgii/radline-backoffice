package domain

import (
	"testing"
	"time"
)

func TestPurchaseOrder_Validation(t *testing.T) {
	now := time.Now()

	t.Run("empty PO number fails", func(t *testing.T) {
		po := PurchaseOrder{
			PONo:       "",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != ErrEmptyPONumber {
			t.Fatalf("expected ErrEmptyPONumber, got %v", err)
		}
	})

	t.Run("empty date fails", func(t *testing.T) {
		po := PurchaseOrder{
			PONo:       "PO-001",
			Date:       time.Time{},
			VendorName: "Acme Corp",
			Items: []PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != ErrInvalidPODate {
			t.Fatalf("expected ErrInvalidPODate, got %v", err)
		}
	})

	t.Run("empty vendor name fails", func(t *testing.T) {
		po := PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "   ",
			Items: []PurchaseOrderItem{
				{Description: "Test Item", Qty: 1, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != ErrEmptyVendorName {
			t.Fatalf("expected ErrEmptyVendorName, got %v", err)
		}
	})

	t.Run("no items fails", func(t *testing.T) {
		po := PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items:      []PurchaseOrderItem{},
		}
		if err := po.Validate(); err != ErrNoItemsInPO {
			t.Fatalf("expected ErrNoItemsInPO, got %v", err)
		}
	})

	t.Run("invalid item qty fails", func(t *testing.T) {
		po := PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []PurchaseOrderItem{
				{Description: "Test Item", Qty: 0, UnitCost: 100},
			},
		}
		if err := po.Validate(); err != ErrInvalidItemQty {
			t.Fatalf("expected ErrInvalidItemQty, got %v", err)
		}
	})

	t.Run("negative item cost fails", func(t *testing.T) {
		po := PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []PurchaseOrderItem{
				{Description: "Test Item", Qty: 5, UnitCost: -10},
			},
		}
		if err := po.Validate(); err != ErrInvalidItemCost {
			t.Fatalf("expected ErrInvalidItemCost, got %v", err)
		}
	})

	t.Run("valid purchase order passes", func(t *testing.T) {
		po := PurchaseOrder{
			PONo:       "PO-001",
			Date:       now,
			VendorName: "Acme Corp",
			Items: []PurchaseOrderItem{
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
		po := PurchaseOrder{
			Freight:  50,
			ApplyVAT: false,
			Items: []PurchaseOrderItem{
				{Qty: 2, UnitCost: 100}, // 200
				{Qty: 3, UnitCost: 50},  // 150
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

	t.Run("subtotal and grand total with 12% vat", func(t *testing.T) {
		po := PurchaseOrder{
			Freight:  100,
			ApplyVAT: true,
			Items: []PurchaseOrderItem{
				{Qty: 10, UnitCost: 100}, // 1000
			},
		}
		po.CalculateTotals()

		if po.Subtotal != 1000 {
			t.Errorf("expected Subtotal 1000, got %f", po.Subtotal)
		}
		// (1000 + 100) * 0.12 = 132
		if po.VATAmt != 132 {
			t.Errorf("expected VATAmt 132, got %f", po.VATAmt)
		}
		// 1000 + 100 + 132 = 1232
		if po.GrandTotal != 1232 {
			t.Errorf("expected GrandTotal 1232, got %f", po.GrandTotal)
		}
	})
}
