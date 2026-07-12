package domain

import (
	"errors"
	"time"
)

// ReceivingLog represents an incoming stock transaction record.
type ReceivingLog struct {
	ID           int
	Supplier     string
	Date         time.Time
	PLNo         string
	ItemID       int
	Qty          float64
	UOM          string
	UnitPrice    float64
	Cost         float64
	TotalCost    float64
	SellingPrice float64
}

// NewReceivingLog creates a new receiving transaction and automatically calculates TotalCost.
func NewReceivingLog(id int, supplier string, date time.Time, plNo string, itemID int, qty float64, uom string, unitPrice, cost float64) (ReceivingLog, error) {
	if supplier == "" {
		return ReceivingLog{}, errors.New("supplier cannot be empty")
	}
	if itemID <= 0 {
		return ReceivingLog{}, errors.New("item ID must be valid")
	}
	if qty <= 0 {
		return ReceivingLog{}, errors.New("quantity must be greater than zero")
	}
	if uom == "" {
		return ReceivingLog{}, errors.New("UOM cannot be empty")
	}
	if cost < 0 {
		return ReceivingLog{}, errors.New("cost cannot be negative")
	}
	if unitPrice < 0 {
		return ReceivingLog{}, errors.New("unit price cannot be negative")
	}
	return ReceivingLog{
		ID:           id,
		Supplier:     supplier,
		Date:         date,
		PLNo:         plNo,
		ItemID:       itemID,
		Qty:          qty,
		UOM:          uom,
		UnitPrice:    unitPrice,
		Cost:         cost,
		TotalCost:    qty * cost,
		SellingPrice: unitPrice,
	}, nil
}

// StockReceiveItem represents a single item line in a stock receive transaction.
type StockReceiveItem struct {
	ItemID       int
	Qty          float64
	UOM          string
	UnitPrice    float64 // unit selling price
	Cost         float64 // unit cost
	TotalCost    float64
	SellingPrice float64
}

// StockReceive represents a stock receive transaction header with multiple items.
type StockReceive struct {
	PLNo     string
	Supplier string
	Date     time.Time
	Items    []StockReceiveItem
}

// NewStockReceive constructs and validates a StockReceive aggregate.
func NewStockReceive(plNo, supplier string, date time.Time, items []StockReceiveItem) (StockReceive, error) {
	if supplier == "" {
		return StockReceive{}, errors.New("supplier cannot be empty")
	}
	if date.IsZero() {
		return StockReceive{}, errors.New("date must be valid")
	}
	if len(items) == 0 {
		return StockReceive{}, errors.New("stocktake must have at least one item")
	}

	validatedItems := make([]StockReceiveItem, len(items))
	for i, item := range items {
		if item.ItemID <= 0 {
			return StockReceive{}, errors.New("item ID must be valid")
		}
		if item.Qty <= 0 {
			return StockReceive{}, errors.New("quantity must be greater than zero")
		}
		if item.UOM == "" {
			return StockReceive{}, errors.New("UOM cannot be empty")
		}
		if item.Cost < 0 {
			return StockReceive{}, errors.New("cost cannot be negative")
		}
		if item.UnitPrice < 0 {
			return StockReceive{}, errors.New("unit price cannot be negative")
		}

		validatedItems[i] = StockReceiveItem{
			ItemID:       item.ItemID,
			Qty:          item.Qty,
			UOM:          item.UOM,
			Cost:         item.Cost,
			UnitPrice:    item.UnitPrice,
			TotalCost:    item.Qty * item.Cost,
			SellingPrice: item.UnitPrice,
		}
	}

	return StockReceive{
		PLNo:     plNo,
		Supplier: supplier,
		Date:     date,
		Items:    validatedItems,
	}, nil
}

// ToReceivingLogs maps the StockReceive aggregate to a slice of flat ReceivingLog records.
func (s StockReceive) ToReceivingLogs() []ReceivingLog {
	logs := make([]ReceivingLog, len(s.Items))
	for i, item := range s.Items {
		logs[i] = ReceivingLog{
			Supplier:     s.Supplier,
			Date:         s.Date,
			PLNo:         s.PLNo,
			ItemID:       item.ItemID,
			Qty:          item.Qty,
			UOM:          item.UOM,
			UnitPrice:    item.UnitPrice,
			Cost:         item.Cost,
			TotalCost:    item.TotalCost,
			SellingPrice: item.SellingPrice,
		}
	}
	return logs
}

