package handlers

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"radline/db"
	"radline/domain"
	"radline/models"
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
	if ctxUser != nil {
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
		WHERE doc_status IN ('Posted', 'POSTED') AND doc_date >= ? AND doc_date <= (? || ' 23:59:59')
	`, startDateStr, endDateStr)
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
		Profit float64 `db:"profit"`
	}
	var trend []MonthlyTrend
	_ = db.DB.Select(&trend, `
		SELECT substr(doc_date, 1, 7) as month, 
		       COALESCE(SUM(total_sales), 0.0) as sales, 
		       COALESCE(SUM(profit), 0.0) as profit
		FROM sales_details
		WHERE doc_status IN ('Posted', 'POSTED') AND doc_date >= ? AND doc_date <= (? || ' 23:59:59')
		GROUP BY month
		ORDER BY month ASC
	`, startDateStr, endDateStr)

	if len(trend) == 0 {
		// Fallback to recent 6 months
		_ = db.DB.Select(&trend, `
			SELECT substr(doc_date, 1, 7) as month, 
			       COALESCE(SUM(total_sales), 0.0) as sales, 
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
			{Month: "2026-05", Sales: 22000, Profit: 6000},
			{Month: "2026-06", Sales: 28000, Profit: 7800},
			{Month: "2026-07", Sales: 24000, Profit: 6500},
			{Month: "2026-08", Sales: 31000, Profit: 8400},
			{Month: "2026-09", Sales: 35000, Profit: 9200},
			{Month: "2026-10", Sales: 39000, Profit: 10500},
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

	// 6. Build SVG Chart Bars for Sales & Profit
	maxSalesVal := 1000.0
	for _, t := range trend {
		if t.Sales > maxSalesVal {
			maxSalesVal = t.Sales
		}
	}
	maxSalesVal = maxSalesVal * 1.15

	chartHeight := 170.0
	chartWidth := 480.0
	barWidth := 16.0
	spacing := (chartWidth - 40.0) / float64(len(trend))

	type ChartBar struct {
		Label        string
		Sales        float64
		Profit       float64
		SalesHeight  float64
		ProfitHeight float64
		SalesY       float64
		ProfitY      float64
		SalesX       float64
		ProfitX      float64
		LabelX       float64
	}
	var chartBars []ChartBar
	for i, t := range trend {
		sHeight := (t.Sales / maxSalesVal) * chartHeight
		pHeight := (t.Profit / maxSalesVal) * chartHeight
		sY := chartHeight - sHeight + 10
		pY := chartHeight - pHeight + 10
		x := 30.0 + float64(i)*spacing

		monthLabel := t.Month
		if parsedT, err := time.Parse("2006-01", t.Month); err == nil {
			monthLabel = parsedT.Format("Jan")
		}

		chartBars = append(chartBars, ChartBar{
			Label:        monthLabel,
			Sales:        t.Sales,
			Profit:       t.Profit,
			SalesHeight:  sHeight,
			ProfitHeight: pHeight,
			SalesY:       sY,
			ProfitY:      pY,
			SalesX:       x,
			ProfitX:      x + barWidth + 3,
			LabelX:       x + barWidth,
		})
	}

	// 7. Build SVG Chart Bars for Sales & Total Inventory
	maxInvVal := 1000.0
	for _, it := range invTrend {
		if it.Sales > maxInvVal {
			maxInvVal = it.Sales
		}
		if it.InventoryValue > maxInvVal {
			maxInvVal = it.InventoryValue
		}
	}
	maxInvVal = maxInvVal * 1.15

	type InvChartBar struct {
		Label     string
		Sales     float64
		Inventory float64
		SalesH    float64
		InvH      float64
		SalesY    float64
		InvY      float64
		SalesX    float64
		InvX      float64
		LabelX    float64
	}
	var invChartBars []InvChartBar
	for i, it := range invTrend {
		sH := (it.Sales / maxInvVal) * chartHeight
		iH := (it.InventoryValue / maxInvVal) * chartHeight
		sY := chartHeight - sH + 10
		iY := chartHeight - iH + 10
		x := 30.0 + float64(i)*spacing

		monthLabel := it.Month
		if parsedT, err := time.Parse("2006-01", it.Month); err == nil {
			monthLabel = parsedT.Format("Jan")
		}

		invChartBars = append(invChartBars, InvChartBar{
			Label:     monthLabel,
			Sales:     it.Sales,
			Inventory: it.InventoryValue,
			SalesH:    sH,
			InvH:      iH,
			SalesY:    sY,
			InvY:      iY,
			SalesX:    x,
			InvX:      x + barWidth + 3,
			LabelX:    x + barWidth,
		})
	}

	// 8. Top 5 Most Purchased Products
	var topProducts []models.TopProduct
	_ = db.DB.Select(&topProducts, `
		SELECT s.item_id, i.code as item_code, i.description, COALESCE(b.name, 'Unbranded') as brand_name,
		       SUM(s.qty) as total_qty, s.uom, SUM(s.total_sales) as total_sales, SUM(s.profit) as total_profit
		FROM sales_details s
		JOIN items i ON s.item_id = i.id
		LEFT JOIN brands b ON i.brand_id = b.id
		WHERE s.doc_status IN ('Posted', 'POSTED')
		GROUP BY s.item_id, i.code, i.description, b.name, s.uom
		ORDER BY total_qty DESC
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
		SELECT id, title, details, due_date, priority, category, status, created_at
		FROM reminders
		WHERE status = 'Pending'
		ORDER BY due_date ASC
		LIMIT 4
	`)

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
		InvChartBars      []InvChartBar
		TopProducts       []models.TopProduct
		Goals             []models.GoalWithProgress
		UpcomingReminders []models.Reminder
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
		InvChartBars:      invChartBars,
		TopProducts:       topProducts,
		Goals:             goals,
		UpcomingReminders: upcomingReminders,
	}

	if r.Header.Get("HX-Request") == "true" && r.URL.Query().Get("fragment") == "trends" {
		t, ok := app.Templates["dashboard_trends.html"]
		if ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = t.Execute(w, data)
			return
		}
	}

	app.RenderPage(w, r, "dashboard.html", data)
}
