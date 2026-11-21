package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rav/db"
	"rav/domain"
	"rav/models"
)

func checkStockAvailableTx(tx *db.TxConn, itemID int, uom string, reqQty float64, excludeSaleID int) error {
	var recCount int
	if err := tx.Get(&recCount, "SELECT COUNT(*) FROM receiving_logs WHERE item_id = ?", itemID); err != nil {
		return fmt.Errorf("failed to verify item stock: %w", err)
	}
	var adjCount int
	if err := tx.Get(&adjCount, "SELECT COUNT(*) FROM inventory_adjustments WHERE item_id = ?", itemID); err != nil {
		return fmt.Errorf("failed to verify item adjustments: %w", err)
	}
	if recCount == 0 && adjCount == 0 {
		// Allow sales when item has no tracked stock movements yet
		return nil
	}

	var factor float64 = 1.0
	var defaultUOM string
	_ = tx.Get(&defaultUOM, "SELECT default_uom FROM items WHERE id = ?", itemID)
	if !strings.EqualFold(strings.TrimSpace(uom), strings.TrimSpace(defaultUOM)) {
		var uomFactor float64
		if err := tx.Get(&uomFactor, "SELECT conversion_factor FROM uom_settings WHERE item_id = ? AND LOWER(muom) = LOWER(?)", itemID, strings.TrimSpace(uom)); err == nil && uomFactor > 0 {
			factor = uomFactor
		}
	}
	reqBaseQty := reqQty * factor

	var totalReceived float64
	_ = tx.Get(&totalReceived, `
		SELECT COALESCE(SUM(r.qty * COALESCE(u.conversion_factor, 1.0)), 0.0)
		FROM receiving_logs r
		LEFT JOIN uom_settings u ON u.item_id = r.item_id AND LOWER(u.muom) = LOWER(r.uom)
		WHERE r.item_id = ?
	`, itemID)

	var totalSold float64
	_ = tx.Get(&totalSold, `
		SELECT COALESCE(SUM(s.qty * COALESCE(u.conversion_factor, 1.0)), 0.0)
		FROM sales_details s
		LEFT JOIN uom_settings u ON u.item_id = s.item_id AND LOWER(u.muom) = LOWER(s.uom)
		WHERE s.item_id = ? AND s.doc_status IN ('Posted', 'POSTED') AND s.id != ?
	`, itemID, excludeSaleID)

	var totalAdjusted float64
	_ = tx.Get(&totalAdjusted, `
		SELECT COALESCE(SUM(a.adjustment_qty * COALESCE(u.conversion_factor, 1.0)), 0.0)
		FROM inventory_adjustments a
		LEFT JOIN uom_settings u ON u.item_id = a.item_id AND LOWER(u.muom) = LOWER(a.uom)
		WHERE a.item_id = ?
	`, itemID)

	onHand := totalReceived - totalSold + totalAdjusted
	if reqBaseQty > onHand+1e-9 {
		return fmt.Errorf("insufficient stock available on hand (requested %.2f, available %.2f)", reqBaseQty, onHand)
	}
	return nil
}

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
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number,
		       COALESCE(s.customer_name, '') as customer_name,
		       COALESCE(s.tin_no, '') as tin_no,
		       COALESCE(s.address, '') as address,
		       s.supplier, s.item_id, s.qty,
		       COALESCE(s.uom, '') as uom,
		       s.price, s.total_sales, s.cost, s.total_cost,
		       COALESCE(s.patong, 0.0) as patong,
		       COALESCE(s.pos_charge, 0.0) as pos_charge,
		       COALESCE(s.wt_2307, 0.0) as wt_2307,
		       COALESCE(s.total_remit, 0.0) as total_remit,
		       COALESCE(s.profit, 0.0) as profit,
		       COALESCE(s.profit_margin, 0.0) as profit_margin,
		       COALESCE(s.remarks, '') as remarks,
		       i.code as item_code, i.description as item_description
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(s.doc_number LIKE ? OR s.customer_name LIKE ? OR s.tin_no LIKE ? OR s.address LIKE ? OR i.code LIKE ? OR i.description LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
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
		log.Printf("ERROR: SalesHandler failed to fetch sales: %v", err)
		http.Error(w, "Failed to load sales records", http.StatusInternalServerError)
		return
	}

	var items []models.Item
	if isEdit {
		_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 500")
	} else {
		_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 25")
	}

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	target := r.Header.Get("HX-Target")
	if r.Header.Get("HX-Request") == "true" && target != "main-content" && target != "#main-content" && target != "" {
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
				"Uoms":  []models.Uom{},
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
	tinNo := r.FormValue("tin_no")
	address := r.FormValue("address")
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
		var errP, errPOS, errWT error
		var remarks string
		if i < len(patongs) && strings.TrimSpace(patongs[i]) != "" {
			patong, errP = strconv.ParseFloat(strings.TrimSpace(patongs[i]), 64)
		}
		if i < len(posCharges) && strings.TrimSpace(posCharges[i]) != "" {
			posCharge, errPOS = strconv.ParseFloat(strings.TrimSpace(posCharges[i]), 64)
		}
		if i < len(wt2307s) && strings.TrimSpace(wt2307s[i]) != "" {
			wt2307, errWT = strconv.ParseFloat(strings.TrimSpace(wt2307s[i]), 64)
		}
		if i < len(remarkss) {
			remarks = remarkss[i]
		}

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || errP != nil || errPOS != nil || errWT != nil {
			setToastTrigger(w, "error", "Invalid numeric input in items.")
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
		setToastTrigger(w, "error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sale.SetCustomerDetails(tinNo, address)

	tx, err := db.DB.Beginx()
	if err != nil {
		setToastTrigger(w, "error", "Database transaction failed.")
		http.Error(w, "Database transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	details := sale.ToSalesDetails()
	if len(details) > 0 {
		// Check stock availability inside transaction before inserting
		for _, sd := range details {
			if err := checkStockAvailableTx(tx, sd.ItemID, sd.UOM, sd.Qty, 0); err != nil {
				setToastTrigger(w, "error", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		stmt, err := tx.Prepare(`
			INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, tin_no, address, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, patong, pos_charge, wt_2307, total_remit, profit, profit_margin, remarks, ref_pl)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			setToastTrigger(w, "error", "Failed to prepare sale detail statement.")
			http.Error(w, "Failed to prepare sale detail statement", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		for _, sd := range details {
			_, err = stmt.Exec(sd.DocType, sd.DocStatus, sd.DocDate, sd.DocNumber, sd.CustomerName, sd.TINNo, sd.Address, sd.Supplier, sd.ItemID, sd.Qty, sd.UOM, sd.Price, sd.TotalSales, sd.Cost, sd.TotalCost, sd.Patong, sd.POSCharge, sd.WT2307, sd.TotalRemit, sd.Profit, sd.ProfitMargin, sd.Remarks, sd.RefPL)
			if err != nil {
				setToastTrigger(w, "error", "Failed to save sale detail.")
				http.Error(w, "Failed to save sale detail", http.StatusInternalServerError)
				return
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		setToastTrigger(w, "error", "Failed to commit sale transaction.")
		http.Error(w, "Failed to commit sale transaction", http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "CREATE_SALE", "Sale", docNumber, "Logged "+docType+" #"+docNumber+" ("+customerName+") with "+strconv.Itoa(len(saleItems))+" items")
	setToastTrigger(w, "success", "Sales logged successfully!", "sales-added")
	w.WriteHeader(http.StatusOK)
}

// NewSalesPageHandler redirects to sales list page
func (app *App) NewSalesPageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/sales", http.StatusSeeOther)
}

// NewSaleRowHandler renders a single empty sale item row template
func (app *App) NewSaleRowHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 20")
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

	app.LogActivity(r, "DELETE_SALE", "Sale", idStr, "Deleted sales record ID "+idStr)
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Sales record deleted successfully!"}}`)
	w.WriteHeader(http.StatusOK)
}

// SaleItemRowDetailsHandler renders a single sale item row template populated with item defaults
func (app *App) SaleItemRowDetailsHandler(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.URL.Query().Get("item_id")
	itemID, _ := strconv.Atoi(itemIDStr)
	supplier := strings.TrimSpace(r.URL.Query().Get("supplier"))

	var items []models.Item
	var err error
	if itemID > 0 {
		err = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY (id = ?) DESC, description ASC LIMIT 20", itemID)
	} else {
		err = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 20")
	}
	if err != nil {
		http.Error(w, "Failed to load items", http.StatusInternalServerError)
		return
	}

	var defaultUOM string
	var lastCost, lastPrice float64
	var lastPLNo string

	if itemID > 0 {
		_ = db.DB.Get(&defaultUOM, "SELECT default_uom FROM items WHERE id = ?", itemID)

		type recBatchRow struct {
			ID           int       `db:"id"`
			Date         time.Time `db:"date"`
			Qty          float64   `db:"qty"`
			Cost         float64   `db:"cost"`
			SellingPrice float64   `db:"selling_price"`
			PLNo         string    `db:"pl_no"`
			Factor       float64   `db:"factor"`
		}
		var recRows []recBatchRow
		var recErr error
		if supplier != "" {
			recErr = db.DB.Select(&recRows, `
				SELECT r.id, r.date, r.qty, COALESCE(r.cost, 0.0) AS cost, COALESCE(r.selling_price, 0.0) AS selling_price,
				       COALESCE(r.pl_no, '') AS pl_no, COALESCE(u.conversion_factor, 1.0) AS factor
				FROM receiving_logs r
				LEFT JOIN uom_settings u ON u.item_id = r.item_id AND LOWER(u.muom) = LOWER(r.uom)
				WHERE r.item_id = ? AND LOWER(r.supplier) = LOWER(?)
				ORDER BY r.date ASC, r.id ASC
			`, itemID, supplier)
		} else {
			recErr = db.DB.Select(&recRows, `
				SELECT r.id, r.date, r.qty, COALESCE(r.cost, 0.0) AS cost, COALESCE(r.selling_price, 0.0) AS selling_price,
				       COALESCE(r.pl_no, '') AS pl_no, COALESCE(u.conversion_factor, 1.0) AS factor
				FROM receiving_logs r
				LEFT JOIN uom_settings u ON u.item_id = r.item_id AND LOWER(u.muom) = LOWER(r.uom)
				WHERE r.item_id = ?
				ORDER BY r.date ASC, r.id ASC
			`, itemID)
		}

		if recErr == nil && len(recRows) > 0 {
			var totalSold float64
			if supplier != "" {
				_ = db.DB.Get(&totalSold, `
					SELECT COALESCE(SUM(s.qty * COALESCE(u.conversion_factor, 1.0)), 0.0)
					FROM sales_details s
					LEFT JOIN uom_settings u ON u.item_id = s.item_id AND LOWER(u.muom) = LOWER(s.uom)
					WHERE s.item_id = ? AND s.doc_status IN ('Posted', 'POSTED') AND LOWER(s.supplier) = LOWER(?)
				`, itemID, supplier)
			} else {
				_ = db.DB.Get(&totalSold, `
					SELECT COALESCE(SUM(s.qty * COALESCE(u.conversion_factor, 1.0)), 0.0)
					FROM sales_details s
					LEFT JOIN uom_settings u ON u.item_id = s.item_id AND LOWER(u.muom) = LOWER(s.uom)
					WHERE s.item_id = ? AND s.doc_status IN ('Posted', 'POSTED')
				`, itemID)
			}

			var totalAdjusted, negAdjusted float64
			if supplier == "" {
				_ = db.DB.Get(&totalAdjusted, `
					SELECT COALESCE(SUM(a.adjustment_qty * COALESCE(u.conversion_factor, 1.0)), 0.0)
					FROM inventory_adjustments a
					LEFT JOIN uom_settings u ON u.item_id = a.item_id AND LOWER(u.muom) = LOWER(a.uom)
					WHERE a.item_id = ?
				`, itemID)
				_ = db.DB.Get(&negAdjusted, `
					SELECT COALESCE(SUM(-a.adjustment_qty * COALESCE(u.conversion_factor, 1.0)), 0.0)
					FROM inventory_adjustments a
					LEFT JOIN uom_settings u ON u.item_id = a.item_id AND LOWER(u.muom) = LOWER(a.uom)
					WHERE a.item_id = ? AND a.adjustment_qty < 0
				`, itemID)
			}

			totalReceived := 0.0
			for _, rl := range recRows {
				f := rl.Factor
				if f <= 0 {
					f = 1.0
				}
				totalReceived += rl.Qty * f
			}

			onHand := totalReceived - totalSold + totalAdjusted
			found := false
			if onHand > 1e-9 {
				totalConsumed := totalSold + negAdjusted
				cumReceived := 0.0
				for _, rl := range recRows {
					f := rl.Factor
					if f <= 0 {
						f = 1.0
					}
					cumReceived += rl.Qty * f
					if cumReceived > totalConsumed+1e-9 {
						lastCost = domain.Round2(rl.Cost / f)
						lastPrice = domain.Round2(rl.SellingPrice / f)
						lastPLNo = rl.PLNo
						found = true
						break
					}
				}
			}

			if !found {
				lastRec := recRows[len(recRows)-1]
				f := lastRec.Factor
				if f <= 0 {
					f = 1.0
				}
				lastCost = domain.Round2(lastRec.Cost / f)
				lastPrice = domain.Round2(lastRec.SellingPrice / f)
				lastPLNo = lastRec.PLNo
			}
		}
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
		setToastTrigger(w, "error", "Invalid sale log ID.")
		http.Error(w, "Invalid sale log ID", http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
	if err != nil {
		setToastTrigger(w, "error", "Failed to parse form.")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
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
	tinNo := r.FormValue("tin_no")
	address := r.FormValue("address")
	supplier := r.FormValue("supplier")
	itemIDStr := r.FormValue("item_id")
	qtyStr := r.FormValue("qty")
	uom := r.FormValue("uom")
	priceStr := r.FormValue("price")
	costStr := r.FormValue("cost")
	patongStr := strings.TrimSpace(r.FormValue("patong"))
	posChargeStr := strings.TrimSpace(r.FormValue("pos_charge"))
	wt2307Str := strings.TrimSpace(r.FormValue("wt_2307"))
	remarks := r.FormValue("remarks")
	refPL := r.FormValue("ref_pl")

	itemID, errItem := strconv.Atoi(itemIDStr)
	qty, errQty := strconv.ParseFloat(qtyStr, 64)
	price, errPrice := strconv.ParseFloat(priceStr, 64)
	cost, errCost := strconv.ParseFloat(costStr, 64)
	var patong, posCharge, wt2307 float64
	var errP, errPOS, errWT error
	if patongStr != "" {
		patong, errP = strconv.ParseFloat(patongStr, 64)
	}
	if posChargeStr != "" {
		posCharge, errPOS = strconv.ParseFloat(posChargeStr, 64)
	}
	if wt2307Str != "" {
		wt2307, errWT = strconv.ParseFloat(wt2307Str, 64)
	}

	parsedDate, dateErr := time.Parse("2006-01-02", docDateStr)
	if dateErr != nil || errItem != nil || errQty != nil || errPrice != nil || errCost != nil || errP != nil || errPOS != nil || errWT != nil {
		setToastTrigger(w, "error", "Invalid numeric or date input.")
		http.Error(w, "Invalid numeric or date input", http.StatusBadRequest)
		return
	}

	salesDetail, err := domain.NewSalesDetail(id, docType, docStatus, parsedDate, docNumber, customerName, supplier, itemID, qty, uom, price, cost, patong, posCharge, wt2307, remarks, refPL)
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

	if strings.EqualFold(salesDetail.DocStatus, "Posted") {
		if err := checkStockAvailableTx(tx, salesDetail.ItemID, salesDetail.UOM, salesDetail.Qty, id); err != nil {
			setToastTrigger(w, "error", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	_, err = tx.Exec(`
		UPDATE sales_details 
		SET doc_type = ?, doc_status = ?, doc_date = ?, doc_number = ?, customer_name = ?, tin_no = ?, address = ?, supplier = ?, item_id = ?, qty = ?, uom = ?, price = ?, total_sales = ?, cost = ?, total_cost = ?, patong = ?, pos_charge = ?, wt_2307 = ?, total_remit = ?, profit = ?, profit_margin = ?, remarks = ?
		WHERE id = ?
	`, salesDetail.DocType, salesDetail.DocStatus, salesDetail.DocDate, salesDetail.DocNumber, salesDetail.CustomerName, tinNo, address, salesDetail.Supplier, salesDetail.ItemID, salesDetail.Qty, salesDetail.UOM, salesDetail.Price, salesDetail.TotalSales, salesDetail.Cost, salesDetail.TotalCost, salesDetail.Patong, salesDetail.POSCharge, salesDetail.WT2307, salesDetail.TotalRemit, salesDetail.Profit, salesDetail.ProfitMargin, salesDetail.Remarks, id)

	if err != nil {
		setToastTrigger(w, "error", "Failed to update sale log.")
		http.Error(w, "Failed to update sale log", http.StatusInternalServerError)
		return
	}

	var updatedSale models.SalesDetailWithItem
	err = tx.Get(&updatedSale, `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.tin_no, s.address, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost,
		       s.patong, s.pos_charge, s.wt_2307, s.total_remit, s.profit, s.profit_margin, s.remarks,
		       i.code as item_code, i.description as item_description
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
		WHERE s.id = ?
	`, id)

	if err != nil {
		setToastTrigger(w, "error", "Failed to retrieve updated sale record.")
		http.Error(w, "Failed to retrieve updated sale record", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		setToastTrigger(w, "error", "Failed to commit sale update.")
		http.Error(w, "Failed to commit sale update", http.StatusInternalServerError)
		return
	}

	setToastTrigger(w, "success", "Sale log updated successfully!")

	isEdit := r.FormValue("is_edit") == "1"
	if isEdit {
		var items []models.Item
		_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 500")
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
		setToastTrigger(w, "error", "Invalid sale log ID.")
		http.Error(w, "Invalid sale log ID", http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
	if err != nil {
		setToastTrigger(w, "error", "Failed to parse form.")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	docStatus := r.FormValue("doc_status")
	if docStatus == "" {
		setToastTrigger(w, "error", "Status cannot be empty.")
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
		setToastTrigger(w, "error", "Invalid status value.")
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE sales_details 
		SET doc_status = ?
		WHERE id = ?
	`, docStatus, id)

	if err != nil {
		setToastTrigger(w, "error", "Failed to update sale status.")
		http.Error(w, "Failed to update sale status", http.StatusInternalServerError)
		return
	}

	var updatedSale models.SalesDetailWithItem
	err = db.DB.Get(&updatedSale, `
		SELECT s.id, s.doc_type, s.doc_status, s.doc_date, s.doc_number, s.customer_name, s.tin_no, s.address, s.supplier,
		       s.item_id, s.qty, s.uom, s.price, s.total_sales, s.cost, s.total_cost,
		       s.patong, s.pos_charge, s.wt_2307, s.total_remit, s.profit, s.profit_margin, s.remarks,
		       i.code as item_code, i.description as item_description
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
		WHERE s.id = ?
	`, id)

	if err != nil {
		setToastTrigger(w, "error", "Failed to retrieve updated sale record.")
		http.Error(w, "Failed to retrieve updated sale record", http.StatusInternalServerError)
		return
	}

	setToastTrigger(w, "success", "Status updated to "+docStatus+"!")

	isEdit := r.FormValue("is_edit") == "1"
	if isEdit {
		var items []models.Item
		_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC LIMIT 500")
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

