package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestDBConn_Rebind(t *testing.T) {
	sqlx.BindDriver("pgx", sqlx.DOLLAR)
	sqlx.BindDriver("sqlite", sqlx.QUESTION)

	pgDB := &DBConn{
		DB:         sqlx.NewDb(nil, "pgx"),
		DriverName: "pgx",
	}

	sqliteDB := &DBConn{
		DB:         sqlx.NewDb(nil, "sqlite"),
		DriverName: "sqlite",
	}

	query := "SELECT * FROM items WHERE id = ? AND code = ?"

	reboundPG := pgDB.Rebind(query)
	expectedPG := "SELECT * FROM items WHERE id = $1 AND code = $2"
	if reboundPG != expectedPG {
		t.Errorf("Expected PG rebound query %q, got %q", expectedPG, reboundPG)
	}

	reboundSQLite := sqliteDB.Rebind(query)
	if reboundSQLite != query {
		t.Errorf("Expected SQLite query %q, got %q", query, reboundSQLite)
	}
}

func TestDBConn_IsPostgres(t *testing.T) {
	pgDB := &DBConn{DriverName: "pgx"}
	if !pgDB.IsPostgres() {
		t.Errorf("Expected IsPostgres to be true for pgx")
	}

	sqliteDB := &DBConn{DriverName: "sqlite"}
	if sqliteDB.IsPostgres() {
		t.Errorf("Expected IsPostgres to be false for sqlite")
	}
}

func TestInitDB_SQLiteInMemory(t *testing.T) {
	err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init in-memory sqlite db: %v", err)
	}

	if DB == nil {
		t.Fatalf("Expected DB to be non-nil after InitDB")
	}

	if DB.IsPostgres() {
		t.Errorf("Expected driver not to be postgres")
	}

	// Verify setting get/set
	err = SetSystemSetting("company_name", "Radline Tools")
	if err != nil {
		t.Fatalf("Failed to set system setting: %v", err)
	}

	val := GetSystemSetting("company_name")
	if val != "Radline Tools" {
		t.Errorf("Expected system setting 'Radline Tools', got %q", val)
	}

	// Verify user preferences get/set/delete
	err = SetUserPreference(1, "dashboard_layout", `[{"id":"card-trends","width":"full","height":"large"}]`)
	if err != nil {
		t.Fatalf("Failed to set user preference: %v", err)
	}

	prefVal, err := GetUserPreference(1, "dashboard_layout")
	if err != nil {
		t.Fatalf("Failed to get user preference: %v", err)
	}
	if prefVal != `[{"id":"card-trends","width":"full","height":"large"}]` {
		t.Errorf("Expected user preference JSON, got %q", prefVal)
	}

	// Verify update on conflict
	err = SetUserPreference(1, "dashboard_layout", `[{"id":"card-workflow","width":"half","height":"small"}]`)
	if err != nil {
		t.Fatalf("Failed to update user preference: %v", err)
	}
	prefVal, err = GetUserPreference(1, "dashboard_layout")
	if err != nil || prefVal != `[{"id":"card-workflow","width":"half","height":"small"}]` {
		t.Errorf("Expected updated preference, got %q", prefVal)
	}

	// Verify delete
	err = DeleteUserPreference(1, "dashboard_layout")
	if err != nil {
		t.Fatalf("Failed to delete user preference: %v", err)
	}
	prefVal, _ = GetUserPreference(1, "dashboard_layout")
	if prefVal != "" {
		t.Errorf("Expected empty string after deleting user preference, got %q", prefVal)
	}

	// Verify RETURNING id works
	var brandID int64
	err = DB.QueryRow("INSERT INTO brands (code, name) VALUES (?, ?) RETURNING id", "TEST_BRAND", "Test Brand Name").Scan(&brandID)
	if err != nil {
		t.Fatalf("Failed to insert brand with RETURNING id: %v", err)
	}
	if brandID == 0 {
		t.Fatalf("Expected non-zero brandID")
	}
}
