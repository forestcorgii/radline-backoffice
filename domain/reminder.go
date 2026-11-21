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
	ID               int64            `json:"id" db:"id"`
	Title            string           `json:"title" db:"title"`
	Details          string           `json:"details" db:"details"`
	DueDate          time.Time        `json:"due_date" db:"due_date"`
	Priority         ReminderPriority `json:"priority" db:"priority"`
	Category         string           `json:"category" db:"category"` // "Payment", "Delivery", "Stock Check", "General"
	Status           ReminderStatus   `json:"status" db:"status"`
	AssignedToUserID *int64           `json:"assigned_to_user_id" db:"assigned_to_user_id"`
	AssignedToName   string           `json:"assigned_to_name" db:"assigned_to_name"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

func (r *Reminder) IsOverdue() bool {
	if r == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(string(r.Status)), string(StatusCompleted)) || r.DueDate.IsZero() {
		return false
	}
	loc := r.DueDate.Location()
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	if r.DueDate.Hour() == 0 && r.DueDate.Minute() == 0 && r.DueDate.Second() == 0 && r.DueDate.Nanosecond() == 0 {
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return r.DueDate.Before(startOfToday)
	}
	return r.DueDate.Before(now)
}

func (r *Reminder) AssignTo(userID *int64, userName string) {
	if userID == nil || *userID <= 0 {
		r.AssignedToUserID = nil
		r.AssignedToName = ""
		return
	}
	idCopy := *userID
	r.AssignedToUserID = &idCopy
	r.AssignedToName = strings.TrimSpace(userName)
}

func (r *Reminder) Unassign() {
	r.AssignedToUserID = nil
	r.AssignedToName = ""
}

func (r *Reminder) IsAssigned() bool {
	return r.AssignedToUserID != nil && *r.AssignedToUserID > 0
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

func ValidateReminderStatus(status ReminderStatus) error {
	if status != StatusPending && status != StatusCompleted {
		return errors.New("status must be Pending or Completed")
	}
	return nil
}
