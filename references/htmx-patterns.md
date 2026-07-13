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

## Related
- [[handlers-overview]] — `Render` and `RenderPage` methods
- [[templates-overview]] — Template structure
- [[sidebar-navigation]] — Navigation setup
- [[ui-design-tokens]] — CSS classes referenced in templates
