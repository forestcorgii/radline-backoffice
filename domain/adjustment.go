package domain

import (
	"errors"
	"time"
)

// InventoryAdjustment represents a manual stock count adjustment.
type InventoryAdjustment struct {
	ID            int
	AdjustmentID  int
	Date          time.Time
	ItemID        int
	UOM           string
	AdjustmentQty float64
	Cost          float64
	Remarks       string
}

// NewInventoryAdjustment creates an inventory adjustment, validating parameters.
func NewInventoryAdjustment(id int, adjustmentID int, date time.Time, itemID int, uom string, qty, cost float64, remarks string) (InventoryAdjustment, error) {
	if itemID <= 0 {
		return InventoryAdjustment{}, errors.New("item ID must be valid")
	}
	if uom == "" {
		return InventoryAdjustment{}, errors.New("UOM cannot be empty")
	}
	if qty == 0 {
		return InventoryAdjustment{}, errors.New("adjustment quantity cannot be zero")
	}
	if cost < 0 {
		return InventoryAdjustment{}, errors.New("cost cannot be negative")
	}
	if remarks == "" {
		return InventoryAdjustment{}, errors.New("remarks cannot be empty")
	}
	return InventoryAdjustment{
		ID:            id,
		AdjustmentID:  adjustmentID,
		Date:          date,
		ItemID:        itemID,
		UOM:           uom,
		AdjustmentQty: qty,
		Cost:          cost,
		Remarks:       remarks,
	}, nil
}

// StockAdjustmentItem represents a single item line in a stock adjustment transaction.
type StockAdjustmentItem struct {
	ItemID int
	Qty    float64
	UOM    string
	Cost   float64
}

// StockAdjustment represents a stock adjustment transaction header with multiple items.
type StockAdjustment struct {
	Date    time.Time
	Remarks string
	Items   []StockAdjustmentItem
}

// NewStockAdjustment constructs and validates a StockAdjustment aggregate.
func NewStockAdjustment(date time.Time, remarks string, items []StockAdjustmentItem) (StockAdjustment, error) {
	if date.IsZero() {
		return StockAdjustment{}, errors.New("date must be valid")
	}
	if remarks == "" {
		return StockAdjustment{}, errors.New("remarks cannot be empty")
	}
	if len(items) == 0 {
		return StockAdjustment{}, errors.New("adjustment must have at least one item")
	}

	validatedItems := make([]StockAdjustmentItem, len(items))
	for i, item := range items {
		if item.ItemID <= 0 {
			return StockAdjustment{}, errors.New("item ID must be valid")
		}
		if item.Qty == 0 {
			return StockAdjustment{}, errors.New("adjustment quantity cannot be zero")
		}
		if item.UOM == "" {
			return StockAdjustment{}, errors.New("UOM cannot be empty")
		}
		if item.Cost < 0 {
			return StockAdjustment{}, errors.New("cost cannot be negative")
		}

		validatedItems[i] = StockAdjustmentItem{
			ItemID: item.ItemID,
			Qty:    item.Qty,
			UOM:    item.UOM,
			Cost:   item.Cost,
		}
	}

	return StockAdjustment{
		Date:    date,
		Remarks: remarks,
		Items:   validatedItems,
	}, nil
}

// ToInventoryAdjustments maps the StockAdjustment aggregate to a slice of flat InventoryAdjustment records.
// The adjustmentID is set to 0 here; the handler assigns the real DB-generated ID before insertion.
func (s StockAdjustment) ToInventoryAdjustments() []InventoryAdjustment {
	adjustments := make([]InventoryAdjustment, len(s.Items))
	for i, item := range s.Items {
		adjustments[i] = InventoryAdjustment{
			Date:          s.Date,
			ItemID:        item.ItemID,
			UOM:           item.UOM,
			AdjustmentQty: item.Qty,
			Cost:          item.Cost,
			Remarks:       s.Remarks,
		}
	}
	return adjustments
}
