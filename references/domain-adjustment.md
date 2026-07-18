# Domain — Adjustment

> Back to [[domain-overview]] · Related: [[domain-stock]], [[handlers-inventory]]

## File
[adjustment.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/adjustment.go)

## InventoryAdjustment Entity

```go
type InventoryAdjustment struct {
    ID            int
    AdjustmentID  int           // FK to stock_adjustments header
    Date          time.Time
    ItemID        int
    UOM           string
    AdjustmentQty float64       // Positive = add stock, Negative = remove stock
    Cost          float64
    Remarks       string
}
```

**Invariants (enforced by `NewInventoryAdjustment`):**
- `ItemID > 0`
- `UOM` cannot be empty
- `AdjustmentQty ≠ 0` (can be positive or negative, but never zero)
- `Cost >= 0`
- `Remarks` cannot be empty (forces documentation)

**Key difference from Receiving/Sales:** Adjustment quantities **can be negative** (stock removal for damage, theft, etc.).

---

## StockAdjustment Aggregate

Groups multiple adjustment items under a shared header (date + remarks).

### Header
```go
type StockAdjustment struct {
    Date    time.Time
    Remarks string
    Items   []StockAdjustmentItem
}
```

### Line Item
```go
type StockAdjustmentItem struct {
    ItemID int
    Qty    float64
    UOM    string
    Cost   float64
}
```

**Invariants (enforced by `NewStockAdjustment`):**
- `Date` cannot be zero
- `Remarks` cannot be empty
- Must have at least 1 item
- Each item: `ItemID > 0`, `Qty ≠ 0`, `UOM` not empty, `Cost >= 0`

### `ToInventoryAdjustments() []InventoryAdjustment`
Flattens the aggregate into individual `InventoryAdjustment` records. Sets `AdjustmentID` to 0 — the **handler** assigns the real DB-generated ID after inserting the header row.

## Database Structure

This uses a **header/detail pattern**:
1. `stock_adjustments` — header table (`id`, `date`, `remarks`)
2. `inventory_adjustments` — detail table (has `adjustment_id` FK to header)

See [[database-schema]] for the full DDL.

## Flow

```
Form POST → Handler parses multi-value fields
          → Constructs []StockAdjustmentItem
          → domain.NewStockAdjustment(date, remarks, items)
          → INSERT header into stock_adjustments, get lastInsertId
          → stockAdj.ToInventoryAdjustments()
          → INSERT each adjustment with real adjustmentID
          → All within a DB transaction
```

## Learnings

### Context: Multi-Item Stock Adjustment with Header Table
**Problem**: The stock adjustment form only allowed a single item per submission, requiring users to submit repeatedly. Grouping adjustments by a shared reason/date required a mechanism to batch items together.
**Enforced Solution**:
- Created a `stock_adjustments` header table (`id INTEGER PRIMARY KEY AUTOINCREMENT`, `date`, `remarks`) and added `adjustment_id INTEGER` FK to `inventory_adjustments`.
- Added `StockAdjustment` aggregate in `domain/adjustment.go` with `StockAdjustmentItem` value objects, full validation, and `ToInventoryAdjustments()` mapper.
- Created `adjustment_item_row.html` fragment with HTMX-driven item selection and UOM auto-fill.
- Updated `inventory.html` to render a multi-item table with "Add Item Row" / "Remove" and "Commit Adjustment" button.
- Added routes `/inventory/adjustments/new-row` and `/inventory/adjustments/item-row-details`.
- Implemented `NewAdjustmentRowHandler` and `AdjustmentItemRowDetailsHandler`.
- Updated `AdjustStockHandler` to parse array form fields, validate via domain aggregate, insert header + items within a single DB transaction.
- Added safe migration for existing `inventory_adjustments` tables missing the `adjustment_id` column.
- Added unit tests for `StockAdjustment` in `domain/adjustment_test.go`.

## Related
- [[domain-stock]] — Adjustments modify global on-hand via `CalculateGlobalOnHand`
- [[handlers-inventory]] — `AdjustStockHandler`
- [[database-schema]] — `stock_adjustments` and `inventory_adjustments` tables
- [[database-migrations]] — Migration for adding `adjustment_id` column
- [[domain-testing]] — `TestNewStockAdjustment_Validation`, `TestStockAdjustment_ToInventoryAdjustments`
