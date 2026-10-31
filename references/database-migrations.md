# Database Migrations

> Back to [[00-index]] · Related: [[database-schema]], [[architecture]]

## File
[db/db.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/db/db.go) — `InitDB()`, `runSQLiteMigrations()`, `runPostgresMigrations()`, `EnsureRemindersTable()`, `EnsureCustomersTable()`

## Startup Migration Strategy
Radline uses **idempotent inline startup migrations** (no external CLI tool). On startup:
- **SQLite path**: `runSQLiteMigrations()` → `createSchemaSQLite()`
- **PostgreSQL path**: `createSchemaPostgres()` → `runPostgresMigrations()`
- **Both paths**: `EnsureRemindersTable()` → `EnsureCustomersTable()` → `seedDefaults()`

### Adding a New Column or Table
1. Add the column/table to **both** `createSchemaSQLite()` and `createSchemaPostgres()`.
2. Add a check-and-alter block in `runSQLiteMigrations()` (`SELECT new_col FROM table LIMIT 0` → `ALTER TABLE ... ADD COLUMN`).
3. Add `ALTER TABLE <table> ADD COLUMN IF NOT EXISTS ...` to the `migrations` slice in `runPostgresMigrations()`.

---

## Learnings & Enforced Migration Invariants

### Context: PostgreSQL Production Table Migrations & `COALESCE` Scanning
- **Problem**: `CREATE TABLE IF NOT EXISTS` in `createSchemaPostgres()` does not add new columns to existing production tables, causing `column does not exist` HTTP 500s. Additionally, `NULL` values in legacy rows crash `sqlx.Select` when scanning into non-pointer Go struct fields.
- **Enforced Solution**:
  - Always maintain 1:1 parity between `runSQLiteMigrations()` and `runPostgresMigrations()`.
  - Wrap nullable columns in `COALESCE(col, '')` or `COALESCE(col, 0.0)` in analytical/list `SELECT` queries, or use pointer fields (`*int`, `*int64`) in `models/` DTOs.

### Context: Strict PostgreSQL `GROUP BY` Multi-Column Projection
- **Problem**: SQLite permits `SELECT i.code, i.description ... GROUP BY t.month, t.item_id`, whereas PostgreSQL rejects any non-aggregated column missing from `GROUP BY` (`SQLSTATE 42803`).
- **Enforced Solution**: Every non-aggregated column in a `SELECT` projection must be explicitly listed in the `GROUP BY` clause (e.g., `GROUP BY t.month, t.item_id, i.code, i.description, i.default_uom`).
