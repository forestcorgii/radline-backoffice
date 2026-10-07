package domain_test

import (
	"testing"

	"radline/domain"
)

func TestEarningGoal_CalculateProgress(t *testing.T) {
	goal := domain.EarningGoal{
		Title:         "October 2026 Target",
		PeriodType:    domain.GoalPeriodMonthly,
		TargetPeriod:  "2026-10",
		TargetRevenue: 100000.0,
		TargetProfit:  30000.0,
	}

	// 50% revenue, 50% profit
	p := goal.CalculateProgress(50000.0, 15000.0)
	if p.RevenueProgress != 50.0 {
		t.Errorf("expected 50.0%% revenue progress, got %f", p.RevenueProgress)
	}
	if p.ProfitProgress != 50.0 {
		t.Errorf("expected 50.0%% profit progress, got %f", p.ProfitProgress)
	}
	if p.RevenueVariance != 50000.0 {
		t.Errorf("expected 50000 variance, got %f", p.RevenueVariance)
	}

	// Exceeded
	pExceeded := goal.CalculateProgress(120000.0, 35000.0)
	if pExceeded.Status != "Exceeded" {
		t.Errorf("expected status Exceeded, got %s", pExceeded.Status)
	}
}

func TestValidateEarningGoal(t *testing.T) {
	err := domain.ValidateEarningGoal("Q4 Goal", "Monthly", "2026-10", 50000, 15000)
	if err != nil {
		t.Errorf("expected valid goal, got err: %v", err)
	}

	errInvalidType := domain.ValidateEarningGoal("Q4 Goal", "Weekly", "2026-W40", 50000, 15000)
	if errInvalidType == nil {
		t.Errorf("expected error for invalid period type Weekly")
	}

	errZeroRev := domain.ValidateEarningGoal("Q4 Goal", "Monthly", "2026-10", 0, 15000)
	if errZeroRev == nil {
		t.Errorf("expected error for zero target revenue")
	}
}
