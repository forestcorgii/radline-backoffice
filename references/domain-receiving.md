# Domain — Receiving

> Back to [[domain-overview]] · Related: [[domain-stock]], [[handlers-inventory]]

## File
[receiving.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/receiving.go)

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
    UnitPrice    float64       // Selling price per unit
    Cost         float64       // Cost per unit
    TotalCost    float64       // Auto-calculated: Qty * Cost
    SellingPrice float64       // Set to UnitPrice on creation
}
```

**Invariants (enforced by `NewReceivingLog`):**
- `Supplier` cannot be empty
- `ItemID` must be > 0
- `Qty` must be > 0
- `UOM` cannot be empty
- `Cost` cannot be negative
- `UnitPrice` cannot be negative

**Auto-computed:** `TotalCost = Qty * Cost`, `SellingPrice = UnitPrice`

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
    UnitPrice    float64
    Cost         float64
    TotalCost    float64       // Auto: Qty * Cost
    SellingPrice float64       // Auto: UnitPrice
}
```

**Invariants (enforced by `NewStockReceive`):**
- `Supplier` cannot be empty
- `Date` cannot be zero
- Must have at least 1 item
- Each item: `ItemID > 0`, `Qty > 0`, `UOM` not empty, `Cost >= 0`, `UnitPrice >= 0`

### `ToReceivingLogs() []ReceivingLog`
Flattens the aggregate into individual `ReceivingLog` records ready for DB insertion. Maps header fields (`Supplier`, `Date`, `PLNo`) onto each line item.

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
