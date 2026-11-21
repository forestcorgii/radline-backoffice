package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"rav/db"
	"rav/domain"
	"rav/models"
)

func setToastTrigger(w http.ResponseWriter, toastType, message string, extraEvents ...string) {
	payload := map[string]interface{}{
		"show-toast": map[string]string{
			"type":    toastType,
			"message": message,
		},
	}
	for _, ev := range extraEvents {
		if ev != "" {
			payload[ev] = ""
		}
	}
	if b, err := json.Marshal(payload); err == nil {
		w.Header().Set("HX-Trigger", string(b))
	}
}

func isUniqueViolationErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "23505")
}

// stockListQuery returns the SQL query for computing per-item stock levels (normalized to Default UOM)
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
			SELECT r.item_id, SUM(r.qty * COALESCE(u.conversion_factor, 1.0)) AS total_received
			FROM receiving_logs r
			LEFT JOIN uom_settings u ON u.item_id = r.item_id AND LOWER(u.muom) = LOWER(r.uom)
			GROUP BY r.item_id
		) r ON r.item_id = i.id
		LEFT JOIN (
			SELECT s.item_id, SUM(s.qty * COALESCE(u.conversion_factor, 1.0)) AS total_sold
			FROM sales_details s
			LEFT JOIN uom_settings u ON u.item_id = s.item_id AND LOWER(u.muom) = LOWER(s.uom)
			WHERE s.doc_status IN ('Posted', 'POSTED')
			GROUP BY s.item_id
		) s ON s.item_id = i.id
		LEFT JOIN (
			SELECT a.item_id, SUM(a.adjustment_qty * COALESCE(u.conversion_factor, 1.0)) AS total_adjusted
			FROM inventory_adjustments a
			LEFT JOIN uom_settings u ON u.item_id = a.item_id AND LOWER(u.muom) = LOWER(a.uom)
			GROUP BY a.item_id
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
	if err := db.DB.Select(&stockItems, query, selectArgs...); err != nil {
		log.Printf("ERROR: InventoryHandler stock list query failed: %v", err)
		http.Error(w, "Failed to load inventory stock list", http.StatusInternalServerError)
		return
	}

	// Populate CurrentCost and CurrentPrice in batch using the oldest PL with stock available (with latest fallback)
	if len(stockItems) > 0 {
		itemIDs := make([]int, len(stockItems))
		for i := range stockItems {
			itemIDs[i] = stockItems[i].ItemID
		}

		// 1. Batch fetch receiving logs with conversion factors ordered chronologically
		type recBatchRow struct {
			ItemID       int       `db:"item_id"`
			ID           int       `db:"id"`
			Date         time.Time `db:"date"`
			Qty          float64   `db:"qty"`
			Cost         float64   `db:"cost"`
			SellingPrice float64   `db:"selling_price"`
			Factor       float64   `db:"factor"`
		}
		var recRows []recBatchRow
		recQuery, recArgs, err := sqlx.In(`
			SELECT r.item_id, r.id, r.date, r.qty, r.cost, COALESCE(r.selling_price, 0.0) AS selling_price,
			       COALESCE(u.conversion_factor, 1.0) AS factor
			FROM receiving_logs r
			LEFT JOIN uom_settings u ON u.item_id = r.item_id AND LOWER(u.muom) = LOWER(r.uom)
			WHERE r.item_id IN (?)
			ORDER BY r.item_id ASC, r.date ASC, r.id ASC
		`, itemIDs)
		if err == nil {
			_ = db.DB.Select(&recRows, recQuery, recArgs...)
		}

		recByItem := make(map[int][]recBatchRow, len(stockItems))
		for _, rr := range recRows {
			recByItem[rr.ItemID] = append(recByItem[rr.ItemID], rr)
		}

		// 2. Batch fetch negative adjustments in default UOM (positive adjustments do not resurrect depleted PLs)
		type negAdjRow struct {
			ItemID int     `db:"item_id"`
			NegQty float64 `db:"neg_qty"`
		}
		var negAdjRows []negAdjRow
		negQuery, negArgs, err := sqlx.In(`
			SELECT a.item_id, COALESCE(SUM(-a.adjustment_qty * COALESCE(u.conversion_factor, 1.0)), 0.0) AS neg_qty
			FROM inventory_adjustments a
			LEFT JOIN uom_settings u ON u.item_id = a.item_id AND LOWER(u.muom) = LOWER(a.uom)
			WHERE a.item_id IN (?) AND a.adjustment_qty < 0
			GROUP BY a.item_id
		`, itemIDs)
		if err == nil {
			_ = db.DB.Select(&negAdjRows, negQuery, negArgs...)
		}
		negAdjByItem := make(map[int]float64, len(negAdjRows))
		for _, nr := range negAdjRows {
			negAdjByItem[nr.ItemID] = nr.NegQty
		}

		for i := range stockItems {
			itemID := stockItems[i].ItemID
			logs := recByItem[itemID]
			if len(logs) == 0 {
				continue
			}

			found := false
			if stockItems[i].OnHand > 1e-9 {
				totalConsumed := stockItems[i].TotalSold + negAdjByItem[itemID]
				cumReceived := 0.0
				for _, rl := range logs {
					factor := rl.Factor
					if factor <= 0 {
						factor = 1.0
					}
					cumReceived += rl.Qty * factor
					if cumReceived > totalConsumed+1e-9 {
						stockItems[i].CurrentCost = domain.Round2(rl.Cost / factor)
						stockItems[i].CurrentPrice = domain.Round2(rl.SellingPrice / factor)
						found = true
						break
					}
				}
			}

			if !found {
				// Fallback to latest receiving log's cost and price scaled to DefaultUOM
				lastRec := logs[len(logs)-1]
				factor := lastRec.Factor
				if factor <= 0 {
					factor = 1.0
				}
				stockItems[i].CurrentCost = domain.Round2(lastRec.Cost / factor)
				stockItems[i].CurrentPrice = domain.Round2(lastRec.SellingPrice / factor)
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
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 25")

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
	unitCosts := r.Form["unit_cost"]
	remarksList := r.Form["remarks"]

	if len(itemIDs) == 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "At least one item is required."}}`)
		http.Error(w, "At least one item is required", http.StatusBadRequest)
		return
	}

	if len(itemIDs) != len(qtys) || len(itemIDs) != len(uoms) || len(itemIDs) != len(unitPrices) || len(itemIDs) != len(unitCosts) {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Mismatch in item fields lengths."}}`)
		http.Error(w, "Mismatch in item fields lengths", http.StatusBadRequest)
		return
	}

	var receiveItems []domain.StockReceiveItem
	for i := range itemIDs {
		itemID, err1 := strconv.Atoi(itemIDs[i])
		qty, err2 := strconv.ParseFloat(qtys[i], 64)
		unitPrice, err3 := strconv.ParseFloat(unitPrices[i], 64)
		unitCost, err4 := strconv.ParseFloat(unitCosts[i], 64)
		uom := uoms[i]

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid numeric input in items."}}`)
			http.Error(w, "Invalid numeric input in items", http.StatusBadRequest)
			return
		}

		var less1, less2 float64
		if i < len(less1s) && less1s[i] != "" {
			less1, _ = strconv.ParseFloat(less1s[i], 64)
		}
		if i < len(less2s) && less2s[i] != "" {
			less2, _ = strconv.ParseFloat(less2s[i], 64)
		}
		markup := 130.0

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
			UnitCost:  unitCost,
			Markup:    markup,
			Remarks:   remarksVal,
		})
	}

	stockReceive, err := domain.NewStockReceive(plNo, supplier, parsedDate, receiveItems)
	if err != nil {
		setToastTrigger(w, "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		setToastTrigger(w, "error", "Database transaction failed.")
		http.Error(w, "Database transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	logs := stockReceive.ToReceivingLogs()

	stmt, err := tx.Prepare(tx.Rebind(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, less1, less2, cost, total_cost, markup, selling_price, remarks)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`))
	if err != nil {
		setToastTrigger(w, "error", "Failed to prepare receiving log statement.")
		http.Error(w, "Failed to prepare receiving log statement", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	for _, rl := range logs {
		_, err = stmt.Exec(rl.Supplier, rl.Date, rl.PLNo, rl.ItemID, rl.Qty, rl.UOM, rl.UnitPrice, rl.Less1, rl.Less2, rl.Cost, rl.TotalCost, rl.Markup, rl.SellingPrice, rl.Remarks)
		if err != nil {
			setToastTrigger(w, "error", "Failed to save receiving log.")
			http.Error(w, "Failed to save receiving log", http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		setToastTrigger(w, "error", "Failed to commit receiving transaction.")
		http.Error(w, "Failed to commit receiving transaction", http.StatusInternalServerError)
		return
	}

	setToastTrigger(w, "success", "Stock received successfully!", "stock-received")
	w.WriteHeader(http.StatusOK)
}

// NewReceivingRowHandler renders a single empty receiving item row template
func (app *App) NewReceivingRowHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 20")
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
	var err error
	if itemID > 0 {
		err = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY (id = ?) DESC, description ASC LIMIT 20", itemID)
	} else {
		err = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 20")
	}
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
			if lastRec.UnitPrice != nil {
				lastPrice = *lastRec.UnitPrice
			}
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
			SELECT u.code FROM (
				SELECT default_uom AS code FROM items WHERE id = ?
				UNION
				SELECT muom AS code FROM uom_settings WHERE item_id = ?
			) AS u WHERE u.code != '' AND u.code IS NOT NULL
			ORDER BY u.code ASC
		`, itemID, itemID)
	}

	app.Render(w, "receiving_item_row.html", map[string]interface{}{
		"Items":          items,
		"SelectedItemID": itemID,
		"DefaultUOM":     defaultUOM,
		"Cost":           lastPrice,
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
		setToastTrigger(w, "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		setToastTrigger(w, "error", "Database transaction failed.")
		http.Error(w, "Database transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert the header into stock_adjustments
	var adjustmentID int64
	err = tx.QueryRow(`INSERT INTO stock_adjustments (date, remarks) VALUES (?, ?) RETURNING id`, stockAdj.Date, stockAdj.Remarks).Scan(&adjustmentID)
	if err != nil {
		setToastTrigger(w, "error", "Failed to save adjustment header.")
		http.Error(w, "Failed to save adjustment header", http.StatusInternalServerError)
		return
	}

	// Insert all item rows
	adjustments := stockAdj.ToInventoryAdjustments()
	if len(adjustments) > 0 {
		query := `
			INSERT INTO inventory_adjustments (adjustment_id, date, item_id, uom, adjustment_qty, cost, remarks)
			VALUES
		`
		var args []interface{}
		var placeholders []string

		for _, adj := range adjustments {
			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?)")
			args = append(args, adjustmentID, adj.Date, adj.ItemID, adj.UOM, adj.AdjustmentQty, adj.Cost, adj.Remarks)
		}

		query += strings.Join(placeholders, ", ")

		query = tx.Rebind(query)
		_, err = tx.Exec(query, args...)
		if err != nil {
			setToastTrigger(w, "error", "Failed to save adjustment items.")
			http.Error(w, "Failed to save adjustment items", http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		setToastTrigger(w, "error", "Failed to commit adjustment transaction.")
		http.Error(w, "Failed to commit adjustment transaction", http.StatusInternalServerError)
		return
	}

	setToastTrigger(w, "success", "Stock adjusted successfully!", "stock-adjusted")
	w.WriteHeader(http.StatusOK)
}

// NewAdjustmentRowHandler renders a single empty adjustment item row template
func (app *App) NewAdjustmentRowHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 20")
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
	var err error
	if itemID > 0 {
		err = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY (id = ?) DESC, description ASC LIMIT 20", itemID)
	} else {
		err = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 20")
	}
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
			SELECT u.code FROM (
				SELECT default_uom AS code FROM items WHERE id = ?
				UNION
				SELECT muom AS code FROM uom_settings WHERE item_id = ?
			) AS u WHERE u.code != '' AND u.code IS NOT NULL
			ORDER BY u.code ASC
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
			SELECT substr(doc_date, 1, 7) AS month, item_id, 0.0 AS received, qty AS sold, 0.0 AS adjusted FROM sales_details WHERE doc_status IN ('Posted', 'POSTED')
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

	query += " GROUP BY t.month, t.item_id, i.code, i.description, i.default_uom"

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
		log.Printf("ERROR: MonthlyInventoryHandler query failed: %v", err)
		http.Error(w, "Failed to load monthly inventory", http.StatusInternalServerError)
		return
	}

	// Get list of unique months for the filter dropdown
	var months []string
	errMonths := db.DB.Select(&months, `
		SELECT DISTINCT substr(date, 1, 7) as m FROM receiving_logs WHERE date IS NOT NULL
		UNION
		SELECT DISTINCT substr(doc_date, 1, 7) as m FROM sales_details WHERE doc_date IS NOT NULL AND doc_status IN ('Posted', 'POSTED')
		UNION
		SELECT DISTINCT substr(date, 1, 7) as m FROM inventory_adjustments WHERE date IS NOT NULL
		ORDER BY m DESC
	`)
	if errMonths != nil {
		log.Printf("ERROR: MonthlyInventoryHandler months query failed: %v", errMonths)
	}

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
		http.Error(w, "Failed to load receiving logs", http.StatusInternalServerError)
		return
	}

	var suppliers []string
	err = db.DB.Select(&suppliers, "SELECT DISTINCT supplier FROM receiving_logs WHERE supplier != '' ORDER BY supplier ASC")
	if err != nil {
		http.Error(w, "Failed to load suppliers", http.StatusInternalServerError)
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
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 25")
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
		http.Error(w, "Failed to load adjustment logs", http.StatusInternalServerError)
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
	_ = db.DB.Select(&adjItems, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 25")
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
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid receiving log ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM receiving_logs WHERE id = ?", id)
	if err != nil {
		setToastTrigger(w, "error", "Failed to delete receiving log.")
		http.Error(w, "Failed to delete receiving log", http.StatusInternalServerError)
		return
	}

	setToastTrigger(w, "success", "Receiving log deleted.")
	w.WriteHeader(http.StatusOK)
}

// EditReceivingLogHandler returns an inline edit form row for a receiving log
func (app *App) EditReceivingLogHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid receiving log ID", http.StatusBadRequest)
		return
	}

	var logRow models.ReceivingLogWithItem
	err = db.DB.Get(&logRow, `
		SELECT r.*, i.code as item_code, i.description as item_description
		FROM receiving_logs r
		JOIN items i ON r.item_id = i.id
		WHERE r.id = ?
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Receiving log not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load receiving log", http.StatusInternalServerError)
		return
	}

	app.Render(w, "receiving_log_edit_row.html", logRow)
}

// UpdateReceivingLogHandler updates a receiving log inline
func (app *App) UpdateReceivingLogHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "Invalid receiving log ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		setToastTrigger(w, "error", "Failed to parse form.")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	qty, errQty := strconv.ParseFloat(r.FormValue("qty"), 64)
	unitPrice, errPrice := strconv.ParseFloat(r.FormValue("unit_price"), 64)
	var less1, less2, markup float64
	var errL1, errL2, errMarkup error
	if s := strings.TrimSpace(r.FormValue("less1")); s != "" {
		less1, errL1 = strconv.ParseFloat(s, 64)
	}
	if s := strings.TrimSpace(r.FormValue("less2")); s != "" {
		less2, errL2 = strconv.ParseFloat(s, 64)
	}
	if s := strings.TrimSpace(r.FormValue("markup")); s != "" {
		markup, errMarkup = strconv.ParseFloat(s, 64)
	}
	remarks := r.FormValue("remarks")

	if errQty != nil || errPrice != nil || errL1 != nil || errL2 != nil || errMarkup != nil {
		setToastTrigger(w, "error", "Invalid numeric input.")
		http.Error(w, "Invalid numeric input", http.StatusBadRequest)
		return
	}

	var existing models.ReceivingLog
	if err := db.DB.Get(&existing, "SELECT id, supplier, date, pl_no, item_id, uom FROM receiving_logs WHERE id = ?", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Receiving log not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load receiving log", http.StatusInternalServerError)
		return
	}

	plNo := ""
	if existing.PLNo != nil {
		plNo = *existing.PLNo
	}

	// Pass cost=0 so domain.NewReceivingLog computes Cost = Round2(UnitPrice * (1 - Less1/100) * (1 - Less2/100))
	domLog, err := domain.NewReceivingLog(
		id,
		existing.Supplier,
		existing.Date,
		plNo,
		existing.ItemID,
		qty,
		existing.UOM,
		unitPrice,
		less1,
		less2,
		0,
		markup,
		remarks,
	)
	if err != nil {
		setToastTrigger(w, "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE receiving_logs
		SET unit_price = ?, less1 = ?, less2 = ?, cost = ?, total_cost = ?, markup = ?, selling_price = ?, remarks = ?, qty = ?
		WHERE id = ?
	`, domLog.UnitPrice, domLog.Less1, domLog.Less2, domLog.Cost, domLog.TotalCost, domLog.Markup, domLog.SellingPrice, domLog.Remarks, domLog.Qty, id)
	if err != nil {
		setToastTrigger(w, "error", "Failed to update receiving log.")
		http.Error(w, "Failed to update receiving log", http.StatusInternalServerError)
		return
	}

	var logRow models.ReceivingLogWithItem
	err = db.DB.Get(&logRow, `
		SELECT r.*, i.code as item_code, i.description as item_description
		FROM receiving_logs r
		JOIN items i ON r.item_id = i.id
		WHERE r.id = ?
	`, id)
	if err != nil {
		http.Error(w, "Failed to retrieve updated receiving log", http.StatusInternalServerError)
		return
	}

	setToastTrigger(w, "success", "Receiving log updated.")
	app.Render(w, "receiving_log_row.html", logRow)
}
