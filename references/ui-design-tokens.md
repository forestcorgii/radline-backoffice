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

### Context: Sales Page Encode Button & Filter Bar Customize View Icon Button
**Problem**: The Sales page had the "+ Encode Sales" button in a multi-button group next to the full-text "Customize View" button in the header bar, differing from the header layout in Brands/Categories and taking up extra header space.
**Enforced Solution**:
- **Header Alignment**: Placed `+ Encode Sales` directly in the header bar (matching Brands/Categories pages) which toggles an inline, collapsible form card (`#encode-sales-card`) directly above the filters.
- **Filter-Bar Icon Buttons**: Removed textual label from the "Customize View" button, converting it into a compact icon-only button (`<svg>` polygon filter icon) situated at the far right end of `.search-filter-bar`.

### Context: Universal Flexbox Cross-Browser Form Layout Compatibility (Edge & Chrome)
**Problem**: Microsoft Edge (or Edge in IE compatibility mode) can fail to resolve CSS Grid `grid-template-columns` inside dynamic HTMX swapped wrappers, causing form items and buttons to collapse or stack vertically.
**Enforced Solution**:
- **Universal Flexbox Horizontal Rows**: Use `display: flex; flex-direction: row; flex-wrap: nowrap;` on `.item-grid-header` and `.item-grid-row` instead of CSS Grid.
- **nth-child Flex Ratios**: Apply explicit `flex: ratio 1 0px; min-width: ...` to header labels and `.item-grid-cell:nth-child(n)` cells to guarantee 100% pixel-perfect column alignment across Edge, Chrome, Safari, and Firefox.
- **Cache Busting & Meta Compatibility**: Include `<meta http-equiv="X-UA-Compatible" content="IE=edge">` in [base.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/base.html) and increment `index.css?v=2.0` query version to force Edge to reload fresh stylesheets.

### Context: Standardizing Radline BackOffice to shadcn/ui Design Tokens & Components
**Problem**: Transforming existing custom styles in `static/index.css` to match the official **shadcn/ui** design language while preserving HTMX and Go HTML template compatibility.
**Enforced Solution**:
- **shadcn HSL Design System Tokens**: Configured standard HSL CSS variables (`--background`, `--foreground`, `--card`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--border`, `--input`, `--ring`, `--radius`) in `:root` and mapped legacy application CSS variables to them.
- **shadcn Primitive Component Specifications**:
  - **Cards (`.card`)**: High-contrast white/slate card with subtle border `1px solid hsl(var(--border))` and `shadow-sm`.
  - **Inputs & Selects (`input`, `select`, `.form-input`)**: Height `2.25rem`, radius `0.5rem`, focus ring `ring-2 ring-ring`.
  - **Buttons (`.btn`, `.btn-secondary`, `.btn-outline`, `.btn-ghost`, `.btn-danger`, `.btn-success`)**: Standard `2.25rem` height, rounded-md corners, high contrast slate dark primary button fill and clear focus rings.
  - **Badges (`.badge`, `.badge-secondary`, `.badge-success`, `.badge-danger`, `.badge-warning`)**: Rounded pill style badges with soft tint background fills and crisp text contrast.
  - **Tables**: Muted uppercase headers, subtle borders, row hover highlights.

### Context: Universal shadcn/ui Alert Dialog for HTMX Confirmations and JS Alerts
**Problem**: Native browser `window.confirm()` popups and `window.alert()` dialogs broke the visual aesthetics of the application and provided inconsistent user experience across different browsers.
**Enforced Solution**:
- **shadcn Alert Dialog Markup & Styling**: Created `.alert-dialog-overlay`, `.alert-dialog-content`, `.alert-dialog-header`, `.alert-dialog-title`, `.alert-dialog-description`, `.alert-dialog-footer` in [static/index.css](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/static/index.css) and added `#shadcn-alert-dialog` modal markup to [templates/base.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/base.html).
- **HTMX Event Interception (`htmx:confirm`)**: Attached a global listener for `htmx:confirm` that prevents the default browser popup and presents the shadcn Alert Dialog with smooth backdrop blur and scale-in animation. Clicking "Continue" calls `evt.detail.issueRequest(true)` to proceed.
- **Window Alert & Confirm Override**: Overrode `window.alert` and `window.confirm` to route through `window.showShadcnAlertDialog()` for consistent UI across the entire application.

### Context: Compact UI Spacing and High Information Density Optimization
**Problem**: UI spacing across form inputs, tables, cards, sidebar, buttons, and section wrappers was excessively tall and sparse, resulting in low information density and unnecessary vertical scrolling.
**Enforced Solution**:
- **Typography & Base Sizing**: Scaled body text to `0.8125rem` (13px) and `line-height: 1.4` for optimal text density.
- **Card & Layout Padding**: Reduced `.card` and `main#main-content` padding to `0.75rem 1rem` (mobile `.card` `0.5rem 0.75rem`).
- **Form Controls & Inputs**: Compacted `.form-group` vertical margin to `0.5rem`, `label` font-size to `0.75rem` with `0.2rem` margin-bottom, and input/select height to `1.875rem` (30px) with `0.25rem 0.5rem` padding.
- **Buttons & Icons**: Reduced primary `.btn` height to `1.875rem` (30px) with `0 0.65rem` padding and `.btn-icon` dimensions to `1.75rem x 1.75rem`.
- **Tables & Rows**: Reduced `th, td` cell padding to `0.3rem 0.5rem` with `font-size: 0.8125rem` (`0.75rem` uppercase headers) and table container top margin to `0.5rem`.
- **Item Entry Grid Cards**: Compacted `.item-card-row` / `.item-grid-row` padding to `0.4rem 0.65rem` with `0.35rem` grid gaps and `0.6875rem` uppercase field labels.

### Context: Stacking Deductions Columns in Sales Log View
**Problem**: Patong, POS Charge, and WT 2307 occupied three separate columns in the sales log view, consuming excessive horizontal screen space.
**Enforced Solution**:
- **Consolidation**: Replaced the three separate columns in the list view with a single combined column.
- **Naming**: The column header is named `Deductions` with a tooltip (`title`) set to `"Patong / POS / WT 2307"`.
- **View Configuration & Customization**: The customize view popover contains a single `Deductions` checkbox mapped to the `patong` column visibility key, and the other two keys (`pos_charge`, `wt_2307`) are removed from customization.
- **Stacked Layout**: 
  - Read-only rows display a vertical stack with labeled values: `P: ₱X.XX`, `C: ₱Y.YY`, `W: ₱Z.ZZ`.
  - Batch edit rows display a vertical stack of three compact inputs with `name="patong"`, `name="pos_charge"`, and `name="wt_2307"`, maintaining separate field updates for form submission.





