# Templates Overview

> Back to [[00-index]] · Related: [[handlers-overview]], [[htmx-patterns]], [[ui-design-tokens]]

## File
[main.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/main.go) — `parseTemplates()` (lines 221–389)

## Template Registration Architecture (`parseTemplates()`)
Templates are pre-compiled at startup into `map[string]*template.Template` with helper functions `add`, `mul`, `derefInt`, and `dict`:

1. **`pages` (rendered with `templates/base.html` via `app.RenderPage`)**:
   - **Analytics & Ops**: `dashboard.html` (+ `dashboard_trends.html`), `goals.html`, `reminders.html`
   - **Masterlist**: `items.html`, `item_new.html`, `customers.html`, `brands.html`, `brand_new.html`, `categories.html`, `category_new.html`, `uoms.html`, `uom_settings.html`, `uom_settings_new.html`
   - **Inventory**: `inventory.html`, `monthly_inventory.html`, `stock_receiving.html`, `receiving_logs.html`, `stock_adjustments.html`, `adjustment_logs.html`
   - **Sales & Forms**: `sales.html`, `sales_new.html`, `form_quotation.html`, `form_purchase_order.html`
   - **Tools & Admin**: `import.html`, `receipt_scanner.html`, `roles.html`, `users.html`, `activity_logs.html`, `settings.html`, `settings_new.html`, `system_settings.html`
2. **`fragments` (rendered standalone without `base.html` via `app.Render`)**:
   - Every partial rendered directly by an HTMX handler (`*_row.html`, `*_rows.html`, `*_edit_row.html`, `*_results.html`, `*_select.html`, `*_item_row.html`, `dashboard_trends.html`, `login.html`) is registered in the `fragments` map.

## Terminology Mapping: "List Views"
When the user refers to **"list views"**, map to these templates:
- **Sales Logs**: [templates/sales.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/sales.html) (`sales_results.html`, `sales_rows.html`, `sale_row.html`, `sale_edit_row.html`)
- **Current Stock Levels**: [templates/inventory.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/inventory.html) (`inventory_stock_rows.html`)
- **Monthly Inventory**: [templates/monthly_inventory.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/monthly_inventory.html) (`monthly_inventory_rows.html`)
- **Receiving Logs**: [templates/receiving_logs.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/receiving_logs.html) (`receiving_logs_results.html`, `receiving_rows.html`, `receiving_log_row.html`, `receiving_log_edit_row.html`)
- **Adjustment Logs**: [templates/adjustment_logs.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/adjustment_logs.html) (`adjustment_logs_results.html`, `adjustment_rows.html`)
- **Items / Customers / Brands / Categories / UOMs**: `items.html`, `customers.html`, `brands.html`, `categories.html`, `uoms.html`

---

## Learnings & Enforced Template Rules

### Context: Transitive Child Templates in `parseTemplates()`
- **Problem**: Calling `{{ template "child.html" . }}` inside a page or fragment throws `no such template` at runtime (HTTP 500) if `child.html` was not included in that template's `ParseFiles(files...)` slice in `main.go`.
- **Enforced Solution**: Whenever a template calls `{{ template "x.html" }}`, add `"templates/x.html"` to the `files` slice of **both** the parent page (`pages` loop) and any parent fragment (`fragments` loop) in `main.go`.

### Context: Reusable Action Form Partials (`receive_stock_form.html`, `adjust_stock_form.html`)
- Extracted into standalone partials included across `inventory.html`, `receiving_logs.html`, and `adjustment_logs.html`. Any handler rendering those pages must provide `ReceivingRowData` / `AdjustmentRowData` (items + UOMs) and include `receiving_item_row.html` / `adjustment_item_row.html` in `parseTemplates()`.

### Context: Strict Struct Field & SQL Projection Alignment
- Templates access **Go struct field names** (case-sensitive, e.g., `.Description`), never raw `db:` tag names. Always verify that the handler's SQL query selects every column required by the target DTO struct.
