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
	if !IsFinite(actualRevenue) {
		actualRevenue = 0
	}
	if !IsFinite(actualProfit) {
		actualProfit = 0
	}
	actualRevenue = Round2(actualRevenue)
	actualProfit = Round2(actualProfit)
	targetRevenue := Round2(g.TargetRevenue)
	targetProfit := Round2(g.TargetProfit)

	var revProgress float64
	if targetRevenue > 0 {
		revProgress = math.Round(((actualRevenue/targetRevenue)*100)*10) / 10
	}

	var profitProgress float64
	if targetProfit > 0 {
		profitProgress = math.Round(((actualProfit/targetProfit)*100)*10) / 10
	}

	status := "In Progress"
	avgProgress := (revProgress + profitProgress) / 2
	if profitProgress < 0 {
		if avgProgress >= 40 {
			status = "In Progress"
		} else {
			status = "Behind"
		}
	} else if revProgress >= 100 && profitProgress >= 100 {
		status = "Exceeded"
	} else if avgProgress >= 75 {
		status = "On Track"
	} else if avgProgress < 50 {
		status = "Behind"
	}

	return GoalProgress{
		RevenueProgress: revProgress,
		ProfitProgress:  profitProgress,
		RevenueVariance: Round2(targetRevenue - actualRevenue),
		ProfitVariance:  Round2(targetProfit - actualProfit),
		Status:          status,
	}
}

func isValidTargetPeriod(periodType, targetPeriod string) bool {
	if periodType == string(GoalPeriodMonthly) {
		if len(targetPeriod) != 7 || targetPeriod[4] != '-' {
			return false
		}
		_, err := time.Parse("2006-01", targetPeriod)
		return err == nil
	}
	if periodType == string(GoalPeriodYearly) {
		if len(targetPeriod) != 4 {
			return false
		}
		_, err := time.Parse("2006", targetPeriod)
		return err == nil
	}
	return false
}

func ValidateEarningGoal(title, periodType, targetPeriod string, targetRevenue, targetProfit float64) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("goal title is required")
	}
	periodType = strings.TrimSpace(periodType)
	if periodType != string(GoalPeriodMonthly) && periodType != string(GoalPeriodYearly) {
		return errors.New("period type must be Monthly or Yearly")
	}
	targetPeriod = strings.TrimSpace(targetPeriod)
	if targetPeriod == "" {
		return errors.New("target period is required")
	}
	if !isValidTargetPeriod(periodType, targetPeriod) {
		if periodType == string(GoalPeriodMonthly) {
			return errors.New("monthly target period must be in YYYY-MM format")
		}
		return errors.New("yearly target period must be in YYYY format")
	}
	if !IsFinite(targetRevenue) || targetRevenue <= 0 {
		return errors.New("target revenue must be greater than zero")
	}
	if !IsFinite(targetProfit) || targetProfit <= 0 {
		return errors.New("target profit must be greater than zero")
	}
	if Round2(targetProfit) > Round2(targetRevenue) {
		return errors.New("target profit cannot exceed target revenue")
	}
	return nil
}
