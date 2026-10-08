# Handlers — Sales

> Back to [[handlers-overview]] · Related: [[routing]], [[domain-sales]]

## File
[handlers/sales.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/handlers/sales.go)

## Handlers

### `SalesHandler` (GET `/sales`)
- Lists all sales with search, supplier filter, and sort
- JOINs with items for item code display
- Search covers: `doc_number`, `customer_name`, `item_code`
- Sort options: `date_desc` (default), `date_asc`, `sales_desc`, `profit_desc`
- HTMX-aware: rows fragment or full page
- Full page includes items list for the add-sale form

### `AddSalesHandler` (POST `/sales/add`)
Multi-item sale flow:
1. Parse header: `doc_date`, `doc_type`, `doc_number`, `customer_name`, `supplier`
2. Parse item arrays: `item_id[]`, `qty[]`, `uom[]`, `price[]`, `cost[]`
3. Construct `[]domain.SaleItem`
4. `domain.NewSale(docType, "POSTED", date, docNo, customer, supplier, items)` — validates all
5. `sale.ToSalesDetails()` to flatten
6. INSERT each detail in a transaction
7. Return updated sales rows + success toast

**Note:** `DocStatus` is always hardcoded to `"POSTED"` — no draft/void workflow yet.

### `NewSaleRowHandler` (GET `/sales/new-row`)
Returns an empty `sale_item_row.html` fragment with item select dropdown.

### `DeleteSalesHandler` (DELETE `/sales/delete/{id}`)
Deletes a single `sales_details` row by ID.

### `SaleItemRowDetailsHandler` (GET `/sales/item-row-details`)
When an item is selected in a sale row:
- Fetches item's `default_uom`
- Fetches last receiving `cost` and `selling_price`
- Returns pre-filled `sale_item_row.html` with defaults

## Data Entry Integration

The `/entry` page (handled by [[handlers-master-data|`EntryHandler`]]) is the primary sales entry point. It loads brands, categories, items, and renders the sales form within a tabbed interface. The `AddSalesHandler` is called from this page via HTMX form submission.

## Learnings

### Context: Customer TIN, Address, Centered Headers & Compact Status Icons in Sales
**Problem**: Sales records lacked Tax Identification Number (TIN) and physical/billing address storage. Furthermore, the Sales listview table suffered horizontal space bloat due to text dropdown selects for status, and header labels were inconsistently aligned.
**Enforced Solution**:
- **TIN No. & Address Integration**:
  - Added `tin_no` and `address` to `sales_details` table with safe startup migrations.
  - Form fields positioned directly next to `customer_name` in both Encode Sales cards and `/sales/new`.
  - Added dedicated toggleable columns in the Sales log table next to Customer Name.
- **Space-Saving Status Icon Dropdown**:
  - Replaced bulky text `<select>` with a 24px circular interactive status icon button (`.status-icon-btn` with `.status-icon-posted` green check, `.status-icon-active` blue circle, `.status-icon-cancelled` red X).
  - Hovering provides full status title tooltip.
  - Clicking triggers an inline quick status popover (`.status-menu-popover`) allowing immediate HTMX status transition without opening the full edit row.
  - Reduced default column width from `110px` to `48px`, recovering `62px` of horizontal space across the grid.
- **Centered Header Labels**:
  - Enforced `.sales-log-table .th { text-align: center; }` to keep all table column labels centered.

## Related
- [[domain-sales]] — `Sale` aggregate and `SalesDetail` entity
- [[domain-stock]] — Sales reduce stock on hand
- [[routing]] — Route definitions
- [[htmx-patterns]] — Multi-row form submission, toast notifications
- [[handlers-master-data]] — Entry page handler

