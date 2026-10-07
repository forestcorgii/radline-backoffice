package domain

import (
	"errors"
	"math"
	"strings"
	"time"
)

// GoalPeriodType defines the cadence of an earning goal
type GoalPeriodType string

const (
	GoalPeriodMonthly GoalPeriodType = "Monthly"
	GoalPeriodYearly  GoalPeriodType = "Yearly"
)

// EarningGoal represents a business revenue and profit target
type EarningGoal struct {
	ID            int64          `json:"id" db:"id"`
	Title         string         `json:"title" db:"title"`
	PeriodType    GoalPeriodType `json:"period_type" db:"period_type"`
	TargetPeriod  string         `json:"target_period" db:"target_period"` // "YYYY-MM" or "YYYY"
	TargetRevenue float64        `json:"target_revenue" db:"target_revenue"`
	TargetProfit  float64        `json:"target_profit" db:"target_profit"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

// GoalProgress computes metrics for a goal against actual performance
type GoalProgress struct {
	RevenueProgress float64 // percentage 0-100+
	ProfitProgress  float64 // percentage 0-100+
	RevenueVariance float64 // remaining or surplus
	ProfitVariance  float64 // remaining or surplus
	Status          string  // "Exceeded", "On Track", "In Progress", "Behind"
}

func (g *EarningGoal) CalculateProgress(actualRevenue, actualProfit float64) GoalProgress {
	var revProgress float64
	if g.TargetRevenue > 0 {
		revProgress = (actualRevenue / g.TargetRevenue) * 100
	}

	var profitProgress float64
	if g.TargetProfit > 0 {
		profitProgress = (actualProfit / g.TargetProfit) * 100
	}

	status := "In Progress"
	avgProgress := (revProgress + profitProgress) / 2
	if revProgress >= 100 && profitProgress >= 100 {
		status = "Exceeded"
	} else if avgProgress >= 75 {
		status = "On Track"
	} else if avgProgress < 50 {
		status = "Behind"
	}

	return GoalProgress{
		RevenueProgress: math.Round(revProgress*10) / 10,
		ProfitProgress:  math.Round(profitProgress*10) / 10,
		RevenueVariance: g.TargetRevenue - actualRevenue,
		ProfitVariance:  g.TargetProfit - actualProfit,
		Status:          status,
	}
}

func ValidateEarningGoal(title, periodType, targetPeriod string, targetRevenue, targetProfit float64) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("goal title is required")
	}
	if periodType != string(GoalPeriodMonthly) && periodType != string(GoalPeriodYearly) {
		return errors.New("period type must be Monthly or Yearly")
	}
	if strings.TrimSpace(targetPeriod) == "" {
		return errors.New("target period is required")
	}
	if targetRevenue <= 0 {
		return errors.New("target revenue must be greater than zero")
	}
	if targetProfit <= 0 {
		return errors.New("target profit must be greater than zero")
	}
	return nil
}
