package domain_test

import (
	"testing"
	"time"

	"radline/domain"
)

func TestNewStockReceive_Validation(t *testing.T) {
	validItem := domain.StockReceiveItem{
		ItemID:    1,
		Qty:       10.0,
		UOM:       "PCS",
		UnitPrice: 150.0,
		Cost:      100.0,
	}

	tests := []struct {
		name     string
		plNo     string
		supplier string
		date     time.Time
		items    []domain.StockReceiveItem
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid stock receive with one item",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items:    []domain.StockReceiveItem{validItem},
			wantErr:  false,
		},
		{
			name:     "valid stock receive with multiple items",
			plNo:     "PL101",
			supplier: "RENOWN",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				validItem,
				{ItemID: 2, Qty: 5.0, UOM: "BOX", UnitPrice: 500.0, Cost: 400.0},
			},
			wantErr: false,
		},
		{
			name:     "empty supplier",
			plNo:     "PL100",
			supplier: "",
			date:     time.Now(),
			items:    []domain.StockReceiveItem{validItem},
			wantErr:  true,
			errMsg:   "supplier cannot be empty",
		},
		{
			name:     "zero date",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Time{},
			items:    []domain.StockReceiveItem{validItem},
			wantErr:  true,
			errMsg:   "date must be valid",
		},
		{
			name:     "no items",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items:    []domain.StockReceiveItem{},
			wantErr:  true,
			errMsg:   "stocktake must have at least one item",
		},
		{
			name:     "invalid item ID",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				{ItemID: 0, Qty: 10.0, UOM: "PCS", UnitPrice: 150.0, Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "item ID must be valid",
		},
		{
			name:     "negative item ID",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				{ItemID: -5, Qty: 10.0, UOM: "PCS", UnitPrice: 150.0, Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "item ID must be valid",
		},
		{
			name:     "zero qty",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				{ItemID: 1, Qty: 0, UOM: "PCS", UnitPrice: 150.0, Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "quantity must be greater than zero",
		},
		{
			name:     "negative qty",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				{ItemID: 1, Qty: -2.5, UOM: "PCS", UnitPrice: 150.0, Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "quantity must be greater than zero",
		},
		{
			name:     "empty UOM",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				{ItemID: 1, Qty: 10.0, UOM: "", UnitPrice: 150.0, Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "UOM cannot be empty",
		},
		{
			name:     "negative unit price",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				{ItemID: 1, Qty: 10.0, UOM: "PCS", UnitPrice: -10.0, Cost: 100.0},
			},
			wantErr: true,
			errMsg:  "unit price cannot be negative",
		},
		{
			name:     "negative cost",
			plNo:     "PL100",
			supplier: "ASCD",
			date:     time.Now(),
			items: []domain.StockReceiveItem{
				{ItemID: 1, Qty: 10.0, UOM: "PCS", UnitPrice: 150.0, Cost: -5.0},
			},
			wantErr: true,
			errMsg:  "cost cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewStockReceive(tt.plNo, tt.supplier, tt.date, tt.items)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStockReceive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("NewStockReceive() error message = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestStockReceive_ToReceivingLogs(t *testing.T) {
	date := time.Now()
	items := []domain.StockReceiveItem{
		{ItemID: 1, Qty: 10.0, UOM: "PCS", UnitPrice: 150.0, Cost: 100.0},
		{ItemID: 2, Qty: 5.0, UOM: "BOX", UnitPrice: 500.0, Cost: 400.0},
	}

	sr, err := domain.NewStockReceive("PL999", "RENOWN", date, items)
	if err != nil {
		t.Fatalf("failed to create StockReceive: %v", err)
	}

	logs := sr.ToReceivingLogs()
	if len(logs) != 2 {
		t.Fatalf("expected 2 receiving logs, got %d", len(logs))
	}

	// Verify first item log mapping
	log1 := logs[0]
	if log1.Supplier != "RENOWN" {
		t.Errorf("expected Supplier RENOWN, got %s", log1.Supplier)
	}
	if !log1.Date.Equal(date) {
		t.Errorf("expected Date %v, got %v", date, log1.Date)
	}
	if log1.PLNo != "PL999" {
		t.Errorf("expected PLNo PL999, got %s", log1.PLNo)
	}
	if log1.ItemID != 1 {
		t.Errorf("expected ItemID 1, got %d", log1.ItemID)
	}
	if log1.Qty != 10.0 {
		t.Errorf("expected Qty 10.0, got %f", log1.Qty)
	}
	if log1.UOM != "PCS" {
		t.Errorf("expected UOM PCS, got %s", log1.UOM)
	}
	if log1.UnitPrice != 150.0 {
		t.Errorf("expected UnitPrice 150.0, got %f", log1.UnitPrice)
	}
	if log1.Cost != 100.0 {
		t.Errorf("expected Cost 100.0, got %f", log1.Cost)
	}
	if log1.TotalCost != 1000.0 {
		t.Errorf("expected TotalCost 1000.0, got %f", log1.TotalCost)
	}
	if log1.SellingPrice != 150.0 {
		t.Errorf("expected SellingPrice 150.0, got %f", log1.SellingPrice)
	}

	// Verify second item log mapping
	log2 := logs[1]
	if log2.ItemID != 2 {
		t.Errorf("expected ItemID 2, got %d", log2.ItemID)
	}
	if log2.Qty != 5.0 {
		t.Errorf("expected Qty 5.0, got %f", log2.Qty)
	}
	if log2.TotalCost != 2000.0 {
		t.Errorf("expected TotalCost 2000.0, got %f", log2.TotalCost)
	}
}
