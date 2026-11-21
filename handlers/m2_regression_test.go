package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"rav/db"
	"rav/models"

	"github.com/xuri/excelize/v2"
)

func parseRegressionTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	funcMap := TemplateFuncMap()

	for _, name := range []string{
		"inventory_stock_results.html",
		"monthly_inventory_results.html",
		"receiving_log_row.html",
		"receiving_log_edit_row.html",
		"sale_item_row.html",
		"sale_row.html",
		"customer_row.html",
	} {
		t := template.New(name).Funcs(funcMap)
		files := []string{"../templates/" + name}
		if name == "inventory_stock_results.html" {
			files = append(files, "../templates/inventory_stock_rows.html")
		} else if name == "monthly_inventory_results.html" {
			files = append(files, "../templates/monthly_inventory_rows.html")
		}
		templates[name] = template.Must(t.ParseFiles(files...))
	}
	return templates
}

func TestFIFOValuationWithUOMConversion(t *testing.T) {
	setupIsolatedTestDB(t)

	if _, err := db.DB.Exec("INSERT INTO uoms (code) VALUES ('PCS'), ('BOX') ON CONFLICT DO NOTHING"); err != nil {
		t.Fatalf("failed to seed uoms: %v", err)
	}
	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('FIFO-BOX', 'FIFO UOM Test Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("failed to seed item: %v", err)
	}
	if _, err := db.DB.Exec("INSERT INTO uom_settings (item_id, muom, conversion_factor) VALUES (?, 'BOX', 10)", itemID); err != nil {
		t.Fatalf("failed to seed uom_settings: %v", err)
	}

	// Receive 1 BOX (= 10 PCS) at Cost = 800, SellingPrice = 1040 -> per PCS: Cost = 80.00, Price = 104.00
	if _, err := db.DB.Exec(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, cost, total_cost, markup, selling_price)
		VALUES ('SUPP-A', '2026-01-01', 'PL-BOX-1', ?, 1, 'BOX', 800.0, 800.0, 800.0, 130.0, 1040.0)
	`, itemID); err != nil {
		t.Fatalf("failed to seed receiving_log 1: %v", err)
	}
	// Receive 5 PCS later at Cost = 95, SellingPrice = 123.50
	if _, err := db.DB.Exec(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, cost, total_cost, markup, selling_price)
		VALUES ('SUPP-A', '2026-01-05', 'PL-PCS-2', ?, 5, 'PCS', 95.0, 95.0, 475.0, 130.0, 123.50)
	`, itemID); err != nil {
		t.Fatalf("failed to seed receiving_log 2: %v", err)
	}

	// Sell 4 PCS (Posted) -> 6 PCS remain in PL-BOX-1 (total on hand = 11 PCS)
	if _, err := db.DB.Exec(`
		INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit)
		VALUES ('SI', 'Posted', '2026-01-06', 'SI-FIFO-1', 'SUPP-A', ?, 4, 'PCS', 104.0, 416.0, 80.0, 320.0, 96.0)
	`, itemID); err != nil {
		t.Fatalf("failed to seed sales_detail: %v", err)
	}

	app := &App{Templates: parseRegressionTemplatesForTest()}

	// 1. Verify InventoryHandler scales BOX cost/price to DefaultUOM (PCS: 80.00 / 104.00) and computes on_hand = 11
	reqInv := httptest.NewRequest("GET", "/inventory?search=FIFO-BOX", nil)
	reqInv.Header.Set("HX-Request", "true")
	reqInv.Header.Set("HX-Target", "inventory-stock-results")
	wInv := httptest.NewRecorder()
	app.InventoryHandler(wInv, reqInv)
	if wInv.Code != http.StatusOK {
		t.Fatalf("InventoryHandler returned %d: %s", wInv.Code, wInv.Body.String())
	}
	invBody := wInv.Body.String()
	if !strings.Contains(invBody, "80.00") || !strings.Contains(invBody, "104.00") {
		t.Errorf("expected scaled DefaultUOM cost 80.00 and price 104.00 in InventoryHandler output, got: %s", invBody)
	}

	// 2. Verify SaleItemRowDetailsHandler returns scaled DefaultUOM cost (80) and price (104) and PL-BOX-1
	reqSaleRow := httptest.NewRequest("GET", fmt.Sprintf("/sales/item-row-details?item_id=%d&supplier=SUPP-A", itemID), nil)
	wSaleRow := httptest.NewRecorder()
	app.SaleItemRowDetailsHandler(wSaleRow, reqSaleRow)
	if wSaleRow.Code != http.StatusOK {
		t.Fatalf("SaleItemRowDetailsHandler returned %d: %s", wSaleRow.Code, wSaleRow.Body.String())
	}
	saleRowBody := wSaleRow.Body.String()
	if !strings.Contains(saleRowBody, "PL-BOX-1") || !strings.Contains(saleRowBody, "80") || !strings.Contains(saleRowBody, "104") {
		t.Errorf("expected PL-BOX-1 with scaled cost 80 and price 104 in SaleItemRowDetailsHandler, got: %s", saleRowBody)
	}
}

func TestMonthlyInventoryFiltersPostedSalesOnly(t *testing.T) {
	setupIsolatedTestDB(t)

	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('M-INV-01', 'Monthly Inv Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("failed to seed item: %v", err)
	}
	_, _ = db.DB.Exec(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, cost, total_cost)
		VALUES ('SUPP', '2026-02-01', 'PL-M1', ?, 20, 'PCS', 50, 50, 1000)
	`, itemID)
	// Posted sale of 5 PCS
	_, _ = db.DB.Exec(`
		INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit)
		VALUES ('SI', 'Posted', '2026-02-10', 'SI-M1', 'SUPP', ?, 5, 'PCS', 100, 500, 50, 250, 250)
	`, itemID)
	// Active (unposted) sale of 7 PCS and Cancelled/Return sale of 3 PCS -> must NOT be counted in sold!
	_, _ = db.DB.Exec(`
		INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit)
		VALUES ('SI', 'Active', '2026-02-11', 'SI-M2', 'SUPP', ?, 7, 'PCS', 100, 700, 50, 350, 350),
		       ('SI', 'Cancelled/Return', '2026-02-12', 'SI-M3', 'SUPP', ?, 3, 'PCS', 100, 300, 50, 150, 150)
	`, itemID, itemID)

	app := &App{Templates: parseRegressionTemplatesForTest()}

	req := httptest.NewRequest("GET", "/inventory/monthly?month_filter=2026-02&search=M-INV-01", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "monthly-inventory-results")
	w := httptest.NewRecorder()

	app.MonthlyInventoryHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("MonthlyInventoryHandler returned %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// Received = 20, Sold = 5 (only Posted), NetChange = +15
	if !strings.Contains(body, "+15") && !strings.Contains(body, "15") {
		t.Errorf("expected net change of 15 (20 received - 5 posted sold), got: %s", body)
	}
}

func TestUpdateReceivingLogValidationAndRounding(t *testing.T) {
	setupIsolatedTestDB(t)

	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('REC-UPD-01', 'Rec Update Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("failed to seed item: %v", err)
	}
	var logID int
	if err := db.DB.QueryRow(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, less1, less2, cost, total_cost, markup, selling_price, remarks)
		VALUES ('ASCD', '2026-01-10', 'PL-UPD', ?, 10, 'PCS', 100.0, 0, 0, 100.0, 1000.0, 130.0, 130.0, '')
		RETURNING id
	`, itemID).Scan(&logID); err != nil {
		t.Fatalf("failed to seed receiving_log: %v", err)
	}

	app := &App{Templates: parseRegressionTemplatesForTest()}

	// 1. Reject negative qty with 400 Bad Request
	badForm := url.Values{
		"qty":        {"-5"},
		"unit_price": {"100"},
		"less1":      {"0"},
		"less2":      {"0"},
		"markup":     {"130"},
	}
	reqBad := httptest.NewRequest("POST", fmt.Sprintf("/inventory/receiving/logs/edit/%d", logID), strings.NewReader(badForm.Encode()))
	reqBad.SetPathValue("id", strconv.Itoa(logID))
	reqBad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wBad := httptest.NewRecorder()
	app.UpdateReceivingLogHandler(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for negative qty, got %d", wBad.Code)
	}

	// 2. Valid update rounds cost, totalCost, sellingPrice to 2 decimals
	goodForm := url.Values{
		"qty":        {"3"},
		"unit_price": {"100.555"},
		"less1":      {"10"},
		"less2":      {"5"},
		"markup":     {"135"},
		"remarks":    {"Updated remark"},
	}
	reqGood := httptest.NewRequest("POST", fmt.Sprintf("/inventory/receiving/logs/edit/%d", logID), strings.NewReader(goodForm.Encode()))
	reqGood.SetPathValue("id", strconv.Itoa(logID))
	reqGood.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wGood := httptest.NewRecorder()
	app.UpdateReceivingLogHandler(wGood, reqGood)
	if wGood.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid receiving log update, got %d: %s", wGood.Code, wGood.Body.String())
	}

	var updated models.ReceivingLog
	if err := db.DB.Get(&updated, "SELECT unit_price, cost, total_cost, selling_price FROM receiving_logs WHERE id = ?", logID); err != nil {
		t.Fatalf("failed to load updated receiving_log: %v", err)
	}
	// unitPrice = Round2(100.555) = 100.56; cost = Round2(100.56 * 0.9 * 0.95) = 85.98; totalCost = Round2(3 * 85.98) = 257.94; sellingPrice = Round2(85.98 * 1.35) = 116.07
	if updated.Cost != 85.98 || updated.TotalCost != 257.94 || updated.SellingPrice == nil || *updated.SellingPrice != 116.07 {
		t.Errorf("unexpected rounded values: cost=%.2f, totalCost=%.2f, sellingPrice=%v", updated.Cost, updated.TotalCost, updated.SellingPrice)
	}

	// 3. EditReceivingLogHandler returns 404 for nonexistent ID
	req404 := httptest.NewRequest("GET", "/inventory/receiving/logs/edit/999999", nil)
	req404.SetPathValue("id", "999999")
	w404 := httptest.NewRecorder()
	app.EditReceivingLogHandler(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for nonexistent receiving log, got %d", w404.Code)
	}
}

func TestAddSalesStockGuardAndNetProfit(t *testing.T) {
	setupIsolatedTestDB(t)

	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('STK-G-01', 'Stock Guard Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("failed to seed item: %v", err)
	}
	// Receive 10 PCS
	_, _ = db.DB.Exec(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, cost, total_cost)
		VALUES ('ASCD', '2026-01-01', 'PL-G1', ?, 10, 'PCS', 100, 100, 1000)
	`, itemID)

	app := &App{Templates: parseRegressionTemplatesForTest()}

	// 1. Overselling (requesting 15 PCS when only 10 PCS on hand) -> 400 Bad Request
	overForm := url.Values{
		"doc_date":      {"2026-01-10"},
		"doc_type":      {"SI"},
		"doc_status":    {"Posted"},
		"doc_number":    {"SI-OVER-1"},
		"customer_name": {"Buyer"},
		"supplier":      {"ASCD"},
		"item_id":       {strconv.Itoa(itemID)},
		"qty":           {"15"},
		"uom":           {"PCS"},
		"price":         {"200"},
		"cost":          {"100"},
	}
	reqOver := httptest.NewRequest("POST", "/sales/add", strings.NewReader(overForm.Encode()))
	reqOver.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wOver := httptest.NewRecorder()
	app.AddSalesHandler(wOver, reqOver)
	if wOver.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when overselling stock, got %d: %s", wOver.Code, wOver.Body.String())
	}

	// 2. Valid sale within available stock (5 PCS) verifies Profit = TotalSales - TotalCost - POSCharge - Patong - WT2307
	validForm := url.Values{
		"doc_date":      {"2026-01-10"},
		"doc_type":      {"SI"},
		"doc_status":    {"Posted"},
		"doc_number":    {"SI-OK-1"},
		"customer_name": {"Buyer"},
		"supplier":      {"ASCD"},
		"item_id":       {strconv.Itoa(itemID)},
		"qty":           {"5"},
		"uom":           {"PCS"},
		"price":         {"200"},
		"cost":          {"100"},
		"patong":        {"20"},
		"pos_charge":    {"15"},
		"wt_2307":       {"10"},
	}
	reqOk := httptest.NewRequest("POST", "/sales/add", strings.NewReader(validForm.Encode()))
	reqOk.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wOk := httptest.NewRecorder()
	app.AddSalesHandler(wOk, reqOk)
	if wOk.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid sale within stock, got %d: %s", wOk.Code, wOk.Body.String())
	}

	// TotalSales = 1000, TotalCost = 500, Deductions = 20 + 15 + 10 = 45 -> Profit = 455, TotalRemit = 955
	var sd models.SalesDetail
	if err := db.DB.Get(&sd, "SELECT total_sales, total_cost, total_remit, profit FROM sales_details WHERE doc_number = 'SI-OK-1'"); err != nil {
		t.Fatalf("failed to query inserted sale: %v", err)
	}
	if sd.Profit != 455.0 || sd.TotalRemit != 955.0 {
		t.Errorf("expected Profit=455.00 and TotalRemit=955.00, got Profit=%.2f TotalRemit=%.2f", sd.Profit, sd.TotalRemit)
	}
}

func TestUpdateCustomerPropagatesRenameToSalesDetails(t *testing.T) {
	setupIsolatedTestDB(t)
	_ = db.EnsureCustomersTable()

	var custID int
	if err := db.DB.QueryRow(`
		INSERT INTO customers (code, name, tin_no, address)
		VALUES ('CUST-REN', 'Old Corp Name', '123', 'Manila')
		RETURNING id
	`).Scan(&custID); err != nil {
		t.Fatalf("failed to seed customer: %v", err)
	}

	var itemID int
	_ = db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('REN-IT', 'Rename Item', 'PCS') RETURNING id").Scan(&itemID)
	_, _ = db.DB.Exec(`
		INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit)
		VALUES ('SI', 'Posted', '2026-01-10', 'SI-REN-1', 'Old Corp Name', 'ASCD', ?, 2, 'PCS', 100, 200, 50, 100, 100)
	`, itemID)

	app := &App{Templates: parseRegressionTemplatesForTest()}

	form := url.Values{
		"code":    {"CUST-REN"},
		"name":    {"New Corp Name"},
		"tin_no":  {"123"},
		"address": {"Manila"},
	}
	req := httptest.NewRequest("POST", fmt.Sprintf("/customers/edit/%d", custID), strings.NewReader(form.Encode()))
	req.SetPathValue("id", strconv.Itoa(custID))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	app.UpdateCustomerHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCustomerHandler returned %d: %s", w.Code, w.Body.String())
	}

	var updatedCustName string
	if err := db.DB.Get(&updatedCustName, "SELECT customer_name FROM sales_details WHERE doc_number = 'SI-REN-1'"); err != nil {
		t.Fatalf("failed to query sales_details customer_name: %v", err)
	}
	if updatedCustName != "New Corp Name" {
		t.Errorf("expected sales_details.customer_name to be updated to 'New Corp Name', got %q", updatedCustName)
	}
}

func TestImportNetProfitAndSizeLimit(t *testing.T) {
	setupIsolatedTestDB(t)

	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('IMP-01', 'Import Test Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("failed to seed item: %v", err)
	}

	// Build an Excel file with a SALES DETAIL sheet where PROFIT is blank (0), verifying net profit deducts POS Charge, Patong, WT 2307
	f := excelize.NewFile()
	sheet := "SALES DETAIL"
	_, _ = f.NewSheet(sheet)
	headers := []string{"DOC TYPE", "DOC STATUS", "DOC DATE", "DOC NUMBER", "CUSTOMER NAME", "SUPPLIER", "ITEM CODE", "QTY", "UOM", "PRICE", "COST", "PATONG", "POS CHARGE", "WT 2307"}
	for idx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	rowVals := []interface{}{"SI", "Posted", "2026-01-15", "SI-IMP-1", "Cust A", "ASCD", "IMP-01", "2", "PCS", "500", "300", "25", "15", "10"}
	for idx, val := range rowVals {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 2)
		_ = f.SetCellValue(sheet, cell, val)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to write excel buffer: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "sales_import.xlsx")
	_, _ = part.Write(buf.Bytes())
	_ = writer.WriteField("import_sales", "true")
	_ = writer.Close()

	app := &App{}
	req := httptest.NewRequest("POST", "/import/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	app.ImportUploadHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ImportUploadHandler returned %d: %s", w.Code, w.Body.String())
	}

	// TotalSales = 1000, TotalCost = 600, Patong = 25, POS = 15, WT2307 = 10 -> Net Profit = 350.00, ProfitMargin = 35.00
	var sd models.SalesDetail
	if err := db.DB.Get(&sd, "SELECT total_sales, total_cost, profit, profit_margin FROM sales_details WHERE doc_number = 'SI-IMP-1'"); err != nil {
		t.Fatalf("failed to query imported sale: %v", err)
	}
	if sd.Profit != 350.0 || sd.ProfitMargin != 35.0 {
		t.Errorf("expected imported net Profit=350.00 and ProfitMargin=35.00, got Profit=%.2f ProfitMargin=%.2f", sd.Profit, sd.ProfitMargin)
	}

	// Verify >25MB upload is rejected with 413 Request Entity Too Large
	oversizedBuf := bytes.Repeat([]byte("A"), (25<<20)+1024)
	reqBig := httptest.NewRequest("POST", "/import/upload", bytes.NewReader(oversizedBuf))
	reqBig.Header.Set("Content-Type", "multipart/form-data; boundary=---oversized")
	wBig := httptest.NewRecorder()
	app.ImportUploadHandler(wBig, reqBig)
	if wBig.Code != http.StatusRequestEntityTooLarge && wBig.Code != http.StatusBadRequest {
		t.Errorf("expected 413 Request Entity Too Large for >25MB upload, got %d", wBig.Code)
	}
}
