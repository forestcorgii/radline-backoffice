package domain

import (
	"errors"
	"strings"
	"time"
)

type ReminderPriority string

const (
	PriorityLow    ReminderPriority = "Low"
	PriorityMedium ReminderPriority = "Medium"
	PriorityHigh   ReminderPriority = "High"
)

type ReminderStatus string

const (
	StatusPending   ReminderStatus = "Pending"
	StatusCompleted ReminderStatus = "Completed"
)

// Reminder represents a scheduled business task or alert
type Reminder struct {
	ID        int64            `json:"id" db:"id"`
	Title     string           `json:"title" db:"title"`
	Details   string           `json:"details" db:"details"`
	DueDate   time.Time        `json:"due_date" db:"due_date"`
	Priority  ReminderPriority `json:"priority" db:"priority"`
	Category  string           `json:"category" db:"category"` // "Payment", "Delivery", "Stock Check", "General"
	Status    ReminderStatus   `json:"status" db:"status"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
}

func (r *Reminder) IsOverdue() bool {
	if r.Status == StatusCompleted {
		return false
	}
	// Consider overdue if due date was before today (midnight)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	return r.DueDate.Before(today) && r.DueDate.Before(now)
}

func ValidateReminder(title string, dueDate time.Time, priority ReminderPriority) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("reminder title is required")
	}
	if dueDate.IsZero() {
		return errors.New("valid due date is required")
	}
	if priority != PriorityLow && priority != PriorityMedium && priority != PriorityHigh {
		return errors.New("priority must be Low, Medium, or High")
	}
	return nil
}
