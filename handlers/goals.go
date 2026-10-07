package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"radline/db"
	"radline/domain"
	"radline/models"
)

// GoalsHandler displays earning targets and tracks performance against actual figures
func (app *App) GoalsHandler(w http.ResponseWriter, r *http.Request) {
	var rawGoals []models.EarningGoal
	err := db.DB.Select(&rawGoals, "SELECT id, title, period_type, target_period, target_revenue, target_profit, created_at FROM earning_goals ORDER BY target_period DESC, id DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var goalsWithProgress []models.GoalWithProgress
	for _, g := range rawGoals {
		// Calculate actual revenue and profit for the target period
		var actual struct {
			Revenue float64 `db:"revenue"`
			Profit  float64 `db:"profit"`
		}

		var query string
		var arg string
		if g.PeriodType == string(domain.GoalPeriodMonthly) {
			// e.g. "2026-10"
			query = `
				SELECT COALESCE(SUM(total_sales), 0.0) as revenue, COALESCE(SUM(profit), 0.0) as profit
				FROM sales_details
				WHERE doc_status IN ('Posted', 'POSTED') AND substr(doc_date, 1, 7) = ?
			`
			arg = g.TargetPeriod
		} else {
			// e.g. "2026"
			query = `
				SELECT COALESCE(SUM(total_sales), 0.0) as revenue, COALESCE(SUM(profit), 0.0) as profit
				FROM sales_details
				WHERE doc_status IN ('Posted', 'POSTED') AND substr(doc_date, 1, 4) = ?
			`
			arg = g.TargetPeriod
		}

		_ = db.DB.Get(&actual, query, arg)

		domGoal := domain.EarningGoal{
			ID:            g.ID,
			Title:         g.Title,
			PeriodType:    domain.GoalPeriodType(g.PeriodType),
			TargetPeriod:  g.TargetPeriod,
			TargetRevenue: g.TargetRevenue,
			TargetProfit:  g.TargetProfit,
			CreatedAt:     g.CreatedAt,
		}
		prog := domGoal.CalculateProgress(actual.Revenue, actual.Profit)

		goalsWithProgress = append(goalsWithProgress, models.GoalWithProgress{
			EarningGoal:     g,
			ActualRevenue:   actual.Revenue,
			ActualProfit:    actual.Profit,
			RevenueProgress: prog.RevenueProgress,
			ProfitProgress:  prog.ProfitProgress,
			RevenueVariance: prog.RevenueVariance,
			ProfitVariance:  prog.ProfitVariance,
			Status:          prog.Status,
		})
	}

	currentMonth := time.Now().Format("2006-01")
	currentYear := time.Now().Format("2006")

	data := struct {
		Goals        []models.GoalWithProgress
		CurrentMonth string
		CurrentYear  string
	}{
		Goals:        goalsWithProgress,
		CurrentMonth: currentMonth,
		CurrentYear:  currentYear,
	}

	app.RenderPage(w, r, "goals.html", data)
}

// AddGoalHandler saves a new monthly or yearly earnings target
func (app *App) AddGoalHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	periodType := strings.TrimSpace(r.FormValue("period_type"))
	targetPeriod := strings.TrimSpace(r.FormValue("target_period"))
	targetRevenue, _ := strconv.ParseFloat(r.FormValue("target_revenue"), 64)
	targetProfit, _ := strconv.ParseFloat(r.FormValue("target_profit"), 64)

	if err := domain.ValidateEarningGoal(title, periodType, targetPeriod, targetRevenue, targetProfit); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := db.DB.Exec(`
		INSERT INTO earning_goals (title, period_type, target_period, target_revenue, target_profit)
		VALUES (?, ?, ?, ?, ?)
	`, title, periodType, targetPeriod, targetRevenue, targetProfit)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to create earning goal"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newID, _ := res.LastInsertId()
	app.LogActivity(r, "CREATE_GOAL", "EarningGoal", fmt.Sprintf("%d", newID), fmt.Sprintf("Created %s target '%s' for period %s", periodType, title, targetPeriod))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Earning goal set successfully!"}}`)
	if r.Header.Get("HX-Target") == "#main-content" {
		w.Header().Set("HX-Location", `{"path": "/goals", "target": "#main-content"}`)
	} else {
		w.Header().Set("HX-Redirect", "/goals")
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteGoalHandler removes an earning goal
func (app *App) DeleteGoalHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid goal ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM earning_goals WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to delete goal"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "DELETE_GOAL", "EarningGoal", idStr, "Deleted earning goal ID "+idStr)
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Earning goal removed"}}`)
	if r.Header.Get("HX-Target") == "#main-content" {
		w.Header().Set("HX-Location", `{"path": "/goals", "target": "#main-content"}`)
	} else {
		w.Header().Set("HX-Redirect", "/goals")
	}
	w.WriteHeader(http.StatusOK)
}
