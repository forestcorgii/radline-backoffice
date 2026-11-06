package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"rav/db"
	"rav/domain"
	"rav/models"
)

// CustomersHandler lists customers with search, filter, and sorting
func (app *App) CustomersHandler(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	filter := r.URL.Query().Get("filter") // all, with_sales, no_sales
	sort := r.URL.Query().Get("sort")     // name_asc, name_desc, code_asc, code_desc, sales_desc

	query := `
		SELECT 
			c.id, c.code, c.name, c.tin_no, c.address, c.contact_person, 
			c.phone, c.email, c.payment_terms, c.notes, c.created_at,
			COALESCE(COUNT(s.id), 0) AS sales_count,
			COALESCE(SUM(s.total_sales), 0.0) AS total_sales
		FROM customers c
		LEFT JOIN sales_details s ON c.name = s.customer_name
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(c.code LIKE ? OR c.name LIKE ? OR c.tin_no LIKE ? OR c.address LIKE ? OR c.contact_person LIKE ?)")
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " GROUP BY c.id, c.code, c.name, c.tin_no, c.address, c.contact_person, c.phone, c.email, c.payment_terms, c.notes, c.created_at"

	if filter == "with_sales" {
		query += " HAVING COUNT(s.id) > 0"
	} else if filter == "no_sales" {
		query += " HAVING COUNT(s.id) = 0"
	}

	limit := GetLimitParam(r)

	switch sort {
	case "name_desc":
		query += " ORDER BY c.name DESC"
	case "code_asc":
		query += " ORDER BY c.code ASC"
	case "code_desc":
		query += " ORDER BY c.code DESC"
	case "sales_desc":
		query += " ORDER BY total_sales DESC, c.name ASC"
	default: // name_asc or empty
		query += " ORDER BY c.name ASC"
	}

	query += " LIMIT ?"
	selectArgs := append(args, limit)

	var customers []models.CustomerWithStats
	err := db.DB.Select(&customers, query, selectArgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Customers []models.CustomerWithStats
	}{
		Customers: customers,
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		app.Render(w, "customers_results.html", data)
	} else {
		app.RenderPage(w, r, "customers.html", data)
	}
}

// AddCustomerHandler handles adding a customer
func (app *App) AddCustomerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))
	name := strings.TrimSpace(r.FormValue("name"))
	tinNo := strings.TrimSpace(r.FormValue("tin_no"))
	address := strings.TrimSpace(r.FormValue("address"))
	contactPerson := strings.TrimSpace(r.FormValue("contact_person"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	email := strings.TrimSpace(r.FormValue("email"))
	paymentTerms := strings.TrimSpace(r.FormValue("payment_terms"))
	notes := strings.TrimSpace(r.FormValue("notes"))

	// Auto-generate code if empty
	if code == "" {
		var maxID int
		_ = db.DB.Get(&maxID, "SELECT COALESCE(MAX(id), 0) FROM customers")
		code = fmt.Sprintf("CUST-%04d", maxID+1)
	}

	domCust, err := domain.NewCustomer(0, code, name, tinNo, address, contactPerson, phone, email, paymentTerms, notes)
	if err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	insertQuery := `
		INSERT INTO customers (code, name, tin_no, address, contact_person, phone, email, payment_terms, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.DB.Exec(insertQuery, domCust.Code, domCust.Name, domCust.TINNo, domCust.Address, domCust.ContactPerson, domCust.Phone, domCust.Email, domCust.PaymentTerms, domCust.Notes)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to add customer. Code or Name might already exist."}}`)
		http.Error(w, "Failed to add customer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Customer added successfully."}, "customer-added": {}}`)
	w.WriteHeader(http.StatusOK)
}

// EditCustomerFormHandler returns the inline edit row
func (app *App) EditCustomerFormHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var cust models.Customer
	err := db.DB.Get(&cust, "SELECT id, code, name, tin_no, address, contact_person, phone, email, payment_terms, notes FROM customers WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	app.Render(w, "customer_edit_row.html", cust)
}

// UpdateCustomerHandler updates a customer inline
func (app *App) UpdateCustomerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	code := strings.TrimSpace(r.FormValue("code"))
	name := strings.TrimSpace(r.FormValue("name"))
	tinNo := strings.TrimSpace(r.FormValue("tin_no"))
	address := strings.TrimSpace(r.FormValue("address"))
	contactPerson := strings.TrimSpace(r.FormValue("contact_person"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	email := strings.TrimSpace(r.FormValue("email"))
	paymentTerms := strings.TrimSpace(r.FormValue("payment_terms"))
	notes := strings.TrimSpace(r.FormValue("notes"))

	domCust, err := domain.NewCustomer(id, code, name, tinNo, address, contactPerson, phone, email, paymentTerms, notes)
	if err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updateQuery := `
		UPDATE customers 
		SET code = ?, name = ?, tin_no = ?, address = ?, contact_person = ?, phone = ?, email = ?, payment_terms = ?, notes = ?
		WHERE id = ?
	`
	_, err = db.DB.Exec(updateQuery, domCust.Code, domCust.Name, domCust.TINNo, domCust.Address, domCust.ContactPerson, domCust.Phone, domCust.Email, domCust.PaymentTerms, domCust.Notes, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update customer."}}`)
		http.Error(w, "Failed to update customer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Customer updated successfully."}}`)

	// Return updated row
	var cust models.CustomerWithStats
	query := `
		SELECT 
			c.id, c.code, c.name, c.tin_no, c.address, c.contact_person, 
			c.phone, c.email, c.payment_terms, c.notes, c.created_at,
			COALESCE(COUNT(s.id), 0) AS sales_count,
			COALESCE(SUM(s.total_sales), 0.0) AS total_sales
		FROM customers c
		LEFT JOIN sales_details s ON c.name = s.customer_name
		WHERE c.id = ?
		GROUP BY c.id, c.code, c.name, c.tin_no, c.address, c.contact_person, c.phone, c.email, c.payment_terms, c.notes, c.created_at
	`
	_ = db.DB.Get(&cust, query, id)
	app.Render(w, "customer_row.html", cust)
}

// CustomerRowHandler renders a single customer row
func (app *App) CustomerRowHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var cust models.CustomerWithStats
	query := `
		SELECT 
			c.id, c.code, c.name, c.tin_no, c.address, c.contact_person, 
			c.phone, c.email, c.payment_terms, c.notes, c.created_at,
			COALESCE(COUNT(s.id), 0) AS sales_count,
			COALESCE(SUM(s.total_sales), 0.0) AS total_sales
		FROM customers c
		LEFT JOIN sales_details s ON c.name = s.customer_name
		WHERE c.id = ?
		GROUP BY c.id, c.code, c.name, c.tin_no, c.address, c.contact_person, c.phone, c.email, c.payment_terms, c.notes, c.created_at
	`
	err := db.DB.Get(&cust, query, id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}
	app.Render(w, "customer_row.html", cust)
}

// DeleteCustomerHandler deletes a customer
func (app *App) DeleteCustomerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	_, err := db.DB.Exec("DELETE FROM customers WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to delete customer."}}`)
		http.Error(w, "Failed to delete customer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Customer deleted successfully."}}`)
	w.WriteHeader(http.StatusOK)
}

// SearchCustomersHandler searches customers for fast debounced autocomplete in forms & sales
func (app *App) SearchCustomersHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if limit > 100 {
		limit = 100
	}

	var customers []models.Customer
	var err error

	if q != "" {
		words := strings.Fields(q)
		var clauses []string
		var args []interface{}
		for _, word := range words {
			term := "%" + word + "%"
			clauses = append(clauses, "(code LIKE ? OR name LIKE ? OR tin_no LIKE ? OR contact_person LIKE ?)")
			args = append(args, term, term, term, term)
		}
		query := "SELECT id, code, name, tin_no, address, contact_person, phone, email, payment_terms, notes FROM customers WHERE " + strings.Join(clauses, " AND ") + " ORDER BY name ASC LIMIT ?"
		args = append(args, limit)
		err = db.DB.Select(&customers, query, args...)
	} else {
		query := "SELECT id, code, name, tin_no, address, contact_person, phone, email, payment_terms, notes FROM customers ORDER BY name ASC LIMIT ?"
		err = db.DB.Select(&customers, query, limit)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.Contains(r.Header.Get("Accept"), "application/json") || r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(customers)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, c := range customers {
		fmt.Fprintf(w, `<div class="dropdown-item" data-id="%d" data-code="%s" data-name="%s" data-tin="%s" data-address="%s" data-contact="%s" data-phone="%s" data-terms="%s" onmousedown="selectCustomerItem(this)"><span style="font-weight:600;">%s</span> <span style="font-size:0.75rem; color:var(--text-muted);">(%s)</span></div>`,
			c.ID, template.HTMLEscapeString(c.Code), template.HTMLEscapeString(c.Name), template.HTMLEscapeString(c.TINNo), template.HTMLEscapeString(c.Address), template.HTMLEscapeString(c.ContactPerson), template.HTMLEscapeString(c.Phone), template.HTMLEscapeString(c.PaymentTerms),
			template.HTMLEscapeString(c.Name), template.HTMLEscapeString(c.Code))
	}
}
