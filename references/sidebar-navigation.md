# Sidebar Navigation

> Back to [[00-index]] · Related: [[ui-design-tokens]], [[responsive-design]]

## File
[templates/base.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/base.html) — lines 15–77 (structure), 82–191 (scripts)

## Structure

```html
<div class="app-layout">
    <div id="sidebar-backdrop" onclick="toggleMobileSidebar()"></div>
    
    <aside id="sidebar" class="sidebar">
        <div class="sidebar-brand">
            <h1 class="brand-full">Radline BackOffice</h1>    <!-- Gradient text -->
            <h1 class="brand-mini">R</h1>
            <button id="sidebar-collapse-btn" class="sidebar-collapse-btn" onclick="toggleSidebarCollapse()">...</button>
            <button class="sidebar-close-btn" onclick="toggleMobileSidebar()">...</button>     <!-- Mobile only -->
        </div>
        
        <nav class="sidebar-nav" hx-target="#main-content" hx-push-url="true">
            <a href="/">Dashboard</a>
            <a href="/sales">Sales</a>
            
            <div class="sidebar-dropdown" id="inventory-dropdown">
                <button class="sidebar-dropdown-trigger">
                    Inventory ▼
                </button>
                <div class="sidebar-dropdown-menu">
                    <a href="/inventory">Overview</a>
                    <a href="/inventory/monthly">Monthly Inventory</a>
                    <a href="/inventory/receiving">Stock Receiving</a>
                    <a href="/inventory/adjustments">Stock Adjustment</a>
                    <a href="/inventory/receiving/logs">Receiving Logs</a>
                    <a href="/inventory/adjustments/logs">Adjustment Logs</a>
                </div>
            </div>

            <div class="sidebar-dropdown" id="masterlist-dropdown">
                <button class="sidebar-dropdown-trigger">
                    Masterlist ▼
                </button>
                <div class="sidebar-dropdown-menu">
                    <a href="/items">Items</a>
                    <a href="/settings">Settings</a>
                </div>
            </div>

            <a href="/import">Import</a>

            <div class="sidebar-dropdown" id="tools-dropdown">
                <button class="sidebar-dropdown-trigger">
                    Tools ▼
                </button>
                <div class="sidebar-dropdown-menu">
                    <a href="/tools/receipt-scanner">Receipt Scanner</a>
                </div>
            </div>
        </nav>
    </aside>

    <div class="main-layout-container">
        <header class="top-header">               <!-- Mobile only -->
            <button id="sidebar-toggle">☰</button>
            <div class="top-header-title">Radline BackOffice</div>
        </header>
        <main id="main-content">
            {{ block "content" . }}{{ end }}
        </main>
    </div>
</div>
```

## Desktop Behavior
- Sidebar is a fixed 220px column, sticky to viewport
- Can be collapsed to a 64px icon-only sidebar via the collapse button (stored in `localStorage`)
- Glassmorphic background with `backdrop-filter: blur(16px)`
- Active link gets blue/indigo background with shadow (active accent indicator bar on the left)
- Sidebar dropdowns toggle open/closed via `toggleSidebarDropdown()`
- Dropdown auto-opens when any sub-link URL is active

## Mobile Behavior (≤ 768px)
- Sidebar is hidden off-screen (`transform: translateX(-100%)`)
- Hamburger button in top header triggers `toggleMobileSidebar()`
- Backdrop overlay appears behind sidebar
- Close button (X) appears in sidebar brand area
- Navigation clicks auto-close sidebar via `closeMobileSidebarOnNav()`

## JavaScript Functions

### `updateActiveNavLink()`
- Runs on: `DOMContentLoaded`, `htmx:afterOnLoad`, `htmx:historyRestore`
- Highlights main links: exact match for `/`, `startsWith` for others
- Highlights dropdown sub-links: exact match only
- Auto-expands Inventory dropdown if any sub-link is active

### `toggleSidebarDropdown(event)`
- Toggles `.open` class on the parent `.sidebar-dropdown`
- Chevron rotates 180° when open

### `toggleMobileSidebar()`
- Toggles `.open` on sidebar and `.show` on backdrop

### `closeMobileSidebarOnNav()`
- Removes `.open` and `.show` — called on every nav link click

## CSS Details

See [[ui-design-tokens]] for full CSS. Key sidebar styles:

| Class | Desktop | Mobile |
|---|---|---|
| `.sidebar` | `position: sticky; width: 220px` | `position: fixed; transform: translateX(-100%)` |
| `.sidebar.collapsed` | `width: 64px` | — |
| `.sidebar.open` | — | `transform: translateX(0)` |
| `.sidebar-close-btn` | `display: none` | `display: flex` |
| `.top-header` | `display: none` | `display: flex` |
| `.sidebar-backdrop.show` | — | `display: block; opacity: 1` |
| `.sidebar-nav a.active` | Indigo text + light indigo bg + left accent | Same |
| `.sidebar-dropdown-menu a.active` | Indigo text + light indigo bg | Same |

## Learnings

### Context: Collapsible Navigation Sidebar with Hover Dropdown Popovers
**Problem**: Full-width sidebars take up significant horizontal viewport space on desktop, squishing complex grid modules, logs, and spreadsheet tables. Simply collapsing the sidebar off-screen makes navigation tedious.
**Enforced Solution**:
- **Icon-Only Collapse State**: Define a `.collapsed` CSS state that shrinks the sidebar from `220px` to `64px`, hides text labels (`.nav-text`), and centers the brand/links navigation icons.
- **Immediate Inline State Restoring**: Embed an IIFE/synchronous script immediately following the `<aside>` tag:
  ```javascript
  if (localStorage.getItem("sidebar-collapsed") === "true") {
      document.getElementById("sidebar").classList.add("collapsed");
  }
  ```
  This guarantees the layout state is parsed and applied before the first page render, completely preventing Flash of Uncollapsed Layout (FOUT).
- **Hover Dropdown Popovers**: When collapsed, override vertical inline menu lists to display absolutely as floating popover panels on the right:
  ```css
  .sidebar.collapsed .sidebar-dropdown:hover .sidebar-dropdown-menu {
      display: flex !important;
      position: absolute;
      left: 100%;
      top: 0;
      background: var(--surface-color);
      border: 1px solid var(--border-color);
      box-shadow: var(--shadow-hover);
      z-index: 1001;
      margin-left: 0.5rem;
      min-width: 180px;
  }
  ```
- **Click Behavior Suppression**: Block inline dropdown expand/collapse click handlers while the sidebar is collapsed to prevent UI interaction bugs.

### Context: Sidebar Active Link Contrast Fix
**Problem**: Active links in the sidebar had poor contrast — blue text (`var(--primary-color)`) on a light blue tinted background (`rgba(53, 37, 205, 0.08)`), making the active state hard to distinguish.
**Enforced Solution**:
- Changed both `.sidebar-nav a.active` and `.sidebar-dropdown-menu a.active` to use `color: var(--primary-color)` (indigo) on `background-color: rgba(53, 37, 205, 0.08)` (light indigo) — the indigo text on a light tint provides good contrast while keeping the color family cohesive.
- Added a left accent bar (`.active::before`) using `var(--primary-color)` as a 4px-wide pill indicator to visually anchor the active item.
- This approach gives a **clean, modern active indicator** with the accent bar providing the visual weight instead of a heavy gradient fill.

### Context: Sidebar Active Link Duplicate Highlighting
**Problem**: The `updateActiveNavLink()` JavaScript used `currentPath.startsWith(href)` to match dropdown sub-links, causing `/inventory` (Overview) and `/inventory/monthly` (Monthly Inventory) to both be highlighted simultaneously when on `/inventory/monthly`.
**Enforced Solution**:
- Changed dropdown sub-link matching to a two-pass algorithm:
  1. **First pass**: Mark all links that match via exact match OR `startsWith(href + "/")` (for child pages like `/items/new` matching `/items`).
  2. **Second pass**: If multiple links in the same dropdown are active, keep only the one with the **longest href** (most specific match) and deactivate the rest.
- This ensures only one sub-link is active at a time while still supporting parent-link highlighting for sub-pages (e.g., `/items/new` highlights `/items`).

### Context: Displaying All Menu and Sub-menu Logo Icons Altogether when Sidebar is Minimized
**Problem**: Previously, when the sidebar was collapsed (`.sidebar.collapsed`), sub-menu links were hidden inside popovers that relied on CSS hover, causing sub-menu icons to be hidden or hard to access when minimized.
**Enforced Solution**:
- **SVG Logos for Sub-menu Items**: Wrap all sub-menu links in `<span class="nav-icon-text">` containing distinct 18x18 SVG icons (`.nav-icon`) matching top-level menu icon style (2px stroke width, `currentColor`).
- **Inline Collapsed Navigation**: When `.sidebar.collapsed`, hide non-link category headers (`.sidebar-dropdown-trigger`) and display all `.sidebar-dropdown-menu` containers inline as a continuous vertical column:
  ```css
  .sidebar.collapsed .sidebar-dropdown-trigger {
      display: none !important;
  }
  .sidebar.collapsed .sidebar-dropdown-menu {
      display: flex !important;
      flex-direction: column;
      gap: 0.125rem;
      padding: 0 !important;
      margin: 0 !important;
      border: none !important;
  }
  .sidebar.collapsed .sidebar-nav a,
  .sidebar.collapsed .sidebar-dropdown-menu a {
      display: flex !important;
      justify-content: center !important;
      align-items: center;
      padding: 0.5rem !important;
  }
  ```
- All menu and sub-menu item icons (Dashboard, Sales, Overview, Monthly Inventory, Receiving Logs, Adjustment Logs, Items, Settings, Import, Receipt Scanner) now display centered altogether in the 64px minimized sidebar column with native tooltips (`title="..."`) on hover.


### Context: Sidebar Navigation Z-Index and Dropdown Cleanup Fix
**Problem**: Users could not click sidebar navigation links or the "+ Add Item Row" button. Page cards/forms containing `.searchable-dropdown` elements were applying `z-index: 9000 !important` and `z-index: 9500 !important` to parent `.card`, `form`, `.table-container`, and `tr` elements. This stacked page content 9x higher than `.sidebar` (`z-index: 1000`), causing content cards to intercept clicks intended for the sidebar. Additionally, selecting item dropdown options failed to clean up container `has-dropdown` states, leaving cards permanently elevated.
**Enforced Solution**:
- **Elevated Sidebar Z-Index**: Increased `.sidebar` `z-index` to `10000` on desktop and `100000` on mobile (and mobile backdrop to `99999`).
- **Purged Card/Form Z-Index Overrides**: Removed destructive `z-index: 9000 !important` and `z-index: 9500 !important` rules from `.card`, `form`, `.table-container`, and `tr` rules. Since `.dropdown-list` is `position: fixed; z-index: 999999 !important;`, parent cards and forms do not require high z-indexes.
- **Immediate Dropdown State Cleanup**: Added `cleanupDropdownState(container)` in `base.html` to instantly strip `active`, `has-dropdown`, `active-dropdown-row`, and `active-dropdown-cell` classes on item selection, blur, escape, or outside clicks.
- **Grid Row Container Focus**: Updated Enter keydown listener to target `#sales-items-tbody`, `#receiving-items-tbody`, `#adjustment-items-tbody` grid containers when auto-focusing newly added item rows.

## Related
- [[ui-design-tokens]] — CSS classes and variables
- [[responsive-design]] — Mobile breakpoints
- [[htmx-patterns]] — SPA navigation via HTMX
- [[templates-overview]] — Base template structure


