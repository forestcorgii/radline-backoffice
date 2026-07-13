# Handlers — Master Data

> Back to [[handlers-overview]] · Related: [[routing]], [[domain-item]]

## File
[handlers/master_data.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/master_data.go)

## Brands Handlers

### `BrandsHandler` (GET `/brands`)
- Accepts query params: `search`, `filter` (all/in_use/unused), `sort`
- Builds dynamic SQL with WHERE/HAVING/ORDER BY
- Item count computed via `LEFT JOIN items` + `COUNT(i.id)`
- HTMX-aware: returns rows fragment or full page

### `AddBrandHandler` (POST `/brands/add`)
- Validates code + name are non-empty
- INSERT with unique constraint on code
- Returns full brand rows list + success toast

### `EditBrandFormHandler` / `UpdateBrandHandler` / `BrandRowHandler`
- Inline edit flow: click Edit → swap row with form → submit → swap back to row
- Uses `r.PathValue("id")` for Go 1.22+ path parameters

### `DeleteBrandHandler` (DELETE `/brands/delete/{id}`)
- FK constraint prevents deletion if brand has items
- Returns conflict error toast on FK violation

---

## Categories Handlers
Mirror the exact same pattern as Brands. See source for details.

---

## Items Handlers

### `ItemsHandler` (GET `/items`)
- Additional filters: `brand_id_filter`, `category_id_filter`
- JOIN with brands + categories for display names
- Full page render includes brand/category lists for filter dropdowns

### `AddItemHandler` (POST `/items/add`)
- Handles nullable `brand_id` and `category_id`:
  ```go
  var brandID interface{}
  brandIDVal, err := strconv.Atoi(brandIDStr)
  if err == nil && brandIDVal > 0 {
      brandID = brandIDVal
  } else {
      brandID = nil  // SQL NULL
  }
  ```

### `EditItemFormHandler`
- Loads item + all brands + all categories for the edit form dropdowns

---

## Entry Page Handler

### `EntryHandler` (GET `/entry`)
- Consolidated data entry page with tabs
- `tab=receiving` and `tab=adjustments` redirect to `/inventory/receiving` and `/inventory/adjustments`
- Default tab: `sales`
- Loads brands, categories, items for the sales entry form

---

## Select Fragment Handlers

These return `<select>` option fragments for HTMX dynamic updates:

| Handler | Route | Template |
|---|---|---|
| `SelectBrandsHandler` | `/entry/select/brands` | `brand_select.html` |
| `SelectCategoriesHandler` | `/entry/select/categories` | `category_select.html` |
| `SelectItemsHandler` | `/entry/select/items` | `item_select.html` |

---

## ItemDefaultsHandler (GET `/entry/item-defaults`)

Returns OOB (Out-of-Band) swap HTML for auto-filling form fields when an item is selected:
- Looks up the item's `default_uom`
- Fetches last receiving `cost` and `selling_price` for the item
- Returns raw HTML with `hx-swap-oob="true"` attributes for inline element replacement
- Handles two contexts: `tab=receiving` and `tab=adjustments` with different fields

See [[htmx-patterns]] for OOB swap details.

## Related
- [[routing]] — Route definitions
- [[domain-item]] — Domain entities for brands/categories/items
- [[database-schema]] — Table structures
- [[htmx-patterns]] — Toast notifications and OOB swaps
- [[templates-overview]] — Template files used
