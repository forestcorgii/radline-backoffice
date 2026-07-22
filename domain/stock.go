package domain

import "sort"

// ItemStock is an aggregate root that manages the inventory transactions for a specific Item.
type ItemStock struct {
	Item          Item
	UomSettings   []UomSetting
	ReceivingLogs []ReceivingLog
	SalesDetails  []SalesDetail
	Adjustments   []InventoryAdjustment
}

// NewItemStock constructs a new ItemStock aggregate.
func NewItemStock(item Item, uomSettings []UomSetting, receivingLogs []ReceivingLog, salesDetails []SalesDetail, adjustments []InventoryAdjustment) ItemStock {
	return ItemStock{
		Item:          item,
		UomSettings:   uomSettings,
		ReceivingLogs: receivingLogs,
		SalesDetails:  salesDetails,
		Adjustments:   adjustments,
	}
}

// CalculateOnHand calculates stock on hand for a specific supplier.
func (s ItemStock) CalculateOnHand(supplier string) float64 {
	totalReceived := 0.0
	for _, r := range s.ReceivingLogs {
		if r.Supplier != supplier {
			continue
		}
		factor := s.getConversionFactor(r.UOM)
		totalReceived += r.Qty * factor
	}

	totalSold := 0.0
	for _, sd := range s.SalesDetails {
		if sd.Supplier != supplier {
			continue
		}
		if sd.DocStatus != "Posted" && sd.DocStatus != "POSTED" {
			continue
		}
		factor := s.getConversionFactor(sd.UOM)
		totalSold += sd.Qty * factor
	}

	return totalReceived - totalSold
}

// CalculateGlobalOnHand calculates global stock on hand across all suppliers, incorporating adjustments.
func (s ItemStock) CalculateGlobalOnHand() float64 {
	totalReceived := 0.0
	for _, r := range s.ReceivingLogs {
		factor := s.getConversionFactor(r.UOM)
		totalReceived += r.Qty * factor
	}

	totalSold := 0.0
	for _, sd := range s.SalesDetails {
		if sd.DocStatus != "Posted" && sd.DocStatus != "POSTED" {
			continue
		}
		factor := s.getConversionFactor(sd.UOM)
		totalSold += sd.Qty * factor
	}

	totalAdjusted := 0.0
	for _, adj := range s.Adjustments {
		factor := s.getConversionFactor(adj.UOM)
		totalAdjusted += adj.AdjustmentQty * factor
	}

	return totalReceived - totalSold + totalAdjusted
}

func (s ItemStock) getConversionFactor(uom string) float64 {
	if uom == s.Item.DefaultUOM {
		return 1.0
	}
	for _, setting := range s.UomSettings {
		if setting.MUOM == uom {
			return setting.ConversionFactor
		}
	}
	return 1.0
}

// GetOldestPLWithStock returns the cost, selling price, and PL No of the oldest receiving log (PL) that has stock available.
// If no stock is available or no receiving logs exist, it returns (0, 0, "", false).
func (s ItemStock) GetOldestPLWithStock() (cost float64, price float64, plNo string, found bool) {
	if len(s.ReceivingLogs) == 0 {
		return 0.0, 0.0, "", false
	}

	onHand := s.CalculateGlobalOnHand()
	if onHand <= 0 {
		return 0.0, 0.0, "", false
	}

	// Sort receiving logs chronologically (oldest first: date ASC, then ID ASC)
	logs := make([]ReceivingLog, len(s.ReceivingLogs))
	copy(logs, s.ReceivingLogs)

	sort.Slice(logs, func(i, j int) bool {
		if logs[i].Date.Equal(logs[j].Date) {
			return logs[i].ID < logs[j].ID
		}
		return logs[i].Date.Before(logs[j].Date)
	})

	// Calculate total received across all logs in default UOM
	totalReceived := 0.0
	for _, r := range logs {
		factor := s.getConversionFactor(r.UOM)
		totalReceived += r.Qty * factor
	}

	// Total consumption in default UOM
	consumed := totalReceived - onHand
	if consumed < 0 {
		consumed = 0
	}

	// Find the oldest log that has remaining stock
	for _, r := range logs {
		factor := s.getConversionFactor(r.UOM)
		receivedQtyDefault := r.Qty * factor
		if consumed >= receivedQtyDefault {
			consumed -= receivedQtyDefault
		} else {
			// This log has remaining stock!
			return r.Cost, r.SellingPrice, r.PLNo, true
		}
	}

	return 0.0, 0.0, "", false
}

