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

## Related
- [[ui-design-tokens]] — Full CSS reference
- [[sidebar-navigation]] — Mobile sidebar behavior
- [[templates-overview]] — Template structure for responsive layout
