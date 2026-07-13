# Routing

> Back to [[00-index]] · Related: [[architecture]], [[handlers-overview]]

## File
[main.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/main.go) — lines 31–86

## Complete Route Table

### Dashboard
| Method | Path | Handler | Notes |
|---|---|---|---|
| `*` | `/` | `DashboardHandler` | Default catch-all |

### Brands CRUD → [[handlers-master-data]]
| Method | Path | Handler | HTMX Response |
|---|---|---|---|
| GET | `/brands` | `BrandsHandler` | Page or rows fragment |
| POST | `/brands/add` | `AddBrandHandler` | Rows fragment + toast |
| GET | `/brands/edit/{id}` | `EditBrandFormHandler` | Inline edit row |
| POST | `/brands/edit/{id}` | `UpdateBrandHandler` | Updated row + toast |
| GET | `/brands/row/{id}` | `BrandRowHandler` | Single row |
| DELETE | `/brands/delete/{id}` | `DeleteBrandHandler` | Empty + toast |

### Categories CRUD → [[handlers-master-data]]
| Method | Path | Handler | HTMX Response |
|---|---|---|---|
| GET | `/categories` | `CategoriesHandler` | Page or rows fragment |
| POST | `/categories/add` | `AddCategoryHandler` | Rows fragment + toast |
| GET | `/categories/edit/{id}` | `EditCategoryFormHandler` | Inline edit row |
| POST | `/categories/edit/{id}` | `UpdateCategoryHandler` | Updated row + toast |
| GET | `/categories/row/{id}` | `CategoryRowHandler` | Single row |
| DELETE | `/categories/delete/{id}` | `DeleteCategoryHandler` | Empty + toast |

### Items CRUD → [[handlers-master-data]]
| Method | Path | Handler | HTMX Response |
|---|---|---|---|
| GET | `/items` | `ItemsHandler` | Page or rows fragment |
| POST | `/items/add` | `AddItemHandler` | Rows fragment + toast |
| GET | `/items/edit/{id}` | `EditItemFormHandler` | Inline edit row |
| POST | `/items/edit/{id}` | `UpdateItemHandler` | Updated row + toast |
| GET | `/items/row/{id}` | `ItemRowHandler` | Single row |
| DELETE | `/items/delete/{id}` | `DeleteItemHandler` | Empty + toast |

### Inventory → [[handlers-inventory]]
| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/inventory` | `InventoryHandler` | Stock overview page |
| GET | `/inventory/stock` | `InventoryHandler` | Alias for `/inventory` |
| GET | `/inventory/monthly` | `MonthlyInventoryHandler` | Monthly movement report |
| GET | `/inventory/receiving` | `StockReceivingPageHandler` | Receiving form + logs |
| GET | `/inventory/receiving/logs` | `ReceivingLogsHandler` | Receiving logs listing |
| POST | `/inventory/receiving/add` | `ReceiveStockHandler` | Multi-item receive |
| GET | `/inventory/receiving/new-row` | `NewReceivingRowHandler` | Empty item row fragment |
| GET | `/inventory/receiving/item-row-details` | `ReceivingItemRowDetailsHandler` | Pre-filled item row |
| GET | `/inventory/adjustments` | `StockAdjustmentsPageHandler` | Adjustment form + logs |
| GET | `/inventory/adjustments/logs` | `AdjustmentLogsHandler` | Adjustment logs listing |
| POST | `/inventory/adjustments/add` | `AdjustStockHandler` | Multi-item adjustment |
| GET | `/inventory/adjustments/new-row` | `NewAdjustmentRowHandler` | Empty item row fragment |
| GET | `/inventory/adjustments/item-row-details` | `AdjustmentItemRowDetailsHandler` | Pre-filled item row |

### Sales → [[handlers-sales]]
| Method | Path | Handler | HTMX Response |
|---|---|---|---|
| GET | `/sales` | `SalesHandler` | Page or rows fragment |
| POST | `/sales/add` | `AddSalesHandler` | Rows fragment + toast |
| GET | `/sales/new-row` | `NewSaleRowHandler` | Empty item row fragment |
| DELETE | `/sales/delete/{id}` | `DeleteSalesHandler` | Empty + toast |

### Data Entry → [[handlers-master-data]]
| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/entry` | `EntryHandler` | Consolidated entry page |
| GET | `/entry/select/brands` | `SelectBrandsHandler` | Brand `<select>` fragment |
| GET | `/entry/select/categories` | `SelectCategoriesHandler` | Category `<select>` fragment |
| GET | `/entry/select/items` | `SelectItemsHandler` | Item `<select>` fragment |
| GET | `/entry/item-defaults` | `ItemDefaultsHandler` | OOB swap for UOM/cost/price |
| GET | `/sales/item-row-details` | `SaleItemRowDetailsHandler` | Pre-filled sale row |

### Static Files
| Path | Source |
|---|---|
| `/static/*` | `static/` directory (CSS) |

## Route Pattern Notes

- Go 1.22+ routing syntax: `"GET /path"`, `"POST /path"`, `"DELETE /path"`
- Path parameters: `{id}` — accessed via `r.PathValue("id")`
- The `/` route uses no method prefix → matches all methods (default handler)

## Related
- [[handlers-overview]] — Handler architecture
- [[htmx-patterns]] — How HTMX interacts with these routes
- [[templates-overview]] — What templates each route renders
