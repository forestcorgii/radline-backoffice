package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkClearExisting(b *testing.B) {
	setupTestDBForImport(b)
	app := &App{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		populateDummyData(b, 50)
		req := createDummyImportRequest(b, true)
		rr := httptest.NewRecorder()
		b.StartTimer()

		app.ImportUploadHandler(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("Expected status OK, got %d", rr.Code)
		}
	}
}
