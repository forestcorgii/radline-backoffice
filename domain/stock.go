package domain

import (
	"errors"
	"sort"
	"strings"
)

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

func (s ItemStock) matchesItemID(recordItemID int) bool {
	if s.Item.ID > 0 && recordItemID > 0 && recordItemID != s.Item.ID {
		return false
	}
	return true
}

// CalculateOnHand calculates stock on hand for a specific supplier.
func (s ItemStock) CalculateOnHand(supplier string) float64 {
	targetSupplier := strings.TrimSpace(supplier)
	totalReceived := 0.0
	for _, r := range s.ReceivingLogs {
		if !s.matchesItemID(r.ItemID) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(r.Supplier), targetSupplier) {
			continue
		}
		factor := s.getConversionFactor(r.UOM)
		totalReceived += r.Qty * factor
	}

	totalSold := 0.0
	for _, sd := range s.SalesDetails {
		if !s.matchesItemID(sd.ItemID) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(sd.Supplier), targetSupplier) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(sd.DocStatus), "Posted") {
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
		if !s.matchesItemID(r.ItemID) {
			continue
		}
		factor := s.getConversionFactor(r.UOM)
		totalReceived += r.Qty * factor
	}

	totalSold := 0.0
	for _, sd := range s.SalesDetails {
		if !s.matchesItemID(sd.ItemID) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(sd.DocStatus), "Posted") {
			continue
		}
		factor := s.getConversionFactor(sd.UOM)
		totalSold += sd.Qty * factor
	}

	totalAdjusted := 0.0
	for _, adj := range s.Adjustments {
		if !s.matchesItemID(adj.ItemID) {
			continue
		}
		factor := s.getConversionFactor(adj.UOM)
		totalAdjusted += adj.AdjustmentQty * factor
	}

	return totalReceived - totalSold + totalAdjusted
}

func (s ItemStock) getConversionFactor(uom string) float64 {
	trimmedUOM := strings.TrimSpace(uom)
	if trimmedUOM != "" && strings.EqualFold(trimmedUOM, strings.TrimSpace(s.Item.DefaultUOM)) {
		return 1.0
	}
	for _, setting := range s.UomSettings {
		if strings.EqualFold(strings.TrimSpace(setting.MUOM), trimmedUOM) {
			if setting.ConversionFactor > 0 {
				return setting.ConversionFactor
			}
			return 1.0
		}
	}
	return 1.0
}

// GetConversionFactorStrict returns the conversion factor for a UOM or an error if blank or unconfigured.
func (s *ItemStock) GetConversionFactorStrict(uom string) (float64, error) {
	if s == nil {
		return 0, errors.New("item stock cannot be nil")
	}
	trimmedUOM := strings.TrimSpace(uom)
	if trimmedUOM == "" {
		return 0, errors.New("UOM cannot be empty")
	}
	if strings.EqualFold(trimmedUOM, strings.TrimSpace(s.Item.DefaultUOM)) {
		return 1.0, nil
	}
	for _, setting := range s.UomSettings {
		if strings.EqualFold(strings.TrimSpace(setting.MUOM), trimmedUOM) {
			if !IsFinite(setting.ConversionFactor) || setting.ConversionFactor <= 0 {
				return 0, errors.New("conversion factor must be greater than zero")
			}
			return setting.ConversionFactor, nil
		}
	}
	return 0, errors.New("unknown UOM: " + trimmedUOM)
}

// GetOldestPLWithStock returns the cost, selling price (in DefaultUOM), and PL No of the oldest receiving log (PL) that has stock available.
// If no stock is available or no receiving logs exist, it returns (0, 0, "", false).
func (s ItemStock) GetOldestPLWithStock() (cost float64, price float64, plNo string, found bool) {
	return s.GetOldestPLWithStockForSupplier("")
}

// GetOldestPLWithStockForSupplier returns the cost, selling price (in DefaultUOM), and PL No of the oldest receiving log (PL)
// that has stock available for the given supplier (or globally if supplier is blank).
func (s ItemStock) GetOldestPLWithStockForSupplier(supplier string) (cost float64, sellingPrice float64, plNo string, found bool) {
	targetSupplier := strings.TrimSpace(supplier)

	// Filter receiving logs by ItemID (and supplier when specified)
	var logs []ReceivingLog
	for _, r := range s.ReceivingLogs {
		if !s.matchesItemID(r.ItemID) {
			continue
		}
		if targetSupplier != "" && !strings.EqualFold(strings.TrimSpace(r.Supplier), targetSupplier) {
			continue
		}
		logs = append(logs, r)
	}
	if len(logs) == 0 {
		return 0.0, 0.0, "", false
	}

	var onHand float64
	if targetSupplier != "" {
		onHand = s.CalculateOnHand(targetSupplier)
	} else {
		onHand = s.CalculateGlobalOnHand()
	}
	if onHand <= 1e-9 {
		return 0.0, 0.0, "", false
	}

	// Sort receiving logs chronologically (oldest first: date ASC, then ID ASC)
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].Date.Equal(logs[j].Date) {
			return logs[i].ID < logs[j].ID
		}
		return logs[i].Date.Before(logs[j].Date)
	})

	// Compute consumption without letting positive adjustments resurrect already-depleted PLs:
	// count posted sales (in default UOM) plus only negative adjustments (when global).
	consumed := 0.0
	for _, sd := range s.SalesDetails {
		if !s.matchesItemID(sd.ItemID) {
			continue
		}
		if targetSupplier != "" && !strings.EqualFold(strings.TrimSpace(sd.Supplier), targetSupplier) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(sd.DocStatus), "Posted") {
			continue
		}
		factor := s.getConversionFactor(sd.UOM)
		consumed += sd.Qty * factor
	}
	if targetSupplier == "" {
		for _, adj := range s.Adjustments {
			if !s.matchesItemID(adj.ItemID) {
				continue
			}
			if adj.AdjustmentQty < 0 {
				factor := s.getConversionFactor(adj.UOM)
				consumed += (-adj.AdjustmentQty) * factor
			}
		}
	}

	// Find the oldest log that has remaining stock
	for _, r := range logs {
		factor := s.getConversionFactor(r.UOM)
		if factor <= 0 {
			factor = 1.0
		}
		receivedQtyDefault := r.Qty * factor
		if consumed+1e-9 >= receivedQtyDefault {
			consumed -= receivedQtyDefault
			if consumed < 0 {
				consumed = 0
			}
		} else {
			// This log has remaining stock! Convert cost & sellingPrice to DefaultUOM.
			return Round2(r.Cost / factor), Round2(r.SellingPrice / factor), r.PLNo, true
		}
	}

	return 0.0, 0.0, "", false
}
