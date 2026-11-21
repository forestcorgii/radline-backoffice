# Routing & Middleware

> Back to [[00-index]] · Related: [[architecture]], [[handlers-overview]], [[htmx-patterns]]

## File
[`main.go`](../main.go) · Middleware in [`handlers/middleware.go`](../handlers/middleware.go)

## Security & Middleware Stack
- **CSRF Protection**: `http.CrossOriginProtection` wraps the router alongside `app.AuthMiddleware(mux)` to block cross-origin state-mutating requests.
- **Session Cookie**: Uses `rav_session` (`HttpOnly`, `SameSiteLaxMode`, `Secure` in production) stored in the `user_sessions` table.
- **Permission Guard**: `app.RequirePermission(permissionKey, handler)` enforces RBAC across all protected routes.

## Complete Route Table

### Public & PWA Assets (Bypass `AuthMiddleware`)
| Method | Path | Handler / Source | Notes |
|---|---|---|---|
| GET | `/login` | `LoginPageHandler` | Login page |
| POST | `/login` | `LoginSubmitHandler` | Session creation (`rav_session` cookie) |
| POST | `/logout` | `LogoutHandler` | Clears session cookie (`GET /logout` is removed to prevent CSRF logout attacks) |
| GET | `/auth/user-chip` | `UserChipHandler` | Sidebar profile chip & `#profile-menu` |
| GET | `/sw.js`, `/manifest.webmanifest`, `/offline.html`, `/favicon.ico` | Inline file handlers | PWA service worker, manifest, offline fallback |
| GET | `/static/*` | `http.FileServer` | Static CSS/JS/icons |

### User Preferences JSON Endpoints (Authenticated)
| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/api/user-preferences` | `GetUserPreferenceHandler` | Fetch user preference(s) from `user_preferences` |
| GET | `/api/user-preferences/{key}` | `GetUserPreferenceHandler` | Fetch single preference key as JSON |
| POST | `/api/user-preferences` | `SaveUserPreferenceHandler` | Upsert preference key/value JSON in `user_preferences` |

### Dashboard, Goals & Operational Calendar
| Method | Path | Handler | Permission | Notes |
|---|---|---|---|---|
| GET | `/{$}` | `DashboardHandler` | `PermDashboardView` | Exact root route (`GET /{$}`); financial KPIs, `fragment=trends` SVG charts, top products |
| POST | `/dashboard/layout` | `SaveDashboardLayoutHandler` | `PermDashboardView` | Saves JSON card order, width & height to `user_preferences` |
| POST | `/dashboard/layout/reset` | `ResetDashboardLayoutHandler` | `PermDashboardView` | Resets user dashboard layout to system default |
| GET | `/goals` | `GoalsHandler` | `PermDashboardView` | Revenue & profit targets (`handlers/goals.go`) |
| POST | `/goals/add` | `AddGoalHandler` | `PermSettingsEdit` | Create earning goal (requires edit permission) |
| DELETE | `/goals/delete/{id}` | `DeleteGoalHandler` | `PermSettingsEdit` | Delete earning goal (requires edit permission) |
| GET | `/calendar`, `/reminders` | `RemindersHandler` | `PermDashboardView` | Monthly 7-col grid + agenda (`handlers/reminders.go`) |
| POST | `/reminders/add`, `/calendar/add` | `AddReminderHandler` | `PermDashboardView` (or `PermSettingsEdit` / authenticated operator) | Create reminder (supports `?action=new` modal trigger) |
| POST | `/reminders/edit/{id}`, `/calendar/edit/{id}` | `EditReminderHandler` | `PermDashboardView` | Update title, due date, priority, status, assignee |
| POST | `/reminders/toggle/{id}`, `/calendar/toggle/{id}` | `ToggleReminderHandler` | `PermDashboardView` | Toggle Pending/Completed |
| POST | `/reminders/reschedule/{id}`, `/calendar/reschedule/{id}` | `RescheduleReminderHandler` | `PermDashboardView` | Drag-and-drop date reschedule |
| DELETE | `/reminders/delete/{id}`, `/calendar/delete/{id}` | `DeleteReminderHandler` | `PermDashboardView` | Delete reminder |

### Master Data CRUD (`PermItemsView` / `PermItemsEdit`) → [[handlers-master-data]], [[domain-customer]]
| Entity | Routes | Handlers |
|---|---|---|
| **Brands** | `GET /brands`, `GET /brands/new`, `GET /brands/select`, `POST /brands/add`, `GET|POST /brands/edit/{id}`, `GET /brands/row/{id}`, `DELETE /brands/delete/{id}` | `BrandsHandler`, `NewBrandPageHandler`, `SelectBrandsHandler`, `AddBrandHandler`, `EditBrandFormHandler`, `UpdateBrandHandler`, `BrandRowHandler`, `DeleteBrandHandler` |
| **Categories** | `GET /categories`, `GET /categories/new`, `GET /categories/select`, `POST /categories/add`, `GET|POST /categories/edit/{id}`, `GET /categories/row/{id}`, `DELETE /categories/delete/{id}` | `CategoriesHandler`, `NewCategoryPageHandler`, `SelectCategoriesHandler`, `AddCategoryHandler`, `EditCategoryFormHandler`, `UpdateCategoryHandler`, `CategoryRowHandler`, `DeleteCategoryHandler` |
| **Items** | `GET /items`, `GET /items/new`, `GET /items/select`, `GET /items/search`, `POST /items/add`, `GET|POST /items/edit/{id}`, `GET /items/row/{id}`, `DELETE /items/delete/{id}` | `ItemsHandler`, `NewItemPageHandler`, `SelectItemsHandler`, `SearchItemsHandler`, `AddItemHandler`, `EditItemFormHandler`, `UpdateItemHandler`, `ItemRowHandler`, `DeleteItemHandler` |
| **Customers** | `GET /customers`, `POST /customers/add`, `GET|POST /customers/edit/{id}`, `GET /customers/row/{id}`, `DELETE /customers/delete/{id}`, `GET /customers/search` | `CustomersHandler`, `AddCustomerHandler`, `EditCustomerFormHandler`, `UpdateCustomerHandler`, `CustomerRowHandler`, `DeleteCustomerHandler`, `SearchCustomersHandler` |

### Inventory, Receiving & Adjustments (`PermInventoryView|Receive|Adjust`) → [[handlers-inventory]]
| Method | Path | Handler | Notes |
|---|---|---|---|
| GET | `/inventory`, `/inventory/stock` | `InventoryHandler` | Stock overview |
| GET | `/inventory/monthly` | `MonthlyInventoryHandler` | Monthly movement report |
| GET / POST | `/inventory/receiving`, `/inventory/receiving/add` | `StockReceivingPageHandler`, `ReceiveStockHandler` | Multi-item stock receiving |
| GET | `/inventory/receiving/logs` | `ReceivingLogsHandler` | Receiving logs list |
| GET / POST / DELETE | `/inventory/receiving/logs/edit/{id}`, `/inventory/receiving/logs/delete/{id}` | `EditReceivingLogHandler`, `UpdateReceivingLogHandler`, `DeleteReceivingLogHandler` | Receiving log inline edit/delete |
| GET | `/inventory/receiving/new-row`, `/inventory/receiving/item-row-details` | `NewReceivingRowHandler`, `ReceivingItemRowDetailsHandler` | Dynamic receiving row fragments |
| GET / POST | `/inventory/adjustments`, `/inventory/adjustments/add` | `StockAdjustmentsPageHandler`, `AdjustStockHandler` | Multi-item stock adjustment |
| GET | `/inventory/adjustments/logs` | `AdjustmentLogsHandler` | Adjustment logs list |
| GET | `/inventory/adjustments/new-row`, `/inventory/adjustments/item-row-details` | `NewAdjustmentRowHandler`, `AdjustmentItemRowDetailsHandler` | Dynamic adjustment row fragments |

### Sales & Printable Forms → [[handlers-sales]]
| Method | Path | Handler | Permission |
|---|---|---|---|
| GET | `/sales`, `/sales/new` | `SalesHandler`, `NewSalesPageHandler` | `PermSalesView`, `PermSalesCreate` |
| POST | `/sales/add`, `/sales/edit/{id}`, `/sales/update-status/{id}` | `AddSalesHandler`, `UpdateSalesHandler`, `UpdateSalesStatusHandler` | `PermSalesCreate`, `PermSalesEdit` |
| GET / DELETE | `/sales/new-row`, `/sales/item-row-details`, `/sales/delete/{id}` | `NewSaleRowHandler`, `SaleItemRowDetailsHandler`, `DeleteSalesHandler` | `PermSalesCreate`, `PermSalesView`, `PermSalesDelete` |
| GET | `/forms/quotation`, `/forms/purchase-order` | `QuotationFormHandler`, `PurchaseOrderFormHandler` | `PermSalesView`, `PermInventoryView` |
| POST | `/forms/purchase-order/save` | `SavePurchaseOrderHandler` | `PermInventoryReceive` |

### UOMs, Settings, RBAC & Tools
| Area | Routes | Handlers |
|---|---|---|
| **UOMs & Conversions** | `GET /uoms`, `POST /uoms/add`, `GET /uoms/row/{id}`, `DELETE /uoms/delete/{id}`, `GET /uoms/select`, `GET /uom-settings`, `GET /uom-settings/new`, `POST /uom-settings/add`, `GET|POST /uom-settings/edit/{id}`, `GET /uom-settings/row/{id}`, `DELETE /uom-settings/delete/{id}` | `UomsHandler`, `AddUomHandler`, `UomRowHandler`, `DeleteUomHandler`, `SelectUomsHandler`, `UomSettingsHandler`, `NewUomSettingPageHandler`, `AddUomSettingHandler`, `EditUomSettingFormHandler`, `UpdateUomSettingHandler`, `UomSettingRowHandler`, `DeleteUomSettingHandler` |
| **System & RBAC** | `GET /settings` (→ `/brands`), `GET /settings/system`, `POST /settings/deepseek`, `GET|POST /settings/roles*`, `GET|POST /settings/users*`, `GET /activity-logs` | `SettingsHandler`, `SystemSettingsHandler`, `SaveDeepSeekConfigHandler`, `RolesHandler`, `UsersHandler`, `ActivityLogsHandler` |
| **Import & Scanner** | `GET /import`, `POST /import/upload`, `GET /tools/receipt-scanner`, `POST /tools/receipt-scanner/parse` | `ImportPageHandler`, `ImportUploadHandler`, `ReceiptScannerHandler`, `ReceiptScannerParseHandler` |

---

## Learnings & Enforced Routing Invariants

### Context: Server Port Configuration Invariance (`:8080`)
- **Problem**: Changing the default listener port in `main.go` when `:8080` is occupied causes configuration drift.
- **Enforced Solution**: Never modify the default `8080` port in `main.go`. Check existing processes with `netstat -ano | findstr :8080` and ask the user before terminating any process.

### Context: Printable Form Views SPA Swapping & Flexible HTMX Target Checks
- **Problem**: Calling `app.Render` instead of `app.RenderPage` on navigable pages nests `base.html` inside `#main-content`. Also, browsers may send `HX-Target: main-content` (without `#`).
- **Enforced Solution**:
  - All full page templates extending `base.html` (including `form_quotation.html` and `form_purchase_order.html`) must use `app.RenderPage(w, r, ...)`.
  - When checking for main-content HTMX requests in redirect/action handlers, accept `target == "#main-content" || target == "main-content" || r.Header.Get("HX-Request") == "true"` and emit `HX-Location: {"path": returnUrl, "target": "#main-content"}`.

### Context: Calendar Production 500 Prevention (`TIMESTAMPTZ` & `repeat(7, minmax(0, 1fr))`)
- **Problem**: Passing string dates to `WHERE due_date >= ?` fails on PostgreSQL `TIMESTAMPTZ` (`operator does not exist: timestamp with time zone >= text`), and `1fr` grid columns expand on long titles.
- **Enforced Solution**: Pass native `time.Time` values for date bounds in `RemindersHandler`, call `db.EnsureRemindersTable()` defensively, and use `grid-template-columns: repeat(7, minmax(0, 1fr))` with `min-width: 0; overflow: hidden` on calendar cells.
