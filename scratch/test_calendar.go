package main

import (
	"fmt"
	"html/template"
	"net/http/httptest"

	"radline/db"
	"radline/handlers"
)

func main() {
	if err := db.InitDB("backoffice.db"); err != nil {
		fmt.Printf("InitDB error: %v\n", err)
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b float64) float64 { return a * b },
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) { return nil, nil },
	}

	t := template.New("reminders.html").Funcs(funcMap)
	t, err := t.ParseFiles("templates/base.html", "templates/reminders.html")
	if err != nil {
		fmt.Printf("ParseFiles error: %v\n", err)
		return
	}

	app := &handlers.App{
		Templates: map[string]*template.Template{
			"reminders.html": t,
		},
	}

	// Test 1: Full page request
	req1 := httptest.NewRequest("GET", "/calendar", nil)
	rec1 := httptest.NewRecorder()
	app.RemindersHandler(rec1, req1)
	fmt.Printf("Full page request status: %d\n", rec1.Code)
	if rec1.Code != 200 {
		fmt.Printf("Full page response: %s\n", rec1.Body.String())
	}

	// Test 2: HX-Request
	req2 := httptest.NewRequest("GET", "/calendar", nil)
	req2.Header.Set("HX-Request", "true")
	rec2 := httptest.NewRecorder()
	app.RemindersHandler(rec2, req2)
	fmt.Printf("HX-Request status: %d\n", rec2.Code)
	if rec2.Code != 200 {
		fmt.Printf("HX-Request response: %s\n", rec2.Body.String())
	}
}
