package main

import (
	"fmt"
	"time"

	"radline/db"
	"radline/models"
)

func main() {
	if err := db.InitDB("backoffice.db"); err != nil {
		fmt.Printf("InitDB error: %v\n", err)
		return
	}

	now := time.Now()
	year, month, _ := now.Date()
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	endOfMonth := time.Date(year, month, lastOfMonth.Day(), 23, 59, 59, 999999999, now.Location())

	// Test passing time.Time
	var monthlyReminders []models.Reminder
	err := db.DB.Select(&monthlyReminders, `
		SELECT id, title, details, due_date, priority, category, status, created_at
		FROM reminders
		WHERE due_date >= ? AND due_date <= ?
		ORDER BY due_date ASC, priority DESC
	`, firstOfMonth, endOfMonth)
	fmt.Printf("Query with time.Time err: %v, count: %d\n", err, len(monthlyReminders))
}
