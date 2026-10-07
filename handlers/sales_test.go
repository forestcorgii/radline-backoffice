package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"radline/db"
	"radline/models"
)

func parseSalesTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	funcMap := template.FuncMap{
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}

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

func TestSalesSearchFragmentHandler(t *testing.T) {
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

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
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	app := &App{
		Templates: parseSalesTemplatesForTest(),
	}

	var items []models.Item
	err = db.DB.Select(&items, "SELECT id, description FROM items LIMIT 5")
	if err != nil || len(items) == 0 {
		t.Logf("No items in DB: %v", err)
		return
	}

	it := items[0]
	req := httptest.NewRequest("GET", "/sales/item-row-details?item_id="+strconv.Itoa(it.ID), nil)
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
}

func TestSaleRowDeductionsDisplay(t *testing.T) {
	app := &App{
		Templates: parseSalesTemplatesForTest(),
	}

	// Case 1: All deductions zero -> renders dash
	saleZero := models.SalesDetailWithItem{
		ID:        1,
		DocStatus: "Active",
		DocDate:   time.Now(),
		Patong:    0,
		POSCharge: 0,
		WT2307:    0,
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
		ID:        2,
		DocStatus: "Active",
		DocDate:   time.Now(),
		Patong:    50.00,
		POSCharge: 0,
		WT2307:    0,
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
		ID:        3,
		DocStatus: "Active",
		DocDate:   time.Now(),
		Patong:    0,
		POSCharge: 12.50,
		WT2307:    5.25,
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

