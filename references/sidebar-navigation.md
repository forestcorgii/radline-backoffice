# Sidebar Navigation & Quick Actions

> Back to [[00-index]] · Related: [[ui-design-tokens]], [[responsive-design]], [[htmx-patterns]], [[templates-overview]]

## File
[templates/base.html](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/templates/base.html) · Styles in [static/index.css](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/static/index.css)

## Current Sidebar Architecture (`254px` Desktop / `64px–68px` Collapsed)
1. **Brand Header (`.sidebar-brand`)**:
   - `.brand-full` (`R` badge + "RAV Enterprise Suite") and `.brand-mini` (collapsed mode).
   - `#sidebar-collapse-btn` (`toggleSidebarCollapse()`) and mobile `.sidebar-close-btn` (`toggleMobileSidebar()`).
   - Immediate inline `<script>` after `<aside id="sidebar">` restores `localStorage.getItem("sidebar-collapsed") === "true"` before first paint (zero FOUT).
2. **Signature `+ New` Quick Action Button (`.quick-create-section`)**:
   - `#quick-create-btn` toggles `#quick-create-menu` anchored directly below (`top: calc(100% + 4px)`).
   - **Context-Aware Menu (`updateDynamicQuickMenu()`)**:
     - **Default (`/`)**: 4-column matrix (Sales & Billing, Stock & Warehousing, Masterlist & Items, Operations & Tools).
     - **Module-Specific Routes**: Filtered single-column quick actions matching the active section (`/inventory*`, `/items|/brands|/categories|/uoms|/customers`, `/sales*`, `/goals*`, `/calendar*`, `/settings/users|/settings/roles`).
   - **Zero-Latency Dispatch (`handleQuickAction`)**: Toggles target form cards/modals directly in-DOM if already on the page, or navigates via `htmx.ajax` (`?action=new` for `/calendar`).
3. **Navigation Links (`nav.sidebar-nav` with `hx-target="#main-content" hx-push-url="true"`)**:
   - **Direct Links**: Dashboard (`/`), Sales (`/sales`), Earning Goals (`/goals`), Calendar (`/calendar`), Import (`/import`).
   - **`#forms-dropdown`**: Quotation Forms (`/forms/quotation`), Purchase Order Forms (`/forms/purchase-order`).
   - **`#inventory-dropdown`**: Overview (`/inventory`), Monthly Inventory (`/inventory/monthly`), Stock Receiving (`/inventory/receiving`), Stock Adjustment (`/inventory/adjustments`), Receiving Logs (`/inventory/receiving/logs`), Adjustment Logs (`/inventory/adjustments/logs`).
   - **`#masterlist-dropdown`**: Items (`/items`), Customers (`/customers`), Brands (`/brands`), Categories (`/categories`), Units of Measure (`/uoms`).
   - **`#tools-dropdown`**: Receipt Scanner (`/tools/receipt-scanner`).
4. **Bottom Profile Flyout (`#user-chip-container` → `#profile-menu`)**:
   - Loaded via `GET /auth/user-chip` (`UserChipHandler` in `handlers/auth.go`).
   - Contains user avatar/role badge and **Administration** links:
     - **User Accounts** (`/settings/users`)
     - **Roles & Permissions** (`/settings/roles`)
     - **Activity Logs** (`/activity-logs`)
     - **Settings (DeepSeek Vision AI)** (`/settings/system`)
     - **Sign Out** (`hx-post="/logout"`)

---

## Key Behaviors & Learnings

### Context: Collapsed Sidebar Mode (`.sidebar.collapsed`)
- When collapsed, `.sidebar-dropdown-trigger` headers are hidden (`display: none !important`) and all `.sidebar-dropdown-menu` items display inline as a continuous vertical column of centered SVG icons with `title` tooltips.
- `#profile-menu` flies out to the right (`left: calc(100% + 8px)`) when collapsed, and floats above the chip (`bottom: calc(100% + 6px)`) when expanded.

### Context: Two-Pass Active Link Highlighting (`updateActiveNavLink`)
- Runs on `DOMContentLoaded`, `htmx:afterOnLoad`, and `htmx:historyRestore`.
- Uses a two-pass longest-prefix match so `/inventory/monthly` activates only "Monthly Inventory" and not `/inventory` ("Overview") simultaneously.
- Active items use indigo text (`var(--primary-color)`), light indigo tint background, and a 4px left accent pill (`.active::before`).

### Context: Z-Index Hierarchy & Dropdown Cleanup
- `.sidebar` uses `z-index: 10000` on desktop and `100000` on mobile (backdrop `99999`).
- Never apply high `z-index` overrides (`9000+`) to `.card`, `form`, or `tr` elements; floating `.dropdown-list` popovers use `position: fixed; z-index: 999999` and `cleanupDropdownState(container)` strips active classes on selection/blur/escape.
