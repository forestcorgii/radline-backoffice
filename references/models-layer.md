# Models Layer (DTOs)

> Back to [[00-index]] · Related: [[domain-overview]], [[database-schema]]

## Files
- [`models/models.go`](../models/models.go) — Core catalog, inventory, sales, purchase order, customer, and dashboard layout DTOs
- [`models/auth.go`](../models/auth.go) — Authentication, RBAC (`Role`, `User`, `UserSession`), and `ActivityLog` DTOs
- [`models/goals_reminders.go`](../models/goals_reminders.go) — `EarningGoal`, `GoalWithProgress`, `Reminder`, `TopProduct`, and `SalesInventoryTrend` DTOs

*(Note: Handlers use `FetchItemsStockBatch` in `handlers/inventory.go` to batch-load and construct `domain.ItemStock` aggregates.)*

## Purpose

The `models` package serves as the **Data Transfer Object (DTO)** layer between the database (`sqlx`) and HTTP handlers/templates. Structs use `db:` tags for `sqlx` struct scanning and `json:` tags where serialized by targeted JSON endpoints.

## Key Distinction: Domain vs Models

| Aspect | `domain/` | `models/` |
|---|---|---|
| Purpose | Business logic & invariants | DB row mapping & template view models |
| Tags | None | `db:"column_name"` (and `json:"..."` where needed) |
| Nullable fields | Value types (`int` `0`, `string` `""`) | Pointer types (`*int`, `*int64`, `*string`, `*float64`, `*time.Time`) |
| Dependencies | stdlib (+ `bcrypt` in `auth.go`) | stdlib (`time`) |
| Computed fields | Calculated in constructors | Scanned from SQL projections or embedded base structs |

## DTO Structs by File

### 1. [`models/models.go`](../models/models.go)
| Struct | Maps To / Purpose | Nullable Pointer Fields & Composition Notes |
|---|---|---|
| `Brand`, `Category` | `brands`, `categories` (+ joined `item_count`) | — |
| `Uom`, `UomSetting`, `UomSettingWithItem` | `uoms`, `uom_settings` (+ item code/description) | — |
| `Item`, `ItemWithRelations` | `items` (+ joined `brand_name`, `category_name`) | `Model *string`, `BrandID *int`, `CategoryID *int`, `Variation *string`, `Remarks *string` (can embed `Item` in `ItemWithRelations`) |
| `ReceivingLog`, `ReceivingLogWithItem` | `receiving_logs` (+ joined `item_code`, `item_description`) | `PLNo *string`, `UnitPrice *float64`, `SellingPrice *float64` (can embed `ReceivingLog`) |
| `SalesDetail`, `SalesDetailWithItem` | `sales_details` (+ joined `item_code`, `item_description`) | `CustomerName *string`, `TINNo *string`, `Address *string`, `RefPL *string` (can embed `SalesDetail`) |
| `InventoryAdjustment`, `InventoryAdjustmentWithItem` | `inventory_adjustments` (+ joined `item_code`, `item_description`) | `AdjustmentID *int`, `Remarks *string` (can embed `InventoryAdjustment`) |
| `ItemStockView` | Inventory stock overview read model | Pre-computed `TotalReceived`, `TotalSold`, `TotalAdjusted`, `OnHand`, `CurrentCost`, `CurrentPrice` |
| `PurchaseOrder`, `PurchaseOrderItem` | `purchase_orders`, `purchase_order_items` | `DueDate *time.Time` |
| `Customer`, `CustomerWithStats` | `customers` (+ aggregated `total_sales`, `sales_count`) | Embeds/extends `Customer` with `json:` tags for `/customers/search?format=json` |
| `DashboardCardConfig` | JSON layout entry in `user_preferences` | `ID`, `Width` (`quarter`/`half`/`full`), `Height` (`small`/`medium`/`large`) |

### 2. [`models/auth.go`](../models/auth.go)
| Struct | Maps To / Purpose | Notes |
|---|---|---|
| `Role` | `roles` (+ `user_count` & `Permissions []string`) | `ID int64` |
| `User` | `users` (+ joined `role_name`) | `ID int64`, `RoleID int64`, `IsActive bool` |
| `UserSession` | `user_sessions` | `ID string` (session token), `UserID int64`, `ExpiresAt time.Time` |
| `ActivityLog` | `activity_logs` | `ID int64`, `UserID *int64` (nullable for system/anonymous events) |

### 3. [`models/goals_reminders.go`](../models/goals_reminders.go)
| Struct | Maps To / Purpose | Notes |
|---|---|---|
| `EarningGoal` | `earning_goals` | `ID int64`, `PeriodType`, `TargetPeriod`, `TargetRevenue`, `TargetProfit` |
| `GoalWithProgress` | Embeds `EarningGoal` | Adds computed `ActualRevenue`, `ActualProfit`, `RevenueProgress`, `ProfitProgress`, `RevenueVariance`, `ProfitVariance`, `Status` |
| `Reminder` | `reminders` (+ joined `assigned_to_name`) | `ID int64`, `AssignedToUserID *int64` (nullable FK to `users`) |
| `TopProduct` | Dashboard top products read model | Ranked by `TotalSales` / `TotalProfit` |
| `SalesInventoryTrend` | Dashboard monthly trend read model | `Month`, `Sales`, `Profit`, `InventoryValue` |

## Nullable Pointer Conversion Pattern
```go
brandID := 0
if dbItem.BrandID != nil {
    brandID = *dbItem.BrandID
}
```
Domain entities use concrete value types (`int` where `0` = unset, `string` where `""` = empty), while `models/` DTOs use pointer types (`*int`, `*int64`, `*string`, `*float64`, `*time.Time`) so `sqlx` can scan SQL `NULL` values without error.

## Learnings

### Context: Nullable Foreign Key & Column DTO Scans
**Problem**: Database query scans into struct fields (like `adjustment_id`, `brand_id`, `assigned_to_user_id`, or nullable text/numeric columns) fail with `converting NULL to int/string/float64 is unsupported` when records contain `NULL`.
**Enforced Solution**:
- Define database-mapped DTO fields that can be `NULL` in SQL as pointer types (`*int`, `*int64`, `*string`, `*float64`, `*time.Time`) or wrap projections in `COALESCE(...)`.
- Safely dereference pointers in handler/mapper functions before constructing pure domain objects.
- Pointers to primitive types are automatically dereferenced by Go's `html/template` when rendered (or via the `derefInt` template helper).

## Related
- [[domain-overview]] — Pure domain types that models map to
- [[domain-stock]] — `ItemStock` aggregate populated via `FetchItemsStockBatch`
- [[database-schema]] — 21 tables these DTOs represent
- [[handlers-overview]] — Handlers consume these DTOs
