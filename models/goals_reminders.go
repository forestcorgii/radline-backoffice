package models

import "time"

type EarningGoal struct {
	ID            int64     `db:"id"`
	Title         string    `db:"title"`
	PeriodType    string    `db:"period_type"`
	TargetPeriod  string    `db:"target_period"`
	TargetRevenue float64   `db:"target_revenue"`
	TargetProfit  float64   `db:"target_profit"`
	CreatedAt     time.Time `db:"created_at"`
}

type GoalWithProgress struct {
	EarningGoal
	ActualRevenue   float64
	ActualProfit    float64
	RevenueProgress float64
	ProfitProgress  float64
	RevenueVariance float64
	ProfitVariance  float64
	Status          string
}

type Reminder struct {
	ID        int64     `db:"id"`
	Title     string    `db:"title"`
	Details   string    `db:"details"`
	DueDate   time.Time `db:"due_date"`
	Priority  string    `db:"priority"`
	Category  string    `db:"category"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}

type TopProduct struct {
	ItemID      int     `db:"item_id"`
	ItemCode    string  `db:"item_code"`
	Description string  `db:"description"`
	BrandName   string  `db:"brand_name"`
	TotalQty    float64 `db:"total_qty"`
	UOM         string  `db:"uom"`
	TotalSales  float64 `db:"total_sales"`
	TotalProfit float64 `db:"total_profit"`
}

type SalesInventoryTrend struct {
	Month          string  `db:"month"`
	Sales          float64 `db:"sales"`
	Profit         float64 `db:"profit"`
	InventoryValue float64 `db:"inventory_value"`
}
