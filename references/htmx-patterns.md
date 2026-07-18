# HTMX Patterns

> Back to [[00-index]] · Related: [[handlers-overview]], [[templates-overview]]

## Core Architecture

Radline uses **HTMX 1.9.10** (loaded via CDN in `base.html`) for all dynamic interactions. There is **no JavaScript framework** — all state lives on the server.

## SPA-Style Navigation

The sidebar navigation uses HTMX for partial page updates:

```html
<nav class="sidebar-nav" hx-target="#main-content" hx-push-url="true">
    <a href="/brands" hx-get="/brands">Brands</a>
</nav>
```

- `hx-target="#main-content"` → Swap only the main content area
- `hx-push-url="true"` → Update browser URL and history
- [[handlers-overview|`RenderPage`]] checks `HX-Request` header and renders only the `{{block "content"}}` portion

**Result:** Clicking sidebar links feels like an SPA without a full page reload.

---

## Toast Notifications Protocol

### Server Side (Go)
```go
w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Brand created!"}}`)
```

Types: `success`, `error`, `warning`, `info`

### Client Side (base.html)
```javascript
document.body.addEventListener('show-toast', function(evt) {
    // Creates toast DOM element with icon + message
    // Auto-removes after 4 seconds with fadeOut animation
});
```

---

## Search, Filter, Sort Pattern

Used in Brands, Categories, Items, Sales, Logs pages:

```html
<input type="search" name="search"
       hx-get="/brands"
       hx-trigger="keyup changed delay:300ms, search"
       hx-target="#brands-tbody"
       hx-include="[name='filter'],[name='sort']">

<select name="filter" class="form-input"
        hx-get="/brands"
        hx-trigger="change"
        hx-target="#brands-tbody"
        hx-include="[name='search'],[name='sort']">
```

**Key rules:**
- `hx-target` points to `<tbody>` (not the whole page) → preserves filter inputs
- `hx-include` gathers all other filter params → maintains state
- `delay:300ms` throttles search input → prevents excessive requests
- See [[handlers-overview]] for the three-way HTMX check in handlers

---

## Inline Edit Pattern (CRUD Tables)

```html
<!-- Display row -->
<tr id="brand-row-5">
    <td>BRD5</td>
    <td>Brand Five</td>
    <td><button hx-get="/brands/edit/5" hx-target="#brand-row-5" hx-swap="outerHTML">Edit</button></td>
</tr>

<!-- Click Edit → swaps to: -->
<tr id="brand-row-5" class="edit-row">
    <td><input name="code" value="BRD5"></td>
    <td><input name="name" value="Brand Five"></td>
    <td>
        <button hx-post="/brands/edit/5" hx-target="#brand-row-5" hx-swap="outerHTML">Save</button>
        <button hx-get="/brands/row/5" hx-target="#brand-row-5" hx-swap="outerHTML">Cancel</button>
    </td>
</tr>
```

---

## Multi-Row Form Submission

For receiving, adjustments, and sales with multiple items:

```html
<tr>
    <td><select name="item_id">...</select></td>
    <td><input name="qty" type="number"></td>
    <td><input name="uom" type="text"></td>
    <td><input name="cost" type="number"></td>
</tr>
<!-- More rows with identical name attributes -->
```

Go server reads as arrays:
```go
r.ParseForm()
itemIDs := r.Form["item_id"]   // []string
qtys := r.Form["qty"]          // []string
```

**Adding rows:** Button calls `hx-get="/inventory/receiving/new-row"` → appends new `<tr>` to the table body.

**Removing rows:** Inline JS `onclick="this.closest('tr').remove()"` — no server roundtrip.

---

## Out-of-Band (OOB) Swaps

Used by `ItemDefaultsHandler` to auto-fill fields when an item is selected:

```html
<!-- Server returns: -->
<input id="receiving_uom" name="uom" value="PCS" hx-swap-oob="true">
<input id="receiving_cost" name="cost" value="80.00" hx-swap-oob="true">
```

These replace elements **anywhere on the page** matching the `id`, regardless of the main `hx-target`.

**Critical:** When calling `htmx.ajax()` programmatically for OOB-only responses, always pass `swap: 'none'`:
```javascript
htmx.ajax('GET', '/endpoint', { values: { id: val }, swap: 'none' })
```

---

## Form Reset After Submission

```html
<form hx-on::after-request="if(event.detail.successful && event.detail.elt === this) this.reset()">
```

**Key guard:** `event.detail.elt === this` prevents the form from resetting when child elements (like "Add Row" buttons) trigger their own HTMX requests.

---

## HX-Location Redirect

After successful multi-item submissions (receiving/adjustments), the handler sends:
```go
w.Header().Set("HX-Location", "/inventory/receiving/logs")
```

This triggers HTMX to navigate to the logs page, similar to `hx-push-url` but triggered from the server.

## Learnings

### Context: Multi-Parameter Search, Filtering, and Sorting with HTMX
**Problem**: Preserving user inputs (e.g. search query, current filter, current sort) across HTMX table refreshes without resetting focus, cursor position, or selection states.
**Enforced Solution**:
- Target only the table body (`<tbody>`) with HTMX updates, leaving search/filter inputs untouched.
- On each search/filter/sort input, use `hx-include` to gather parameters from other input fields (e.g., `hx-include="[name='filter'],[name='sort']"`).
- Use `hx-trigger="keyup changed delay:300ms, search"` on text inputs to throttle requests, and `hx-trigger="change"` on dropdowns to trigger updates instantly.

### Context: Multi-Row / Tabular Form Submission in HTMX (No JSON)
**Problem**: Encoding a master-detail structure (e.g., a transaction header with multiple line items) in HTML forms without using complex client-side JSON serialization, which violates standard hypermedia constraints.
**Enforced Solution**:
- Use identical `name` attributes for the fields across all dynamic rows (e.g. `name="item_id"`, `name="qty"`, `name="uom"`).
- HTMX submits these as standard multi-value form parameters. In Go, call `r.ParseForm()` and access them as slices (`r.Form["item_id"]`, `r.Form["qty"]`).
- Validate that all input slices have matching lengths before constructing domain models.
- Implement inline browser-side JS like `onclick="this.closest('tr').remove()"` for row removal to avoid unnecessary server roundtrips.
- Clean up extra rows on successful submission by listening to the `hx-on::after-request` event (e.g. trimming the items table back to a single default row).

### Context: Programmatic HTMX AJAX & Out-of-Band (OOB) Swaps
**Problem**: Invoking `htmx.ajax(method, path, context)` programmatically from a change or click handler without setting a `swap` style will target the `<body>` element by default, causing the entire page content to be replaced with empty space.
**Enforced Solution**:
- Always pass `swap: 'none'` in the context parameters of programmatic `htmx.ajax(...)` calls if they are solely intended to execute server logic or rely on Out-of-Band (OOB) swaps:
  ```javascript
  htmx.ajax('GET', '/endpoint', { values: { id: val }, swap: 'none' })
  ```

### Context: HTMX Event Bubbling and Form Resets
**Problem**: Inline event handlers on forms like `hx-on::after-request="this.reset()"` will listen to all HTMX-related requests that bubble up from child elements, causing the form to reset unexpectedly when sub-actions (like dynamically adding table rows) are triggered.
**Enforced Solution**:
- Inside form `hx-on::after-request` or `htmx:afterRequest` listeners, validate that the target element is the form itself using `event.detail.elt === this` before performing a reset, ensuring sub-requests do not trigger form resets.

### Context: Unified Form Entry and Log Views (HTMX Same-Page Updates)
**Problem**: Consolidating a data entry form and its corresponding history log table onto a single page without full page reloads, while ensuring the table updates dynamically upon form submission.
**Enforced Solution**:
- Place the data entry `<form>` and historical logs `<table>` in the same tab/page container.
- Configure the `<form>` to submit to the creation endpoint using `hx-target="#tbody-id"` and `hx-swap="innerHTML"`.
- The creation handler should return the updated rows HTML fragment (`*_rows.html`) rather than a full page or redirects.
- Chain a client-side cleanup/reset trigger using `hx-on::after-request="if(event.detail.successful && event.detail.elt === this) this.reset()"` to clear the form upon successful database insertion.

### Context: HTMX Out-of-Band (OOB) Pagination and Implicit Template Registration
**Problem**: Rendering list views with thousands of items (e.g. Sales, Items) causes UI sluggishness and N+1 query bottlenecks. Adding pagination while preserving user search/filter/sorting states requires updating both the table body and pagination controls without losing input focus.
**Enforced Solution**:
- **Implicit Template Fragment**: Define pagination markup in a separate template file (e.g. `pagination.html`) without `{{ define }}` wrappers. This allows the template to be compiled implicitly with its filename as the key.
- **Fragment Registration**: Register `"pagination.html"` inside the `fragments` list in `main.go` so it is registered standalone in `app.Templates`.
- **Double Template Rendering**: In handlers on HTMX requests, render the rows template (e.g. `item_rows.html`), and then call `app.Render(w, "pagination.html", paginationView)` to append it.
- **HTMX OOB Swaps**: The pagination container uses `hx-swap-oob="true"` with a matching DOM ID (e.g. `id="items-pagination"`). HTMX automatically swaps the body rows into the target table body and swaps the pagination block into the footer out-of-band.
- **Filter Retention**: Pagination buttons use `hx-include` to gather active filter inputs (e.g. search, sort), while filters omit the page parameter (defaulting to 1 on trigger) so that modifying search/sort automatically resets pagination.

### Context: SPA Layout Viewport Scroll Reset on Transition
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

### Context: Conditional HTMX Out-of-Band (OOB) Swaps for SPA Page Transitions
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

### Context: Ditching Pagination in favor of Limit Selector and Instant Auto-Submission Filtering
**Problem**: Traditional pagination adds complexity (page counts, offsets, navigation buttons, out-of-band updates) and friction for user searches. Need a simpler way to view the top rows of filtered datasets and query/filter instantly without a manual submit button.
**Enforced Solution**:
- **Ditch Pagination**: Delete all page navigation elements and templates (such as `pagination.html`), and remove pagination-related parsing configurations from `main.go`.
- **Add Limit Selector**: Introduce a dropdown in all filter bars named `limit` that lets users select how many top rows to display (e.g., options `25`, `50`, `100`, `200`, `500` rows, defaulting to `25`).
- **Simplify Queries**: Replace `LIMIT ? OFFSET ?` in SQL statements with `LIMIT ?`, passing the parsed limit directly. Remove count queries (e.g. `SELECT COUNT(*)`) completely from the list handlers.
- **Enable Auto-Submission**: Remove manual "Apply"/"Go" buttons from filter forms. Trigger filtering instantly using HTMX attributes on the inputs:
  - Text search: `hx-trigger="keyup changed delay:300ms, search"` (includes debounce delay).
  - Select filters (including the new limit dropdown): `hx-trigger="change"` (triggers immediately).
  - Ensure all inputs target the table results container and specify `hx-include="closest form"` to preserve the values of other filter elements.

## Related
- [[handlers-overview]] — `Render` and `RenderPage` methods
- [[templates-overview]] — Template structure
- [[sidebar-navigation]] — Navigation setup
- [[ui-design-tokens]] — CSS classes referenced in templates
