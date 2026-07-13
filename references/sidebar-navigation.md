# Sidebar Navigation

> Back to [[00-index]] · Related: [[ui-design-tokens]], [[responsive-design]]

## File
[templates/base.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/base.html) — lines 15–77 (structure), 82–191 (scripts)

## Structure

```html
<div class="app-layout">
    <div id="sidebar-backdrop" onclick="toggleMobileSidebar()">
    
    <aside id="sidebar" class="sidebar">
        <div class="sidebar-brand">
            <h1>Radline BackOffice</h1>            <!-- Gradient text -->
            <button class="sidebar-close-btn">     <!-- Mobile only -->
        </div>
        
        <nav class="sidebar-nav" hx-target="#main-content" hx-push-url="true">
            <a href="/">Dashboard</a>
            <a href="/entry">Data Entry</a>
            <a href="/brands">Brands</a>
            <a href="/categories">Categories</a>
            <a href="/items">Items</a>
            
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
            
            <a href="/sales">Sales</a>
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
- Sidebar is a fixed 260px column, sticky to viewport
- Glassmorphic background with `backdrop-filter: blur(16px)`
- Active link gets blue background with shadow
- Inventory dropdown toggles open/closed via `toggleSidebarDropdown()`
- Dropdown auto-opens when any `/inventory/*` URL is active

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
| `.sidebar` | `position: sticky; width: 260px` | `position: fixed; transform: translateX(-100%)` |
| `.sidebar.open` | — | `transform: translateX(0)` |
| `.sidebar-close-btn` | `display: none` | `display: flex` |
| `.top-header` | `display: none` | `display: flex` |
| `.sidebar-backdrop.show` | — | `display: block; opacity: 1` |
| `.sidebar-nav a.active` | Blue bg + white text | Same |
| `.sidebar-dropdown-menu a.active` | Blue text + light blue bg | Same |

## Related
- [[ui-design-tokens]] — CSS classes and variables
- [[responsive-design]] — Mobile breakpoints
- [[htmx-patterns]] — SPA navigation via HTMX
- [[templates-overview]] — Base template structure
