# Handlers Overview

> Back to [[00-index]] · Related: [[routing]], [[templates-overview]], [[htmx-patterns]]

## Core File
[`handlers/handlers.go`](../handlers/handlers.go)

## `App` Struct & Rendering Methods
```go
type App struct {
    Templates map[string]*template.Template
}
```
Both `Render` and `RenderPage` explicitly set `w.Header().Set("Content-Type", "text/html; charset=utf-8")` so Go never sniffs fragments starting with `<tr>` or whitespace as `text/plain`.

1. **`Render(w, name, data)`**: Executes a standalone fragment template (`ExecuteTemplate(w, name, data)`). Used for inline table/row HTMX swaps.
2. **`RenderPage(w, r, name, data)`**: Checks `r.Header.Get("HX-Request") == "true"`:
   - **HTMX Navigation (`HX-Request: true`)** → Executes only the `"content"` block (`ExecuteTemplate(w, "content", data)`) for `#main-content` swap.
   - **Direct Browser Navigation** → Executes the full page with `base.html`.

### Standard 3-Way HTMX Check in List Handlers
```go
if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
    app.Render(w, "entity_results.html", data) // Filter/search/limit swap
} else {
    app.RenderPage(w, r, "entity.html", data)  // Full page or SPA #main-content swap
}
```

## Complete Handler File Map (17 Files in `handlers/`)

| File | Responsibility | Reference |
|---|---|---|
| [`handlers.go`](../handlers/handlers.go) | `App`, `Render`, `RenderPage`, `DashboardHandler` (KPIs + SVG trend charts), `GetUserPreferenceHandler`, `SaveUserPreferenceHandler` | This file |
| [`middleware.go`](../handlers/middleware.go) | `AuthMiddleware`, `RequirePermission`, `LogActivity` | [[routing]] |
| [`auth.go`](../handlers/auth.go) | Login/logout (`LoginPageHandler`, `LoginSubmitHandler`, `LogoutHandler`), `UserChipHandler` (`#user-chip-container`) | [[routing]], [[domain-customer]] |
| [`roles_users.go`](../handlers/roles_users.go) | Roles & Permissions CRUD (`RolesHandler`, `AddRoleHandler`, `UpdateRoleHandler`, `DeleteRoleHandler`), Users CRUD (`UsersHandler`, `AddUserHandler`, `UpdateUserHandler`) | [[routing]], [[domain-customer]] |
| [`activity_logs.go`](../handlers/activity_logs.go) | Activity & audit trail logs (`ActivityLogsHandler`) | [[routing]], [[domain-customer]] |
| [`master_data.go`](../handlers/master_data.go) | Brands, Categories, Items CRUD + `SearchItemsHandler` (`/items/search`) | [[handlers-master-data]] |
| [`customers.go`](../handlers/customers.go) | Customers CRUD + `SearchCustomersHandler` (`/customers/search`) | [[domain-customer]] |
| [`inventory.go`](../handlers/inventory.go) | Stock Overview, Monthly Inventory, Receiving & Adjustments CRUD/logs, `FetchItemsStockBatch` | [[handlers-inventory]] |
| [`sales.go`](../handlers/sales.go) | Sales CRUD, batch edit, inline status popover (`UpdateSalesStatusHandler`) | [[handlers-sales]] |
| [`forms.go`](../handlers/forms.go) | Printable Quotation & Purchase Order forms (`SavePurchaseOrderHandler`) | [[routing]], [[domain-customer]] |
| [`goals.go`](../handlers/goals.go) | Earning Goals CRUD & progress tracking (`GoalsHandler`, `AddGoalHandler`, `DeleteGoalHandler`) | [[routing]], [[domain-customer]] |
| [`reminders.go`](../handlers/reminders.go) | Operational Calendar & Reminders CRUD/rescheduling | [[routing]], [[domain-customer]] |
| [`settings.go`](../handlers/settings.go) | System/DeepSeek Vision settings, UOM predefined list CRUD (`UomsHandler`, `SelectUomsHandler`) | [[routing]] |
| [`uom_settings.go`](../handlers/uom_settings.go) | UOM conversion rules CRUD (`UomSettingsHandler`) | [[routing]] |
| [`import.go`](../handlers/import.go) | Excel batch import (`excelize`) | [[routing]] |
| [`receipt_scanner.go`](../handlers/receipt_scanner.go) | AI Receipt Vision Scanner (`ReceiptScannerHandler`, `ReceiptScannerParseHandler`) | [[routing]] |
| [`pagination.go`](../handlers/pagination.go) | `GetLimitParam(r)` helper (defaults to `25`, max `500`) | [[htmx-patterns]] |
