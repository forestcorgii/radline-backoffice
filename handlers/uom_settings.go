package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"radline/db"
	"radline/models"
)

// UomsHandler lists all UOMs with optional search, sorting, and limit parameters.
func (app *App) UomsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	filter := r.URL.Query().Get("filter") // all, in_use, unused
	sort := r.URL.Query().Get("sort")     // code_asc, code_desc

	query := `
		SELECT u.id, u.code
		FROM uoms u
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "u.code LIKE ?")
		args = append(args, "%"+search+"%")
	}

	if filter == "in_use" {
		whereClauses = append(whereClauses, "u.code IN (SELECT DISTINCT default_uom FROM items UNION SELECT DISTINCT muom FROM uom_settings)")
	} else if filter == "unused" {
		whereClauses = append(whereClauses, "u.code NOT IN (SELECT DISTINCT default_uom FROM items UNION SELECT DISTINCT muom FROM uom_settings)")
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	limit := GetLimitParam(r)

	switch sort {
	case "code_desc":
		query += " ORDER BY u.code DESC"
	default: // code_asc or empty
		query += " ORDER BY u.code ASC"
	}

	query += " LIMIT ?"
	selectArgs := append(args, limit)

	var uoms []models.Uom
	err := db.DB.Select(&uoms, query, selectArgs...)
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
	} else {
		http.Redirect(w, r, "/settings", http.StatusMovedPermanently)
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
	w.Header().Set("HX-Location", "/settings")
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

