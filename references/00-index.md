# 🗂️ Radline BackOffice — Knowledge Vault & Fast Retrieval Index

> **Module:** `rav` · **Port:** `:8080` (invariant — never change in `main.go`)  
> **Stack:** Go 1.26 · Dual SQLite (`modernc.org/sqlite`) & PostgreSQL (`pgx/v5`) · HTMX 1.9 (`/static/htmx.min.js`) · `html/template` · `sqlx`  
> **Architecture:** Server-Side Rendered Hypermedia (DDD domain-first TDD, vanilla CSS shadcn/QuickBooks Indigo tokens)

---

## ⚡ Fast Task-to-Reference Router (Read Only What You Need)

| If your task involves... | Read ONLY this file first | Key Invariants Summary |
|---|---|---|
| **UI, CSS, Colors, Buttons, Cards, Collapsible Columns** | [ui-design-tokens.md](ui-design-tokens.md) | 12px high-density scale, HSL `--primary` (`243 75% 59%`), `.card`/`.btn`/`.badge`, Universal List Views Engine, bump `index.css?v=` |
| **HTMX Swaps, Toasts, Filters, Dropdowns, `<script>` tags** | [htmx-patterns.md](htmx-patterns.md) | **Never** use top-level `let`/`const` in partials (use `window.*`); explicit `hx-target="#main-content"`; `limit` selector (no pagination); `HX-Trigger` `show-toast` |
| **Sidebar, `+ New` Quick Create, Profile Menu, Z-Index** | [sidebar-navigation.md](sidebar-navigation.md) | 254px/68px sidebar, context-aware `#quick-create-menu`, `#profile-menu` (Users, Roles, Logs, System Settings), `.sidebar` `z-index: 10000` |
| **Mobile Breakpoints, Table Overflow, Column Resizing** | [responsive-design.md](responsive-design.md) | `.table-container` horizontal scroll, `white-space: nowrap`, `position: fixed; z-index: 999999` popovers, `.table-col-resizer` (`6px`) |
| **Routes, Auth, Permissions (`RequirePermission`), PWA** | [routing.md](routing.md) | Complete route map, `rav_session` cookie, CSRF protection (`http.CrossOriginProtection`), 17 RBAC permission keys, `/sw.js` & `/manifest.webmanifest` auth bypass |
| **Database Tables, SQLite + Postgres Queries, Indexes** | [database-schema.md](database-schema.md) | 21 tables, `DBConn` auto-rebinds `?`→`$1`, use `RETURNING id`, `ON CONFLICT DO NOTHING`, `substr(date, 1, 7)` |
| **Adding Columns/Tables, Postgres 500s, `GROUP BY`** | [database-migrations.md](database-migrations.md) | Update **both** `runSQLiteMigrations()` & `runPostgresMigrations()`, `COALESCE` nullable cols, list all non-agg cols in `GROUP BY` |
| **DTO Structs & Nullable FKs (`models/`)** | [models-layer.md](models-layer.md) | `models/` uses pointer types (`*int`, `*int64`, `*string`, `*float64`) for nullable columns; `domain/` uses value types (`0` / `""` = unset) |
| **Domain Models & Business Rules (`domain/`)** | [domain-overview.md](domain-overview.md) | Pure Go (stdlib + `golang.org/x/crypto/bcrypt` in `auth.go`); validate in `New*()` / `Validate*()` constructors |
| **Testing (`go test ./...`)** | [domain-testing.md](domain-testing.md) | Black-box `domain/*_test.go` (`package domain_test`), `db/db_test.go`, `handlers/*_test.go`, `e2e/*_test.go` |
| **HTTP Handlers & `Render` vs `RenderPage`** | [handlers-overview.md](handlers-overview.md) | 3-way HTMX check (`HX-Request` + `HX-Target != "main-content"` → fragment, else `RenderPage`) across 17 handler files |
| **Templates & `parseTemplates()` in `main.go`** | [templates-overview.md](templates-overview.md) | Register transitive child templates in both `pages` and `fragments` slices in `main.go` |

---

## 📌 Complete Map of Contents

### Architecture & Core
- [[architecture]] ([architecture.md](architecture.md)) — System architecture, package layering rules, data flow
- [[tech-stack]] ([tech-stack.md](tech-stack.md)) — Go 1.26, `modernc.org/sqlite`, `pgx/v5`, `sqlx`, `excelize`, `bcrypt`, HTMX

### Domain Layer (`domain/`)
- [[domain-overview]] ([domain-overview.md](domain-overview.md)) — Complete entity & aggregate directory
- [[domain-item]] ([domain-item.md](domain-item.md)) — `Item`, `Brand`, `Category`, `UomSetting`
- [[domain-customer]] ([domain-customer.md](domain-customer.md)) — `Customer`, `User`/`Role`/`ActivityLog` (RBAC), `EarningGoal`, `Reminder`, `PurchaseOrder`
- [[domain-stock]] ([domain-stock.md](domain-stock.md)) — `ItemStock` aggregate root, UOM conversion & FIFO (`GetOldestPLWithStock`)
- [[domain-receiving]] ([domain-receiving.md](domain-receiving.md)) — `ReceivingLog` & `StockReceive` aggregate
- [[domain-sales]] ([domain-sales.md](domain-sales.md)) — `SalesDetail` & `Sale` aggregate (remittance, deductions, net profit margin)
- [[domain-adjustment]] ([domain-adjustment.md](domain-adjustment.md)) — `InventoryAdjustment` & `StockAdjustment` aggregate
- [[domain-testing]] ([domain-testing.md](domain-testing.md)) — Domain unit tests, DB/handler integration tests, and E2E test suite

### Persistence (`db/`, `models/`)
- [[database-schema]] ([database-schema.md](database-schema.md)) — Dual SQLite/PostgreSQL engine, all 21 tables, indexes, SQL portability rules
- [[database-migrations]] ([database-migrations.md](database-migrations.md)) — Startup migration parity (`runSQLiteMigrations` + `runPostgresMigrations`)
- [[models-layer]] ([models-layer.md](models-layer.md)) — DTO structs (`models/models.go`, `models/auth.go`, `models/goals_reminders.go`)

### HTTP & Templates (`handlers/`, `templates/`)
- [[routing]] ([routing.md](routing.md)) — Complete route table, CSRF protection, RBAC permissions, PWA endpoints
- [[handlers-overview]] ([handlers-overview.md](handlers-overview.md)) — Handler architecture, `App` struct, `Render` vs `RenderPage`, all 17 handler files
- [[handlers-master-data]] ([handlers-master-data.md](handlers-master-data.md)) — Brands, Categories, Items, `/items/search`, select fragments
- [[handlers-inventory]] ([handlers-inventory.md](handlers-inventory.md)) — Inventory overview, Monthly Inventory, Receiving, Adjustments
- [[handlers-sales]] ([handlers-sales.md](handlers-sales.md)) — Sales CRUD, batch edit, inline status popover
- [[templates-overview]] ([templates-overview.md](templates-overview.md)) — `parseTemplates()` hierarchy, fragments registration, "List Views" map
- [[htmx-patterns]] ([htmx-patterns.md](htmx-patterns.md)) — HTMX conventions, OOB swaps, toast protocol, `window.*` partial script rules

### Frontend (`static/`, `templates/base.html`)
- [[ui-design-tokens]] ([ui-design-tokens.md](ui-design-tokens.md)) — HSL CSS tokens, component classes, Universal List Views Engine, fixed popovers
- [[sidebar-navigation]] ([sidebar-navigation.md](sidebar-navigation.md)) — Sidebar layout, `+ New` Quick Actions menu, `#profile-menu`
- [[responsive-design]] ([responsive-design.md](responsive-design.md)) — Mobile breakpoints, table scroll, dynamic column resizing
