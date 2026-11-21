package domain_test

import (
	"math"
	"testing"

	"rav/domain"
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

	// Net loss (negative profit) should never classify as Exceeded or On Track
	pLossHighRev := goal.CalculateProgress(200000.0, -3000.0) // revProgress = 200%, profitProgress = -10%, avg = 95%
	if pLossHighRev.Status != "In Progress" {
		t.Errorf("expected status 'In Progress' when profit is negative (avg >= 40), got %s", pLossHighRev.Status)
	}
	pLossLowRev := goal.CalculateProgress(50000.0, -15000.0) // revProgress = 50%, profitProgress = -50%, avg = 0%
	if pLossLowRev.Status != "Behind" {
		t.Errorf("expected status 'Behind' when profit is negative (avg < 40), got %s", pLossLowRev.Status)
	}
}

func TestValidateEarningGoal(t *testing.T) {
	err := domain.ValidateEarningGoal("Q4 Goal", "Monthly", "2026-10", 50000, 15000)
	if err != nil {
		t.Errorf("expected valid goal, got err: %v", err)
	}

	if err := domain.ValidateEarningGoal("FY 2026", "Yearly", "2026", 500000, 150000); err != nil {
		t.Errorf("expected valid yearly goal, got err: %v", err)
	}

	if err := domain.ValidateEarningGoal("Bad Month", "Monthly", "2026-13", 50000, 15000); err == nil {
		t.Errorf("expected error for invalid month 2026-13")
	}

	if err := domain.ValidateEarningGoal("Bad Year", "Yearly", "2026-10", 50000, 15000); err == nil {
		t.Errorf("expected error for YYYY-MM on Yearly period type")
	}

	errInvalidType := domain.ValidateEarningGoal("Q4 Goal", "Weekly", "2026-W40", 50000, 15000)
	if errInvalidType == nil {
		t.Errorf("expected error for invalid period type Weekly")
	}

	errZeroRev := domain.ValidateEarningGoal("Q4 Goal", "Monthly", "2026-10", 0, 15000)
	if errZeroRev == nil {
		t.Errorf("expected error for zero target revenue")
	}

	if err := domain.ValidateEarningGoal("Profit > Rev", "Monthly", "2026-10", 10000, 20000); err == nil {
		t.Errorf("expected error when targetProfit > targetRevenue")
	}

	if err := domain.ValidateEarningGoal("NaN Rev", "Monthly", "2026-10", math.NaN(), 15000); err == nil {
		t.Errorf("expected error for NaN targetRevenue")
	}
}
