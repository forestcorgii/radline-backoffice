package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"radline/db"
)

func parseTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	funcMap := template.FuncMap{
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
	}

	pages := []string{"receiving_logs.html"}
	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"../templates/base.html", "../templates/" + page, "../templates/receiving_rows.html", "../templates/receiving_logs_results.html", "../templates/receiving_log_row.html"}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	fragments := []string{"receiving_logs_results.html"}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcMap)
		files := []string{"../templates/" + frag, "../templates/receiving_rows.html", "../templates/receiving_log_row.html"}
		t = template.Must(t.ParseFiles(files...))
		templates[frag] = t
	}

	return templates
}

func TestReceivingLogsHandler(t *testing.T) {
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
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
