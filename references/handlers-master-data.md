# Handlers — Master Data (Brands, Categories, Items)

> Back to [[handlers-overview]] · Related: [[routing]], [[domain-item]], [[domain-customer]]

## File
[handlers/master_data.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/master_data.go) (See [handlers/customers.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/customers.go) and [[domain-customer]] for Customers CRUD).

## Brands & Categories Handlers
- **List (`BrandsHandler`, `CategoriesHandler`)**: Accepts `search`, `filter` (`all`, `in_use`, `unused`), `sort`, and `limit` (`GetLimitParam`). Computes `item_count` via `LEFT JOIN items` + `COUNT(i.id)`.
- **Create (`AddBrandHandler`, `AddCategoryHandler`)**: Validates via `domain.NewBrand` / `domain.NewCategory`, inserts row, emits `HX-Trigger` toast (+ `brand-added` / `category-added` events for select refreshes).
- **Inline Edit & Delete**: `Edit*FormHandler` (`GET /.../edit/{id}`), `Update*Handler` (`POST /.../edit/{id}`), `*RowHandler` (`GET /.../row/{id}`), `Delete*Handler` (`DELETE /.../delete/{id}`, blocked with error toast on FK violation).

## Items Handlers
- **`ItemsHandler` (`GET /items`)**: Supports `search`, `brand_id_filter`, `category_id_filter`, `sort`, `limit`. Joins `brands` and `categories`.
- **`AddItemHandler` / `UpdateItemHandler`**: Maps optional `brand_id` and `category_id` (`0` or empty → SQL `NULL`).
- **`SearchItemsHandler` (`GET /items/search?q=&limit=`)**:
  - High-performance debounced endpoint (default `LIMIT 20`, hard cap `50`).
  - Uses indexed single-table scans and subqueries (`i.brand_id IN (SELECT id FROM brands WHERE name LIKE ?)`) instead of multi-table outer joins.
  - Returns `.dropdown-item` HTML by default, or lowercase JSON (`id, code, description, default_uom`) when `format=json` or `Accept: application/json`.

## Select Fragment Handlers
Return `<option>` fragments for dynamic HTMX `<select>` refreshes:
- `SelectBrandsHandler` (`GET /brands/select` → `brand_select.html`)
- `SelectCategoriesHandler` (`GET /categories/select` → `category_select.html`)
- `SelectItemsHandler` (`GET /items/select` → `item_select.html`)
- `SelectUomsHandler` (`GET /uoms/select` → `uom_select.html`, in `handlers/settings.go`)
