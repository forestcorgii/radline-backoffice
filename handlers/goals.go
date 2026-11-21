package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rav/db"
	"rav/domain"
	"rav/models"
)

type periodSalesTotals struct {
	Period  string  `db:"period"`
	Revenue float64 `db:"revenue"`
	Profit  float64 `db:"profit"`
}

// computeGoalsProgress calculates progress for a slice of goals without N+1 queries.
func computeGoalsProgress(rawGoals []models.EarningGoal) []models.GoalWithProgress {
	if len(rawGoals) == 0 {
		return nil
	}

	hasMonthly := false
	hasYearly := false
	for _, g := range rawGoals {
		if g.PeriodType == string(domain.GoalPeriodMonthly) {
			hasMonthly = true
		} else {
			hasYearly = true
		}
	}

	monthlyMap := make(map[string]periodSalesTotals)
	if hasMonthly && db.DB != nil {
		var rows []periodSalesTotals
		_ = db.DB.Select(&rows, `
			SELECT substr(doc_date, 1, 7) as period,
			       COALESCE(SUM(total_sales), 0.0) as revenue,
			       COALESCE(SUM(profit), 0.0) as profit
			FROM sales_details
			WHERE doc_status IN ('Posted', 'POSTED')
			GROUP BY period
		`)
		for _, r := range rows {
			monthlyMap[r.Period] = r
		}
	}

	yearlyMap := make(map[string]periodSalesTotals)
	if hasYearly && db.DB != nil {
		var rows []periodSalesTotals
		_ = db.DB.Select(&rows, `
			SELECT substr(doc_date, 1, 4) as period,
			       COALESCE(SUM(total_sales), 0.0) as revenue,
			       COALESCE(SUM(profit), 0.0) as profit
			FROM sales_details
			WHERE doc_status IN ('Posted', 'POSTED')
			GROUP BY period
		`)
		for _, r := range rows {
			yearlyMap[r.Period] = r
		}
	}

	goalsWithProgress := make([]models.GoalWithProgress, 0, len(rawGoals))
	for _, g := range rawGoals {
		var actual periodSalesTotals
		if g.PeriodType == string(domain.GoalPeriodMonthly) {
			actual = monthlyMap[g.TargetPeriod]
		} else {
			actual = yearlyMap[g.TargetPeriod]
		}

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
	return goalsWithProgress
}

// GoalsHandler displays earning targets and tracks performance against actual figures
func (app *App) GoalsHandler(w http.ResponseWriter, r *http.Request) {
	var rawGoals []models.EarningGoal
	err := db.DB.Select(&rawGoals, "SELECT id, title, period_type, target_period, target_revenue, target_profit, created_at FROM earning_goals ORDER BY target_period DESC, id DESC")
	if err != nil {
		log.Printf("GoalsHandler DB error: %v", err)
		http.Error(w, "Failed to load earning goals", http.StatusInternalServerError)
		return
	}

	goalsWithProgress := computeGoalsProgress(rawGoals)

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
		setHXTriggerToast(w, "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var newID int64
	err := db.DB.QueryRow(`
		INSERT INTO earning_goals (title, period_type, target_period, target_revenue, target_profit)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`, title, periodType, targetPeriod, targetRevenue, targetProfit).Scan(&newID)
	if err != nil {
		log.Printf("AddGoalHandler insert error: %v", err)
		setHXTriggerToast(w, "error", "Failed to create earning goal")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "CREATE_GOAL", "EarningGoal", fmt.Sprintf("%d", newID), fmt.Sprintf("Created %s target '%s' for period %s", periodType, title, targetPeriod))
	setHXTriggerToast(w, "success", "Earning goal set successfully!")
	target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#")
	if target == "main-content" {
		setHXLocation(w, "/goals", "#main-content")
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
		log.Printf("DeleteGoalHandler delete error: %v", err)
		setHXTriggerToast(w, "error", "Failed to delete goal")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "DELETE_GOAL", "EarningGoal", idStr, "Deleted earning goal ID "+idStr)
	setHXTriggerToast(w, "success", "Earning goal removed")
	target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#")
	if target == "main-content" {
		setHXLocation(w, "/goals", "#main-content")
	} else {
		w.Header().Set("HX-Redirect", "/goals")
	}
	w.WriteHeader(http.StatusOK)
}
