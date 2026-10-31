---
name: radline-context-router
description: >-
  Fast reference router for the Radline BackOffice repository. Use this skill
  when you need to locate architecture rules, database schema, HTMX conventions,
  UI design tokens, routes, or domain invariants without reading multiple
  reference files.
---

# Radline Fast Context Router

Start at [`references/00-index.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/00-index.md) and read **only** the specific reference file matching your task:

| Task / Keyword | Target Reference File | What It Contains |
|---|---|---|
| **Routes, Auth, RBAC, PWA, Port `:8080`** | [`references/routing.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/routing.md) | Complete route table, `RequirePermission`, `AuthMiddleware`, PWA bypass, port invariant |
| **CSS, Colors, Buttons, Cards, Table Columns, Filters** | [`references/ui-design-tokens.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/ui-design-tokens.md) | HSL `:root` tokens, `.card`/`.btn`/`.badge`, compact 12px density, Excel collapsible columns, popovers |
| **HTMX Swaps, Toasts, Dropdown Search, Partial Scripts** | [`references/htmx-patterns.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/htmx-patterns.md) | `HX-Trigger` toasts, `HX-Location`, `limit` filter auto-submit, `window.*` script scope, `/items/search` |
| **Sidebar, `+ New` Quick Create, Profile Menu** | [`references/sidebar-navigation.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/sidebar-navigation.md) | 254px/68px sidebar, context-aware `#quick-create-menu`, `#profile-menu`, z-index hierarchy |
| **Mobile Breakpoints, Table Scroll, Column Resize** | [`references/responsive-design.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/responsive-design.md) | `640px`/`768px`/`1024px` breakpoints, `.table-container`, fixed popovers, `localStorage` column widths |
| **SQLite + Postgres Dual DB, Tables, Indexes** | [`references/database-schema.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-schema.md) | All 17 tables, `DBConn` `?`→`$1` rebinding, `RETURNING id`, `substr()` polyfills, batch loading |
| **Startup Schema Migrations, `GROUP BY`, `COALESCE`** | [`references/database-migrations.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-migrations.md) | `runSQLiteMigrations()` + `runPostgresMigrations()` parity, strict Postgres `GROUP BY` & `TIMESTAMPTZ` |
| **Domain Models & Pure DDD Invariants** | [`references/domain-overview.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-overview.md) | Entity/Aggregate map (`Item`, `Customer`, `ItemStock`, `StockReceive`, `Sale`, `StockAdjustment`, `PurchaseOrder`, `EarningGoal`, `Reminder`, `User`/`Role`) |
| **Domain Unit Testing (`go test ./domain/`)** | [`references/domain-testing.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-testing.md) | Black-box `package domain_test` table-driven test conventions |
| **HTTP Handlers & `Render` vs `RenderPage`** | [`references/handlers-overview.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/handlers-overview.md) | 3-way HTMX rendering check, `App` struct, handler file map |
| **Template Parsing (`parseTemplates` in `main.go`)** | [`references/templates-overview.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/templates-overview.md) | `pages` vs `fragments` registration in `main.go`, transitive fragment dependencies, listview map |

