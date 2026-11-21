---
name: radline-ui-htmx
description: >-
  Workflow and guardrails for building or updating HTML templates, HTMX
  interactions, list views, forms, and CSS in Radline BackOffice. Activate when
  editing files in templates/, static/, or UI-rendering handlers.
---

# Radline UI & HTMX Workflow

## 1. Read Only What You Need
- **Tokens & CSS classes**: [`references/ui-design-tokens.md`](../../../references/ui-design-tokens.md)
- **HTMX attributes, scripts & headers**: [`references/htmx-patterns.md`](../../../references/htmx-patterns.md)
- **Template registration in `main.go`**: [`references/templates-overview.md`](../../../references/templates-overview.md)

## 2. Mandatory Guardrails
1. **No Bare `let`/`const` in Swapped Templates**: Any `<script>` inside a page/partial template must attach state and functions to `window.*` or wrap in an IIFE `(function(){ ... })()` so repeated HTMX swaps never throw `Identifier has already been declared`.
2. **Explicit Navigation Targets**: Any link/form inside `<main id="main-content">` that navigates pages must set `hx-target="#main-content" hx-push-url="true"`.
3. **3-Way Handler Rendering**:
   - Full page / SPA nav (`HX-Target: main-content` or `#main-content`): `app.RenderPage(w, r, "page.html", data)`
   - Inline table/filter swap (`HX-Request == "true"` and target is not `main-content`): `app.Render(w, "results_or_rows.html", data)`
4. **Template Registration**: Every new fragment invoked via `{{ template "..." }}` or `app.Render` must be added to both the parent page's `files` slice and the standalone `fragments` slice in `parseTemplates()` ([`main.go`](../../../main.go)).
5. **Fixed Viewport Popovers**: Dropdown lists (`.dropdown-list`) and row status popovers (`.status-menu-popover`) inside tables must use `position: fixed; z-index: 999999` positioned via `getBoundingClientRect()` to escape table/card clipping.
6. **Cache Busting**: When modifying [`static/index.css`](../../../static/index.css), [`static/js/base-ui.js`](../../../static/js/base-ui.js), or [`static/js/table-engine.js`](../../../static/js/table-engine.js), bump the `?v=` query parameter in [`templates/base.html`](../../../templates/base.html).

