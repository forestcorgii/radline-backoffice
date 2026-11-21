---
name: radline-context-router
description: >-
  Fast reference router for the Radline BackOffice repository. Use this skill
  when you need to locate architecture rules, database schema, HTMX conventions,
  UI design tokens, routes, or domain invariants without reading multiple
  reference files.
---

# Radline Fast Context Router

Route directly to the target `references/<topic>.md` file matching your task (skip reading `references/00-index.md` unless you need the full map of contents):

| Task / Keyword | Target Reference File | What It Contains |
|---|---|---|
| **Routes, Auth, RBAC, PWA, Port `:8080`** | [`references/routing.md`](../../../references/routing.md) | Complete route table, `RequirePermission`, `AuthMiddleware`, CSRF protection, PWA bypass, port invariant |
| **CSS, Colors, Buttons, Cards, Table Columns, Filters** | [`references/ui-design-tokens.md`](../../../references/ui-design-tokens.md) | HSL `:root` tokens, `.card`/`.btn`/`.badge`, compact 12px density, Universal List Views Engine, popovers |
| **HTMX Swaps, Toasts, Dropdown Search, Partial Scripts** | [`references/htmx-patterns.md`](../../../references/htmx-patterns.md) | `HX-Trigger` toasts, `HX-Location`, `limit` filter auto-submit, `window.*` script scope, `/items/search` |
| **Sidebar, `+ New` Quick Create, Profile Menu** | [`references/sidebar-navigation.md`](../../../references/sidebar-navigation.md) | 254px/68px sidebar, context-aware `#quick-create-menu`, `#profile-menu`, z-index hierarchy |
| **Mobile Breakpoints, Table Scroll, Column Resize** | [`references/responsive-design.md`](../../../references/responsive-design.md) | `640px`/`768px`/`1024px` breakpoints, `.table-container`, fixed popovers, dynamic column widths |
| **SQLite + Postgres Dual DB, Tables, Indexes** | [`references/database-schema.md`](../../../references/database-schema.md) | All 21 tables, `DBConn` `?`→`$1` rebinding, `RETURNING id`, `substr()` polyfills, batch loading |
| **Startup Schema Migrations, `GROUP BY`, `COALESCE`** | [`references/database-migrations.md`](../../../references/database-migrations.md) | `runSQLiteMigrations()` + `runPostgresMigrations()` parity, strict Postgres `GROUP BY` & `TIMESTAMPTZ` |
| **Domain Models & Pure DDD Invariants** | [`references/domain-overview.md`](../../../references/domain-overview.md) | Entity/Aggregate map (`Item`, `Customer`, `ItemStock`, `StockReceive`, `Sale`, `StockAdjustment`, `PurchaseOrder`, `EarningGoal`, `Reminder`, `User`/`Role`) |
| **Domain Unit & Integration Testing** | [`references/domain-testing.md`](../../../references/domain-testing.md) | `domain/*_test.go` (`package domain_test`), `db/db_test.go`, `handlers/*_test.go`, `e2e/*_test.go` |
| **HTTP Handlers & `Render` vs `RenderPage`** | [`references/handlers-overview.md`](../../../references/handlers-overview.md) | 3-way HTMX rendering check, `App` struct, all 17 handler files |
| **Template Parsing (`parseTemplates` in `main.go`)** | [`references/templates-overview.md`](../../../references/templates-overview.md) | `pages` vs `fragments` registration in `main.go`, transitive fragment dependencies, listview map |

