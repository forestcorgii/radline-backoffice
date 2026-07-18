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

## Learnings

### Context: Frontend UI/UX, CSS Design System, and HTMX Toast Notifications
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

### Context: Scalable UI/UX, Large Datasets, and Autocomplete Search
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

### Context: UI Layout Compactness and Information Density Optimization
**Problem**: The BackOffice UI spacing was too sparse, limiting the amount of details (table rows, form controls, and layout cards) visible on screen at once, requiring excessive scrolling.
**Enforced Solution**:
- **Global Sizing**: Enforce a global `font-size: 0.875rem` on the HTML `body` selector to scale all standard elements proportionally.
- **Sidebar & Viewport**: Reduce the default sidebar width to `220px` (from `260px`) and expand `main#main-content` max-width to `1440px` (from `1200px`) to maximize screen estate utilization on wide monitors.
- **Spacing Reduction**: Shrink card padding to `1rem`, table cell padding to `0.5rem 0.75rem`, form input paddings to `0.45rem 0.75rem`, button paddings to `0.45rem 1rem`, form group vertical spacing to `0.75rem`, grid gap to `1rem`, and sub-tab margin/paddings.

### Context: Stitch AI Premium Design System Integration
**Problem**: Modifying UI colors and themes in static CSS files without a unified, professional design scheme can lead to inconsistent accents, high-contrast borders, and off-brand visual layout fragments.
**Enforced Solution**:
- **Design System Generation**: Created a project in Stitch and generated a Material Design Fidelity theme seed using royal indigo (`#4f46e5`).
- **Core Tokens**: Applied the generated color tokens to `:root` CSS custom properties:
  - Background: Soft lavender warm off-white (`#fcf8ff`)
  - Accent brand: Royal Indigo (`#3525cd` / `#4f46e5`)
  - Card hover scale: Responsive transform lift with an indigo-tinted shadow glow.
- **Badge and Semantic Utilities**: Standardized all status elements to use semantic utility classes (`.badge-primary`, `.badge-secondary`, `.text-success`, `.text-danger`) rather than hardcoded inline styles.
- **Refactoring Chart Fills**: Chart colors (e.g. SVG rect elements) and financial summary card borders are dynamically styled using the new CSS variables to preserve uniform brand colors.

### Context: Action Column Buttons to Icon-Based Buttons
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

## Related
- [[responsive-design]] — Breakpoints and mobile overrides
- [[sidebar-navigation]] — Sidebar-specific CSS
- [[htmx-patterns]] — Toast notification styling
- [[templates-overview]] — HTML templates using these classes
