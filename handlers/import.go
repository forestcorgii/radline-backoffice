package handlers

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"radline/db"

	"github.com/xuri/excelize/v2"
)

// ImportPageHandler renders the import data page
func (app *App) ImportPageHandler(w http.ResponseWriter, r *http.Request) {
	app.RenderPage(w, r, "import.html", nil)
}

// ImportUploadHandler handles the Excel file upload and runs the database import inside a transaction
func (app *App) ImportUploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(50 << 20) // Limit upload to 50MB
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to parse form."}}`)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "No file uploaded."}}`)
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse checkbox options
	importBrands := r.FormValue("import_brands") == "true"
	importCategories := r.FormValue("import_categories") == "true"
	importItems := r.FormValue("import_items") == "true"
	importUoms := r.FormValue("import_uoms") == "true"
	importReceiving := r.FormValue("import_receiving") == "true"
	importSales := r.FormValue("import_sales") == "true"
	importAdjustments := r.FormValue("import_adjustments") == "true"
	clearExisting := r.FormValue("clear_existing") == "true"

	// Create a temp file to open with excelize (since excelize.OpenReader needs an io.Reader, but opening file directly can sometimes be cleaner for cell formats)
	tempFile, err := os.CreateTemp("", "radline_upload_*.xlsx")
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to create temp file."}}`)
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to save temp file."}}`)
		http.Error(w, "Failed to save temp file", http.StatusInternalServerError)
		return
	}

	// Open the Excel file (read-only mode is not directly a function flag, but we parse rows using iterator for speed)
	f, err := excelize.OpenFile(tempFile.Name())
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to open Excel file. Make sure it is a valid .xlsx file."}}`)
		http.Error(w, fmt.Sprintf("Failed to open excel: %v", err), http.StatusBadRequest)
		return
	}
	defer f.Close()

	// Begin Transaction
	tx, err := db.DB.Beginx()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to start database transaction."}}`)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// If clear existing is checked, execute deletes in correct order
	if clearExisting {
		tablesToClear := []string{
			"inventory_adjustments",
			"stock_adjustments",
			"sales_details",
			"receiving_logs",
			"uom_settings",
			"items",
			"brands",
			"categories",
		}
		for _, tbl := range tablesToClear {
			_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s", tbl))
			if err != nil {
				w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "Failed to clear table %s."}}`, tbl))
				http.Error(w, fmt.Sprintf("Failed to clear table %s: %v", tbl, err), http.StatusInternalServerError)
				return
			}
			// Reset sqlite sequence
			_, _ = tx.Exec("DELETE FROM sqlite_sequence WHERE name = ?", tbl)
		}
	}

	// Track imported counts for summary
	summary := struct {
		Brands      int
		Categories  int
		Items       int
		Uoms        int
		Receiving   int
		Sales       int
		Adjustments int
	}{}

	// 1. BRAND IMPORT
	if importBrands {
		rows, err := f.Rows("BRAND")
		if err == nil {
			defer rows.Close()
			colMap, errFind := findHeaderRowAndMap(rows, "BRAND CODE")
			if errFind == nil {
				for rows.Next() {
					row, errRow := rows.Columns()
					if errRow != nil {
						continue
					}
					code := getValByHeader(row, colMap, "BRAND CODE")
					name := getValByHeader(row, colMap, "BRAND")
					if code == "" || name == "" {
						continue
					}
					_, err = tx.Exec("INSERT OR IGNORE INTO brands (code, name) VALUES (?, ?)", code, name)
					if err == nil {
						summary.Brands++
					}
				}
			}
		}
	}

	// 2. CATEGORY IMPORT
	if importCategories {
		rows, err := f.Rows("CATEGORY")
		if err == nil {
			defer rows.Close()
			colMap, errFind := findHeaderRowAndMap(rows, "CATEGORY CODE")
			if errFind == nil {
				for rows.Next() {
					row, errRow := rows.Columns()
					if errRow != nil {
						continue
					}
					code := getValByHeader(row, colMap, "CATEGORY CODE")
					name := getValByHeader(row, colMap, "CATEGORY")
					if code == "" || name == "" {
						continue
					}
					_, err = tx.Exec("INSERT OR IGNORE INTO categories (code, name) VALUES (?, ?)", code, name)
					if err == nil {
						summary.Categories++
					}
				}
			}
		}
	}

	// Caches for fast lookups
	brandCodeToID := make(map[string]int)
	categoryCodeToID := make(map[string]int)
	itemCodeToID := make(map[string]int)

	// Helper to load caches
	loadCaches := func() error {
		// Brands
		type CodeID struct {
			ID   int    `db:"id"`
			Code string `db:"code"`
		}
		var brands []CodeID
		err = tx.Select(&brands, "SELECT id, code FROM brands")
		if err != nil {
			return err
		}
		for _, b := range brands {
			brandCodeToID[strings.ToLower(b.Code)] = b.ID
		}

		// Categories
		var categories []CodeID
		err = tx.Select(&categories, "SELECT id, code FROM categories")
		if err != nil {
			return err
		}
		for _, c := range categories {
			categoryCodeToID[strings.ToLower(c.Code)] = c.ID
		}

		// Items
		var items []CodeID
		err = tx.Select(&items, "SELECT id, code FROM items")
		if err != nil {
			return err
		}
		for _, i := range items {
			itemCodeToID[strings.ToLower(i.Code)] = i.ID
		}
		return nil
	}

	// Load caches initial state
	if err = loadCaches(); err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to cache database keys."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. ITEM LIST IMPORT
	if importItems {
		rows, err := f.Rows("ITEM LIST")
		if err == nil {
			defer rows.Close()
			colMap, errFind := findHeaderRowAndMap(rows, "ITEM CODE")
			if errFind == nil {
				for rows.Next() {
					row, errRow := rows.Columns()
					if errRow != nil {
						continue
					}
					code := getValByHeader(row, colMap, "ITEM CODE")
					desc := getValByHeader(row, colMap, "ITEM DESCRIPTION")
					defaultUom := getValByHeader(row, colMap, "DEFAULT UOM")
					if code == "" || desc == "" || defaultUom == "" {
						continue
					}
					model := getValByHeader(row, colMap, "MODEL")
					brandCode := getValByHeader(row, colMap, "BRAND CODE")
					catCode := getValByHeader(row, colMap, "CATEGORY CODE")
					variation := getValByHeader(row, colMap, "VARIATION")
					remarks := getValByHeader(row, colMap, "REMARKS")

					var brandID *int
					if brandCode != "" {
						if id, ok := brandCodeToID[strings.ToLower(brandCode)]; ok {
							brandID = &id
						}
					}
					var categoryID *int
					if catCode != "" {
						if id, ok := categoryCodeToID[strings.ToLower(catCode)]; ok {
							categoryID = &id
						}
					}

					_, err = tx.Exec(`
						INSERT OR IGNORE INTO items (code, description, default_uom, model, brand_id, category_id, variation, remarks)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					`, code, desc, defaultUom, model, brandID, categoryID, variation, remarks)
					if err == nil {
						summary.Items++
					}
				}
				// Reload item cache to include newly imported items
				if err = loadCaches(); err != nil {
					w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update item cache."}}`)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// 4. UOM SETTINGS IMPORT
	if importUoms {
		rows, err := f.Rows("UOM SETTINGS")
		if err == nil {
			defer rows.Close()
			colMap, errFind := findHeaderRowAndMap(rows, "ITEM CODE")
			if errFind == nil {
				for rows.Next() {
					row, errRow := rows.Columns()
					if errRow != nil {
						continue
					}
					itemCode := getValByHeader(row, colMap, "ITEM CODE")
					defaultUom := getValByHeader(row, colMap, "DEFAULT UOM")
					muom := getValByHeader(row, colMap, "MUOM")
					convStr := getValByHeader(row, colMap, "CONVERSION")

					if itemCode == "" || muom == "" || convStr == "" {
						continue
					}
					if strings.ToLower(muom) == strings.ToLower(defaultUom) {
						// Skip base UOM setting
						continue
					}
					factor := parseExcelFloat(convStr)
					if factor <= 0 {
						continue
					}

					itemID, ok := itemCodeToID[strings.ToLower(itemCode)]
					if !ok {
						continue
					}

					_, err = tx.Exec(`
						INSERT INTO uom_settings (item_id, muom, conversion_factor)
						VALUES (?, ?, ?)
					`, itemID, muom, factor)
					if err == nil {
						summary.Uoms++
					}
				}
			}
		}
	}

	// 5. RECEIVING IMPORT
	if importReceiving {
		rows, err := f.Rows("RECEIVING")
		if err == nil {
			defer rows.Close()
			colMap, errFind := findHeaderRowAndMap(rows, "SUPPLIER")
			if errFind == nil {
				for rows.Next() {
					row, errRow := rows.Columns()
					if errRow != nil {
						continue
					}
					supplier := getValByHeader(row, colMap, "SUPPLIER")
					dateStr := getValByHeader(row, colMap, "DATE")
					itemCode := getValByHeader(row, colMap, "ITEM CODE")
					qtyStr := getValByHeader(row, colMap, "QTY")
					uom := getValByHeader(row, colMap, "UOM")

					if supplier == "" || dateStr == "" || itemCode == "" || qtyStr == "" || uom == "" {
						continue
					}

					itemID, ok := itemCodeToID[strings.ToLower(itemCode)]
					if !ok {
						continue
					}

					date, errDate := parseExcelDate(dateStr)
					if errDate != nil {
						continue
					}

					qty := parseExcelFloat(qtyStr)
					plNo := getValByHeader(row, colMap, "PL NO.")
					unitPrice := parseExcelFloat(getValByHeader(row, colMap, "UNIT PRICE "))
					cost := parseExcelFloat(getValByHeader(row, colMap, "COST"))
					totalCost := parseExcelFloat(getValByHeader(row, colMap, "TOTAL COST"))
					sellingPrice := parseExcelFloat(getValByHeader(row, colMap, "SELLING PRICE"))

					// Calculate defaults if missing
					if cost <= 0 && unitPrice > 0 {
						cost = unitPrice
					}
					if totalCost <= 0 {
						totalCost = qty * cost
					}

					_, err = tx.Exec(`
						INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, cost, total_cost, selling_price)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, supplier, date, plNo, itemID, qty, uom, unitPrice, cost, totalCost, sellingPrice)
					if err == nil {
						summary.Receiving++
					}
				}
			}
		}
	}

	// 6. SALES DETAIL IMPORT
	if importSales {
		rows, err := f.Rows("SALES DETAIL")
		if err == nil {
			defer rows.Close()
			colMap, errFind := findHeaderRowAndMap(rows, "DOC TYPE")
			if errFind == nil {
				for rows.Next() {
					row, errRow := rows.Columns()
					if errRow != nil {
						continue
					}
					docType := getValByHeader(row, colMap, "DOC TYPE")
					dateStr := getValByHeader(row, colMap, "DOC DATE")
					itemCode := getValByHeader(row, colMap, "ITEM CODE")
					qtyStr := getValByHeader(row, colMap, "QTY")
					uom := getValByHeader(row, colMap, "UOM")

					if docType == "" || dateStr == "" || itemCode == "" || qtyStr == "" || uom == "" {
						continue
					}

					itemID, ok := itemCodeToID[strings.ToLower(itemCode)]
					if !ok {
						continue
					}

					date, errDate := parseExcelDate(dateStr)
					if errDate != nil {
						continue
					}

					docStatus := getValByHeader(row, colMap, "DOC STATUS")
					if docStatus == "" {
						docStatus = "POSTED"
					}
					docNumber := getValByHeader(row, colMap, "DOC NUMBER")
					customerName := getValByHeader(row, colMap, "CUSTOMER NAME")
					supplier := getValByHeader(row, colMap, "SUPPLIER")
					qty := parseExcelFloat(qtyStr)
					price := parseExcelFloat(getValByHeader(row, colMap, "PRICE"))
					totalSales := parseExcelFloat(getValByHeader(row, colMap, "TOTAL SALES"))
					if totalSales <= 0 {
						totalSales = parseExcelFloat(getValByHeader(row, colMap, "TOTAL PRICE"))
					}
					cost := parseExcelFloat(getValByHeader(row, colMap, "COST"))
					totalCost := parseExcelFloat(getValByHeader(row, colMap, "TOTAL COST"))
					profit := parseExcelFloat(getValByHeader(row, colMap, "PROFIT"))

					patong := parseExcelFloat(getValByHeader(row, colMap, "PATONG"))
					posCharge := parseExcelFloat(getValByHeader(row, colMap, "POS CHARGE"))
					if posCharge == 0 {
						posCharge = parseExcelFloat(getValByHeader(row, colMap, "POS_CHARGE"))
					}
					wt2307 := parseExcelFloat(getValByHeader(row, colMap, "WT 2307"))
					if wt2307 == 0 {
						wt2307 = parseExcelFloat(getValByHeader(row, colMap, "WT_2307"))
					}
					totalRemit := parseExcelFloat(getValByHeader(row, colMap, "TOTAL REMIT"))
					if totalRemit == 0 {
						totalRemit = parseExcelFloat(getValByHeader(row, colMap, "TOTAL_REMIT"))
					}
					profitMargin := parseExcelFloat(getValByHeader(row, colMap, "PROFIT MARGIN"))
					if profitMargin == 0 {
						profitMargin = parseExcelFloat(getValByHeader(row, colMap, "PROFIT_MARGIN"))
					}
					remarks := getValByHeader(row, colMap, "REMARKS")
					refPL := getValByHeader(row, colMap, "REF PL")
					if refPL == "" {
						refPL = getValByHeader(row, colMap, "PL NO.")
					}

					var refPLVal *string
					if refPL != "" {
						refPLVal = &refPL
					}

					// Calculate defaults if missing
					if totalSales <= 0 {
						totalSales = qty * price
					}
					if totalCost <= 0 {
						totalCost = qty * cost
					}
					if profit == 0 && totalSales > 0 {
						profit = totalSales - totalCost
					}
					if totalRemit == 0 && totalSales > 0 {
						totalRemit = totalSales - (patong + posCharge + wt2307)
					}
					if profitMargin == 0 && totalSales > 0 {
						profitMargin = (profit / totalSales) * 100
					}

					_, err = tx.Exec(`
						INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, customer_name, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, patong, pos_charge, wt_2307, total_remit, profit, profit_margin, remarks, ref_pl)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, docType, docStatus, date, docNumber, customerName, supplier, itemID, qty, uom, price, totalSales, cost, totalCost, patong, posCharge, wt2307, totalRemit, profit, profitMargin, remarks, refPLVal)
					if err == nil {
						summary.Sales++
					}
				}
			}
		}
	}

	// 7. INV. ADJUSTMENT LOG IMPORT
	if importAdjustments {
		rows, err := f.Rows("INV. ADJUSTMENT LOG")
		if err == nil {
			defer rows.Close()
			colMap, errFind := findHeaderRowAndMap(rows, "DATE")
			if errFind == nil {
				for rows.Next() {
					row, errRow := rows.Columns()
					if errRow != nil {
						continue
					}
					dateStr := getValByHeader(row, colMap, "DATE")
					itemCode := getValByHeader(row, colMap, "ITEM CODE")
					qtyStr := getValByHeader(row, colMap, "ADJUSTMENT")
					uom := getValByHeader(row, colMap, "UOM")

					if dateStr == "" || itemCode == "" || qtyStr == "" || uom == "" {
						continue
					}

					itemID, ok := itemCodeToID[strings.ToLower(itemCode)]
					if !ok {
						continue
					}

					date, errDate := parseExcelDate(dateStr)
					if errDate != nil {
						continue
					}

					qty := parseExcelFloat(qtyStr)
					cost := parseExcelFloat(getValByHeader(row, colMap, "COST"))
					remarks := getValByHeader(row, colMap, "REMARKS")
					plNo := getValByHeader(row, colMap, "PL NO.")

					// Create a stock_adjustments header for each row to safely store the date and remarks
					adjRemark := remarks
					if plNo != "" && plNo != "0" {
						adjRemark = fmt.Sprintf("%s (PL: %s)", remarks, plNo)
					}
					if adjRemark == "" {
						adjRemark = "Excel Imported Adjustment"
					}

					res, errHeader := tx.Exec(`
						INSERT INTO stock_adjustments (date, remarks)
						VALUES (?, ?)
					`, date, adjRemark)
					if errHeader != nil {
						continue
					}

					headerID, errID := res.LastInsertId()
					if errID != nil {
						continue
					}

					_, err = tx.Exec(`
						INSERT INTO inventory_adjustments (adjustment_id, date, item_id, uom, adjustment_qty, cost, remarks)
						VALUES (?, ?, ?, ?, ?, ?, ?)
					`, headerID, date, itemID, uom, qty, cost, remarks)
					if err == nil {
						summary.Adjustments++
					}
				}
			}
		}
	}

	// Commit Transaction!
	err = tx.Commit()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to commit database transaction."}}`)
		http.Error(w, "Database error on commit", http.StatusInternalServerError)
		return
	}

	// Successful Import! Set success toast
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Data imported successfully!"}}`)
	w.WriteHeader(http.StatusOK)

	// Render beautiful summary result
	fmt.Fprintf(w, `
		<div class="card" style="border-left: 4px solid var(--primary-color); background: var(--surface-color); padding: 1rem; border-radius: 6px;">
			<h3 style="color: var(--primary-color); margin-top: 0; margin-bottom: 0.75rem;">Import Completed Successfully</h3>
			<ul style="margin: 0; padding-left: 1.25rem; font-size: 0.875rem; display: flex; flex-direction: column; gap: 0.35rem;">
				<li><strong>Brands:</strong> Imported %d records</li>
				<li><strong>Categories:</strong> Imported %d records</li>
				<li><strong>Items:</strong> Imported %d records</li>
				<li><strong>UOM Settings:</strong> Imported %d conversion configurations</li>
				<li><strong>Receiving Logs:</strong> Imported %d records</li>
				<li><strong>Sales Details:</strong> Imported %d records</li>
				<li><strong>Stock Adjustments:</strong> Imported %d items</li>
			</ul>
		</div>
	`, summary.Brands, summary.Categories, summary.Items, summary.Uoms, summary.Receiving, summary.Sales, summary.Adjustments)
}

// Helper: Find Header Row and Map its columns
func findHeaderRowAndMap(rows *excelize.Rows, headerKeyword string) (map[string]int, error) {
	colMap := make(map[string]int)
	foundHeader := false
	for rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		// Check if this row is the header
		isHeader := false
		for _, cell := range row {
			if strings.TrimSpace(strings.ToUpper(cell)) == strings.ToUpper(headerKeyword) {
				isHeader = true
				break
			}
		}
		if isHeader {
			for idx, colName := range row {
				cleanName := strings.ToLower(strings.TrimSpace(colName))
				colMap[cleanName] = idx
			}
			foundHeader = true
			break
		}
	}
	if !foundHeader {
		return nil, fmt.Errorf("header containing '%s' not found", headerKeyword)
	}
	return colMap, nil
}

// Helper: Safely access slice values
func getRowCell(row []string, index int) string {
	if index < len(row) {
		return strings.TrimSpace(row[index])
	}
	return ""
}

// Helper: Fetch row values mapped by column header name
func getValByHeader(row []string, colMap map[string]int, headerName string) string {
	idx, ok := colMap[strings.ToLower(headerName)]
	if !ok {
		return ""
	}
	return getRowCell(row, idx)
}

// Helper: Robust float parsing
func parseExcelFloat(val string) float64 {
	val = strings.ReplaceAll(val, ",", "")
	val = strings.ReplaceAll(val, "$", "")
	val = strings.ReplaceAll(val, "₱", "")
	val = strings.TrimSpace(val)
	if val == "" || val == "-" || strings.ToLower(val) == "none" || strings.ToLower(val) == "null" {
		return 0.0
	}
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

// Helper: Robust date parsing supporting float serial numbers and string formats
func parseExcelDate(val string) (time.Time, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	// Try serial float parsing first
	days, err := strconv.ParseFloat(val, 64)
	if err == nil {
		// Excel date epoch is Dec 30, 1899
		excelEpoch := time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)
		intDays := int(math.Floor(days))
		frac := days - float64(intDays)

		t := excelEpoch.AddDate(0, 0, intDays)
		// Add fraction of day in nanoseconds
		ns := frac * float64(time.Hour) * 24
		t = t.Add(time.Duration(ns))
		return t, nil
	}

	// Standard text formats
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"01-02-06",
		"01/02/06",
		"1/2/2006",
		"01/02/2006",
		"01-02-2006",
		"2006/01/02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"02-Jan-06",
		"2-Jan-06",
		"02-Jan-2006",
		"2-Jan-2006",
	}

	for _, fmtStr := range formats {
		t, err := time.Parse(fmtStr, val)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", val)
}
