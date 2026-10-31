# Handlers Overview

> Back to [[00-index]] · Related: [[routing]], [[templates-overview]], [[htmx-patterns]]

## Core File
[handlers/handlers.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/handlers.go)

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

## Complete Handler File Map (`handlers/`)

| File | Responsibility | Reference |
|---|---|---|
| [handlers.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/handlers.go) | `App`, `Render`, `RenderPage`, `DashboardHandler` (KPIs + SVG trend charts) | This file |
| [middleware.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/middleware.go) | `AuthMiddleware`, `RequirePermission`, `LogActivity` | [[routing]] |
| [auth.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/auth.go), [roles_users.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/roles_users.go), [activity_logs.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/activity_logs.go) | Login/logout, `#user-chip-container`, Roles/Permissions CRUD, Users CRUD, Activity Logs | [[routing]], [[domain-customer]] |
| [master_data.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/master_data.go) | Brands, Categories, Items CRUD + `SearchItemsHandler` (`/items/search`) | [[handlers-master-data]] |
| [customers.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/customers.go) | Customers CRUD + `SearchCustomersHandler` (`/customers/search`) | [[domain-customer]] |
| [inventory.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/inventory.go) | Stock Overview, Monthly Inventory, Receiving & Adjustments CRUD/logs | [[handlers-inventory]] |
| [sales.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/sales.go) | Sales CRUD, batch edit, inline status popover (`UpdateSalesStatusHandler`) | [[handlers-sales]] |
| [forms.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/forms.go) | Printable Quotation & Purchase Order forms (`SavePurchaseOrderHandler`) | [[routing]], [[domain-customer]] |
| [goals.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/goals.go), [reminders.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/reminders.go) | Earning Goals & Operational Calendar / Reminders | [[routing]], [[domain-customer]] |
| [settings.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/settings.go), [uom_settings.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/uom_settings.go) | System/DeepSeek Vision settings, UOM predefined list, UOM conversion rules | [[routing]] |
| [import.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/import.go), [receipt_scanner.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/receipt_scanner.go) | Excel batch import (`excelize`) and AI Receipt Vision Scanner | [[routing]] |
| [pagination.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/pagination.go) | `GetLimitParam(r)` helper (defaults to `25`, max `500`) | [[htmx-patterns]] |
