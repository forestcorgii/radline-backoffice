---
name: radline-ddd-feature
description: >-
  Step-by-step workflow for adding or modifying domain logic, database tables,
  migrations, DTOs, and HTTP handlers in Radline BackOffice. Activate when
  working in domain/, db/, models/, or handlers/.
---

# Radline DDD & Dual-Database Feature Workflow

## 1. Domain Layer (`domain/`) — Pure Go + TDD First
1. Consult [`references/domain-overview.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-overview.md) and [`references/domain-testing.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/domain-testing.md).
2. Write or update table-driven tests in `domain/*_test.go` (`package domain_test`) first.
3. Implement entities/aggregates in `domain/` using stdlib only (no `sqlx` or `net/http` imports).
4. Verify with `go test ./domain/... -v`.

## 2. Persistence Layer (`db/db.go`, `models/`) — SQLite + PostgreSQL Parity
1. Consult [`references/database-schema.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-schema.md) and [`references/database-migrations.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/database-migrations.md).
2. **Dual Schema & Migrations**:
   - Update **both** `createSchemaSQLite()` and `createSchemaPostgres()` in [`db/db.go`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/db/db.go).
   - Add startup column/table migrations to **both** `runSQLiteMigrations()` and `runPostgresMigrations()`.
3. **SQL Portability Rules**:
   - Use `?` placeholders everywhere (`db.DB` auto-rebinds `?` to `$1` on Postgres).
   - Use `RETURNING id` with `QueryRow().Scan(&id)` instead of `LastInsertId()`.
   - Use `ON CONFLICT DO NOTHING` instead of `INSERT OR IGNORE`.
   - Pass `time.Time` values for date comparisons against `TIMESTAMPTZ` columns; use `substr(col, 1, 7)` for `YYYY-MM` extraction.
   - Enumerate **all** non-aggregated `SELECT` columns in `GROUP BY` (required by PostgreSQL).
   - Wrap nullable columns in `COALESCE(col, '')` / `COALESCE(col, 0.0)` or use pointer fields (`*int`, `*int64`) in `models/` DTOs.

## 3. Handlers & Routing (`handlers/`, `main.go`)
1. Consult [`references/handlers-overview.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/handlers-overview.md) and [`references/routing.md`](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/references/routing.md).
2. Wrap routes in `main.go` with `app.RequirePermission(domain.Perm..., handler)`.
3. Log mutations via `app.LogActivity(r, action, entityType, entityID, details)`.
4. Never change the default `:8080` fallback port in `main.go`.

