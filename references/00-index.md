# 🗂️ RAV BackOffice — Knowledge Vault & Fast Retrieval Index

> **Module:** `rav` · **Port:** `:8080` (invariant — never change in `main.go`)  
> **Stack:** Go 1.26 · Dual SQLite (`modernc.org/sqlite`) & PostgreSQL (`pgx/v5`) · HTMX 1.9 · `html/template` · `sqlx`  
> **Architecture:** Server-Side Rendered Hypermedia (DDD domain-only TDD, vanilla CSS shadcn/QuickBooks Indigo tokens)

---

## ⚡ Fast Task-to-Reference Router (Read Only What You Need)

| If your task involves... | Read ONLY this file first | Key Invariants Summary |
|---|---|---|
| **UI, CSS, Colors, Buttons, Cards, Collapsible Columns** | [ui-design-tokens.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/ui-design-tokens.md) | 12px high-density scale, HSL `--primary` (`243 75% 59%`), `.card`/`.btn`/`.badge`, `initExcelCollapsibleColumns()`, bump `index.css?v=` |
| **HTMX Swaps, Toasts, Filters, Dropdowns, `<script>` tags** | [htmx-patterns.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/htmx-patterns.md) | **Never** use top-level `let`/`const` in partials (use `window.*`); explicit `hx-target="#main-content"`; `limit` selector (no pagination); `HX-Trigger` `show-toast` |
| **Sidebar, `+ New` Quick Create, Profile Menu, Z-Index** | [sidebar-navigation.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/sidebar-navigation.md) | 254px/68px sidebar, context-aware `#quick-create-menu`, `#profile-menu` (Users, Roles, Logs, System Settings), `.sidebar` `z-index: 10000` |
| **Mobile Breakpoints, Table Overflow, Column Resizing** | [responsive-design.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/responsive-design.md) | `.table-container` horizontal scroll, `white-space: nowrap`, `position: fixed; z-index: 999999` popovers, `localStorage` column widths |
| **Routes, Auth, Permissions (`RequirePermission`), PWA** | [routing.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/routing.md) | Complete route map, `rav_session` cookie (with legacy `radline_session` support), 17 RBAC permission keys, `/sw.js` & `/manifest.webmanifest` auth bypass |
| **Database Tables, SQLite + Postgres Queries, Indexes** | [database-schema.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-schema.md) | 17 tables, `DBConn` auto-rebinds `?`→`$1`, use `RETURNING id`, `ON CONFLICT DO NOTHING`, `substr(date, 1, 7)` |
| **Adding Columns/Tables, Postgres 500s, `GROUP BY`** | [database-migrations.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-migrations.md) | Update **both** `runSQLiteMigrations()` & `runPostgresMigrations()`, `COALESCE` nullable cols, list all non-agg cols in `GROUP BY` |
| **DTO Structs & Nullable FKs (`models/`)** | [models-layer.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/models-layer.md) | `models/` uses `*int` for nullable FKs; `domain/` uses `int` (`0` = unset). `CalculateStockOnHand` bridge |
| **Domain Models & Business Rules (`domain/`)** | [domain-overview.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-overview.md) | Pure Go (no DB/HTTP imports); validate in `New*()` / `Validate*()` constructors |
| **Domain TDD (`go test ./domain/...`)** | [domain-testing.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-testing.md) | Black-box `package domain_test` table-driven tests |
| **HTTP Handlers & `Render` vs `RenderPage`** | [handlers-overview.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/handlers-overview.md) | 3-way HTMX check (`HX-Request` + `HX-Target != "main-content"` → fragment, else `RenderPage`) |
| **Templates & `parseTemplates()` in `main.go`** | [templates-overview.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/templates-overview.md) | Register transitive child templates in both `pages` and `fragments` slices in `main.go` |

---

## 📌 Complete Map of Contents

### Architecture & Core
- [[architecture]] ([architecture.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/architecture.md)) — System architecture, package layering rules, data flow
- [[tech-stack]] ([tech-stack.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/tech-stack.md)) — Go 1.26, `modernc.org/sqlite`, `pgx/v5`, `sqlx`, `excelize`, HTMX

### Domain Layer (`domain/`)
- [[domain-overview]] ([domain-overview.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-overview.md)) — Complete entity & aggregate directory
- [[domain-item]] ([domain-item.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-item.md)) — `Item`, `Brand`, `Category`, `UomSetting`
- [[domain-customer]] ([domain-customer.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-customer.md)) — `Customer`, `User`/`Role`/`ActivityLog` (RBAC), `EarningGoal`, `Reminder`, `PurchaseOrder`
- [[domain-stock]] ([domain-stock.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-stock.md)) — `ItemStock` aggregate root, UOM conversion & FIFO (`GetOldestPLWithStock`)
- [[domain-receiving]] ([domain-receiving.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-receiving.md)) — `ReceivingLog` & `StockReceive` aggregate
- [[domain-sales]] ([domain-sales.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-sales.md)) — `SalesDetail` & `Sale` aggregate (remittance, deductions, profit margin)
- [[domain-adjustment]] ([domain-adjustment.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-adjustment.md)) — `InventoryAdjustment` & `StockAdjustment` aggregate
- [[domain-testing]] ([domain-testing.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-testing.md)) — Domain-only TDD strategy and test coverage map

### Persistence (`db/`, `models/`)
- [[database-schema]] ([database-schema.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-schema.md)) — Dual SQLite/PostgreSQL engine, all 17 tables, indexes, SQL portability rules
- [[database-migrations]] ([database-migrations.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-migrations.md)) — Startup migration parity (`runSQLiteMigrations` + `runPostgresMigrations`)
- [[models-layer]] ([models-layer.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/models-layer.md)) — DTO structs (`models/models.go`, `models/auth.go`, `models/goals_reminders.go`, `models/stock.go`)

### HTTP & Templates (`handlers/`, `templates/`)
- [[routing]] ([routing.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/routing.md)) — Complete route table, RBAC permissions, PWA endpoints
- [[handlers-overview]] ([handlers-overview.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/handlers-overview.md)) — Handler architecture, `App` struct, `Render` vs `RenderPage`
- [[handlers-master-data]] ([handlers-master-data.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/handlers-master-data.md)) — Brands, Categories, Items, `/items/search`, select fragments
- [[handlers-inventory]] ([handlers-inventory.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/handlers-inventory.md)) — Inventory overview, Monthly Inventory, Receiving, Adjustments
- [[handlers-sales]] ([handlers-sales.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/handlers-sales.md)) — Sales CRUD, batch edit, inline status popover
- [[templates-overview]] ([templates-overview.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/templates-overview.md)) — `parseTemplates()` hierarchy, fragments registration, "List Views" map
- [[htmx-patterns]] ([htmx-patterns.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/htmx-patterns.md)) — HTMX conventions, OOB swaps, toast protocol, `window.*` partial script rules

### Frontend (`static/`, `templates/base.html`)
- [[ui-design-tokens]] ([ui-design-tokens.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/ui-design-tokens.md)) — HSL CSS tokens, component classes, collapsible columns, fixed popovers
- [[sidebar-navigation]] ([sidebar-navigation.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/sidebar-navigation.md)) — Sidebar layout, `+ New` Quick Actions menu, `#profile-menu`
- [[responsive-design]] ([responsive-design.md](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/responsive-design.md)) — Mobile breakpoints, table scroll, dynamic column resizing
