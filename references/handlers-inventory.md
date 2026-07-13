# Handlers — Inventory

> Back to [[handlers-overview]] · Related: [[routing]], [[domain-receiving]], [[domain-adjustment]]

## File
[handlers/inventory.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/inventory.go)

## Stock Overview

### `InventoryHandler` (GET `/inventory`, `/inventory/stock`)
- Uses `stockListQuery()` — a SQL query that computes per-item stock via subquery JOINs:
  ```sql
  (COALESCE(received, 0) - COALESCE(sold, 0) + COALESCE(adjusted, 0)) AS on_hand
  ```
- Supports `search` and `stock_filter` (in_stock/out_of_stock/low_stock) query params
- Low stock threshold: `on_hand > 0 AND on_hand <= 10`
- HTMX-aware: rows fragment or full page

**Note:** This uses raw SQL aggregation for performance, not the [[domain-stock|domain `ItemStock` aggregate]]. The domain aggregate is used when precise per-supplier calculation is needed (e.g., in [[models-layer|`CalculateStockOnHand`]]).

---

## Monthly Inventory

### `MonthlyInventoryHandler` (GET `/inventory/monthly`)
- Complex UNION ALL query grouping receiving, sales, and adjustments by month + item
- Provides per-month breakdown: qty received, sold, adjusted, net change
- Filter by month dropdown populated from distinct months across all tables

---

## Stock Receiving

### `StockReceivingPageHandler` (GET `/inventory/receiving`)
Renders the receiving form + recent logs. Loads:
- All items (for the item select dropdown)
- Recent receiving logs with item codes

### `ReceiveStockHandler` (POST `/inventory/receiving/add`)
Multi-item stock receive flow:
1. Parse form arrays: `item_id[]`, `qty[]`, `uom[]`, `cost[]`, `unit_price[]`
2. Validate lengths match
3. Construct `[]domain.StockReceiveItem`
4. Call `domain.NewStockReceive(plNo, supplier, date, items)` for validation
5. `stockReceive.ToReceivingLogs()` to flatten
6. INSERT all logs in a `sqlx` transaction
7. On success: toast + `HX-Location: /inventory/receiving/logs` redirect

### `NewReceivingRowHandler` (GET `/inventory/receiving/new-row`)
Returns an empty `receiving_item_row.html` fragment for dynamically adding item rows.

### `ReceivingItemRowDetailsHandler` (GET `/inventory/receiving/item-row-details`)
When an item is selected in a receiving row:
- Fetches the item's `default_uom`
- Fetches last receiving `cost` and `selling_price`
- Returns a pre-filled `receiving_item_row.html` fragment

---

## Stock Adjustments

### `StockAdjustmentsPageHandler` (GET `/inventory/adjustments`)
Same pattern as receiving: form + recent logs.

### `AdjustStockHandler` (POST `/inventory/adjustments/add`)
Multi-item adjustment flow:
1. Parse form arrays: `item_id[]`, `adjustment_qty[]`, `uom[]`, `cost[]`
2. Construct `domain.NewStockAdjustment(date, remarks, items)`
3. INSERT header into `stock_adjustments`, get `lastInsertId`
4. `stockAdj.ToInventoryAdjustments()` to flatten
5. INSERT each with the real `adjustment_id`
6. All within a single transaction
7. On success: toast + `HX-Location: /inventory/adjustments/logs`

### `NewAdjustmentRowHandler` / `AdjustmentItemRowDetailsHandler`
Same pattern as receiving row handlers, but for adjustment fields.

---

## Log Views

### `ReceivingLogsHandler` (GET `/inventory/receiving/logs`)
- Searchable by PL number or item code
- Filterable by supplier
- Sortable by date or quantity

### `AdjustmentLogsHandler` (GET `/inventory/adjustments/logs`)
- Searchable by remarks, item code, or adjustment ID
- Sortable by date or quantity

## Related
- [[domain-receiving]] — `StockReceive` aggregate used by `ReceiveStockHandler`
- [[domain-adjustment]] — `StockAdjustment` aggregate used by `AdjustStockHandler`
- [[domain-stock]] — How these transactions affect stock on hand
- [[htmx-patterns]] — Multi-row form submission, HX-Location redirect
- [[routing]] — Route definitions
