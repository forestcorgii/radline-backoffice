package handlers

import (
	"bytes"
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

func parseM1aTemplatesForTest(t *testing.T) map[string]*template.Template {
	t.Helper()
	templates := make(map[string]*template.Template)
	funcMap := TemplateFuncMap()

	parsePage := func(page string, extras ...string) {
		files := append([]string{"../templates/base.html", "../templates/" + page}, extras...)
		templates[page] = template.Must(template.New(page).Funcs(funcMap).ParseFiles(files...))
	}
	parseFrag := func(frag string, extras ...string) {
		files := append([]string{"../templates/" + frag}, extras...)
		templates[frag] = template.Must(template.New(frag).Funcs(funcMap).ParseFiles(files...))
	}

	parsePage("items.html", "../templates/item_row.html", "../templates/item_rows.html", "../templates/item_edit_row.html", "../templates/brand_select.html", "../templates/category_select.html", "../templates/uom_select.html", "../templates/items_results.html")
	parsePage("inventory.html", "../templates/inventory_stock_rows.html", "../templates/receiving_item_row.html", "../templates/adjustment_item_row.html", "../templates/receive_stock_form.html", "../templates/adjust_stock_form.html")
	parsePage("receiving_logs.html", "../templates/receiving_rows.html", "../templates/receiving_logs_results.html", "../templates/receiving_log_row.html", "../templates/receiving_item_row.html", "../templates/receive_stock_form.html", "../templates/item_select.html")
	parsePage("adjustment_logs.html", "../templates/adjustment_rows.html", "../templates/adjustment_item_row.html", "../templates/adjust_stock_form.html")
	parsePage("activity_logs.html", "../templates/activity_logs_results.html")
	parsePage("sales.html", "../templates/sales_rows.html", "../templates/sale_item_row.html", "../templates/sales_results.html", "../templates/item_select.html", "../templates/uom_select.html", "../templates/sale_row.html", "../templates/sale_edit_row.html")

	parseFrag("item_row.html")
	parseFrag("item_edit_row.html")
	parseFrag("items_results.html", "../templates/item_row.html", "../templates/item_rows.html")
	parseFrag("inventory_stock_results.html", "../templates/inventory_stock_rows.html")
	parseFrag("receiving_item_row.html")
	parseFrag("adjustment_item_row.html")
	parseFrag("receiving_log_row.html")
	parseFrag("receiving_log_edit_row.html")
	parseFrag("receiving_logs_results.html", "../templates/receiving_rows.html", "../templates/receiving_log_row.html")
	parseFrag("adjustment_rows.html")
	parseFrag("adjustment_logs_results.html", "../templates/adjustment_rows.html")
	parseFrag("activity_logs_results.html")
	parseFrag("sale_row.html")
	parseFrag("sale_edit_row.html")
	parseFrag("sale_item_row.html")
	parseFrag("sales_results.html", "../templates/sales_rows.html", "../templates/sale_row.html", "../templates/sale_edit_row.html")

	return templates
}

func TestReceiveStockHandler_UnitCostBoundsCheck(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{}

	form := url.Values{}
	form.Set("date", "2026-01-15")
	form.Set("supplier", "ASCD")
	form.Set("pl_no", "PL-BOUNDS")
	form.Add("item_id", "1")
	form.Add("item_id", "2")
	form.Add("qty", "5")
	form.Add("qty", "3")
	form.Add("uom", "PCS")
	form.Add("uom", "PCS")
	form.Add("unit_price", "100")
	form.Add("unit_price", "200")
	// Only 1 unit_cost for 2 items -> must return 400 without panicking
	form.Add("unit_cost", "100")

	req := httptest.NewRequest(http.MethodPost, "/inventory/receiving/add", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	app.ReceiveStockHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for mismatched unit_cost length, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAddSalesAndDropdownsAndReceivingIDValidation(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{Templates: parseM1aTemplatesForTest(t)}

	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('M1A-01', 'M1a Test Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("Seed item failed: %v", err)
	}
	if _, err := db.DB.Exec("INSERT INTO uom_settings (item_id, muom, conversion_factor) VALUES (?, 'BOX', 10)", itemID); err != nil {
		t.Fatalf("Seed uom_settings failed: %v", err)
	}

	// 1. AddSalesHandler uses tx.Prepare(...)
	form := url.Values{}
	form.Set("doc_date", "2026-02-01")
	form.Set("doc_type", "SI")
	form.Set("doc_number", "SI-M1A-1")
	form.Set("customer_name", "Walk-In")
	form.Set("supplier", "ASCD")
	form.Set("doc_status", "Posted")
	form.Add("item_id", strconv.Itoa(itemID))
	form.Add("qty", "2")
	form.Add("uom", "PCS")
	form.Add("price", "150")
	form.Add("cost", "100")

	req := httptest.NewRequest(http.MethodPost, "/sales/add", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	app.AddSalesHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("AddSalesHandler expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var saleCount int
	if err := db.DB.Get(&saleCount, "SELECT COUNT(*) FROM sales_details WHERE doc_number = 'SI-M1A-1'"); err != nil || saleCount != 1 {
		t.Fatalf("Expected 1 persisted sale detail, got %d (err=%v)", saleCount, err)
	}

	// 2. UOM UNION dropdown queries with AS u alias
	for _, path := range []string{
		"/inventory/receiving/item-row-details?item_id=" + strconv.Itoa(itemID),
		"/inventory/adjustments/item-row-details?item_id=" + strconv.Itoa(itemID),
		"/sales/item-row-details?item_id=" + strconv.Itoa(itemID),
	} {
		reqU := httptest.NewRequest(http.MethodGet, path, nil)
		wU := httptest.NewRecorder()
		switch {
		case strings.Contains(path, "receiving"):
			app.ReceivingItemRowDetailsHandler(wU, reqU)
		case strings.Contains(path, "adjustments"):
			app.AdjustmentItemRowDetailsHandler(wU, reqU)
		default:
			app.SaleItemRowDetailsHandler(wU, reqU)
		}
		if wU.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d: %s", path, wU.Code, wU.Body.String())
		}
		if !strings.Contains(wU.Body.String(), "BOX") || !strings.Contains(wU.Body.String(), "PCS") {
			t.Errorf("%s expected both BOX and PCS options in dropdown, got: %s", path, wU.Body.String())
		}
	}

	// 3. Non-integer IDs return 400 Bad Request in ReceivingLog handlers
	for _, tc := range []struct {
		name string
		fn   http.HandlerFunc
	}{
		{"DeleteReceivingLogHandler", app.DeleteReceivingLogHandler},
		{"EditReceivingLogHandler", app.EditReceivingLogHandler},
		{"UpdateReceivingLogHandler", app.UpdateReceivingLogHandler},
	} {
		reqBad := httptest.NewRequest(http.MethodPost, "/test/abc", nil)
		reqBad.SetPathValue("id", "abc")
		wBad := httptest.NewRecorder()
		tc.fn(wBad, reqBad)
		if wBad.Code != http.StatusBadRequest {
			t.Errorf("%s expected 400 on invalid id 'abc', got %d", tc.name, wBad.Code)
		}
	}
}

func buildTestImportWorkbook(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()

	// Rename Sheet1 to ITEM LIST
	f.SetSheetName("Sheet1", "ITEM LIST")
	_ = f.SetSheetRow("ITEM LIST", "A1", &[]interface{}{"ITEM CODE", "ITEM DESCRIPTION", "DEFAULT UOM"})
	_ = f.SetSheetRow("ITEM LIST", "A2", &[]interface{}{"IMP-01", "Imported Hammer", "PCS"})

	_, _ = f.NewSheet("UOM SETTINGS")
	_ = f.SetSheetRow("UOM SETTINGS", "A1", &[]interface{}{"ITEM CODE", "DEFAULT UOM", "MUOM", "CONVERSION"})
	_ = f.SetSheetRow("UOM SETTINGS", "A2", &[]interface{}{"IMP-01", "PCS", "BOX", "12"})
	// Duplicate (item_id, muom) row to verify ON CONFLICT (item_id, muom) DO NOTHING
	_ = f.SetSheetRow("UOM SETTINGS", "A3", &[]interface{}{"IMP-01", "PCS", "BOX", "12"})

	_, _ = f.NewSheet("RECEIVING")
	_ = f.SetSheetRow("RECEIVING", "A1", &[]interface{}{"SUPPLIER", "DATE", "PL NO.", "ITEM CODE", "QTY", "UOM", "UNIT PRICE", "COST", "TOTAL COST", "SELLING PRICE"})
	_ = f.SetSheetRow("RECEIVING", "A2", &[]interface{}{"ASCD", "2026-01-20", "PL-IMP-1", "IMP-01", "10", "PCS", "150.50", "150.50", "1505.00", "195.65"})

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("Failed to write workbook: %v", err)
	}
	return buf.Bytes()
}

func makeMultipartUploadReq(t *testing.T, xlsxBytes []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "import.xlsx")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := fw.Write(xlsxBytes); err != nil {
		t.Fatalf("Write xlsx failed: %v", err)
	}
	for _, field := range []string{"import_items", "import_uoms", "import_receiving"} {
		_ = mw.WriteField(field, "true")
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/import/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestImportUploadHandler_UnitPriceAndUOMConflictAndRollback(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{}
	xlsx := buildTestImportWorkbook(t)

	// 1. Happy path: UNIT PRICE parsed (150.50 != 0) & duplicate UOM SETTINGS ignored without error
	req := makeMultipartUploadReq(t, xlsx)
	w := httptest.NewRecorder()
	app.ImportUploadHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ImportUploadHandler expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var unitPrice float64
	if err := db.DB.Get(&unitPrice, "SELECT unit_price FROM receiving_logs WHERE pl_no = 'PL-IMP-1'"); err != nil {
		t.Fatalf("Failed to query imported receiving_log: %v", err)
	}
	if unitPrice != 150.50 {
		t.Errorf("Expected imported unit_price=150.50, got %v", unitPrice)
	}

	var uomCount int
	if err := db.DB.Get(&uomCount, "SELECT COUNT(*) FROM uom_settings"); err != nil {
		t.Fatalf("Failed to count uom_settings: %v", err)
	}
	if uomCount != 1 {
		t.Errorf("Expected 1 uom_settings row after duplicate ON CONFLICT, got %d", uomCount)
	}

	// 2. Rollback test: force SQL error mid-transaction on receiving_logs -> item/uom inserts must roll back
	setupIsolatedTestDB(t)
	if _, err := db.DB.Exec(`
		CREATE TRIGGER abort_receiving_insert BEFORE INSERT ON receiving_logs
		BEGIN
			SELECT RAISE(ABORT, 'forced import failure');
		END;
	`); err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	reqFail := makeMultipartUploadReq(t, xlsx)
	wFail := httptest.NewRecorder()
	app.ImportUploadHandler(wFail, reqFail)
	if wFail.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500 on mid-import SQL error, got %d: %s", wFail.Code, wFail.Body.String())
	}

	var itemCount int
	if err := db.DB.Get(&itemCount, "SELECT COUNT(*) FROM items WHERE code = 'IMP-01'"); err != nil {
		t.Fatalf("Failed to query items after rollback: %v", err)
	}
	if itemCount != 0 {
		t.Errorf("Expected transaction rollback to revert IMP-01 item insert, found %d rows", itemCount)
	}
}

func TestNullableSQLScansAndTemplates_ZeroNilArtifacts(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{Templates: parseM1aTemplatesForTest(t)}

	// 1. activity_logs with user_id = NULL
	if _, err := db.DB.Exec(`
		INSERT INTO activity_logs (user_id, username, action, entity_type, entity_id, details, ip_address)
		VALUES (NULL, 'anon', 'LOGIN', 'Session', '1', 'null user_id test', '127.0.0.1')
	`); err != nil {
		t.Fatalf("Insert null activity_log failed: %v", err)
	}
	reqAct := httptest.NewRequest(http.MethodGet, "/activity-logs", nil)
	wAct := httptest.NewRecorder()
	app.ActivityLogsHandler(wAct, reqAct)
	if wAct.Code != http.StatusOK {
		t.Fatalf("ActivityLogsHandler expected 200, got %d: %s", wAct.Code, wAct.Body.String())
	}

	// 2. items with NULL model, brand_id, category_id, variation, remarks
	var nullItemID int
	if err := db.DB.QueryRow(`
		INSERT INTO items (code, description, default_uom, model, brand_id, category_id, variation, remarks)
		VALUES ('NULL-ITEM', 'Null Fields Item', 'PCS', NULL, NULL, NULL, NULL, NULL)
		RETURNING id
	`).Scan(&nullItemID); err != nil {
		t.Fatalf("Insert null item failed: %v", err)
	}

	reqItems := httptest.NewRequest(http.MethodGet, "/items", nil)
	wItems := httptest.NewRecorder()
	app.ItemsHandler(wItems, reqItems)
	if wItems.Code != http.StatusOK {
		t.Fatalf("ItemsHandler expected 200, got %d", wItems.Code)
	}
	if strings.Contains(wItems.Body.String(), "<nil>") {
		t.Errorf("ItemsHandler rendered <nil> in HTML")
	}

	reqEditItem := httptest.NewRequest(http.MethodGet, "/items/edit/"+strconv.Itoa(nullItemID), nil)
	reqEditItem.SetPathValue("id", strconv.Itoa(nullItemID))
	wEditItem := httptest.NewRecorder()
	app.EditItemFormHandler(wEditItem, reqEditItem)
	if wEditItem.Code != http.StatusOK {
		t.Fatalf("EditItemFormHandler expected 200, got %d", wEditItem.Code)
	}
	if strings.Contains(wEditItem.Body.String(), "<nil>") {
		t.Errorf("EditItemFormHandler rendered <nil> in input value: %s", wEditItem.Body.String())
	}

	reqSearch := httptest.NewRequest(http.MethodGet, "/items/search?q=NULL-ITEM", nil)
	wSearch := httptest.NewRecorder()
	app.SearchItemsHandler(wSearch, reqSearch)
	if wSearch.Code != http.StatusOK {
		t.Fatalf("SearchItemsHandler expected 200, got %d", wSearch.Code)
	}

	// 3. receiving_logs with NULL pl_no, unit_price, selling_price
	var nullRecID int
	if err := db.DB.QueryRow(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, less1, less2, cost, total_cost, markup, selling_price, remarks)
		VALUES ('ASCD', '2026-01-15', NULL, ?, 10, 'PCS', NULL, 0, 0, 50.0, 500.0, 130.0, NULL, '')
		RETURNING id
	`, nullItemID).Scan(&nullRecID); err != nil {
		t.Fatalf("Insert null receiving_log failed: %v", err)
	}

	reqInv := httptest.NewRequest(http.MethodGet, "/inventory", nil)
	wInv := httptest.NewRecorder()
	app.InventoryHandler(wInv, reqInv)
	if wInv.Code != http.StatusOK {
		t.Fatalf("InventoryHandler expected 200, got %d: %s", wInv.Code, wInv.Body.String())
	}

	reqRecDetails := httptest.NewRequest(http.MethodGet, "/inventory/receiving/item-row-details?item_id="+strconv.Itoa(nullItemID), nil)
	wRecDetails := httptest.NewRecorder()
	app.ReceivingItemRowDetailsHandler(wRecDetails, reqRecDetails)
	if wRecDetails.Code != http.StatusOK {
		t.Fatalf("ReceivingItemRowDetailsHandler expected 200, got %d", wRecDetails.Code)
	}

	reqRecLogs := httptest.NewRequest(http.MethodGet, "/inventory/receiving/logs", nil)
	wRecLogs := httptest.NewRecorder()
	app.ReceivingLogsHandler(wRecLogs, reqRecLogs)
	if wRecLogs.Code != http.StatusOK {
		t.Fatalf("ReceivingLogsHandler expected 200, got %d", wRecLogs.Code)
	}
	if body := wRecLogs.Body.String(); strings.Contains(body, "<nil>") || strings.Contains(body, "%!f") {
		t.Errorf("ReceivingLogsHandler rendered <nil> or %%!f: %s", body)
	}

	reqEditRec := httptest.NewRequest(http.MethodGet, "/inventory/receiving/logs/edit/"+strconv.Itoa(nullRecID), nil)
	reqEditRec.SetPathValue("id", strconv.Itoa(nullRecID))
	wEditRec := httptest.NewRecorder()
	app.EditReceivingLogHandler(wEditRec, reqEditRec)
	if wEditRec.Code != http.StatusOK {
		t.Fatalf("EditReceivingLogHandler expected 200, got %d: %s", wEditRec.Code, wEditRec.Body.String())
	}
	if body := wEditRec.Body.String(); strings.Contains(body, "<nil>") || strings.Contains(body, "%!f") {
		t.Errorf("EditReceivingLogHandler rendered <nil> or %%!f: %s", body)
	}

	// 4. sales_details with NULL customer_name, tin_no, address, ref_pl
	var nullSaleID int
	if err := db.DB.QueryRow(`
		INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, tin_no, address, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, patong, pos_charge, wt_2307, total_remit, profit, profit_margin, remarks, ref_pl)
		VALUES ('SI', 'Active', '2026-01-20', 'SI-NULL-1', NULL, NULL, NULL, 'ASCD', ?, 1, 'PCS', 100, 100, 50, 50, 0, 0, 0, 100, 50, 50, '', NULL)
		RETURNING id
	`, nullItemID).Scan(&nullSaleID); err != nil {
		t.Fatalf("Insert null sales_detail failed: %v", err)
	}

	for _, isEdit := range []string{"0", "1"} {
		reqSales := httptest.NewRequest(http.MethodGet, "/sales?is_edit="+isEdit, nil)
		wSales := httptest.NewRecorder()
		app.SalesHandler(wSales, reqSales)
		if wSales.Code != http.StatusOK {
			t.Fatalf("SalesHandler(is_edit=%s) expected 200, got %d", isEdit, wSales.Code)
		}
		if strings.Contains(wSales.Body.String(), "<nil>") {
			t.Errorf("SalesHandler(is_edit=%s) rendered <nil> in HTML", isEdit)
		}
	}

	reqStatus := httptest.NewRequest(http.MethodPost, "/sales/update-status/"+strconv.Itoa(nullSaleID)+"?doc_status=Posted", nil)
	reqStatus.SetPathValue("id", strconv.Itoa(nullSaleID))
	wStatus := httptest.NewRecorder()
	app.UpdateSalesStatusHandler(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK {
		t.Fatalf("UpdateSalesStatusHandler expected 200, got %d: %s", wStatus.Code, wStatus.Body.String())
	}
	if strings.Contains(wStatus.Body.String(), "<nil>") {
		t.Errorf("UpdateSalesStatusHandler rendered <nil> in HTML: %s", wStatus.Body.String())
	}

	// 5. inventory_adjustments with NULL adjustment_id and remarks
	if _, err := db.DB.Exec(`
		INSERT INTO inventory_adjustments (adjustment_id, date, item_id, uom, adjustment_qty, cost, remarks)
		VALUES (NULL, '2026-01-22', ?, 'PCS', -1, 50.0, NULL)
	`, nullItemID); err != nil {
		t.Fatalf("Insert null inventory_adjustment failed: %v", err)
	}
	reqAdj := httptest.NewRequest(http.MethodGet, "/inventory/adjustments/logs", nil)
	wAdj := httptest.NewRecorder()
	app.AdjustmentLogsHandler(wAdj, reqAdj)
	if wAdj.Code != http.StatusOK {
		t.Fatalf("AdjustmentLogsHandler expected 200, got %d: %s", wAdj.Code, wAdj.Body.String())
	}
	if strings.Contains(wAdj.Body.String(), "<nil>") {
		t.Errorf("AdjustmentLogsHandler rendered <nil> in HTML: %s", wAdj.Body.String())
	}

	// 6. SavePurchaseOrderHandler apply_vat=false & apply_vat=true roundtrip
	for _, wantVAT := range []bool{false, true} {
		poNo := "PO-VAT-" + strconv.FormatBool(wantVAT)
		form := url.Values{}
		form.Set("po_no", poNo)
		form.Set("date", "2026-01-25")
		form.Set("vendor_name", "VAT Test Vendor")
		form.Set("apply_vat", strconv.FormatBool(wantVAT))
		form.Add("description", "Bolt")
		form.Add("qty", "1")
		form.Add("unit_cost", "100")

		reqPO := httptest.NewRequest(http.MethodPost, "/forms/purchase-order/save", strings.NewReader(form.Encode()))
		reqPO.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		wPO := httptest.NewRecorder()
		app.SavePurchaseOrderHandler(wPO, reqPO)
		if wPO.Code != http.StatusOK {
			t.Fatalf("SavePurchaseOrderHandler(%v) expected 200, got %d: %s", wantVAT, wPO.Code, wPO.Body.String())
		}
		var po models.PurchaseOrder
		if err := db.DB.Get(&po, "SELECT * FROM purchase_orders WHERE po_no = ?", poNo); err != nil {
			t.Fatalf("Query PO failed: %v", err)
		}
		if po.ApplyVAT != wantVAT {
			t.Errorf("Expected ApplyVAT=%v, got %v", wantVAT, po.ApplyVAT)
		}
	}
}
