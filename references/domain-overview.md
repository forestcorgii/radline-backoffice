# Domain Overview

> Back to [[00-index]] · Related: [[architecture]]

## Philosophy

The `domain/` package implements **Domain-Driven Design (DDD)** principles:
- **Pure Go** — No database tags, no HTTP, no template imports.
- **Validation at construction** — All `New*()` constructors enforce invariants. Invalid data never enters the domain.
- **Aggregates** — Complex multi-entity transactions are modeled as aggregates (`StockReceive`, `Sale`, `StockAdjustment`) that validate the whole before allowing persistence.

## Entity & Aggregate Map

```
domain/
├── brand.go         → Brand entity           → [[domain-item]]
├── category.go      → Category entity         → [[domain-item]]
├── item.go          → Item entity (core)      → [[domain-item]]
├── uom_setting.go   → UomSetting value object → [[domain-item]]
├── receiving.go     → ReceivingLog entity     → [[domain-receiving]]
│                      StockReceive aggregate  → [[domain-receiving]]
├── sales.go         → SalesDetail entity      → [[domain-sales]]
│                      Sale aggregate          → [[domain-sales]]
├── adjustment.go    → InventoryAdjustment     → [[domain-adjustment]]
│                      StockAdjustment agg.    → [[domain-adjustment]]
├── stock.go         → ItemStock aggregate     → [[domain-stock]]
│
├── stock_test.go       → [[domain-testing]]
├── receiving_test.go   → [[domain-testing]]
├── sales_test.go       → [[domain-testing]]
└── adjustment_test.go  → [[domain-testing]]
```

## DDD Glossary (Project-Specific)

| Concept | Domain Term | Example |
|---|---|---|
| Entity | `Brand`, `Category`, `Item` | Has ID, mutable name/code |
| Value Object | `UomSetting` | Immutable conversion factor |
| Aggregate Root | `ItemStock`, `Sale`, `StockReceive`, `StockAdjustment` | Validates entire transaction graph |
| Factory Method | `New*()` constructors | `NewItem()`, `NewSale()` |
| Computed Field | `TotalCost`, `Profit`, `TotalSales` | Auto-calculated, never stored directly in domain |

## How Data Flows Through the Domain

```
HTTP Form → Handler parses fields
         → Handler constructs domain aggregate via New*()
         → Domain validates constraints (returns error on failure)
         → Handler calls To*() to flatten aggregate into DB records
         → Handler inserts records via sqlx transaction
         → Handler renders HTML response
```

See [[handlers-overview]] for how handlers use the domain layer.

## Related
- [[domain-item]] — Master data entities
- [[domain-stock]] — Stock calculation aggregate
- [[domain-receiving]] — Receiving aggregate
- [[domain-sales]] — Sales aggregate
- [[domain-adjustment]] — Adjustment aggregate
- [[domain-testing]] — Test coverage and strategy
- [[models-layer]] — How domain maps to DB DTOs
