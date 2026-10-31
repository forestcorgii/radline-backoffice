# Handlers — Sales

> Back to [[handlers-overview]] · Related: [[routing]], [[domain-sales]], [[htmx-patterns]]

## File
[handlers/sales.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/sales.go)

## Handlers
- **`SalesHandler` (`GET /sales`)**:
  - Supports `search` (`doc_number`, `customer_name`, `item_code`), `supplier`, `doc_type`, `start_date`, `end_date`, `sort` (`date_desc`, `date_asc`, `sales_desc`, `profit_desc`), `limit`, and batch `edit_mode`.
  - Wraps nullable text/numeric projections in `COALESCE(...)` for PostgreSQL compatibility.
  - Renders `sales_results.html` on inline HTMX filter/search requests or `sales.html` via `RenderPage`.
- **`AddSalesHandler` (`POST /sales/add`)**:
  - Parses header (`doc_date`, `doc_type`, `doc_number`, `customer_name`, `tin_no`, `address`, `supplier`) and line-item arrays (`item_id[]`, `qty[]`, `uom[]`, `price[]`, `cost[]`, `patong[]`, `pos_charge[]`, `wt_2307[]`, `remarks[]`, `ref_pl[]`).
  - Validates via `domain.NewSale(...)` and inserts each `SalesDetail` inside a DB transaction.
- **`UpdateSalesHandler` (`POST /sales/edit/{id}`)**: Updates an existing sale line item (inline or batch edit mode) and recalculates financial totals.
- **`UpdateSalesStatusHandler` (`POST /sales/update-status/{id}`)**: Quick inline status transition (`ACTIVE`, `POSTED`, `CANCELLED`) triggered from the 24px circular `.status-icon-btn` popover (`.status-menu-popover`) without entering batch edit mode.
- **`NewSaleRowHandler` (`GET /sales/new-row`) & `SaleItemRowDetailsHandler` (`GET /sales/item-row-details`)**:
  - Renders `sale_item_row.html`. When an item is selected, fetches `ItemStock` and runs `GetOldestPLWithStock()` (FIFO) to auto-fill UOM, cost, selling price, and reference PL (`ref_pl`).
  - Uses `ORDER BY (id = ?) DESC, description ASC LIMIT 20` so the selected item is always present in the initial dropdown list.
- **`DeleteSalesHandler` (`DELETE /sales/delete/{id}`)**: Deletes a `sales_details` row and emits a toast.
