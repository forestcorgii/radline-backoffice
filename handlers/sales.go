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
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	datePreset := r.URL.Query().Get("date_preset")
	sort := r.URL.Query().Get("sort") // date_desc, date_asc, sales_desc, profit_desc
	isEdit := r.FormValue("is_edit") == "1" || r.URL.Query().Get("is_edit") == "1"

	baseQuery := `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost,
		       s.patong, s.pos_charge, s.wt_2307, s.total_remit, s.profit, s.profit_margin, s.remarks,
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

	if startDateStr != "" {
		if parsedStart, err := time.Parse("2006-01-02", startDateStr); err == nil {
			whereClauses = append(whereClauses, "s.doc_date >= ?")
			args = append(args, parsedStart)
		}
	}

	if endDateStr != "" {
		if parsedEnd, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Query up to the end of the specified day
			endOfDay := parsedEnd.Add(24 * time.Hour).Add(-time.Second)
			whereClauses = append(whereClauses, "s.doc_date <= ?")
			args = append(args, endOfDay)
		}
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
			Sales      []models.SalesDetailWithItem
			IsEdit     bool
			Items      []models.Item
			Uoms       []models.Uom
			StartDate  string
			EndDate    string
			DatePreset string
		}{
			Sales:      sales,
			IsEdit:     isEdit,
			Items:      items,
			Uoms:       uoms,
			StartDate:  startDateStr,
			EndDate:    endDateStr,
			DatePreset: datePreset,
		}
		app.Render(w, "sales_results.html", data)
	} else {
		data := struct {
			Sales          []models.SalesDetailWithItem
			IsEdit         bool
			Items          []models.Item
			Uoms           []models.Uom
			SalesRowData   interface{}
			StartDate      string
			EndDate        string
			DatePreset     string
			Search         string
			DocTypeFilter  string
			SupplierFilter string
			Sort           string
			Limit          int
		}{
			Sales:  sales,
			IsEdit: isEdit,
			Items:  items,
			Uoms:   uoms,
			SalesRowData: map[string]interface{}{
				"Items": items,
				"Uoms":  uoms,
			},
			StartDate:      startDateStr,
			EndDate:        endDateStr,
			DatePreset:     datePreset,
			Search:         search,
			DocTypeFilter:  docTypeFilter,
			SupplierFilter: supplierFilter,
			Sort:           sort,
			Limit:          limit,
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
	docStatus := r.FormValue("doc_status")
	if docStatus == "" {
		docStatus = "Active"
	}

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
	patongs := r.Form["patong"]
	posCharges := r.Form["pos_charge"]
	wt2307s := r.Form["wt_2307"]
	remarkss := r.Form["remarks"]
	refPLs := r.Form["ref_pl"]

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
		
		var refPL string
		if i < len(refPLs) {
			refPL = refPLs[i]
		}

		var patong, posCharge, wt2307 float64
		var remarks string
		if i < len(patongs) && patongs[i] != "" {
			patong, _ = strconv.ParseFloat(patongs[i], 64)
		}
		if i < len(posCharges) && posCharges[i] != "" {
			posCharge, _ = strconv.ParseFloat(posCharges[i], 64)
		}
		if i < len(wt2307s) && wt2307s[i] != "" {
			wt2307, _ = strconv.ParseFloat(wt2307s[i], 64)
		}
		if i < len(remarkss) {
			remarks = remarkss[i]
		}

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric input in items."}}`)
			http.Error(w, "Invalid numeric input in items", http.StatusBadRequest)
			return
		}

		saleItems = append(saleItems, domain.SaleItem{
			ItemID:    itemID,
			Qty:       qty,
			UOM:       uom,
			Price:     price,
			Cost:      cost,
			Patong:    patong,
			POSCharge: posCharge,
			WT2307:    wt2307,
			Remarks:   remarks,
			RefPL:     refPL,
		})
	}

	sale, err := domain.NewSale(docType, docStatus, parsedDate, docNumber, customerName, supplier, saleItems)
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
			INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, patong, pos_charge, wt_2307, total_remit, profit, profit_margin, remarks, ref_pl)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, sd.DocType, sd.DocStatus, sd.DocDate, sd.DocNumber, sd.CustomerName, sd.Supplier, sd.ItemID, sd.Qty, sd.UOM, sd.Price, sd.TotalSales, sd.Cost, sd.TotalCost, sd.Patong, sd.POSCharge, sd.WT2307, sd.TotalRemit, sd.Profit, sd.ProfitMargin, sd.Remarks, sd.RefPL)
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

	// No item selected yet — pass empty Uoms so template shows placeholder
	app.Render(w, "sale_item_row.html", map[string]interface{}{
		"Items": items,
		"Uoms":  []models.Uom{},
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
					COALESCE((SELECT SUM(s.qty) FROM sales_details s WHERE s.item_id = ? AND s.doc_status IN ('Posted', 'POSTED')), 0) AS total_sold,
					COALESCE((SELECT SUM(a.adjustment_qty) FROM inventory_adjustments a WHERE a.item_id = ?), 0) AS total_adjusted
			),
			oh AS (SELECT (total_received - total_sold + total_adjusted) AS on_hand FROM item_stats)
			SELECT r.cost, r.selling_price, r.pl_no
			FROM receiving_logs r, oh
			WHERE r.item_id = ? AND oh.on_hand > 0
			  AND (SELECT COALESCE(SUM(r2.qty), 0) FROM receiving_logs r2
			       WHERE r2.item_id = ? AND (r2.date < r.date OR (r2.date = r.date AND r2.id < r.id)))
			      < (SELECT COALESCE(SUM(s.qty), 0) FROM sales_details s WHERE s.item_id = ? AND s.doc_status IN ('Posted', 'POSTED'))
			ORDER BY r.date ASC, r.id ASC
			LIMIT 1
		`, itemID, itemID, itemID, itemID, itemID, itemID)
		if err == nil {
			lastCost = result.Cost
			lastPrice = result.SellingPrice
			lastPLNo = result.PLNo
		} else {
			var lastRec models.ReceivingLog
			err := db.DB.Get(&lastRec, "SELECT cost, selling_price, pl_no FROM receiving_logs WHERE item_id = ? ORDER BY date DESC, id DESC LIMIT 1", itemID)
			if err == nil {
				lastCost = lastRec.Cost
				lastPrice = lastRec.SellingPrice
				lastPLNo = lastRec.PLNo
			}
		}
	}

	// Load only UOMs valid for this item: default_uom + any muom from uom_settings
	var uoms []models.Uom
	if itemID > 0 {
		_ = db.DB.Select(&uoms, `
			SELECT code FROM (
				SELECT default_uom AS code FROM items WHERE id = ?
				UNION
				SELECT muom AS code FROM uom_settings WHERE item_id = ?
			) ORDER BY code ASC
		`, itemID, itemID)
	}

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
	docStatus := r.FormValue("doc_status")
	if docStatus == "" {
		docStatus = "Active"
	}
	docNumber := r.FormValue("doc_number")
	customerName := r.FormValue("customer_name")
	supplier := r.FormValue("supplier")
	itemIDStr := r.FormValue("item_id")
	qtyStr := r.FormValue("qty")
	uom := r.FormValue("uom")
	priceStr := r.FormValue("price")
	costStr := r.FormValue("cost")
	patongStr := r.FormValue("patong")
	posChargeStr := r.FormValue("pos_charge")
	wt2307Str := r.FormValue("wt_2307")
	remarks := r.FormValue("remarks")

	itemID, errItem := strconv.Atoi(itemIDStr)
	qty, errQty := strconv.ParseFloat(qtyStr, 64)
	price, errPrice := strconv.ParseFloat(priceStr, 64)
	cost, errCost := strconv.ParseFloat(costStr, 64)
	patong, _ := strconv.ParseFloat(patongStr, 64)
	posCharge, _ := strconv.ParseFloat(posChargeStr, 64)
	wt2307, _ := strconv.ParseFloat(wt2307Str, 64)

	parsedDate, dateErr := time.Parse("2006-01-02", docDateStr)
	if dateErr != nil || errItem != nil || errQty != nil || errPrice != nil || errCost != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric or date input."}}`)
		http.Error(w, "Invalid numeric or date input", http.StatusBadRequest)
		return
	}

	salesDetail, err := domain.NewSalesDetail(id, docType, docStatus, parsedDate, docNumber, customerName, supplier, itemID, qty, uom, price, cost, patong, posCharge, wt2307, remarks, "")
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "` + err.Error() + `"}}`)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE sales_details 
		SET doc_type = ?, doc_status = ?, doc_date = ?, doc_number = ?, customer_name = ?, supplier = ?, item_id = ?, qty = ?, uom = ?, price = ?, total_sales = ?, cost = ?, total_cost = ?, patong = ?, pos_charge = ?, wt_2307 = ?, total_remit = ?, profit = ?, profit_margin = ?, remarks = ?
		WHERE id = ?
	`, salesDetail.DocType, salesDetail.DocStatus, salesDetail.DocDate, salesDetail.DocNumber, salesDetail.CustomerName, salesDetail.Supplier, salesDetail.ItemID, salesDetail.Qty, salesDetail.UOM, salesDetail.Price, salesDetail.TotalSales, salesDetail.Cost, salesDetail.TotalCost, salesDetail.Patong, salesDetail.POSCharge, salesDetail.WT2307, salesDetail.TotalRemit, salesDetail.Profit, salesDetail.ProfitMargin, salesDetail.Remarks, id)

	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update sale log."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var updatedSale models.SalesDetailWithItem
	err = db.DB.Get(&updatedSale, `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost,
		       s.patong, s.pos_charge, s.wt_2307, s.total_remit, s.profit, s.profit_margin, s.remarks,
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

// UpdateSalesStatusHandler updates only the doc_status of an existing sales detail record
func (app *App) UpdateSalesStatusHandler(w http.ResponseWriter, r *http.Request) {
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

	docStatus := r.FormValue("doc_status")
	if docStatus == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Status cannot be empty."}}`)
		http.Error(w, "Status cannot be empty", http.StatusBadRequest)
		return
	}

	// Validate status is one of the allowed values
	allowed := false
	for _, s := range []string{"Active", "Posted", "Cancelled/Return"} {
		if docStatus == s {
			allowed = true
			break
		}
	}
	if !allowed {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid status value."}}`)
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE sales_details 
		SET doc_status = ?
		WHERE id = ?
	`, docStatus, id)

	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update sale status."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var updatedSale models.SalesDetailWithItem
	err = db.DB.Get(&updatedSale, `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost,
		       s.patong, s.pos_charge, s.wt_2307, s.total_remit, s.profit, s.profit_margin, s.remarks,
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

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Status updated to `+docStatus+`!"}}`)

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

