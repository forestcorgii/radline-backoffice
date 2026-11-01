# UI Design Tokens & Component Patterns

> Back to [[00-index]] · Related: [[htmx-patterns]], [[sidebar-navigation]], [[responsive-design]], [[templates-overview]]

## Canonical Stylesheet
[static/index.css](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/static/index.css) (linked in [templates/base.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/base.html) as `/static/index.css?v=6.1` — always increment `?v=` when changing CSS).

## Current Theme & Base Typography (QuickBooks / shadcn Royal Indigo)
```css
:root {
    --background: 0 0% 100%;             /* #ffffff */
    --foreground: 232 38% 15%;           /* #191c2b */
    --card: 0 0% 100%;
    --primary: 243 75% 59%;              /* Royal Indigo #4f46e5 */
    --primary-dark: 244 75% 48%;         /* #3730a3 */
    --primary-tint: 243 100% 96%;        /* #eef2ff */
    --secondary: 235 48% 96%;            /* #eef1fa */
    --muted: 235 32% 96%;
    --muted-foreground: 230 14% 48%;     /* #6c7285 */
    --accent: 235 60% 97%;               /* #f0f3ff */
    --destructive: 0 84.2% 60.2%;
    --border: 232 20% 90%;               /* #e0e3ed */
    --input: 232 20% 88%;
    --ring: 243 75% 59%;
    --radius: 0.5rem;

    /* Mapped Application Aliases */
    --bg-color: hsl(232 30% 98%);        /* #f8f9fd */
    --surface-color: hsl(var(--card));
    --primary-color: hsl(var(--primary));
    --primary-hover: hsl(var(--primary-dark));
    --text-main: hsl(var(--foreground));
    --text-muted: hsl(var(--muted-foreground));
    --border-color: hsl(var(--border));
}
```
- **High-Density Base Scale**: `body` uses `font-size: 0.75rem` (12px), `line-height: 1.35`, `font-variant-numeric: tabular-nums`, and `'Inter', system-ui, sans-serif`.
- **Never hardcode HEX/RGB** in templates when a CSS variable or semantic class exists.

## Core Component Classes

| Category | Classes | Current Spec |
|---|---|---|
| **Cards & Layout** | `.card`, `main#main-content` | White surface, `1px solid var(--border-color)`, compact padding (`0.75rem 1rem`), max-width `1440px` |
| **Buttons** | `.btn`, `.btn-secondary`, `.btn-outline`, `.btn-ghost`, `.btn-success`, `.btn-danger` | Compact height `1.75rem`–`1.875rem`, radius `0.375rem`–`0.5rem` |
| **Icon Actions** | `.btn-icon`, `.actions-cell`, `.btn-table-action.btn-delete` | `1.75rem × 1.75rem` inline-flex with Lucide SVGs (`stroke="currentColor"`) and `title` tooltip. Row delete stays neutral gray until hover |
| **Inputs & Filters** | `.form-input`, `.form-group`, `.search-filter-bar`, `.date-filters-row`, `.filter-item` | Compact height `1.75rem`–`1.875rem`, single-line filter toolbars without stacked labels |
| **Tables / Listviews** | `.table-container`, `.th`/`th`, `.td`/`td`, `.edit-row` | Form-sheet aesthetic: `#f8fafc` headers (`2px solid #cbd5e1` bottom border), `white-space: nowrap`, zebra striping on `:nth-child(even)`, `#f1f5f9` hover |
| **Item Entry Grids** | `.item-grid-header`, `.item-grid-row`, `.item-grid-cell` | `display: flex; flex-direction: row; flex-wrap: nowrap` with `:nth-child(n)` flex ratios (cross-browser Edge/Chrome safe) |
| **Badges & Statuses** | `.badge`, `.badge-active`, `.badge-inactive`, `.status-icon-btn` (`.status-icon-active`, `-posted`, `-cancelled`), `.supplier-select-*` | Subtle pastel pills; Sales Log uses 24px circular `.status-icon-btn` triggering fixed `.status-menu-popover` |
| **Dialogs & Toasts** | `#shadcn-alert-dialog` (`.alert-dialog-*`), `.toast` (`.toast-success`, `-error`, `-warning`, `-info`) | Intercepts `htmx:confirm`, `window.alert`, and `window.confirm` via `window.showShadcnAlertDialog()` |
| **Dashboard KPIs** | `.qb-workflow-ribbon`, `.qb-kpi-card`, `.sub-tabs`, `.sub-tab-item.active` | QuickBooks 4-step workflow hub and KPI metric cards |

---

## Learnings & Enforced UI Patterns

### Context: Universal shadcn/ui Alert Dialog for HTMX Confirmations and JS Alerts
- **Problem**: Native `window.confirm()` / `window.alert()` popups break visual consistency.
- **Enforced Solution**: `#shadcn-alert-dialog` in `templates/base.html` intercepts `htmx:confirm` (calling `evt.detail.issueRequest(true)` on confirm) and overrides `window.alert` / `window.confirm` via `window.showShadcnAlertDialog()`.

### Context: Listviews Form-Sheet Aesthetic & Direct Column Header Collapse / Expand
- **Problem**: Wide tables need spreadsheet-like column hiding/unhiding and clean headers without outer redundant `.header-bar` blocks.
- **Enforced Solution**:
  - **Column Collapse**: `window.initExcelCollapsibleColumns()` in `templates/base.html` provides:
    1. Hover collapse button (`.btn-th-collapse`) on collapsible headers and right-click context menu (`showColumnContextMenu`).
    2. Fully hidden collapsed cells (`display: none !important` on matching `th`/`.th` and `td`/`.td`).
    3. Excel double-line divider (`||`) with `.excel-unhide-btn` (`[+]`) on header borders and a `.collapsed-cols-bar` chip bar mounted **outside/above** `.table-container` (so it doesn't clip on horizontal scroll).
    4. Persists in `localStorage` (`radline_collapsed_cols_<tableId>`) and syncs with view popovers via `radline-excel-cols-updated`.
  - **Strict Card Header Div Matching**: Always close all nested `.customize-view-container` and header flex wrappers before `<form>` and `.table-container` begin.

### Context: Fixed Viewport Popovers & Sub-100ms Form Item Search
- **Problem**: Table stacking contexts clip inline popovers (like Sales row status menus or item pickers), and unindexed `LEFT JOIN` item searches lag.
- **Enforced Solution**:
  - **Fixed Viewport Positioning**: Row popovers (`.status-menu-popover`, `.dropdown-list`) compute coordinates via `getBoundingClientRect()` and use `position: fixed; z-index: 999999;`.
  - **Fast Item Search**: `db/db.go` indexes `items(code, description, model, brand_id, category_id)`. `SearchItemsHandler` (`GET /items/search`) uses subquery filters (`LIMIT 20`, max `50`), paired with client-side caching (`_itemSearchHtmlCache` / `_itemSearchJsonCache`).

### Context: Sales Log Stacked Deductions & Dynamic Status/Supplier Color Coding
- **Problem**: Separate columns for `Patong`, `POS Charge`, and `WT 2307` wasted horizontal space, and status/supplier selects needed theme-consistent color coding.
- **Enforced Solution**:
  - **Stacked Deductions Column**: Consolidated into a single `Deductions` column (`data-col="patong"`). Read-only rows stack non-zero values (`P:`, `C:`, `W:`) or `—`; edit rows stack three compact inputs (`name="patong"`, `pos_charge`, `wt_2307`).
  - **Dynamic Select Color Sync**: Use `.status-select-*` and `.supplier-select-*` classes in `static/index.css` updated on change via `window.updateSelectColorClass(el)`.

### Context: Customizable Dashboard Grid, Sizing Tokens & Drag-and-Drop Repositioning
- **Problem**: Fixed dashboard layouts restrict operators with varied workflows and monitor setups from prioritizing relevant KPIs, charts, and operational widgets.
- **Enforced Solution**:
  - **12-Column Responsive Grid**: `.dashboard-grid` (`display: grid; grid-template-columns: repeat(12, 1fr); gap: 1rem;`).
  - **Modular Width Tokens**: `.dash-w-quarter` (span 3 on desktop, 6 on tablet, 12 on mobile), `.dash-w-half` (span 6), `.dash-w-full` (span 12).
  - **Modular Height Tokens**: `.dash-h-sm` (155px), `.dash-h-md` (310px), `.dash-h-lg` (480px). Internal `.dashboard-card-body` uses `flex: 1 1 auto; min-height: 0; overflow-y: auto;` to handle arbitrary content height cleanly.
  - **Drag-and-Drop & Accessibility**: HTML5 Drag & Drop with `.dash-drop-indicator` and `.is-dragging` styles, coupled with Left/Right directional move buttons in card settings popovers (`.dash-config-popover`).
  - **Per-User Database Sync**: Persisted in `user_preferences` (`pref_key = 'dashboard_layout'`) with zero layout shift on server-rendered page load.
