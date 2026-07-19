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
	Less1        float64
	Less2        float64
	Cost         float64
	TotalCost    float64
	Markup       float64
	SellingPrice float64
	Remarks      string
}

// NewReceivingLog creates a new receiving transaction and automatically calculates TotalCost.
// Cost (unit cost) = UnitPrice × (1 − Less1/100) × (1 − Less2/100)
// TotalCost = Qty × Cost
// SellingPrice = Cost × Markup / 100
func NewReceivingLog(id int, supplier string, date time.Time, plNo string, itemID int, qty float64, uom string, unitPrice, less1, less2, cost, markup float64, remarks string) (ReceivingLog, error) {
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
	if unitPrice < 0 {
		return ReceivingLog{}, errors.New("unit price cannot be negative")
	}
	if markup <= 0 {
		markup = 130
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
		Less1:        less1,
		Less2:        less2,
		Cost:         cost,
		TotalCost:    qty * cost,
		Markup:       markup,
		SellingPrice: cost * markup / 100,
		Remarks:      remarks,
	}, nil
}

// StockReceiveItem represents a single item line in a stock receive transaction.
type StockReceiveItem struct {
	ItemID       int
	Qty          float64
	UOM          string
	UnitPrice    float64 // supplier list price before discounts
	Less1        float64 // first discount percentage
	Less2        float64 // second discount percentage
	Cost         float64 // unit cost after discounts
	TotalCost    float64
	Markup       float64 // markup percentage (default 130)
	SellingPrice float64
	Remarks      string
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
		if item.UnitPrice < 0 {
			return StockReceive{}, errors.New("unit price cannot be negative")
		}

		markup := item.Markup
		if markup <= 0 {
			markup = 130
		}

		unitCost := item.UnitPrice * (1 - item.Less1/100) * (1 - item.Less2/100)

		validatedItems[i] = StockReceiveItem{
			ItemID:       item.ItemID,
			Qty:          item.Qty,
			UOM:          item.UOM,
			UnitPrice:    item.UnitPrice,
			Less1:        item.Less1,
			Less2:        item.Less2,
			Cost:         unitCost,
			TotalCost:    item.Qty * unitCost,
			Markup:       markup,
			SellingPrice: unitCost * markup / 100,
			Remarks:      item.Remarks,
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
			Less1:        item.Less1,
			Less2:        item.Less2,
			Cost:         item.Cost,
			TotalCost:    item.TotalCost,
			Markup:       item.Markup,
			SellingPrice: item.SellingPrice,
			Remarks:      item.Remarks,
		}
	}
	return logs
}

