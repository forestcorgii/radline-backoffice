# Domain — Testing Strategy

> Back to [[domain-overview]] · Related: [[architecture]]

## Philosophy

Tests are written **exclusively for the domain layer** — no HTTP handler tests, no DB integration tests. This follows the DDD principle of keeping domain logic pure and independently verifiable.

All test files live in the `domain` package using `package domain_test` (black-box testing).

## Test Files

| File | Covers | Key Tests |
|---|---|---|
| [stock_test.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/stock_test.go) | Brand, Category, Item, UomSetting, ReceivingLog, SalesDetail, InventoryAdjustment validation + `ItemStock.CalculateOnHand` + `ItemStock.CalculateGlobalOnHand` | 7 test functions |
| [receiving_test.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/receiving_test.go) | `StockReceive` aggregate validation + `ToReceivingLogs` mapping | 2 test functions (11 subtests + 1 mapping) |
| [sales_test.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/sales_test.go) | `Sale` aggregate validation + `ToSalesDetails` mapping | 2 test functions (13 subtests + 1 mapping) |
| [adjustment_test.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/adjustment_test.go) | `StockAdjustment` aggregate validation + `ToInventoryAdjustments` mapping | 2 test functions (11 subtests + 1 mapping) |

## Test Pattern

All validation tests use **table-driven tests** with `t.Run()` subtests:

```go
tests := []struct {
    name    string
    // ... test inputs
    wantErr bool
    errMsg  string // exact error message match
}{...}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        _, err := domain.NewXxx(...)
        if (err != nil) != tt.wantErr {
            t.Errorf(...)
        }
        if tt.wantErr && err.Error() != tt.errMsg {
            t.Errorf(...)
        }
    })
}
```

## Coverage Highlights

### Entity Validation
Every `New*()` constructor has tests for:
- ✅ Valid creation (happy path)
- ✅ Each required field being empty
- ✅ Numeric constraints (zero, negative)
- ✅ Computed field verification (`TotalCost`, `Profit`, etc.)

### Aggregate Validation
Each aggregate (`StockReceive`, `Sale`, `StockAdjustment`) tests:
- ✅ Valid creation with 1 item
- ✅ Valid creation with multiple items
- ✅ Empty supplier/date/remarks
- ✅ Empty items list
- ✅ Invalid item ID, zero/negative qty, empty UOM, negative cost/price

### Aggregate Mapping
`To*()` methods are tested for:
- ✅ Correct number of output records
- ✅ Header fields mapped to each record
- ✅ Computed fields (TotalCost, Profit) calculated correctly

### Stock Calculation
`TestItemStock_CalculateOnHand` is a comprehensive integration-style domain test:
- Tests multi-UOM conversion (PCS + BOX)
- Tests multi-supplier filtering
- Tests global on-hand including adjustments
- Verifies exact numeric expectations

## Running Tests

```bash
cd radline
go test ./domain/ -v
```

## Learnings

### Context: DDD Domain Isolation and Testing Strategy
**Problem**: Embedding database querying logic or framework-specific DTOs directly within business calculations violates domain model sovereignty and makes unit testing complex or impossible without active database/mocking structures.
**Enforced Solution**:
- Implement pure domain models under a `domain` package, completely free of `db` tag structures, SQL engines, or UI templates.
- Define complex business concepts as aggregates (such as `domain.ItemStock`), value objects, or entities that encapsulate their invariants and compute metrics purely in memory.
- Within persistence and integration wrappers (e.g. `models/stock.go`), load the necessary data from database tables, map the data to the pure domain objects, run the calculations, and return the outputs.
- Write unit tests (`*_test.go`) exclusively within the `domain` package, keeping them fast, self-contained, and isolated from external dependencies.

## Related
- [[domain-overview]] — Domain layer philosophy
- [[domain-stock]] — Stock calculation under test
- [[domain-receiving]], [[domain-sales]], [[domain-adjustment]] — Entities under test
