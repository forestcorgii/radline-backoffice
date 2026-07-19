package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"radline/db"
)

func parseSalesTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	funcMap := template.FuncMap{
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}

	pages := []string{"sales.html"}
	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"../templates/base.html", "../templates/" + page, "../templates/sales_rows.html", "../templates/sale_item_row.html", "../templates/sales_results.html", "../templates/item_select.html", "../templates/uom_select.html", "../templates/sale_row.html", "../templates/sale_edit_row.html"}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	fragments := []string{"sales_results.html", "sales_rows.html", "sale_row.html", "sale_edit_row.html"}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcMap)
		files := []string{"../templates/" + frag}
		if frag == "sales_results.html" {
			files = append(files, "../templates/sales_rows.html", "../templates/sale_row.html", "../templates/sale_edit_row.html")
		} else if frag == "sales_rows.html" {
			files = append(files, "../templates/sale_row.html", "../templates/sale_edit_row.html")
		}
		t = template.Must(t.ParseFiles(files...))
		templates[frag] = t
	}

	return templates
}

func TestSalesSearchFragmentHandler(t *testing.T) {
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	app := &App{
		Templates: parseSalesTemplatesForTest(),
	}

	// 1. Fragment request via HTMX search (Normal mode)
	req := httptest.NewRequest("GET", "/sales?search=test", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "sales-results")
	w := httptest.NewRecorder()

	app.SalesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for sales search fragment, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. Fragment request via HTMX search (Edit mode)
	reqEdit := httptest.NewRequest("GET", "/sales?search=test&is_edit=1", nil)
	reqEdit.Header.Set("HX-Request", "true")
	reqEdit.Header.Set("HX-Target", "sales-results")
	wEdit := httptest.NewRecorder()

	app.SalesHandler(wEdit, reqEdit)

	if wEdit.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for sales search fragment in edit mode, got %d. Body: %s", wEdit.Code, wEdit.Body.String())
	}
}
