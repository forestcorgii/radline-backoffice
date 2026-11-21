# Domain — Sales

> Back to [[domain-overview]] · Related: [[domain-stock]], [[handlers-sales]]

## File
[`domain/sales.go`](../domain/sales.go)

## SalesDetail Entity

```go
type SalesDetail struct {
    ID           int
    DocType      string        // e.g. "SI" (Sales Invoice), "DR" (Delivery Receipt)
    DocStatus    string        // Canonical: "Active", "Posted", "Cancelled/Return"
    DocDate      time.Time
    DocNumber    string        // Invoice/DR number
    CustomerName string
    TINNo        string        // Customer Tax Identification Number
    Address      string        // Customer billing/delivery address
    Supplier     string        // Source supplier for stock allocation
    ItemID       int
    Qty          float64
    UOM          string
    Price        float64       // Unit selling price
    TotalSales   float64       // Auto: round2(Qty * Price)
    Cost         float64       // Unit cost
    TotalCost    float64       // Auto: round2(Qty * Cost)
    Patong       float64       // Additional markup/fee deduction (>= 0)
    POSCharge    float64       // POS charge deduction (>= 0)
    WT2307       float64       // Withholding tax 2307 deduction (>= 0)
    TotalRemit   float64       // Auto: round2(TotalSales - (Patong + POSCharge + WT2307))
    Profit       float64       // Auto: round2(TotalSales - TotalCost - POSCharge - Patong - WT2307)
    ProfitMargin float64       // Auto: round2((Profit / TotalSales) * 100)
    Remarks      string
    RefPL        string        // Reference Packing List
}
```

**Invariants (enforced by `NewSalesDetail` & `NewSale`):**
- `DocType`, `DocNumber`, `Supplier`, `UOM` cannot be empty
- `DocStatus` must be one of the canonical values: `"Active"`, `"Posted"`, or `"Cancelled/Return"` (never bare `"CANCELLED"`)
- `ItemID > 0`, `Qty > 0`
- `Price >= 0`, `Cost >= 0`
- Deduction bounds: `Patong >= 0`, `POSCharge >= 0`, `WT2307 >= 0`, and `(Patong + POSCharge + WT2307) <= TotalSales`
- All numeric inputs reject `NaN`/`Inf` and computed financial fields use 2-decimal `float64` rounding (`round2`).

**Auto-computed Financial Formulas:**
- `TotalSales = round2(Qty * Price)`
- `TotalCost = round2(Qty * Cost)`
- `TotalRemit = round2(TotalSales - (Patong + POSCharge + WT2307))`
- `Profit = round2(TotalSales - TotalCost - POSCharge - Patong - WT2307)`
- `ProfitMargin = round2((Profit / TotalSales) * 100)` if `TotalSales > 0` else `0`

---

## Sale Aggregate

Groups multiple items under a single sales document.

### Header
```go
type Sale struct {
    DocType      string
    DocStatus    string        // Canonical: "Active", "Posted", "Cancelled/Return"
    DocDate      time.Time
    DocNumber    string
    CustomerName string
    TINNo        string
    Address      string
    Supplier     string
    Items        []SaleItem
}
```

### `Sale.SetCustomerDetails(tinNo, address string)`
Attaches optional customer `TINNo` and `Address` metadata onto the `Sale` aggregate header so `ToSalesDetails()` propagates them to every flat `SalesDetail` record.

### Line Item
```go
type SaleItem struct {
    ItemID       int
    Qty          float64
    UOM          string
    Price        float64
    Cost         float64
    TotalSales   float64
    TotalCost    float64
    Patong       float64
    POSCharge    float64
    WT2307       float64
    TotalRemit   float64
    Profit       float64
    ProfitMargin float64
    Remarks      string
    RefPL        string
}
```

**Invariants (enforced by `NewSale`):**
Same as `NewSalesDetail`, plus:
- `DocDate` cannot be zero
- Must have at least 1 item

### `ToSalesDetails() []SalesDetail`
Flattens the aggregate into individual `SalesDetail` records ready for DB insertion. Maps header fields (`DocType`, `DocStatus`, `DocDate`, `DocNumber`, `CustomerName`, `TINNo`, `Address`, `Supplier`) onto each line item.

## Business Context

- **DocType**: Distinguishes document type (e.g., `SI` = Sales Invoice, `DR` = Delivery Receipt)
- **DocStatus**: Canonical values are `"Active"`, `"Posted"`, and `"Cancelled/Return"` (never `"CANCELLED"`). Records with status `"Cancelled/Return"` are excluded from stock deductions and active revenue KPIs.
- **Supplier**: Ties the sale to a specific supplier for stock tracking in [[domain-stock]]
- **Total Remit & Net Profit**: `TotalRemit` and `Profit` both deduct `Patong`, `POSCharge`, and `WT2307`, ensuring net profitability reflects all transaction deductions.

## Related
- [[domain-stock]] — Sales reduce stock on hand
- [[domain-item]] — Items referenced by ItemID
- [[handlers-sales]] — HTTP handlers for sales CRUD
- [[htmx-patterns]] — Multi-row form submission
- [[domain-testing]] — `TestNewSale_Validation`, `TestSale_ToSalesDetails`
