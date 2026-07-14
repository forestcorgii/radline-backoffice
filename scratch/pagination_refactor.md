# Pagination Refactor Plan

## Problems
1. **Boilerplate duplication**: Every handler manually builds `Pagination`, counts, queries, etc.
2. **No page size choice**: Hardcoded to 100. Users may want 25, 50, 100, 200.
3. **Poor navigation**: Only 5 page buttons - no First/Last, no ellipsis for large page counts.
4. **Missing hx-include on pagination buttons**: The `Include` field is set but never used in template.
5. **Fragile OOB swap**: Relies on `TargetID` in a `<div>` with `hx-swap-oob="true"`, which causes issues when HTMX wraps table fragments.
6. **Redundant count query**: Each handler does its own `SELECT COUNT(*)` wrapper.

## New Design

### Go: `handlers/pagination.go`
- `PaginationParams` - parsed from request (page, pageSize)
- `PaginatedResult` - holds results + pagination state
- `BuildPaginationView()` - unified builder
- `GetPaginationParams(r)` - extracts page + pageSize from query
- Smart window with ellipsis (shows: `1 ... 5 6 7 8 9 ... 20`)
- Page size selector (25, 50, 100, 200)

### Template: `templates/pagination.html`
- Cleaner responsive layout with page size dropdown
- First/Prev/Page Numbers/Ellipsis/Next/Last buttons
- Proper hx-include to preserve search/filter state
- Uses data attributes instead of fragile OOB swap
- Works both inside and outside forms

### CSS: `static/index.css`
- Compact mobile-first pagination
- Page size dropdown styling
- Active page emphasis
