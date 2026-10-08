package db

import (
	"database/sql"
	"fmt"
	"log"
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

var DB *DBConn

func InitDB(datasource string) error {
	driver := "sqlite"
	if strings.HasPrefix(datasource, "postgres://") || strings.HasPrefix(datasource, "postgresql://") {
		driver = "pgx"
	}

	sqlx.BindDriver("pgx", sqlx.DOLLAR)
	sqlx.BindDriver("sqlite", sqlx.QUESTION)

	rawDB, err := sqlx.Connect(driver, datasource)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", driver, err)
	}

	DB = &DBConn{
		DB:         rawDB,
		DriverName: driver,
	}

	if driver == "pgx" {
		DB.SetMaxOpenConns(25)
		DB.SetMaxIdleConns(5)
		DB.SetConnMaxLifetime(5 * time.Minute)
		DB.SetConnMaxIdleTime(2 * time.Minute)

		if err := createSchemaPostgres(); err != nil {
			return err
		}
	} else {
		// Enable foreign key constraints in SQLite
		if _, err := DB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			return err
		}
		runSQLiteMigrations()
		if err := createSchemaSQLite(); err != nil {
			return err
		}
	}

	_ = EnsureRemindersTable()
	seedDefaults()
	return nil
}

func runSQLiteMigrations() {
	// Safe migration check for receiving_logs
	_, err := DB.Exec("SELECT total_cost FROM receiving_logs LIMIT 0")
	if err != nil {
		var rowCount int
		errCount := DB.Get(&rowCount, "SELECT COUNT(*) FROM receiving_logs")
		if errCount == nil {
			if rowCount == 0 {
				_, _ = DB.Exec("DROP TABLE IF EXISTS receiving_logs;")
			} else {
				_, _ = DB.Exec("ALTER TABLE receiving_logs ADD COLUMN total_cost REAL NOT NULL DEFAULT 0.0;")
				_, _ = DB.Exec("ALTER TABLE receiving_logs ADD COLUMN selling_price REAL;")
			}
		}
	}

	// Safe migration check for sales_details
	_, err = DB.Exec("SELECT doc_type FROM sales_details LIMIT 0")
	if err != nil {
		var rowCount int
		errCount := DB.Get(&rowCount, "SELECT COUNT(*) FROM sales_details")
		if errCount == nil {
			if rowCount == 0 {
				_, _ = DB.Exec("DROP TABLE IF EXISTS sales_details;")
			} else {
				_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN doc_type TEXT NOT NULL DEFAULT 'SI';")
				_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN doc_status TEXT NOT NULL DEFAULT 'POSTED';")
				_, _ = DB.Exec("ALTER TABLE sales_details RENAME COLUMN invoice_no TO doc_number;")
				_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN customer_name TEXT;")
				_, _ = DB.Exec("ALTER TABLE sales_details RENAME COLUMN unit_price TO price;")
				_, _ = DB.Exec("ALTER TABLE sales_details RENAME COLUMN margin_amount TO profit;")
				_, _ = DB.Exec("ALTER TABLE sales_details RENAME COLUMN date TO doc_date;")
			}
		}
	}

	// Safe migration check for receiving_logs: rename channel to supplier
	_, err = DB.Exec("SELECT supplier FROM receiving_logs LIMIT 0")
	if err != nil {
		_, errTable := DB.Exec("SELECT channel FROM receiving_logs LIMIT 0")
		if errTable == nil {
			_, _ = DB.Exec("ALTER TABLE receiving_logs RENAME COLUMN channel TO supplier;")
		}
	}

	// Safe migration: add less1, less2, markup, remarks columns to receiving_logs
	_, err = DB.Exec("SELECT less1 FROM receiving_logs LIMIT 0")
	if err != nil {
		_, _ = DB.Exec("ALTER TABLE receiving_logs ADD COLUMN less1 REAL NOT NULL DEFAULT 0;")
		_, _ = DB.Exec("ALTER TABLE receiving_logs ADD COLUMN less2 REAL NOT NULL DEFAULT 0;")
		_, _ = DB.Exec("ALTER TABLE receiving_logs ADD COLUMN markup REAL NOT NULL DEFAULT 130;")
		_, _ = DB.Exec("ALTER TABLE receiving_logs ADD COLUMN remarks TEXT NOT NULL DEFAULT '';")
	}

	// Safe migration check for sales_details: rename channel to supplier
	_, err = DB.Exec("SELECT supplier FROM sales_details LIMIT 0")
	if err != nil {
		_, errTable := DB.Exec("SELECT channel FROM sales_details LIMIT 0")
		if errTable == nil {
			_, _ = DB.Exec("ALTER TABLE sales_details RENAME COLUMN channel TO supplier;")
		}
	}

	// Safe migration: add adjustment_id column to inventory_adjustments if it doesn't exist
	_, err = DB.Exec("SELECT adjustment_id FROM inventory_adjustments LIMIT 0")
	if err != nil {
		var rowCount int
		errCount := DB.Get(&rowCount, "SELECT COUNT(*) FROM inventory_adjustments")
		if errCount == nil {
			_, _ = DB.Exec("ALTER TABLE inventory_adjustments ADD COLUMN adjustment_id INTEGER;")
		}
	}

	// Safe migration: add ref_pl, patong, pos_charge, wt_2307, total_remit, profit_margin, remarks columns to sales_details if they don't exist
	_, err = DB.Exec("SELECT ref_pl FROM sales_details LIMIT 0")
	if err != nil {
		var rowCount int
		errCount := DB.Get(&rowCount, "SELECT COUNT(*) FROM sales_details")
		if errCount == nil {
			_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN ref_pl TEXT;")
		}
	}

	_, err = DB.Exec("SELECT patong FROM sales_details LIMIT 0")
	if err != nil {
		var rowCount int
		errCount := DB.Get(&rowCount, "SELECT COUNT(*) FROM sales_details")
		if errCount == nil {
			_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN patong REAL NOT NULL DEFAULT 0.0;")
			_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN pos_charge REAL NOT NULL DEFAULT 0.0;")
			_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN wt_2307 REAL NOT NULL DEFAULT 0.0;")
			_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN total_remit REAL NOT NULL DEFAULT 0.0;")
			_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN profit_margin REAL NOT NULL DEFAULT 0.0;")
			_, _ = DB.Exec("ALTER TABLE sales_details ADD COLUMN remarks TEXT NOT NULL DEFAULT '';")
		}
	}
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
		FOREIGN KEY(item_id) REFERENCES items(id)
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
		FOREIGN KEY(adjustment_id) REFERENCES stock_adjustments(id),
		FOREIGN KEY(item_id) REFERENCES items(id)
	);

	CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_items_code ON items(code);
	CREATE INDEX IF NOT EXISTS idx_uom_settings_item_id ON uom_settings(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_item_id ON receiving_logs(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_date ON receiving_logs(date);
	CREATE INDEX IF NOT EXISTS idx_sales_details_item_id ON sales_details(item_id);
	CREATE INDEX IF NOT EXISTS idx_sales_details_doc_date ON sales_details(doc_date);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_item_id ON inventory_adjustments(item_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_date ON inventory_adjustments(date);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_adj_id ON inventory_adjustments(adjustment_id);

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
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_earning_goals_period ON earning_goals(target_period);
	CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date);
	CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);
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
		item_id INTEGER NOT NULL REFERENCES items(id),
		muom TEXT NOT NULL,
		conversion_factor DOUBLE PRECISION NOT NULL
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
		adjustment_id INTEGER REFERENCES stock_adjustments(id),
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
	CREATE INDEX IF NOT EXISTS idx_uom_settings_item_id ON uom_settings(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_item_id ON receiving_logs(item_id);
	CREATE INDEX IF NOT EXISTS idx_receiving_logs_date ON receiving_logs(date);
	CREATE INDEX IF NOT EXISTS idx_sales_details_item_id ON sales_details(item_id);
	CREATE INDEX IF NOT EXISTS idx_sales_details_doc_date ON sales_details(doc_date);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_item_id ON inventory_adjustments(item_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_date ON inventory_adjustments(date);
	CREATE INDEX IF NOT EXISTS idx_inventory_adjustments_adj_id ON inventory_adjustments(adjustment_id);

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
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_earning_goals_period ON earning_goals(target_period);
	CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date);
	CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);

	CREATE OR REPLACE FUNCTION substr(val timestamptz, s int, l int) RETURNS text AS $$
	SELECT substr(to_char(val, 'YYYY-MM-DD HH24:MI:SS'), s, l);
	$$ LANGUAGE SQL IMMUTABLE;

	CREATE OR REPLACE FUNCTION substr(val timestamp, s int, l int) RETURNS text AS $$
	SELECT substr(to_char(val, 'YYYY-MM-DD HH24:MI:SS'), s, l);
	$$ LANGUAGE SQL IMMUTABLE;

	CREATE OR REPLACE FUNCTION substr(val date, s int, l int) RETURNS text AS $$
	SELECT substr(to_char(val, 'YYYY-MM-DD'), s, l);
	$$ LANGUAGE SQL IMMUTABLE;
	`
	_, err := DB.DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create postgres schema: %w", err)
	}
	return nil
}

func seedDefaults() {
	// Seed default UOM values if table is empty
	var uomCount int
	err := DB.Get(&uomCount, "SELECT COUNT(*) FROM uoms")
	if err == nil && uomCount == 0 {
		_, errSeed := DB.Exec(`
			INSERT INTO uoms (code) VALUES
			('PC/S'), ('PCS'), ('BOX'), ('PACK/S'), ('PACK'), ('CASE'), ('BAG/S'), ('BAG'), ('BOT'), ('CAN/S'), ('CAN'), ('DOZ'), ('FT'), ('GAL'), ('KG/S'), ('KG'), ('L/S'), ('L'), ('MTR/S'), ('MTR'), ('PAIR/S'), ('ROLL/S'), ('SET/S'), ('UNIT/S')
		`)
		if errSeed != nil {
			log.Printf("Failed to seed default UOMs: %v", errSeed)
		}
	}

	seedAuthDefaults()
	seedGoalsAndRemindersDefaults()
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
				assigned_to_user_id INTEGER REFERENCES users(id),
				created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_earning_goals_period ON earning_goals(target_period);
			CREATE INDEX IF NOT EXISTS idx_reminders_due_date ON reminders(due_date);
			CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);
		`)
		return err
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
			assigned_to_user_id INTEGER REFERENCES users(id),
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
		if DB.IsPostgres() {
			_, _ = DB.Exec("ALTER TABLE reminders ADD COLUMN IF NOT EXISTS assigned_to_user_id INTEGER REFERENCES users(id);")
		} else {
			_, _ = DB.Exec("ALTER TABLE reminders ADD COLUMN assigned_to_user_id INTEGER;")
		}
	} else {
		_ = rows.Close()
	}

	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_reminders_assigned_to ON reminders(assigned_to_user_id);")

	return nil
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

func seedAuthDefaults() {
	var roleCount int
	err := DB.Get(&roleCount, "SELECT COUNT(*) FROM roles")
	if err != nil || roleCount > 0 {
		return
	}

	// Insert Admin role
	var adminRoleID int64
	err = DB.QueryRow("INSERT INTO roles (name, description) VALUES ('Admin', 'Full administrative access across all system modules') RETURNING id").Scan(&adminRoleID)
	if err != nil {
		log.Printf("Failed to seed Admin role: %v", err)
		return
	}

	// Insert Manager role
	var managerRoleID int64
	_ = DB.QueryRow("INSERT INTO roles (name, description) VALUES ('Manager', 'Access to sales, inventory operations, and viewing logs') RETURNING id").Scan(&managerRoleID)

	// Insert Staff role
	var staffRoleID int64
	_ = DB.QueryRow("INSERT INTO roles (name, description) VALUES ('Staff', 'Daily sales entry and inventory view access') RETURNING id").Scan(&staffRoleID)

	allPerms := []string{
		"dashboard:view", "sales:view", "sales:create", "sales:edit", "sales:delete",
		"inventory:view", "inventory:receive", "inventory:adjust",
		"items:view", "items:edit", "settings:view", "settings:edit",
		"settings:roles", "settings:users", "logs:view", "tools:import", "tools:scanner",
	}

	// Assign all perms to Admin
	for _, p := range allPerms {
		_, _ = DB.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", adminRoleID, p)
	}

	// Assign Manager perms
	managerPerms := []string{
		"dashboard:view", "sales:view", "sales:create", "sales:edit",
		"inventory:view", "inventory:receive", "inventory:adjust",
		"items:view", "items:edit", "settings:view", "logs:view", "tools:scanner",
	}
	for _, p := range managerPerms {
		_, _ = DB.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", managerRoleID, p)
	}

	// Assign Staff perms
	staffPerms := []string{
		"dashboard:view", "sales:view", "sales:create", "inventory:view", "items:view",
	}
	for _, p := range staffPerms {
		_, _ = DB.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", staffRoleID, p)
	}

	// Seed default users: admin / admin123 and staff / staff123
	bytesAdmin, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	bytesStaff, _ := bcrypt.GenerateFromPassword([]byte("staff123"), bcrypt.DefaultCost)

	_, _ = DB.Exec(`
		INSERT INTO users (username, password_hash, full_name, role_id, is_active)
		VALUES (?, ?, 'System Administrator', ?, TRUE)
	`, "admin", string(bytesAdmin), adminRoleID)

	_, _ = DB.Exec(`
		INSERT INTO users (username, password_hash, full_name, role_id, is_active)
		VALUES (?, ?, 'Staff Operator', ?, TRUE)
	`, "staff", string(bytesStaff), staffRoleID)
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
