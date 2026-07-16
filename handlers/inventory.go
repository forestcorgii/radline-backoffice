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

	// Count total records matching search/filters
	countQuery := "SELECT COUNT(*) FROM (" + query + ")"
	var totalRecords int
	_ = db.DB.Get(&totalRecords, countQuery, args...)

	params := GetPaginationParams(r)
	pagination := BuildPagination(params, totalRecords)

	query += " LIMIT ? OFFSET ?"
	selectArgs := append(args, params.PageSize, params.Offset())

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

	paginationView := pagination.BuildView("/inventory/stock", "#inventory-stock-results", "inventory",
		[]string{"search", "stock_filter"}, r)

	// If HTMX request for filtering, return just the rows fragment and pagination
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			StockItems     []models.ItemStockView
			PaginationView PaginationView
		}{
			StockItems:     stockItems,
			PaginationView: paginationView,
		}
		app.Render(w, "inventory_stock_results.html", data)
		return
	}

	data := struct {
		StockItems     []models.ItemStockView
		PaginationView PaginationView
	}{
		StockItems:     stockItems,
		PaginationView: paginationView,
	}
	app.RenderPage(w, r, "inventory.html", data)
}

// StockReceivingPageHandler renders the standalone stock receiving page
func (app *App) StockReceivingPageHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT * FROM items ORDER BY code ASC")

	var receivingLogs []models.ReceivingLogWithItem
	_ = db.DB.Select(&receivingLogs, `
		SELECT r.*, i.code as item_code
		FROM receiving_logs r
		JOIN items i ON r.item_id = i.id
		ORDER BY r.date DESC, r.id DESC
	`)

	data := struct {
		Items            []models.Item
		ReceivingLogs    []models.ReceivingLogWithItem
		ReceivingRowData interface{}
	}{
		Items:         items,
		ReceivingLogs: receivingLogs,
		ReceivingRowData: map[string]interface{}{
			"Items": items,
		},
	}

	app.RenderPage(w, r, "stock_receiving.html", data)
}

// StockAdjustmentsPageHandler renders the standalone stock adjustments page
func (app *App) StockAdjustmentsPageHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT * FROM items ORDER BY code ASC")

	var adjustmentLogs []models.InventoryAdjustmentWithItem
	_ = db.DB.Select(&adjustmentLogs, `
		SELECT a.*, i.code as item_code
		FROM inventory_adjustments a
		JOIN items i ON a.item_id = i.id
		ORDER BY a.date DESC, a.id DESC
	`)

	data := struct {
		Items             []models.Item
		AdjustmentLogs    []models.InventoryAdjustmentWithItem
		AdjustmentRowData interface{}
	}{
		Items:          items,
		AdjustmentLogs: adjustmentLogs,
		AdjustmentRowData: map[string]interface{}{
			"Items": items,
		},
	}

	app.RenderPage(w, r, "stock_adjustments.html", data)
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
	costs := r.Form["cost"]
	unitPrices := r.Form["unit_price"]

	if len(itemIDs) == 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "At least one item is required."}}`)
		http.Error(w, "At least one item is required", http.StatusBadRequest)
		return
	}

	if len(itemIDs) != len(qtys) || len(itemIDs) != len(uoms) || len(itemIDs) != len(costs) || len(itemIDs) != len(unitPrices) {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Mismatch in item fields lengths."}}`)
		http.Error(w, "Mismatch in item fields lengths", http.StatusBadRequest)
		return
	}

	var receiveItems []domain.StockReceiveItem
	for i := range itemIDs {
		itemID, err1 := strconv.Atoi(itemIDs[i])
		qty, err2 := strconv.ParseFloat(qtys[i], 64)
		cost, err3 := strconv.ParseFloat(costs[i], 64)
		unitPrice, err4 := strconv.ParseFloat(unitPrices[i], 64)
		uom := uoms[i]

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric input in items."}}`)
			http.Error(w, "Invalid numeric input in items", http.StatusBadRequest)
			return
		}

		receiveItems = append(receiveItems, domain.StockReceiveItem{
			ItemID:    itemID,
			Qty:       qty,
			UOM:       uom,
			Cost:      cost,
			UnitPrice: unitPrice,
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
			INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, cost, total_cost, selling_price)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rl.Supplier, rl.Date, rl.PLNo, rl.ItemID, rl.Qty, rl.UOM, rl.UnitPrice, rl.Cost, rl.TotalCost, rl.SellingPrice)
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

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Stock received successfully!"}}`)
	w.Header().Set("HX-Location", "/inventory/receiving/logs")
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
	app.Render(w, "receiving_item_row.html", map[string]interface{}{
		"Items": items,
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

	app.Render(w, "receiving_item_row.html", map[string]interface{}{
		"Items":          items,
		"SelectedItemID": itemID,
		"DefaultUOM":     defaultUOM,
		"Cost":           lastCost,
		"Price":          lastPrice,
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

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Stock adjusted successfully!"}}`)
	w.Header().Set("HX-Location", "/inventory/adjustments/logs")
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
	app.Render(w, "adjustment_item_row.html", map[string]interface{}{
		"Items": items,
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

	app.Render(w, "adjustment_item_row.html", map[string]interface{}{
		"Items":          items,
		"SelectedItemID": itemID,
		"DefaultUOM":     defaultUOM,
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

	// Count total records matching search/filters
	countQuery := "SELECT COUNT(*) FROM (" + query + ")"
	var totalRecords int
	err := db.DB.Get(&totalRecords, countQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params := GetPaginationParams(r)
	pagination := BuildPagination(params, totalRecords)

	query += " ORDER BY t.month DESC, i.code ASC LIMIT ? OFFSET ?"
	selectArgs := append(args, params.PageSize, params.Offset())

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
	err = db.DB.Select(&rows, query, selectArgs...)
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

	paginationView := pagination.BuildView("/inventory/monthly", "#monthly-inventory-results", "monthly-inventory",
		[]string{"search", "month_filter"}, r)

	// If HTMX request for content filter, only render the table rows fragment and pagination
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			Rows           []MonthlyInventoryRow
			PaginationView PaginationView
		}{
			Rows:           rows,
			PaginationView: paginationView,
		}
		app.Render(w, "monthly_inventory_results.html", data)
		return
	}

	data := struct {
		Rows           []MonthlyInventoryRow
		Months         []string
		MonthFilter    string
		PaginationView PaginationView
	}{
		Rows:           rows,
		Months:         months,
		MonthFilter:    monthFilter,
		PaginationView: paginationView,
	}

	app.RenderPage(w, r, "monthly_inventory.html", data)
}

// ReceivingLogsHandler lists receiving logs with filtering
func (app *App) ReceivingLogsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	supplierFilter := r.URL.Query().Get("supplier_filter")
	sort := r.URL.Query().Get("sort")

	baseQuery := `
		SELECT r.*, i.code as item_code
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

	// Count total records
	countQuery := "SELECT COUNT(*) FROM (" + baseQuery + ")"
	var totalRecords int
	err := db.DB.Get(&totalRecords, countQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params := GetPaginationParams(r)
	pagination := BuildPagination(params, totalRecords)

	query := baseQuery
	switch sort {
	case "date_asc":
		query += " ORDER BY r.date ASC, r.id ASC"
	case "qty_desc":
		query += " ORDER BY r.qty DESC"
	default:
		query += " ORDER BY r.date DESC, r.id DESC"
	}
	query += " LIMIT ? OFFSET ?"
	selectArgs := append(args, params.PageSize, params.Offset())

	var receivingLogs []models.ReceivingLogWithItem
	err = db.DB.Select(&receivingLogs, query, selectArgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	paginationView := pagination.BuildView("/inventory/receiving/logs", "#receiving-logs-results", "receiving",
		[]string{"search", "supplier_filter", "sort"}, r)

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			ReceivingLogs  []models.ReceivingLogWithItem
			PaginationView PaginationView
		}{
			ReceivingLogs:  receivingLogs,
			PaginationView: paginationView,
		}
		app.Render(w, "receiving_logs_results.html", data)
		return
	}

	data := struct {
		ReceivingLogs  []models.ReceivingLogWithItem
		PaginationView PaginationView
	}{
		ReceivingLogs:  receivingLogs,
		PaginationView: paginationView,
	}

	app.RenderPage(w, r, "receiving_logs.html", data)
}

// AdjustmentLogsHandler lists stock adjustments with filtering
func (app *App) AdjustmentLogsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")

	baseQuery := `
		SELECT a.*, i.code as item_code
		FROM inventory_adjustments a
		JOIN items i ON a.item_id = i.id
	`
	var args []interface{}
	var conditions []string

	if search != "" {
		conditions = append(conditions, "(a.remarks LIKE ? OR i.code LIKE ? OR CAST(a.adjustment_id AS TEXT) LIKE ?)")
		wildcard := "%" + search + "%"
		args = append(args, wildcard, wildcard, wildcard)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM (" + baseQuery + ")"
	var totalRecords int
	err := db.DB.Get(&totalRecords, countQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params := GetPaginationParams(r)
	pagination := BuildPagination(params, totalRecords)

	query := baseQuery
	switch sort {
	case "date_asc":
		query += " ORDER BY a.date ASC, a.id ASC"
	case "qty_desc":
		query += " ORDER BY ABS(a.adjustment_qty) DESC"
	default:
		query += " ORDER BY a.date DESC, a.id DESC"
	}
	query += " LIMIT ? OFFSET ?"
	selectArgs := append(args, params.PageSize, params.Offset())

	var adjustmentLogs []models.InventoryAdjustmentWithItem
	err = db.DB.Select(&adjustmentLogs, query, selectArgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	paginationView := pagination.BuildView("/inventory/adjustments/logs", "#adjustment-logs-results", "adjustment",
		[]string{"search", "sort"}, r)

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			AdjustmentLogs []models.InventoryAdjustmentWithItem
			PaginationView PaginationView
		}{
			AdjustmentLogs: adjustmentLogs,
			PaginationView: paginationView,
		}
		app.Render(w, "adjustment_logs_results.html", data)
		return
	}

	data := struct {
		AdjustmentLogs []models.InventoryAdjustmentWithItem
		PaginationView PaginationView
	}{
		AdjustmentLogs: adjustmentLogs,
		PaginationView: paginationView,
	}

	app.RenderPage(w, r, "adjustment_logs.html", data)
}
