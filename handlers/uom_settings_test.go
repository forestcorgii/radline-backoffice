package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"radline/db"
)

func parseUomTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	pages := []string{"uoms.html"}
	for _, page := range pages {
		t := template.New(page)
		files := []string{
			"../templates/base.html",
			"../templates/" + page,
			"../templates/uoms_results.html",
			"../templates/uom_rows.html",
			"../templates/uom_row.html",
			"../templates/uom_settings_results.html",
			"../templates/uom_setting_rows.html",
			"../templates/uom_setting_row.html",
		}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	fragments := []string{"uoms_results.html", "uom_settings_results.html"}
	for _, frag := range fragments {
		t := template.New(frag)
		files := []string{"../templates/" + frag}
		if frag == "uoms_results.html" {
			files = append(files, "../templates/uom_rows.html", "../templates/uom_row.html")
		} else if frag == "uom_settings_results.html" {
			files = append(files, "../templates/uom_setting_rows.html", "../templates/uom_setting_row.html")
		}
		t = template.Must(t.ParseFiles(files...))
		templates[frag] = t
	}

	return templates
}

func TestUomSettingsFilteringAndSimplification(t *testing.T) {
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	app := &App{
		Templates: parseUomTemplatesForTest(),
	}

	// 1. Test UomSettingsHandler with HTMX filter request
	req := httptest.NewRequest("GET", "/uom-settings?search=BOX&sort=code_asc&limit=10", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "uom-settings-results")
	w := httptest.NewRecorder()

	app.UomSettingsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for UomSettingsHandler HTMX filter, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. Test UomsHandler with HTMX request (simplified without filter parameters)
	reqUoms := httptest.NewRequest("GET", "/uoms", nil)
	reqUoms.Header.Set("HX-Request", "true")
	reqUoms.Header.Set("HX-Target", "uoms-results")
	wUoms := httptest.NewRecorder()

	app.UomsHandler(wUoms, reqUoms)
	if wUoms.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for UomsHandler HTMX request, got %d. Body: %s", wUoms.Code, wUoms.Body.String())
	}

	// 3. Test Full Page render of /uoms
	reqPage := httptest.NewRequest("GET", "/uoms", nil)
	wPage := httptest.NewRecorder()

	app.UomsHandler(wPage, reqPage)
	if wPage.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /uoms full page render, got %d. Body: %s", wPage.Code, wPage.Body.String())
	}
}
