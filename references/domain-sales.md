# Domain — Sales

> Back to [[domain-overview]] · Related: [[domain-stock]], [[handlers-sales]]

## File
[sales.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/sales.go)

## SalesDetail Entity

```go
type SalesDetail struct {
    ID           int
    DocType      string        // e.g. "SI" (Sales Invoice), "DR" (Delivery Receipt)
    DocStatus    string        // e.g. "POSTED"
    DocDate      time.Time
    DocNumber    string        // Invoice/DR number
    CustomerName string
    Supplier     string        // Source supplier for stock allocation
    ItemID       int
    Qty          float64
    UOM          string
    Price        float64       // Selling price per unit
    TotalSales   float64       // Auto: Qty * Price
    Cost         float64       // Cost per unit
    TotalCost    float64       // Auto: Qty * Cost
    Profit       float64       // Auto: TotalSales - TotalCost
}
```

**Invariants (enforced by `NewSalesDetail`):**
- `DocType`, `DocNumber`, `Supplier`, `UOM` cannot be empty
- `ItemID > 0`, `Qty > 0`
- `Price >= 0`, `Cost >= 0`

**Auto-computed:** `TotalSales = Qty * Price`, `TotalCost = Qty * Cost`, `Profit = TotalSales - TotalCost`

---

## Sale Aggregate

Groups multiple items under a single sales document.

### Header
```go
type Sale struct {
    DocType      string
    DocStatus    string
    DocDate      time.Time
    DocNumber    string
    CustomerName string
    Supplier     string
    Items        []SaleItem
}
```

### Line Item
```go
type SaleItem struct {
    ItemID     int
    Qty        float64
    UOM        string
    Price      float64
    Cost       float64
    TotalSales float64    // Auto: Qty * Price
    TotalCost  float64    // Auto: Qty * Cost
    Profit     float64    // Auto: TotalSales - TotalCost
}
```

**Invariants (enforced by `NewSale`):**
Same as `NewSalesDetail`, plus:
- `Date` cannot be zero
- Must have at least 1 item

### `ToSalesDetails() []SalesDetail`
Flattens the aggregate into individual `SalesDetail` records ready for DB insertion. Maps header fields onto each line item.

## Business Context

- **DocType**: Distinguishes document type (e.g., `SI` = Sales Invoice, `DR` = Delivery Receipt)
- **DocStatus**: Currently always `"POSTED"` — set in the handler, not via form
- **Supplier**: Ties the sale to a specific supplier for stock tracking in [[domain-stock]]
- **Profit**: Always auto-calculated as `(Price - Cost) * Qty`

## Related
- [[domain-stock]] — Sales reduce stock on hand
- [[domain-item]] — Items referenced by ItemID
- [[handlers-sales]] — HTTP handlers for sales CRUD
- [[htmx-patterns]] — Multi-row form submission
- [[domain-testing]] — `TestNewSale_Validation`, `TestSale_ToSalesDetails`
