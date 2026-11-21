package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// DBConn wraps sqlx.DB to automatically rebind SQL parameter placeholders
// across different SQL dialects (SQLite '?' vs PostgreSQL '$1, $2, ...').
type DBConn struct {
	*sqlx.DB
	DriverName string
}

func (db *DBConn) Rebind(query string) string {
	if db == nil || db.DB == nil {
		return query
	}
	return db.DB.Rebind(query)
}

func (db *DBConn) Get(dest interface{}, query string, args ...interface{}) error {
	return db.DB.Get(dest, db.Rebind(query), args...)
}

func (db *DBConn) Select(dest interface{}, query string, args ...interface{}) error {
	return db.DB.Select(dest, db.Rebind(query), args...)
}

func (db *DBConn) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.DB.Exec(db.Rebind(query), args...)
}

func (db *DBConn) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(db.Rebind(query), args...)
}

func (db *DBConn) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	return db.DB.Queryx(db.Rebind(query), args...)
}

func (db *DBConn) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRow(db.Rebind(query), args...)
}

func (db *DBConn) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	return db.DB.QueryRowx(db.Rebind(query), args...)
}

func (db *DBConn) Prepare(query string) (*sql.Stmt, error) {
	return db.DB.Prepare(db.Rebind(query))
}

func (db *DBConn) Preparex(query string) (*sqlx.Stmt, error) {
	return db.DB.Preparex(db.Rebind(query))
}

func (db *DBConn) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.DB.PrepareContext(ctx, db.Rebind(query))
}

func (db *DBConn) PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error) {
	return db.DB.PreparexContext(ctx, db.Rebind(query))
}

func (db *DBConn) Beginx() (*TxConn, error) {
	tx, err := db.DB.Beginx()
	if err != nil {
		return nil, err
	}
	return &TxConn{Tx: tx, db: db}, nil
}

func (db *DBConn) IsPostgres() bool {
	return db != nil && (db.DriverName == "pgx" || db.DriverName == "postgres")
}

// TxConn wraps sqlx.Tx to automatically rebind parameters for transactions.
type TxConn struct {
	*sqlx.Tx
	db *DBConn
}

func (tx *TxConn) Rebind(query string) string {
	if tx == nil || tx.Tx == nil {
		return query
	}
	return tx.Tx.Rebind(query)
}

func (tx *TxConn) Exec(query string, args ...interface{}) (sql.Result, error) {
	return tx.Tx.Exec(tx.Rebind(query), args...)
}

func (tx *TxConn) Get(dest interface{}, query string, args ...interface{}) error {
	return tx.Tx.Get(dest, tx.Rebind(query), args...)
}

func (tx *TxConn) Select(dest interface{}, query string, args ...interface{}) error {
	return tx.Tx.Select(dest, tx.Rebind(query), args...)
}

func (tx *TxConn) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return tx.Tx.Query(tx.Rebind(query), args...)
}

func (tx *TxConn) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	return tx.Tx.Queryx(tx.Rebind(query), args...)
}

func (tx *TxConn) QueryRow(query string, args ...interface{}) *sql.Row {
	return tx.Tx.QueryRow(tx.Rebind(query), args...)
}

func (tx *TxConn) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	return tx.Tx.QueryRowx(tx.Rebind(query), args...)
}

func (tx *TxConn) Prepare(query string) (*sql.Stmt, error) {
	return tx.Tx.Prepare(tx.Rebind(query))
}

func (tx *TxConn) Preparex(query string) (*sqlx.Stmt, error) {
	return tx.Tx.Preparex(tx.Rebind(query))
}

func (tx *TxConn) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return tx.Tx.PrepareContext(ctx, tx.Rebind(query))
}

func (tx *TxConn) PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error) {
	return tx.Tx.PreparexContext(ctx, tx.Rebind(query))
}

var DB *DBConn

func InitDB(datasource string) error {
	driver := "sqlite"
	if strings.HasPrefix(datasource, "postgres://") || strings.HasPrefix(datasource, "postgresql://") {
		driver = "pgx"
	}

	sqlx.BindDriver("pgx", sqlx.DOLLAR)
	sqlx.BindDriver("sqlite", sqlx.QUESTION)

	if DB != nil && DB.DB != nil {
		_ = DB.DB.Close()
		DB = nil
	}

	rawDB, err := sqlx.Connect(driver, datasource)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", driver, err)
	}

	DB = &DBConn{
		DB:         rawDB,
		DriverName: driver,
	}

	initErr := func() error {
		if driver == "pgx" {
			DB.SetMaxOpenConns(25)
			DB.SetMaxIdleConns(5)
			DB.SetConnMaxLifetime(5 * time.Minute)
			DB.SetConnMaxIdleTime(2 * time.Minute)

			if err := createSchemaPostgres(); err != nil {
				return err
			}
			if err := runPostgresMigrations(); err != nil {
				return err
			}
		} else {
			DB.SetMaxOpenConns(1)
			if _, err := DB.Exec("PRAGMA journal_mode = WAL;"); err != nil {
				return fmt.Errorf("failed to enable SQLite WAL mode: %w", err)
			}
			if _, err := DB.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
				return fmt.Errorf("failed to set SQLite busy_timeout: %w", err)
			}
			if _, err := DB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
				return fmt.Errorf("failed to enable SQLite foreign_keys: %w", err)
			}
			if err := createSchemaSQLite(); err != nil {
				return err
			}
			if err := runSQLiteMigrations(); err != nil {
				return err
			}
		}

		if err := EnsureRemindersTable(); err != nil {
			return err
		}
		if err := EnsureCustomersTable(); err != nil {
			return err
		}
		if err := seedDefaults(); err != nil {
			return err
		}
		return nil
	}()
	if initErr != nil {
		_ = rawDB.Close()
		DB = nil
		return initErr
	}
	return nil
}

func sqliteHasColumn(tx *TxConn, table, column string) (bool, error) {
	type pragmaCol struct {
		CID       int     `db:"cid"`
		Name      string  `db:"name"`
		Type      string  `db:"type"`
		NotNull   int     `db:"notnull"`
		DfltValue *string `db:"dflt_value"`
		PK        int     `db:"pk"`
	}
	var cols []pragmaCol
	if err := tx.Select(&cols, "PRAGMA table_info("+table+")"); err != nil {
		return false, fmt.Errorf("inspecting table %s: %w", table, err)
	}
	for _, c := range cols {
		if strings.EqualFold(c.Name, column) {
			return true, nil
		}
	}
	return false, nil
}

func runSQLiteMigrations() error {
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin SQLite migration transaction: %w", err)
	}
	defer tx.Rollback()

	// receiving_logs migrations
	hasTotalCost, err := sqliteHasColumn(tx, "receiving_logs", "total_cost")
	if err != nil {
		return err
	}
	if !hasTotalCost {
		if _, err := tx.Exec("ALTER TABLE receiving_logs ADD COLUMN total_cost REAL NOT NULL DEFAULT 0.0;"); err != nil {
			return err
		}
	}
	hasSellingPrice, err := sqliteHasColumn(tx, "receiving_logs", "selling_price")
	if err != nil {
		return err
	}
	if !hasSellingPrice {
		if _, err := tx.Exec("ALTER TABLE receiving_logs ADD COLUMN selling_price REAL;"); err != nil {
			return err
		}
	}
	hasRecSupplier, err := sqliteHasColumn(tx, "receiving_logs", "supplier")
	if err != nil {
		return err
	}
	if !hasRecSupplier {
		hasRecChannel, err := sqliteHasColumn(tx, "receiving_logs", "channel")
		if err != nil {
			return err
		}
		if hasRecChannel {
			if _, err := tx.Exec("ALTER TABLE receiving_logs RENAME COLUMN channel TO supplier;"); err != nil {
				return err
			}
		}
	}
	recCols := []struct {
		name string
		ddl  string
	}{
		{"less1", "ALTER TABLE receiving_logs ADD COLUMN less1 REAL NOT NULL DEFAULT 0;"},
		{"less2", "ALTER TABLE receiving_logs ADD COLUMN less2 REAL NOT NULL DEFAULT 0;"},
		{"markup", "ALTER TABLE receiving_logs ADD COLUMN markup REAL NOT NULL DEFAULT 130;"},
		{"remarks", "ALTER TABLE receiving_logs ADD COLUMN remarks TEXT NOT NULL DEFAULT '';"},
	}
	for _, c := range recCols {
		has, err := sqliteHasColumn(tx, "receiving_logs", c.name)
		if err != nil {
			return err
		}
		if !has {
			if _, err := tx.Exec(c.ddl); err != nil {
				return err
			}
		}
	}

	// sales_details migrations
	hasDocType, err := sqliteHasColumn(tx, "sales_details", "doc_type")
	if err != nil {
		return err
	}
	if !hasDocType {
		if _, err := tx.Exec("ALTER TABLE sales_details ADD COLUMN doc_type TEXT NOT NULL DEFAULT 'SI';"); err != nil {
			return err
		}
	}
	hasDocStatus, err := sqliteHasColumn(tx, "sales_details", "doc_status")
	if err != nil {
		return err
	}
	if !hasDocStatus {
		if _, err := tx.Exec("ALTER TABLE sales_details ADD COLUMN doc_status TEXT NOT NULL DEFAULT 'POSTED';"); err != nil {
			return err
		}
	}
	salesRenames := []struct {
		newCol string
		oldCol string
	}{
		{"doc_number", "invoice_no"},
		{"price", "unit_price"},
		{"profit", "margin_amount"},
		{"doc_date", "date"},
		{"supplier", "channel"},
	}
	for _, r := range salesRenames {
		hasNew, err := sqliteHasColumn(tx, "sales_details", r.newCol)
		if err != nil {
			return err
		}
		if !hasNew {
			hasOld, err := sqliteHasColumn(tx, "sales_details", r.oldCol)
			if err != nil {
				return err
			}
			if hasOld {
				if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE sales_details RENAME COLUMN %s TO %s;", r.oldCol, r.newCol)); err != nil {
					return err
				}
			}
		}
	}
	salesAdds := []struct {
		name string
		ddl  string
	}{
		{"customer_name", "ALTER TABLE sales_details ADD COLUMN customer_name TEXT;"},
		{"ref_pl", "ALTER TABLE sales_details ADD COLUMN ref_pl TEXT;"},
		{"patong", "ALTER TABLE sales_details ADD COLUMN patong REAL NOT NULL DEFAULT 0.0;"},
		{"pos_charge", "ALTER TABLE sales_details ADD COLUMN pos_charge REAL NOT NULL DEFAULT 0.0;"},
		{"wt_2307", "ALTER TABLE sales_details ADD COLUMN wt_2307 REAL NOT NULL DEFAULT 0.0;"},
		{"total_remit", "ALTER TABLE sales_details ADD COLUMN total_remit REAL NOT NULL DEFAULT 0.0;"},
		{"profit_margin", "ALTER TABLE sales_details ADD COLUMN profit_margin REAL NOT NULL DEFAULT 0.0;"},
		{"remarks", "ALTER TABLE sales_details ADD COLUMN remarks TEXT NOT NULL DEFAULT '';"},
		{"tin_no", "ALTER TABLE sales_details ADD COLUMN tin_no TEXT DEFAULT '';"},
		{"address", "ALTER TABLE sales_details ADD COLUMN address TEXT DEFAULT '';"},
	}
	for _, c := range salesAdds {
		has, err := sqliteHasColumn(tx, "sales_details", c.name)
		if err != nil {
			return err
		}
		if !has {
			if _, err := tx.Exec(c.ddl); err != nil {
				return err
			}
		}
	}

	// inventory_adjustments migrations
	hasAdjID, err := sqliteHasColumn(tx, "inventory_adjustments", "adjustment_id")
	if err != nil {
		return err
	}
	if !hasAdjID {
		if _, err := tx.Exec("ALTER TABLE inventory_adjustments ADD COLUMN adjustment_id INTEGER;"); err != nil {
			return err
		}
	}

	// reminders migrations
	hasAssignedTo, err := sqliteHasColumn(tx, "reminders", "assigned_to_user_id")
	if err != nil {
		return err
	}
	if !hasAssignedTo {
		if _, err := tx.Exec("ALTER TABLE reminders ADD COLUMN assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;"); err != nil {
			return err
		}
	}

	// Safe deduplication + unique index for uom_settings(item_id, muom)
	if _, err := tx.Exec("DELETE FROM uom_settings WHERE id NOT IN (SELECT MIN(id) FROM uom_settings GROUP BY item_id, muom);"); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_uom_settings_item_muom ON uom_settings(item_id, muom);"); err != nil {
		return err
	}

	// Ensure indexes exist on legacy and fresh databases after migrations run
	indexStmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_receiving_logs_supplier ON receiving_logs(supplier);",
		"CREATE INDEX IF NOT EXISTS idx_receiving_logs_pl_no ON receiving_logs(pl_no);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_doc_date ON sales_details(doc_date);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_doc_number ON sales_details(doc_number);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_doc_status ON sales_details(doc_status);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_supplier ON sales_details(supplier);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_customer_name ON sales_details(customer_name);",
		"CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_adj_id ON inventory_adjustments(adjustment_id);",
		"CREATE INDEX IF NOT EXISTS idx_reminders_assigned_to ON reminders(assigned_to_user_id);",
	}
	for _, stmt := range indexStmts {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func runPostgresMigrations() error {
	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin PostgreSQL migration transaction: %w", err)
	}
	defer tx.Rollback()

	migrations := []string{
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS doc_type TEXT NOT NULL DEFAULT 'SI';",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS doc_status TEXT NOT NULL DEFAULT 'POSTED';",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS customer_name TEXT DEFAULT '';",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS tin_no TEXT DEFAULT '';",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS address TEXT DEFAULT '';",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS supplier TEXT NOT NULL DEFAULT 'OTHER';",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS patong DOUBLE PRECISION NOT NULL DEFAULT 0.0;",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS pos_charge DOUBLE PRECISION NOT NULL DEFAULT 0.0;",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS wt_2307 DOUBLE PRECISION NOT NULL DEFAULT 0.0;",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS total_remit DOUBLE PRECISION NOT NULL DEFAULT 0.0;",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS profit_margin DOUBLE PRECISION NOT NULL DEFAULT 0.0;",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS remarks TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE sales_details ADD COLUMN IF NOT EXISTS ref_pl TEXT;",
		"ALTER TABLE receiving_logs ADD COLUMN IF NOT EXISTS supplier TEXT NOT NULL DEFAULT 'OTHER';",
		"ALTER TABLE receiving_logs ADD COLUMN IF NOT EXISTS less1 DOUBLE PRECISION NOT NULL DEFAULT 0;",
		"ALTER TABLE receiving_logs ADD COLUMN IF NOT EXISTS less2 DOUBLE PRECISION NOT NULL DEFAULT 0;",
		"ALTER TABLE receiving_logs ADD COLUMN IF NOT EXISTS markup DOUBLE PRECISION NOT NULL DEFAULT 130;",
		"ALTER TABLE receiving_logs ADD COLUMN IF NOT EXISTS remarks TEXT NOT NULL DEFAULT '';",
		"ALTER TABLE receiving_logs ADD COLUMN IF NOT EXISTS total_cost DOUBLE PRECISION NOT NULL DEFAULT 0.0;",
		"ALTER TABLE receiving_logs ADD COLUMN IF NOT EXISTS selling_price DOUBLE PRECISION;",
		"ALTER TABLE inventory_adjustments ADD COLUMN IF NOT EXISTS adjustment_id INTEGER REFERENCES stock_adjustments(id) ON DELETE CASCADE;",
		"ALTER TABLE reminders ADD COLUMN IF NOT EXISTS assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;",
		"CREATE TABLE IF NOT EXISTS purchase_orders (id SERIAL PRIMARY KEY, po_no TEXT NOT NULL UNIQUE, date TIMESTAMPTZ NOT NULL, due_date TIMESTAMPTZ, vendor_name TEXT NOT NULL, vendor_tin TEXT DEFAULT '', vendor_address TEXT DEFAULT '', vendor_contact TEXT DEFAULT '', ship_to TEXT DEFAULT '', payment_terms TEXT DEFAULT '', shipping_method TEXT DEFAULT '', prepared_by TEXT DEFAULT '', freight DOUBLE PRECISION NOT NULL DEFAULT 0.0, apply_vat BOOLEAN NOT NULL DEFAULT FALSE, vat_amt DOUBLE PRECISION NOT NULL DEFAULT 0.0, subtotal DOUBLE PRECISION NOT NULL DEFAULT 0.0, grand_total DOUBLE PRECISION NOT NULL DEFAULT 0.0, notes TEXT DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP);",
		"CREATE TABLE IF NOT EXISTS purchase_order_items (id SERIAL PRIMARY KEY, po_id INTEGER NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE, item_code TEXT DEFAULT '', description TEXT NOT NULL, uom TEXT DEFAULT '', qty DOUBLE PRECISION NOT NULL DEFAULT 0.0, unit_cost DOUBLE PRECISION NOT NULL DEFAULT 0.0, total_amount DOUBLE PRECISION NOT NULL DEFAULT 0.0);",
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'purchase_orders'
				  AND column_name = 'apply_vat' AND data_type IN ('integer', 'smallint', 'bigint')
			) THEN
				ALTER TABLE purchase_orders ALTER COLUMN apply_vat DROP DEFAULT;
				ALTER TABLE purchase_orders ALTER COLUMN apply_vat TYPE BOOLEAN USING (apply_vat <> 0);
				ALTER TABLE purchase_orders ALTER COLUMN apply_vat SET DEFAULT FALSE;
			END IF;
		END $$;`,
		"DELETE FROM uom_settings WHERE id NOT IN (SELECT MIN(id) FROM uom_settings GROUP BY item_id, muom);",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_uom_settings_item_muom ON uom_settings(item_id, muom);",
		"CREATE INDEX IF NOT EXISTS idx_receiving_logs_supplier ON receiving_logs(supplier);",
		"CREATE INDEX IF NOT EXISTS idx_receiving_logs_pl_no ON receiving_logs(pl_no);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_doc_number ON sales_details(doc_number);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_doc_status ON sales_details(doc_status);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_supplier ON sales_details(supplier);",
		"CREATE INDEX IF NOT EXISTS idx_sales_details_customer_name ON sales_details(customer_name);",
		"CREATE INDEX IF NOT EXISTS idx_reminders_assigned_to ON reminders(assigned_to_user_id);",
	}

	for _, stmt := range migrations {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("PostgreSQL migration failed (%s): %w", stmt, err)
		}
	}
	return tx.Commit()
}

func createSchemaSQLite() error {
	schema := `
	CREATE TABLE IF NOT EXISTS brands (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS uoms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL,
		default_uom TEXT NOT NULL,
		model TEXT,
		brand_id INTEGER,
		category_id INTEGER,
		variation TEXT,
		remarks TEXT,
		FOREIGN KEY(brand_id) REFERENCES brands(id),
		FOREIGN KEY(category_id) REFERENCES categories(id)
	);

	CREATE TABLE IF NOT EXISTS uom_settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER NOT NULL,
		muom TEXT NOT NULL,
		conversion_factor REAL NOT NULL,
		UNIQUE(item_id, muom),
		FOREIGN KEY(item_id) REFERENCES items(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS receiving_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		supplier TEXT NOT NULL,
		date DATETIME NOT NULL,
		pl_no TEXT,
		item_id INTEGER NOT NULL,
		qty REAL NOT NULL,
		uom TEXT NOT NULL,
		unit_price REAL,
		less1 REAL NOT NULL DEFAULT 0,
		less2 REAL NOT NULL DEFAULT 0,
		cost REAL NOT NULL,
		total_cost REAL NOT NULL,
		markup REAL NOT NULL DEFAULT 130,
		selling_price REAL,
		remarks TEXT NOT NULL DEFAULT '',
		FOREIGN KEY(item_id) REFERENCES items(id)
	);

	CREATE TABLE IF NOT EXISTS sales_details (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		doc_type TEXT NOT NULL,
		doc_status TEXT NOT NULL DEFAULT 'POSTED',
		doc_date DATETIME NOT NULL,
		doc_number TEXT NOT NULL,
		customer_name TEXT,
		tin_no TEXT DEFAULT '',
		address TEXT DEFAULT '',
		supplier TEXT NOT NULL,
		item_id INTEGER NOT NULL,
		qty REAL NOT NULL,
		uom TEXT NOT NULL,
		price REAL NOT NULL,
		total_sales REAL NOT NULL,
		cost REAL NOT NULL,
		total_cost REAL NOT NULL,
		patong REAL NOT NULL DEFAULT 0.0,
		pos_charge REAL NOT NULL DEFAULT 0.0,
		wt_2307 REAL NOT NULL DEFAULT 0.0,
		total_remit REAL NOT NULL DEFAULT 0.0,
		profit REAL NOT NULL,
		profit_margin REAL NOT NULL DEFAULT 0.0,
		remarks TEXT NOT NULL DEFAULT '',
		ref_pl TEXT,
		FOREIGN KEY(item_id) REFERENCES items(id)
	);

	CREATE TABLE IF NOT EXISTS stock_adjustments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date DATETIME NOT NULL,
		remarks TEXT
	);

	CREATE TABLE IF NOT EXISTS inventory_adjustments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		adjustment_id INTEGER,
		date DATETIME NOT NULL,
		item_id INTEGER NOT NULL,
		uom TEXT NOT NULL,
		adjustment_qty REAL NOT NULL,
		cost REAL NOT NULL,
		remarks TEXT,
		FOREIGN KEY(adjustment_id) REFERENCES stock_adjustments(id) ON DELETE CASCADE,
		FOREIGN KEY(item_id) REFERENCES items(id)
	);

	CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_items_code ON items(code);
	CREATE INDEX IF NOT EXISTS idx_items_description ON items(description);
	CREATE INDEX IF NOT EXISTS idx_items_model ON items(model);
	CREATE INDEX IF NOT EXISTS idx_items_brand_id ON items(brand_id);
	CREATE INDEX IF NOT EXISTS idx_items_category_id ON items(category_id);
	CREATE INDEX IF NOT EXISTS idx_uom_settings_item_id ON uom_settings(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_item_id ON receiving_logs(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_date ON receiving_logs(date);
	CREATE INDEX IF NOT EXISTS idx_sales_details_item_id ON sales_details(item_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_item_id ON inventory_adjustments(item_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_date ON inventory_adjustments(date);

	CREATE TABLE IF NOT EXISTS roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS role_permissions (
		role_id INTEGER NOT NULL,
		permission_key TEXT NOT NULL,
		PRIMARY KEY(role_id, permission_key),
		FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT NOT NULL,
		role_id INTEGER NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(role_id) REFERENCES roles(id)
	);

	CREATE TABLE IF NOT EXISTS user_sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS user_preferences (
		user_id INTEGER NOT NULL,
		pref_key TEXT NOT NULL,
		pref_value TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, pref_key),
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS activity_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		username TEXT NOT NULL,
		action TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL DEFAULT '',
		details TEXT NOT NULL DEFAULT '',
		ip_address TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs(created_at);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_username ON activity_logs(username);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_entity_type ON activity_logs(entity_type);

	CREATE TABLE IF NOT EXISTS earning_goals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		period_type TEXT NOT NULL,
		target_period TEXT NOT NULL,
		target_revenue REAL NOT NULL,
		target_profit REAL NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS reminders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		details TEXT NOT NULL DEFAULT '',
		due_date DATETIME NOT NULL,
		priority TEXT NOT NULL DEFAULT 'Medium',
		category TEXT NOT NULL DEFAULT 'General',
		status TEXT NOT NULL DEFAULT 'Pending',
		assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_earning_goals_period ON earning_goals(target_period);
	CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date);
	CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);

	CREATE TABLE IF NOT EXISTS purchase_orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		po_no TEXT NOT NULL UNIQUE,
		date DATETIME NOT NULL,
		due_date DATETIME,
		vendor_name TEXT NOT NULL,
		vendor_tin TEXT DEFAULT '',
		vendor_address TEXT DEFAULT '',
		vendor_contact TEXT DEFAULT '',
		ship_to TEXT DEFAULT '',
		payment_terms TEXT DEFAULT '',
		shipping_method TEXT DEFAULT '',
		prepared_by TEXT DEFAULT '',
		freight REAL NOT NULL DEFAULT 0.0,
		apply_vat BOOLEAN NOT NULL DEFAULT 0,
		vat_amt REAL NOT NULL DEFAULT 0.0,
		subtotal REAL NOT NULL DEFAULT 0.0,
		grand_total REAL NOT NULL DEFAULT 0.0,
		notes TEXT DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS purchase_order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		po_id INTEGER NOT NULL,
		item_code TEXT DEFAULT '',
		description TEXT NOT NULL,
		uom TEXT DEFAULT '',
		qty REAL NOT NULL DEFAULT 0.0,
		unit_cost REAL NOT NULL DEFAULT 0.0,
		total_amount REAL NOT NULL DEFAULT 0.0,
		FOREIGN KEY(po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_purchase_orders_po_no ON purchase_orders(po_no);
	CREATE INDEX IF NOT EXISTS idx_purchase_orders_date ON purchase_orders(date);
	CREATE INDEX IF NOT EXISTS idx_purchase_order_items_po_id ON purchase_order_items(po_id);

	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		tin_no TEXT NOT NULL DEFAULT '',
		address TEXT NOT NULL DEFAULT '',
		contact_person TEXT NOT NULL DEFAULT '',
		phone TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		payment_terms TEXT NOT NULL DEFAULT 'Net 30 Days',
		notes TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_customers_code ON customers(code);
	CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);
	`
	_, err := DB.DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create sqlite schema: %w", err)
	}
	return nil
}

func createSchemaPostgres() error {
	schema := `
	CREATE TABLE IF NOT EXISTS brands (
		id SERIAL PRIMARY KEY,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS categories (
		id SERIAL PRIMARY KEY,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS uoms (
		id SERIAL PRIMARY KEY,
		code TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY,
		code TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL,
		default_uom TEXT NOT NULL,
		model TEXT,
		brand_id INTEGER REFERENCES brands(id),
		category_id INTEGER REFERENCES categories(id),
		variation TEXT,
		remarks TEXT
	);

	CREATE TABLE IF NOT EXISTS uom_settings (
		id SERIAL PRIMARY KEY,
		item_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
		muom TEXT NOT NULL,
		conversion_factor DOUBLE PRECISION NOT NULL,
		UNIQUE(item_id, muom)
	);

	CREATE TABLE IF NOT EXISTS receiving_logs (
		id SERIAL PRIMARY KEY,
		supplier TEXT NOT NULL,
		date TIMESTAMPTZ NOT NULL,
		pl_no TEXT,
		item_id INTEGER NOT NULL REFERENCES items(id),
		qty DOUBLE PRECISION NOT NULL,
		uom TEXT NOT NULL,
		unit_price DOUBLE PRECISION,
		less1 DOUBLE PRECISION NOT NULL DEFAULT 0,
		less2 DOUBLE PRECISION NOT NULL DEFAULT 0,
		cost DOUBLE PRECISION NOT NULL,
		total_cost DOUBLE PRECISION NOT NULL,
		markup DOUBLE PRECISION NOT NULL DEFAULT 130,
		selling_price DOUBLE PRECISION,
		remarks TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS sales_details (
		id SERIAL PRIMARY KEY,
		doc_type TEXT NOT NULL,
		doc_status TEXT NOT NULL DEFAULT 'POSTED',
		doc_date TIMESTAMPTZ NOT NULL,
		doc_number TEXT NOT NULL,
		customer_name TEXT,
		tin_no TEXT DEFAULT '',
		address TEXT DEFAULT '',
		supplier TEXT NOT NULL,
		item_id INTEGER NOT NULL REFERENCES items(id),
		qty DOUBLE PRECISION NOT NULL,
		uom TEXT NOT NULL,
		price DOUBLE PRECISION NOT NULL,
		total_sales DOUBLE PRECISION NOT NULL,
		cost DOUBLE PRECISION NOT NULL,
		total_cost DOUBLE PRECISION NOT NULL,
		patong DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		pos_charge DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		wt_2307 DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		total_remit DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		profit DOUBLE PRECISION NOT NULL,
		profit_margin DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		remarks TEXT NOT NULL DEFAULT '',
		ref_pl TEXT
	);

	CREATE TABLE IF NOT EXISTS stock_adjustments (
		id SERIAL PRIMARY KEY,
		date TIMESTAMPTZ NOT NULL,
		remarks TEXT
	);

	CREATE TABLE IF NOT EXISTS inventory_adjustments (
		id SERIAL PRIMARY KEY,
		adjustment_id INTEGER REFERENCES stock_adjustments(id) ON DELETE CASCADE,
		date TIMESTAMPTZ NOT NULL,
		item_id INTEGER NOT NULL REFERENCES items(id),
		uom TEXT NOT NULL,
		adjustment_qty DOUBLE PRECISION NOT NULL,
		cost DOUBLE PRECISION NOT NULL,
		remarks TEXT
	);

	CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_items_code ON items(code);
	CREATE INDEX IF NOT EXISTS idx_items_description ON items(description);
	CREATE INDEX IF NOT EXISTS idx_items_model ON items(model);
	CREATE INDEX IF NOT EXISTS idx_items_brand_id ON items(brand_id);
	CREATE INDEX IF NOT EXISTS idx_items_category_id ON items(category_id);
	CREATE INDEX IF NOT EXISTS idx_uom_settings_item_id ON uom_settings(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_item_id ON receiving_logs(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_date ON receiving_logs(date);
	CREATE INDEX IF NOT EXISTS idx_sales_details_item_id ON sales_details(item_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_item_id ON inventory_adjustments(item_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_date ON inventory_adjustments(date);

	CREATE TABLE IF NOT EXISTS roles (
		id SERIAL PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS role_permissions (
		role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
		permission_key TEXT NOT NULL,
		PRIMARY KEY(role_id, permission_key)
	);

	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT NOT NULL,
		role_id INTEGER NOT NULL REFERENCES roles(id),
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at TIMESTAMPTZ NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_preferences (
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		pref_key TEXT NOT NULL,
		pref_value TEXT NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, pref_key)
	);

	CREATE TABLE IF NOT EXISTS activity_logs (
		id SERIAL PRIMARY KEY,
		user_id INTEGER,
		username TEXT NOT NULL,
		action TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL DEFAULT '',
		details TEXT NOT NULL DEFAULT '',
		ip_address TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs(created_at);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_username ON activity_logs(username);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_entity_type ON activity_logs(entity_type);

	CREATE TABLE IF NOT EXISTS earning_goals (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		period_type TEXT NOT NULL,
		target_period TEXT NOT NULL,
		target_revenue DOUBLE PRECISION NOT NULL,
		target_profit DOUBLE PRECISION NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS reminders (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		details TEXT NOT NULL DEFAULT '',
		due_date TIMESTAMPTZ NOT NULL,
		priority TEXT NOT NULL DEFAULT 'Medium',
		category TEXT NOT NULL DEFAULT 'General',
		status TEXT NOT NULL DEFAULT 'Pending',
		assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_earning_goals_period ON earning_goals(target_period);
	CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date);
	CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);

	CREATE TABLE IF NOT EXISTS purchase_orders (
		id SERIAL PRIMARY KEY,
		po_no TEXT NOT NULL UNIQUE,
		date TIMESTAMPTZ NOT NULL,
		due_date TIMESTAMPTZ,
		vendor_name TEXT NOT NULL,
		vendor_tin TEXT DEFAULT '',
		vendor_address TEXT DEFAULT '',
		vendor_contact TEXT DEFAULT '',
		ship_to TEXT DEFAULT '',
		payment_terms TEXT DEFAULT '',
		shipping_method TEXT DEFAULT '',
		prepared_by TEXT DEFAULT '',
		freight DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		apply_vat BOOLEAN NOT NULL DEFAULT FALSE,
		vat_amt DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		subtotal DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		grand_total DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		notes TEXT DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS purchase_order_items (
		id SERIAL PRIMARY KEY,
		po_id INTEGER NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
		item_code TEXT DEFAULT '',
		description TEXT NOT NULL,
		uom TEXT DEFAULT '',
		qty DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		unit_cost DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		total_amount DOUBLE PRECISION NOT NULL DEFAULT 0.0
	);

	CREATE INDEX IF NOT EXISTS idx_purchase_orders_po_no ON purchase_orders(po_no);
	CREATE INDEX IF NOT EXISTS idx_purchase_orders_date ON purchase_orders(date);
	CREATE INDEX IF NOT EXISTS idx_purchase_order_items_po_id ON purchase_order_items(po_id);

	CREATE TABLE IF NOT EXISTS customers (
		id SERIAL PRIMARY KEY,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		tin_no TEXT NOT NULL DEFAULT '',
		address TEXT NOT NULL DEFAULT '',
		contact_person TEXT NOT NULL DEFAULT '',
		phone TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		payment_terms TEXT NOT NULL DEFAULT 'Net 30 Days',
		notes TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_customers_code ON customers(code);
	CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);

	CREATE OR REPLACE FUNCTION substr(val timestamptz, s int, l int) RETURNS text AS $$
	SELECT substr(to_char(val AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), s, l);
	$$ LANGUAGE SQL STABLE;

	CREATE OR REPLACE FUNCTION substr(val timestamp, s int, l int) RETURNS text AS $$
	SELECT substr(to_char(val, 'YYYY-MM-DD HH24:MI:SS'), s, l);
	$$ LANGUAGE SQL STABLE;

	CREATE OR REPLACE FUNCTION substr(val date, s int, l int) RETURNS text AS $$
	SELECT substr(to_char(val, 'YYYY-MM-DD'), s, l);
	$$ LANGUAGE SQL STABLE;
	`
	_, err := DB.DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create postgres schema: %w", err)
	}
	return nil
}

func seedDefaults() error {
	// Seed default UOM values if table is empty
	var uomCount int
	err := DB.Get(&uomCount, "SELECT COUNT(*) FROM uoms")
	if err != nil {
		return fmt.Errorf("failed to check uoms count: %w", err)
	}
	if uomCount == 0 {
		_, errSeed := DB.Exec(`
			INSERT INTO uoms (code) VALUES
			('PC/S'), ('PCS'), ('BOX'), ('PACK/S'), ('PACK'), ('CASE'), ('BAG/S'), ('BAG'), ('BOT'), ('CAN/S'), ('CAN'), ('DOZ'), ('FT'), ('GAL'), ('KG/S'), ('KG'), ('L/S'), ('L'), ('MTR/S'), ('MTR'), ('PAIR/S'), ('ROLL/S'), ('SET/S'), ('UNIT/S')
		`)
		if errSeed != nil {
			return fmt.Errorf("failed to seed default UOMs: %w", errSeed)
		}
	}

	if err := seedAuthDefaults(); err != nil {
		return err
	}
	seedGoalsAndRemindersDefaults()
	return nil
}

// EnsureRemindersTable guarantees the earning_goals and reminders tables and indexes exist
func EnsureRemindersTable() error {
	if DB == nil {
		return nil
	}
	if DB.IsPostgres() {
		_, err := DB.Exec(`
			CREATE TABLE IF NOT EXISTS earning_goals (
				id SERIAL PRIMARY KEY,
				title TEXT NOT NULL,
				period_type TEXT NOT NULL,
				target_period TEXT NOT NULL,
				target_revenue DOUBLE PRECISION NOT NULL,
				target_profit DOUBLE PRECISION NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE IF NOT EXISTS reminders (
				id SERIAL PRIMARY KEY,
				title TEXT NOT NULL,
				details TEXT NOT NULL DEFAULT '',
				due_date TIMESTAMPTZ NOT NULL,
				priority TEXT NOT NULL DEFAULT 'Medium',
				category TEXT NOT NULL DEFAULT 'General',
				status TEXT NOT NULL DEFAULT 'Pending',
				assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_earning_goals_period ON earning_goals(target_period);
			CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date);
			CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);
		`)
		if err != nil {
			return err
		}
		_, _ = DB.Exec("ALTER TABLE reminders ADD COLUMN IF NOT EXISTS assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;")
		_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_reminders_assigned_to ON reminders(assigned_to_user_id);")
		return nil
	}
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS earning_goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			period_type TEXT NOT NULL,
			target_period TEXT NOT NULL,
			target_revenue REAL NOT NULL,
			target_profit REAL NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS reminders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
			due_date DATETIME NOT NULL,
			priority TEXT NOT NULL DEFAULT 'Medium',
			category TEXT NOT NULL DEFAULT 'General',
			status TEXT NOT NULL DEFAULT 'Pending',
			assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_earning_goals_period ON earning_goals(target_period);
		CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date);
		CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);
	`)
	if err != nil {
		return err
	}

	// Safe migration check: add assigned_to_user_id if not present
	rows, err := DB.Query("SELECT assigned_to_user_id FROM reminders LIMIT 0")
	if err != nil {
		_, _ = DB.Exec("ALTER TABLE reminders ADD COLUMN assigned_to_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;")
	} else {
		_ = rows.Close()
	}

	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_reminders_assigned_to ON reminders(assigned_to_user_id);")
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_items_description ON items(description);")
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_items_model ON items(model);")
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_items_brand_id ON items(brand_id);")
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_items_category_id ON items(category_id);")

	return nil
}

// EnsureCustomersTable guarantees the customers table and indexes exist
func EnsureCustomersTable() error {
	if DB == nil {
		return nil
	}
	if DB.IsPostgres() {
		_, err := DB.Exec(`
			CREATE TABLE IF NOT EXISTS customers (
				id SERIAL PRIMARY KEY,
				code TEXT UNIQUE NOT NULL,
				name TEXT NOT NULL,
				tin_no TEXT NOT NULL DEFAULT '',
				address TEXT NOT NULL DEFAULT '',
				contact_person TEXT NOT NULL DEFAULT '',
				phone TEXT NOT NULL DEFAULT '',
				email TEXT NOT NULL DEFAULT '',
				payment_terms TEXT NOT NULL DEFAULT 'Net 30 Days',
				notes TEXT NOT NULL DEFAULT '',
				created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_customers_code ON customers(code);
			CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);
		`)
		return err
	}
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS customers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			tin_no TEXT NOT NULL DEFAULT '',
			address TEXT NOT NULL DEFAULT '',
			contact_person TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			payment_terms TEXT NOT NULL DEFAULT 'Net 30 Days',
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_customers_code ON customers(code);
		CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);
	`)
	return err
}

func seedGoalsAndRemindersDefaults() {
	var goalCount int
	_ = DB.Get(&goalCount, "SELECT COUNT(*) FROM earning_goals")
	if goalCount == 0 {
		now := time.Now()
		curMonth := now.Format("2006-01")
		curYear := now.Format("2006")
		_, _ = DB.Exec(`
			INSERT INTO earning_goals (title, period_type, target_period, target_revenue, target_profit)
			VALUES 
			(?, 'Monthly', ?, 500000.0, 140000.0),
			(?, 'Yearly', ?, 4500000.0, 1200000.0)
		`, "Monthly Revenue & Profit Sprint", curMonth, "Annual Milestone Growth", curYear)
	}

	var remCount int
	_ = DB.Get(&remCount, "SELECT COUNT(*) FROM reminders")
	if remCount == 0 {
		now := time.Now()
		d1 := now.AddDate(0, 0, 2)
		d2 := now.AddDate(0, 0, 5)
		d3 := now.AddDate(0, 0, 7)
		d4 := now.AddDate(0, 0, 10)
		_, _ = DB.Exec(`
			INSERT INTO reminders (title, details, due_date, priority, category, status)
			VALUES
			('Monthly Stock Inventory Reconciliation', 'Audit physical count against warehouse records', ?, 'High', 'Stock Check', 'Pending'),
			('Supplier Delivery Verification', 'Cross-check PL documents with incoming hardware crates', ?, 'Medium', 'Delivery', 'Pending'),
			('BIR 2307 Withholding Tax Filing', 'Prepare tax certificates for accredited vendors', ?, 'High', 'Payment', 'Pending'),
			('Hardware Fasteners Restock Review', 'Evaluate screw and bolt stock levels against low-stock threshold', ?, 'Low', 'General', 'Pending')
		`, d1, d2, d3, d4)
	}
}

func resolveSeedPassword(envVar, devFallback string, isDevOrTest, required bool) (password string, generated bool, err error) {
	if pw := strings.TrimSpace(os.Getenv(envVar)); pw != "" {
		if len(pw) < 8 || len([]byte(pw)) > 72 {
			return "", false, fmt.Errorf("%s must be at least 8 characters and at most 72 bytes", envVar)
		}
		return pw, false, nil
	}
	if isDevOrTest {
		return devFallback, false, nil
	}
	if !required {
		return "", false, nil
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", false, fmt.Errorf("failed to generate random bootstrap password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), true, nil
}

func seedAuthDefaults() error {
	var roleCount int
	err := DB.Get(&roleCount, "SELECT COUNT(*) FROM roles")
	if err != nil {
		return fmt.Errorf("failed to check roles count: %w", err)
	}
	if roleCount > 0 {
		return nil
	}

	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	isDevOrTest := (appEnv == "" || appEnv == "development" || appEnv == "dev" || appEnv == "test" || appEnv == "local") && !(DB.IsPostgres() && appEnv == "")

	adminPassword, adminGenerated, err := resolveSeedPassword("ADMIN_SEED_PASSWORD", "admin123", isDevOrTest, true)
	if err != nil {
		return err
	}
	staffPassword, _, err := resolveSeedPassword("STAFF_SEED_PASSWORD", "staff123", isDevOrTest, false)
	if err != nil {
		return err
	}

	bytesAdmin, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin seed password: %w", err)
	}
	var bytesStaff []byte
	if staffPassword != "" {
		bytesStaff, err = bcrypt.GenerateFromPassword([]byte(staffPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash staff seed password: %w", err)
		}
	}

	tx, err := DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin auth seed transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert Admin role
	var adminRoleID int64
	if err := tx.QueryRow("INSERT INTO roles (name, description) VALUES ('Admin', 'Full administrative access across all system modules') RETURNING id").Scan(&adminRoleID); err != nil {
		return fmt.Errorf("failed to seed Admin role: %w", err)
	}

	// Insert Manager role
	var managerRoleID int64
	if err := tx.QueryRow("INSERT INTO roles (name, description) VALUES ('Manager', 'Access to sales, inventory operations, and viewing logs') RETURNING id").Scan(&managerRoleID); err != nil {
		return fmt.Errorf("failed to seed Manager role: %w", err)
	}

	// Insert Staff role
	var staffRoleID int64
	if err := tx.QueryRow("INSERT INTO roles (name, description) VALUES ('Staff', 'Daily sales entry and inventory view access') RETURNING id").Scan(&staffRoleID); err != nil {
		return fmt.Errorf("failed to seed Staff role: %w", err)
	}

	allPerms := []string{
		"dashboard:view", "sales:view", "sales:create", "sales:edit", "sales:delete",
		"inventory:view", "inventory:receive", "inventory:adjust",
		"items:view", "items:edit", "settings:view", "settings:edit",
		"settings:roles", "settings:users", "logs:view", "tools:import", "tools:scanner",
	}

	// Assign all perms to Admin
	for _, p := range allPerms {
		if _, err := tx.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", adminRoleID, p); err != nil {
			return fmt.Errorf("failed to seed Admin permission %s: %w", p, err)
		}
	}

	// Assign Manager perms
	managerPerms := []string{
		"dashboard:view", "sales:view", "sales:create", "sales:edit",
		"inventory:view", "inventory:receive", "inventory:adjust",
		"items:view", "items:edit", "settings:view", "logs:view", "tools:scanner",
	}
	for _, p := range managerPerms {
		if _, err := tx.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", managerRoleID, p); err != nil {
			return fmt.Errorf("failed to seed Manager permission %s: %w", p, err)
		}
	}

	// Assign Staff perms
	staffPerms := []string{
		"dashboard:view", "sales:view", "sales:create", "inventory:view", "items:view",
	}
	for _, p := range staffPerms {
		if _, err := tx.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", staffRoleID, p); err != nil {
			return fmt.Errorf("failed to seed Staff permission %s: %w", p, err)
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO users (username, password_hash, full_name, role_id, is_active)
		VALUES (?, ?, 'System Administrator', ?, TRUE)
	`, "admin", string(bytesAdmin), adminRoleID); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	if staffPassword != "" {
		if _, err := tx.Exec(`
			INSERT INTO users (username, password_hash, full_name, role_id, is_active)
			VALUES (?, ?, 'Staff Operator', ?, TRUE)
		`, "staff", string(bytesStaff), staffRoleID); err != nil {
			return fmt.Errorf("failed to seed staff user: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit auth seed transaction: %w", err)
	}

	if adminGenerated {
		log.Printf("BOOTSTRAP NOTICE: Created initial admin user 'admin' with generated one-time password: %s", adminPassword)
	}
	return nil
}

// GetSystemSetting retrieves a configuration value by key from system_settings
func GetSystemSetting(key string) string {
	if DB == nil {
		return ""
	}
	var val string
	err := DB.Get(&val, "SELECT value FROM system_settings WHERE key = ? LIMIT 1", key)
	if err != nil {
		return ""
	}
	return val
}

// SetSystemSetting inserts or updates a configuration key-value pair in system_settings
func SetSystemSetting(key, value string) error {
	if DB == nil {
		return nil
	}
	_, err := DB.Exec(`
		INSERT INTO system_settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	return err
}

// GetUserPreference retrieves a user preference value by user_id and pref_key
func GetUserPreference(userID int64, key string) (string, error) {
	if DB == nil {
		return "", fmt.Errorf("database connection is nil")
	}
	var val string
	err := DB.Get(&val, "SELECT pref_value FROM user_preferences WHERE user_id = ? AND pref_key = ? LIMIT 1", userID, key)
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetUserPreference inserts or updates a user preference
func SetUserPreference(userID int64, key, value string) error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}
	_, err := DB.Exec(`
		INSERT INTO user_preferences (user_id, pref_key, pref_value, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, pref_key) DO UPDATE SET pref_value = EXCLUDED.pref_value, updated_at = CURRENT_TIMESTAMP
	`, userID, key, value)
	return err
}

// DeleteUserPreference deletes a user preference
func DeleteUserPreference(userID int64, key string) error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}
	_, err := DB.Exec("DELETE FROM user_preferences WHERE user_id = ? AND pref_key = ?", userID, key)
	return err
}

// GetAllUserPreferences retrieves all preferences for a user as a key-value map
func GetAllUserPreferences(userID int64) (map[string]string, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	type prefRow struct {
		Key   string `db:"pref_key"`
		Value string `db:"pref_value"`
	}
	var rows []prefRow
	err := DB.Select(&rows, "SELECT pref_key, pref_value FROM user_preferences WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	res := make(map[string]string, len(rows))
	for _, r := range rows {
		res[r.Key] = r.Value
	}
	return res, nil
}
