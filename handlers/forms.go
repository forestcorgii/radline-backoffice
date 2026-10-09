package handlers

import (
	"fmt"
	"net/http"
	"time"

	"radline/db"
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

// PurchaseOrderFormHandler renders the printable Purchase Order Form page
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
