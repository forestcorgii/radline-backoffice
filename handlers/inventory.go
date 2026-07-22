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

// stockListQuery returns the SQL query for computing per-item stock levels
func stockListQuery(search, stockFilter string) (string, []interface{}) {
	var args []interface{}

	query := `
		SELECT
			i.id AS item_id,
			i.code AS item_code,
			i.description,
			i.default_uom,
			COALESCE(r.total_received, 0) AS total_received,
			COALESCE(s.total_sold, 0) AS total_sold,
			COALESCE(a.total_adjusted, 0) AS total_adjusted,
			(COALESCE(r.total_received, 0) - COALESCE(s.total_sold, 0) + COALESCE(a.total_adjusted, 0)) AS on_hand,
			0.0 AS current_cost,
			0.0 AS current_price
		FROM items i
		LEFT JOIN (
			SELECT item_id, SUM(qty) AS total_received FROM receiving_logs GROUP BY item_id
		) r ON r.item_id = i.id
		LEFT JOIN (
			SELECT item_id, SUM(qty) AS total_sold FROM sales_details GROUP BY item_id
		) s ON s.item_id = i.id
		LEFT JOIN (
			SELECT item_id, SUM(adjustment_qty) AS total_adjusted FROM inventory_adjustments GROUP BY item_id
		) a ON a.item_id = i.id
	`

	var conditions []string
	if search != "" {
		conditions = append(conditions, "(i.code LIKE ? OR i.description LIKE ?)")
		wildcard := "%" + search + "%"
		args = append(args, wildcard, wildcard)
	}

	switch stockFilter {
	case "in_stock":
		conditions = append(conditions, "(COALESCE(r.total_received, 0) - COALESCE(s.total_sold, 0) + COALESCE(a.total_adjusted, 0)) > 0")
	case "out_of_stock":
		conditions = append(conditions, "(COALESCE(r.total_received, 0) - COALESCE(s.total_sold, 0) + COALESCE(a.total_adjusted, 0)) <= 0")
	case "low_stock":
		conditions = append(conditions, "(COALESCE(r.total_received, 0) - COALESCE(s.total_sold, 0) + COALESCE(a.total_adjusted, 0)) > 0 AND (COALESCE(r.total_received, 0) - COALESCE(s.total_sold, 0) + COALESCE(a.total_adjusted, 0)) <= 10")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY i.code ASC"
	return query, args
}

// InventoryHandler renders the inventory landing page with stock list
func (app *App) InventoryHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	stockFilter := r.URL.Query().Get("stock_filter")

	query, args := stockListQuery(search, stockFilter)

	limit := GetLimitParam(r)

	query += " LIMIT ?"
	selectArgs := append(args, limit)

	var stockItems []models.ItemStockView
	_ = db.DB.Select(&stockItems, query, selectArgs...)

	// Populate CurrentCost and CurrentPrice in batch using the oldest PL with stock available (with latest fallback)
	// Uses a single efficient SQL query per item instead of loading all transaction data into memory
	if len(stockItems) > 0 {
		for i := range stockItems {
			itemID := stockItems[i].ItemID
			// Try to get the oldest PL with remaining stock
			type CostPriceResult struct {
				Cost         float64 `db:"cost"`
				SellingPrice float64 `db:"selling_price"`
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
				SELECT r.cost, r.selling_price
				FROM receiving_logs r, oh
				WHERE r.item_id = ? AND oh.on_hand > 0
				  AND (SELECT COALESCE(SUM(r2.qty), 0) FROM receiving_logs r2
				       WHERE r2.item_id = ? AND (r2.date < r.date OR (r2.date = r.date AND r2.id < r.id)))
				      < (SELECT COALESCE(SUM(s.qty), 0) FROM sales_details s WHERE s.item_id = ?)
				ORDER BY r.date ASC, r.id ASC
				LIMIT 1
			`, itemID, itemID, itemID, itemID, itemID, itemID)
			if err == nil {
				stockItems[i].CurrentCost = result.Cost
				stockItems[i].CurrentPrice = result.SellingPrice
			} else {
				// Fallback to latest receiving log's cost and price
				var lastRec models.ReceivingLog
				err := db.DB.Get(&lastRec, "SELECT cost, selling_price FROM receiving_logs WHERE item_id = ? ORDER BY date DESC, id DESC LIMIT 1", itemID)
				if err == nil {
					stockItems[i].CurrentCost = lastRec.Cost
					stockItems[i].CurrentPrice = lastRec.SellingPrice
				}
			}
		}
	}

	// If HTMX request for filtering, return just the rows fragment
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			StockItems []models.ItemStockView
		}{
			StockItems: stockItems,
		}
		app.Render(w, "inventory_stock_results.html", data)
		return
	}

	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	rowData := map[string]interface{}{
		"Items": items,
		"Uoms":  uoms,
	}

	data := struct {
		StockItems        []models.ItemStockView
		Items             []models.Item
		Uoms              []models.Uom
		ReceivingRowData  interface{}
		AdjustmentRowData interface{}
	}{
		StockItems:        stockItems,
		Items:             items,
		Uoms:              uoms,
		ReceivingRowData:  rowData,
		AdjustmentRowData: rowData,
	}
	app.RenderPage(w, r, "inventory.html", data)
}

// StockReceivingPageHandler redirects to inventory overview page
func (app *App) StockReceivingPageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

// StockAdjustmentsPageHandler redirects to inventory overview page
func (app *App) StockAdjustmentsPageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

// ReceiveStockHandler records incoming inventory
func (app *App) ReceiveStockHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to parse form."}}`)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	supplier := r.FormValue("supplier")
	plNo := r.FormValue("pl_no")
	dateStr := r.FormValue("date")

	parsedDate, dateErr := time.Parse("2006-01-02", dateStr)
	if dateErr != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid date format. Expected YYYY-MM-DD."}}`)
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	itemIDs := r.Form["item_id"]
	qtys := r.Form["qty"]
	uoms := r.Form["uom"]
	unitPrices := r.Form["unit_price"]
	less1s := r.Form["less1"]
	less2s := r.Form["less2"]
	markups := r.Form["markup"]
	remarksList := r.Form["remarks"]

	if len(itemIDs) == 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "At least one item is required."}}`)
		http.Error(w, "At least one item is required", http.StatusBadRequest)
		return
	}

	if len(itemIDs) != len(qtys) || len(itemIDs) != len(uoms) || len(itemIDs) != len(unitPrices) {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Mismatch in item fields lengths."}}`)
		http.Error(w, "Mismatch in item fields lengths", http.StatusBadRequest)
		return
	}

	var receiveItems []domain.StockReceiveItem
	for i := range itemIDs {
		itemID, err1 := strconv.Atoi(itemIDs[i])
		qty, err2 := strconv.ParseFloat(qtys[i], 64)
		unitPrice, err3 := strconv.ParseFloat(unitPrices[i], 64)
		uom := uoms[i]

		if err1 != nil || err2 != nil || err3 != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric input in items."}}`)
			http.Error(w, "Invalid numeric input in items", http.StatusBadRequest)
			return
		}

		var less1, less2, markup float64
		if i < len(less1s) && less1s[i] != "" {
			less1, _ = strconv.ParseFloat(less1s[i], 64)
		}
		if i < len(less2s) && less2s[i] != "" {
			less2, _ = strconv.ParseFloat(less2s[i], 64)
		}
		if i < len(markups) && markups[i] != "" {
			markup, _ = strconv.ParseFloat(markups[i], 64)
		}
		if markup <= 0 {
			markup = 130
		}

		var remarksVal string
		if i < len(remarksList) {
			remarksVal = remarksList[i]
		}

		receiveItems = append(receiveItems, domain.StockReceiveItem{
			ItemID:    itemID,
			Qty:       qty,
			UOM:       uom,
			UnitPrice: unitPrice,
			Less1:     less1,
			Less2:     less2,
			Markup:    markup,
			Remarks:   remarksVal,
		})
	}

	stockReceive, err := domain.NewStockReceive(plNo, supplier, parsedDate, receiveItems)
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

	logs := stockReceive.ToReceivingLogs()
	for _, rl := range logs {
		_, err = tx.Exec(`
			INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, less1, less2, cost, total_cost, markup, selling_price, remarks)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rl.Supplier, rl.Date, rl.PLNo, rl.ItemID, rl.Qty, rl.UOM, rl.UnitPrice, rl.Less1, rl.Less2, rl.Cost, rl.TotalCost, rl.Markup, rl.SellingPrice, rl.Remarks)
		if err != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to save receiving log."}}`)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to commit receiving transaction."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Stock received successfully!"}, "stock-received": ""}`)
	w.WriteHeader(http.StatusOK)
}

// NewReceivingRowHandler renders a single empty receiving item row template
func (app *App) NewReceivingRowHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// No item selected yet — pass empty Uoms so template shows placeholder
	app.Render(w, "receiving_item_row.html", map[string]interface{}{
		"Items": items,
		"Uoms":  []models.Uom{},
	})
}

// ReceivingItemRowDetailsHandler renders a single receiving item row template populated with item defaults
func (app *App) ReceivingItemRowDetailsHandler(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.URL.Query().Get("item_id")
	itemID, _ := strconv.Atoi(itemIDStr)

	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var defaultUOM string
	var lastPrice, lastLess1, lastLess2, lastMarkup float64

	if itemID > 0 {
		_ = db.DB.Get(&defaultUOM, "SELECT default_uom FROM items WHERE id = ?", itemID)

		// Get last receiving pricing data
		var lastRec models.ReceivingLog
		err := db.DB.Get(&lastRec, "SELECT unit_price, less1, less2, markup FROM receiving_logs WHERE item_id = ? ORDER BY date DESC, id DESC LIMIT 1", itemID)
		if err == nil {
			lastPrice = lastRec.UnitPrice
			lastLess1 = lastRec.Less1
			lastLess2 = lastRec.Less2
			lastMarkup = lastRec.Markup
		}
	}

	if lastMarkup <= 0 {
		lastMarkup = 130
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

	app.Render(w, "receiving_item_row.html", map[string]interface{}{
		"Items":          items,
		"SelectedItemID": itemID,
		"DefaultUOM":     defaultUOM,
		"Price":          lastPrice,
		"Less1":          lastLess1,
		"Less2":          lastLess2,
		"Markup":         lastMarkup,
		"Uoms":           uoms,
	})
}

// AdjustStockHandler records a multi-item manual inventory adjustment
func (app *App) AdjustStockHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to parse form."}}`)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dateStr := r.FormValue("date")
	remarks := r.FormValue("remarks")

	parsedDate, dateErr := time.Parse("2006-01-02", dateStr)
	if dateErr != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid date format. Expected YYYY-MM-DD."}}`)
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	itemIDs := r.Form["item_id"]
	qtys := r.Form["adjustment_qty"]
	uoms := r.Form["uom"]
	costs := r.Form["cost"]

	if len(itemIDs) == 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "At least one item is required."}}`)
		http.Error(w, "At least one item is required", http.StatusBadRequest)
		return
	}

	if len(itemIDs) != len(qtys) || len(itemIDs) != len(uoms) || len(itemIDs) != len(costs) {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Mismatch in item fields lengths."}}`)
		http.Error(w, "Mismatch in item fields lengths", http.StatusBadRequest)
		return
	}

	var adjItems []domain.StockAdjustmentItem
	for i := range itemIDs {
		itemID, err1 := strconv.Atoi(itemIDs[i])
		qty, err2 := strconv.ParseFloat(qtys[i], 64)
		cost, err3 := strconv.ParseFloat(costs[i], 64)
		uom := uoms[i]

		if err1 != nil || err2 != nil || err3 != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric input in items."}}`)
			http.Error(w, "Invalid numeric input in items", http.StatusBadRequest)
			return
		}

		adjItems = append(adjItems, domain.StockAdjustmentItem{
			ItemID: itemID,
			Qty:    qty,
			UOM:    uom,
			Cost:   cost,
		})
	}

	stockAdj, err := domain.NewStockAdjustment(parsedDate, remarks, adjItems)
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

	// Insert the header into stock_adjustments
	result, err := tx.Exec(`INSERT INTO stock_adjustments (date, remarks) VALUES (?, ?)`, stockAdj.Date, stockAdj.Remarks)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to save adjustment header."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	adjustmentID, _ := result.LastInsertId()

	// Insert all item rows
	adjustments := stockAdj.ToInventoryAdjustments()
	for _, adj := range adjustments {
		_, err = tx.Exec(`
			INSERT INTO inventory_adjustments (adjustment_id, date, item_id, uom, adjustment_qty, cost, remarks)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, adjustmentID, adj.Date, adj.ItemID, adj.UOM, adj.AdjustmentQty, adj.Cost, adj.Remarks)
		if err != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to save adjustment item."}}`)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to commit adjustment transaction."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Stock adjusted successfully!"}, "stock-adjusted": ""}`)
	w.WriteHeader(http.StatusOK)
}

// NewAdjustmentRowHandler renders a single empty adjustment item row template
func (app *App) NewAdjustmentRowHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// No item selected yet — pass empty Uoms so template shows placeholder
	app.Render(w, "adjustment_item_row.html", map[string]interface{}{
		"Items": items,
		"Uoms":  []models.Uom{},
	})
}

// AdjustmentItemRowDetailsHandler renders a populated adjustment item row when an item is selected
func (app *App) AdjustmentItemRowDetailsHandler(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.URL.Query().Get("item_id")
	itemID, _ := strconv.Atoi(itemIDStr)

	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var defaultUOM string
	if itemID > 0 {
		_ = db.DB.Get(&defaultUOM, "SELECT default_uom FROM items WHERE id = ?", itemID)
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

	app.Render(w, "adjustment_item_row.html", map[string]interface{}{
		"Items":          items,
		"SelectedItemID": itemID,
		"DefaultUOM":     defaultUOM,
		"Uoms":           uoms,
	})
}

// MonthlyInventoryHandler renders the monthly inventory movement report
func (app *App) MonthlyInventoryHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	monthFilter := r.URL.Query().Get("month_filter")

	query := `
		SELECT 
			t.month,
			t.item_id,
			i.code AS item_code,
			i.description,
			i.default_uom,
			SUM(t.received) AS qty_received,
			SUM(t.sold) AS qty_sold,
			SUM(t.adjusted) AS qty_adjusted,
			(SUM(t.received) - SUM(t.sold) + SUM(t.adjusted)) AS net_change
		FROM (
			SELECT substr(date, 1, 7) AS month, item_id, qty AS received, 0.0 AS sold, 0.0 AS adjusted FROM receiving_logs
			UNION ALL
			SELECT substr(doc_date, 1, 7) AS month, item_id, 0.0 AS received, qty AS sold, 0.0 AS adjusted FROM sales_details
			UNION ALL
			SELECT substr(date, 1, 7) AS month, item_id, 0.0 AS received, 0.0 AS sold, adjustment_qty AS adjusted FROM inventory_adjustments
		) t
		JOIN items i ON t.item_id = i.id
	`
	var args []interface{}
	var conditions []string

	if search != "" {
		conditions = append(conditions, "(i.code LIKE ? OR i.description LIKE ?)")
		wildcard := "%" + search + "%"
		args = append(args, wildcard, wildcard)
	}

	if monthFilter != "" && monthFilter != "all" {
		conditions = append(conditions, "t.month = ?")
		args = append(args, monthFilter)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY t.month, t.item_id"

	limit := GetLimitParam(r)

	query += " ORDER BY t.month DESC, i.code ASC LIMIT ?"
	selectArgs := append(args, limit)

	type MonthlyInventoryRow struct {
		Month       string  `db:"month"`
		ItemID      int     `db:"item_id"`
		ItemCode    string  `db:"item_code"`
		Description string  `db:"description"`
		DefaultUOM  string  `db:"default_uom"`
		QtyReceived float64 `db:"qty_received"`
		QtySold     float64 `db:"qty_sold"`
		QtyAdjusted float64 `db:"qty_adjusted"`
		NetChange   float64 `db:"net_change"`
	}

	var rows []MonthlyInventoryRow
	err := db.DB.Select(&rows, query, selectArgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get list of unique months for the filter dropdown
	var months []string
	_ = db.DB.Select(&months, `
		SELECT DISTINCT substr(date, 1, 7) as m FROM receiving_logs WHERE date IS NOT NULL
		UNION
		SELECT DISTINCT substr(doc_date, 1, 7) as m FROM sales_details WHERE doc_date IS NOT NULL
		UNION
		SELECT DISTINCT substr(date, 1, 7) as m FROM inventory_adjustments WHERE date IS NOT NULL
		ORDER BY m DESC
	`)

	// If HTMX request for content filter, only render the table rows fragment
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			Rows []MonthlyInventoryRow
		}{
			Rows: rows,
		}
		app.Render(w, "monthly_inventory_results.html", data)
		return
	}

	data := struct {
		Rows        []MonthlyInventoryRow
		Months      []string
		MonthFilter string
	}{
		Rows:        rows,
		Months:      months,
		MonthFilter: monthFilter,
	}

	app.RenderPage(w, r, "monthly_inventory.html", data)
}

// ReceivingLogsHandler lists receiving logs with filtering
func (app *App) ReceivingLogsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	supplierFilter := r.URL.Query().Get("supplier_filter")
	sort := r.URL.Query().Get("sort")

	baseQuery := `
		SELECT r.*, i.code as item_code, i.description as item_description
		FROM receiving_logs r
		JOIN items i ON r.item_id = i.id
	`
	var args []interface{}
	var conditions []string

	if search != "" {
		conditions = append(conditions, "(r.pl_no LIKE ? OR i.code LIKE ?)")
		wildcard := "%" + search + "%"
		args = append(args, wildcard, wildcard)
	}

	if supplierFilter != "" && supplierFilter != "all" {
		conditions = append(conditions, "r.supplier = ?")
		args = append(args, supplierFilter)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := GetLimitParam(r)

	query := baseQuery
	switch sort {
	case "date_asc":
		query += " ORDER BY r.date ASC, r.id ASC"
	case "qty_desc":
		query += " ORDER BY r.qty DESC"
	default:
		query += " ORDER BY r.date DESC, r.id DESC"
	}
	query += " LIMIT ?"
	selectArgs := append(args, limit)

	var receivingLogs []models.ReceivingLogWithItem
	err := db.DB.Select(&receivingLogs, query, selectArgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var suppliers []string
	err = db.DB.Select(&suppliers, "SELECT DISTINCT supplier FROM receiving_logs WHERE supplier != '' ORDER BY supplier ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			ReceivingLogs []models.ReceivingLogWithItem
		}{
			ReceivingLogs: receivingLogs,
		}
		app.Render(w, "receiving_logs_results.html", data)
		return
	}

	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	receivingRowData := map[string]interface{}{
		"Items": items,
		"Uoms":  uoms,
	}

	data := struct {
		ReceivingLogs    []models.ReceivingLogWithItem
		Suppliers        []string
		SupplierFilter   string
		ReceivingRowData interface{}
	}{
		ReceivingLogs:    receivingLogs,
		Suppliers:        suppliers,
		SupplierFilter:   supplierFilter,
		ReceivingRowData: receivingRowData,
	}

	app.RenderPage(w, r, "receiving_logs.html", data)
}

// AdjustmentLogsHandler lists stock adjustments with filtering
func (app *App) AdjustmentLogsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")

	baseQuery := `
		SELECT a.*, i.code as item_code, i.description as item_description
		FROM inventory_adjustments a
		JOIN items i ON a.item_id = i.id
	`
	var args []interface{}
	var conditions []string

	if search != "" {
		conditions = append(conditions, "(a.remarks LIKE ? OR i.description LIKE ? OR CAST(a.adjustment_id AS TEXT) LIKE ?)")
		wildcard := "%" + search + "%"
		args = append(args, wildcard, wildcard, wildcard)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := GetLimitParam(r)

	query := baseQuery
	switch sort {
	case "date_asc":
		query += " ORDER BY a.date ASC, a.id ASC"
	case "qty_desc":
		query += " ORDER BY ABS(a.adjustment_qty) DESC"
	default:
		query += " ORDER BY a.date DESC, a.id DESC"
	}
	query += " LIMIT ?"
	selectArgs := append(args, limit)

	var adjustmentLogs []models.InventoryAdjustmentWithItem
	err := db.DB.Select(&adjustmentLogs, query, selectArgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			AdjustmentLogs []models.InventoryAdjustmentWithItem
		}{
			AdjustmentLogs: adjustmentLogs,
		}
		app.Render(w, "adjustment_logs_results.html", data)
		return
	}

	var adjItems []models.Item
	_ = db.DB.Select(&adjItems, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")
	var adjUoms []models.Uom
	_ = db.DB.Select(&adjUoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	adjustmentRowData := map[string]interface{}{
		"Items": adjItems,
		"Uoms":  adjUoms,
	}

	data := struct {
		AdjustmentLogs    []models.InventoryAdjustmentWithItem
		AdjustmentRowData interface{}
	}{
		AdjustmentLogs:    adjustmentLogs,
		AdjustmentRowData: adjustmentRowData,
	}

	app.RenderPage(w, r, "adjustment_logs.html", data)
}

// DeleteReceivingLogHandler deletes a receiving log by ID
func (app *App) DeleteReceivingLogHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	_, err := db.DB.Exec("DELETE FROM receiving_logs WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to delete receiving log."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Receiving log deleted."}}`)
	w.WriteHeader(http.StatusOK)
}

// EditReceivingLogHandler returns an inline edit form row for a receiving log
func (app *App) EditReceivingLogHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	var log models.ReceivingLogWithItem
	err := db.DB.Get(&log, `
		SELECT r.*, i.code as item_code, i.description as item_description
		FROM receiving_logs r
		JOIN items i ON r.item_id = i.id
		WHERE r.id = ?
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.Render(w, "receiving_log_edit_row.html", log)
}

// UpdateReceivingLogHandler updates a receiving log inline
func (app *App) UpdateReceivingLogHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	unitPrice, _ := strconv.ParseFloat(r.FormValue("unit_price"), 64)
	less1, _ := strconv.ParseFloat(r.FormValue("less1"), 64)
	less2, _ := strconv.ParseFloat(r.FormValue("less2"), 64)
	markup, _ := strconv.ParseFloat(r.FormValue("markup"), 64)
	remarks := r.FormValue("remarks")
	qty, _ := strconv.ParseFloat(r.FormValue("qty"), 64)

	if markup <= 0 {
		markup = 130
	}

	unitCost := unitPrice * (1 - less1/100) * (1 - less2/100)
	totalCost := qty * unitCost
	sellingPrice := unitCost * markup / 100

	_, err := db.DB.Exec(`
		UPDATE receiving_logs
		SET unit_price = ?, less1 = ?, less2 = ?, cost = ?, total_cost = ?, markup = ?, selling_price = ?, remarks = ?, qty = ?
		WHERE id = ?
	`, unitPrice, less1, less2, unitCost, totalCost, markup, sellingPrice, remarks, qty, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update receiving log."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var log models.ReceivingLogWithItem
	err = db.DB.Get(&log, `
		SELECT r.*, i.code as item_code, i.description as item_description
		FROM receiving_logs r
		JOIN items i ON r.item_id = i.id
		WHERE r.id = ?
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Receiving log updated."}}`)
	app.Render(w, "receiving_log_row.html", log)
}
