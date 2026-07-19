package handlers

import (
	"net/http"
	"strconv"

	"radline/db"
	"radline/models"
)

// SettingsHandler lists Brands, Categories, and UOM settings
func (app *App) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch UOM settings
	uomQuery := `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		ORDER BY i.code ASC, u.muom ASC
	`
	var uomSettings []models.UomSettingWithItem
	err := db.DB.Select(&uomSettings, uomQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Fetch Brands (initial paginated view)
	brandQuery := `
		SELECT b.id, b.code, b.name, COUNT(i.id) as item_count
		FROM brands b
		LEFT JOIN items i ON b.id = i.brand_id
		GROUP BY b.id, b.code, b.name
		ORDER BY b.code ASC
		LIMIT ?
	`
	var brands []models.Brand
	_ = db.DB.Select(&brands, brandQuery, DefaultPageSize)

	// 3. Fetch Categories (initial limited view)
	categoryQuery := `
		SELECT c.id, c.code, c.name, COUNT(i.id) as item_count
		FROM categories c
		LEFT JOIN items i ON c.id = i.category_id
		GROUP BY c.id, c.code, c.name
		ORDER BY c.code ASC
		LIMIT ?
	`
	var categories []models.Category
	_ = db.DB.Select(&categories, categoryQuery, DefaultPageSize)

	// 4. Fetch Items (for inline UOM form dropdown in settings page)
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")

	// 5. Fetch Predefined Uoms (initial limited view)
	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC LIMIT ?", DefaultPageSize)

	data := struct {
		UomSettings []models.UomSettingWithItem
		Brands      []models.Brand
		Categories  []models.Category
		Items       []models.Item
		Uoms        []models.Uom
	}{
		UomSettings: uomSettings,
		Brands:      brands,
		Categories:  categories,
		Items:       items,
		Uoms:        uoms,
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		app.Render(w, "uom_settings_results.html", data)
	} else {
		app.RenderPage(w, r, "settings.html", data)
	}
}

// NewUomSettingPageHandler redirects to uom-settings list page
func (app *App) NewUomSettingPageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/uom-settings", http.StatusSeeOther)
}

// AddUomSettingHandler inserts a new UOM setting into the database
func (app *App) AddUomSettingHandler(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.FormValue("item_id")
	muom := r.FormValue("muom")
	factorStr := r.FormValue("conversion_factor")

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil || itemID <= 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid item selected."}}`)
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	if muom == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "MUOM is required."}}`)
		http.Error(w, "MUOM is required", http.StatusBadRequest)
		return
	}

	factor, err := strconv.ParseFloat(factorStr, 64)
	if err != nil || factor <= 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Conversion rate must be greater than zero."}}`)
		http.Error(w, "Invalid conversion factor", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO uom_settings (item_id, muom, conversion_factor)
		VALUES (?, ?, ?)
	`, itemID, muom, factor)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to add UOM setting. The item might already have this MUOM."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "UOM Setting added successfully!"}, "uom-setting-added": ""}`)
	w.WriteHeader(http.StatusOK)
}

// EditUomSettingFormHandler renders the inline edit form for a UOM setting row
func (app *App) EditUomSettingFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	var uomSetting models.UomSettingWithItem
	err = db.DB.Get(&uomSetting, `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		WHERE u.id = ?
	`, id)
	if err != nil {
		http.Error(w, "UOM setting not found", http.StatusNotFound)
		return
	}

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	data := struct {
		Setting models.UomSettingWithItem
		Uoms    []models.Uom
	}{
		Setting: uomSetting,
		Uoms:    uoms,
	}

	app.Render(w, "uom_setting_edit_row.html", data)
}

// UpdateUomSettingHandler updates an existing UOM setting
func (app *App) UpdateUomSettingHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	muom := r.FormValue("muom")
	factorStr := r.FormValue("conversion_factor")

	if muom == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "MUOM is required."}}`)
		http.Error(w, "MUOM is required", http.StatusBadRequest)
		return
	}

	factor, err := strconv.ParseFloat(factorStr, 64)
	if err != nil || factor <= 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Conversion rate must be greater than zero."}}`)
		http.Error(w, "Invalid conversion factor", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("UPDATE uom_settings SET muom = ?, conversion_factor = ? WHERE id = ?", muom, factor, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update UOM setting."}}`)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var uomSetting models.UomSettingWithItem
	err = db.DB.Get(&uomSetting, `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		WHERE u.id = ?
	`, id)
	if err != nil {
		http.Error(w, "Failed to retrieve UOM setting", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "UOM setting updated successfully!"}}`)
	app.Render(w, "uom_setting_row.html", uomSetting)
}

// UomSettingRowHandler renders a single UOM setting row
func (app *App) UomSettingRowHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	var uomSetting models.UomSettingWithItem
	err = db.DB.Get(&uomSetting, `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		WHERE u.id = ?
	`, id)
	if err != nil {
		http.Error(w, "UOM setting not found", http.StatusNotFound)
		return
	}

	app.Render(w, "uom_setting_row.html", uomSetting)
}

// DeleteUomSettingHandler deletes a UOM setting
func (app *App) DeleteUomSettingHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM uom_settings WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to delete UOM setting."}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "UOM setting deleted successfully!"}}`)
	w.WriteHeader(http.StatusOK)
}
