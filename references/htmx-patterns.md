# HTMX Patterns

> Back to [[00-index]] · Related: [[handlers-overview]], [[templates-overview]], [[ui-design-tokens]]

## Core Architecture
Radline uses **HTMX** (`/static/htmx.min.js` in `templates/base.html`) for all dynamic interactions. All application state is server-rendered hypermedia (HTML fragments), with JSON used only where explicitly needed by client-side helpers (`/items/search?format=json`, `/customers/search?format=json`).

---

## 1. SPA-Style Navigation & Explicit Targets
- `<nav class="sidebar-nav" hx-target="#main-content" hx-push-url="true">` swaps page content into `#main-content`.
- **Links/Buttons Inside `#main-content`**: Do **not** inherit the sidebar's `hx-target`. Every page navigation link or form inside `<main id="main-content">` **must** explicitly set `hx-target="#main-content" hx-push-url="true"`.
- **Scroll Reset**: `base.html` listens to `htmx:afterSwap` and resets `window.scrollTo(0, 0)` only when `evt.detail.target.id === "main-content"`.
- **Server-Driven Navigation (`HX-Location`)**: After form submissions that navigate or refresh `#main-content`, emit:
  ```go
  w.Header().Set("HX-Location", `{"path": "/target-url", "target": "#main-content"}`)
  ```
  Avoid `HX-Redirect` for in-app transitions because it forces a full browser reload and drops pending `HX-Trigger` toasts.

---

## 2. Toast Notifications Protocol (`HX-Trigger`)
```go
w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Saved successfully!"}}`)
```
- **Types**: `"success"`, `"error"`, `"warning"`, `"info"`.
- Can be combined with custom domain events in the same JSON header, e.g.:
  `{"show-toast": {"type": "success", "message": "Added!"}, "customer-added": {}}`

---

## 3. Auto-Submitting Filter Bars & Limit Selector (No Pagination)
- **No Page-Number Pagination**: Traditional pagination (`pagination.html`, `OFFSET`) was removed in favor of a **`limit` dropdown** (`25`, `50`, `100`, `200`, `500`, default `25`) parsed via `GetLimitParam(r)` in `handlers/pagination.go`.
- **Instant Filtering Pattern**:
  - Text search: `hx-trigger="keyup changed delay:300ms, search"`
  - Select filters & `limit` dropdown: `hx-trigger="change"`
  - Target the results container (`hx-target="#<entity>-results"` or `<tbody>`) with `hx-include="closest form"` and `hx-indicator="#<page>-loading-indicator"`.

---

## 4. Inline Row Editing & Multi-Row Form Submission
- **Inline Edit**: Display row `#<entity>-row-{id}` swaps `outerHTML` with `GET /<entity>/edit/{id}` (`.edit-row`), saves via `POST /<entity>/edit/{id}`, or cancels via `GET /<entity>/row/{id}`.
- **Multi-Row Forms (Receiving, Adjustments, Sales)**:
  - Dynamic rows share identical `name` attributes (`item_id`, `qty`, `uom`, `cost`, `price`).
  - Go handlers call `r.ParseForm()` and read slices (`r.Form["item_id"]`), validating equal slice lengths before constructing domain aggregates.
  - Row removal is client-side (`this.closest('tr').remove()` or `.item-grid-row` removal).
- **Safe Form Reset Guard**:
  ```html
  <form hx-on::after-request="if(event.detail.successful && event.detail.elt === this) this.reset()">
  ```
  Always check `event.detail.elt === this` so child HTMX requests (like "Add Row") bubbling up do not reset the form.

---

## 5. Out-of-Band (OOB) Swaps & Programmatic `htmx.ajax`
- **Programmatic OOB Calls**: Calling `htmx.ajax()` without a `swap` option defaults to swapping `<body>`! Always pass `swap: 'none'` when relying on OOB swaps or side-effect endpoints:
  ```javascript
  htmx.ajax('GET', '/endpoint', { values: { id: val }, swap: 'none' });
  ```
- **Conditional `hx-swap-oob`**: If a template component is rendered both during full/SPA page loads and inline filter swaps, render `hx-swap-oob="true"` only when `r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content"`.

---

## 6. Critical Scripting Rules in HTMX Partials

### Context: Never Use Top-Level `let` or `const` in Swapped Templates
- **Problem**: HTMX executes `<script>` blocks in swapped partials within the global browser scope. Top-level `let` or `const` throws `Uncaught SyntaxError: Identifier '...' has already been declared` on the second navigation, crashing HTMX DOM insertion.
- **Enforced Solution**: Attach all state, timers, and functions to `window.*` (e.g., `window._calendarDragActive = false;`) or wrap in an IIFE `(function(){ ... })()`. Keep shared dropdown/input helpers centrally in `templates/base.html`.

### Context: Programmatic Hidden Input & Date Preset Events
- **Hidden Inputs**: When updating `<input type="hidden">` carrying `hx-trigger="change"`, fire both `hidden.dispatchEvent(new Event('change', { bubbles: true }))` and `if (window.htmx) htmx.trigger(hidden, 'change')`.
- **Coordinated Date Range Presets**: Use document-level delegation and a guard flag (`isSettingDatePreset = true`) when populating `start_date` and `end_date` programmatically, dispatching a single `change` event on `end_date` to avoid circular triggers.

### Context: Debounced Server-Side Item & Customer Search
- **Endpoints**:
  - `GET /items/search?q=&limit=` (`SearchItemsHandler`): `LIKE` on code/description/model/brand/category, default `20`, cap `50`.
  - `GET /customers/search?q=&limit=` (`SearchCustomersHandler`): `LIKE` on code/name/tin_no/contact_person, default `20`, cap `100`.
- **Initial Row Render**: Row/form handlers load only `LIMIT 20` items; `*RowDetailsHandler` uses `ORDER BY (id = ?) DESC, description ASC LIMIT 20` so the selected item is guaranteed to be included.

### Context: Calendar Drag & Drop Rescheduling via HTMX
- Guard click handlers with `_calendarDragActive` (cleared 120ms after `dragend`) so dropping a chip does not open the edit modal. On drop to a new date, call `htmx.ajax('POST', '/reminders/reschedule/' + id, { target: '#main-content', values: { due_date: targetDate } })`.
