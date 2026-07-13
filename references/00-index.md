# 🗂️ Radline BackOffice — Knowledge Vault

> **Module:** `radline`  
> **Stack:** Go 1.26 · SQLite (modernc) · HTMX 1.9 · html/template · sqlx  
> **Architecture:** Server-Side Rendered Hypermedia (no SPA, no JSON API)  
> **Domain Pattern:** Domain-Driven Design (DDD) with domain-only TDD

---

## 📌 Map of Contents

### Architecture & Core
- [[architecture]] — High-level system architecture, layering, and data flow
- [[tech-stack]] — Technology choices, dependencies, and rationale

### Domain Layer (`domain/`)
- [[domain-overview]] — Domain entities, aggregates, value objects
- [[domain-item]] — `Item`, `Brand`, `Category`, `UomSetting` entities
- [[domain-stock]] — `ItemStock` aggregate root & stock calculation
- [[domain-receiving]] — `ReceivingLog`, `StockReceive` aggregate
- [[domain-sales]] — `SalesDetail`, `Sale` aggregate
- [[domain-adjustment]] — `InventoryAdjustment`, `StockAdjustment` aggregate
- [[domain-testing]] — TDD strategy and test coverage map

### Persistence (`db/`, `models/`)
- [[database-schema]] — SQLite schema, tables, foreign keys
- [[database-migrations]] — Inline startup migration strategy
- [[models-layer]] — DTO structs, `models.go` and `models/stock.go`

### HTTP & Templates (`handlers/`, `templates/`)
- [[routing]] — All routes and their handler mapping
- [[handlers-overview]] — Handler architecture and `App` struct
- [[handlers-master-data]] — Brand, Category, Item, Entry CRUD handlers
- [[handlers-inventory]] — Inventory, Receiving, Adjustment handlers
- [[handlers-sales]] — Sales handlers
- [[templates-overview]] — Template hierarchy (base + pages + fragments)
- [[htmx-patterns]] — HTMX conventions, OOB swaps, toast protocol

### Frontend (`static/`, `templates/base.html`)
- [[ui-design-tokens]] — CSS variables, design system classes
- [[sidebar-navigation]] — Sidebar layout, dropdowns, mobile responsiveness
- [[responsive-design]] — Mobile breakpoints, table overflow, grid utilities

---

## 🔄 Quick Navigation

| Area | Entry Point |
|---|---|
| **Start coding** | [[architecture]] → [[routing]] |
| **Add a new entity** | [[domain-overview]] → [[database-schema]] → [[handlers-overview]] → [[templates-overview]] |
| **Fix a UI bug** | [[ui-design-tokens]] → [[htmx-patterns]] → [[templates-overview]] |
| **Add a domain feature** | [[domain-overview]] → [[domain-testing]] |
| **Understand stock math** | [[domain-stock]] → [[models-layer]] |
| **Debug HTMX** | [[htmx-patterns]] → [[handlers-overview]] |
