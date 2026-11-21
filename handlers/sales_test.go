package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"rav/db"
	"rav/models"
)

func parseSalesTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	funcMap := TemplateFuncMap()

	pages := []string{"sales.html"}
	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"../templates/base.html", "../templates/" + page, "../templates/sales_rows.html", "../templates/sale_item_row.html", "../templates/sales_results.html", "../templates/item_select.html", "../templates/uom_select.html", "../templates/sale_row.html", "../templates/sale_edit_row.html"}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	fragments := []string{"sales_results.html", "sales_rows.html", "sale_row.html", "sale_edit_row.html", "sale_item_row.html"}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcMap)
		files := []string{"../templates/" + frag}
		if frag == "sales_results.html" {
			files = append(files, "../templates/sales_rows.html", "../templates/sale_row.html", "../templates/sale_edit_row.html")
		} else if frag == "sales_rows.html" {
			files = append(files, "../templates/sale_row.html", "../templates/sale_edit_row.html")
		}
		t = template.Must(t.ParseFiles(files...))
		templates[frag] = t
	}

	return templates
}

func seedSalesTestFixture(t *testing.T) int {
	t.Helper()
	if _, err := db.DB.Exec("INSERT INTO uoms (code) VALUES ('PCS'), ('BOX') ON CONFLICT DO NOTHING"); err != nil {
		t.Fatalf("Failed to seed uoms: %v", err)
	}
	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('SALE-01', 'Test Sale Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("Failed to seed item: %v", err)
	}
	if _, err := db.DB.Exec("INSERT INTO uom_settings (item_id, muom, conversion_factor) VALUES (?, 'BOX', 10)", itemID); err != nil {
		t.Fatalf("Failed to seed uom_settings: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, cost, total_cost, markup, selling_price)
		VALUES ('ASCD', '2026-01-05', 'PL-SALE-1', ?, 50, 'PCS', 100.0, 100.0, 5000.0, 130.0, 130.0)
	`, itemID); err != nil {
		t.Fatalf("Failed to seed receiving_log: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, tin_no, address, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, patong, pos_charge, wt_2307, total_remit, profit, profit_margin, remarks)
		VALUES ('SI', 'Posted', '2026-01-10', 'SI-TEST-1', 'Test Customer', '123-456', 'Manila', 'ASCD', ?, 2, 'PCS', 130.0, 260.0, 100.0, 200.0, 0, 0, 0, 260.0, 60.0, 23.08, '')
	`, itemID); err != nil {
		t.Fatalf("Failed to seed sales_detail: %v", err)
	}
	return itemID
}

func TestSalesSearchFragmentHandler(t *testing.T) {
	setupIsolatedTestDB(t)
	seedSalesTestFixture(t)

	app := &App{
		Templates: parseSalesTemplatesForTest(),
	}

	// 1. Fragment request via HTMX search (Normal mode)
	req := httptest.NewRequest("GET", "/sales?search=test", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "sales-results")
	w := httptest.NewRecorder()

	app.SalesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for sales search fragment, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. Fragment request via HTMX search (Edit mode)
	reqEdit := httptest.NewRequest("GET", "/sales?search=test&is_edit=1", nil)
	reqEdit.Header.Set("HX-Request", "true")
	reqEdit.Header.Set("HX-Target", "sales-results")
	wEdit := httptest.NewRecorder()

	app.SalesHandler(wEdit, reqEdit)

	if wEdit.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for sales search fragment in edit mode, got %d. Body: %s", wEdit.Code, wEdit.Body.String())
	}
}

func TestSaleItemRowDetailsHandler(t *testing.T) {
	setupIsolatedTestDB(t)
	itemID := seedSalesTestFixture(t)

	app := &App{
		Templates: parseSalesTemplatesForTest(),
	}

	req := httptest.NewRequest("GET", "/sales/item-row-details?item_id="+strconv.Itoa(itemID), nil)
	w := httptest.NewRecorder()
	app.SaleItemRowDetailsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "name=\"ref_pl\"") {
		t.Errorf("Response body missing ref_pl field")
	}
	if !strings.Contains(body, "name=\"uom\"") {
		t.Errorf("Response body missing uom field")
	}
	if !strings.Contains(body, "BOX") {
		t.Errorf("Response body missing UOM option BOX from uom_settings")
	}
}

func TestSaleRowDeductionsDisplay(t *testing.T) {
	app := &App{
		Templates: parseSalesTemplatesForTest(),
	}

	// Case 1: All deductions zero -> renders dash
	saleZero := models.SalesDetailWithItem{
		SalesDetail: models.SalesDetail{
			ID:        1,
			DocStatus: "Active",
			DocDate:   time.Now(),
			Patong:    0,
			POSCharge: 0,
			WT2307:    0,
		},
	}
	w1 := httptest.NewRecorder()
	app.Render(w1, "sale_row.html", saleZero)
	body1 := w1.Body.String()
	if strings.Contains(body1, "P: ₱") || strings.Contains(body1, "C: ₱") || strings.Contains(body1, "W: ₱") {
		t.Errorf("Expected no deduction labels for zero deductions, got: %s", body1)
	}
	if !strings.Contains(body1, "—") {
		t.Errorf("Expected dash for zero deductions, got: %s", body1)
	}

	// Case 2: Only Patong non-zero -> renders only P: ₱50.00
	salePatongOnly := models.SalesDetailWithItem{
		SalesDetail: models.SalesDetail{
			ID:        2,
			DocStatus: "Active",
			DocDate:   time.Now(),
			Patong:    50.00,
			POSCharge: 0,
			WT2307:    0,
		},
	}
	w2 := httptest.NewRecorder()
	app.Render(w2, "sale_row.html", salePatongOnly)
	body2 := w2.Body.String()
	if !strings.Contains(body2, "P: ₱50.00") {
		t.Errorf("Expected 'P: ₱50.00', got: %s", body2)
	}
	if strings.Contains(body2, "C: ₱") || strings.Contains(body2, "W: ₱") {
		t.Errorf("Expected only Patong, but found other deductions: %s", body2)
	}

	// Case 3: Only POS Charge and WT2307 non-zero -> renders C: ₱12.50 and W: ₱5.25
	saleOther := models.SalesDetailWithItem{
		SalesDetail: models.SalesDetail{
			ID:        3,
			DocStatus: "Active",
			DocDate:   time.Now(),
			Patong:    0,
			POSCharge: 12.50,
			WT2307:    5.25,
		},
	}
	w3 := httptest.NewRecorder()
	app.Render(w3, "sale_row.html", saleOther)
	body3 := w3.Body.String()
	if strings.Contains(body3, "P: ₱") {
		t.Errorf("Expected no Patong label, got: %s", body3)
	}
	if !strings.Contains(body3, "C: ₱12.50") {
		t.Errorf("Expected 'C: ₱12.50', got: %s", body3)
	}
	if !strings.Contains(body3, "W: ₱5.25") {
		t.Errorf("Expected 'W: ₱5.25', got: %s", body3)
	}
}

func TestSalesPageNavigation(t *testing.T) {
	setupIsolatedTestDB(t)
	seedSalesTestFixture(t)

	app := &App{
		Templates: parseSalesTemplatesForTest(),
	}

	// 1. Full page GET /sales
	req := httptest.NewRequest("GET", "/sales", nil)
	w := httptest.NewRecorder()
	app.SalesHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for full page GET /sales, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. HTMX SPA navigation GET /sales with HX-Target: main-content
	reqSPA := httptest.NewRequest("GET", "/sales", nil)
	reqSPA.Header.Set("HX-Request", "true")
	reqSPA.Header.Set("HX-Target", "main-content")
	wSPA := httptest.NewRecorder()
	app.SalesHandler(wSPA, reqSPA)
	if wSPA.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for SPA navigation GET /sales, got %d. Body: %s", wSPA.Code, wSPA.Body.String())
	}
}


