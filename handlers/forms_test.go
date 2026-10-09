package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"radline/db"
	"radline/models"
)

func parseFormsTemplatesForTest() map[string]*template.Template {
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
			d := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				k, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				d[k] = values[i+1]
			}
			return d, nil
		},
	}

	pages := []string{"form_purchase_order.html", "form_quotation.html"}
	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"../templates/base.html", "../templates/" + page}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	return templates
}

func TestForms_PurchaseOrderAndSearch(t *testing.T) {
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	app := &App{
		Templates: parseFormsTemplatesForTest(),
	}

	// 1. Test GET /forms/purchase-order
	t.Run("GET purchase order form renders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/forms/purchase-order", nil)
		w := httptest.NewRecorder()
		app.PurchaseOrderFormHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Purchase Order Form") {
			t.Errorf("expected page title in body")
		}
		if !strings.Contains(body, "Save Purchase Order") {
			t.Errorf("expected Save button in body")
		}
		if strings.Contains(body, "Print / Save PDF") {
			t.Errorf("Print / Save PDF button should have been removed")
		}
	})

	// 2. Test GET /forms/quotation
	t.Run("GET quotation form renders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/forms/quotation", nil)
		w := httptest.NewRecorder()
		app.QuotationFormHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	// 3. Test SearchItemsHandler
	t.Run("GET /items/search with json format and limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/items/search?limit=5&format=json", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		app.SearchItemsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var items []models.Item
		if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
			t.Fatalf("failed to decode JSON response: %v", err)
		}
		if len(items) > 5 {
			t.Errorf("expected at most 5 items, got %d", len(items))
		}
	})

	t.Run("GET /items/search html dropdown fragment", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/items/search?limit=10", nil)
		w := httptest.NewRecorder()
		app.SearchItemsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		body := w.Body.String()
		if len(body) > 0 && !strings.Contains(body, "dropdown-item") {
			t.Errorf("expected dropdown-item in response, got %s", body)
		}
	})

	// 4. Test SavePurchaseOrderHandler
	t.Run("POST /forms/purchase-order/save valid submission", func(t *testing.T) {
		testPONo := fmt.Sprintf("TEST-PO-%d", time.Now().UnixNano())
		form := url.Values{}
		form.Set("po_no", testPONo)
		form.Set("date", "2026-10-09")
		form.Set("due_date", "2026-10-20")
		form.Set("vendor_name", "Test Supplier Supplies Co.")
		form.Set("vendor_tin", "987-654-321")
		form.Set("vendor_address", "123 Industrial Rd, Manila")
		form.Set("vendor_contact", "Jane Doe")
		form.Set("ship_to", "Radline Main Warehouse")
		form.Set("payment_terms", "Net 30 Days")
		form.Set("shipping_method", "Delivery Truck")
		form.Set("prepared_by", "Purchasing Test Officer")
		form.Set("freight", "150.00")
		form.Set("apply_vat", "true")
		form.Set("notes", "Handle with extreme care")

		form.Add("item_code", "ITM-PO-1")
		form.Add("description", "Heavy Duty Steel Bracket")
		form.Add("uom", "PCS")
		form.Add("qty", "10")
		form.Add("unit_cost", "250.00")

		form.Add("item_code", "ITM-PO-2")
		form.Add("description", "Industrial Bolt M10")
		form.Add("uom", "BOX")
		form.Add("qty", "5")
		form.Add("unit_cost", "120.00")

		req := httptest.NewRequest("POST", "/forms/purchase-order/save", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		app.SavePurchaseOrderHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		triggerHdr := w.Header().Get("HX-Trigger")
		if !strings.Contains(triggerHdr, "show-toast") || !strings.Contains(triggerHdr, "success") {
			t.Errorf("expected success show-toast in HX-Trigger, got %s", triggerHdr)
		}

		// Verify database persistence
		var po models.PurchaseOrder
		err := db.DB.Get(&po, "SELECT * FROM purchase_orders WHERE po_no = ?", testPONo)
		if err != nil {
			t.Fatalf("failed to query saved purchase order: %v", err)
		}

		if po.VendorName != "Test Supplier Supplies Co." {
			t.Errorf("expected vendor name %s, got %s", "Test Supplier Supplies Co.", po.VendorName)
		}
		// Subtotal: (10*250) + (5*120) = 2500 + 600 = 3100
		// Freight: 150
		// VAT: (3100 + 150) * 0.12 = 390
		// Grand total: 3100 + 150 + 390 = 3640
		if po.Subtotal != 3100 {
			t.Errorf("expected subtotal 3100, got %f", po.Subtotal)
		}
		if po.GrandTotal != 3640 {
			t.Errorf("expected grand total 3640, got %f", po.GrandTotal)
		}

		var items []models.PurchaseOrderItem
		err = db.DB.Select(&items, "SELECT * FROM purchase_order_items WHERE po_id = ? ORDER BY id ASC", po.ID)
		if err != nil {
			t.Fatalf("failed to query purchase order items: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
		if items[0].Description != "Heavy Duty Steel Bracket" || items[0].Qty != 10 || items[0].UnitCost != 250 {
			t.Errorf("item 0 details mismatch: %+v", items[0])
		}
	})

	t.Run("POST /forms/purchase-order/save empty vendor fails", func(t *testing.T) {
		form := url.Values{}
		form.Set("po_no", "PO-INVALID-1")
		form.Set("date", "2026-10-09")
		form.Set("vendor_name", "")
		form.Add("description", "Some Item")
		form.Add("qty", "1")

		req := httptest.NewRequest("POST", "/forms/purchase-order/save", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		app.SavePurchaseOrderHandler(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for empty vendor, got %d", w.Code)
		}
	})
}
