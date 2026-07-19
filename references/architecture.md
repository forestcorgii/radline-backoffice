# Architecture

> Back to [[00-index]]

## Overview

Radline BackOffice is a **server-side rendered (SSR) web application** for managing a distribution/wholesale business. It follows a strict **hypermedia-driven architecture** using [[htmx-patterns|HTMX]] — no JSON API, no SPA framework.

```
┌─────────────────────────────────────────────────┐
│                   Browser                        │
│  ┌────────┐  ┌─────────┐  ┌──────────────────┐  │
│  │ HTMX   │  │ base.html│  │ index.css        │  │
│  │ 1.9.10 │  │ sidebar  │  │ design tokens    │  │
│  └────┬───┘  └────┬────┘  └──────────────────┘  │
│       │           │                              │
│       ▼           ▼                              │
│   HTTP Requests (GET/POST/DELETE)                │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│              Go HTTP Server (:8080)              │
│  ┌──────────────────────────────────────┐       │
│  │ main.go — Routes & Template Parsing  │       │
│  └──────────────────┬───────────────────┘       │
│                     │                            │
│  ┌──────────────────────────────────────┐       │
│  │ handlers/  — HTTP Handlers (App)     │       │
│  │   handlers.go   (Render, Dashboard)  │       │
│  │   master_data.go (Brands/Cat/Items)  │       │
│  │   inventory.go  (Stock/Recv/Adj)     │       │
│  │   sales.go      (Sales CRUD)         │       │
│  │   import.go     (Excel Data Import)  │       │
│  │   settings.go   (System Config/UOMs) │       │
│  │   uom_settings.go (UOM Conversions)  │       │
│  │   receipt_scanner.go (OCR Scanner)   │       │
│  │   pagination.go (Limit helper)       │       │
│  └──────────┬───────────────────────────┘       │
│             │ uses                               │
│  ┌──────────▼──────────┐  ┌─────────────────┐   │
│  │ domain/             │  │ models/         │   │
│  │ Pure business logic │  │ DTOs + Stock    │   │
│  │ (no DB, no HTTP)    │  │ Calculator      │   │
│  └─────────────────────┘  └──────┬──────────┘   │
│                                  │               │
│  ┌───────────────────────────────▼──────────┐   │
│  │ db/db.go — SQLite Init, Schema, Migrate  │   │
│  │ Global: db.DB (*sqlx.DB)                 │   │
│  └──────────────────────────────────────────┘   │
│                                                  │
│  ┌──────────────────────────────────────────┐   │
│  │ SQLite File: backoffice.db               │   │
│  └──────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

## Layering Rules

| Layer | Package | Allowed Dependencies | Forbidden |
|---|---|---|---|
| **Domain** | `domain/` | stdlib only (`errors`, `time`) | `db`, `net/http`, `html/template` |
| **Models/DTO** | `models/` | `domain/`, `database/sql`, `sqlx` | `net/http`, `html/template` |
| **Persistence** | `db/` | `sqlx`, `modernc.org/sqlite` | `domain/`, `handlers/` |
| **Handlers** | `handlers/` | `domain/`, `models/`, `db/` | — |
| **Entry** | `main.go` | `db/`, `handlers/`, `html/template` | — |

## Key Architectural Decisions

1. **No JSON API** — All handler responses return HTML fragments. See [[htmx-patterns]].
2. **Global DB singleton** — `db.DB` is a package-level `*sqlx.DB`. See [[database-schema]].
3. **Template map** — All templates are pre-parsed at startup into `map[string]*template.Template`. See [[templates-overview]].
4. **Domain isolation** — Business logic lives in pure Go structs with no infrastructure deps. See [[domain-overview]].
5. **Inline migrations** — No migration tool; schema changes handled on startup. See [[database-migrations]].

## Related
- [[tech-stack]] — Dependency details
- [[routing]] — Complete route table
- [[domain-overview]] — Domain layer deep dive
