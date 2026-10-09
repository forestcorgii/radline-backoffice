package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"radline/db"
	"radline/domain"
	"radline/models"
)

// QuotationFormHandler renders the printable Quotation Form page
func (app *App) QuotationFormHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	now := time.Now()
	defaultQuoteNo := fmt.Sprintf("QT-%s-%03d", now.Format("060102"), now.Minute()*60+now.Second())

	user := GetCurrentUser(r)
	preparedBy := ""
	if user != nil {
		preparedBy = user.User.FullName
	}

	app.RenderPage(w, r, "form_quotation.html", map[string]interface{}{
		"Items":          items,
		"Uoms":           uoms,
		"DefaultQuoteNo": defaultQuoteNo,
		"CurrentDate":    now.Format("2006-01-02"),
		"PreparedBy":     preparedBy,
	})
}

// PurchaseOrderFormHandler renders the Purchase Order Form page
func (app *App) PurchaseOrderFormHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	now := time.Now()
	defaultPONo := fmt.Sprintf("PO-%s-%03d", now.Format("060102"), now.Minute()*60+now.Second())

	user := GetCurrentUser(r)
	preparedBy := ""
	if user != nil {
		preparedBy = user.User.FullName
	}

	app.RenderPage(w, r, "form_purchase_order.html", map[string]interface{}{
		"Items":       items,
		"Uoms":        uoms,
		"DefaultPONo": defaultPONo,
		"CurrentDate": now.Format("2006-01-02"),
		"PreparedBy":  preparedBy,
	})
}

// SavePurchaseOrderHandler validates and saves a Purchase Order to the database
func (app *App) SavePurchaseOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to parse form data."}}`)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	poNo := strings.TrimSpace(r.FormValue("po_no"))
	if poNo == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Purchase order number is required."}}`)
		http.Error(w, "Purchase order number is required", http.StatusBadRequest)
		return
	}

	dateStr := strings.TrimSpace(r.FormValue("date"))
	poDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		poDate = time.Now()
	}

	var dueDate *time.Time
	dueDateStr := strings.TrimSpace(r.FormValue("due_date"))
	if dueDateStr != "" {
		if parsedDue, err := time.Parse("2006-01-02", dueDateStr); err == nil {
			dueDate = &parsedDue
		}
	}

	vendorName := strings.TrimSpace(r.FormValue("vendor_name"))
	if vendorName == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Vendor / Supplier name is required."}}`)
		http.Error(w, "Vendor name is required", http.StatusBadRequest)
		return
	}

	freight, _ := strconv.ParseFloat(r.FormValue("freight"), 64)
	applyVAT := r.FormValue("apply_vat") == "true" || r.FormValue("apply_vat") == "on" || r.FormValue("apply_vat") == "1"

	itemCodes := r.Form["item_code"]
	itemDescs := r.Form["description"]
	itemUoms := r.Form["uom"]
	itemQtys := r.Form["qty"]
	itemCosts := r.Form["unit_cost"]

	var poItems []domain.PurchaseOrderItem
	for i := 0; i < len(itemDescs); i++ {
		desc := strings.TrimSpace(itemDescs[i])
		code := ""
		if i < len(itemCodes) {
			code = strings.TrimSpace(itemCodes[i])
		}
		if desc == "" && code == "" {
			continue
		}

		uom := "PCS"
		if i < len(itemUoms) && strings.TrimSpace(itemUoms[i]) != "" {
			uom = strings.TrimSpace(itemUoms[i])
		}

		qty := 0.0
		if i < len(itemQtys) {
			qty, _ = strconv.ParseFloat(itemQtys[i], 64)
		}

		cost := 0.0
		if i < len(itemCosts) {
			cost, _ = strconv.ParseFloat(itemCosts[i], 64)
		}

		poItems = append(poItems, domain.PurchaseOrderItem{
			ItemCode:    code,
			Description: desc,
			UOM:         uom,
			Qty:         qty,
			UnitCost:    cost,
		})
	}

	domainPO := domain.PurchaseOrder{
		PONo:           poNo,
		Date:           poDate,
		DueDate:        dueDate,
		VendorName:     vendorName,
		VendorTIN:      strings.TrimSpace(r.FormValue("vendor_tin")),
		VendorAddress:  strings.TrimSpace(r.FormValue("vendor_address")),
		VendorContact:  strings.TrimSpace(r.FormValue("vendor_contact")),
		ShipTo:         strings.TrimSpace(r.FormValue("ship_to")),
		PaymentTerms:   strings.TrimSpace(r.FormValue("payment_terms")),
		ShippingMethod: strings.TrimSpace(r.FormValue("shipping_method")),
		PreparedBy:     strings.TrimSpace(r.FormValue("prepared_by")),
		Freight:        freight,
		ApplyVAT:       applyVAT,
		Notes:          strings.TrimSpace(r.FormValue("notes")),
		Items:          poItems,
	}

	domainPO.CalculateTotals()

	if err := domainPO.Validate(); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to start database transaction."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	insertPOQuery := `
		INSERT INTO purchase_orders (
			po_no, date, due_date, vendor_name, vendor_tin, vendor_address, vendor_contact,
			ship_to, payment_terms, shipping_method, prepared_by, freight, apply_vat,
			vat_amt, subtotal, grand_total, notes
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?
		) RETURNING id
	`
	var poID int
	err = tx.QueryRow(
		insertPOQuery,
		domainPO.PONo,
		domainPO.Date,
		domainPO.DueDate,
		domainPO.VendorName,
		domainPO.VendorTIN,
		domainPO.VendorAddress,
		domainPO.VendorContact,
		domainPO.ShipTo,
		domainPO.PaymentTerms,
		domainPO.ShippingMethod,
		domainPO.PreparedBy,
		domainPO.Freight,
		domainPO.ApplyVAT,
		domainPO.VATAmt,
		domainPO.Subtotal,
		domainPO.GrandTotal,
		domainPO.Notes,
	).Scan(&poID)

	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to save purchase order header. PO number might already exist."}}`)
		http.Error(w, "Failed to save purchase order: "+err.Error(), http.StatusInternalServerError)
		return
	}

	insertItemQuery := `
		INSERT INTO purchase_order_items (
			po_id, item_code, description, uom, qty, unit_cost, total_amount
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	for _, item := range domainPO.Items {
		_, err := tx.Exec(
			insertItemQuery,
			poID,
			item.ItemCode,
			item.Description,
			item.UOM,
			item.Qty,
			item.UnitCost,
			item.TotalAmount,
		)
		if err != nil {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to save purchase order item."}}`)
			http.Error(w, "Failed to save purchase order item: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to commit purchase order transaction."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	nextPONo := fmt.Sprintf("PO-%s-%03d", now.Format("060102"), now.Minute()*60+now.Second())

	successMsg := fmt.Sprintf("Purchase Order %s saved to system successfully!", domainPO.PONo)
	triggerJSON := fmt.Sprintf(`{"show-toast": {"type": "success", "message": "%s"}, "po-saved": {"next_po_no": "%s"}}`, successMsg, nextPONo)
	w.Header().Set("HX-Trigger", triggerJSON)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"po_id":      poID,
		"po_no":      domainPO.PONo,
		"next_po_no": nextPONo,
		"message":    successMsg,
	})
}
