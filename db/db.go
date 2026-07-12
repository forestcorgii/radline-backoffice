package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var DB *sqlx.DB

func InitDB(datasource string) error {
	var err error
	DB, err = sqlx.Connect("sqlite", datasource)
	if err != nil {
		return err
	}

	// Enable foreign key constraints in SQLite
	_, err = DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return err
	}

	// Safe migration check for receiving_logs
	_, err = DB.Exec("SELECT total_cost FROM receiving_logs LIMIT 0")
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
		// Table exists but column doesn't — add it
		var rowCount int
		errCount := DB.Get(&rowCount, "SELECT COUNT(*) FROM inventory_adjustments")
		if errCount == nil {
			_, _ = DB.Exec("ALTER TABLE inventory_adjustments ADD COLUMN adjustment_id INTEGER;")
		}
	}

	createSchema()
	return nil
}

func createSchema() {
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
		cost REAL NOT NULL,
		total_cost REAL NOT NULL,
		selling_price REAL,
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
		profit REAL NOT NULL,
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
	`
	_, err := DB.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
}
