# Domain — Customer, Auth, Goals, Reminders & Purchase Orders

> Back to [[00-index]] · Related: [[domain-overview]], [[database-schema]], [[routing]]

## Customer Entity
**File:** [`domain/customer.go`](../domain/customer.go) · **Handler:** [`handlers/customers.go`](../handlers/customers.go)

```go
type Customer struct {
    ID, Code, Name, TINNo, Address, ContactPerson, Phone, Email, PaymentTerms, Notes
}
```
- **Invariants (`NewCustomer`)**: `Code` and `Name` cannot be empty (whitespace trimmed). `PaymentTerms` defaults to `"Net 30 Days"` if empty.
- **Handler Note**: `AddCustomerHandler` auto-generates `CUST-%04d` if `code` is omitted in the form, and emits `HX-Trigger: {"show-toast": ..., "customer-added": {}}`.
- **Search (`GET /customers/search`)**: Multi-word `LIKE` across `code`, `name`, `tin_no`, `contact_person` (`LIMIT 20`, max `100`). Returns `.dropdown-item` HTML (`onmousedown="selectCustomerItem(this)"`) or JSON (`format=json`).

---

## Auth, RBAC & Activity Log Entities
**File:** [`domain/auth.go`](../domain/auth.go) · **Handlers:** [`handlers/auth.go`](../handlers/auth.go), [`handlers/roles_users.go`](../handlers/roles_users.go), [`handlers/activity_logs.go`](../handlers/activity_logs.go)

- **`User` & `Role`**: `Role.HasPermission(key)` always returns `true` for `"Admin"` (case-insensitive) or if `key` is in `Role.Permissions`.
- **Validations**: `ValidateUsername` (≥ 3 chars, no whitespace/slashes), `ValidateRole` (≥ 2 chars), `HashPassword` (≥ 4 chars, `golang.org/x/crypto/bcrypt`).
- **17 Permission Keys**: `dashboard:view`, `sales:view|create|edit|delete`, `inventory:view|receive|adjust`, `items:view|edit`, `settings:view|edit|roles|users`, `logs:view`, `tools:import|scanner`.

---

## EarningGoal Entity
**File:** [`domain/goal.go`](../domain/goal.go) · **Handler:** [`handlers/goals.go`](../handlers/goals.go)

- **Invariants (`ValidateEarningGoal`)**: Non-empty `Title` and `TargetPeriod` (`"YYYY-MM"` or `"YYYY"`), `PeriodType` in `{"Monthly", "Yearly"}`, `TargetRevenue > 0`, `TargetProfit > 0`.
- **`CalculateProgress(actualRev, actualProfit)`**: Returns `GoalProgress` with status `"Exceeded"` (both ≥ 100%), `"On Track"` (avg ≥ 75%), `"In Progress"` (50–74%), or `"Behind"` (< 50%).

---

## Reminder Entity
**File:** [`domain/reminder.go`](../domain/reminder.go) · **Handler:** [`handlers/reminders.go`](../handlers/reminders.go)

- **Invariants (`ValidateReminder`)**: Non-empty `Title`, non-zero `DueDate`, `Priority` in `{"Low", "Medium", "High"}`, `Status` in `{"Pending", "Completed"}`.
- **Methods**: `IsOverdue()`, `AssignTo(userID, userName)`, `Unassign()`, `IsAssigned()`.

---

## PurchaseOrder Aggregate
**File:** [`domain/purchase_order.go`](../domain/purchase_order.go) · **Handler:** [`handlers/forms.go`](../handlers/forms.go)

- **Invariants (`Validate`)**: Non-empty `PONo` and `VendorName`, non-zero `Date`, at least 1 valid `PurchaseOrderItem` with `Qty > 0` and `UnitCost >= 0`.
- **`CalculateTotals()`**: Computes each item's `TotalAmount = Qty * UnitCost`, `Subtotal`, optional 12% VAT (`VATAmt = (Subtotal + Freight) * 0.12` when `ApplyVAT` is true), and `GrandTotal`.

