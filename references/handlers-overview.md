# Handlers Overview

> Back to [[00-index]] · Related: [[routing]], [[templates-overview]]

## File
[handlers/handlers.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/handlers.go)

## App Struct

```go
type App struct {
    Templates map[string]*template.Template
}
```

All handlers are methods on `*App`. Templates are injected at startup from [[templates-overview|`main.go:parseTemplates()`]].

## Core Rendering Methods

### `Render(w, name, data)`
Renders a template **by name** using `ExecuteTemplate(w, name, data)`. Used for:
- Fragment responses (HTMX table row swaps)
- Templates without a base layout

### `RenderPage(w, r, name, data)`
Smart renderer that checks the `HX-Request` header:
- **HTMX request** (`HX-Request: true`) → Renders only the `{{block "content" .}}` portion, enabling SPA-like content swaps into `#main-content`
- **Full page load** (direct browser navigation) → Renders the full template including `base.html`

This is the key mechanism enabling [[htmx-patterns|SPA-style navigation]] without duplicating templates.

## Dashboard Handler
**In:** [handlers.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/handlers.go#L44-L169)

The dashboard aggregates financial data:
1. Queries `SUM(total_sales)`, `SUM(total_cost)`, `SUM(profit)` from `sales_details`
2. Calculates margin percentage
3. Fetches monthly trend data (last 6 months)
4. Falls back to mock data if no sales exist
5. Computes SVG bar chart coordinates server-side

**Chart rendering:** The bar chart is rendered as inline SVG with coordinates computed in Go — no JavaScript charting library.

## Handler Organization

| File | Domain Area | Handlers |
|---|---|---|
| [handlers.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/handlers.go) | Core + Dashboard | `Render`, `RenderPage`, `DashboardHandler` |
| [master_data.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/master_data.go) | Brands, Categories, Items, Entry | 20+ handlers → [[handlers-master-data]] |
| [inventory.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/inventory.go) | Inventory, Receiving, Adjustments | 12 handlers → [[handlers-inventory]] |
| [sales.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/sales.go) | Sales | 5 handlers → [[handlers-sales]] |

## Common Handler Patterns

### CRUD Pattern (Brands/Categories/Items)
```
ListHandler    → SELECT + Render page or rows
AddHandler     → Validate → INSERT → Render rows + toast
EditFormHandler → SELECT → Render inline edit row
UpdateHandler  → Validate → UPDATE → SELECT → Render row + toast
RowHandler     → SELECT → Render single row
DeleteHandler  → DELETE → toast + empty response
```

### Multi-Item Transaction Pattern (Receiving/Sales/Adjustments)
```
PageHandler    → Load items list + recent logs → Render page
AddHandler     → ParseForm → Build domain aggregate → DB transaction → toast
NewRowHandler  → Load items → Render empty row fragment
ItemRowDetails → Load item defaults (UOM/cost/price) → Render pre-filled row
```

### HTMX-Aware List Rendering
```go
if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content" {
    app.Render(w, "rows_fragment.html", data)  // Just table body
} else {
    app.RenderPage(w, r, "full_page.html", data)  // Full page or content block
}
```

This three-way check handles:
1. **Full page load** — Render entire page with base layout
2. **HTMX navigation** (`HX-Target: main-content`) — Render content block only
3. **HTMX filter/search** (`HX-Target: tbody`) — Render rows fragment only

## Learnings

### Context: Refactoring Consolidated Forms to List-Linked Dedicated Pages
**Problem**: Placing all data entry forms (Brands, Categories, Items, Sales) on a single consolidated "/entry" tab page made navigation unintuitive, disconnected data entry from list views, and led to bloated HTML and routing.
**Enforced Solution**:
- **Surgical Placement**: Added a `.header-bar` at the top of each list page (`brands.html`, `categories.html`, `items.html`, `sales.html`) with a dedicated top-right action button (e.g. `+ Add Brand`) that navigates to its own creation route (e.g. `/brands/new`).
- **Post-Submission Redirect (HX-Location)**: Rather than rendering inline list updates or leaving the user on a blank form, handlers set the `HX-Trigger` for the toast notification and use `HX-Location` (e.g. `w.Header().Set("HX-Location", "/brands")`) to redirect the user back to the list page upon successful submission.
- **Dynamic Select Autocomplete Refresh**: Maintained HTMX event listeners (e.g. `hx-trigger="brand-added from:body"`) on selection dropdowns to fetch updated select components (e.g. `/brands/select`) dynamically whenever a dependency object is created elsewhere.
- **Purge Obsolete Pages**: Deleted the legacy `entry.html` template, removed the `Data Entry` sidebar nav link, and registered the new pages in `main.go`.

### Context: Go HTTP Fragment Response Sniffing (Content-Type text/plain)
**Problem**: Sending HTML fragments starting with table rows `<tr>` or whitespace without setting response headers causes the Go standard library to sniff the content type as `text/plain; charset=utf-8`. HTMX ignores plain text responses, causing requests to hang in the `htmx-request` class.
**Enforced Solution**:
- Centrally set the `Content-Type` header to `text/html; charset=utf-8` in all rendering utilities (e.g. `Render` and `RenderPage`) before executing template files.

## Related
- [[routing]] — Route-to-handler mapping
- [[templates-overview]] — Template hierarchy and parsing
- [[htmx-patterns]] — HTMX interaction patterns
- [[handlers-master-data]], [[handlers-inventory]], [[handlers-sales]] — Specific handlers
