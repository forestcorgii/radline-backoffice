# Database Migrations

> Back to [[00-index]] · Related: [[database-schema]]

## File
[db/db.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/db/db.go) — `InitDB()` function, lines 12–91

## Strategy

The project uses **inline startup migrations** instead of a migration framework. On every application start, `InitDB()` runs a series of "check-and-fix" queries before calling `createSchema()`.

### Pattern

```go
// 1. Try selecting the NEW column
_, err = DB.Exec("SELECT new_column FROM table LIMIT 0")
if err != nil {
    // Column doesn't exist yet
    // 2. Check if there's data to preserve
    var rowCount int
    errCount := DB.Get(&rowCount, "SELECT COUNT(*) FROM table")
    if errCount == nil {
        if rowCount == 0 {
            // No data → safe to DROP and recreate
            DB.Exec("DROP TABLE IF EXISTS table;")
        } else {
            // Has data → ALTER TABLE to add column
            DB.Exec("ALTER TABLE table ADD COLUMN new_column TYPE DEFAULT value;")
        }
    }
}
```

## Active Migrations (in order)

### 1. `receiving_logs` — Add `total_cost` and `selling_price`
- Checks if `total_cost` column exists
- If missing and table is empty → DROP and recreate
- If missing and table has data → `ALTER TABLE ... ADD COLUMN`

### 2. `sales_details` — Major schema evolution
- Checks if `doc_type` column exists
- If missing and table has data, applies a sequence of:
  - Add `doc_type TEXT NOT NULL DEFAULT 'SI'`
  - Add `doc_status TEXT NOT NULL DEFAULT 'POSTED'`
  - Rename `invoice_no` → `doc_number`
  - Add `customer_name TEXT`
  - Rename `unit_price` → `price`
  - Rename `margin_amount` → `profit`
  - Rename `date` → `doc_date`

### 3. `receiving_logs` — Rename `channel` → `supplier`
- Checks if `supplier` column exists
- If missing, checks if `channel` column exists → rename

### 4. `sales_details` — Rename `channel` → `supplier`
- Same pattern as above

### 5. `inventory_adjustments` — Add `adjustment_id`
- Checks if `adjustment_id` column exists
- If missing → `ALTER TABLE ... ADD COLUMN adjustment_id INTEGER`

## Ordering Rules

1. Migrations run **before** `createSchema()` — so they fix existing tables first
2. `createSchema()` uses `CREATE TABLE IF NOT EXISTS` — safe for new databases
3. Each migration is **idempotent** — runs safely even if already applied
4. Migrations use `LIMIT 0` queries as existence checks (cheaper than schema introspection)

## Adding New Migrations

When evolving the schema:
1. Add a new check-and-fix block in `InitDB()` **after** existing migrations
2. Update `createSchema()` to include the new column in the `CREATE TABLE` statement
3. The migration block handles existing databases; the schema handles new databases

## Related
- [[database-schema]] — Current table definitions
- [[tech-stack]] — Why no migration framework
- [[architecture]] — System initialization flow
