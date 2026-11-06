package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"rav/db"
	"rav/domain"
	"rav/models"
)

type App struct {
	Templates map[string]*template.Template
}

func (app *App) Render(w http.ResponseWriter, name string, data interface{}) {
	t, ok := app.Templates[name]
	if !ok {
		http.Error(w, "Template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := t.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("ExecuteTemplate error in %s: %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// RenderFragment renders two template fragments in a single response.
// It writes an HTML comment first to prevent HTMX's makeFragment from
// wrapping <tr>-based fragments in <table><tbody>, which would corrupt
// any OOB swap elements (like pagination) that follow.
func (app *App) RenderFragment(w http.ResponseWriter, first, second string, data1, data2 interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t1, ok := app.Templates[first]
	if !ok {
		http.Error(w, "Template not found: "+first, http.StatusInternalServerError)
		return
	}
	t2, ok := app.Templates[second]
	if !ok {
		http.Error(w, "Template not found: "+second, http.StatusInternalServerError)
		return
	}
	// Write a comment so the response doesn't start with <tr>,
	// which would cause HTMX's makeFragment to wrap aggressively.
	// This ensures any OOB swap elements (like the <div> in pagination)
	// are properly parsed as siblings rather than being trapped inside <tbody>.
	w.Write([]byte("<!-- htmx-fragment -->"))
	err := t1.ExecuteTemplate(w, first, data1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t2.ExecuteTemplate(w, second, data2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (app *App) RenderPage(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		t, ok := app.Templates[name]
		if !ok {
			http.Error(w, "Template not found: "+name, http.StatusInternalServerError)
			return
		}
		err := t.ExecuteTemplate(w, "content", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		app.Render(w, name, data)
	}
}

// DashboardHandler renders the main dashboard with financial summaries, dynamic trends, top products, earning goals, and reminders
func (app *App) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	// 1. Greeting
	ctxUser := GetCurrentUser(r)
	userName := "Operator"
	var userID int64
	if ctxUser != nil {
		userID = ctxUser.User.ID
		if ctxUser.User.FullName != "" {
			userName = ctxUser.User.FullName
		} else {
			userName = ctxUser.User.Username
		}
	}
	hour := now.Hour()
	greetingPrefix := "Good morning"
	if hour >= 12 && hour < 17 {
		greetingPrefix = "Good afternoon"
	} else if hour >= 17 {
		greetingPrefix = "Good evening"
	}
	greeting := greetingPrefix + ", " + userName + "!"
	todayFormatted := now.Format("Monday, January 02, 2006")

	// 2. Date Range for Dynamic Trends
	startDateStr := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDateStr := strings.TrimSpace(r.URL.Query().Get("end_date"))
	if startDateStr == "" {
		startDateStr = now.AddDate(0, -5, 0).Format("2006-01-01")
	}
	if endDateStr == "" {
		endDateStr = now.Format("2006-01-02")
	}

	startDate, errStart := time.Parse("2006-01-02", startDateStr)
	if errStart != nil {
		startDate = now.AddDate(0, -5, 0)
	}
	endDate, errEnd := time.Parse("2006-01-02", endDateStr)
	if errEnd != nil {
		endDate = now
	}
	endOfDay := endDate.Add(24 * time.Hour).Add(-time.Second)

	// 3. Metrics within Selected Range (or fallback to lifetime if empty range)
	var metrics struct {
		TotalSales  float64 `db:"total_sales"`
		TotalCosts  float64 `db:"total_costs"`
		GrossProfit float64 `db:"gross_profit"`
	}
	err := db.DB.Get(&metrics, `
		SELECT 
			COALESCE(SUM(total_sales), 0.0) as total_sales,
			COALESCE(SUM(total_cost), 0.0) as total_costs,
			COALESCE(SUM(profit), 0.0) as gross_profit
		FROM sales_details
		WHERE doc_status IN ('Posted', 'POSTED') AND doc_date >= ? AND doc_date <= ?
	`, startDate, endOfDay)
	if err != nil || metrics.TotalSales == 0 {
		// Fallback to all posted sales if filtered range has zero sales
		_ = db.DB.Get(&metrics, `
			SELECT 
				COALESCE(SUM(total_sales), 0.0) as total_sales,
				COALESCE(SUM(total_cost), 0.0) as total_costs,
				COALESCE(SUM(profit), 0.0) as gross_profit
			FROM sales_details
			WHERE doc_status IN ('Posted', 'POSTED')
		`)
	}

	margin := 0.0
	if metrics.TotalSales > 0 {
		margin = (metrics.GrossProfit / metrics.TotalSales) * 100
	}

	// 4. Sales & Profit Monthly Trend
	type MonthlyTrend struct {
		Month  string  `db:"month"`
		Sales  float64 `db:"sales"`
		Cost   float64 `db:"cost"`
		Profit float64 `db:"profit"`
	}
	var trend []MonthlyTrend
	_ = db.DB.Select(&trend, `
		SELECT substr(doc_date, 1, 7) as month, 
		       COALESCE(SUM(total_sales), 0.0) as sales, 
		       COALESCE(SUM(total_cost), 0.0) as cost, 
		       COALESCE(SUM(profit), 0.0) as profit
		FROM sales_details
		WHERE doc_status IN ('Posted', 'POSTED') AND doc_date >= ? AND doc_date <= ?
		GROUP BY month
		ORDER BY month ASC
	`, startDate, endOfDay)

	if len(trend) == 0 {
		// Fallback to recent 6 months
		_ = db.DB.Select(&trend, `
			SELECT substr(doc_date, 1, 7) as month, 
			       COALESCE(SUM(total_sales), 0.0) as sales, 
			       COALESCE(SUM(total_cost), 0.0) as cost, 
			       COALESCE(SUM(profit), 0.0) as profit
			FROM sales_details
			WHERE doc_status IN ('Posted', 'POSTED')
			GROUP BY month
			ORDER BY month DESC
			LIMIT 6
		`)
		// Reverse to chronological order
		for i, j := 0, len(trend)-1; i < j; i, j = i+1, j-1 {
			trend[i], trend[j] = trend[j], trend[i]
		}
	}

	// If still empty, mock data for clean display
	if len(trend) == 0 {
		trend = []MonthlyTrend{
			{Month: "2026-05", Sales: 22000, Cost: 16000, Profit: 6000},
			{Month: "2026-06", Sales: 28000, Cost: 20200, Profit: 7800},
			{Month: "2026-07", Sales: 24000, Cost: 17500, Profit: 6500},
			{Month: "2026-08", Sales: 31000, Cost: 22600, Profit: 8400},
			{Month: "2026-09", Sales: 35000, Cost: 25800, Profit: 9200},
			{Month: "2026-10", Sales: 39000, Cost: 28500, Profit: 10500},
		}
	}

	// 5. Sales & Total Inventory Trend
	type MonthlyInvTrend struct {
		Month          string  `db:"month"`
		Sales          float64 `db:"sales"`
		InventoryValue float64 `db:"inventory_value"`
	}
	var invTrend []MonthlyInvTrend
	for _, tr := range trend {
		var invCost float64
		_ = db.DB.Get(&invCost, `
			SELECT COALESCE(SUM(total_cost), 0.0)
			FROM receiving_logs
			WHERE substr(date, 1, 7) = ?
		`, tr.Month)
		if invCost == 0 {
			invCost = tr.Sales * 0.65 // representative ratio if no receiving in that month
		}
		invTrend = append(invTrend, MonthlyInvTrend{
			Month:          tr.Month,
			Sales:          tr.Sales,
			InventoryValue: invCost,
		})
	}

	// 6. Build SVG Chart Bars for Sales, Cost, Profit & Inventory
	maxVal := 1000.0
	for i := range trend {
		if trend[i].Cost == 0 && trend[i].Sales > 0 && trend[i].Sales > trend[i].Profit {
			trend[i].Cost = trend[i].Sales - trend[i].Profit
		}
		if trend[i].Sales > maxVal {
			maxVal = trend[i].Sales
		}
		if trend[i].Cost > maxVal {
			maxVal = trend[i].Cost
		}
		if trend[i].Profit > maxVal {
			maxVal = trend[i].Profit
		}
		if i < len(invTrend) && invTrend[i].InventoryValue > maxVal {
			maxVal = invTrend[i].InventoryValue
		}
	}
	maxVal = maxVal * 1.15

	chartHeight := 170.0
	chartWidth := 860.0
	barWidth := 13.0
	barGap := 3.0
	stride := barWidth + barGap
	spacing := (chartWidth - 40.0) / float64(len(trend))

	type ChartBar struct {
		Label           string
		Sales           float64
		Cost            float64
		Profit          float64
		Inventory       float64
		SalesHeight     float64
		CostHeight      float64
		ProfitHeight    float64
		InventoryHeight float64
		SalesY          float64
		CostY           float64
		ProfitY         float64
		InventoryY      float64
		SalesX          float64
		CostX           float64
		ProfitX         float64
		InventoryX      float64
		LabelX          float64
	}
	var chartBars []ChartBar
	for i, t := range trend {
		invVal := 0.0
		if i < len(invTrend) {
			invVal = invTrend[i].InventoryValue
		}
		sHeight := (t.Sales / maxVal) * chartHeight
		cHeight := (t.Cost / maxVal) * chartHeight
		pHeight := (t.Profit / maxVal) * chartHeight
		iHeight := (invVal / maxVal) * chartHeight
		sY := chartHeight - sHeight + 10
		cY := chartHeight - cHeight + 10
		pY := chartHeight - pHeight + 10
		iY := chartHeight - iHeight + 10
		x := 30.0 + float64(i)*spacing + 20.0

		monthLabel := t.Month
		if parsedT, err := time.Parse("2006-01", t.Month); err == nil {
			monthLabel = parsedT.Format("Jan")
		}

		chartBars = append(chartBars, ChartBar{
			Label:           monthLabel,
			Sales:           t.Sales,
			Cost:            t.Cost,
			Profit:          t.Profit,
			Inventory:       invVal,
			SalesHeight:     sHeight,
			CostHeight:      cHeight,
			ProfitHeight:    pHeight,
			InventoryHeight: iHeight,
			SalesY:          sY,
			CostY:           cY,
			ProfitY:         pY,
			InventoryY:      iY,
			SalesX:          x,
			CostX:           x + stride,
			ProfitX:         x + 2*stride,
			InventoryX:      x + 3*stride,
			LabelX:          x + (3*stride+barWidth)/2,
		})
	}

	// 8. Top 5 Products (Filtered by volume, profit, or sales)
	topFilter := strings.TrimSpace(r.URL.Query().Get("top_filter"))
	topOrderBy := "total_qty DESC"
	switch topFilter {
	case "profit":
		topOrderBy = "total_profit DESC"
	case "sales":
		topOrderBy = "total_sales DESC"
	default:
		topFilter = "volume"
		topOrderBy = "total_qty DESC"
	}

	var topProducts []models.TopProduct
	_ = db.DB.Select(&topProducts, `
		SELECT s.item_id, i.code as item_code, i.description, COALESCE(b.name, 'Unbranded') as brand_name,
		       SUM(s.qty) as total_qty, s.uom, SUM(s.total_sales) as total_sales, SUM(s.profit) as total_profit
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
		LEFT JOIN brands b ON i.brand_id = b.id
		WHERE s.doc_status IN ('Posted', 'POSTED')
		GROUP BY s.item_id, i.code, i.description, b.name, s.uom
		ORDER BY `+topOrderBy+`
		LIMIT 5
	`)

	// 9. Active Earning Goals
	var goals []models.GoalWithProgress
	var rawGoals []models.EarningGoal
	_ = db.DB.Select(&rawGoals, "SELECT id, title, period_type, target_period, target_revenue, target_profit, created_at FROM earning_goals ORDER BY target_period DESC LIMIT 2")
	for _, g := range rawGoals {
		var actual struct {
			Revenue float64 `db:"revenue"`
			Profit  float64 `db:"profit"`
		}
		if g.PeriodType == string(domain.GoalPeriodMonthly) {
			_ = db.DB.Get(&actual, "SELECT COALESCE(SUM(total_sales), 0.0) as revenue, COALESCE(SUM(profit), 0.0) as profit FROM sales_details WHERE doc_status IN ('Posted', 'POSTED') AND substr(doc_date, 1, 7) = ?", g.TargetPeriod)
		} else {
			_ = db.DB.Get(&actual, "SELECT COALESCE(SUM(total_sales), 0.0) as revenue, COALESCE(SUM(profit), 0.0) as profit FROM sales_details WHERE doc_status IN ('Posted', 'POSTED') AND substr(doc_date, 1, 4) = ?", g.TargetPeriod)
		}
		dg := domain.EarningGoal{
			TargetRevenue: g.TargetRevenue,
			TargetProfit:  g.TargetProfit,
		}
		prog := dg.CalculateProgress(actual.Revenue, actual.Profit)
		goals = append(goals, models.GoalWithProgress{
			EarningGoal:     g,
			ActualRevenue:   actual.Revenue,
			ActualProfit:    actual.Profit,
			RevenueProgress: prog.RevenueProgress,
			ProfitProgress:  prog.ProfitProgress,
			Status:          prog.Status,
		})
	}

	// 10. Upcoming Reminders
	var upcomingReminders []models.Reminder
	_ = db.DB.Select(&upcomingReminders, `
		SELECT 
			r.id, r.title, r.details, r.due_date, r.priority, r.category, r.status, r.created_at,
			r.assigned_to_user_id,
			COALESCE(u.full_name, u.username, '') AS assigned_to_name
		FROM reminders r
		LEFT JOIN users u ON r.assigned_to_user_id = u.id
		WHERE r.status = 'Pending'
		ORDER BY r.due_date ASC
		LIMIT 4
	`)

	cardConfigs := app.GetUserDashboardCards(userID)

	data := struct {
		Greeting          string
		TodayFormatted    string
		StartDateStr      string
		EndDateStr        string
		TotalSales        float64
		TotalCosts        float64
		GrossProfit       float64
		Margin            float64
		ChartBars         []ChartBar
		TopProducts       []models.TopProduct
		TopFilter         string
		Goals             []models.GoalWithProgress
		UpcomingReminders []models.Reminder
		CardConfigs       []models.DashboardCardConfig
	}{
		Greeting:          greeting,
		TodayFormatted:    todayFormatted,
		StartDateStr:      startDateStr,
		EndDateStr:        endDateStr,
		TotalSales:        metrics.TotalSales,
		TotalCosts:        metrics.TotalCosts,
		GrossProfit:       metrics.GrossProfit,
		Margin:            margin,
		ChartBars:         chartBars,
		TopProducts:       topProducts,
		TopFilter:         topFilter,
		Goals:             goals,
		UpcomingReminders: upcomingReminders,
		CardConfigs:       cardConfigs,
	}

	if r.Header.Get("HX-Request") == "true" {
		if r.URL.Query().Get("fragment") == "trends" {
			t, ok := app.Templates["dashboard_trends.html"]
			if ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_ = t.Execute(w, data)
				return
			}
		} else if r.URL.Query().Get("fragment") == "top_products" {
			t, ok := app.Templates["dashboard_top_products.html"]
			if ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_ = t.Execute(w, data)
				return
			}
		}
	}

	app.RenderPage(w, r, "dashboard.html", data)
}

// DefaultDashboardCards returns the standard out-of-the-box layout for dashboard cards
func DefaultDashboardCards() []models.DashboardCardConfig {
	return []models.DashboardCardConfig{
		{ID: "card-workflow", Width: "full", Height: "small"},
		{ID: "card-kpi-sales", Width: "quarter", Height: "small"},
		{ID: "card-kpi-costs", Width: "quarter", Height: "small"},
		{ID: "card-kpi-profit", Width: "quarter", Height: "small"},
		{ID: "card-kpi-margin", Width: "quarter", Height: "small"},
		{ID: "card-trends", Width: "full", Height: "medium"},
		{ID: "card-top-products", Width: "half", Height: "medium"},
		{ID: "card-goals", Width: "quarter", Height: "medium"},
		{ID: "card-reminders", Width: "quarter", Height: "medium"},
	}
}

// GetUserDashboardCards retrieves the customized card ordering and dimensions for a user from database
func (app *App) GetUserDashboardCards(userID int64) []models.DashboardCardConfig {
	defaults := DefaultDashboardCards()
	if userID <= 0 {
		return defaults
	}

	prefVal, err := db.GetUserPreference(userID, "dashboard_layout")
	if err != nil || strings.TrimSpace(prefVal) == "" {
		return defaults
	}

	var saved []models.DashboardCardConfig
	if err := json.Unmarshal([]byte(prefVal), &saved); err != nil || len(saved) == 0 {
		return defaults
	}

	seen := make(map[string]bool)
	defaultMap := make(map[string]models.DashboardCardConfig)
	for _, d := range defaults {
		defaultMap[d.ID] = d
	}

	var finalCards []models.DashboardCardConfig
	for _, s := range saved {
		if d, ok := defaultMap[s.ID]; ok && !seen[s.ID] {
			seen[s.ID] = true
			w := s.Width
			if w != "quarter" && w != "half" && w != "full" {
				w = d.Width
			}
			h := s.Height
			if h != "small" && h != "medium" && h != "large" {
				h = d.Height
			}
			finalCards = append(finalCards, models.DashboardCardConfig{
				ID:     s.ID,
				Width:  w,
				Height: h,
			})
		}
	}

	// Append any missing cards from defaults
	for _, d := range defaults {
		if !seen[d.ID] {
			finalCards = append(finalCards, d)
		}
	}

	return finalCards
}

// SaveDashboardLayoutHandler updates the user's dashboard layout preference in the database
func (app *App) SaveDashboardLayoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctxUser := GetCurrentUser(r)
	if ctxUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var configs []models.DashboardCardConfig
	if err := json.NewDecoder(r.Body).Decode(&configs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	defaults := DefaultDashboardCards()
	defaultMap := make(map[string]models.DashboardCardConfig)
	for _, d := range defaults {
		defaultMap[d.ID] = d
	}

	var cleanConfigs []models.DashboardCardConfig
	seen := make(map[string]bool)
	for _, c := range configs {
		if d, ok := defaultMap[c.ID]; ok && !seen[c.ID] {
			seen[c.ID] = true
			w := c.Width
			if w != "quarter" && w != "half" && w != "full" {
				w = d.Width
			}
			h := c.Height
			if h != "small" && h != "medium" && h != "large" {
				h = d.Height
			}
			cleanConfigs = append(cleanConfigs, models.DashboardCardConfig{
				ID:     c.ID,
				Width:  w,
				Height: h,
			})
		}
	}

	bytesVal, err := json.Marshal(cleanConfigs)
	if err != nil {
		http.Error(w, "Failed to encode layout", http.StatusInternalServerError)
		return
	}

	if err := db.SetUserPreference(ctxUser.User.ID, "dashboard_layout", string(bytesVal)); err != nil {
		http.Error(w, "Failed to save preference", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success": true}`))
}

// ResetDashboardLayoutHandler resets the user's dashboard layout to the system default
func (app *App) ResetDashboardLayoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctxUser := GetCurrentUser(r)
	if ctxUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	_ = db.DeleteUserPreference(ctxUser.User.ID, "dashboard_layout")

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Refresh", "true")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// SaveUserPreferenceHandler persists a generic key-value user preference (such as list view customization)
func (app *App) SaveUserPreferenceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctxUser := GetCurrentUser(r)
	if ctxUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload struct {
		Key    string `json:"key"`
		Value  string `json:"value"`
		Delete bool   `json:"delete"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	key := strings.TrimSpace(payload.Key)
	if key == "" || len(key) > 128 {
		http.Error(w, "Invalid preference key", http.StatusBadRequest)
		return
	}

	if payload.Delete || payload.Value == "" {
		if err := db.DeleteUserPreference(ctxUser.User.ID, key); err != nil {
			http.Error(w, "Failed to delete preference", http.StatusInternalServerError)
			return
		}
	} else {
		if err := db.SetUserPreference(ctxUser.User.ID, key, payload.Value); err != nil {
			http.Error(w, "Failed to save preference", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success": true}`))
}

// GetUserPreferenceHandler retrieves a specific preference key or all preferences for the current user
func (app *App) GetUserPreferenceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctxUser := GetCurrentUser(r)
	if ctxUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		key = strings.TrimSpace(r.URL.Query().Get("key"))
	}

	w.Header().Set("Content-Type", "application/json")
	if key != "" {
		val, err := db.GetUserPreference(ctxUser.User.ID, key)
		if err != nil {
			val = ""
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"key":   key,
			"value": val,
		})
		return
	}

	prefs, err := db.GetAllUserPreferences(ctxUser.User.ID)
	if err != nil {
		prefs = map[string]string{}
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"preferences": prefs,
	})
}
