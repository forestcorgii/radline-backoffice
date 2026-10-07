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
