package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"radline/db"
)

func BenchmarkAdjustStockHandler(b *testing.B) {
	// Initialize in-memory sqlite db for tests
	db.InitTestDB()

	// Ensure there are enough items
	for i := 1; i <= 1000; i++ {
		db.DB.Exec("INSERT INTO items (code, description, default_uom) VALUES (?, ?, ?)", "ITEM"+strconv.Itoa(i), "Desc "+strconv.Itoa(i), "pcs")
	}

	app := &App{} // Or initialized correctly if needed

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		form := url.Values{}
		form.Add("date", time.Now().Format("2006-01-02"))
		form.Add("remarks", "benchmark adjustment")

		for j := 1; j <= 500; j++ {
			form.Add("item_id", strconv.Itoa(j))
			form.Add("adjustment_qty", "1")
			form.Add("uom", "pcs")
			form.Add("cost", "10.0")
		}

		req, err := http.NewRequest("POST", "/inventory/adjust", bytes.NewBufferString(form.Encode()))
		if err != nil {
			b.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()

		b.StartTimer()
		app.AdjustStockHandler(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("expected status OK, got %v", rr.Code)
		}
	}
}
