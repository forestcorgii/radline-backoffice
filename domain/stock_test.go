package domain_test

import (
	"testing"
	"time"

	"radline/domain"
)

func TestNewBrand_Validation(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		brandNm string
		wantErr bool
	}{
		{"valid brand", "BRD1", "Brand One", false},
		{"empty code", "", "Brand One", true},
		{"empty name", "BRD1", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewBrand(1, tt.code, tt.brandNm)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBrand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewCategory_Validation(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		catNm   string
		wantErr bool
	}{
		{"valid category", "CAT1", "Category One", false},
		{"empty code", "", "Category One", true},
		{"empty name", "CAT1", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewCategory(1, tt.code, tt.catNm)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCategory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewItem_Validation(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		description string
		defaultUOM  string
		wantErr     bool
	}{
		{"valid item", "ITM1", "Item One", "PCS", false},
		{"empty code", "", "Item One", "PCS", true},
		{"empty description", "ITM1", "", "PCS", true},
		{"empty default UOM", "ITM1", "Item One", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewItem(1, tt.code, tt.description, tt.defaultUOM, "Model 1", 1, 1, "Var", "Remarks")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewUomSetting_Validation(t *testing.T) {
	tests := []struct {
		name    string
		muom    string
		factor  float64
		wantErr bool
	}{
		{"valid uom", "BOX", 10.0, false},
		{"empty uom", "", 10.0, true},
		{"zero factor", "BOX", 0.0, true},
		{"negative factor", "BOX", -5.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewUomSetting(tt.muom, tt.factor)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUomSetting() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewReceivingLog_Validation(t *testing.T) {
	tests := []struct {
		name      string
		supplier  string
		qty       float64
		uom       string
		unitPrice float64
		cost      float64
		wantErr   bool
	}{
		{"valid log", "ASCD", 10.0, "PCS", 100.0, 80.0, false},
		{"empty supplier", "", 10.0, "PCS", 100.0, 80.0, true},
		{"zero qty", "ASCD", 0.0, "PCS", 100.0, 80.0, true},
		{"negative qty", "ASCD", -1.0, "PCS", 100.0, 80.0, true},
		{"empty uom", "ASCD", 10.0, "", 100.0, 80.0, true},
		{"negative unit price", "ASCD", 10.0, "PCS", -5.0, 80.0, true},
		{"negative cost", "ASCD", 10.0, "PCS", 100.0, -1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := domain.NewReceivingLog(1, tt.supplier, time.Now(), "PL1", 1, tt.qty, tt.uom, tt.unitPrice, tt.cost)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReceivingLog() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && rl.TotalCost != tt.qty*tt.cost {
				t.Errorf("TotalCost calculation incorrect: got %f, want %f", rl.TotalCost, tt.qty*tt.cost)
			}
		})
	}
}

func TestNewSalesDetail_Validation(t *testing.T) {
	tests := []struct {
		name      string
		docType   string
		docNo     string
		supplier  string
		qty       float64
		uom       string
		price     float64
		cost      float64
		wantErr   bool
	}{
		{"valid sale", "SI", "INV100", "ASCD", 5.0, "PCS", 150.0, 100.0, false},
		{"empty doctype", "", "INV100", "ASCD", 5.0, "PCS", 150.0, 100.0, true},
		{"empty doc number", "SI", "", "ASCD", 5.0, "PCS", 150.0, 100.0, true},
		{"empty supplier", "SI", "INV100", "", 5.0, "PCS", 150.0, 100.0, true},
		{"zero qty", "SI", "INV100", "ASCD", 0.0, "PCS", 150.0, 100.0, true},
		{"negative qty", "SI", "INV100", "ASCD", -1.0, "PCS", 150.0, 100.0, true},
		{"empty uom", "SI", "INV100", "ASCD", 5.0, "", 150.0, 100.0, true},
		{"negative price", "SI", "INV100", "ASCD", 5.0, "PCS", -1.0, 100.0, true},
		{"negative cost", "SI", "INV100", "ASCD", 5.0, "PCS", 150.0, -1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd, err := domain.NewSalesDetail(1, tt.docType, "POSTED", time.Now(), tt.docNo, "Customer", tt.supplier, 1, tt.qty, tt.uom, tt.price, tt.cost, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSalesDetail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				expectedSales := tt.qty * tt.price
				expectedCost := tt.qty * tt.cost
				expectedProfit := expectedSales - expectedCost
				if sd.TotalSales != expectedSales {
					t.Errorf("TotalSales calculation incorrect: got %f, want %f", sd.TotalSales, expectedSales)
				}
				if sd.TotalCost != expectedCost {
					t.Errorf("TotalCost calculation incorrect: got %f, want %f", sd.TotalCost, expectedCost)
				}
				if sd.Profit != expectedProfit {
					t.Errorf("Profit calculation incorrect: got %f, want %f", sd.Profit, expectedProfit)
				}
			}
		})
	}
}

func TestNewInventoryAdjustment_Validation(t *testing.T) {
	tests := []struct {
		name    string
		qty     float64
		cost    float64
		remarks string
		wantErr bool
	}{
		{"valid adjustment positive", 5.0, 100.0, "Found items", false},
		{"valid adjustment negative", -2.0, 100.0, "Damaged items", false},
		{"zero adjustment", 0.0, 100.0, "No change", true},
		{"negative cost", 5.0, -10.0, "Negative cost", true},
		{"empty remarks", 5.0, 100.0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewInventoryAdjustment(1, 0, time.Now(), 1, "PCS", tt.qty, tt.cost, tt.remarks)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewInventoryAdjustment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestItemStock_CalculateOnHand(t *testing.T) {
	// Setup Item
	item, err := domain.NewItem(1, "ITM1", "Test Item", "PCS", "M1", 1, 1, "", "")
	if err != nil {
		t.Fatalf("failed to setup item: %v", err)
	}

	// Setup UOM conversion settings: 1 BOX = 10 PCS
	boxUOM, err := domain.NewUomSetting("BOX", 10.0)
	if err != nil {
		t.Fatalf("failed to setup UOM: %v", err)
	}
	uomSettings := []domain.UomSetting{boxUOM}

	// Setup Receiving Logs
	// 10 PCS in Supplier A
	rl1, _ := domain.NewReceivingLog(1, "SupplierA", time.Now(), "PL1", 1, 10.0, "PCS", 100.0, 80.0)
	// 2 BOX in Supplier A (equals 20 PCS)
	rl2, _ := domain.NewReceivingLog(2, "SupplierA", time.Now(), "PL2", 1, 2.0, "BOX", 1000.0, 800.0)
	// 5 PCS in Supplier B
	rl3, _ := domain.NewReceivingLog(3, "SupplierB", time.Now(), "PL3", 1, 5.0, "PCS", 100.0, 80.0)

	receivingLogs := []domain.ReceivingLog{rl1, rl2, rl3}

	// Setup Sales Details
	// 3 PCS sold in Supplier A
	s1, _ := domain.NewSalesDetail(1, "SI", "POSTED", time.Now(), "INV1", "Cust", "SupplierA", 1, 3.0, "PCS", 120.0, 80.0, "")
	// 1 BOX sold in Supplier A (equals 10 PCS)
	s2, _ := domain.NewSalesDetail(2, "SI", "POSTED", time.Now(), "INV2", "Cust", "SupplierA", 1, 1.0, "BOX", 1200.0, 800.0, "")
	// 2 PCS sold in Supplier B
	s3, _ := domain.NewSalesDetail(3, "SI", "POSTED", time.Now(), "INV3", "Cust", "SupplierB", 1, 2.0, "PCS", 120.0, 80.0, "")

	salesDetails := []domain.SalesDetail{s1, s2, s3}

	// Setup Adjustments (not supplier-specific)
	// 2 PCS adjusted positive, 1 BOX adjusted negative (equals -10 PCS)
	adj1, _ := domain.NewInventoryAdjustment(1, 0, time.Now(), 1, "PCS", 2.0, 80.0, "Found")
	adj2, _ := domain.NewInventoryAdjustment(2, 0, time.Now(), 1, "BOX", -1.0, 800.0, "Stolen")

	adjustments := []domain.InventoryAdjustment{adj1, adj2}

	stock := domain.NewItemStock(item, uomSettings, receivingLogs, salesDetails, adjustments)

	// Calculate stock on hand for SupplierA
	// Received: 10 PCS + 2 BOX (20 PCS) = 30 PCS
	// Sold: 3 PCS + 1 BOX (10 PCS) = 13 PCS
	// Expected: 30 - 13 = 17 PCS
	onHandA := stock.CalculateOnHand("SupplierA")
	if onHandA != 17.0 {
		t.Errorf("expected SupplierA stock on hand to be 17.0, got %f", onHandA)
	}

	// Calculate stock on hand for SupplierB
	// Received: 5 PCS
	// Sold: 2 PCS
	// Expected: 3 PCS
	onHandB := stock.CalculateOnHand("SupplierB")
	if onHandB != 3.0 {
		t.Errorf("expected SupplierB stock on hand to be 3.0, got %f", onHandB)
	}

	// Calculate global stock on hand (includes adjustments)
	// Total Received: 10 + 20 + 5 = 35 PCS
	// Total Sold: 3 + 10 + 2 = 15 PCS
	// Total Adjusted: +2 - 10 = -8 PCS
	// Expected: 35 - 15 - 8 = 12 PCS
	globalOnHand := stock.CalculateGlobalOnHand()
	if globalOnHand != 12.0 {
		t.Errorf("expected Global stock on hand to be 12.0, got %f", globalOnHand)
	}
}

func TestItemStock_GetOldestPLWithStock(t *testing.T) {
	item, err := domain.NewItem(1, "ITM1", "Test Item", "PCS", "M1", 1, 1, "", "")
	if err != nil {
		t.Fatalf("failed to setup item: %v", err)
	}

	// 1. Empty logs
	stockEmpty := domain.NewItemStock(item, nil, nil, nil, nil)
	if _, _, _, found := stockEmpty.GetOldestPLWithStock(); found {
		t.Errorf("expected false for empty receiving logs")
	}

	// Setup receiving logs:
	// PL1 on Day 1: 10 PCS @ cost 100, price 150
	// PL2 on Day 2: 15 PCS @ cost 110, price 160
	// PL3 on Day 3: 20 PCS @ cost 120, price 170
	t1 := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	t3 := t2.Add(24 * time.Hour)

	rl1, _ := domain.NewReceivingLog(1, "SupplierA", t1, "PL1", 1, 10.0, "PCS", 150.0, 100.0)
	rl2, _ := domain.NewReceivingLog(2, "SupplierA", t2, "PL2", 1, 15.0, "PCS", 160.0, 110.0)
	rl3, _ := domain.NewReceivingLog(3, "SupplierB", t3, "PL3", 1, 20.0, "PCS", 170.0, 120.0)
	receivingLogs := []domain.ReceivingLog{rl1, rl2, rl3}

	// Case A: No sales, no adjustments (OnHand = 45)
	// Oldest PL with stock should be PL1 (cost 100, price 150, plNo PL1)
	stockA := domain.NewItemStock(item, nil, receivingLogs, nil, nil)
	costA, priceA, plNoA, foundA := stockA.GetOldestPLWithStock()
	if !foundA || costA != 100.0 || priceA != 150.0 || plNoA != "PL1" {
		t.Errorf("Case A failed: got cost %f, price %f, plNo %q, found %t; expected 100.0, 150.0, PL1, true", costA, priceA, plNoA, foundA)
	}

	// Case B: Sales of 8 PCS (OnHand = 37)
	// Oldest PL with stock should still be PL1 (PL1 has 2 PCS remaining)
	sB, _ := domain.NewSalesDetail(1, "SI", "POSTED", t2, "INV1", "Cust", "SupplierA", 1, 8.0, "PCS", 150.0, 100.0, "")
	stockB := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sB}, nil)
	costB, priceB, plNoB, foundB := stockB.GetOldestPLWithStock()
	if !foundB || costB != 100.0 || priceB != 150.0 || plNoB != "PL1" {
		t.Errorf("Case B failed: got cost %f, price %f, plNo %q, found %t; expected 100.0, 150.0, PL1, true", costB, priceB, plNoB, foundB)
	}

	// Case C: Sales of 12 PCS (OnHand = 33)
	// PL1 (10 PCS) is fully consumed. PL2 has 13 PCS remaining.
	// Oldest PL with stock should be PL2 (cost 110, price 160, plNo PL2)
	sC, _ := domain.NewSalesDetail(1, "SI", "POSTED", t2, "INV1", "Cust", "SupplierA", 1, 12.0, "PCS", 150.0, 100.0, "")
	stockC := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sC}, nil)
	costC, priceC, plNoC, foundC := stockC.GetOldestPLWithStock()
	if !foundC || costC != 110.0 || priceC != 160.0 || plNoC != "PL2" {
		t.Errorf("Case C failed: got cost %f, price %f, plNo %q, found %t; expected 110.0, 160.0, PL2, true", costC, priceC, plNoC, foundC)
	}

	// Case D: Sales of 28 PCS (OnHand = 17)
	// PL1 (10 PCS) and PL2 (15 PCS) are fully consumed. PL3 has 17 PCS remaining.
	// Oldest PL with stock should be PL3 (cost 120, price 170, plNo PL3)
	sD, _ := domain.NewSalesDetail(1, "SI", "POSTED", t3, "INV1", "Cust", "SupplierA", 1, 28.0, "PCS", 150.0, 100.0, "")
	stockD := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sD}, nil)
	costD, priceD, plNoD, foundD := stockD.GetOldestPLWithStock()
	if !foundD || costD != 120.0 || priceD != 170.0 || plNoD != "PL3" {
		t.Errorf("Case D failed: got cost %f, price %f, plNo %q, found %t; expected 120.0, 170.0, PL3, true", costD, priceD, plNoD, foundD)
	}

	// Case E: Sales of 45 PCS (OnHand = 0)
	// All consumed. Returns found = false.
	sE, _ := domain.NewSalesDetail(1, "SI", "POSTED", t3, "INV1", "Cust", "SupplierA", 1, 45.0, "PCS", 150.0, 100.0, "")
	stockE := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sE}, nil)
	if _, _, _, foundE := stockE.GetOldestPLWithStock(); foundE {
		t.Errorf("expected false when onHand is 0")
	}
}
