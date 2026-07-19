# Templates Overview

> Back to [[00-index]] · Related: [[handlers-overview]], [[htmx-patterns]]

## File
[main.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/main.go) — `parseTemplates()` function, lines 91–167

## Template Hierarchy

```
templates/
├── base.html              ← Layout shell (sidebar, nav, toast, scripts)
│
├── PAGES (rendered with base.html)
│   ├── dashboard.html
│   ├── brands.html          + brand_row, brand_rows, brand_edit_row
│   ├── categories.html      + category_row, category_rows, category_edit_row
│   ├── items.html           + item_row, item_rows, item_edit_row
│   ├── inventory.html       + inventory_stock_rows
│   ├── stock_receiving.html + receiving_rows, item_select, receiving_item_row
│   ├── stock_adjustments.html + adjustment_rows, item_select, adjustment_item_row
│   ├── sales.html           + sales_rows
│   ├── entry.html           + brand_select, category_select, item_select, sale_item_row
│   ├── monthly_inventory.html + monthly_inventory_rows
│   ├── receiving_logs.html  + receiving_rows
│   └── adjustment_logs.html + adjustment_rows
│
├── FRAGMENTS (rendered standalone, no base.html)
│   ├── brand_row.html / brand_rows.html / brand_edit_row.html
│   ├── category_row.html / category_rows.html / category_edit_row.html
│   ├── item_row.html / item_rows.html / item_edit_row.html
│   ├── receiving_rows.html / adjustment_rows.html / sales_rows.html
│   ├── inventory_stock_rows.html / monthly_inventory_rows.html
│   ├── brand_select.html / category_select.html / item_select.html
│   └── sale_item_row.html / receiving_item_row.html / adjustment_item_row.html
```

## How Template Parsing Works

### Pages (with base layout)
Each page is parsed with `base.html` + the page template + associated sub-templates:

```go
t := template.New(page).Funcs(funcMap)
files := []string{"templates/base.html", "templates/" + page}
// Add associated fragments
if page == "brands.html" {
    files = append(files, "templates/brand_row.html", ...)
}
t = template.Must(t.ParseFiles(files...))
templates[page] = t
```

### Fragments (standalone)
Fragment templates are parsed independently (no base.html):

```go
t := template.New(frag).Funcs(funcMap)
files := []string{"templates/" + frag}
// Some fragments compose others
if frag == "brand_rows.html" {
    files = append(files, "templates/brand_row.html")
}
```

## Template Functions

```go
funcMap := template.FuncMap{
    "derefInt": func(p *int) int {
        if p == nil { return 0 }
        return *p
    },
}
```

Used in templates to safely dereference nullable `*int` fields (e.g., `BrandID`, `CategoryID`).

## Base Layout (`base.html`)

**Structure:**
```html
<body>
  <div class="app-layout">
    <div id="sidebar-backdrop">        <!-- Mobile overlay -->
    <aside id="sidebar">               <!-- Sidebar nav -->
    <div class="main-layout-container">
      <header class="top-header">      <!-- Mobile top bar -->
      <main id="main-content">
        {{ block "content" . }}{{ end }}  <!-- Page content injected here -->
      </main>
    </div>
  </div>
  <div id="toast-container">           <!-- Toast notifications -->
  <script>                             <!-- Nav highlighting, sidebar toggle, toast listener -->
</body>
```

**Key JavaScript (inline):**
- `updateActiveNavLink()` — Highlights sidebar link matching current URL
- `toggleSidebarDropdown()` — Opens/closes Inventory submenu
- `toggleMobileSidebar()` / `closeMobileSidebarOnNav()` — Mobile sidebar control
- Toast listener — Listens for `show-toast` custom events from HTMX

See [[sidebar-navigation]] and [[htmx-patterns]] for details.

## Template Naming Convention

| Pattern | Purpose | Example |
|---|---|---|
| `{entity}.html` | Full page | `brands.html`, `sales.html` |
| `{entity}_row.html` | Single table row | `brand_row.html` |
| `{entity}_rows.html` | Table body (loops over _row) | `brand_rows.html` |
| `{entity}_edit_row.html` | Inline edit form row | `brand_edit_row.html` |
| `{entity}_select.html` | `<select>` options fragment | `brand_select.html` |
| `{entity}_item_row.html` | Multi-item form row | `receiving_item_row.html` |

## Related
- [[handlers-overview]] — How `Render` and `RenderPage` use templates
- [[htmx-patterns]] — How HTMX triggers fragment rendering
- [[ui-design-tokens]] — CSS classes used in templates
- [[sidebar-navigation]] — Base template navigation structure

## Learnings

### Context: Correct Template Context Mapping and Matching Database Fields
**Problem**: After copy-pasting code or templates from another page (e.g., `receiving_logs.html` copied from `adjustment_logs.html`), the rendered table did not load because:
- The template referenced incorrect context fields (`.AdjustmentLogs` instead of `.ReceivingLogs`).
- The query in `inventory.go` did not select all required columns (`i.description as item_description`), causing the fields to be empty.
- The template `receiving_rows.html` referenced a non-existent struct field `.ItemDescription` instead of `.Description`.
**Enforced Solution**:
- **Strict Context Mapping**: Ensure all templates match the exact data structure returned by the Go HTTP handler.
- **Select All Required Columns**: Keep SQL queries aligned with destination DTO/Model structs by selecting all fields explicitly (e.g., `i.description as item_description` to map to `db:"item_description"` tag).
- **Correct Go Struct Field Reference**: Access fields in html templates using the exact case-sensitive Go struct field name (e.g., `.Description`), not the database tag name or a mismatched name.

### Context: Missing Child Templates in Fragment Parsing Causing HTMX 500 Errors
**Problem**: When fetching a sub-fragment or container result template via HTMX (e.g., `sales_results.html` or `sales_rows.html`), Go's `html/template` threw an execution error (`no such template "sale_edit_row.html"`) because child templates referenced inside `{{ template ... }}` calls were omitted from the `files` slice in `parseTemplates()`.
**Enforced Solution**:
- **Include All Transitive Dependencies**: When declaring fragment file lists in `main.go` `parseTemplates()`, always include all leaf row and edit templates (e.g., `sale_row.html`, `sale_edit_row.html`) that any parent container template calls via `{{ template }}`.
- **Register All Standalone Fragments**: Ensure all individual editable or toggleable row templates are registered in the `fragments` slice so `app.Templates["<frag_name>.html"]` exists for standalone handler responses.

