package main

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"radline/db"
	"radline/models"
)

func main() {
	_ = db.InitDB("backoffice.db")

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b float64) float64 { return a * b },
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) { return nil, nil },
	}

	t := template.New("dashboard.html").Funcs(funcMap)
	t = template.Must(t.ParseFiles("templates/base.html", "templates/dashboard.html", "templates/dashboard_trends.html"))

	// Let's execute with dummy goals and dummy reminders
	data := struct {
		Greeting          string
		TodayFormatted    string
		StartDateStr      string
		EndDateStr        string
		TotalSales        float64
		TotalCosts        float64
		GrossProfit       float64
		Margin            float64
		ChartBars         []interface{}
		InvChartBars      []interface{}
		TopProducts       []models.TopProduct
		Goals             []models.GoalWithProgress
		UpcomingReminders []models.Reminder
	}{
		Greeting:       "Good morning, Admin!",
		TodayFormatted: "Tuesday, Oct 06, 2026",
		Goals: []models.GoalWithProgress{
			{
				EarningGoal: models.EarningGoal{
					Title:         "October Goal",
					TargetRevenue: 50000,
					TargetProfit:  15000,
				},
				ActualRevenue:   25000,
				ActualProfit:    7500,
				RevenueProgress: 50.0,
				ProfitProgress:  50.0,
				Status:          "In Progress",
			},
		},
		UpcomingReminders: []models.Reminder{
			{
				ID:       1,
				Title:    "Stock Check",
				DueDate:  time.Now(),
				Priority: "High",
				Category: "Inventory",
				Status:   "Pending",
			},
		},
	}

	var buf bytes.Buffer
	err := t.ExecuteTemplate(&buf, "dashboard.html", data)
	if err != nil {
		fmt.Printf("EXECUTION ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS! Rendered %d bytes\n", buf.Len())
	}
}
