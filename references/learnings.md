# Learnings Log

> 📚 **Full Knowledge Vault:** See [[00-index]] for the interconnected Obsidian-style reference docs covering architecture, domain models, handlers, templates, HTMX patterns, and UI design tokens.

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

## Context: UI Layout Compactness and Information Density Optimization
**Problem**: The BackOffice UI spacing was too sparse, limiting the amount of details (table rows, form controls, and layout cards) visible on screen at once, requiring excessive scrolling.
**Enforced Solution**:
- **Global Sizing**: Enforce a global `font-size: 0.875rem` on the HTML `body` selector to scale all standard elements proportionally.
- **Sidebar & Viewport**: Reduce the default sidebar width to `220px` (from `260px`) and expand `main#main-content` max-width to `1440px` (from `1200px`) to maximize screen estate utilization on wide monitors.
- **Spacing Reduction**: Shrink card padding to `1rem`, table cell padding to `0.5rem 0.75rem`, form input paddings to `0.45rem 0.75rem`, button paddings to `0.45rem 1rem`, form group vertical spacing to `0.75rem`, grid gap to `1rem`, and sub-tab margin/paddings.

## Context: Refactoring Consolidated Forms to List-Linked Dedicated Pages
**Problem**: Placing all data entry forms (Brands, Categories, Items, Sales) on a single consolidated "/entry" tab page made navigation unintuitive, disconnected data entry from list views, and led to bloated HTML and routing.
**Enforced Solution**:
- **Surgical Placement**: Added a `.header-bar` at the top of each list page (`brands.html`, `categories.html`, `items.html`, `sales.html`) with a dedicated top-right action button (e.g. `+ Add Brand`) that navigates to its own creation route (e.g. `/brands/new`).
- **Post-Submission Redirect (HX-Location)**: Rather than rendering inline list updates or leaving the user on a blank form, handlers set the `HX-Trigger` for the toast notification and use `HX-Location` (e.g. `w.Header().Set("HX-Location", "/brands")`) to redirect the user back to the list page upon successful submission.
- **Dynamic Select Autocomplete Refresh**: Maintained HTMX event listeners (e.g. `hx-trigger="brand-added from:body"`) on selection dropdowns to fetch updated select components (e.g. `/brands/select`) dynamically whenever a dependency object is created elsewhere.
- **Purge Obsolete Pages**: Deleted the legacy `entry.html` template, removed the `Data Entry` sidebar nav link, and registered the new pages in `main.go`.

## Context: Timezone-Safe SQLite Date Extraction
**Problem**: In SQLite, using `strftime('%Y-%m', date)` on ISO-8601 strings containing timezone offsets (e.g. `+08:00` or `Z`) can return `NULL` or empty, causing database scan errors in Go (e.g. `converting NULL to string is unsupported`).
**Enforced Solution**:
- Replace `strftime('%Y-%m', date_column)` with `substr(date_column, 1, 7)` in SQLite queries where YYYY-MM extraction is required. Since ISO-8601 datetimes consistently begin with `YYYY-MM-DD`, substring extraction is timezone-safe, parsing-independent, and extremely robust.

## Context: Nullable Foreign Key DTO Scans
**Problem**: Database query scans into struct fields (like `adjustment_id` or `brand_id`) fail with `converting NULL to int is unsupported` when records contain `NULL` for those columns (e.g. legacy adjustments without headers).
**Enforced Solution**:
- Define database-mapped DTO fields that can be `NULL` as pointer types (e.g. `AdjustmentID *int` instead of `int`).
- Safely dereference them in mapper layers (e.g., in `models/stock.go`) before constructing pure domain objects (which require concrete, non-pointer types like `int`).
- Pointers to standard types are automatically dereferenced in Go's `html/template` package when rendered.

## Context: FIFO Cost & Selling Price Tracking (Oldest PL with Stock)
**Problem**: Tracking and auto-populating inventory item prices and costs based on the oldest transaction batch (PL/Receiving Log) that still has available stock.
**Enforced Solution**:
- **Domain FIFO Computation**: Implement `GetOldestPLWithStock()` on the `ItemStock` domain aggregate:
  1. Sort all receiving logs chronologically (`Date ASC, ID ASC`).
  2. Compute net consumed stock `consumed = totalReceived - onHand`.
  3. Loop through sorted logs, deducting each log's quantity from `consumed`.
  4. The first log whose received quantity is greater than the remaining `consumed` value has active stock under FIFO. Return its cost and price.
- **Auto-population in Form Handlers**: In HTMX handlers that fetch pre-filled form detail rows (e.g. `/sales/item-row-details`), fetch the item stock aggregate, run the FIFO calculation, and fall back to the latest receiving log if no stock is currently on hand.

## Context: HTMX Out-of-Band (OOB) Pagination and Implicit Template Registration
**Problem**: Rendering list views with thousands of items (e.g. Sales, Items) causes UI sluggishness and N+1 query bottlenecks. Adding pagination while preserving user search/filter/sorting states requires updating both the table body and pagination controls without losing input focus.
**Enforced Solution**:
- **Implicit Template Fragment**: Define pagination markup in a separate template file (e.g. [pagination.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/pagination.html)) without `{{ define }}` wrappers. This allows the template to be compiled implicitly with its filename as the key.
- **Fragment Registration**: Register `"pagination.html"` inside the `fragments` list in `main.go` so it is registered standalone in `app.Templates`.
- **Double Template Rendering**: In handlers on HTMX requests, render the rows template (e.g. `item_rows.html`), and then call `app.Render(w, "pagination.html", paginationView)` to append it.
- **HTMX OOB Swaps**: The pagination container uses `hx-swap-oob="true"` with a matching DOM ID (e.g. `id="items-pagination"`). HTMX automatically swaps the body rows into the target table body and swaps the pagination block into the footer out-of-band.
- **Filter Retention**: Pagination buttons use `hx-include` to gather active filter inputs (e.g. search, sort), while filters omit the page parameter (defaulting to 1 on trigger) so that modifying search/sort automatically resets pagination.


## Context: Go HTTP Fragment Response Sniffing (Content-Type text/plain)
**Problem**: Sending HTML fragments starting with table rows `<tr>` or whitespace without setting response headers causes the Go standard library to sniff the content type as `text/plain; charset=utf-8`. HTMX ignores plain text responses, causing requests to hang in the `htmx-request` class.
**Enforced Solution**:
- Centrally set the `Content-Type` header to `text/html; charset=utf-8` in all rendering utilities (e.g. `Render` and `RenderPage`) before executing template files.

## Context: SPA Layout Viewport Scroll Reset on Transition
**Problem**: When navigating between pages using HTMX swaps targeting a main layout container (e.g., `#main-content`), the page scroll position does not reset to the top, remaining at the scroll offset of the previous page.
**Enforced Solution**:
- Add a global HTMX swap listener:
  ```javascript
  document.body.addEventListener("htmx:afterSwap", function(evt) {
      if (evt.detail.target.id === "main-content") {
          window.scrollTo(0, 0);
      }
  });
  ```
- This resets scroll position back to top on layout transitions, but preserves scroll offsets when searching, paginating, or filtering.

## Context: Batch Querying and Database Indexing for Page Performance
**Problem**: Fetching domain models individually inside page rendering loops creates an N+1 query pattern, which results in significant page load latency when handling larger datasets.
**Enforced Solution**:
- **Batch Querying**: Implement batch loader functions (e.g. `FetchItemsStockBatch(db, itemIDs)`) that execute single SQL `IN (?)` queries across all needed tables rather than executing separate queries per row loop.
- **Database Indexing**: Add standard indices on search text fields (e.g., `code`) and foreign key columns (`item_id`, `adjustment_id`) in SQLite schema definition to prevent full table scans on group by / filter queries.

## Context: Conditional HTMX Out-of-Band (OOB) Swaps for SPA Page Transitions
**Problem**: Marking elements (like a pagination block) with `hx-swap-oob="true"` globally causes HTMX to discard those elements when they are returned as part of a layout transition request targeting `#main-content`, because the target ID of the OOB element does not exist in the DOM before the swap.
**Enforced Solution**:
- **Conditional hx-swap-oob**: Use a boolean flag like `IsOOB` in the view data. Render `hx-swap-oob="true"` in the template conditionally:
  ```html
  <div id="{{ .TargetID }}-pagination" class="pagination-root"{{ if .IsOOB }} hx-swap-oob="true"{{ end }}>
  ```
- **Automatic Request Detection**: Set `IsOOB` dynamically in the view builder function (e.g. `BuildView` or handler mapping) by checking:
  ```go
  isOOB := r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content"
  ```
  This ensures the element renders as a standard inline component during full page loads and sidebar clicks (targeting `main-content`), while correctly acting as an OOB swap element during inline search, filtering, and page-change updates.

## Context: Inline Add Forms in Settings Section (No Navigate-Away)
**Problem**: Clicking "+ Add Brand/Category/UOM Setting" in the Settings page navigated to separate `/brands/new`, `/categories/new`, or `/uom-settings/new` pages, requiring a round-trip and losing context of where the user was.
**Enforced Solution**: 
- **Inline Toggle Forms**: Replaced `<a>` navigation links in `settings.html` with `<button>` elements that toggle a hidden inline form (`display: none <-> block`) within the same card section via `toggleAddForm(formId)` JavaScript.
- **Form Contents**: Each inline form is a compact card with a dashed border and light blue background, placed directly above the search/filter bar. It contains the minimal fields needed, styled to match the existing compact design.
- **Post-Submit Cleanup**: Forms use `hx-swap="none"` and `onsubmit="setTimeout(function(){ cancelAddForm('form-id') }, 100)"` to hide the form briefly after submission, before `HX-Location` redirects back to `/settings`.
- **Items for Inline UOM Form**: The `SettingsHandler` now also fetches `.Items` (from `items` table) to populate the searchable dropdown in the inline UOM setting form.
- **Searchable Dropdown Reuse**: The dropdown JavaScript functions (`showDropdown`, `filterDropdown`, `selectDropdownItem`, etc.) were duplicated from `settings_new.html` into `settings.html`'s script block for the inline UOM item selector.
- The standalone pages (`brand_new.html`, `category_new.html`, `uom_settings_new.html`, `settings_new.html`) are preserved for direct URL access but no longer linked from the Settings page.

## Context: Action Column Buttons to Icon-Based Buttons
**Problem**: Textual buttons ("Edit", "Delete", "Save", "Cancel", "Remove") in table rows and Actions columns take up significant horizontal screen space, squishing inputs and tables on smaller viewports.
**Enforced Solution**:
- **CSS Utility Classes**: Added `.btn-icon` and `.actions-cell` helper classes in [index.css](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/static/index.css):
  ```css
  .btn-icon {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 0;
      width: 1.85rem;
      height: 1.85rem;
      min-width: 1.85rem;
      border-radius: 0.375rem;
  }
  .btn-icon svg {
      width: 0.9rem;
      height: 0.9rem;
  }
  .actions-cell {
      display: flex;
      gap: 0.35rem;
      align-items: center;
  }
  ```
- **Lucide Inline SVGs**: Replaced text inside buttons with lightweight, inline SVG icons (using `stroke="currentColor"` to dynamically inherit parent text colors) in all row/edit templates:
  - **Edit**: secondary pencil icon
  - **Delete / Remove**: red trash icon
  - **Save**: green checkmark icon
  - **Cancel**: secondary X icon
- **A11y Tooltips**: Provided `title` attributes on all icon buttons to guarantee hover clarity and accessibility.


