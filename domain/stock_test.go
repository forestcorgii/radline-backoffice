package domain_test

import (
	"math"
	"testing"
	"time"

	"rav/domain"
)

func TestRound2AndIsFinite(t *testing.T) {
	if domain.Round2(12.345) != 12.35 {
		t.Errorf("expected 12.35, got %f", domain.Round2(12.345))
	}
	if !domain.IsFinite(42.5) {
		t.Errorf("expected 42.5 to be finite")
	}
	if domain.IsFinite(math.NaN()) {
		t.Errorf("expected NaN not to be finite")
	}
	if domain.IsFinite(math.Inf(1)) || domain.IsFinite(math.Inf(-1)) {
		t.Errorf("expected ±Inf not to be finite")
	}
}

func TestNewBrand_Validation(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		brandNm string
		wantErr bool
	}{
		{"valid brand", "BRD1", "Brand One", false},
		{"empty code", "", "Brand One", true},
		{"whitespace code", "   ", "Brand One", true},
		{"empty name", "BRD1", "", true},
		{"whitespace name", "BRD1", " \t ", true},
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
		{"whitespace code", "   ", "Category One", true},
		{"empty name", "CAT1", "", true},
		{"whitespace name", "CAT1", "  ", true},
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
		{"whitespace uom", "   ", 10.0, true},
		{"zero factor", "BOX", 0.0, true},
		{"negative factor", "BOX", -5.0, true},
		{"NaN factor", "BOX", math.NaN(), true},
		{"Inf factor", "BOX", math.Inf(1), true},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := domain.NewReceivingLog(1, tt.supplier, time.Now(), "PL1", 1, tt.qty, tt.uom, tt.unitPrice, 0, 0, tt.cost, 130, "")
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
		name     string
		docType  string
		docNo    string
		supplier string
		qty      float64
		uom      string
		price    float64
		cost     float64
		wantErr  bool
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
			sd, err := domain.NewSalesDetail(1, tt.docType, "POSTED", time.Now(), tt.docNo, "Customer", tt.supplier, 1, tt.qty, tt.uom, tt.price, tt.cost, 0, 0, 0, "", "")
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
		date    time.Time
		qty     float64
		cost    float64
		remarks string
		wantErr bool
	}{
		{"valid adjustment positive", time.Now(), 5.0, 100.0, "Found items", false},
		{"valid adjustment negative", time.Now(), -2.0, 100.0, "Damaged items", false},
		{"zero date", time.Time{}, 5.0, 100.0, "Found items", true},
		{"zero adjustment", time.Now(), 0.0, 100.0, "No change", true},
		{"negative cost", time.Now(), 5.0, -10.0, "Negative cost", true},
		{"empty remarks", time.Now(), 5.0, 100.0, "", true},
		{"whitespace remarks", time.Now(), 5.0, 100.0, "   ", true},
		{"NaN qty", time.Now(), math.NaN(), 100.0, "NaN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewInventoryAdjustment(1, 0, tt.date, 1, "PCS", tt.qty, tt.cost, tt.remarks)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewInventoryAdjustment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestItemStock_CalculateOnHand(t *testing.T) {
	item, err := domain.NewItem(1, "ITM1", "Test Item", "PCS", "M1", 1, 1, "", "")
	if err != nil {
		t.Fatalf("failed to setup item: %v", err)
	}

	boxUOM, err := domain.NewUomSetting("BOX", 10.0)
	if err != nil {
		t.Fatalf("failed to setup UOM: %v", err)
	}
	uomSettings := []domain.UomSetting{boxUOM}

	rl1, _ := domain.NewReceivingLog(1, "SupplierA", time.Now(), "PL1", 1, 10.0, "pcs", 100.0, 0, 0, 80.0, 130, "")
	rl2, _ := domain.NewReceivingLog(2, "SupplierA", time.Now(), "PL2", 1, 2.0, "box", 1000.0, 0, 0, 800.0, 130, "")
	rl3, _ := domain.NewReceivingLog(3, "SupplierB", time.Now(), "PL3", 1, 5.0, "PCS", 100.0, 0, 0, 80.0, 130, "")
	// Unrelated item ID 2 should be ignored
	rlOtherItem, _ := domain.NewReceivingLog(4, "SupplierA", time.Now(), "PL4", 2, 100.0, "PCS", 100.0, 0, 0, 80.0, 130, "")

	receivingLogs := []domain.ReceivingLog{rl1, rl2, rl3, rlOtherItem}

	s1, _ := domain.NewSalesDetail(1, "SI", "POSTED", time.Now(), "INV1", "Cust", "SupplierA", 1, 3.0, "PCS", 120.0, 80.0, 0, 0, 0, "", "")
	s2, _ := domain.NewSalesDetail(2, "SI", "POSTED", time.Now(), "INV2", "Cust", "SupplierA", 1, 1.0, "box", 1200.0, 800.0, 0, 0, 0, "", "")
	s3, _ := domain.NewSalesDetail(3, "SI", "POSTED", time.Now(), "INV3", "Cust", "SupplierB", 1, 2.0, "PCS", 120.0, 80.0, 0, 0, 0, "", "")
	sOtherItem, _ := domain.NewSalesDetail(4, "SI", "POSTED", time.Now(), "INV4", "Cust", "SupplierA", 2, 50.0, "PCS", 120.0, 80.0, 0, 0, 0, "", "")

	salesDetails := []domain.SalesDetail{s1, s2, s3, sOtherItem}

	adj1, _ := domain.NewInventoryAdjustment(1, 0, time.Now(), 1, "PCS", 2.0, 80.0, "Found")
	adj2, _ := domain.NewInventoryAdjustment(2, 0, time.Now(), 1, "BOX", -1.0, 800.0, "Stolen")
	adjOtherItem, _ := domain.NewInventoryAdjustment(3, 0, time.Now(), 2, "PCS", 50.0, 80.0, "Other")

	adjustments := []domain.InventoryAdjustment{adj1, adj2, adjOtherItem}

	stock := domain.NewItemStock(item, uomSettings, receivingLogs, salesDetails, adjustments)

	onHandA := stock.CalculateOnHand("SupplierA")
	if onHandA != 17.0 {
		t.Errorf("expected SupplierA stock on hand to be 17.0, got %f", onHandA)
	}

	onHandB := stock.CalculateOnHand("SupplierB")
	if onHandB != 3.0 {
		t.Errorf("expected SupplierB stock on hand to be 3.0, got %f", onHandB)
	}

	globalOnHand := stock.CalculateGlobalOnHand()
	if globalOnHand != 12.0 {
		t.Errorf("expected Global stock on hand to be 12.0, got %f", globalOnHand)
	}

	// Strict UOM conversion factor checks
	factorBox, err := stock.GetConversionFactorStrict("box")
	if err != nil || factorBox != 10.0 {
		t.Errorf("expected strict conversion factor 10.0 for 'box', got %f, err=%v", factorBox, err)
	}
	factorPcs, err := stock.GetConversionFactorStrict("pcs")
	if err != nil || factorPcs != 1.0 {
		t.Errorf("expected strict conversion factor 1.0 for 'pcs', got %f, err=%v", factorPcs, err)
	}
	if _, err := stock.GetConversionFactorStrict(""); err == nil {
		t.Errorf("expected error for empty UOM in GetConversionFactorStrict")
	}
	if _, err := stock.GetConversionFactorStrict("GALLON"); err == nil {
		t.Errorf("expected error for unknown UOM in GetConversionFactorStrict")
	}
}

func TestItemStock_GetOldestPLWithStock(t *testing.T) {
	item, err := domain.NewItem(1, "ITM1", "Test Item", "PCS", "M1", 1, 1, "", "")
	if err != nil {
		t.Fatalf("failed to setup item: %v", err)
	}

	stockEmpty := domain.NewItemStock(item, nil, nil, nil, nil)
	if _, _, _, found := stockEmpty.GetOldestPLWithStock(); found {
		t.Errorf("expected false for empty receiving logs")
	}

	t1 := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	t3 := t2.Add(24 * time.Hour)

	rl1, _ := domain.NewReceivingLog(1, "SupplierA", t1, "PL1", 1, 10.0, "PCS", 150.0, 0, 0, 100.0, 130, "")
	rl2, _ := domain.NewReceivingLog(2, "SupplierA", t2, "PL2", 1, 15.0, "PCS", 160.0, 0, 0, 110.0, 130, "")
	rl3, _ := domain.NewReceivingLog(3, "SupplierB", t3, "PL3", 1, 20.0, "PCS", 170.0, 0, 0, 120.0, 130, "")
	receivingLogs := []domain.ReceivingLog{rl1, rl2, rl3}

	// Case A: No sales, no adjustments (OnHand = 45)
	stockA := domain.NewItemStock(item, nil, receivingLogs, nil, nil)
	costA, priceA, plNoA, foundA := stockA.GetOldestPLWithStock()
	if !foundA || costA != 100.0 || priceA != 130.0 || plNoA != "PL1" {
		t.Errorf("Case A failed: got cost %f, price %f, plNo %q, found %t; expected 100.0, 130.0, PL1, true", costA, priceA, plNoA, foundA)
	}

	// Case B: Sales of 8 PCS (OnHand = 37)
	sB, _ := domain.NewSalesDetail(1, "SI", "POSTED", t2, "INV1", "Cust", "SupplierA", 1, 8.0, "PCS", 150.0, 100.0, 0, 0, 0, "", "")
	stockB := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sB}, nil)
	costB, priceB, plNoB, foundB := stockB.GetOldestPLWithStock()
	if !foundB || costB != 100.0 || priceB != 130.0 || plNoB != "PL1" {
		t.Errorf("Case B failed: got cost %f, price %f, plNo %q, found %t; expected 100.0, 130.0, PL1, true", costB, priceB, plNoB, foundB)
	}

	// Case C: Sales of 12 PCS (OnHand = 33)
	sC, _ := domain.NewSalesDetail(1, "SI", "POSTED", t2, "INV1", "Cust", "SupplierA", 1, 12.0, "PCS", 150.0, 100.0, 0, 0, 0, "", "")
	stockC := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sC}, nil)
	costC, priceC, plNoC, foundC := stockC.GetOldestPLWithStock()
	if !foundC || costC != 110.0 || priceC != 143.0 || plNoC != "PL2" {
		t.Errorf("Case C failed: got cost %f, price %f, plNo %q, found %t; expected 110.0, 143.0, PL2, true", costC, priceC, plNoC, foundC)
	}

	// Case D: Sales of 28 PCS (OnHand = 17)
	sD, _ := domain.NewSalesDetail(1, "SI", "POSTED", t3, "INV1", "Cust", "SupplierA", 1, 28.0, "PCS", 150.0, 100.0, 0, 0, 0, "", "")
	stockD := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sD}, nil)
	costD, priceD, plNoD, foundD := stockD.GetOldestPLWithStock()
	if !foundD || costD != 120.0 || priceD != 156.0 || plNoD != "PL3" {
		t.Errorf("Case D failed: got cost %f, price %f, plNo %q, found %t; expected 120.0, 156.0, PL3, true", costD, priceD, plNoD, foundD)
	}

	// Case E: Sales of 45 PCS (OnHand = 0)
	sE, _ := domain.NewSalesDetail(1, "SI", "POSTED", t3, "INV1", "Cust", "SupplierA", 1, 45.0, "PCS", 150.0, 100.0, 0, 0, 0, "", "")
	stockE := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sE}, nil)
	if _, _, _, foundE := stockE.GetOldestPLWithStock(); foundE {
		t.Errorf("expected false when onHand is 0")
	}

	// Case F: Positive adjustment should NOT resurrect an already-depleted PL1
	posAdj, _ := domain.NewInventoryAdjustment(1, 0, t3, 1, "PCS", 5.0, 110.0, "Found extra")
	stockF := domain.NewItemStock(item, nil, receivingLogs, []domain.SalesDetail{sC}, []domain.InventoryAdjustment{posAdj})
	costF, _, plNoF, foundF := stockF.GetOldestPLWithStock()
	if !foundF || plNoF != "PL2" || costF != 110.0 {
		t.Errorf("Case F failed: positive adjustment resurrected PL1, got plNo=%q cost=%f", plNoF, costF)
	}

	// Case G: Multi-UOM receiving log converts Cost & SellingPrice to DefaultUOM
	boxUOM, _ := domain.NewUomSetting("BOX", 10.0)
	rlBox, _ := domain.NewReceivingLog(10, "SupplierA", t1, "PL-BOX", 1, 2.0, "BOX", 1000.0, 0, 0, 800.0, 130, "")
	stockBox := domain.NewItemStock(item, []domain.UomSetting{boxUOM}, []domain.ReceivingLog{rlBox}, nil, nil)
	costBox, priceBox, plBox, foundBox := stockBox.GetOldestPLWithStock()
	if !foundBox || plBox != "PL-BOX" || costBox != 80.0 || priceBox != 104.0 {
		t.Errorf("Case G failed: expected cost 80.0 and price 104.0 per PCS, got cost=%f price=%f pl=%q", costBox, priceBox, plBox)
	}

	// Case H: GetOldestPLWithStockForSupplier filters by supplier
	costSupB, priceSupB, plSupB, foundSupB := stockA.GetOldestPLWithStockForSupplier("SupplierB")
	if !foundSupB || plSupB != "PL3" || costSupB != 120.0 || priceSupB != 156.0 {
		t.Errorf("Case H failed: expected PL3 for SupplierB, got pl=%q cost=%f price=%f found=%t", plSupB, costSupB, priceSupB, foundSupB)
	}
}

func TestItemStock_CalculateOnHand_Status(t *testing.T) {
	item := domain.Item{ID: 1, Code: "ITEM001", DefaultUOM: "PCS"}
	rl, _ := domain.NewReceivingLog(1, "SupplierA", time.Now(), "PL1", 1, 10.0, "PCS", 100.0, 0, 0, 80.0, 130, "")

	s1, _ := domain.NewSalesDetail(1, "SI", "Posted", time.Now(), "INV1", "Cust", "SupplierA", 1, 3.0, "PCS", 150.0, 80.0, 0, 0, 0, "", "")
	s2, _ := domain.NewSalesDetail(2, "SI", "Active", time.Now(), "INV2", "Cust", "SupplierA", 1, 2.0, "PCS", 150.0, 80.0, 0, 0, 0, "", "")
	s3, _ := domain.NewSalesDetail(3, "SI", "Cancelled/Return", time.Now(), "INV3", "Cust", "SupplierA", 1, 4.0, "PCS", 150.0, 80.0, 0, 0, 0, "", "")

	sales := []domain.SalesDetail{s1, s2, s3}
	stock := domain.NewItemStock(item, nil, []domain.ReceivingLog{rl}, sales, nil)

	onHandSupplier := stock.CalculateOnHand("SupplierA")
	if onHandSupplier != 7.0 {
		t.Errorf("expected CalculateOnHand to be 7.0, got %f", onHandSupplier)
	}

	onHandGlobal := stock.CalculateGlobalOnHand()
	if onHandGlobal != 7.0 {
		t.Errorf("expected CalculateGlobalOnHand to be 7.0, got %f", onHandGlobal)
	}
}
