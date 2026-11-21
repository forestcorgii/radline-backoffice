# Testing Strategy (Domain Unit Tests, DB/Handler Integration & E2E)

> Back to [[domain-overview]] · Related: [[architecture]]

## Philosophy

Radline BackOffice follows a **domain-first TDD strategy** complemented by multi-layer integration and end-to-end verification:
1. **Pure Domain Unit Tests (`domain/*_test.go`)** — Fast, isolated, table-driven black-box tests (`package domain_test`) covering all business invariants, financial formulas, and aggregate mappings without database or HTTP dependencies.
2. **Database & Handler Integration Tests (`db/db_test.go`, `handlers/*_test.go`)** — Verify schema initialization, dual-engine migrations, RBAC middleware, CSRF protection, and HTTP handler flows against an isolated SQLite test database.
3. **Opaque-Box E2E Tests (`e2e/*_test.go`)** — Verify end-to-end HTTP workflows, session authentication, and rendered HTML responses.

## Domain Test Files (`domain/`)

| File | Covers |
|---|---|
| [`stock_test.go`](../domain/stock_test.go) | `ItemStock.CalculateOnHand`, `CalculateGlobalOnHand`, `GetOldestPLWithStock` (FIFO), multi-UOM conversions, and status filtering |
| [`uom_setting_test.go`](../domain/uom_setting_test.go) | `UomSetting` validation and conversion factor constraints |
| [`receiving_test.go`](../domain/receiving_test.go) | `ReceivingLog` & `StockReceive` aggregate validation, cascading discount formula (`Less1`, `Less2`), `Markup` (default `130`), and `ToReceivingLogs` mapping |
| [`sales_test.go`](../domain/sales_test.go) | `SalesDetail` & `Sale` aggregate validation, canonical `DocStatus` (`"Active"`, `"Posted"`, `"Cancelled/Return"`), `SetCustomerDetails`, net `Profit` formula (`TotalSales - TotalCost - POSCharge - Patong - WT2307`), deduction bounds, and `ToSalesDetails` mapping |
| [`adjustment_test.go`](../domain/adjustment_test.go) | `InventoryAdjustment` & `StockAdjustment` aggregate validation (positive/negative quantities, required remarks) and `ToInventoryAdjustments` mapping |
| [`auth_test.go`](../domain/auth_test.go) | `User`, `Role.HasPermission`, `ValidateUsername`, `ValidateRole`, `HashPassword`, `CheckPasswordHash` |
| [`customer_test.go`](../domain/customer_test.go) | `Customer` validation (`NewCustomer`), default payment terms, whitespace trimming |
| [`goal_test.go`](../domain/goal_test.go) | `EarningGoal` validation (`ValidateEarningGoal`) and `CalculateProgress` status thresholds (`Exceeded`, `On Track`, `In Progress`, `Behind`) |
| [`purchase_order_test.go`](../domain/purchase_order_test.go) | `PurchaseOrder` & `PurchaseOrderItem` validation and `CalculateTotals` (subtotal, freight, 12% VAT, grand total) |
| [`reminder_test.go`](../domain/reminder_test.go) | `Reminder` validation (`ValidateReminder`), `IsOverdue`, `AssignTo`, `Unassign`, `IsAssigned` |
| [`brand_test.go`](../domain/brand_test.go) | `Brand` validation (`NewBrand`) |
| [`category_test.go`](../domain/category_test.go) | `Category` validation (`NewCategory`) |
| [`item_test.go`](../domain/item_test.go) | `Item` validation (`NewItem`) |

## Integration & E2E Test Files

| Layer | Files | Purpose |
|---|---|---|
| **Database (`db/`)** | `db/db_test.go` | Verifies SQLite schema creation, idempotent migrations, default seeding, and `DBConn`/`TxConn` query execution |
| **Handlers (`handlers/`)** | `handlers/*_test.go` | Tests HTTP handlers, RBAC permission guards, auth flows, and HTMX fragment vs full-page rendering |
| **End-to-End (`e2e/`)** | `e2e/*_test.go` | Opaque-box HTTP server tests exercising full user workflows |

## Test Pattern

All domain validation tests use **table-driven tests** with `t.Run()` subtests:

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

### Entity & Aggregate Validation
Every `New*()` / `Validate*()` constructor has tests for:
- ✅ Valid creation (happy path, single and multi-item)
- ✅ Required string fields being empty or whitespace-only
- ✅ Numeric constraints (zero, negative, `NaN`, `Inf`, discount and deduction bounds)
- ✅ Computed field verification (`Cost`, `TotalCost`, `SellingPrice`, `TotalSales`, `TotalRemit`, `Profit`, `ProfitMargin`, `VATAmt`, `GrandTotal`) with 2-decimal `float64` rounding

### Stock Calculation & FIFO
`TestItemStock_CalculateOnHand` and FIFO tests verify:
- Multi-UOM conversion (`PCS` + `BOX`)
- Multi-supplier filtering and global on-hand with adjustments
- Exclusion of `"Cancelled/Return"` sales from stock deductions
- `GetOldestPLWithStock()` chronological FIFO batch consumption

## Running Tests

```bash
go test ./domain/... -v
go test ./...
```

## Learnings

### Context: DDD Domain Isolation and Testing Strategy
**Problem**: Embedding database querying logic or framework-specific DTOs directly within business calculations violates domain model sovereignty and makes unit testing slow or dependent on active database connections.
**Enforced Solution**:
- Implement pure domain models under the `domain` package, completely free of `db` tag structures, SQL drivers, or UI templates.
- Define complex business concepts as aggregates (`domain.ItemStock`, `domain.StockReceive`, `domain.Sale`, `domain.StockAdjustment`, `domain.PurchaseOrder`), value objects, or entities that encapsulate their invariants and compute metrics purely in memory.
- Write fast table-driven unit tests (`domain/*_test.go`) first, and complement them with database/handler integration tests (`db/db_test.go`, `handlers/*_test.go`) and E2E tests (`e2e/*_test.go`).

## Related
- [[domain-overview]] — Domain layer philosophy
- [[domain-stock]] — Stock calculation under test
- [[domain-receiving]], [[domain-sales]], [[domain-adjustment]] — Entities under test
