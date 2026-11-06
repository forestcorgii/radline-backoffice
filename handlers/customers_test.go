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

	"rav/db"
	"rav/models"
)

func parseCustomerTemplatesForTest() map[string]*template.Template {
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

	pages := []string{"customers.html"}
	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{
			"../templates/base.html", "../templates/" + page,
			"../templates/customers_results.html", "../templates/customer_rows.html",
			"../templates/customer_row.html", "../templates/customer_edit_row.html",
		}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	fragments := []string{"customers_results.html", "customer_rows.html", "customer_row.html", "customer_edit_row.html"}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcMap)
		files := []string{"../templates/" + frag}
		if frag == "customers_results.html" {
			files = append(files, "../templates/customer_rows.html", "../templates/customer_row.html")
		} else if frag == "customer_rows.html" {
			files = append(files, "../templates/customer_row.html")
		}
		t = template.Must(t.ParseFiles(files...))
		templates[frag] = t
	}

	return templates
}

func setupTestCustomerDB(t *testing.T) *App {
	testDB := fmt.Sprintf("file:test_customers_%d?mode=memory&cache=shared", time.Now().UnixNano())
	if err := db.InitDB(testDB); err != nil {
		t.Fatalf("failed to init in-memory db: %v", err)
	}
	_ = db.EnsureCustomersTable()

	tmpl := parseCustomerTemplatesForTest()
	return &App{Templates: tmpl}
}

func TestCustomerHandlers(t *testing.T) {
	app := setupTestCustomerDB(t)

	// 1. Test AddCustomerHandler
	t.Run("POST /customers/add creates a customer", func(t *testing.T) {
		form := url.Values{}
		form.Set("code", "TEST-CUST-1")
		form.Set("name", "Test Customer Corp")
		form.Set("tin_no", "111-222-333-000")
		form.Set("address", "123 Test Street, Manila")
		form.Set("contact_person", "Maria Santos")
		form.Set("phone", "0917-111-2222")
		form.Set("email", "maria@test.com")
		form.Set("payment_terms", "Net 15 Days")

		req := httptest.NewRequest("POST", "/customers/add", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		app.AddCustomerHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		trigger := w.Header().Get("HX-Trigger")
		if !strings.Contains(trigger, "customer-added") {
			t.Errorf("expected customer-added in HX-Trigger, got: %s", trigger)
		}
	})

	// 2. Test CustomersHandler (Full page and HTMX)
	t.Run("GET /customers lists customers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/customers", nil)
		w := httptest.NewRecorder()

		app.CustomersHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Test Customer Corp") {
			t.Errorf("expected customer name in page, got: %s", body)
		}
	})

	t.Run("GET /customers with HX-Request returns partial", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/customers?search=Test", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", "customers-results")
		w := httptest.NewRecorder()

		app.CustomersHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Test Customer Corp") {
			t.Errorf("expected customer name in partial, got: %s", body)
		}
		if strings.Contains(body, "<!DOCTYPE html>") {
			t.Errorf("did not expect base.html shell in partial response")
		}
	})

	// 3. Test SearchCustomersHandler (JSON & HTML)
	t.Run("GET /customers/search with format=json", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/customers/search?q=Test&format=json", nil)
		w := httptest.NewRecorder()

		app.SearchCustomersHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var res []models.Customer
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if len(res) == 0 {
			t.Fatalf("expected at least 1 customer in search results, got 0")
		}
		if res[0].Name != "Test Customer Corp" {
			t.Errorf("expected customer name Test Customer Corp, got %s", res[0].Name)
		}
		if res[0].TINNo != "111-222-333-000" {
			t.Errorf("expected TIN 111-222-333-000, got %s", res[0].TINNo)
		}
	})

	t.Run("GET /customers/search returns HTML dropdown items", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/customers/search?q=Test", nil)
		w := httptest.NewRecorder()

		app.SearchCustomersHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "dropdown-item") || !strings.Contains(body, "Test Customer Corp") {
			t.Errorf("expected dropdown-item HTML for Test Customer Corp, got: %s", body)
		}
	})

	// 4. Test Edit and Update Customer
	t.Run("POST /customers/edit/{id} updates customer", func(t *testing.T) {
		var cust models.Customer
		err := db.DB.Get(&cust, "SELECT id FROM customers WHERE code = 'TEST-CUST-1'")
		if err != nil {
			t.Fatalf("failed to query test customer: %v", err)
		}

		form := url.Values{}
		form.Set("code", "TEST-CUST-1")
		form.Set("name", "Test Customer Corp Updated")
		form.Set("tin_no", "999-888-777-000")
		form.Set("address", "Updated Address")
		form.Set("payment_terms", "Net 30 Days")

		req := httptest.NewRequest("POST", fmt.Sprintf("/customers/edit/%d", cust.ID), strings.NewReader(form.Encode()))
		req.SetPathValue("id", fmt.Sprintf("%d", cust.ID))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		app.UpdateCustomerHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		body := w.Body.String()
		if !strings.Contains(body, "Test Customer Corp Updated") {
			t.Errorf("expected updated name in rendered row, got: %s", body)
		}
	})

	// 5. Test DeleteCustomerHandler
	t.Run("DELETE /customers/delete/{id} deletes customer", func(t *testing.T) {
		var cust models.Customer
		err := db.DB.Get(&cust, "SELECT id FROM customers WHERE code = 'TEST-CUST-1'")
		if err != nil {
			t.Fatalf("failed to query test customer: %v", err)
		}

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/customers/delete/%d", cust.ID), nil)
		req.SetPathValue("id", fmt.Sprintf("%d", cust.ID))
		w := httptest.NewRecorder()

		app.DeleteCustomerHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var count int
		_ = db.DB.Get(&count, "SELECT COUNT(*) FROM customers WHERE id = ?", cust.ID)
		if count != 0 {
			t.Errorf("expected customer to be deleted, count = %d", count)
		}
	})
}
