# UI Design Tokens & Component Patterns

> Back to [[00-index]] · Related: [[htmx-patterns]], [[sidebar-navigation]], [[responsive-design]], [[templates-overview]]

## Canonical Stylesheet & Extracted JS Modules
- **Stylesheet**: [`static/index.css`](../static/index.css) (linked in [`templates/base.html`](../templates/base.html) as `/static/index.css?v=8.0` — always increment `?v=` when changing CSS).
- **Extracted Frontend Scripts** (linked in [`templates/base.html`](../templates/base.html)):
  - [`static/js/base-ui.js`](../static/js/base-ui.js) (`/static/js/base-ui.js?v=8.0`) — Sidebar collapse/mobile drawers, `+ New` Quick Actions menu, `#shadcn-alert-dialog` HTMX confirmation interceptors, toasts, listview search clear/reset, and searchable dropdown positioning/caching.
  - [`static/js/table-engine.js`](../static/js/table-engine.js) (`/static/js/table-engine.js?v=8.0`) — Universal List Views Engine (Excel collapsible columns, `.table-col-resizer` 60-FPS resizing, semantic column groups, drag-and-drop column reordering, header themes, Comfortable/Compact density toggle, and hybrid `localStorage` + `POST /api/user-preferences` sync).

## Current Theme & Base Typography (Enterprise Blue & Breathing Room v7.0)
```css
:root {
    --background: 0 0% 100%;             /* #ffffff */
    --foreground: 222 47% 11%;           /* Deep Navy Slate #0f172a */
    --card: 0 0% 100%;
    --primary: 217 91% 53%;              /* Enterprise Blue #2563eb */
    --primary-dark: 221 83% 45%;         /* Deep Ocean Blue #1d4ed8 */
    --primary-tint: 214 100% 97%;        /* Ice Blue Tint #eff6ff */
    --secondary: 214 32% 95%;            /* Cool Slate #eef2f6 */
    --muted: 214 32% 96%;
    --muted-foreground: 215 16% 45%;     /* Slate Gray #606f85 */
    --accent: 214 60% 96%;               /* #eff5fc */
    --destructive: 0 84.2% 60.2%;
    --border: 214 32% 90%;               /* #dfe7f0 */
    --input: 214 28% 86%;
    --ring: 217 91% 53%;
    --radius: 0.625rem;                  /* 10px */
    --radius-sm: 0.5rem;                 /* 8px */

    /* Mapped Application Aliases */
    --bg-color: hsl(214 33% 98%);        /* #f8fafc */
    --surface-color: hsl(var(--card));
    --primary-color: hsl(var(--primary));
    --primary-hover: hsl(var(--primary-dark));
    --text-main: hsl(var(--foreground));
    --text-muted: hsl(var(--muted-foreground));
    --border-color: hsl(var(--border));
}
```
- **Breathing Room Base Scale**: `body` uses `font-size: 0.84375rem` (13.5px), `line-height: 1.5`, `font-variant-numeric: tabular-nums`, and `'Inter', system-ui, sans-serif`.
- **Never hardcode HEX/RGB** in templates when a CSS variable or semantic class exists.

## Core Component Classes

| Category | Classes | Current Spec |
|---|---|---|
| **Cards & Layout** | `.card`, `main#main-content` | White surface, `1px solid var(--border-color)`, generous padding (`1.25rem 1.5rem` card, `1.25rem 1.75rem` main), max-width `1560px` |
| **Buttons** | `.btn`, `.btn-secondary`, `.btn-outline`, `.btn-ghost`, `.btn-success`, `.btn-danger`, `.btn-sm` | Standardized height `2.25rem` (`36px`), `.btn-sm` `1.875rem` (`30px`), radius `0.5rem` |
| **Icon Actions** | `.btn-icon`, `.actions-cell`, `.btn-table-action.btn-delete` | `2rem × 2rem` inline-flex with Lucide SVGs (`stroke="currentColor"`) and `title` tooltip. Row delete stays neutral gray until hover |
| **Inputs & Filters** | `.form-input`, `.form-group`, `.search-filter-bar`, `.date-filters-row`, `.filter-item` | Standardized height `2.25rem` (`36px`), `0.8125rem` font-size, 3px soft focus ring (`0 0 0 3px hsl(var(--ring) / 0.16)`) |
| **Tables / Listviews** | `.table-container`, `.th`/`th`, `.td`/`td`, `.edit-row`, `.excel-table` | Form-sheet aesthetic: `#f8fafc` headers (`2px solid #cbd5e1` bottom border), `0.625rem 0.875rem` cell padding (`13px` text), `.excel-table` rows `2.375rem` (`38px`) |
| **Item Entry Grids** | `.item-grid-header`, `.item-grid-row`, `.item-grid-cell` | `display: flex; flex-direction: row; flex-wrap: nowrap` with `:nth-child(n)` flex ratios (cross-browser Edge/Chrome safe) |
| **Badges & Statuses** | `.badge`, `.badge-active`, `.badge-inactive`, `.status-icon-btn` (`.status-icon-active`, `-posted`, `-cancelled`), `.supplier-select-*` | Subtle pastel pills; Sales Log uses 24px circular `.status-icon-btn` triggering fixed `.status-menu-popover` |
| **Dialogs & Toasts** | `#shadcn-alert-dialog` (`.alert-dialog-*`), `.toast` (`.toast-success`, `-error`, `-warning`, `-info`) | Intercepts `htmx:confirm`, `window.alert`, and `window.confirm` via `window.showShadcnAlertDialog()` |
| **Dashboard KPIs** | `.qb-workflow-ribbon`, `.qb-kpi-card`, `.sub-tabs`, `.sub-tab-item.active` | QuickBooks 4-step workflow hub and KPI metric cards |

---

## Learnings & Enforced UI Patterns

### Context: Universal shadcn/ui Alert Dialog for HTMX Confirmations and JS Alerts
- **Problem**: Native `window.confirm()` / `window.alert()` popups break visual consistency.
- **Enforced Solution**: `#shadcn-alert-dialog` in `templates/base.html` (powered by `/static/js/base-ui.js?v=6.5`) intercepts `htmx:confirm` (calling `evt.detail.issueRequest(true)` on confirm) and overrides `window.alert` / `window.confirm` via `window.showShadcnAlertDialog()`.

### Context: Listviews Form-Sheet Aesthetic & Direct Column Header Collapse / Expand
- **Problem**: Wide tables need spreadsheet-like column hiding/unhiding and clean headers without outer redundant `.header-bar` blocks.
- **Enforced Solution**:
  - **Column Collapse**: `window.initExcelCollapsibleColumns()` in `/static/js/table-engine.js?v=6.5` provides:
    1. Hover collapse button (`.btn-th-collapse`) on collapsible headers and right-click context menu (`showColumnContextMenu`).
    2. Fully hidden collapsed cells (`display: none !important` on matching `th`/`.th` and `td`/`.td`).
    3. Excel double-line divider (`||`) with `.excel-unhide-btn` (`[+]`) on header borders and a `.collapsed-cols-bar` chip bar mounted **outside/above** `.table-container` (so it doesn't clip on horizontal scroll).
    4. Persists in `localStorage` (`radline_collapsed_cols_<tableId>`) and syncs with `user_preferences` and view popovers via `radline-excel-cols-updated`.
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

### Context: Universal List Views Engine (Header Themes, Column Accents, Resizing, Grouping & Reordering)
- **Problem**: Operators need spreadsheet-grade customization across all tabular list views (Sales, Inventory, Monthly Inventory, Items, Receiving, Adjustments) including customizable header colors, interactive column resizing, semantic column grouping with group hide/unhide, column reordering, and cross-device user persistence without extra header rows cluttering table height.
- **Enforced Solution**:
  - **Customizable Header Colors**: Table-wide header themes (`[data-header-theme="slate|indigo|emerald|amber|navy|rose"]`) selectable from `.customize-theme-swatches` in the Customize View popover or header context menu, paired with individual column top-border & tint accents (`[data-col-accent="indigo|emerald|amber|rose|purple|sky"]`) via right-click context menu.
  - **60-FPS Column Resizing**: Active `.table-col-resizer` handles (`width: 6px; cursor: col-resize`) on every `th[data-col]`, injecting dynamic `<style id="radline-universal-widths-<tableId>">` rules on drag and resetting on double-click.
  - **Predefined Semantic Column Groups**: Configured in `window.RADLINE_TABLE_GROUPS` in `/static/js/table-engine.js?v=6.5`. Automatically upgrades `.customize-view-popover` into collapsible `.customize-group-block` sections with master group checkboxes (`.customize-group-cb`) and renders consolidated `.collapsed-group-chip` pills in `.collapsed-cols-bar` when an entire group is hidden.
  - **Column Reordering**: HTML5 Drag & Drop directly on `.th[data-col]` headers (with `.col-drop-left` / `.col-drop-right` indicators) and `←`/`→` buttons (`.btn-col-move`) in the Customize View popover and context menu, reordering header and body cells in DOM across initial loads and `htmx:afterSwap`.
  - **Hybrid Local + Database Persistence**: `window.saveRadlinePreference()` writes immediately to `localStorage` for zero layout shift and debounces `POST /api/user-preferences` (`SaveUserPreferenceHandler` / `GetUserPreferenceHandler` in `handlers/handlers.go`) to sync with the `user_preferences` table.


### Context: Enterprise Blue Theme & Breathing Room Density Scale (v7.0)
- **Problem**: The legacy 12px purple-indigo theme (`#4f46e5`) felt cramped during extended back-office data entry and lacked clear visual separation between containers, tables, and form controls.
- **Enforced Solution**:
  - **Refined Enterprise Blue Palette**: `--primary: 217 91% 53%` (`#2563eb`), `--primary-dark: 221 83% 45%` (`#1d4ed8`), `--primary-tint: 214 100% 97%` (`#eff6ff`), `--foreground: 222 47% 11%` (`#0f172a`), and `--border: 214 32% 90%` (`#dfe7f0`).
  - **More Breathing Room Scale**:
    - `body`: `0.84375rem` (`13.5px`), `line-height: 1.5`.
    - `main#main-content`: `padding: 1.25rem 1.75rem; max-width: 1560px;`.
    - `.card`: `padding: 1.25rem 1.5rem; margin-bottom: 1rem; border-radius: 0.625rem;`.
    - Standardized Form Controls & Buttons (`.form-input`, `input`, `select`, `.btn`): `height: 2.25rem` (`36px`), `font-size: 0.8125rem`, soft 3px focus ring (`0 0 0 3px hsl(var(--ring) / 0.16)`). Compact button variant `.btn-sm` uses `height: 1.875rem` (`30px`).
    - Table Density (`th, td`, `.th, .td`): `padding: 0.625rem 0.875rem` (`10px 14px`), `font-size: 0.8125rem` (`13px`). `.excel-table` row height expanded to `2.375rem` (`38px`).
    - Dashboard Height Tokens: `.dash-h-sm: 175px`, `.dash-h-md: 340px`, `.dash-h-lg: 510px`, `.dashboard-grid` gap `1.25rem`.

### Context: Crisp Enterprise SaaS Listviews, Summary Bars & Density Toggle (v8.0)
- **Problem**: Listviews had non-sticky headers, left-aligned financial numbers, duplicate `class="..."` attributes that dropped color classes, stray `+`/`-` signs (including a `-$` typo on `sold`), inconsistent toolbars, and no live summary/totals bars.
- **Enforced Solution**:
  - **Sticky Headers & Frozen Left Columns**: `.table-container` uses `max-height: calc(100vh - 285px); overflow: auto;` with sticky `.th`/`th` (`top: 0; z-index: 15`) and `.sticky-col-1` (`left: 0; z-index: 10/22`) on key left columns.
  - **Right-Aligned Tabular Numerics (No `+`/`-` Signs)**: Numeric columns use `.th.num` / `.td.num` (`text-align: right; font-variant-numeric: tabular-nums`) and rely on color classes (`text-success`, `text-danger`, `text-primary`) without `+` or `-` prefixes.
  - **Density Toggle**: `[data-table-density="compact"]` vs `comfortable` injected into `.customize-view-popover` by `/static/js/table-engine.js?v=8.0` and persisted via `radline_table_density`.
  - **Unified Search & Live Summary Bar**: `.listview-search-wrap` adds a leading search icon and `×` clear button (`window.clearListSearch`); `.listview-summary-bar` renders live row counts and financial/stock totals inside `#*-results` HTMX targets.
  - **Conditional Sales Date Range**: `sales.html` hides `start_date` and `end_date` whenever `date_preset != "custom"`.
