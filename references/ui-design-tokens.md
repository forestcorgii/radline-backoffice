# UI Design Tokens

> Back to [[00-index]] · Related: [[templates-overview]], [[responsive-design]]

## File
[static/index.css](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/static/index.css)

## CSS Custom Properties (Design Tokens)

```css
:root {
    --bg-color: #faf9f6;           /* Page background (warm off-white) */
    --surface-color: #ffffff;      /* Card/panel backgrounds */
    --primary-color: #2563eb;      /* Blue — buttons, active states */
    --primary-hover: #1d4ed8;      /* Darker blue — button hover */
    --text-main: #1c1917;          /* Primary text (near-black warm) */
    --text-muted: #78716c;         /* Secondary text, labels */
    --border-color: #e7e5e4;       /* Borders, dividers */
    --glass-bg: rgba(255,255,255,0.8);     /* Glassmorphism background */
    --glass-border: rgba(28,25,23,0.08);   /* Glassmorphism border */
    --shadow: 0 4px 6px -1px rgba(0,0,0,0.03), 0 2px 4px -2px rgba(0,0,0,0.03);
}
```

**Typography:** `'Inter', system-ui, -apple-system, sans-serif` (Google Fonts CDN in `base.html`)

## Component Classes

### Cards
```css
.card                    /* Glassmorphism container — blur, border, shadow, hover lift */
```

### Buttons
```css
.btn                     /* Primary blue button */
.btn-secondary           /* Gray (#78716c) */
.btn-success             /* Green (#10b981) */
.btn-danger              /* Red (#ef4444) */
```

### Forms
```css
.form-group              /* Wrapper with margin-bottom */
.form-input              /* Compact input for inline editing (0.5rem padding) */
input[type="text/number/date"]  /* Full-width inputs (0.75rem padding) */
```

### Tables
```css
.table-container         /* Overflow-x: auto wrapper for mobile */
.table-scroll-md         /* min-width: 800px on the table */
.table-scroll-lg         /* min-width: 1000px on the table */
.edit-row td             /* Reduced padding for inline edit rows */
```

### Search & Filters
```css
.search-filter-bar       /* Flex container — column on mobile, row on desktop */
.filter-item             /* Individual filter wrapper */
select.form-input        /* Custom arrow dropdown styling */
```

### Badges
```css
.badge                   /* Pill-shaped inline label */
.badge-active            /* Green — in use */
.badge-inactive          /* Red — unused */
```

### Sub-tabs
```css
.sub-tabs                /* Horizontal tab bar with bottom border */
.sub-tab-item            /* Individual tab */
.sub-tab-item.active     /* Active tab — blue accent */
```

### Grid Utilities
```css
.grid                    /* CSS Grid with 1.5rem gap */
.grid-cols-2             /* 1 col → 2 col @ 640px */
.grid-cols-3             /* 1 → 2 @ 640px → 3 @ 1024px */
.grid-cols-4             /* 1 → 2 @ 640px → 4 @ 1024px */
.grid-cols-5             /* 1 → 2 @ 640px → 3 @ 768px → 5 @ 1024px */
.grid-adj-header         /* 1 col → 1fr 2fr @ 640px */
```

### Spacing
```css
.mt-4                    /* margin-top: 1rem */
.mb-4                    /* margin-bottom: 1rem */
```

### Animations
```css
#main-content > *        /* fadeIn 0.22s — smooth page transition */
.toast                   /* slideIn 0.3s — toast entry */
@keyframes fadeOut       /* toast exit animation */
```

## Toast Styling

```css
.toast                   /* Glassmorphic notification */
.toast-success           /* Left border: green #10b981 */
.toast-error             /* Left border: red #ef4444 */
.toast-warning           /* Left border: yellow #f59e0b */
.toast-info              /* Left border: blue #3b82f6 */
```

See [[htmx-patterns]] for the toast protocol.

## Design Principles

1. **Glassmorphism** — Cards use `backdrop-filter: blur(12px)` with semi-transparent backgrounds
2. **Warm neutrals** — `#faf9f6` background, `#1c1917` text (warm-toned, not pure black/white)
3. **Subtle interactions** — Cards lift 2px on hover, buttons scale on click
4. **Page transitions** — Content fades in with subtle translateY for HTMX swaps

## Rules for Templates
- **Never hardcode colors** — Always use `var(--variable-name)`
- **Use existing classes** — Check this list before creating new CSS
- **Font size convention** — Body text: 0.875rem, Labels: 0.875rem, Table headers: uppercase 0.875rem

## Related
- [[responsive-design]] — Breakpoints and mobile overrides
- [[sidebar-navigation]] — Sidebar-specific CSS
- [[htmx-patterns]] — Toast notification styling
- [[templates-overview]] — HTML templates using these classes
