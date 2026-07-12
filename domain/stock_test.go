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
			sd, err := domain.NewSalesDetail(1, tt.docType, "POSTED", time.Now(), tt.docNo, "Customer", tt.supplier, 1, tt.qty, tt.uom, tt.price, tt.cost)
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
	s1, _ := domain.NewSalesDetail(1, "SI", "POSTED", time.Now(), "INV1", "Cust", "SupplierA", 1, 3.0, "PCS", 120.0, 80.0)
	// 1 BOX sold in Supplier A (equals 10 PCS)
	s2, _ := domain.NewSalesDetail(2, "SI", "POSTED", time.Now(), "INV2", "Cust", "SupplierA", 1, 1.0, "BOX", 1200.0, 800.0)
	// 2 PCS sold in Supplier B
	s3, _ := domain.NewSalesDetail(3, "SI", "POSTED", time.Now(), "INV3", "Cust", "SupplierB", 1, 2.0, "PCS", 120.0, 80.0)

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
