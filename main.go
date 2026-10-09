package main

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"

	"radline/db"
	"radline/domain"
	"radline/handlers"
)

func main() {
	// Initialize Database (PostgreSQL via DATABASE_URL in production, SQLite locally)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		if os.Getenv("APP_ENV") == "production" {
			log.Fatalf("DATABASE_URL environment variable is required when APP_ENV=production")
		}
		dbURL = "backoffice.db"
	}

	err := db.InitDB(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Parse Templates
	templates := parseTemplates()

	// Initialize App Handlers
	app := &handlers.App{
		Templates: templates,
	}

	mux := http.NewServeMux()

	// Serve Static Files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// PWA Endpoints
	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Service-Worker-Allowed", "/")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "static/sw.js")
	})

	mux.HandleFunc("GET /manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeFile(w, r, "static/manifest.webmanifest")
	})

	mux.HandleFunc("GET /offline.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, "static/offline.html")
	})

	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/icons/icon-192.png")
	})

	// Public Authentication Routes
	mux.HandleFunc("GET /login", app.LoginPageHandler)
	mux.HandleFunc("POST /login", app.LoginSubmitHandler)
	mux.HandleFunc("POST /logout", app.LogoutHandler)
	mux.HandleFunc("GET /logout", app.LogoutHandler)
	mux.HandleFunc("GET /auth/user-chip", app.UserChipHandler)

	// Protected Dashboard
	mux.HandleFunc("/", app.RequirePermission(domain.PermDashboardView, app.DashboardHandler))

	// Roles & Users Management (RBAC)
	mux.HandleFunc("GET /settings/roles", app.RequirePermission(domain.PermSettingsRoles, app.RolesHandler))
	mux.HandleFunc("POST /settings/roles/add", app.RequirePermission(domain.PermSettingsRoles, app.AddRoleHandler))
	mux.HandleFunc("POST /settings/roles/update/{id}", app.RequirePermission(domain.PermSettingsRoles, app.UpdateRoleHandler))
	mux.HandleFunc("DELETE /settings/roles/delete/{id}", app.RequirePermission(domain.PermSettingsRoles, app.DeleteRoleHandler))

	mux.HandleFunc("GET /settings/users", app.RequirePermission(domain.PermSettingsUsers, app.UsersHandler))
	mux.HandleFunc("POST /settings/users/add", app.RequirePermission(domain.PermSettingsUsers, app.AddUserHandler))
	mux.HandleFunc("POST /settings/users/update/{id}", app.RequirePermission(domain.PermSettingsUsers, app.UpdateUserHandler))

	// Activity / Audit Logs
	mux.HandleFunc("GET /activity-logs", app.RequirePermission(domain.PermLogsView, app.ActivityLogsHandler))

	// Brands CRUD
	mux.HandleFunc("GET /brands", app.RequirePermission(domain.PermItemsView, app.BrandsHandler))
	mux.HandleFunc("GET /brands/new", app.RequirePermission(domain.PermItemsEdit, app.NewBrandPageHandler))
	mux.HandleFunc("GET /brands/select", app.RequirePermission(domain.PermItemsView, app.SelectBrandsHandler))
	mux.HandleFunc("POST /brands/add", app.RequirePermission(domain.PermItemsEdit, app.AddBrandHandler))
	mux.HandleFunc("GET /brands/edit/{id}", app.RequirePermission(domain.PermItemsEdit, app.EditBrandFormHandler))
	mux.HandleFunc("POST /brands/edit/{id}", app.RequirePermission(domain.PermItemsEdit, app.UpdateBrandHandler))
	mux.HandleFunc("GET /brands/row/{id}", app.RequirePermission(domain.PermItemsView, app.BrandRowHandler))
	mux.HandleFunc("DELETE /brands/delete/{id}", app.RequirePermission(domain.PermItemsEdit, app.DeleteBrandHandler))

	// Categories CRUD
	mux.HandleFunc("GET /categories", app.RequirePermission(domain.PermItemsView, app.CategoriesHandler))
	mux.HandleFunc("GET /categories/new", app.RequirePermission(domain.PermItemsEdit, app.NewCategoryPageHandler))
	mux.HandleFunc("GET /categories/select", app.RequirePermission(domain.PermItemsView, app.SelectCategoriesHandler))
	mux.HandleFunc("POST /categories/add", app.RequirePermission(domain.PermItemsEdit, app.AddCategoryHandler))
	mux.HandleFunc("GET /categories/edit/{id}", app.RequirePermission(domain.PermItemsEdit, app.EditCategoryFormHandler))
	mux.HandleFunc("POST /categories/edit/{id}", app.RequirePermission(domain.PermItemsEdit, app.UpdateCategoryHandler))
	mux.HandleFunc("GET /categories/row/{id}", app.RequirePermission(domain.PermItemsView, app.CategoryRowHandler))
	mux.HandleFunc("DELETE /categories/delete/{id}", app.RequirePermission(domain.PermItemsEdit, app.DeleteCategoryHandler))

	// Items CRUD
	mux.HandleFunc("GET /items", app.RequirePermission(domain.PermItemsView, app.ItemsHandler))
	mux.HandleFunc("GET /items/new", app.RequirePermission(domain.PermItemsEdit, app.NewItemPageHandler))
	mux.HandleFunc("GET /items/select", app.RequirePermission(domain.PermItemsView, app.SelectItemsHandler))
	mux.HandleFunc("GET /items/search", app.RequirePermission(domain.PermItemsView, app.SearchItemsHandler))
	mux.HandleFunc("POST /items/add", app.RequirePermission(domain.PermItemsEdit, app.AddItemHandler))
	mux.HandleFunc("GET /items/edit/{id}", app.RequirePermission(domain.PermItemsEdit, app.EditItemFormHandler))
	mux.HandleFunc("POST /items/edit/{id}", app.RequirePermission(domain.PermItemsEdit, app.UpdateItemHandler))
	mux.HandleFunc("GET /items/row/{id}", app.RequirePermission(domain.PermItemsView, app.ItemRowHandler))
	mux.HandleFunc("DELETE /items/delete/{id}", app.RequirePermission(domain.PermItemsEdit, app.DeleteItemHandler))

	// Inventory (Landing, Receiving & Adjustments)
	mux.HandleFunc("GET /inventory", app.RequirePermission(domain.PermInventoryView, app.InventoryHandler))
	mux.HandleFunc("GET /inventory/stock", app.RequirePermission(domain.PermInventoryView, app.InventoryHandler))
	mux.HandleFunc("GET /inventory/monthly", app.RequirePermission(domain.PermInventoryView, app.MonthlyInventoryHandler))
	mux.HandleFunc("GET /inventory/receiving", app.RequirePermission(domain.PermInventoryReceive, app.StockReceivingPageHandler))
	mux.HandleFunc("GET /inventory/receiving/logs", app.RequirePermission(domain.PermInventoryView, app.ReceivingLogsHandler))
	mux.HandleFunc("POST /inventory/receiving/add", app.RequirePermission(domain.PermInventoryReceive, app.ReceiveStockHandler))
	mux.HandleFunc("GET /inventory/receiving/new-row", app.RequirePermission(domain.PermInventoryReceive, app.NewReceivingRowHandler))
	mux.HandleFunc("GET /inventory/receiving/item-row-details", app.RequirePermission(domain.PermInventoryReceive, app.ReceivingItemRowDetailsHandler))
	mux.HandleFunc("DELETE /inventory/receiving/logs/delete/{id}", app.RequirePermission(domain.PermInventoryReceive, app.DeleteReceivingLogHandler))
	mux.HandleFunc("GET /inventory/receiving/logs/edit/{id}", app.RequirePermission(domain.PermInventoryReceive, app.EditReceivingLogHandler))
	mux.HandleFunc("POST /inventory/receiving/logs/edit/{id}", app.RequirePermission(domain.PermInventoryReceive, app.UpdateReceivingLogHandler))
	mux.HandleFunc("GET /inventory/adjustments", app.RequirePermission(domain.PermInventoryAdjust, app.StockAdjustmentsPageHandler))
	mux.HandleFunc("GET /inventory/adjustments/logs", app.RequirePermission(domain.PermInventoryView, app.AdjustmentLogsHandler))
	mux.HandleFunc("POST /inventory/adjustments/add", app.RequirePermission(domain.PermInventoryAdjust, app.AdjustStockHandler))
	mux.HandleFunc("GET /inventory/adjustments/new-row", app.RequirePermission(domain.PermInventoryAdjust, app.NewAdjustmentRowHandler))
	mux.HandleFunc("GET /inventory/adjustments/item-row-details", app.RequirePermission(domain.PermInventoryAdjust, app.AdjustmentItemRowDetailsHandler))

	// Sales Routing
	mux.HandleFunc("GET /sales", app.RequirePermission(domain.PermSalesView, app.SalesHandler))
	mux.HandleFunc("GET /sales/new", app.RequirePermission(domain.PermSalesCreate, app.NewSalesPageHandler))
	mux.HandleFunc("POST /sales/add", app.RequirePermission(domain.PermSalesCreate, app.AddSalesHandler))
	mux.HandleFunc("POST /sales/edit/{id}", app.RequirePermission(domain.PermSalesEdit, app.UpdateSalesHandler))
	mux.HandleFunc("POST /sales/update-status/{id}", app.RequirePermission(domain.PermSalesEdit, app.UpdateSalesStatusHandler))
	mux.HandleFunc("GET /sales/new-row", app.RequirePermission(domain.PermSalesCreate, app.NewSaleRowHandler))
	mux.HandleFunc("DELETE /sales/delete/{id}", app.RequirePermission(domain.PermSalesDelete, app.DeleteSalesHandler))
	mux.HandleFunc("GET /sales/item-row-details", app.RequirePermission(domain.PermSalesView, app.SaleItemRowDetailsHandler))

	// Settings CRUD
	mux.HandleFunc("GET /settings", app.RequirePermission(domain.PermSettingsView, app.SettingsHandler))
	mux.HandleFunc("POST /settings/deepseek", app.RequirePermission(domain.PermSettingsEdit, app.SaveDeepSeekConfigHandler))
	mux.HandleFunc("GET /uom-settings/new", app.RequirePermission(domain.PermSettingsEdit, app.NewUomSettingPageHandler))
	mux.HandleFunc("POST /uom-settings/add", app.RequirePermission(domain.PermSettingsEdit, app.AddUomSettingHandler))
	mux.HandleFunc("GET /uom-settings/edit/{id}", app.RequirePermission(domain.PermSettingsEdit, app.EditUomSettingFormHandler))
	mux.HandleFunc("POST /uom-settings/edit/{id}", app.RequirePermission(domain.PermSettingsEdit, app.UpdateUomSettingHandler))
	mux.HandleFunc("GET /uom-settings/row/{id}", app.RequirePermission(domain.PermSettingsView, app.UomSettingRowHandler))
	mux.HandleFunc("DELETE /uom-settings/delete/{id}", app.RequirePermission(domain.PermSettingsEdit, app.DeleteUomSettingHandler))

	// UOM Predefined List CRUD
	mux.HandleFunc("GET /uoms", app.RequirePermission(domain.PermSettingsView, app.UomsHandler))
	mux.HandleFunc("POST /uoms/add", app.RequirePermission(domain.PermSettingsEdit, app.AddUomHandler))
	mux.HandleFunc("GET /uoms/row/{id}", app.RequirePermission(domain.PermSettingsView, app.UomRowHandler))
	mux.HandleFunc("DELETE /uoms/delete/{id}", app.RequirePermission(domain.PermSettingsEdit, app.DeleteUomHandler))
	mux.HandleFunc("GET /uoms/select", app.RequirePermission(domain.PermSettingsView, app.SelectUomsHandler))

	// Import Routing
	mux.HandleFunc("GET /import", app.RequirePermission(domain.PermToolsImport, app.ImportPageHandler))
	mux.HandleFunc("POST /import/upload", app.RequirePermission(domain.PermToolsImport, app.ImportUploadHandler))

	// Earning Goals Routing
	mux.HandleFunc("GET /goals", app.RequirePermission(domain.PermDashboardView, app.GoalsHandler))
	mux.HandleFunc("POST /goals/add", app.RequirePermission(domain.PermDashboardView, app.AddGoalHandler))
	mux.HandleFunc("DELETE /goals/delete/{id}", app.RequirePermission(domain.PermDashboardView, app.DeleteGoalHandler))

	// Reminders & Calendar Routing
	mux.HandleFunc("GET /calendar", app.RequirePermission(domain.PermDashboardView, app.RemindersHandler))
	mux.HandleFunc("GET /reminders", app.RequirePermission(domain.PermDashboardView, app.RemindersHandler))
	mux.HandleFunc("POST /reminders/add", app.RequirePermission(domain.PermDashboardView, app.AddReminderHandler))
	mux.HandleFunc("POST /calendar/add", app.RequirePermission(domain.PermDashboardView, app.AddReminderHandler))
	mux.HandleFunc("POST /reminders/edit/{id}", app.RequirePermission(domain.PermDashboardView, app.EditReminderHandler))
	mux.HandleFunc("POST /calendar/edit/{id}", app.RequirePermission(domain.PermDashboardView, app.EditReminderHandler))
	mux.HandleFunc("POST /reminders/toggle/{id}", app.RequirePermission(domain.PermDashboardView, app.ToggleReminderHandler))
	mux.HandleFunc("POST /calendar/toggle/{id}", app.RequirePermission(domain.PermDashboardView, app.ToggleReminderHandler))
	mux.HandleFunc("DELETE /reminders/delete/{id}", app.RequirePermission(domain.PermDashboardView, app.DeleteReminderHandler))
	mux.HandleFunc("DELETE /calendar/delete/{id}", app.RequirePermission(domain.PermDashboardView, app.DeleteReminderHandler))
	mux.HandleFunc("POST /reminders/reschedule/{id}", app.RequirePermission(domain.PermDashboardView, app.RescheduleReminderHandler))
	mux.HandleFunc("POST /calendar/reschedule/{id}", app.RequirePermission(domain.PermDashboardView, app.RescheduleReminderHandler))

	// Receipt Scanner Routing
	mux.HandleFunc("GET /tools/receipt-scanner", app.RequirePermission(domain.PermToolsScanner, app.ReceiptScannerHandler))
	mux.HandleFunc("POST /tools/receipt-scanner/parse", app.RequirePermission(domain.PermToolsScanner, app.ReceiptScannerParseHandler))

	// Forms Routing (Quotation & Purchase Order)
	mux.HandleFunc("GET /forms/quotation", app.RequirePermission(domain.PermSalesView, app.QuotationFormHandler))
	mux.HandleFunc("GET /forms/purchase-order", app.RequirePermission(domain.PermInventoryView, app.PurchaseOrderFormHandler))
	mux.HandleFunc("POST /forms/purchase-order/save", app.RequirePermission(domain.PermInventoryView, app.SavePurchaseOrderHandler))

	// Wrap entire router with AuthMiddleware
	handler := app.AuthMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func parseTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
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
		"receipt_scanner.html", "roles.html", "users.html", "activity_logs.html",
		"goals.html", "reminders.html",
		"form_quotation.html", "form_purchase_order.html",
	}

	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"templates/base.html", "templates/" + page}
		if page == "dashboard.html" {
			files = append(files, "templates/dashboard_trends.html")
		} else if page == "brands.html" {
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
		} else if page == "activity_logs.html" {
			files = append(files, "templates/activity_logs_results.html")
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
		"activity_logs_results.html",
		"dashboard_trends.html",
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

	// Standalone Login Template
	templates["login.html"] = template.Must(template.New("login.html").Funcs(funcMap).ParseFiles("templates/login.html"))

	return templates
}
