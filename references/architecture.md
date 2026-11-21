# Architecture

> Back to [[00-index]]

## Overview

Radline BackOffice (Go module `rav`) is a **server-side rendered (SSR) web application** for managing a distribution/wholesale business. It follows a **hypermedia-driven architecture** using [[htmx-patterns|HTMX]] (`/static/htmx.min.js`) as its primary interaction mechanism, supplemented by targeted JSON endpoints for user preferences (`GET/POST /api/user-preferences*`) and autocomplete (`?format=json` on `/items/search` and `/customers/search`).

```
┌───────────────────────────────────────────────────────────┐
│                         Browser                           │
│  ┌───────────────┐  ┌──────────┐  ┌────────────────────┐  │
│  │ htmx.min.js   │  │base.html │  │ index.css?v=6.5    │  │
│  │ base-ui.js    │  │ sidebar  │  │ design tokens      │  │
│  │ table-engine.js  └────┬─────┘  └────────────────────┘  │
│  └───────┬───────┘       │                                │
│          ▼               ▼                                │
│   HTTP Requests (GET/POST/DELETE — HTML & targeted JSON)  │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────┐
│                 Go HTTP Server (:8080)                    │
│  ┌────────────────────────────────────────────────┐       │
│  │ main.go — Routes, CSRF & Template Parsing      │       │
│  └───────────────────────┬────────────────────────┘       │
│                          │                                │
│  ┌───────────────────────▼────────────────────────┐       │
│  │ handlers/ — 17 HTTP Handler Files (App)        │       │
│  │   handlers.go        (Render, Dashboard, Prefs)│       │
│  │   middleware.go      (Auth, RBAC, Audit Log)   │       │
│  │   auth.go            (Login, Logout, UserChip) │       │
│  │   roles_users.go     (Roles & Users CRUD)      │       │
│  │   activity_logs.go   (Audit Trail Logs)        │       │
│  │   master_data.go     (Brands, Cat, Items)      │       │
│  │   customers.go       (Customers CRUD & Search) │       │
│  │   inventory.go       (Stock, Monthly, Recv/Adj)│       │
│  │   sales.go           (Sales CRUD, Batch, Status│       │
│  │   forms.go           (Quotation & PO Forms)    │       │
│  │   goals.go           (Earning Goals)           │       │
│  │   reminders.go       (Calendar & Reminders)    │       │
│  │   settings.go        (System Config & UOMs)    │       │
│  │   uom_settings.go    (UOM Conversions)         │       │
│  │   import.go          (Excel Data Import)       │       │
│  │   receipt_scanner.go (DeepSeek Vision Scanner) │       │
│  │   pagination.go      (GetLimitParam helper)    │       │
│  └───────────────┬────────────────────────────────┘       │
│                  │ uses                                   │
│  ┌───────────────▼───────┐  ┌─────────────────────────┐   │
│  │ domain/               │  │ models/                 │   │
│  │ Pure business logic   │  │ DTOs (models.go,        │   │
│  │ (stdlib + bcrypt)     │  │ auth.go, goals_reminders│   │
│  └───────────────────────┘  └────────────┬────────────┘   │
│                                          │                │
│  ┌───────────────────────────────────────▼────────────┐   │
│  │ db/db.go — Dual-Engine Init, Schema, Migrations    │   │
│  │ Global: db.DB (*db.DBConn) & *db.TxConn            │   │
│  └───────────────────────────────────────┬────────────┘   │
│                                          │                │
│  ┌───────────────────────────────────────▼────────────┐   │
│  │ SQLite (modernc.org/sqlite, WAL, backoffice.db)    │   │
│  │ OR PostgreSQL (github.com/jackc/pgx/v5)            │   │
│  └────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────┘
```

## Layering Rules

| Layer | Package | Allowed Dependencies | Forbidden |
|---|---|---|---|
| **Domain** | `domain/` | stdlib (`errors`, `time`, `math`, `strings`) + `golang.org/x/crypto/bcrypt` in `domain/auth.go` | `db`, `sqlx`, `net/http`, `html/template` |
| **Models/DTO** | `models/` | stdlib (`time`) | `net/http`, `html/template`, `db` |
| **Persistence** | `db/` | `sqlx`, `modernc.org/sqlite`, `github.com/jackc/pgx/v5/stdlib`, `golang.org/x/crypto/bcrypt` | `handlers/` |
| **Handlers** | `handlers/` | `domain/`, `models/`, `db/` | — |
| **Entry** | `main.go` | `db/`, `domain/`, `handlers/`, `html/template`, `net/http` | — |

## Key Architectural Decisions

1. **SSR HTML/HTMX Primary + Targeted JSON Endpoints** — Handlers primarily return HTML pages/fragments for HTMX swaps, with targeted JSON endpoints for user preferences (`GET/POST /api/user-preferences*`) and autocomplete (`?format=json` on `/items/search` and `/customers/search`). See [[htmx-patterns]].
2. **Dual Database Engine (`*db.DBConn` & `*db.TxConn`)** — `db.DB` wraps `*sqlx.DB` in `*db.DBConn` (and transactions in `*db.TxConn`), automatically rebinding `?` placeholders to `$1, $2, ...` on PostgreSQL (`github.com/jackc/pgx/v5`) while running pure-Go SQLite (`modernc.org/sqlite` with WAL, `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`) in local development. See [[database-schema]].
3. **Template map** — All templates are pre-parsed at startup into `map[string]*template.Template`. See [[templates-overview]].
4. **Domain isolation** — Business logic lives in `domain/` with zero database or HTTP dependencies. See [[domain-overview]].
5. **Inline migrations** — Idempotent startup migrations maintain 1:1 parity across SQLite and PostgreSQL (`runSQLiteMigrations` and `runPostgresMigrations`). See [[database-migrations]].

## Related
- [[tech-stack]] — Dependency details
- [[routing]] — Complete route table
- [[domain-overview]] — Domain layer deep dive
