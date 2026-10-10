package handlers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"radline/db"

	"github.com/xuri/excelize/v2"
)

func setupTestDBForImport(t testing.TB) {
	err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}
}

func populateDummyData(t testing.TB, count int) {
	tx, err := db.DB.Beginx()
	if err != nil {
		t.Fatalf("Failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	for i := 1; i <= count; i++ {
		bCode := fmt.Sprintf("B%d", i)
		cCode := fmt.Sprintf("C%d", i)
		iCode := fmt.Sprintf("ITEM%d", i)

		var brandID int
		err := tx.QueryRow("INSERT INTO brands (code, name) VALUES (?, ?) RETURNING id", bCode, "Brand "+bCode).Scan(&brandID)
		if err != nil {
			t.Fatalf("Failed to insert brand: %v", err)
		}

		var catID int
		err = tx.QueryRow("INSERT INTO categories (code, name) VALUES (?, ?) RETURNING id", cCode, "Category "+cCode).Scan(&catID)
		if err != nil {
			t.Fatalf("Failed to insert category: %v", err)
		}

		var itemID int
		err = tx.QueryRow("INSERT INTO items (code, description, default_uom, brand_id, category_id) VALUES (?, ?, 'pcs', ?, ?) RETURNING id", iCode, "Item "+iCode, brandID, catID).Scan(&itemID)
		if err != nil {
			t.Fatalf("Failed to insert item: %v", err)
		}

		_, _ = tx.Exec("INSERT INTO uom_settings (item_id, muom, conversion_factor) VALUES (?, 'BOX', 10)", itemID)
		_, _ = tx.Exec("INSERT INTO receiving_logs (supplier, date, item_id, qty, uom, unit_price, cost, total_cost) VALUES ('Supp', '2025-01-01', ?, 10, 'pcs', 5, 5, 50)", itemID)
		_, _ = tx.Exec("INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit) VALUES ('SI', 'Posted', '2025-01-01', 'INV1', 'Supp', ?, 1, 'pcs', 10, 10, 5, 5, 5)", itemID)

		var adjHeaderID int
		_ = tx.QueryRow("INSERT INTO stock_adjustments (date, remarks) VALUES ('2025-01-01', 'Test adj') RETURNING id").Scan(&adjHeaderID)
		_, _ = tx.Exec("INSERT INTO inventory_adjustments (adjustment_id, date, item_id, uom, adjustment_qty, cost) VALUES (?, '2025-01-01', ?, 'pcs', 1, 5)", adjHeaderID, itemID)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit dummy data: %v", err)
	}
}

func createDummyImportRequest(t testing.TB, clearExisting bool) *http.Request {
	f := excelize.NewFile()
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("Failed to write excel to buffer: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.xlsx")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	_, _ = part.Write(buf.Bytes())

	if clearExisting {
		_ = writer.WriteField("clear_existing", "true")
	}

	_ = writer.Close()

	req, err := http.NewRequest("POST", "/import/upload", body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestClearExisting(t *testing.T) {
	setupTestDBForImport(t)
	populateDummyData(t, 10)

	app := &App{}

	// Verify counts before clearing
	var count int
	_ = db.DB.Get(&count, "SELECT COUNT(*) FROM items")
	if count != 10 {
		t.Fatalf("Expected 10 items before clear, got %d", count)
	}

	req := createDummyImportRequest(t, true)
	rr := httptest.NewRecorder()

	app.ImportUploadHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	tablesToVerify := []string{
		"inventory_adjustments",
		"stock_adjustments",
		"sales_details",
		"receiving_logs",
		"uom_settings",
		"items",
		"brands",
		"categories",
	}

	for _, tbl := range tablesToVerify {
		var tblCount int
		err := db.DB.Get(&tblCount, fmt.Sprintf("SELECT COUNT(*) FROM %s", tbl))
		if err != nil {
			t.Fatalf("Error counting rows in %s: %v", tbl, err)
		}
		if tblCount != 0 {
			t.Errorf("Expected 0 rows in %s after clear_existing, got %d", tbl, tblCount)
		}
	}
}
