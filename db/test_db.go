package db

import (
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func InitTestDB() {
	dbConn, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}

	DB = &DBConn{
		DB:         dbConn,
		DriverName: "sqlite",
	}

	createTablesQuery := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL,
		default_uom TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS stock_adjustments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date DATE NOT NULL,
		remarks TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS inventory_adjustments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		adjustment_id INTEGER,
		date DATE NOT NULL,
		item_id INTEGER NOT NULL,
		uom TEXT NOT NULL,
		adjustment_qty REAL NOT NULL,
		cost REAL NOT NULL DEFAULT 0.0,
		remarks TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	DB.MustExec(createTablesQuery)
}
