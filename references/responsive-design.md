# Responsive Design

> Back to [[00-index]] · Related: [[ui-design-tokens]], [[sidebar-navigation]]

## Breakpoints

| Breakpoint | Width | Purpose |
|---|---|---|
| Small | `640px` | Grid columns (2-col), header bar stacking |
| Medium | `768px` | Sidebar toggle, search bar layout, grid (3-col for 5-col grids) |
| Large | `1024px` | Full grid columns (3, 4, 5-col) |
| Extra small | `480px` | Toast positioning |

## Mobile-First Approach

The CSS uses a **mobile-first** strategy for grids:
```css
.grid-cols-3 { grid-template-columns: 1fr; }              /* Mobile: 1 col */

@media (min-width: 640px) {
    .grid-cols-3 { grid-template-columns: repeat(2, 1fr); }  /* Tablet: 2 col */
}
@media (min-width: 1024px) {
    .grid-cols-3 { grid-template-columns: repeat(3, 1fr); }  /* Desktop: 3 col */
}
```

## Table Overflow Strategy

Wide tables (receiving, sales, monthly inventory) use horizontal scroll on mobile:

```html
<div class="table-container">
    <table class="table-scroll-lg">   <!-- min-width: 1000px -->
        ...
    </table>
</div>
```

- `.table-container` — `overflow-x: auto; -webkit-overflow-scrolling: touch`
- `.table-scroll-md` — `min-width: 800px` (for simpler tables)
- `.table-scroll-lg` — `min-width: 1000px` (for complex tables)
- Custom scrollbar styling (WebKit): 6px height, subtle track, darker thumb on hover

## Sidebar Mobile Override

At `≤ 768px`:
```css
.sidebar { position: fixed; transform: translateX(-100%); }
.sidebar.open { transform: translateX(0); }
.top-header { display: flex; }        /* Shows hamburger + title */
main#main-content { padding: 1rem; }  /* Reduced from 2rem */
.card { padding: 1rem; border-radius: 0.75rem; }
```

See [[sidebar-navigation]] for full mobile behavior.

## Search/Filter Bar

At `≤ 768px`:
```css
.search-filter-bar { flex-direction: column; }  /* Stacks vertically */
.filter-item { width: 100%; }                   /* Full width */
```

At `≥ 768px`:
```css
.search-filter-bar { flex-direction: row; }     /* Horizontal */
.filter-item { width: 12rem; }                  /* Fixed width */
```

## Header Bar

At `≤ 640px`:
```css
.header-bar { flex-direction: column; text-align: center; }
```

## Toast Positioning

At `≤ 480px`:
```css
.toast-container { left: 1rem; right: 1rem; }  /* Full width */
.toast { width: 100%; }
```

## Learnings

### Context: Mobile-Responsive Spreadsheet Tables and Form Elements
**Problem**: Tabular forms and logs containing inputs, selects, and many data cells compress to unusable widths or overflow card blocks on mobile viewports.
**Enforced Solution**:
- Wrap all wide tables in a layout container with horizontal overflow support: `.table-container { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }`.
- Style scrollbars with subtle, custom styled WebKit tracks to keep the UX sleek and premium.
- Apply min-width constraints (e.g. `.table-scroll-md` at 800px, `.table-scroll-lg` at 1000px) directly to the table element to prevent input squishing and preserve tabular layout integrity on mobile devices.
- Replace hardcoded filter widths with responsive `.filter-item` wrappers, and grid layouts with responsive media-query classes (e.g. `.grid-cols-2`, `.grid-cols-3`, `.grid-cols-5`) to stack vertically on mobile and span horizontally on desktop.

### Context: Enforcing Table Cell Text Wrap Prevention with Responsive Scrolling
**Problem**: Text columns and data cells inside table list views (such as items description, categories, supplier names) wrap into multiple lines when the table gets squeezed or on smaller viewports, making the rows overly tall and layout cluttered.
**Enforced Solution**:
- **Table Container Overflow**: Set `.table-container` to `overflow-x: auto; -webkit-overflow-scrolling: touch;` to enable horizontal scrolling when tables exceed container width, matching mobile-responsive standards.
- **Prevent Cell Wrap**: Add `white-space: nowrap;` to the global `th` and `td` stylesheet definitions. This forces columns to expand dynamically to fit their content perfectly without line breaks.

### Context: Card, Form, and Table Stacking Context for Searchable Dropdowns
**Problem**: Cards with `backdrop-filter: blur()` create root WebKit/Blink stacking contexts that clip child absolute elements. In forms across Sales, Receive Stock, and Stock Adjustments, form action buttons and card footers rendered after `.table-container` in DOM order were painted on top of `.dropdown-list`.
**Enforced Solution**:
- **Fixed Viewport Popover Positioning (`updateDropdownPosition`)**: Set `.dropdown-list` to `position: fixed` dynamically calculated via `input.getBoundingClientRect()`. This completely breaks out of table containers, scroll boxes, backdrop filters, forms, and cards, floating the dropdown popover on the top-most viewport layer (`z-index: 999999`) above all elements. Auto-flips above the input if space below is constrained.
- **DOM Rendering Performance Cap**: Limited `filterDropdown()` to display at most 50 matching items at a time (`visibleCount < 50`) to eliminate DOM reflow lag when iterating thousands of items.
- **Input Text Preservation**: Preserved typed and selected item text in `hideDropdown()` without wiping `input.value` to blank.

## Related
- [[ui-design-tokens]] — Full CSS reference
- [[sidebar-navigation]] — Mobile sidebar behavior
- [[templates-overview]] — Template structure for responsive layout
