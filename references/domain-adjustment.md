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

## Related
- [[domain-stock]] — Adjustments modify global on-hand via `CalculateGlobalOnHand`
- [[handlers-inventory]] — `AdjustStockHandler`
- [[database-schema]] — `stock_adjustments` and `inventory_adjustments` tables
- [[database-migrations]] — Migration for adding `adjustment_id` column
- [[domain-testing]] — `TestNewStockAdjustment_Validation`, `TestStockAdjustment_ToInventoryAdjustments`
