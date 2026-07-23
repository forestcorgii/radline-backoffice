# Routing

> Back to [[00-index]] · Related: [[architecture]], [[handlers-overview]]

## File
[main.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/main.go) — lines 31–110

## Complete Route Table

### Dashboard
| Method | Path | Handler | Notes |
|---|---|---|---|
| `*` | `/` | `DashboardHandler` | Default catch-all |

### Brands CRUD → [[handlers-master-data]]
| Method | Path | Handler | HTMX Response |
|---|---|---|---|
| GET | `/brands` | `BrandsHandler` | Page or rows fragment |
| GET | `/brands/new` | `NewBrandPageHandler` | Add brand form page |
| GET | `/brands/select` | `SelectBrandsHandler` | Brand select option list |
| POST | `/brands/add` | `AddBrandHandler` | Rows fragment + toast |
| GET | `/brands/edit/{id}` | `EditBrandFormHandler` | Inline edit row |
| POST | `/brands/edit/{id}` | `UpdateBrandHandler` | Updated row + toast |
| GET | `/brands/row/{id}` | `BrandRowHandler` | Single row |
| DELETE | `/brands/delete/{id}` | `DeleteBrandHandler` | Empty + toast |

### Categories CRUD → [[handlers-master-data]]
| Method | Path | Handler | HTMX Response |
|---|---|---|---|
| GET | `/categories` | `CategoriesHandler` | Page or rows fragment |
| GET | `/categories/new` | `NewCategoryPageHandler` | Add category form page |
| GET | `/categories/select` | `SelectCategoriesHandler` | Category select option list |
| POST | `/categories/add` | `AddCategoryHandler` | Rows fragment + toast |
| GET | `/categories/edit/{id}` | `EditCategoryFormHandler` | Inline edit row |
| POST | `/categories/edit/{id}` | `UpdateCategoryHandler` | Updated row + toast |
| GET | `/categories/row/{id}` | `CategoryRowHandler` | Single row |
| DELETE | `/categories/delete/{id}` | `DeleteCategoryHandler` | Empty + toast |

### Items CRUD → [[handlers-master-data]]
| Method | Path | Handler | HTMX Response |
|---|---|---|---|
| GET | `/items` | `ItemsHandler` | Page or rows fragment |
| GET | `/items/new` | `NewItemPageHandler` | Add item form page |
| GET | `/items/select` | `SelectItemsHandler` | Item select option list |
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
| GET | `/sales/new` | `NewSalesPageHandler` | Add sales form page |
| POST | `/sales/add` | `AddSalesHandler` | Rows fragment + toast |
| GET | `/sales/new-row` | `NewSaleRowHandler` | Empty item row fragment |
| DELETE | `/sales/delete/{id}` | `DeleteSalesHandler` | Empty + toast |
| GET | `/sales/item-row-details` | `SaleItemRowDetailsHandler` | Pre-filled sale row |

### Settings & UOM Settings CRUD → [[handlers-overview]]
| Method | Path | Handler | HTMX Response / Notes |
|---|---|---|---|
| GET | `/settings` | `SettingsHandler` | Config page with list of UOM settings and predefined UOMs |
| GET | `/uom-settings/new` | `NewUomSettingPageHandler` | Add UOM conversion form page |
| POST | `/uom-settings/add` | `AddUomSettingHandler` | Rows fragment + toast |
| GET | `/uom-settings/edit/{id}` | `EditUomSettingFormHandler` | Inline edit row |
| POST | `/uom-settings/edit/{id}` | `UpdateUomSettingHandler` | Updated row + toast |
| GET | `/uom-settings/row/{id}` | `UomSettingRowHandler` | Single row |
| DELETE | `/uom-settings/delete/{id}` | `DeleteUomSettingHandler` | Empty + toast |

### Predefined UOM List CRUD
| Method | Path | Handler | HTMX Response / Notes |
|---|---|---|---|
| GET | `/uoms` | `UomsHandler` | Renders UOM predefined list rows |
| POST | `/uoms/add` | `AddUomHandler` | Rows fragment + toast |
| GET | `/uoms/row/{id}` | `UomRowHandler` | Single row |
| DELETE | `/uoms/delete/{id}` | `DeleteUomHandler` | Empty + toast |
| GET | `/uoms/select` | `SelectUomsHandler` | Predefined UOM select option list |

### Import Routing
| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/import` | `ImportPageHandler` | Consolidated data import page |
| POST | `/import/upload` | `ImportUploadHandler` | Excel spreadsheet upload and parse |

### Tools / Receipt Vision
| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/tools/receipt-scanner` | `ReceiptScannerHandler` | Receipt vision scanner uploader page |
| POST | `/tools/receipt-scanner/parse` | `ReceiptScannerParseHandler` | DeepSeek Vision parser and structured result renderer |

### Static Files
| Path | Source |
|---|---|
| `/static/*` | `static/` directory (CSS/JS) |

## Route Pattern Notes

- Go 1.22+ routing syntax: `"GET /path"`, `"POST /path"`, `"DELETE /path"`
- Path parameters: `{id}` — accessed via `r.PathValue("id")`
- The `/` route uses no method prefix → matches all methods (default handler)

## Related
- [[handlers-overview]] — Handler architecture
- [[htmx-patterns]] — How HTMX interacts with these routes
- [[templates-overview]] — What templates each route renders

---

### Context: Server Port Configuration Invariance

**Problem:**
Starting the Go local development server occasionally fails due to the default port `8080` being bound by another process. Changing the hardcoded listener port in `main.go` violates configuration invariants and can cause configuration drift or deployment failures.

**Enforced Solution:**
Never change the configured listener port in `main.go` (keep it as `:8080`). If port `8080` is in use:
- Check for existing processes running on port `8080` (e.g. using `netstat -ano | findstr :8080`).
- If you need to stop the old process, prompt/explain to the user why it needs to be stopped, and ask for permission before running process termination commands like `taskkill` or `kill`.
- Never commit port number modifications to `main.go` under any circumstances.
