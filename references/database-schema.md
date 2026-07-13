# Database Schema

> Back to [[00-index]] · Related: [[architecture]], [[database-migrations]]

## File
[db/db.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/db/db.go)

## Connection

```go
var DB *sqlx.DB  // Global singleton

func InitDB(datasource string) error {
    DB, err = sqlx.Connect("sqlite", datasource)  // "sqlite" not "sqlite3"!
    // Enable FK constraints
    DB.Exec("PRAGMA foreign_keys = ON;")
    // Run inline migrations
    // Create schema
}
```

**Driver:** `modernc.org/sqlite` (pure Go). See [[tech-stack]] for rationale.

## Tables

### `brands`
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `code` | TEXT | UNIQUE NOT NULL |
| `name` | TEXT | NOT NULL |

### `categories`
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `code` | TEXT | UNIQUE NOT NULL |
| `name` | TEXT | NOT NULL |

### `items`
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `code` | TEXT | UNIQUE NOT NULL |
| `description` | TEXT | NOT NULL |
| `default_uom` | TEXT | NOT NULL |
| `model` | TEXT | — |
| `brand_id` | INTEGER | FK → `brands(id)` |
| `category_id` | INTEGER | FK → `categories(id)` |
| `variation` | TEXT | — |
| `remarks` | TEXT | — |

### `uom_settings`
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `item_id` | INTEGER | NOT NULL, FK → `items(id)` |
| `muom` | TEXT | NOT NULL |
| `conversion_factor` | REAL | NOT NULL |

### `receiving_logs`
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `supplier` | TEXT | NOT NULL |
| `date` | DATETIME | NOT NULL |
| `pl_no` | TEXT | — |
| `item_id` | INTEGER | NOT NULL, FK → `items(id)` |
| `qty` | REAL | NOT NULL |
| `uom` | TEXT | NOT NULL |
| `unit_price` | REAL | — |
| `cost` | REAL | NOT NULL |
| `total_cost` | REAL | NOT NULL |
| `selling_price` | REAL | — |

### `sales_details`
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `doc_type` | TEXT | NOT NULL |
| `doc_status` | TEXT | NOT NULL DEFAULT 'POSTED' |
| `doc_date` | DATETIME | NOT NULL |
| `doc_number` | TEXT | NOT NULL |
| `customer_name` | TEXT | — |
| `supplier` | TEXT | NOT NULL |
| `item_id` | INTEGER | NOT NULL, FK → `items(id)` |
| `qty` | REAL | NOT NULL |
| `uom` | TEXT | NOT NULL |
| `price` | REAL | NOT NULL |
| `total_sales` | REAL | NOT NULL |
| `cost` | REAL | NOT NULL |
| `total_cost` | REAL | NOT NULL |
| `profit` | REAL | NOT NULL |

### `stock_adjustments` (Header)
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `date` | DATETIME | NOT NULL |
| `remarks` | TEXT | — |

### `inventory_adjustments` (Detail)
| Column | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `adjustment_id` | INTEGER | FK → `stock_adjustments(id)` |
| `date` | DATETIME | NOT NULL |
| `item_id` | INTEGER | NOT NULL, FK → `items(id)` |
| `uom` | TEXT | NOT NULL |
| `adjustment_qty` | REAL | NOT NULL |
| `cost` | REAL | NOT NULL |
| `remarks` | TEXT | — |

## Entity Relationship Diagram

```
brands ──────┐
             │ 1:N
categories ──┤
             │ 1:N
items ───────┤──→ uom_settings (1:N)
             │
             ├──→ receiving_logs (1:N)
             │
             ├──→ sales_details (1:N)
             │
             └──→ inventory_adjustments (1:N) ──→ stock_adjustments (N:1 header)
```

## Foreign Key Enforcement

SQLite does **not** enforce foreign keys by default. The `PRAGMA foreign_keys = ON` executed on startup enables cascade enforcement. This prevents:
- Deleting a brand/category that has items
- Deleting an item that has receiving logs, sales, or adjustments

## Related
- [[database-migrations]] — How schema evolves at startup
- [[models-layer]] — DTO structs mapping these tables
- [[domain-overview]] — Domain entities these tables represent
