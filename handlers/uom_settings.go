package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"radline/db"
	"radline/models"
)

// UomsHandler lists all UOMs and renders the UOM management page.
func (app *App) UomsHandler(w http.ResponseWriter, r *http.Request) {
	var uoms []models.Uom
	err := db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		data := struct {
			Uoms []models.Uom
		}{
			Uoms: uoms,
		}
		app.Render(w, "uoms_results.html", data)
		return
	}

	// Fetch initial UOM Conversion Settings (default limit 25, ordered by code ASC)
	uomSettings, _ := app.queryUomSettings("", "", "code_asc", 25)

	// Fetch all conversion settings for UOM Calculator dropdown so none are excluded
	var allUomSettings []models.UomSettingWithItem
	_ = db.DB.Select(&allUomSettings, `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		ORDER BY i.code ASC, u.muom ASC
	`)

	// Fetch Items for conversion form dropdown
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY description ASC")

	data := struct {
		Uoms           []models.Uom
		UomSettings    []models.UomSettingWithItem
		AllUomSettings []models.UomSettingWithItem
		Items          []models.Item
	}{
		Uoms:           uoms,
		UomSettings:    uomSettings,
		AllUomSettings: allUomSettings,
		Items:          items,
	}
	app.RenderPage(w, r, "uoms.html", data)
}

func (app *App) queryUomSettings(search, muomFilter, sort string, limit int) ([]models.UomSettingWithItem, error) {
	query := `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(i.code LIKE ? OR i.description LIKE ? OR u.muom LIKE ? OR i.default_uom LIKE ?)")
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if muomFilter != "" && muomFilter != "all" {
		whereClauses = append(whereClauses, "u.muom = ?")
		args = append(args, muomFilter)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	switch sort {
	case "code_desc":
		query += " ORDER BY i.code DESC, u.muom ASC"
	case "name_asc":
		query += " ORDER BY i.description ASC, i.code ASC"
	case "name_desc":
		query += " ORDER BY i.description DESC, i.code ASC"
	case "muom_asc":
		query += " ORDER BY u.muom ASC, i.code ASC"
	case "muom_desc":
		query += " ORDER BY u.muom DESC, i.code ASC"
	case "factor_desc":
		query += " ORDER BY u.conversion_factor DESC, i.code ASC"
	case "factor_asc":
		query += " ORDER BY u.conversion_factor ASC, i.code ASC"
	default: // code_asc or empty
		query += " ORDER BY i.code ASC, u.muom ASC"
	}

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	var uomSettings []models.UomSettingWithItem
	err := db.DB.Select(&uomSettings, query, args...)
	return uomSettings, err
}

// UomSettingsHandler handles /uom-settings requests with search, filter, and sorting
func (app *App) UomSettingsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	muom := r.URL.Query().Get("muom")
	sort := r.URL.Query().Get("sort")
	limit := GetLimitParam(r)

	uomSettings, err := app.queryUomSettings(search, muom, sort, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		UomSettings []models.UomSettingWithItem
	}{
		UomSettings: uomSettings,
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		app.Render(w, "uom_settings_results.html", data)
	} else {
		http.Redirect(w, r, "/uoms", http.StatusSeeOther)
	}
}

// AddUomHandler adds a new UOM to the predefined list.
func (app *App) AddUomHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.FormValue("code"))

	if code == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Code is required."}}`)
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	code = strings.ToUpper(code)

	_, err := db.DB.Exec("INSERT INTO uoms (code) VALUES (?)", code)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to add UOM. Code might already exist."}}`)
		http.Error(w, "Failed to insert UOM", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "UOM added successfully!"}, "uom-added": ""}`)
	w.WriteHeader(http.StatusOK)
}

// UomRowHandler renders a single UOM row fragment.
func (app *App) UomRowHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM ID", http.StatusBadRequest)
		return
	}

	var uom models.Uom
	err = db.DB.Get(&uom, `
		SELECT u.id, u.code
		FROM uoms u
		WHERE u.id = ?
	`, id)
	if err != nil {
		http.Error(w, "UOM not found", http.StatusNotFound)
		return
	}

	app.Render(w, "uom_row.html", uom)
}

// DeleteUomHandler deletes a UOM if it is not currently in use by items or UOM rules.
func (app *App) DeleteUomHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM ID", http.StatusBadRequest)
		return
	}

	// Check if in use as Default UOM in items
	var count int
	err = db.DB.Get(&count, `
		SELECT COUNT(*) FROM items WHERE default_uom IN (SELECT code FROM uoms WHERE id = ?)
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Cannot delete UOM because it is in use by one or more items."}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if in use as MUOM in uom_settings
	err = db.DB.Get(&count, `
		SELECT COUNT(*) FROM uom_settings WHERE muom IN (SELECT code FROM uoms WHERE id = ?)
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Cannot delete UOM because it is used in one or more UOM rules."}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM uoms WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to delete UOM."}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "UOM deleted successfully!"}, "uom-added": ""}`)
	w.WriteHeader(http.StatusOK)
}

// SelectUomsHandler renders the updated UOM select fragment
func (app *App) SelectUomsHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "default_uom"
	}
	selected := r.URL.Query().Get("selected")

	var uoms []models.Uom
	err := db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.Render(w, "uom_select.html", map[string]interface{}{
		"Name":        name,
		"Uoms":        uoms,
		"SelectedUOM": selected,
	})
}

