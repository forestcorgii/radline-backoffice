package domain_test

import (
	"testing"
	"time"

	"radline/domain"
)

func TestNewStockAdjustment_Validation(t *testing.T) {
	validItem := domain.StockAdjustmentItem{
		ItemID: 1,
		Qty:    -5.0,
		UOM:    "PCS",
		Cost:   100.0,
	}

	tests := []struct {
		name    string
		date    time.Time
		remarks string
		items   []domain.StockAdjustmentItem
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid adjustment with one item",
			date:    time.Now(),
			remarks: "Annual count correction",
			items:   []domain.StockAdjustmentItem{validItem},
			wantErr: false,
		},
		{
			name:    "valid adjustment with multiple items",
			date:    time.Now(),
			remarks: "Damaged inventory write-off",
			items: []domain.StockAdjustmentItem{
				validItem,
				{ItemID: 2, Qty: 10.0, UOM: "BOX", Cost: 400.0},
			},
			wantErr: false,
		},
		{
			name:    "zero date",
			date:    time.Time{},
			remarks: "Some reason",
			items:   []domain.StockAdjustmentItem{validItem},
			wantErr: true,
			errMsg:  "date must be valid",
		},
		{
			name:    "empty remarks",
			date:    time.Now(),
			remarks: "",
			items:   []domain.StockAdjustmentItem{validItem},
			wantErr: true,
			errMsg:  "remarks cannot be empty",
		},
		{
			name:    "no items",
			date:    time.Now(),
			remarks: "Some reason",
			items:   []domain.StockAdjustmentItem{},
			wantErr: true,
			errMsg:  "adjustment must have at least one item",
		},
		{
			name:    "invalid item ID",
			date:    time.Now(),
			remarks: "Some reason",
			items: []domain.StockAdjustmentItem{
				{ItemID: 0, Qty: -5.0, UOM: "PCS", Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "item ID must be valid",
		},
		{
			name:    "negative item ID",
			date:    time.Now(),
			remarks: "Some reason",
			items: []domain.StockAdjustmentItem{
				{ItemID: -3, Qty: -5.0, UOM: "PCS", Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "item ID must be valid",
		},
		{
			name:    "zero qty",
			date:    time.Now(),
			remarks: "Some reason",
			items: []domain.StockAdjustmentItem{
				{ItemID: 1, Qty: 0, UOM: "PCS", Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "adjustment quantity cannot be zero",
		},
		{
			name:    "empty UOM",
			date:    time.Now(),
			remarks: "Some reason",
			items: []domain.StockAdjustmentItem{
				{ItemID: 1, Qty: -5.0, UOM: "", Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "UOM cannot be empty",
		},
		{
			name:    "negative cost",
			date:    time.Now(),
			remarks: "Some reason",
			items: []domain.StockAdjustmentItem{
				{ItemID: 1, Qty: -5.0, UOM: "PCS", Cost: -10.0},
			},
			wantErr: true,
			errMsg:  "cost cannot be negative",
		},
		{
			name:    "positive qty is valid (adding stock)",
			date:    time.Now(),
			remarks: "Found missing inventory",
			items: []domain.StockAdjustmentItem{
				{ItemID: 1, Qty: 15.0, UOM: "PCS", Cost: 50.0},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewStockAdjustment(tt.date, tt.remarks, tt.items)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStockAdjustment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("NewStockAdjustment() error message = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestStockAdjustment_ToInventoryAdjustments(t *testing.T) {
	date := time.Now()
	items := []domain.StockAdjustmentItem{
		{ItemID: 1, Qty: -5.0, UOM: "PCS", Cost: 100.0},
		{ItemID: 2, Qty: 10.0, UOM: "BOX", Cost: 400.0},
	}

	sa, err := domain.NewStockAdjustment(date, "Damaged goods write-off", items)
	if err != nil {
		t.Fatalf("failed to create StockAdjustment: %v", err)
	}

	adjustments := sa.ToInventoryAdjustments()
	if len(adjustments) != 2 {
		t.Fatalf("expected 2 inventory adjustments, got %d", len(adjustments))
	}

	// Verify first item
	adj1 := adjustments[0]
	if !adj1.Date.Equal(date) {
		t.Errorf("expected Date %v, got %v", date, adj1.Date)
	}
	if adj1.ItemID != 1 {
		t.Errorf("expected ItemID 1, got %d", adj1.ItemID)
	}
	if adj1.AdjustmentQty != -5.0 {
		t.Errorf("expected AdjustmentQty -5.0, got %f", adj1.AdjustmentQty)
	}
	if adj1.UOM != "PCS" {
		t.Errorf("expected UOM PCS, got %s", adj1.UOM)
	}
	if adj1.Cost != 100.0 {
		t.Errorf("expected Cost 100.0, got %f", adj1.Cost)
	}
	if adj1.Remarks != "Damaged goods write-off" {
		t.Errorf("expected Remarks 'Damaged goods write-off', got %s", adj1.Remarks)
	}

	// Verify second item
	adj2 := adjustments[1]
	if adj2.ItemID != 2 {
		t.Errorf("expected ItemID 2, got %d", adj2.ItemID)
	}
	if adj2.AdjustmentQty != 10.0 {
		t.Errorf("expected AdjustmentQty 10.0, got %f", adj2.AdjustmentQty)
	}
	if adj2.UOM != "BOX" {
		t.Errorf("expected UOM BOX, got %s", adj2.UOM)
	}
	if adj2.Cost != 400.0 {
		t.Errorf("expected Cost 400.0, got %f", adj2.Cost)
	}
}
