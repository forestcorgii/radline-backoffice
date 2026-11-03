package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"radline/db"
	"radline/models"
)

func TestDefaultDashboardCards(t *testing.T) {
	cards := DefaultDashboardCards()
	if len(cards) != 9 {
		t.Fatalf("Expected 9 default cards, got %d", len(cards))
	}

	expectedIDs := []string{
		"card-workflow",
		"card-kpi-sales",
		"card-kpi-costs",
		"card-kpi-profit",
		"card-kpi-margin",
		"card-trends",
		"card-top-products",
		"card-goals",
		"card-reminders",
	}

	for i, expected := range expectedIDs {
		if cards[i].ID != expected {
			t.Errorf("Card at index %d expected %s, got %s", i, expected, cards[i].ID)
		}
	}
}

func TestGetUserDashboardCards(t *testing.T) {
	_ = db.InitDB(":memory:")
	app := &App{}

	// Case 1: user with no preference returns default layout
	cards := app.GetUserDashboardCards(999)
	if len(cards) != 9 {
		t.Fatalf("Expected 9 cards, got %d", len(cards))
	}

	// Case 2: user with custom preference (use seeded user ID 1 = admin)
	customOrder := []models.DashboardCardConfig{
		{ID: "card-trends", Width: "half", Height: "large"},
		{ID: "card-workflow", Width: "full", Height: "small"},
	}
	bytesVal, _ := json.Marshal(customOrder)
	err := db.SetUserPreference(1, "dashboard_layout", string(bytesVal))
	if err != nil {
		t.Fatalf("Failed to set user preference: %v", err)
	}

	customCards := app.GetUserDashboardCards(1)
	if len(customCards) != 9 {
		t.Fatalf("Expected 9 total cards (with missing cards appended), got %d", len(customCards))
	}
	if customCards[0].ID != "card-trends" || customCards[0].Width != "half" || customCards[0].Height != "large" {
		t.Errorf("Expected first card to be card-trends (half, large), got %+v", customCards[0])
	}
	if customCards[1].ID != "card-workflow" {
		t.Errorf("Expected second card to be card-workflow, got %s", customCards[1].ID)
	}
}

func TestSaveAndResetDashboardLayoutHandler(t *testing.T) {
	_ = db.InitDB(":memory:")
	app := &App{}

	// 1. Unauthorized
	req := httptest.NewRequest(http.MethodPost, "/dashboard/layout", bytes.NewBufferString("[]"))
	rec := httptest.NewRecorder()
	app.SaveDashboardLayoutHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", rec.Code)
	}

	// 2. Authorized POST Save (use seeded user ID 1)
	userCtx := &CurrentUserContext{
		User: models.User{
			ID:       1,
			Username: "admin",
		},
	}
	payload := []models.DashboardCardConfig{
		{ID: "card-kpi-profit", Width: "half", Height: "small"},
		{ID: "card-trends", Width: "full", Height: "large"},
	}
	payloadBytes, _ := json.Marshal(payload)

	req = httptest.NewRequest(http.MethodPost, "/dashboard/layout", bytes.NewBuffer(payloadBytes))
	req = req.WithContext(context.WithValue(req.Context(), UserContextKey, userCtx))
	rec = httptest.NewRecorder()

	app.SaveDashboardLayoutHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify in DB
	prefVal, err := db.GetUserPreference(1, "dashboard_layout")
	if err != nil || prefVal == "" {
		t.Fatalf("Expected saved preference in DB, got error: %v", err)
	}

	// 3. Authorized POST Reset
	reqReset := httptest.NewRequest(http.MethodPost, "/dashboard/layout/reset", nil)
	reqReset = reqReset.WithContext(context.WithValue(reqReset.Context(), UserContextKey, userCtx))
	reqReset.Header.Set("HX-Request", "true")
	recReset := httptest.NewRecorder()

	app.ResetDashboardLayoutHandler(recReset, reqReset)
	if recReset.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on HTMX reset, got %d", recReset.Code)
	}
	if recReset.Header().Get("HX-Refresh") != "true" {
		t.Errorf("Expected HX-Refresh: true header on reset")
	}

	// Verify preference deleted from DB
	prefValAfter, _ := db.GetUserPreference(1, "dashboard_layout")
	if prefValAfter != "" {
		t.Errorf("Expected preference deleted from DB, got %q", prefValAfter)
	}
}

func TestDashboardTrendsWithCost(t *testing.T) {
	_ = db.InitDB(":memory:")

	tmpl, err := template.ParseFiles("../templates/dashboard_trends.html")
	if err != nil {
		t.Fatalf("Failed to parse dashboard_trends.html: %v", err)
	}

	app := &App{
		Templates: map[string]*template.Template{
			"dashboard_trends.html": tmpl,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/?fragment=trends", nil)
	req.Header.Set("HX-Request", "true")
	userCtx := &CurrentUserContext{
		User: models.User{ID: 1, Username: "admin", RoleID: 1},
	}
	req = req.WithContext(context.WithValue(req.Context(), UserContextKey, userCtx))
	rec := httptest.NewRecorder()

	app.DashboardHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for trends fragment, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "toggle-chart-cost") {
		t.Errorf("Expected rendered trends fragment to contain toggle-chart-cost checkbox")
	}
	if !strings.Contains(body, "chart-bar-cost") {
		t.Errorf("Expected rendered trends fragment to contain chart-bar-cost SVG bars")
	}
	if !strings.Contains(body, "toggleChartSeries('cost'") {
		t.Errorf("Expected rendered trends fragment to contain toggleChartSeries call for cost")
	}
}

func TestDashboardTopProductsFilter(t *testing.T) {
	_ = db.InitDB(":memory:")

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	tmpl, err := template.New("dashboard_top_products.html").Funcs(funcMap).ParseFiles("../templates/dashboard_top_products.html")
	if err != nil {
		t.Fatalf("Failed to parse dashboard_top_products.html: %v", err)
	}

	app := &App{
		Templates: map[string]*template.Template{
			"dashboard_top_products.html": tmpl,
		},
	}

	userCtx := &CurrentUserContext{
		User: models.User{ID: 1, Username: "admin", RoleID: 1},
	}

	// Seed test items & sales_details
	if _, err := db.DB.Exec(`INSERT INTO items (id, code, description, default_uom) VALUES (101, 'ITEM-VOL', 'High Volume Item', 'PCS')`); err != nil {
		t.Fatalf("Failed to insert item 101: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO items (id, code, description, default_uom) VALUES (102, 'ITEM-PROF', 'High Profit Item', 'PCS')`); err != nil {
		t.Fatalf("Failed to insert item 102: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO items (id, code, description, default_uom) VALUES (103, 'ITEM-SALES', 'High Sales Item', 'PCS')`); err != nil {
		t.Fatalf("Failed to insert item 103: %v", err)
	}

	if _, err := db.DB.Exec(`INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit) VALUES ('DR', 'Posted', '2026-01-15', 'DR-001', 'Supp1', 101, 500, 'PCS', 10.0, 5000.0, 9.0, 4500.0, 500.0)`); err != nil {
		t.Fatalf("Failed to insert sales_detail 101: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit) VALUES ('DR', 'Posted', '2026-01-15', 'DR-002', 'Supp1', 102, 10, 'PCS', 1500.0, 15000.0, 700.0, 7000.0, 8000.0)`); err != nil {
		t.Fatalf("Failed to insert sales_detail 102: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO sales_details (doc_type, doc_status, doc_date, doc_number, supplier, item_id, qty, uom, price, total_sales, cost, total_cost, profit) VALUES ('DR', 'Posted', '2026-01-15', 'DR-003', 'Supp1', 103, 5, 'PCS', 10000.0, 50000.0, 9600.0, 48000.0, 2000.0)`); err != nil {
		t.Fatalf("Failed to insert sales_detail 103: %v", err)
	}

	// 1. Default (Volume) - Item 101 should appear first
	reqVol := httptest.NewRequest(http.MethodGet, "/?fragment=top_products&top_filter=volume", nil)
	reqVol.Header.Set("HX-Request", "true")
	reqVol = reqVol.WithContext(context.WithValue(reqVol.Context(), UserContextKey, userCtx))
	recVol := httptest.NewRecorder()
	app.DashboardHandler(recVol, reqVol)
	if recVol.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for volume filter, got %d. Body: %s", recVol.Code, recVol.Body.String())
	}
	bodyVol := recVol.Body.String()
	if !strings.Contains(bodyVol, "Highest Volume Bought") {
		t.Errorf("Expected body to contain 'Highest Volume Bought'")
	}
	if !strings.Contains(bodyVol, "High Volume Item") {
		t.Errorf("Expected body to contain 'High Volume Item'")
	}
	// Volume item should appear before profit and sales items
	idxVolItem := strings.Index(bodyVol, "High Volume Item")
	idxProfItem := strings.Index(bodyVol, "High Profit Item")
	if idxVolItem == -1 || (idxProfItem != -1 && idxVolItem > idxProfItem) {
		t.Errorf("Expected High Volume Item to be ranked first in volume view")
	}

	// 2. Most Profitable - Item 102 should appear first
	reqProfit := httptest.NewRequest(http.MethodGet, "/?fragment=top_products&top_filter=profit", nil)
	reqProfit.Header.Set("HX-Request", "true")
	reqProfit = reqProfit.WithContext(context.WithValue(reqProfit.Context(), UserContextKey, userCtx))
	recProfit := httptest.NewRecorder()
	app.DashboardHandler(recProfit, reqProfit)
	if recProfit.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for profit filter, got %d. Body: %s", recProfit.Code, recProfit.Body.String())
	}
	bodyProfit := recProfit.Body.String()
	if !strings.Contains(bodyProfit, "Most Profitable Items") {
		t.Errorf("Expected body to contain 'Most Profitable Items'")
	}
	idxProfItemInProfit := strings.Index(bodyProfit, "High Profit Item")
	idxVolItemInProfit := strings.Index(bodyProfit, "High Volume Item")
	if idxProfItemInProfit == -1 || (idxVolItemInProfit != -1 && idxProfItemInProfit > idxVolItemInProfit) {
		t.Errorf("Expected High Profit Item to be ranked first in profit view")
	}

	// 3. Highest Sales - Item 103 should appear first
	reqSales := httptest.NewRequest(http.MethodGet, "/?fragment=top_products&top_filter=sales", nil)
	reqSales.Header.Set("HX-Request", "true")
	reqSales = reqSales.WithContext(context.WithValue(reqSales.Context(), UserContextKey, userCtx))
	recSales := httptest.NewRecorder()
	app.DashboardHandler(recSales, reqSales)
	if recSales.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for sales filter, got %d. Body: %s", recSales.Code, recSales.Body.String())
	}
	bodySales := recSales.Body.String()
	if !strings.Contains(bodySales, "Highest Sales Revenue") {
		t.Errorf("Expected body to contain 'Highest Sales Revenue'")
	}
	idxSalesItemInSales := strings.Index(bodySales, "High Sales Item")
	idxProfItemInSales := strings.Index(bodySales, "High Profit Item")
	if idxSalesItemInSales == -1 || (idxProfItemInSales != -1 && idxSalesItemInSales > idxProfItemInSales) {
		t.Errorf("Expected High Sales Item to be ranked first in sales view")
	}
}
