package handlers

import (
	"testing"

	"rav/db"
)

func setupIsolatedTestDB(t *testing.T) {
	t.Helper()
	if err := db.InitTestDB(); err != nil {
		t.Fatalf("Failed to init isolated test DB: %v", err)
	}
	t.Cleanup(func() {
		if db.DB != nil {
			_ = db.DB.Close()
			db.DB = nil
		}
	})
}
