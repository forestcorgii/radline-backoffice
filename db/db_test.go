package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
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

func TestTxConn_PrepareAndRebind(t *testing.T) {
	sqlx.BindDriver("pgx", sqlx.DOLLAR)
	pgDB := &DBConn{
		DB:         sqlx.NewDb(nil, "pgx"),
		DriverName: "pgx",
	}
	rebound := pgDB.Rebind("INSERT INTO items (code, description) VALUES (?, ?)")
	expected := "INSERT INTO items (code, description) VALUES ($1, $2)"
	if rebound != expected {
		t.Fatalf("Expected %q, got %q", expected, rebound)
	}
	// Rebind must be idempotent for callers like ReceiveStockHandler that call tx.Prepare(tx.Rebind(...))
	if twice := pgDB.Rebind(rebound); twice != expected {
		t.Fatalf("Expected idempotent rebind %q, got %q", expected, twice)
	}

	if err := InitTestDB(); err != nil {
		t.Fatalf("InitTestDB failed: %v", err)
	}
	t.Cleanup(func() {
		if DB != nil {
			_ = DB.Close()
			DB = nil
		}
	})

	tx, err := DB.Beginx()
	if err != nil {
		t.Fatalf("Beginx failed: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO uoms (code) VALUES (?)")
	if err != nil {
		t.Fatalf("tx.Prepare failed: %v", err)
	}
	if _, err := stmt.Exec("TUBE"); err != nil {
		t.Fatalf("prepared stmt exec failed: %v", err)
	}
	_ = stmt.Close()

	xstmt, err := tx.Preparex("SELECT code FROM uoms WHERE code = ?")
	if err != nil {
		t.Fatalf("tx.Preparex failed: %v", err)
	}
	var got string
	if err := xstmt.Get(&got, "TUBE"); err != nil {
		t.Fatalf("prepared xstmt Get failed: %v", err)
	}
	_ = xstmt.Close()
	if got != "TUBE" {
		t.Fatalf("Expected TUBE, got %q", got)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
}

func TestSQLite_PragmasPoolAndForeignKeys(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "wal_test.db")
	if err := InitDB(tmpFile); err != nil {
		t.Fatalf("InitDB(file) failed: %v", err)
	}
	t.Cleanup(func() {
		if DB != nil {
			_ = DB.Close()
			DB = nil
		}
	})

	if maxConns := DB.Stats().MaxOpenConnections; maxConns != 1 {
		t.Errorf("Expected MaxOpenConnections == 1, got %d", maxConns)
	}

	var journalMode string
	if err := DB.Get(&journalMode, "PRAGMA journal_mode;"); err != nil {
		t.Fatalf("PRAGMA journal_mode failed: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Errorf("Expected journal_mode=wal, got %q", journalMode)
	}

	var busyTimeout int
	if err := DB.Get(&busyTimeout, "PRAGMA busy_timeout;"); err != nil {
		t.Fatalf("PRAGMA busy_timeout failed: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("Expected busy_timeout=5000, got %d", busyTimeout)
	}

	var fkEnabled int
	if err := DB.Get(&fkEnabled, "PRAGMA foreign_keys;"); err != nil {
		t.Fatalf("PRAGMA foreign_keys failed: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("Expected foreign_keys=1, got %d", fkEnabled)
	}

	// Verify FK enforcement rejects nonexistent item_id
	_, err := DB.Exec(`
		INSERT INTO receiving_logs (supplier, date, item_id, qty, uom, cost, total_cost)
		VALUES ('ASCD', '2026-01-01', 999999, 1, 'PCS', 10, 10)
	`)
	if err == nil {
		t.Fatalf("Expected FK constraint error inserting receiving_log with nonexistent item_id")
	}
}

func TestSeedAuthDefaults_EnvGuarded(t *testing.T) {
	t.Cleanup(func() {
		if DB != nil {
			_ = DB.Close()
			DB = nil
		}
	})

	// 1. Dev/test default (empty APP_ENV on SQLite) seeds admin/admin123 & staff/staff123
	t.Setenv("APP_ENV", "")
	t.Setenv("ADMIN_SEED_PASSWORD", "")
	t.Setenv("STAFF_SEED_PASSWORD", "")
	if err := InitDB(":memory:"); err != nil {
		t.Fatalf("InitDB dev failed: %v", err)
	}
	var adminHash, staffHash string
	if err := DB.Get(&adminHash, "SELECT password_hash FROM users WHERE username = 'admin'"); err != nil {
		t.Fatalf("Expected seeded admin user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(adminHash), []byte("admin123")); err != nil {
		t.Errorf("Expected dev admin password to be admin123: %v", err)
	}
	if err := DB.Get(&staffHash, "SELECT password_hash FROM users WHERE username = 'staff'"); err != nil {
		t.Fatalf("Expected seeded staff user in dev: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(staffHash), []byte("staff123")); err != nil {
		t.Errorf("Expected dev staff password to be staff123: %v", err)
	}

	// 2. Production without env overrides generates random admin password and skips staff
	t.Setenv("APP_ENV", "production")
	t.Setenv("ADMIN_SEED_PASSWORD", "")
	t.Setenv("STAFF_SEED_PASSWORD", "")
	if err := InitDB(":memory:"); err != nil {
		t.Fatalf("InitDB production failed: %v", err)
	}
	if err := DB.Get(&adminHash, "SELECT password_hash FROM users WHERE username = 'admin'"); err != nil {
		t.Fatalf("Expected admin user in production: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(adminHash), []byte("admin123")); err == nil {
		t.Fatalf("Production admin must NOT use hardcoded admin123 password")
	}
	var staffCount int
	if err := DB.Get(&staffCount, "SELECT COUNT(*) FROM users WHERE username = 'staff'"); err != nil {
		t.Fatalf("Failed checking staff user count: %v", err)
	}
	if staffCount != 0 {
		t.Fatalf("Expected no staff user seeded in production without STAFF_SEED_PASSWORD, got %d", staffCount)
	}

	// 3. Production with explicit ADMIN_SEED_PASSWORD and STAFF_SEED_PASSWORD honors both
	t.Setenv("APP_ENV", "production")
	t.Setenv("ADMIN_SEED_PASSWORD", "ProdAdminPass#2026")
	t.Setenv("STAFF_SEED_PASSWORD", "ProdStaffPass#2026")
	if err := InitDB(":memory:"); err != nil {
		t.Fatalf("InitDB production with env passwords failed: %v", err)
	}
	if err := DB.Get(&adminHash, "SELECT password_hash FROM users WHERE username = 'admin'"); err != nil {
		t.Fatalf("Expected admin user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(adminHash), []byte("ProdAdminPass#2026")); err != nil {
		t.Errorf("Admin password override did not match: %v", err)
	}
	if err := DB.Get(&staffHash, "SELECT password_hash FROM users WHERE username = 'staff'"); err != nil {
		t.Fatalf("Expected staff user with STAFF_SEED_PASSWORD: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(staffHash), []byte("ProdStaffPass#2026")); err != nil {
		t.Errorf("Staff password override did not match: %v", err)
	}

	// 4. Short seed password fails validation
	t.Setenv("ADMIN_SEED_PASSWORD", "short")
	if err := InitDB(":memory:"); err == nil {
		t.Fatalf("Expected error for <8 char ADMIN_SEED_PASSWORD")
	}
}

func TestInitTestDB_21TablesAndOnDeleteAndLegacyMigration(t *testing.T) {
	t.Cleanup(func() {
		if DB != nil {
			_ = DB.Close()
			DB = nil
		}
	})

	if err := InitTestDB(); err != nil {
		t.Fatalf("InitTestDB failed: %v", err)
	}

	expectedTables := []string{
		"brands", "categories", "uoms", "items", "uom_settings",
		"receiving_logs", "sales_details", "stock_adjustments", "inventory_adjustments",
		"system_settings", "roles", "role_permissions", "users", "user_sessions",
		"user_preferences", "activity_logs", "earning_goals", "reminders",
		"purchase_orders", "purchase_order_items", "customers",
	}
	for _, tbl := range expectedTables {
		var cnt int
		if err := DB.Get(&cnt, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl); err != nil || cnt != 1 {
			t.Errorf("Expected table %q to exist in InitTestDB(), cnt=%d err=%v", tbl, cnt, err)
		}
	}

	// Verify ON DELETE CASCADE on uom_settings(item_id), RESTRICT on receiving_logs(item_id),
	// ON DELETE CASCADE on inventory_adjustments(adjustment_id), and ON DELETE SET NULL on reminders(assigned_to_user_id)
	var itemFreeID, itemUsedID int64
	if err := DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('FREE-1', 'Free Item', 'PCS') RETURNING id").Scan(&itemFreeID); err != nil {
		t.Fatalf("Insert itemFree failed: %v", err)
	}
	if _, err := DB.Exec("INSERT INTO uom_settings (item_id, muom, conversion_factor) VALUES (?, 'BOX', 12)", itemFreeID); err != nil {
		t.Fatalf("Insert uom_settings failed: %v", err)
	}
	if _, err := DB.Exec("DELETE FROM items WHERE id = ?", itemFreeID); err != nil {
		t.Fatalf("Expected deleting item with only uom_settings to cascade, got error: %v", err)
	}
	var uomSettingCount int
	_ = DB.Get(&uomSettingCount, "SELECT COUNT(*) FROM uom_settings WHERE item_id = ?", itemFreeID)
	if uomSettingCount != 0 {
		t.Errorf("Expected uom_settings to cascade delete, got %d remaining", uomSettingCount)
	}

	if err := DB.QueryRow("INSERT INTO items (code, description, default_uom) VALUES ('USED-1', 'Used Item', 'PCS') RETURNING id").Scan(&itemUsedID); err != nil {
		t.Fatalf("Insert itemUsed failed: %v", err)
	}
	if _, err := DB.Exec("INSERT INTO receiving_logs (supplier, date, item_id, qty, uom, cost, total_cost) VALUES ('ASCD', '2026-01-01', ?, 5, 'PCS', 10, 50)", itemUsedID); err != nil {
		t.Fatalf("Insert receiving_log failed: %v", err)
	}
	if _, err := DB.Exec("DELETE FROM items WHERE id = ?", itemUsedID); err == nil {
		t.Errorf("Expected FK RESTRICT error deleting item referenced by receiving_logs")
	}

	var adjHeaderID int64
	if err := DB.QueryRow("INSERT INTO stock_adjustments (date, remarks) VALUES ('2026-01-01', 'hdr') RETURNING id").Scan(&adjHeaderID); err != nil {
		t.Fatalf("Insert stock_adjustment failed: %v", err)
	}
	if _, err := DB.Exec("INSERT INTO inventory_adjustments (adjustment_id, date, item_id, uom, adjustment_qty, cost) VALUES (?, '2026-01-01', ?, 'PCS', 1, 10)", adjHeaderID, itemUsedID); err != nil {
		t.Fatalf("Insert inventory_adjustment failed: %v", err)
	}
	if _, err := DB.Exec("DELETE FROM stock_adjustments WHERE id = ?", adjHeaderID); err != nil {
		t.Fatalf("Expected deleting stock_adjustments header to cascade, got %v", err)
	}
	var adjCount int
	_ = DB.Get(&adjCount, "SELECT COUNT(*) FROM inventory_adjustments WHERE adjustment_id = ?", adjHeaderID)
	if adjCount != 0 {
		t.Errorf("Expected inventory_adjustments to cascade delete, got %d", adjCount)
	}

	var remUserID int64
	if err := DB.QueryRow("INSERT INTO users (username, password_hash, full_name, role_id, is_active) VALUES ('rem_del_u', 'h', 'Rem Del', 1, TRUE) RETURNING id").Scan(&remUserID); err != nil {
		t.Fatalf("Insert user failed: %v", err)
	}
	var remID int64
	if err := DB.QueryRow("INSERT INTO reminders (title, due_date, assigned_to_user_id) VALUES ('Rem Set Null', '2026-02-01', ?) RETURNING id", remUserID).Scan(&remID); err != nil {
		t.Fatalf("Insert reminder failed: %v", err)
	}
	if _, err := DB.Exec("DELETE FROM users WHERE id = ?", remUserID); err != nil {
		t.Fatalf("Expected deleting assigned user to succeed with ON DELETE SET NULL, got %v", err)
	}
	var assignedAfter *int64
	if err := DB.Get(&assignedAfter, "SELECT assigned_to_user_id FROM reminders WHERE id = ?", remID); err != nil {
		t.Fatalf("Query reminder after user delete failed: %v", err)
	}
	if assignedAfter != nil {
		t.Errorf("Expected assigned_to_user_id to be NULL after user delete, got %v", *assignedAfter)
	}

	// Verify purchase_orders.apply_vat bool roundtrip
	if _, err := DB.Exec("INSERT INTO purchase_orders (po_no, date, vendor_name, apply_vat) VALUES ('PO-BOOL-1', '2026-01-01', 'Vendor', ?), ('PO-BOOL-2', '2026-01-01', 'Vendor', ?)", true, false); err != nil {
		t.Fatalf("Insert purchase_orders failed: %v", err)
	}
	var vatTrue, vatFalse bool
	if err := DB.Get(&vatTrue, "SELECT apply_vat FROM purchase_orders WHERE po_no = 'PO-BOOL-1'"); err != nil || !vatTrue {
		t.Errorf("Expected apply_vat=true, got %v (err=%v)", vatTrue, err)
	}
	if err := DB.Get(&vatFalse, "SELECT apply_vat FROM purchase_orders WHERE po_no = 'PO-BOOL-2'"); err != nil || vatFalse {
		t.Errorf("Expected apply_vat=false, got %v (err=%v)", vatFalse, err)
	}

	// Verify legacy SQLite DB migration: missing columns, duplicate uom_settings, and new indexes
	legacyDir := t.TempDir()
	legacyPath := filepath.Join(legacyDir, "legacy.db")
	rawLegacy, err := sqlx.Connect("sqlite", legacyPath)
	if err != nil {
		t.Fatalf("Open legacy db failed: %v", err)
	}
	legacySchema := `
		CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT UNIQUE NOT NULL, description TEXT NOT NULL, default_uom TEXT NOT NULL, model TEXT, brand_id INTEGER, category_id INTEGER, variation TEXT, remarks TEXT);
		CREATE TABLE uom_settings (id INTEGER PRIMARY KEY AUTOINCREMENT, item_id INTEGER NOT NULL, muom TEXT NOT NULL, conversion_factor REAL NOT NULL);
		CREATE TABLE reminders (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, details TEXT NOT NULL DEFAULT '', due_date DATETIME NOT NULL, priority TEXT NOT NULL DEFAULT 'Medium', category TEXT NOT NULL DEFAULT 'General', status TEXT NOT NULL DEFAULT 'Pending', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO items (code, description, default_uom) VALUES ('LEG-1', 'Legacy Item', 'PCS');
		INSERT INTO uom_settings (item_id, muom, conversion_factor) VALUES (1, 'BOX', 10), (1, 'BOX', 12);
	`
	if _, err := rawLegacy.Exec(legacySchema); err != nil {
		_ = rawLegacy.Close()
		t.Fatalf("Seed legacy schema failed: %v", err)
	}
	_ = rawLegacy.Close()

	if err := InitDB(legacyPath); err != nil {
		t.Fatalf("InitDB on legacy DB failed: %v", err)
	}
	defer func() {
		if DB != nil {
			_ = DB.Close()
			DB = nil
		}
		_ = os.Remove(legacyPath)
	}()
	var dedupCount int
	if err := DB.Get(&dedupCount, "SELECT COUNT(*) FROM uom_settings WHERE item_id = 1 AND muom = 'BOX'"); err != nil {
		t.Fatalf("Query deduped uom_settings failed: %v", err)
	}
	if dedupCount != 1 {
		t.Errorf("Expected duplicate uom_settings to be deduplicated to 1 row, got %d", dedupCount)
	}
	if _, err := DB.Exec("INSERT INTO uom_settings (item_id, muom, conversion_factor) VALUES (1, 'BOX', 24)"); err == nil {
		t.Errorf("Expected unique index idx_uom_settings_item_muom to reject duplicate insert")
	}
	expectedIndexes := []string{
		"idx_uom_settings_item_muom",
		"idx_receiving_logs_supplier",
		"idx_receiving_logs_pl_no",
		"idx_sales_details_doc_number",
		"idx_sales_details_doc_status",
		"idx_sales_details_supplier",
		"idx_sales_details_customer_name",
		"idx_reminders_assigned_to",
	}
	for _, idxName := range expectedIndexes {
		var cnt int
		if err := DB.Get(&cnt, "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idxName); err != nil || cnt != 1 {
			t.Errorf("Expected index %q to exist, cnt=%d err=%v", idxName, cnt, err)
		}
	}
}
