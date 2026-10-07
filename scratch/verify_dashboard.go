package main

import (
	"fmt"
	"log"

	"radline/db"
	"radline/models"
)

func main() {
	err := db.InitDB("backoffice.db")
	if err != nil {
		log.Fatalf("InitDB failed: %v", err)
	}

	var goalCount int
	_ = db.DB.Get(&goalCount, "SELECT COUNT(*) FROM earning_goals")
	fmt.Printf("Goal count: %d\n", goalCount)

	var remCount int
	_ = db.DB.Get(&remCount, "SELECT COUNT(*) FROM reminders")
	fmt.Printf("Reminder count: %d\n", remCount)

	var rawGoals []models.EarningGoal
	err = db.DB.Select(&rawGoals, "SELECT id, title, period_type, target_period, target_revenue, target_profit, created_at FROM earning_goals ORDER BY target_period DESC LIMIT 2")
	fmt.Printf("Raw goals query err: %v, count: %d\n", err, len(rawGoals))

	var upcomingReminders []models.Reminder
	err = db.DB.Select(&upcomingReminders, `
		SELECT id, title, details, due_date, priority, category, status, created_at
		FROM reminders
		WHERE status = 'Pending'
		ORDER BY due_date ASC
		LIMIT 4
	`)
	fmt.Printf("Upcoming reminders query err: %v, count: %d\n", err, len(upcomingReminders))
}
