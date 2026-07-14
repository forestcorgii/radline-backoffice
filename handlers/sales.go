package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"radline/db"
	"radline/domain"
	"radline/models"
)

// SalesHandler lists all sales records with search, filter, and sort
func (app *App) SalesHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	supplierFilter := r.URL.Query().Get("supplier_filter")
	sort := r.URL.Query().Get("sort") // date_desc, date_asc, sales_desc, profit_desc

	query := `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost, s.profit,
		       i.code as item_code
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(s.doc_number LIKE ? OR s.customer_name LIKE ? OR i.code LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if supplierFilter != "" && supplierFilter != "all" {
		whereClauses = append(whereClauses, "s.supplier = ?")
		args = append(args, supplierFilter)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	switch sort {
	case "date_asc":
		query += " ORDER BY s.doc_date ASC, s.id ASC"
	case "sales_desc":
		query += " ORDER BY s.total_sales DESC"
	case "profit_desc":
		query += " ORDER BY s.profit DESC"
	default: // date_desc or empty
		query += " ORDER BY s.doc_date DESC, s.id DESC"
	}

	var sales []models.SalesDetailWithItem
	err := db.DB.Select(&sales, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		app.Render(w, "sales_rows.html", sales)
	} else {
		var items []models.Item
		_ = db.DB.Select(&items, "SELECT * FROM items ORDER BY code ASC")

		data := struct {
			Sales []models.SalesDetailWithItem
			Items []models.Item
		}{
			Sales: sales,
			Items: items,
		}
		app.RenderPage(w, r, "sales.html", data)
	}
}

// AddSalesHandler encodes a new sales record
func (app *App) AddSalesHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to parse form."}}`)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	docDateStr := r.FormValue("doc_date")
	docType := r.FormValue("doc_type")
	docNumber := r.FormValue("doc_number")
	customerName := r.FormValue("customer_name")
	supplier := r.FormValue("supplier")

	parsedDate, dateErr := time.Parse("2006-01-02", docDateStr)
	if dateErr != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid date format. Expected YYYY-MM-DD."}}`)
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	itemIDs := r.Form["item_id"]
	qtys := r.Form["qty"]
	uoms := r.Form["uom"]
	prices := r.Form["price"]
	costs := r.Form["cost"]

	if len(itemIDs) == 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "At least one item is required."}}`)
		http.Error(w, "At least one item is required", http.StatusBadRequest)
		return
	}

	if len(itemIDs) != len(qtys) || len(itemIDs) != len(uoms) || len(itemIDs) != len(prices) || len(itemIDs) != len(costs) {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Mismatch in item fields lengths."}}`)
		http.Error(w, "Mismatch in item fields lengths", http.StatusBadRequest)
		return
	}

	var saleItems []domain.SaleItem
	for i := range itemIDs {
		itemID, err1 := strconv.Atoi(itemIDs[i])
		qty, err2 := strconv.ParseFloat(qtys[i], 64)
		price, err3 := strconv.ParseFloat(prices[i], 64)
		cost, err4 := strconv.ParseFloat(costs[i], 64)
		uom := uoms[i]

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric input in items."}}`)
			http.Error(w, "Invalid numeric input in items", http.StatusBadRequest)
			return
		}

		saleItems = append(saleItems, domain.SaleItem{
			ItemID: itemID,
			Qty:    qty,
			UOM:    uom,
			Price:  price,
			Cost:   cost,
		})
	}

	sale, err := domain.NewSale(docType, "POSTED", parsedDate, docNumber, customerName, supplier, saleItems)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "`+err.Error()+`"}}`)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Database transaction failed."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	details := sale.ToSalesDetails()
	for _, sd := range details {
		_, err = tx.Exec(`
			INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, sd.DocType, sd.DocStatus, sd.DocDate, sd.DocNumber, sd.CustomerName, sd.Supplier, sd.ItemID, sd.Qty, sd.UOM, sd.Price, sd.TotalSales, sd.Cost, sd.TotalCost, sd.Profit)
		if err != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to save sale detail."}}`)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to commit sale transaction."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Sales logged successfully!"}}`)
	w.Header().Set("HX-Location", "/sales")
	w.WriteHeader(http.StatusOK)
}

// NewSalesPageHandler renders the standalone sales encoding page
func (app *App) NewSalesPageHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY code ASC")

	data := struct {
		Items        []models.Item
		SalesRowData interface{}
	}{
		Items: items,
		SalesRowData: map[string]interface{}{
			"Items": items,
		},
	}

	app.RenderPage(w, r, "sales_new.html", data)
}

// NewSaleRowHandler renders a single empty sale item row template
func (app *App) NewSaleRowHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY code ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.Render(w, "sale_item_row.html", map[string]interface{}{
		"Items": items,
	})
}

// DeleteSalesHandler removes a sales record
func (app *App) DeleteSalesHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid sales record ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM sales_details WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to delete sales record."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Sales record deleted successfully!"}}`)
	w.WriteHeader(http.StatusOK)
}

// SaleItemRowDetailsHandler renders a single sale item row template populated with item defaults
func (app *App) SaleItemRowDetailsHandler(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.URL.Query().Get("item_id")
	itemID, _ := strconv.Atoi(itemIDStr)

	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY code ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var defaultUOM string
	var lastCost, lastPrice float64

	if itemID > 0 {
		_ = db.DB.Get(&defaultUOM, "SELECT default_uom FROM items WHERE id = ?", itemID)

		// Get last receiving cost and price
		var lastRec models.ReceivingLog
		err := db.DB.Get(&lastRec, "SELECT cost, selling_price FROM receiving_logs WHERE item_id = ? ORDER BY date DESC, id DESC LIMIT 1", itemID)
		if err == nil {
			lastCost = lastRec.Cost
			lastPrice = lastRec.SellingPrice
		}
	}

	app.Render(w, "sale_item_row.html", map[string]interface{}{
		"Items":          items,
		"SelectedItemID": itemID,
		"DefaultUOM":     defaultUOM,
		"Cost":           lastCost,
		"Price":          lastPrice,
	})
}
