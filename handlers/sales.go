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
	docTypeFilter := r.URL.Query().Get("doc_type_filter")
	supplierFilter := r.URL.Query().Get("supplier_filter")
	sort := r.URL.Query().Get("sort") // date_desc, date_asc, sales_desc, profit_desc
	isEdit := r.FormValue("is_edit") == "1" || r.URL.Query().Get("is_edit") == "1"

	baseQuery := `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost, s.profit,
		       i.code as item_code, i.description as item_description
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(s.doc_number LIKE ? OR s.customer_name LIKE ? OR i.code LIKE ? OR i.description LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if supplierFilter != "" && supplierFilter != "all" {
		whereClauses = append(whereClauses, "s.supplier = ?")
		args = append(args, supplierFilter)
	}

	if docTypeFilter != "" && docTypeFilter != "all" {
		whereClauses = append(whereClauses, "s.doc_type = ?")
		args = append(args, docTypeFilter)
	}

	if len(whereClauses) > 0 {
		baseQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	limit := GetLimitParam(r)

	query := baseQuery
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
	query += " LIMIT ?"
	selectArgs := append(args, limit)

	var sales []models.SalesDetailWithItem
	err := db.DB.Select(&sales, query, selectArgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			Sales  []models.SalesDetailWithItem
			IsEdit bool
			Items  []models.Item
			Uoms   []models.Uom
		}{
			Sales:  sales,
			IsEdit: isEdit,
			Items:  items,
			Uoms:   uoms,
		}
		app.Render(w, "sales_results.html", data)
	} else {
		data := struct {
			Sales        []models.SalesDetailWithItem
			IsEdit       bool
			Items        []models.Item
			Uoms         []models.Uom
			SalesRowData interface{}
		}{
			Sales:  sales,
			IsEdit: isEdit,
			Items:  items,
			Uoms:   uoms,
			SalesRowData: map[string]interface{}{
				"Items": items,
				"Uoms":  uoms,
			},
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
	refPLs := r.Form["ref_pl"]

	if len(itemIDs) == 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "At least one item is required."}}`)
		http.Error(w, "At least one item is required", http.StatusBadRequest)
		return
	}

	if len(itemIDs) != len(qtys) || len(itemIDs) != len(uoms) || len(itemIDs) != len(prices) || len(itemIDs) != len(costs) || len(itemIDs) != len(refPLs) {
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
		refPL := refPLs[i]

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
			RefPL:  refPL,
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
			INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit, ref_pl)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, sd.DocType, sd.DocStatus, sd.DocDate, sd.DocNumber, sd.CustomerName, sd.Supplier, sd.ItemID, sd.Qty, sd.UOM, sd.Price, sd.TotalSales, sd.Cost, sd.TotalCost, sd.Profit, sd.RefPL)
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

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Sales logged successfully!"}, "sales-added": ""}`)
	w.WriteHeader(http.StatusOK)
}

// NewSalesPageHandler redirects to sales list page
func (app *App) NewSalesPageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/sales", http.StatusSeeOther)
}

// NewSaleRowHandler renders a single empty sale item row template
func (app *App) NewSaleRowHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	app.Render(w, "sale_item_row.html", map[string]interface{}{
		"Items": items,
		"Uoms":  uoms,
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
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var defaultUOM string
	var lastCost, lastPrice float64
	var lastPLNo string

	if itemID > 0 {
		_ = db.DB.Get(&defaultUOM, "SELECT default_uom FROM items WHERE id = ?", itemID)

		// Get oldest PL with stock available using a single efficient SQL query
		// This replaces the old approach of loading ALL transactions into memory via FetchItemStock
		type CostPriceResult struct {
			Cost         float64 `db:"cost"`
			SellingPrice float64 `db:"selling_price"`
			PLNo         string  `db:"pl_no"`
		}
		var result CostPriceResult
		err := db.DB.Get(&result, `
			WITH item_stats AS (
				SELECT
					COALESCE((SELECT SUM(r.qty) FROM receiving_logs r WHERE r.item_id = ?), 0) AS total_received,
					COALESCE((SELECT SUM(s.qty) FROM sales_details s WHERE s.item_id = ?), 0) AS total_sold,
					COALESCE((SELECT SUM(a.adjustment_qty) FROM inventory_adjustments a WHERE a.item_id = ?), 0) AS total_adjusted
			),
			oh AS (SELECT (total_received - total_sold + total_adjusted) AS on_hand FROM item_stats)
			SELECT r.cost, r.selling_price, r.pl_no
			FROM receiving_logs r, oh
			WHERE r.item_id = ? AND oh.on_hand > 0
			  AND (SELECT COALESCE(SUM(r2.qty), 0) FROM receiving_logs r2
			       WHERE r2.item_id = ? AND (r2.date < r.date OR (r2.date = r.date AND r2.id < r.id)))
			      < (SELECT COALESCE(SUM(s.qty), 0) FROM sales_details s WHERE s.item_id = ?)
			ORDER BY r.date ASC, r.id ASC
			LIMIT 1
		`, itemID, itemID, itemID, itemID, itemID, itemID)
		if err == nil {
			lastCost = result.Cost
			lastPrice = result.SellingPrice
			lastPLNo = result.PLNo
		} else {
			// Fallback to last receiving log
			var lastRec models.ReceivingLog
			err := db.DB.Get(&lastRec, "SELECT cost, selling_price, pl_no FROM receiving_logs WHERE item_id = ? ORDER BY date DESC, id DESC LIMIT 1", itemID)
			if err == nil {
				lastCost = lastRec.Cost
				lastPrice = lastRec.SellingPrice
				lastPLNo = lastRec.PLNo
			}
		}
	}

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	app.Render(w, "sale_item_row.html", map[string]interface{}{
		"Items":          items,
		"SelectedItemID": itemID,
		"DefaultUOM":     defaultUOM,
		"Cost":           lastCost,
		"Price":          lastPrice,
		"PLNo":           lastPLNo,
		"Uoms":           uoms,
	})
}

// UpdateSalesHandler updates an existing sales detail record
func (app *App) UpdateSalesHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid sale log ID."}}`)
		http.Error(w, "Invalid sale log ID", http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
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
	itemIDStr := r.FormValue("item_id")
	qtyStr := r.FormValue("qty")
	uom := r.FormValue("uom")
	priceStr := r.FormValue("price")
	costStr := r.FormValue("cost")

	itemID, errItem := strconv.Atoi(itemIDStr)
	qty, errQty := strconv.ParseFloat(qtyStr, 64)
	price, errPrice := strconv.ParseFloat(priceStr, 64)
	cost, errCost := strconv.ParseFloat(costStr, 64)

	parsedDate, dateErr := time.Parse("2006-01-02", docDateStr)
	if dateErr != nil || errItem != nil || errQty != nil || errPrice != nil || errCost != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric or date input."}}`)
		http.Error(w, "Invalid numeric or date input", http.StatusBadRequest)
		return
	}

	salesDetail, err := domain.NewSalesDetail(id, docType, "POSTED", parsedDate, docNumber, customerName, supplier, itemID, qty, uom, price, cost, "")
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "` + err.Error() + `"}}`)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE sales_details 
		SET doc_type = ?, doc_date = ?, doc_number = ?, customer_name = ?, supplier = ?, item_id = ?, qty = ?, uom = ?, price = ?, total_sales = ?, cost = ?, total_cost = ?, profit = ?
		WHERE id = ?
	`, salesDetail.DocType, salesDetail.DocDate, salesDetail.DocNumber, salesDetail.CustomerName, salesDetail.Supplier, salesDetail.ItemID, salesDetail.Qty, salesDetail.UOM, salesDetail.Price, salesDetail.TotalSales, salesDetail.Cost, salesDetail.TotalCost, salesDetail.Profit, id)

	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update sale log."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var updatedSale models.SalesDetailWithItem
	err = db.DB.Get(&updatedSale, `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost, s.profit,
		       i.code as item_code, i.description as item_description
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
		WHERE s.id = ?
	`, id)

	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to retrieve updated sale record."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Sale log updated successfully!"}}`)

	isEdit := r.FormValue("is_edit") == "1"
	if isEdit {
		var items []models.Item
		_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
		var uoms []models.Uom
		_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

		app.Render(w, "sale_edit_row.html", map[string]interface{}{
			"Sale":  updatedSale,
			"Items": items,
			"Uoms":  uoms,
		})
	} else {
		app.Render(w, "sale_row.html", updatedSale)
	}
}
