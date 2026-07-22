package main

import (
	"errors"
	"html/template"
	"log"
	"net/http"

	"radline/db"
	"radline/handlers"
)

func main() {
	// Initialize Database
	err := db.InitDB("backoffice.db")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Parse Templates
	templates := parseTemplates()

	// Initialize App Handlers
	app := &handlers.App{
		Templates: templates,
	}

	// Serve Static Files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes
	http.HandleFunc("/", app.DashboardHandler)

	// Brands CRUD
	http.HandleFunc("GET /brands", app.BrandsHandler)
	http.HandleFunc("GET /brands/new", app.NewBrandPageHandler)
	http.HandleFunc("GET /brands/select", app.SelectBrandsHandler)
	http.HandleFunc("POST /brands/add", app.AddBrandHandler)
	http.HandleFunc("GET /brands/edit/{id}", app.EditBrandFormHandler)
	http.HandleFunc("POST /brands/edit/{id}", app.UpdateBrandHandler)
	http.HandleFunc("GET /brands/row/{id}", app.BrandRowHandler)
	http.HandleFunc("DELETE /brands/delete/{id}", app.DeleteBrandHandler)

	// Categories CRUD
	http.HandleFunc("GET /categories", app.CategoriesHandler)
	http.HandleFunc("GET /categories/new", app.NewCategoryPageHandler)
	http.HandleFunc("GET /categories/select", app.SelectCategoriesHandler)
	http.HandleFunc("POST /categories/add", app.AddCategoryHandler)
	http.HandleFunc("GET /categories/edit/{id}", app.EditCategoryFormHandler)
	http.HandleFunc("POST /categories/edit/{id}", app.UpdateCategoryHandler)
	http.HandleFunc("GET /categories/row/{id}", app.CategoryRowHandler)
	http.HandleFunc("DELETE /categories/delete/{id}", app.DeleteCategoryHandler)

	// Items CRUD
	http.HandleFunc("GET /items", app.ItemsHandler)
	http.HandleFunc("GET /items/new", app.NewItemPageHandler)
	http.HandleFunc("GET /items/select", app.SelectItemsHandler)
	http.HandleFunc("POST /items/add", app.AddItemHandler)
	http.HandleFunc("GET /items/edit/{id}", app.EditItemFormHandler)
	http.HandleFunc("POST /items/edit/{id}", app.UpdateItemHandler)
	http.HandleFunc("GET /items/row/{id}", app.ItemRowHandler)
	http.HandleFunc("DELETE /items/delete/{id}", app.DeleteItemHandler)

	// Inventory (Landing, Receiving & Adjustments)
	http.HandleFunc("GET /inventory", app.InventoryHandler)
	http.HandleFunc("GET /inventory/stock", app.InventoryHandler)
	http.HandleFunc("GET /inventory/monthly", app.MonthlyInventoryHandler)
	http.HandleFunc("GET /inventory/receiving", app.StockReceivingPageHandler)
	http.HandleFunc("GET /inventory/receiving/logs", app.ReceivingLogsHandler)
	http.HandleFunc("POST /inventory/receiving/add", app.ReceiveStockHandler)
	http.HandleFunc("GET /inventory/receiving/new-row", app.NewReceivingRowHandler)
	http.HandleFunc("GET /inventory/receiving/item-row-details", app.ReceivingItemRowDetailsHandler)
	http.HandleFunc("DELETE /inventory/receiving/logs/delete/{id}", app.DeleteReceivingLogHandler)
	http.HandleFunc("GET /inventory/receiving/logs/edit/{id}", app.EditReceivingLogHandler)
	http.HandleFunc("POST /inventory/receiving/logs/edit/{id}", app.UpdateReceivingLogHandler)
	http.HandleFunc("GET /inventory/adjustments", app.StockAdjustmentsPageHandler)
	http.HandleFunc("GET /inventory/adjustments/logs", app.AdjustmentLogsHandler)
	http.HandleFunc("POST /inventory/adjustments/add", app.AdjustStockHandler)
	http.HandleFunc("GET /inventory/adjustments/new-row", app.NewAdjustmentRowHandler)
	http.HandleFunc("GET /inventory/adjustments/item-row-details", app.AdjustmentItemRowDetailsHandler)

	// Sales Routing
	http.HandleFunc("GET /sales", app.SalesHandler)
	http.HandleFunc("GET /sales/new", app.NewSalesPageHandler)
	http.HandleFunc("POST /sales/add", app.AddSalesHandler)
	http.HandleFunc("POST /sales/edit/{id}", app.UpdateSalesHandler)
	http.HandleFunc("GET /sales/new-row", app.NewSaleRowHandler)
	http.HandleFunc("DELETE /sales/delete/{id}", app.DeleteSalesHandler)
	http.HandleFunc("GET /sales/item-row-details", app.SaleItemRowDetailsHandler)

	// Settings CRUD
	http.HandleFunc("GET /settings", app.SettingsHandler)
	http.HandleFunc("POST /settings/deepseek", app.SaveDeepSeekConfigHandler)
	http.HandleFunc("GET /uom-settings/new", app.NewUomSettingPageHandler)
	http.HandleFunc("POST /uom-settings/add", app.AddUomSettingHandler)
	http.HandleFunc("GET /uom-settings/edit/{id}", app.EditUomSettingFormHandler)
	http.HandleFunc("POST /uom-settings/edit/{id}", app.UpdateUomSettingHandler)
	http.HandleFunc("GET /uom-settings/row/{id}", app.UomSettingRowHandler)
	http.HandleFunc("DELETE /uom-settings/delete/{id}", app.DeleteUomSettingHandler)

	// UOM Predefined List CRUD
	http.HandleFunc("GET /uoms", app.UomsHandler)
	http.HandleFunc("POST /uoms/add", app.AddUomHandler)
	http.HandleFunc("GET /uoms/row/{id}", app.UomRowHandler)
	http.HandleFunc("DELETE /uoms/delete/{id}", app.DeleteUomHandler)
	http.HandleFunc("GET /uoms/select", app.SelectUomsHandler)

	// Import Routing
	http.HandleFunc("GET /import", app.ImportPageHandler)
	http.HandleFunc("POST /import/upload", app.ImportUploadHandler)

	// Receipt Scanner Routing
	http.HandleFunc("GET /tools/receipt-scanner", app.ReceiptScannerHandler)
	http.HandleFunc("POST /tools/receipt-scanner/parse", app.ReceiptScannerParseHandler)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func parseTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}

	// Pages that use base.html
	pages := []string{
		"dashboard.html", "brands.html", "categories.html", "items.html", "uom_settings.html",
		"inventory.html", "stock_receiving.html", "stock_adjustments.html",
		"sales.html", "monthly_inventory.html", "receiving_logs.html", "adjustment_logs.html",
		"brand_new.html", "category_new.html", "item_new.html", "sales_new.html",
		"import.html", "settings.html", "settings_new.html", "uom_settings_new.html",
		"receipt_scanner.html",
	}

	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"templates/base.html", "templates/" + page}
		if page == "brands.html" {
			files = append(files, "templates/brand_row.html", "templates/brand_rows.html", "templates/brand_edit_row.html", "templates/brands_results.html")
		} else if page == "categories.html" {
			files = append(files, "templates/category_row.html", "templates/category_rows.html", "templates/category_edit_row.html", "templates/categories_results.html")
		} else if page == "items.html" {
			files = append(files, "templates/item_row.html", "templates/item_rows.html", "templates/item_edit_row.html", "templates/brand_select.html", "templates/category_select.html", "templates/uom_select.html", "templates/items_results.html")
		} else if page == "uom_settings.html" {
			files = append(files, "templates/uom_setting_row.html", "templates/uom_setting_rows.html", "templates/uom_setting_edit_row.html", "templates/uom_settings_results.html")
		} else if page == "inventory.html" {
			files = append(files, "templates/inventory_stock_rows.html", "templates/receiving_item_row.html", "templates/adjustment_item_row.html", "templates/receive_stock_form.html", "templates/adjust_stock_form.html")
		} else if page == "stock_receiving.html" {
			files = append(files, "templates/receiving_rows.html", "templates/item_select.html", "templates/receiving_item_row.html", "templates/receive_stock_form.html")
		} else if page == "stock_adjustments.html" {
			files = append(files, "templates/adjustment_rows.html", "templates/item_select.html", "templates/adjustment_item_row.html", "templates/adjust_stock_form.html")
		} else if page == "sales.html" {
			files = append(files, "templates/sales_rows.html", "templates/sale_item_row.html", "templates/sales_results.html", "templates/item_select.html", "templates/uom_select.html", "templates/sale_row.html", "templates/sale_edit_row.html")
		} else if page == "item_new.html" {
			files = append(files, "templates/brand_select.html", "templates/category_select.html", "templates/uom_select.html")
		} else if page == "sales_new.html" {
			files = append(files, "templates/item_select.html", "templates/sale_item_row.html")
		} else if page == "monthly_inventory.html" {
			files = append(files, "templates/monthly_inventory_rows.html")
		} else if page == "receiving_logs.html" {
			files = append(files, "templates/receiving_rows.html", "templates/receiving_logs_results.html", "templates/receiving_log_row.html", "templates/receiving_item_row.html", "templates/receive_stock_form.html")
		} else if page == "adjustment_logs.html" {
			files = append(files, "templates/adjustment_rows.html", "templates/adjustment_item_row.html", "templates/adjust_stock_form.html")
		} else if page == "settings.html" {
			files = append(files,
				"templates/brand_row.html", "templates/brand_rows.html", "templates/brand_edit_row.html",
				"templates/category_row.html", "templates/category_rows.html", "templates/category_edit_row.html",
				"templates/uom_setting_row.html", "templates/uom_setting_rows.html", "templates/uom_setting_edit_row.html", "templates/uom_settings_results.html",
				"templates/uom_row.html", "templates/uom_rows.html", "templates/uoms_results.html", "templates/uom_select.html",
			)
		} else if page == "settings_new.html" || page == "uom_settings_new.html" {
			files = append(files, "templates/item_select.html")
		}

		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	fragments := []string{
		"brand_row.html", "brand_rows.html", "brand_edit_row.html",
		"category_row.html", "category_rows.html", "category_edit_row.html",
		"item_row.html", "item_rows.html", "item_edit_row.html",
		"receiving_rows.html", "adjustment_rows.html", "sales_rows.html", "inventory_stock_rows.html",
		"brand_select.html", "category_select.html", "item_select.html",
		"sale_item_row.html", "receiving_item_row.html", "adjustment_item_row.html",
		"receive_stock_form.html", "adjust_stock_form.html",
		"monthly_inventory_rows.html",
		"brands_results.html", "categories_results.html", "items_results.html",
		"inventory_stock_results.html", "sales_results.html", "monthly_inventory_results.html",
		"receiving_logs_results.html", "adjustment_logs_results.html",
		"receiving_log_edit_row.html", "receiving_log_row.html",
		"uom_setting_row.html", "uom_setting_rows.html", "uom_setting_edit_row.html", "uom_settings_results.html",
		"uom_row.html", "uom_rows.html", "uoms_results.html", "uom_select.html",
		"receipt_scanner_results.html",
		"sale_row.html", "sale_edit_row.html",
	}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcMap)
		files := []string{"templates/" + frag}
		if frag == "brand_rows.html" {
			files = append(files, "templates/brand_row.html")
		} else if frag == "category_rows.html" {
			files = append(files, "templates/category_row.html")
		} else if frag == "item_rows.html" {
			files = append(files, "templates/item_row.html")
		} else if frag == "brands_results.html" {
			files = append(files, "templates/brand_row.html", "templates/brand_rows.html")
		} else if frag == "categories_results.html" {
			files = append(files, "templates/category_row.html", "templates/category_rows.html")
		} else if frag == "items_results.html" {
			files = append(files, "templates/item_row.html", "templates/item_rows.html")
		} else if frag == "inventory_stock_results.html" {
			files = append(files, "templates/inventory_stock_rows.html")
		} else if frag == "sales_rows.html" {
			files = append(files, "templates/sale_row.html", "templates/sale_edit_row.html")
		} else if frag == "sales_results.html" {
			files = append(files, "templates/sales_rows.html", "templates/sale_row.html", "templates/sale_edit_row.html")
		} else if frag == "monthly_inventory_results.html" {
			files = append(files, "templates/monthly_inventory_rows.html")
		} else if frag == "receiving_logs_results.html" {
			files = append(files, "templates/receiving_rows.html", "templates/receiving_log_row.html")
		} else if frag == "adjustment_logs_results.html" {
			files = append(files, "templates/adjustment_rows.html")
		} else if frag == "uom_setting_rows.html" {
			files = append(files, "templates/uom_setting_row.html")
		} else if frag == "uom_settings_results.html" {
			files = append(files, "templates/uom_setting_row.html", "templates/uom_setting_rows.html")
		} else if frag == "uom_rows.html" {
			files = append(files, "templates/uom_row.html")
		} else if frag == "uoms_results.html" {
			files = append(files, "templates/uom_row.html", "templates/uom_rows.html")
		}
		t = template.Must(t.ParseFiles(files...))
		templates[frag] = t
	}

	return templates
}
