# Domain — ItemStock Aggregate (Stock Calculation)

> Back to [[domain-overview]] · Related: [[domain-receiving]], [[domain-sales]], [[domain-adjustment]]

## File
[stock.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/stock.go)

## ItemStock Aggregate Root

```go
type ItemStock struct {
    Item          Item
    UomSettings   []UomSetting
    ReceivingLogs []ReceivingLog
    SalesDetails  []SalesDetail
    Adjustments   []InventoryAdjustment
}
```

This is the **core aggregate root** for inventory calculations. It aggregates all transaction data for a single [[domain-item|Item]] and provides pure computation methods.

## Methods

### `CalculateOnHand(supplier string) float64`
Calculates stock on hand **for a specific supplier**, excluding adjustments.

**Formula:**
```
OnHand = Σ(Received * UOM_Factor) - Σ(Sold * UOM_Factor)
```

- Only includes receiving/sales records matching the given `supplier`
- Converts all quantities to the item's `DefaultUOM` via [[domain-item|UomSetting]] conversion factors

### `CalculateGlobalOnHand() float64`
Calculates stock on hand **across all suppliers**, including adjustments.

**Formula:**
```
GlobalOnHand = Σ(All Received * Factor) - Σ(All Sold * Factor) + Σ(All Adjusted * Factor)
```

- No supplier filter — sums everything
- Includes `InventoryAdjustment` records (positive or negative)

### `getConversionFactor(uom string) float64` (private)
Returns the conversion factor for a given UOM:
1. If `uom == Item.DefaultUOM` → returns `1.0`
2. Searches `UomSettings` for a matching `MUOM` → returns its `ConversionFactor`
3. Fallback → returns `1.0` (treats unknown UOM as 1:1)

## UOM Conversion Example

```
Item DefaultUOM: PCS
UomSettings: [{ MUOM: "BOX", ConversionFactor: 10 }]

Received: 2 BOX → 2 * 10 = 20 PCS
Sold:     3 PCS → 3 * 1  = 3 PCS
OnHand = 20 - 3 = 17 PCS
```

## Integration Point

The [[models-layer|`models/stock.go`]] file contains `CalculateStockOnHand()`, which:
1. Loads the item, UOM settings, receiving logs, sales, and adjustments from the DB
2. Maps each DB DTO to the corresponding domain type
3. Constructs `domain.NewItemStock(...)` 
4. Calls `stock.CalculateOnHand(supplierName)`

This keeps the domain calculation pure and the DB loading in the infrastructure layer.

## Important Notes

- **`CalculateOnHand` does NOT include adjustments** — it's supplier-scoped and adjustments are not supplier-specific
- **`CalculateGlobalOnHand` DOES include adjustments** — used for the inventory overview page
- The SQL-based stock view in [[handlers-inventory|`stockListQuery`]] uses raw SQL aggregation for the overview table, which is a simpler (but less domain-pure) approach

## Learnings

### Context: FIFO Cost & Selling Price Tracking (Oldest PL with Stock)
**Problem**: Tracking and auto-populating inventory item prices and costs based on the oldest transaction batch (PL/Receiving Log) that still has available stock.
**Enforced Solution**:
- **Domain FIFO Computation**: Implement `GetOldestPLWithStock()` on the `ItemStock` domain aggregate:
  1. Sort all receiving logs chronologically (`Date ASC, ID ASC`).
  2. Compute net consumed stock `consumed = totalReceived - onHand`.
  3. Loop through sorted logs, deducting each log's quantity from `consumed`.
  4. The first log whose received quantity is greater than the remaining `consumed` value has active stock under FIFO. Return its cost and price.
- **Auto-population in Form Handlers**: In HTMX handlers that fetch pre-filled form detail rows (e.g. `/sales/item-row-details`), fetch the item stock aggregate, run the FIFO calculation, and fall back to the latest receiving log if no stock is currently on hand.

## Related
- [[domain-item]] — `Item` and `UomSetting` definitions
- [[domain-receiving]] — `ReceivingLog` entity
- [[domain-sales]] — `SalesDetail` entity
- [[domain-adjustment]] — `InventoryAdjustment` entity
- [[models-layer]] — `CalculateStockOnHand` integration
- [[domain-testing]] — Test cases including UOM conversion
