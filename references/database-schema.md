# Database Schema & Dual-Engine Architecture

> Back to [[00-index]] · Related: [[database-migrations]], [[models-layer]], [[domain-overview]]

## File
[db/db.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/db/db.go)

## Dual-Engine Connection (`DBConn` & `TxConn`)
- **Driver Detection**: `InitDB(datasource)` connects via `pgx` (`github.com/jackc/pgx/v5/stdlib`) when `datasource` starts with `postgres://` or `postgresql://` (required when `APP_ENV=production`), otherwise uses pure-Go SQLite (`modernc.org/sqlite`, driver name `"sqlite"`, file `backoffice.db`) with `PRAGMA foreign_keys = ON;`.
- **Automatic Placeholder Rebinding**: `db.DB` (`*DBConn`) and `TxConn` wrap `*sqlx.DB` / `*sqlx.Tx` and automatically call `Rebind(query)` on every `Get`, `Select`, `Exec`, `Query`, and `QueryRow`, translating `?` into `$1, $2, ...` on PostgreSQL.

## Complete Table Catalog (17 Tables)

| Domain Area | Table | Key Columns & FKs |
|---|---|---|
| **Catalog Masterlist** | `brands` | `id`, `code` (UNIQUE), `name` |
| | `categories` | `id`, `code` (UNIQUE), `name` |
| | `uoms` | `id`, `code` (UNIQUE) |
| | `items` | `id`, `code` (UNIQUE), `description`, `default_uom`, `model`, `brand_id` (FK `brands`), `category_id` (FK `categories`), `variation`, `remarks` |
| | `uom_settings` | `id`, `item_id` (FK `items`), `muom`, `conversion_factor` |
| | `customers` | `id`, `code` (UNIQUE), `name`, `tin_no`, `address`, `contact_person`, `phone`, `email`, `payment_terms`, `notes`, `created_at` |
| **Inventory & Procurement** | `receiving_logs` | `id`, `supplier`, `date`, `pl_no`, `item_id` (FK `items`), `qty`, `uom`, `unit_price`, `less1`, `less2`, `cost`, `total_cost`, `markup`, `selling_price`, `remarks` |
| | `stock_adjustments` | `id`, `date`, `remarks` (Header table) |
| | `inventory_adjustments` | `id`, `adjustment_id` (FK `stock_adjustments`), `date`, `item_id` (FK `items`), `uom`, `adjustment_qty`, `cost`, `remarks` |
| | `purchase_orders` | `id`, `po_no` (UNIQUE), `date`, `due_date`, `vendor_name`, `vendor_tin`, `vendor_address`, `vendor_contact`, `ship_to`, `payment_terms`, `shipping_method`, `prepared_by`, `freight`, `apply_vat`, `vat_amt`, `subtotal`, `grand_total`, `notes`, `created_at` |
| | `purchase_order_items` | `id`, `po_id` (FK `purchase_orders` CASCADE), `item_code`, `description`, `uom`, `qty`, `unit_cost`, `total_amount` |
| **Sales & Targets** | `sales_details` | `id`, `doc_type`, `doc_status` (`'POSTED'`), `doc_date`, `doc_number`, `customer_name`, `tin_no`, `address`, `supplier`, `item_id` (FK `items`), `qty`, `uom`, `price`, `total_sales`, `cost`, `total_cost`, `patong`, `pos_charge`, `wt_2307`, `total_remit`, `profit`, `profit_margin`, `remarks`, `ref_pl` |
| | `earning_goals` | `id`, `title`, `period_type`, `target_period` (`YYYY-MM` or `YYYY`), `target_revenue`, `target_profit`, `created_at` |
| | `reminders` | `id`, `title`, `details`, `due_date`, `priority`, `category`, `status`, `assigned_to_user_id` (FK `users`), `created_at` |
| **Security, Config & Audit** | `roles` | `id`, `name` (UNIQUE), `description`, `created_at` |
| | `role_permissions` | `role_id` (FK `roles` CASCADE), `permission_key` — PK `(role_id, permission_key)` |
| | `users` | `id`, `username` (UNIQUE), `password_hash`, `full_name`, `role_id` (FK `roles`), `is_active`, `created_at` |
| | `user_sessions` | `id` (PK token), `user_id` (FK `users` CASCADE), `expires_at` |
| | `activity_logs` | `id`, `user_id`, `username`, `action`, `entity_type`, `entity_id`, `details`, `ip_address`, `created_at` |
| | `system_settings` | `key` (PK), `value` |

---

## Learnings & Enforced SQL Portability Rules

### Context: Universal SQLite + PostgreSQL Query Rules
- **`RETURNING id`**: Always use `INSERT INTO ... RETURNING id` with `QueryRow(...).Scan(&id)` instead of `result.LastInsertId()` (which fails on PostgreSQL `pgx`).
- **`ON CONFLICT DO NOTHING`**: Use ANSI `ON CONFLICT DO NOTHING` instead of SQLite-only `INSERT OR IGNORE`.
- **Timezone-Safe Month Extraction (`substr`)**: Use `substr(date_col, 1, 7)` instead of `strftime('%Y-%m', date_col)`. `db/db.go` defines immutable PostgreSQL `substr(timestamptz|timestamp|date, int, int)` polyfill functions so `substr(date_col, 1, 7)` works identically on both engines.
- **Batch Loading**: Use `FetchItemsStockBatch(db, itemIDs)` (`IN (?)`) rather than N+1 queries inside loops.
