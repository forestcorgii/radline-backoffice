# Domain Overview

> Back to [[00-index]] · Related: [[architecture]], [[domain-testing]]

## Philosophy
The `domain/` package implements **Domain-Driven Design (DDD)**:
- **Pure Go** — No SQL drivers, no HTTP, no template imports (only `stdlib` + `golang.org/x/crypto/bcrypt` in `auth.go`).
- **Validation at construction** — All `New*()` / `Validate*()` functions enforce invariants before persistence.
- **Aggregates** — Multi-item transactions (`ItemStock`, `StockReceive`, `Sale`, `StockAdjustment`, `PurchaseOrder`) validate and compute totals in memory.

## Complete Entity & Aggregate Map (`domain/`)

| File | Entities / Aggregates | Reference Doc |
|---|---|---|
| [brand.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/brand.go), [category.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/category.go), [item.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/item.go), [uom_setting.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/uom_setting.go) | `Brand`, `Category`, `Item` entities; `UomSetting` value object | [[domain-item]] |
| [customer.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/customer.go) | `Customer` entity (`NewCustomer`) | [[domain-customer]] |
| [stock.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/stock.go) | `ItemStock` aggregate root (`CalculateOnHand`, `CalculateGlobalOnHand`, `GetOldestPLWithStock` FIFO) | [[domain-stock]] |
| [receiving.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/receiving.go) | `ReceivingLog` entity, `StockReceive` aggregate (`ToReceivingLogs`) | [[domain-receiving]] |
| [sales.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/sales.go) | `SalesDetail` entity, `Sale` aggregate (`ToSalesDetails`) | [[domain-sales]] |
| [adjustment.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/adjustment.go) | `InventoryAdjustment` entity, `StockAdjustment` aggregate (`ToInventoryAdjustments`) | [[domain-adjustment]] |
| [purchase_order.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/purchase_order.go) | `PurchaseOrder` aggregate (`CalculateTotals`, `Validate`), `PurchaseOrderItem` | [[domain-customer]] |
| [goal.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/goal.go) | `EarningGoal` entity (`CalculateProgress`, `ValidateEarningGoal`), `GoalProgress` | [[domain-customer]] |
| [reminder.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/reminder.go) | `Reminder` entity (`IsOverdue`, `AssignTo`, `ValidateReminder`) | [[domain-customer]] |
| [auth.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/auth.go) | `User`, `Role` (`HasPermission`), `ActivityLog`, `PermissionDef`, `HashPassword` | [[domain-customer]] |
| `*_test.go` | `stock_test.go`, `uom_setting_test.go`, `receiving_test.go`, `sales_test.go`, `adjustment_test.go`, `customer_test.go`, `goal_test.go`, `reminder_test.go`, `purchase_order_test.go`, `auth_test.go` | [[domain-testing]] |
