package domain_test

import (
	"testing"
	"time"

	"radline/domain"
)

func TestReminder_IsOverdue(t *testing.T) {
	yesterday := time.Now().Add(-24 * time.Hour)
	tomorrow := time.Now().Add(24 * time.Hour)

	rPendingPast := domain.Reminder{
		DueDate: yesterday,
		Status:  domain.StatusPending,
	}
	if !rPendingPast.IsOverdue() {
		t.Errorf("expected pending reminder from yesterday to be overdue")
	}

	rCompletedPast := domain.Reminder{
		DueDate: yesterday,
		Status:  domain.StatusCompleted,
	}
	if rCompletedPast.IsOverdue() {
		t.Errorf("expected completed reminder NOT to be overdue")
	}

	rPendingFuture := domain.Reminder{
		DueDate: tomorrow,
		Status:  domain.StatusPending,
	}
	if rPendingFuture.IsOverdue() {
		t.Errorf("expected future reminder NOT to be overdue")
	}
}

func TestValidateReminder(t *testing.T) {
	err := domain.ValidateReminder("Supplier payment due", time.Now(), domain.PriorityHigh)
	if err != nil {
		t.Errorf("expected valid reminder, got error: %v", err)
	}

	errEmptyTitle := domain.ValidateReminder("", time.Now(), domain.PriorityLow)
	if errEmptyTitle == nil {
		t.Errorf("expected error for empty title")
	}

	errZeroDate := domain.ValidateReminder("Check inventory", time.Time{}, domain.PriorityLow)
	if errZeroDate == nil {
		t.Errorf("expected error for zero due date")
	}
}

func TestReminder_Assignment(t *testing.T) {
	r := domain.Reminder{Title: "Audit warehouse"}
	if r.IsAssigned() {
		t.Errorf("expected new reminder not to be assigned")
	}

	uid := int64(42)
	r.AssignTo(&uid, "Alice")
	if !r.IsAssigned() {
		t.Errorf("expected reminder to be assigned")
	}
	if *r.AssignedToUserID != 42 || r.AssignedToName != "Alice" {
		t.Errorf("unexpected assigned user data: %v, %s", r.AssignedToUserID, r.AssignedToName)
	}

	r.Unassign()
	if r.IsAssigned() {
		t.Errorf("expected reminder to be unassigned")
	}
	if r.AssignedToUserID != nil || r.AssignedToName != "" {
		t.Errorf("expected nil user and empty name after unassign")
	}
}

func TestValidateReminderStatus(t *testing.T) {
	if err := domain.ValidateReminderStatus(domain.StatusPending); err != nil {
		t.Errorf("expected pending status to be valid, got: %v", err)
	}
	if err := domain.ValidateReminderStatus(domain.StatusCompleted); err != nil {
		t.Errorf("expected completed status to be valid, got: %v", err)
	}
	if err := domain.ValidateReminderStatus("Invalid"); err == nil {
		t.Errorf("expected invalid status to return error")
	}
}

