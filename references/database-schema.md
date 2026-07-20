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
| `patong` | REAL | NOT NULL DEFAULT 0.0 |
| `pos_charge` | REAL | NOT NULL DEFAULT 0.0 |
| `wt_2307` | REAL | NOT NULL DEFAULT 0.0 |
| `total_remit` | REAL | NOT NULL DEFAULT 0.0 |
| `profit` | REAL | NOT NULL |
| `profit_margin` | REAL | NOT NULL DEFAULT 0.0 |
| `remarks` | TEXT | NOT NULL DEFAULT '' |
| `ref_pl` | TEXT | — |

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

## Learnings

### Context: Timezone-Safe SQLite Date Extraction
**Problem**: In SQLite, using `strftime('%Y-%m', date)` on ISO-8601 strings containing timezone offsets (e.g. `+08:00` or `Z`) can return `NULL` or empty, causing database scan errors in Go (e.g. `converting NULL to string is unsupported`).
**Enforced Solution**:
- Replace `strftime('%Y-%m', date_column)` with `substr(date_column, 1, 7)` in SQLite queries where YYYY-MM extraction is required. Since ISO-8601 datetimes consistently begin with `YYYY-MM-DD`, substring extraction is timezone-safe, parsing-independent, and extremely robust.

### Context: Batch Querying and Database Indexing for Page Performance
**Problem**: Fetching domain models individually inside page rendering loops creates an N+1 query pattern, which results in significant page load latency when handling larger datasets.
**Enforced Solution**:
- **Batch Querying**: Implement batch loader functions (e.g. `FetchItemsStockBatch(db, itemIDs)`) that execute single SQL `IN (?)` queries across all needed tables rather than executing separate queries per row loop.
- **Database Indexing**: Add standard indices on search text fields (e.g., `code`) and foreign key columns (`item_id`, `adjustment_id`) in SQLite schema definition to prevent full table scans on group by / filter queries.

### Context: Sales Financial Field Calculations & Auto-Remittance Schema
**Problem**: Managing comprehensive sales accounting requires tracking custom charges (`patong`, `pos_charge`, `wt_2307`), net remit, profit margin percentages, and remarks across encoding, batch edit, customize column views, and Excel import.
**Enforced Solution**:
- **Database Schema**: `sales_details` includes `patong`, `pos_charge`, `wt_2307`, `total_remit`, `profit_margin`, and `remarks` with default safe values.
- **Invariants & Formulas**:
  - `Total Sales (Total Price)` = `Qty * Price`
  - `Total Cost` = `Qty * Cost`
  - `Total Remit` = `Total Sales - (Patong + POS Charge + WT 2307)`
  - `Profit` = `Total Sales - Total Cost`
  - `Profit Margin (%)` = `(Profit / Total Sales) * 100` if `Total Sales > 0` else `0%`
- **UI & Import Alignment**: All fields are supported across list view columns, customize view popovers, real-time JS client-side encoding, and Excel sheet import.

## Related
- [[00-index]]
- [[database-migrations]]
- [[domain-sales]]
- [[database-migrations]] — How schema evolves at startup
- [[models-layer]] — DTO structs mapping these tables
- [[domain-overview]] — Domain entities these tables represent
