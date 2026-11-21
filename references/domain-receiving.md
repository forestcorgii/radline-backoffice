# Domain — Receiving

> Back to [[domain-overview]] · Related: [[domain-stock]], [[handlers-inventory]]

## File
[`domain/receiving.go`](../domain/receiving.go)

## ReceivingLog Entity

```go
type ReceivingLog struct {
    ID           int
    Supplier     string
    Date         time.Time
    PLNo         string        // Packing List number
    ItemID       int
    Qty          float64
    UOM          string
    UnitPrice    float64       // Supplier list price per unit before discounts
    Less1        float64       // First discount percentage (0..100)
    Less2        float64       // Second discount percentage (0..100)
    Cost         float64       // Net unit cost after discounts
    TotalCost    float64       // Auto-calculated: round2(Qty * Cost)
    Markup       float64       // Markup percentage (defaults to 130 when <= 0)
    SellingPrice float64       // Auto-calculated: round2(Cost * Markup / 100)
    Remarks      string
}
```

**Invariants (enforced by `NewReceivingLog`):**
- `Supplier` cannot be empty
- `ItemID` must be > 0
- `Qty` must be > 0 (and finite)
- `UOM` cannot be empty
- `UnitPrice` cannot be negative (`>= 0`)
- `Less1`, `Less2` must be within `0 <= Less1, Less2 <= 100`
- `Cost` cannot be negative (`>= 0`)
- `Markup` defaults to `130` when `<= 0`
- All numeric inputs reject `NaN` / `Inf` and computed currency values use 2-decimal `float64` rounding (`round2`).

**Pricing & Cost Formula:**
- `Cost = round2(UnitPrice * (1 - Less1/100) * (1 - Less2/100))`
- `TotalCost = round2(Qty * Cost)`
- `SellingPrice = round2(Cost * Markup / 100)` (default `Markup = 130`)

---

## StockReceive Aggregate

The **multi-item stock receive** aggregate groups multiple items under a single receiving transaction.

### Header
```go
type StockReceive struct {
    PLNo     string
    Supplier string
    Date     time.Time
    Items    []StockReceiveItem
}
```

### Line Item
```go
type StockReceiveItem struct {
    ItemID       int
    Qty          float64
    UOM          string
    UnitPrice    float64       // Supplier list price before discounts
    Less1        float64       // First discount percentage (0..100)
    Less2        float64       // Second discount percentage (0..100)
    Cost         float64       // Auto: round2(UnitPrice * (1 - Less1/100) * (1 - Less2/100))
    TotalCost    float64       // Auto: round2(Qty * Cost)
    Markup       float64       // Markup percentage (default 130)
    SellingPrice float64       // Auto: round2(Cost * Markup / 100)
    Remarks      string
}
```

**Invariants (enforced by `NewStockReceive`):**
- `Supplier` cannot be empty
- `Date` cannot be zero
- Must have at least 1 item
- Each item: `ItemID > 0`, `Qty > 0`, `UOM` not empty, `UnitPrice >= 0`, `0 <= Less1, Less2 <= 100`, `Cost >= 0`, `Markup` defaults to `130` if `<= 0`, no `NaN`/`Inf`, 2-decimal `round2` currency rounding.

### `ToReceivingLogs() []ReceivingLog`
Flattens the aggregate into individual `ReceivingLog` records ready for DB insertion. Maps header fields (`Supplier`, `Date`, `PLNo`) and all pricing/discount fields (`UnitPrice`, `Less1`, `Less2`, `Cost`, `TotalCost`, `Markup`, `SellingPrice`, `Remarks`) onto each line item.

## Flow

```
Form POST → Handler parses multi-value fields
          → Constructs []StockReceiveItem
          → domain.NewStockReceive(plNo, supplier, date, items)
          → stockReceive.ToReceivingLogs()
          → INSERT each log in a DB transaction
```

See [[handlers-inventory]] for the `ReceiveStockHandler` implementation.

## Learnings

### Context: Multi-Item Stock Receiving UI
**Problem**: Previously the receiving form only allowed a single item per stocktake, requiring users to submit repeatedly. Need a dynamic table to add multiple items in one transaction.
**Enforced Solution**:
- Created `receiving_item_row.html` fragment with HTMX-driven item selection and defaults.
- Updated `inventory.html` to render a multi-item table with “Add Item Row” and removal.
- Added routes `/inventory/receiving/new-row` and `/inventory/receiving/item-row-details`.
- Implemented `NewReceivingRowHandler` and `ReceivingItemRowDetailsHandler`.
- Added `StockReceive` aggregate in `domain/receiving.go` with validation and conversion to `ReceivingLog`.
- Updated `ReceiveStockHandler` to parse array form fields, validate via domain, and insert within a transaction.
- Updated template parsing to include `receiving_item_row.html`.
- Added unit tests for `StockReceive`.

## Related
- [[domain-stock]] — ReceivingLogs feed into stock calculations
- [[domain-item]] — Items referenced by ItemID
- [[handlers-inventory]] — HTTP handler for receiving
- [[htmx-patterns]] — Multi-row form submission pattern
- [[domain-testing]] — `TestNewStockReceive_Validation`, `TestStockReceive_ToReceivingLogs`
