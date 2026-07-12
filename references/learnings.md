# Learnings Log

## Context: BackOffice System Go Rewrite
**Problem**: Migrating from Python Flask to Go + HTMX. Ensuring adherence to the `golang-htmx-dev` rules.
**Enforced Solution**: 
- Handlers will strictly return HTML fragments.
- State is managed purely on the server via HTMX attributes.
- Use `html/template` combined with `sqlx` for database querying and rendering.
- No JSON endpoints are allowed unless strictly necessary for an external integration (none planned).

## Context: SQLite Database Driver on Windows/CGO-disabled Environments
**Problem**: Compiling the project on environments without a C compiler (like a default Windows/VS Code setup) fails because `github.com/mattn/go-sqlite3` requires CGO (`CGO_ENABLED=1`) and a GCC toolchain.
**Enforced Solution**:
- Use `modernc.org/sqlite`, a pure Go implementation of SQLite that does not require CGO.
- Register/connect the driver in `sqlx` as `"sqlite"` instead of `"sqlite3"`.

## Context: Multi-Parameter Search, Filtering, and Sorting with HTMX
**Problem**: Preserving user inputs (e.g. search query, current filter, current sort) across HTMX table refreshes without resetting focus, cursor position, or selection states.
**Enforced Solution**:
- Target only the table body (`<tbody>`) with HTMX updates, leaving search/filter inputs untouched.
- On each search/filter/sort input, use `hx-include` to gather parameters from other input fields (e.g., `hx-include="[name='filter'],[name='sort']"`).
- Use `hx-trigger="keyup changed delay:300ms, search"` on text inputs to throttle requests, and `hx-trigger="change"` on dropdowns to trigger updates instantly.

## Context: DDD Domain Isolation and Testing Strategy
**Problem**: Embedding database querying logic or framework-specific DTOs directly within business calculations violates domain model sovereignty and makes unit testing complex or impossible without active database/mocking structures.
**Enforced Solution**:
- Implement pure domain models under a `domain` package, completely free of `db` tag structures, SQL engines, or UI templates.
- Define complex business concepts as aggregates (such. as `domain.ItemStock`), value objects, or entities that encapsulate their invariants and compute metrics purely in memory.
- Within persistence and integration wrappers (e.g. `models/stock.go`), load the necessary data from database tables, map the data to the pure domain objects, run the calculations, and return the outputs.
- Write unit tests (`*_test.go`) exclusively within the `domain` package, keeping them fast, self-contained, and isolated from external dependencies.

## Context: Safe Dynamic SQLite Column Renaming in Go Startups
**Problem**: Renaming a table column requires schema updates. Doing so without migration frameworks (like golang-migrate) or manual CLI scripts can cause existing deployments to fail or crash if the column name doesn't exist yet, or cause data loss if handled incorrectly.
**Enforced Solution**:
- Implement inline dynamic check-and-run queries inside database startup initialization:
  1. Try querying the new column with a `LIMIT 0` query.
  2. If it errors (indicating the column does not exist), check if the old column exists.
  3. If the old column exists, execute `ALTER TABLE ... RENAME COLUMN old_col TO new_col;`.
  4. This handles both new database schema generation (which directly uses the new column name) and existing database schemas dynamically on server startup.

## Context: Multi-Row / Tabular Form Submission in HTMX (No JSON)
**Problem**: Encoding a master-detail structure (e.g., a transaction header with multiple line items) in HTML forms without using complex client-side JSON serialization, which violates standard hypermedia constraints.
**Enforced Solution**:
- Use identical `name` attributes for the fields across all dynamic rows (e.g. `name="item_id"`, `name="qty"`, `name="uom"`).
- HTMX submits these as standard multi-value form parameters. In Go, call `r.ParseForm()` and access them as slices (`r.Form["item_id"]`, `r.Form["qty"]`).
- Validate that all input slices have matching lengths before constructing domain models.
- Implement inline browser-side JS like `onclick="this.closest('tr').remove()"` for row removal to avoid unnecessary server roundtrips.
- Clean up extra rows on successful submission by listening to the `hx-on::after-request` event (e.g. trimming the items table back to a single default row).

## Context: Frontend UI/UX, CSS Design System, and HTMX Toast Notifications
**Problem**: Modifying or creating new UI templates without violating the established visual aesthetics (glassmorphic styling, standard typography, color harmony) or breaking the toast feedback system.
**Enforced Solution**:
- **Design Token Invariance**: Do not hardcode HEX or RGB colors in HTML templates. Always reference the CSS variables defined in [index.css](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/static/index.css) (e.g. `--primary-color`, `--bg-color`, `--surface-color`, `--text-main`, `--text-muted`, `--border-color`, `--glass-bg`, `--glass-border`, `--shadow`).
- **Standard Component Classes**: Leverage existing CSS classes:
  - Containers: `.card` (for boxed content with blur/backdrop and shadows).
  - Actions: `.btn` (primary blue), `.btn-secondary` (gray), `.btn-success` (green), and `.btn-danger` (red).
  - Lists and Filters: `.search-filter-bar` (flex row layout), `.form-input` / `.form-group` for inputs.
  - Status Indicators: `.badge .badge-active` (green badge) and `.badge .badge-inactive` (red badge).
- **HTMX Toast Notifications Protocol**: Always use the custom `show-toast` event to report back status from server actions.
  - In Go HTTP handlers, trigger toasts by adding the `HX-Trigger` header in the HTTP response:
    ```go
    w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Brand created successfully!"}}`)
    ```
  - Standard types are `"success"`, `"error"`, `"warning"`, and `"info"`.
- **UI Verification Protocol**:
  - Before declaring a UI task done, run the application (`go run main.go`) and use the `browser_subagent` to render the affected pages, verify layout alignment, check colors, and capture screenshots for visual validation.

## Context: Scalable UI/UX, Large Datasets, and Autocomplete Search
**Problem**: Rendering thousands of dynamic records (e.g. items, customers) in standard `<select>` elements causes performance degradation and poor user search experience.
**Enforced Solution**:
- **Avoid Plain Dropdowns for Large Datasets**: Do not use standard `<select>` dropdowns for data models that scale beyond a few dozen records (e.g., Items). Keep dropdowns reserved only for small, static enums (e.g. Doc Types, Suppliers).
- **Use HTMX Searchable Inputs / Autocomplete**:
  - Implement a text input with search behavior:
    ```html
    <input type="text" 
           placeholder="Search items..." 
           hx-get="/entry/search-items" 
           hx-trigger="keyup changed delay:300ms" 
           hx-target="#search-results" 
           hx-indicator="#loading-spinner">
    ```
  - Alternatively, use HTML `<datalist>` populated dynamically by backend matching to leverage native autocomplete overlays.
- **Enforce Backend Search Limits**: All backend endpoints rendering selection options or list fragments must enforce a strict pagination or limit constraint (e.g., `LIMIT 50`) to keep HTTP payload and database query times minimal.
- **Keyboard Friendliness**: Ensure search suggestion overlays support keyboard navigation (e.g., Enter key to select the first match) or simple navigation cues where possible.

## Context: Programmatic HTMX AJAX & Out-of-Band (OOB) Swaps
**Problem**: Invoking `htmx.ajax(method, path, context)` programmatically from a change or click handler without setting a `swap` style will target the `<body>` element by default, causing the entire page content to be replaced with empty space.
**Enforced Solution**:
- Always pass `swap: 'none'` in the context parameters of programmatic `htmx.ajax(...)` calls if they are solely intended to execute server logic or rely on Out-of-Band (OOB) swaps:
  ```javascript
  htmx.ajax('GET', '/endpoint', { values: { id: val }, swap: 'none' })
  ```

## Context: HTMX Event Bubbling and Form Resets
**Problem**: Inline event handlers on forms like `hx-on::after-request="this.reset()"` will listen to all HTMX-related requests that bubble up from child elements, causing the form to reset unexpectedly when sub-actions (like dynamically adding table rows) are triggered.
**Enforced Solution**:
- Inside form `hx-on::after-request` or `htmx:afterRequest` listeners, validate that the target element is the form itself using `event.detail.elt === this` before performing a reset, ensuring sub-requests do not trigger form resets.

## Context: Unified Form Entry and Log Views (HTMX Same-Page Updates)
**Problem**: Consolidating a data entry form and its corresponding history log table onto a single page without full page reloads, while ensuring the table updates dynamically upon form submission.
**Enforced Solution**:
- Place the data entry `<form>` and historical logs `<table>` in the same tab/page container.
- Configure the `<form>` to submit to the creation endpoint using `hx-target="#tbody-id"` and `hx-swap="innerHTML"`.
- The creation handler should return the updated rows HTML fragment (`*_rows.html`) rather than a full page or redirects.
- Chain a client-side cleanup/reset trigger using `hx-on::after-request="if(event.detail.successful && event.detail.elt === this) this.reset()"` to clear the form upon successful database insertion.

## Context: Multi-Item Stock Receiving UI
**Problem**: Previously the receiving form only allowed a single item per stocktake, requiring users to submit repeatedly. Need a dynamic table to add multiple items in one transaction.
**Enforced Solution**:
- Created `receiving_item_row.html` fragment with HTMX-driven item selection and defaults.
- Updated `inventory.html` to render a multi-item table with “Add Item Row” and removal.
- Added routes `/inventory/receiving/new-row` and `/inventory/receiving/item-row-details`.
- Implemented `NewReceivingRowHandler` and `ReceivingItemRowDetailsHandler`.
- Added `StockReceive` aggregate in `domain/receiving.go` with validation and conversion to `ReceivingLog`.
- Updated `ReceiveStockHandler` to parse array form fields, validate via domain, and insert within a transaction.
- Updated template parsing to include `receiving_item_row.html`.
- Added unit tests for `StockReceive`.

## Context: Multi-Item Stock Adjustment with Header Table
**Problem**: The stock adjustment form only allowed a single item per submission, requiring users to submit repeatedly. Grouping adjustments by a shared reason/date required a mechanism to batch items together.
**Enforced Solution**:
- Created a `stock_adjustments` header table (`id INTEGER PRIMARY KEY AUTOINCREMENT`, `date`, `remarks`) and added `adjustment_id INTEGER` FK to `inventory_adjustments`.
- Added `StockAdjustment` aggregate in `domain/adjustment.go` with `StockAdjustmentItem` value objects, full validation, and `ToInventoryAdjustments()` mapper.
- Created `adjustment_item_row.html` fragment with HTMX-driven item selection and UOM auto-fill.
- Updated `inventory.html` to render a multi-item table with "Add Item Row" / "Remove" and "Commit Adjustment" button.
- Added routes `/inventory/adjustments/new-row` and `/inventory/adjustments/item-row-details`.
- Implemented `NewAdjustmentRowHandler` and `AdjustmentItemRowDetailsHandler`.
- Updated `AdjustStockHandler` to parse array form fields, validate via domain aggregate, insert header + items within a single DB transaction.
- Added safe migration for existing `inventory_adjustments` tables missing the `adjustment_id` column.
- Added unit tests for `StockAdjustment` in `domain/adjustment_test.go`.

## Context: Mobile-Responsive Spreadsheet Tables and Form Elements
**Problem**: Tabular forms and logs containing inputs, selects, and many data cells compress to unusable widths or overflow card blocks on mobile viewports.
**Enforced Solution**:
- Wrap all wide tables in a layout container with horizontal overflow support: `.table-container { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }`.
- Style scrollbars with subtle, custom styled WebKit tracks to keep the UX sleek and premium.
- Apply min-width constraints (e.g. `.table-scroll-md` at 800px, `.table-scroll-lg` at 1000px) directly to the table element to prevent input squishing and preserve tabular layout integrity on mobile devices.
- Replace hardcoded filter widths with responsive `.filter-item` wrappers, and grid layouts with responsive media-query classes (e.g. `.grid-cols-2`, `.grid-cols-3`, `.grid-cols-5`) to stack vertically on mobile and span horizontally on desktop.
