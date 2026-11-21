# Domain Overview

> Back to [[00-index]] · Related: [[architecture]], [[domain-testing]]

## Philosophy
The `domain/` package in Radline BackOffice (Go module `rav`) implements **Domain-Driven Design (DDD)**:
- **Pure Go** — No SQL drivers, no HTTP, no template imports (only `stdlib` + `golang.org/x/crypto/bcrypt` in `domain/auth.go`).
- **Validation at construction** — All `New*()` / `Validate*()` functions enforce invariants before persistence.
- **Aggregates** — Multi-item transactions (`ItemStock`, `StockReceive`, `Sale`, `StockAdjustment`, `PurchaseOrder`) validate and compute totals in memory.

## Complete Entity & Aggregate Map (`domain/`)

| File | Entities / Aggregates | Reference Doc |
|---|---|---|
| [`brand.go`](../domain/brand.go), [`category.go`](../domain/category.go), [`item.go`](../domain/item.go), [`uom_setting.go`](../domain/uom_setting.go) | `Brand`, `Category`, `Item` entities; `UomSetting` value object | [[domain-item]] |
| [`customer.go`](../domain/customer.go) | `Customer` entity (`NewCustomer`) | [[domain-customer]] |
| [`stock.go`](../domain/stock.go) | `ItemStock` aggregate root (`CalculateOnHand`, `CalculateGlobalOnHand`, `GetOldestPLWithStock` FIFO) | [[domain-stock]] |
| [`receiving.go`](../domain/receiving.go) | `ReceivingLog` entity, `StockReceive` aggregate (`ToReceivingLogs`) | [[domain-receiving]] |
| [`sales.go`](../domain/sales.go) | `SalesDetail` entity, `Sale` aggregate (`SetCustomerDetails`, `ToSalesDetails`) | [[domain-sales]] |
| [`adjustment.go`](../domain/adjustment.go) | `InventoryAdjustment` entity, `StockAdjustment` aggregate (`ToInventoryAdjustments`) | [[domain-adjustment]] |
| [`purchase_order.go`](../domain/purchase_order.go) | `PurchaseOrder` aggregate (`CalculateTotals`, `Validate`), `PurchaseOrderItem` | [[domain-customer]] |
| [`goal.go`](../domain/goal.go) | `EarningGoal` entity (`CalculateProgress`, `ValidateEarningGoal`), `GoalProgress` | [[domain-customer]] |
| [`reminder.go`](../domain/reminder.go) | `Reminder` entity (`IsOverdue`, `AssignTo`, `ValidateReminder`) | [[domain-customer]] |
| [`auth.go`](../domain/auth.go) | `User`, `Role` (`HasPermission`), `ActivityLog`, `PermissionDef`, `HashPassword`, `CheckPasswordHash` | [[domain-customer]] |
| `*_test.go` | `stock_test.go`, `uom_setting_test.go`, `receiving_test.go`, `sales_test.go`, `adjustment_test.go`, `auth_test.go`, `customer_test.go`, `goal_test.go`, `purchase_order_test.go`, `reminder_test.go`, `brand_test.go`, `category_test.go`, `item_test.go` | [[domain-testing]] |
