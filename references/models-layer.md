# Models Layer (DTOs)

> Back to [[00-index]] · Related: [[domain-overview]], [[database-schema]]

## Files
- [models/models.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/models/models.go) — All DTO struct definitions
- [models/stock.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/models/stock.go) — Stock calculation bridge function

## Purpose

The `models` package serves as the **Data Transfer Object (DTO)** layer between the database and handlers. Structs use `db:` tags for `sqlx` struct scanning.

## Key Distinction: Domain vs Models

| Aspect | `domain/` | `models/` |
|---|---|---|
| Purpose | Business logic & invariants | DB row mapping |
| Tags | None | `db:"column_name"` |
| Nullable fields | `int` (0 = unset) | `*int` (nil = NULL) |
| Dependencies | stdlib only | `database/sql`, `sqlx`, `domain/` |
| Computed fields | Calculated in constructors | Stored as-is from DB |

## DTO Structs

### Base DTOs (match DB tables 1:1)
| Struct | Maps To |
|---|---|
| `Brand` | `brands` (+ `item_count` computed via JOIN) |
| `Category` | `categories` (+ `item_count` computed via JOIN) |
| `Item` | `items` |
| `UomSetting` | `uom_settings` |
| `ReceivingLog` | `receiving_logs` |
| `SalesDetail` | `sales_details` |
| `InventoryAdjustment` | `inventory_adjustments` |

### Extended DTOs (JOINed data)
| Struct | Base + | Extra Fields |
|---|---|---|
| `ItemWithRelations` | `Item` | `BrandName`, `CategoryName` |
| `ReceivingLogWithItem` | `ReceivingLog` | `ItemCode` |
| `SalesDetailWithItem` | `SalesDetail` | `ItemCode` |
| `InventoryAdjustmentWithItem` | `InventoryAdjustment` | `ItemCode` |

### Read Model
| Struct | Purpose |
|---|---|
| `ItemStockView` | Pre-computed stock overview (used by SQL aggregation in [[handlers-inventory]]) |

## `CalculateStockOnHand` Bridge Function

**File:** [models/stock.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/models/stock.go)

This function bridges the DB layer and the domain layer:

```
1. Load Item DTO from DB
2. Map Item DTO → domain.Item (handling *int → int conversion)
3. Load UomSettings from DB → map to []domain.UomSetting
4. Load ReceivingLogs (filtered by supplier) → map to []domain.ReceivingLog
5. Load SalesDetails (filtered by supplier) → map to []domain.SalesDetail
6. Load InventoryAdjustments → map to []domain.InventoryAdjustment
7. Construct domain.NewItemStock(...)
8. Return stock.CalculateOnHand(supplierName)
```

### Nullable Pointer Conversion Pattern
```go
brandID := 0
if dbItem.BrandID != nil {
    brandID = *dbItem.BrandID
}
```
This is used because [[domain-item|domain entities]] use `int` (0 = unset) while models use `*int` (nil = SQL NULL).

## Related
- [[domain-overview]] — Pure domain types that models map to
- [[domain-stock]] — `ItemStock.CalculateOnHand` called from `CalculateStockOnHand`
- [[database-schema]] — Tables these DTOs represent
- [[handlers-overview]] — Handlers consume these DTOs
