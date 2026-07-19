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
		UnitPrice: 100.0,
		Less1:     0,
		Less2:     0,
		Markup:    130,
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
				{ItemID: 2, Qty: 5.0, UOM: "BOX", UnitPrice: 500.0, Markup: 130},
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
				{ItemID: 0, Qty: 10.0, UOM: "PCS", UnitPrice: 150.0, Markup: 130},
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
				{ItemID: -5, Qty: 10.0, UOM: "PCS", UnitPrice: 150.0, Markup: 130},
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
				{ItemID: 1, Qty: 0, UOM: "PCS", UnitPrice: 150.0, Markup: 130},
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
				{ItemID: 1, Qty: -2.5, UOM: "PCS", UnitPrice: 150.0, Markup: 130},
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
				{ItemID: 1, Qty: 10.0, UOM: "", UnitPrice: 150.0, Markup: 130},
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
				{ItemID: 1, Qty: 10.0, UOM: "PCS", UnitPrice: -10.0, Markup: 130},
			},
			wantErr: true,
			errMsg:  "unit price cannot be negative",
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
		{ItemID: 1, Qty: 10.0, UOM: "PCS", UnitPrice: 100.0, Less1: 10, Less2: 5, Markup: 130},
		{ItemID: 2, Qty: 5.0, UOM: "BOX", UnitPrice: 500.0, Less1: 0, Less2: 0, Markup: 150},
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
	// UnitPrice=100, Less1=10%, Less2=5%
	// UnitCost = 100 * (1-0.10) * (1-0.05) = 100 * 0.9 * 0.95 = 85.5
	// TotalCost = 10 * 85.5 = 855
	// SellingPrice = 85.5 * 130/100 = 111.15
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
	if log1.UnitPrice != 100.0 {
		t.Errorf("expected UnitPrice 100.0, got %f", log1.UnitPrice)
	}
	if log1.Less1 != 10.0 {
		t.Errorf("expected Less1 10.0, got %f", log1.Less1)
	}
	if log1.Less2 != 5.0 {
		t.Errorf("expected Less2 5.0, got %f", log1.Less2)
	}
	expectedCost1 := 85.5
	if log1.Cost != expectedCost1 {
		t.Errorf("expected Cost %f, got %f", expectedCost1, log1.Cost)
	}
	expectedTotal1 := 855.0
	if log1.TotalCost != expectedTotal1 {
		t.Errorf("expected TotalCost %f, got %f", expectedTotal1, log1.TotalCost)
	}
	if log1.Markup != 130.0 {
		t.Errorf("expected Markup 130.0, got %f", log1.Markup)
	}
	expectedSelling1 := 85.5 * 130 / 100 // 111.15
	if log1.SellingPrice != expectedSelling1 {
		t.Errorf("expected SellingPrice %f, got %f", expectedSelling1, log1.SellingPrice)
	}

	// Verify second item log mapping
	// UnitPrice=500, Less1=0, Less2=0 → UnitCost=500
	// TotalCost = 5 * 500 = 2500
	// SellingPrice = 500 * 150/100 = 750
	log2 := logs[1]
	if log2.ItemID != 2 {
		t.Errorf("expected ItemID 2, got %d", log2.ItemID)
	}
	if log2.Qty != 5.0 {
		t.Errorf("expected Qty 5.0, got %f", log2.Qty)
	}
	if log2.Cost != 500.0 {
		t.Errorf("expected Cost 500.0, got %f", log2.Cost)
	}
	if log2.TotalCost != 2500.0 {
		t.Errorf("expected TotalCost 2500.0, got %f", log2.TotalCost)
	}
	if log2.Markup != 150.0 {
		t.Errorf("expected Markup 150.0, got %f", log2.Markup)
	}
	if log2.SellingPrice != 750.0 {
		t.Errorf("expected SellingPrice 750.0, got %f", log2.SellingPrice)
	}
}
