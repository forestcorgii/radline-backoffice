package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"radline/db"
	"radline/models"
)

// BrandsHandler lists all brands
func (app *App) BrandsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	filter := r.URL.Query().Get("filter") // all, in_use, unused
	sort := r.URL.Query().Get("sort")     // code_asc, code_desc, name_asc, name_desc, items_asc, items_desc

	query := `
		SELECT b.id, b.code, b.name, COUNT(i.id) as item_count
		FROM brands b
		LEFT JOIN items i ON b.id = i.brand_id
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(b.code LIKE ? OR b.name LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " GROUP BY b.id, b.code, b.name"

	if filter == "in_use" {
		query += " HAVING COUNT(i.id) > 0"
	} else if filter == "unused" {
		query += " HAVING COUNT(i.id) = 0"
	}

	switch sort {
	case "code_desc":
		query += " ORDER BY b.code DESC"
	case "name_asc":
		query += " ORDER BY b.name ASC"
	case "name_desc":
		query += " ORDER BY b.name DESC"
	case "items_asc":
		query += " ORDER BY item_count ASC, b.code ASC"
	case "items_desc":
		query += " ORDER BY item_count DESC, b.code ASC"
	default: // code_asc or empty
		query += " ORDER BY b.code ASC"
	}

	var brands []models.Brand
	err := db.DB.Select(&brands, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		app.Render(w, "brand_rows.html", brands)
	} else {
		app.RenderPage(w, r, "brands.html", brands)
	}
}

// AddBrandHandler adds a new brand via HTMX
func (app *App) AddBrandHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	name := r.FormValue("name")

	if code == "" || name == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Code and Name are required."}}`)
		http.Error(w, "Code and Name are required", http.StatusBadRequest)
		return
	}

	_, err := db.DB.Exec("INSERT INTO brands (code, name) VALUES (?, ?)", code, name)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to add brand. Code might already be in use."}}`)
		http.Error(w, "Failed to insert brand", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Brand added successfully!"}, "brand-added": ""}`)

	// Fetch updated list and render just the rows for HTMX swap
	var brands []models.Brand
	_ = db.DB.Select(&brands, `
		SELECT b.id, b.code, b.name, COUNT(i.id) as item_count
		FROM brands b
		LEFT JOIN items i ON b.id = i.brand_id
		GROUP BY b.id, b.code, b.name
		ORDER BY b.code ASC
	`)
	app.Render(w, "brand_rows.html", brands)
}

// EditBrandFormHandler renders the inline edit form for a brand
func (app *App) EditBrandFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid brand ID", http.StatusBadRequest)
		return
	}

	var brand models.Brand
	err = db.DB.Get(&brand, `
		SELECT b.id, b.code, b.name, COUNT(i.id) as item_count
		FROM brands b
		LEFT JOIN items i ON b.id = i.brand_id
		WHERE b.id = ?
		GROUP BY b.id, b.code, b.name
	`, id)
	if err != nil {
		http.Error(w, "Brand not found", http.StatusNotFound)
		return
	}

	app.Render(w, "brand_edit_row.html", brand)
}

// UpdateBrandHandler updates a brand
func (app *App) UpdateBrandHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid brand ID", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	name := r.FormValue("name")

	if code == "" || name == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Code and Name are required."}}`)
		http.Error(w, "Code and Name are required", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("UPDATE brands SET code = ?, name = ? WHERE id = ?", code, name, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update brand. Code might already be in use."}}`)
		http.Error(w, "Failed to update brand", http.StatusInternalServerError)
		return
	}

	var brand models.Brand
	err = db.DB.Get(&brand, `
		SELECT b.id, b.code, b.name, COUNT(i.id) as item_count
		FROM brands b
		LEFT JOIN items i ON b.id = i.brand_id
		WHERE b.id = ?
		GROUP BY b.id, b.code, b.name
	`, id)
	if err != nil {
		http.Error(w, "Failed to retrieve brand", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Brand updated successfully!"}}`)
	app.Render(w, "brand_row.html", brand)
}

// BrandRowHandler renders a single brand row
func (app *App) BrandRowHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid brand ID", http.StatusBadRequest)
		return
	}

	var brand models.Brand
	err = db.DB.Get(&brand, `
		SELECT b.id, b.code, b.name, COUNT(i.id) as item_count
		FROM brands b
		LEFT JOIN items i ON b.id = i.brand_id
		WHERE b.id = ?
		GROUP BY b.id, b.code, b.name
	`, id)
	if err != nil {
		http.Error(w, "Brand not found", http.StatusNotFound)
		return
	}

	app.Render(w, "brand_row.html", brand)
}

// DeleteBrandHandler deletes a brand
func (app *App) DeleteBrandHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid brand ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM brands WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Cannot delete brand. It is in use by one or more items."}}`)
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Brand deleted successfully!"}}`)
	w.WriteHeader(http.StatusOK)
}

// CategoriesHandler lists all categories
func (app *App) CategoriesHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	filter := r.URL.Query().Get("filter") // all, in_use, unused
	sort := r.URL.Query().Get("sort")     // code_asc, code_desc, name_asc, name_desc, items_asc, items_desc

	query := `
		SELECT c.id, c.code, c.name, COUNT(i.id) as item_count
		FROM categories c
		LEFT JOIN items i ON c.id = i.category_id
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(c.code LIKE ? OR c.name LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " GROUP BY c.id, c.code, c.name"

	if filter == "in_use" {
		query += " HAVING COUNT(i.id) > 0"
	} else if filter == "unused" {
		query += " HAVING COUNT(i.id) = 0"
	}

	switch sort {
	case "code_desc":
		query += " ORDER BY c.code DESC"
	case "name_asc":
		query += " ORDER BY c.name ASC"
	case "name_desc":
		query += " ORDER BY c.name DESC"
	case "items_asc":
		query += " ORDER BY item_count ASC, c.code ASC"
	case "items_desc":
		query += " ORDER BY item_count DESC, c.code ASC"
	default: // code_asc or empty
		query += " ORDER BY c.code ASC"
	}

	var categories []models.Category
	err := db.DB.Select(&categories, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		app.Render(w, "category_rows.html", categories)
	} else {
		app.RenderPage(w, r, "categories.html", categories)
	}
}

// AddCategoryHandler adds a new category via HTMX
func (app *App) AddCategoryHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	name := r.FormValue("name")

	if code == "" || name == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Code and Name are required."}}`)
		http.Error(w, "Code and Name are required", http.StatusBadRequest)
		return
	}

	_, err := db.DB.Exec("INSERT INTO categories (code, name) VALUES (?, ?)", code, name)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to add category. Code might already be in use."}}`)
		http.Error(w, "Failed to insert category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Category added successfully!"}, "category-added": ""}`)

	var categories []models.Category
	_ = db.DB.Select(&categories, `
		SELECT c.id, c.code, c.name, COUNT(i.id) as item_count
		FROM categories c
		LEFT JOIN items i ON c.id = i.category_id
		GROUP BY c.id, c.code, c.name
		ORDER BY c.code ASC
	`)
	app.Render(w, "category_rows.html", categories)
}

// EditCategoryFormHandler renders the inline edit form for a category
func (app *App) EditCategoryFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	err = db.DB.Get(&category, `
		SELECT c.id, c.code, c.name, COUNT(i.id) as item_count
		FROM categories c
		LEFT JOIN items i ON c.id = i.category_id
		WHERE c.id = ?
		GROUP BY c.id, c.code, c.name
	`, id)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	app.Render(w, "category_edit_row.html", category)
}

// UpdateCategoryHandler updates a category
func (app *App) UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	name := r.FormValue("name")

	if code == "" || name == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Code and Name are required."}}`)
		http.Error(w, "Code and Name are required", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("UPDATE categories SET code = ?, name = ? WHERE id = ?", code, name, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update category. Code might already be in use."}}`)
		http.Error(w, "Failed to update category", http.StatusInternalServerError)
		return
	}

	var category models.Category
	err = db.DB.Get(&category, `
		SELECT c.id, c.code, c.name, COUNT(i.id) as item_count
		FROM categories c
		LEFT JOIN items i ON c.id = i.category_id
		WHERE c.id = ?
		GROUP BY c.id, c.code, c.name
	`, id)
	if err != nil {
		http.Error(w, "Failed to retrieve category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Category updated successfully!"}}`)
	app.Render(w, "category_row.html", category)
}

// CategoryRowHandler renders a single category row
func (app *App) CategoryRowHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	err = db.DB.Get(&category, `
		SELECT c.id, c.code, c.name, COUNT(i.id) as item_count
		FROM categories c
		LEFT JOIN items i ON c.id = i.category_id
		WHERE c.id = ?
		GROUP BY c.id, c.code, c.name
	`, id)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	app.Render(w, "category_row.html", category)
}

// DeleteCategoryHandler deletes a category
func (app *App) DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Cannot delete category. It is in use by one or more items."}}`)
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Category deleted successfully!"}}`)
	w.WriteHeader(http.StatusOK)
}

// ItemsHandler lists all items with search, filter, and sort
func (app *App) ItemsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	brandIDFilterStr := r.URL.Query().Get("brand_id_filter")
	categoryIDFilterStr := r.URL.Query().Get("category_id_filter")
	sort := r.URL.Query().Get("sort") // code_asc, code_desc, description_asc, description_desc

	brandIDFilter, _ := strconv.Atoi(brandIDFilterStr)
	categoryIDFilter, _ := strconv.Atoi(categoryIDFilterStr)

	query := `
		SELECT i.id, i.code, i.description, i.default_uom, i.model, i.brand_id, i.category_id, i.variation, i.remarks,
		       COALESCE(b.name, '') as brand_name, COALESCE(c.name, '') as category_name
		FROM items i
		LEFT JOIN brands b ON i.brand_id = b.id
		LEFT JOIN categories c ON i.category_id = c.id
	`
	var args []interface{}
	var whereClauses []string

	if search != "" {
		whereClauses = append(whereClauses, "(i.code LIKE ? OR i.description LIKE ? OR i.model LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if brandIDFilter > 0 {
		whereClauses = append(whereClauses, "i.brand_id = ?")
		args = append(args, brandIDFilter)
	}

	if categoryIDFilter > 0 {
		whereClauses = append(whereClauses, "i.category_id = ?")
		args = append(args, categoryIDFilter)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	switch sort {
	case "code_desc":
		query += " ORDER BY i.code DESC"
	case "description_asc":
		query += " ORDER BY i.description ASC"
	case "description_desc":
		query += " ORDER BY i.description DESC"
	default: // code_asc or empty
		query += " ORDER BY i.code ASC"
	}

	var items []models.ItemWithRelations
	err := db.DB.Select(&items, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
		app.Render(w, "item_rows.html", items)
	} else {
		var brands []models.Brand
		_ = db.DB.Select(&brands, "SELECT * FROM brands ORDER BY name ASC")

		var categories []models.Category
		_ = db.DB.Select(&categories, "SELECT * FROM categories ORDER BY name ASC")

		data := struct {
			Items      []models.ItemWithRelations
			Brands     []models.Brand
			Categories []models.Category
		}{
			Items:      items,
			Brands:     brands,
			Categories: categories,
		}
		app.RenderPage(w, r, "items.html", data)
	}
}

// AddItemHandler adds a new item
func (app *App) AddItemHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	description := r.FormValue("description")
	model := r.FormValue("model")
	defaultUom := r.FormValue("default_uom")
	brandIDStr := r.FormValue("brand_id")
	categoryIDStr := r.FormValue("category_id")

	if code == "" || description == "" || defaultUom == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Code, Description and Default UOM are required."}}`)
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var brandID interface{}
	brandIDVal, err := strconv.Atoi(brandIDStr)
	if err == nil && brandIDVal > 0 {
		brandID = brandIDVal
	} else {
		brandID = nil
	}

	var categoryID interface{}
	categoryIDVal, err := strconv.Atoi(categoryIDStr)
	if err == nil && categoryIDVal > 0 {
		categoryID = categoryIDVal
	} else {
		categoryID = nil
	}

	_, err = db.DB.Exec(`
		INSERT INTO items (code, description, model, brand_id, category_id, default_uom, variation, remarks)
		VALUES (?, ?, ?, ?, ?, ?, '', '')
	`, code, description, model, brandID, categoryID, defaultUom)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to add item. Code might already be in use."}}`)
		http.Error(w, "Failed to insert item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Item added successfully!"}, "item-added": ""}`)

	var items []models.ItemWithRelations
	_ = db.DB.Select(&items, `
		SELECT i.id, i.code, i.description, i.default_uom, i.model, i.brand_id, i.category_id, i.variation, i.remarks,
		       COALESCE(b.name, '') as brand_name, COALESCE(c.name, '') as category_name
		FROM items i
		LEFT JOIN brands b ON i.brand_id = b.id
		LEFT JOIN categories c ON i.category_id = c.id
		ORDER BY i.code ASC
	`)
	app.Render(w, "item_rows.html", items)
}

// EditItemFormHandler renders the inline edit form for an item
func (app *App) EditItemFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	var item models.Item
	err = db.DB.Get(&item, "SELECT * FROM items WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	var brands []models.Brand
	_ = db.DB.Select(&brands, "SELECT * FROM brands ORDER BY name ASC")

	var categories []models.Category
	_ = db.DB.Select(&categories, "SELECT * FROM categories ORDER BY name ASC")

	data := struct {
		Item       models.Item
		Brands     []models.Brand
		Categories []models.Category
	}{
		Item:       item,
		Brands:     brands,
		Categories: categories,
	}

	app.Render(w, "item_edit_row.html", data)
}

// UpdateItemHandler updates an item
func (app *App) UpdateItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	description := r.FormValue("description")
	model := r.FormValue("model")
	defaultUom := r.FormValue("default_uom")
	brandIDStr := r.FormValue("brand_id")
	categoryIDStr := r.FormValue("category_id")

	if code == "" || description == "" || defaultUom == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Code, Description and Default UOM are required."}}`)
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var brandID interface{}
	brandIDVal, err := strconv.Atoi(brandIDStr)
	if err == nil && brandIDVal > 0 {
		brandID = brandIDVal
	} else {
		brandID = nil
	}

	var categoryID interface{}
	categoryIDVal, err := strconv.Atoi(categoryIDStr)
	if err == nil && categoryIDVal > 0 {
		categoryID = categoryIDVal
	} else {
		categoryID = nil
	}

	_, err = db.DB.Exec(`
		UPDATE items
		SET code = ?, description = ?, model = ?, brand_id = ?, category_id = ?, default_uom = ?
		WHERE id = ?
	`, code, description, model, brandID, categoryID, defaultUom, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update item. Code might already be in use."}}`)
		http.Error(w, "Failed to update item", http.StatusInternalServerError)
		return
	}

	var item models.ItemWithRelations
	err = db.DB.Get(&item, `
		SELECT i.id, i.code, i.description, i.default_uom, i.model, i.brand_id, i.category_id, i.variation, i.remarks,
		       COALESCE(b.name, '') as brand_name, COALESCE(c.name, '') as category_name
		FROM items i
		LEFT JOIN brands b ON i.brand_id = b.id
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.id = ?
	`, id)
	if err != nil {
		http.Error(w, "Failed to retrieve item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Item updated successfully!"}}`)
	app.Render(w, "item_row.html", item)
}

// ItemRowHandler renders a single item row
func (app *App) ItemRowHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	var item models.ItemWithRelations
	err = db.DB.Get(&item, `
		SELECT i.id, i.code, i.description, i.default_uom, i.model, i.brand_id, i.category_id, i.variation, i.remarks,
		       COALESCE(b.name, '') as brand_name, COALESCE(c.name, '') as category_name
		FROM items i
		LEFT JOIN brands b ON i.brand_id = b.id
		LEFT JOIN categories c ON i.category_id = c.id
		WHERE i.id = ?
	`, id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	app.Render(w, "item_row.html", item)
}

// DeleteItemHandler deletes an item
func (app *App) DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM items WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Cannot delete item. It is referenced by stock logs or sales details."}}`)
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Item deleted successfully!"}}`)
	w.WriteHeader(http.StatusOK)
}

// EntryHandler renders the consolidated data entry page
func (app *App) EntryHandler(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	if tab == "receiving" || tab == "adjustments" {
		http.Redirect(w, r, "/inventory/"+tab, http.StatusMovedPermanently)
		return
	}
	if tab == "" {
		tab = "sales" // default tab
	}

	var brands []models.Brand
	_ = db.DB.Select(&brands, "SELECT id, code, name FROM brands ORDER BY code ASC")

	var categories []models.Category
	_ = db.DB.Select(&categories, "SELECT id, code, name FROM categories ORDER BY code ASC")

	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY code ASC")

	data := struct {
		ActiveTab    string
		Brands       []models.Brand
		Categories   []models.Category
		Items        []models.Item
		SalesRowData interface{}
	}{
		ActiveTab:  tab,
		Brands:     brands,
		Categories: categories,
		Items:      items,
		SalesRowData: map[string]interface{}{
			"Items": items,
		},
	}

	app.RenderPage(w, r, "entry.html", data)
}

// SelectBrandsHandler renders the updated brand select fragment
func (app *App) SelectBrandsHandler(w http.ResponseWriter, r *http.Request) {
	var brands []models.Brand
	err := db.DB.Select(&brands, "SELECT id, code, name FROM brands ORDER BY code ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.Render(w, "brand_select.html", brands)
}

// SelectCategoriesHandler renders the updated category select fragment
func (app *App) SelectCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	var categories []models.Category
	err := db.DB.Select(&categories, "SELECT id, code, name FROM categories ORDER BY code ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.Render(w, "category_select.html", categories)
}

// SelectItemsHandler renders the updated item select fragment
func (app *App) SelectItemsHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	err := db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY code ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.Render(w, "item_select.html", items)
}

// ItemDefaultsHandler returns OOB swaps for UOM, cost, and price based on selected item
func (app *App) ItemDefaultsHandler(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.URL.Query().Get("item_id")
	itemID, _ := strconv.Atoi(itemIDStr)
	tab := r.URL.Query().Get("tab")

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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if tab == "receiving" {
		costVal := ""
		if lastCost > 0 {
			costVal = strconv.FormatFloat(lastCost, 'f', 2, 64)
		}
		priceVal := ""
		if lastPrice > 0 {
			priceVal = strconv.FormatFloat(lastPrice, 'f', 2, 64)
		}

		w.Write([]byte(`
			<input type="text" id="receiving_uom" name="uom" value="` + defaultUOM + `" placeholder="e.g. BOX, PCS" class="form-input" required hx-swap-oob="true">
			<input type="number" step="any" id="receiving_cost" name="cost" value="` + costVal + `" class="form-input" required hx-swap-oob="true">
			<input type="number" step="any" id="receiving_unit_price" name="unit_price" value="` + priceVal + `" class="form-input" required hx-swap-oob="true">
		`))
	} else if tab == "adjustments" {
		costVal := ""
		if lastCost > 0 {
			costVal = strconv.FormatFloat(lastCost, 'f', 2, 64)
		}

		w.Write([]byte(`
			<input type="text" id="adjustments_uom" name="uom" value="` + defaultUOM + `" placeholder="e.g. PCS" class="form-input" required hx-swap-oob="true">
			<input type="number" step="any" id="adjustments_cost" name="cost" value="` + costVal + `" class="form-input" required hx-swap-oob="true">
		`))
	}
}

