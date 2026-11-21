package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"rav/db"
)

func parseTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	funcMap := TemplateFuncMap()

	pages := []string{"receiving_logs.html"}
	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"../templates/base.html", "../templates/" + page, "../templates/receiving_rows.html", "../templates/receiving_logs_results.html", "../templates/receiving_log_row.html", "../templates/receiving_item_row.html", "../templates/receive_stock_form.html", "../templates/item_select.html"}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	fragments := []string{"receiving_logs_results.html", "receiving_log_row.html", "receiving_log_edit_row.html"}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcMap)
		files := []string{"../templates/" + frag}
		if frag == "receiving_logs_results.html" {
			files = append(files, "../templates/receiving_rows.html", "../templates/receiving_log_row.html")
		}
		t = template.Must(t.ParseFiles(files...))
		templates[frag] = t
	}

	return templates
}

func TestReceivingLogsHandler(t *testing.T) {
	setupIsolatedTestDB(t)

	if _, err := db.DB.Exec("INSERT INTO uoms (code) VALUES ('PCS') ON CONFLICT DO NOTHING"); err != nil {
		t.Fatalf("Failed to seed uom: %v", err)
	}
	var itemID int
	if err := db.DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('TEST-01', 'Test Receiving Item', 'PCS') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("Failed to seed item: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO receiving_logs (supplier, date, pl_no, item_id, qty, uom, unit_price, less1, less2, cost, total_cost, markup, selling_price, remarks)
		VALUES ('ASCD', '2026-01-10', 'PL-TEST', ?, 10, 'PCS', 100.0, 0, 0, 100.0, 1000.0, 130.0, 130.0, '')
	`, itemID); err != nil {
		t.Fatalf("Failed to seed receiving_log: %v", err)
	}

	app := &App{
		Templates: parseTemplatesForTest(),
	}

	// 1. Full page request via HTMX
	req := httptest.NewRequest("GET", "/inventory/receiving/logs", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "main-content")
	w := httptest.NewRecorder()

	app.ReceivingLogsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for full page, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. Fragment request via HTMX search
	req2 := httptest.NewRequest("GET", "/inventory/receiving/logs?search=test", nil)
	req2.Header.Set("HX-Request", "true")
	req2.Header.Set("HX-Target", "receiving-logs-results")
	w2 := httptest.NewRecorder()

	app.ReceivingLogsHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for fragment, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}
