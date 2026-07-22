package domain_test

import (
	"math"
	"testing"
	"time"

	"radline/domain"
)

func TestNewSale_Validation(t *testing.T) {
	validItem := domain.SaleItem{
		ItemID: 1,
		Qty:    5.0,
		UOM:    "PCS",
		Price:  100.0,
		Cost:   80.0,
	}

	tests := []struct {
		name      string
		docType   string
		docNo     string
		supplier  string
		docDate   time.Time
		items     []domain.SaleItem
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "valid sale with one item",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{validItem},
			wantErr:  false,
		},
		{
			name:     "valid sale with multiple items",
			docType:  "DR",
			docNo:    "INV101",
			supplier: "RENOWN",
			docDate:  time.Now(),
			items: []domain.SaleItem{
				validItem,
				{ItemID: 2, Qty: 10.0, UOM: "BOX", Price: 500.0, Cost: 400.0},
			},
			wantErr: false,
		},
		{
			name:     "empty doc type",
			docType:  "",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{validItem},
			wantErr:  true,
			errMsg:   "doc type cannot be empty",
		},
		{
			name:     "empty doc number",
			docType:  "SI",
			docNo:    "",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{validItem},
			wantErr:  true,
			errMsg:   "doc number cannot be empty",
		},
		{
			name:     "empty supplier",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "",
			docDate:  time.Now(),
			items:    []domain.SaleItem{validItem},
			wantErr:  true,
			errMsg:   "supplier cannot be empty",
		},
		{
			name:     "zero doc date",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Time{},
			items:    []domain.SaleItem{validItem},
			wantErr:  true,
			errMsg:   "doc date must be valid",
		},
		{
			name:     "no items",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{},
			wantErr:  true,
			errMsg:   "sale must have at least one item",
		},
		{
			name:     "invalid item ID",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{{ItemID: 0, Qty: 5.0, UOM: "PCS", Price: 100.0, Cost: 80.0}},
			wantErr:  true,
			errMsg:   "item ID must be valid",
		},
		{
			name:     "zero qty",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{{ItemID: 1, Qty: 0.0, UOM: "PCS", Price: 100.0, Cost: 80.0}},
			wantErr:  true,
			errMsg:   "quantity must be greater than zero",
		},
		{
			name:     "negative qty",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{{ItemID: 1, Qty: -1.5, UOM: "PCS", Price: 100.0, Cost: 80.0}},
			wantErr:  true,
			errMsg:   "quantity must be greater than zero",
		},
		{
			name:     "empty item UOM",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{{ItemID: 1, Qty: 5.0, UOM: "", Price: 100.0, Cost: 80.0}},
			wantErr:  true,
			errMsg:   "UOM cannot be empty",
		},
		{
			name:     "negative price",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{{ItemID: 1, Qty: 5.0, UOM: "PCS", Price: -10.0, Cost: 80.0}},
			wantErr:  true,
			errMsg:   "price cannot be negative",
		},
		{
			name:     "negative cost",
			docType:  "SI",
			docNo:    "INV100",
			supplier: "ASCD",
			docDate:  time.Now(),
			items:    []domain.SaleItem{{ItemID: 1, Qty: 5.0, UOM: "PCS", Price: 100.0, Cost: -5.0}},
			wantErr:  true,
			errMsg:   "cost cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewSale(tt.docType, "POSTED", tt.docDate, tt.docNo, "Customer", tt.supplier, tt.items)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSale() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("NewSale() error message = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestSale_ToSalesDetails(t *testing.T) {
	items := []domain.SaleItem{
		{ItemID: 1, Qty: 2.0, UOM: "PCS", Price: 150.0, Cost: 100.0, Patong: 10.0, POSCharge: 5.0, WT2307: 2.0, Remarks: "Test item 1"},
		{ItemID: 2, Qty: 3.0, UOM: "BOX", Price: 50.0, Cost: 30.0, Patong: 0.0, POSCharge: 0.0, WT2307: 0.0},
	}
	docDate := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)

	sale, err := domain.NewSale("SI", "POSTED", docDate, "INV-XYZ", "John Doe", "SupplierA", items)
	if err != nil {
		t.Fatalf("failed to create valid sale: %v", err)
	}

	details := sale.ToSalesDetails()
	if len(details) != 2 {
		t.Fatalf("expected 2 sales details, got %d", len(details))
	}

	// Verify calculations and mappings for first item
	d1 := details[0]
	if d1.DocType != "SI" || d1.DocNumber != "INV-XYZ" || d1.Supplier != "SupplierA" || d1.CustomerName != "John Doe" {
		t.Errorf("incorrect header mappings on first detail: %+v", d1)
	}
	if d1.ItemID != 1 || d1.Qty != 2.0 || d1.UOM != "PCS" || d1.Price != 150.0 || d1.Cost != 100.0 {
		t.Errorf("incorrect item mappings on first detail: %+v", d1)
	}
	if d1.TotalSales != 300.0 || d1.TotalCost != 200.0 || d1.Profit != 100.0 {
		t.Errorf("incorrect calculated values on first detail: %+v", d1)
	}
	expectedRemit1 := 300.0 - (10.0 + 5.0 + 2.0) // 283.0
	if d1.TotalRemit != expectedRemit1 {
		t.Errorf("expected TotalRemit %f, got %f", expectedRemit1, d1.TotalRemit)
	}
	expectedMargin1 := (100.0 / 300.0) * 100.0 // 33.333333333333336
	if math.Abs(d1.ProfitMargin-expectedMargin1) > 0.0001 {
		t.Errorf("expected ProfitMargin %f, got %f", expectedMargin1, d1.ProfitMargin)
	}

	// Verify calculations and mappings for second item
	d2 := details[1]
	if d2.DocType != "SI" || d2.DocNumber != "INV-XYZ" || d2.Supplier != "SupplierA" || d2.CustomerName != "John Doe" {
		t.Errorf("incorrect header mappings on second detail: %+v", d2)
	}
	if d2.ItemID != 2 || d2.Qty != 3.0 || d2.UOM != "BOX" || d2.Price != 50.0 || d2.Cost != 30.0 {
		t.Errorf("incorrect item mappings on second detail: %+v", d2)
	}
	if d2.TotalSales != 150.0 || d2.TotalCost != 90.0 || d2.Profit != 60.0 {
		t.Errorf("incorrect calculated values on second detail: %+v", d2)
	}
	if d2.TotalRemit != 150.0 {
		t.Errorf("expected TotalRemit 150.0, got %f", d2.TotalRemit)
	}
	expectedMargin2 := (60.0 / 150.0) * 100.0 // 40%
	if d2.ProfitMargin != expectedMargin2 {
		t.Errorf("expected ProfitMargin %f, got %f", expectedMargin2, d2.ProfitMargin)
	}
}

func TestNewSale_StatusValidation(t *testing.T) {
	validItem := domain.SaleItem{
		ItemID: 1,
		Qty:    5.0,
		UOM:    "PCS",
		Price:  100.0,
		Cost:   80.0,
	}

	tests := []struct {
		name      string
		docStatus string
		wantErr   bool
	}{
		{"valid active", "Active", false},
		{"valid posted", "Posted", false},
		{"valid cancelled/return", "Cancelled/Return", false},
		{"valid uppercase active", "ACTIVE", false},
		{"valid uppercase posted", "POSTED", false},
		{"valid uppercase cancelled/return", "CANCELLED/RETURN", false},
		{"invalid status value", "InvalidStatus", true},
		{"empty status", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewSale("SI", tt.docStatus, time.Now(), "INV100", "Customer", "ASCD", []domain.SaleItem{validItem})
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSale() with status %q: error = %v, wantErr %v", tt.docStatus, err, tt.wantErr)
			}
		})
	}
}
