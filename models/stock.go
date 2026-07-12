package models

import (
	"database/sql"

	"github.com/jmoiron/sqlx"

	"radline/domain"
)

// CalculateStockOnHand calculates the exact stock level for an item within a specific supplier normalized to Default UOM
func CalculateStockOnHand(db *sqlx.DB, itemID int, supplierName string) (float64, error) {
	// Fetch item DTO
	var dbItem Item
	err := db.Get(&dbItem, "SELECT * FROM items WHERE id = ?", itemID)
	if err != nil {
		return 0, err
	}

	// Map Item to domain.Item
	brandID := 0
	if dbItem.BrandID != nil {
		brandID = *dbItem.BrandID
	}
	categoryID := 0
	if dbItem.CategoryID != nil {
		categoryID = *dbItem.CategoryID
	}
	dItem, err := domain.NewItem(
		dbItem.ID,
		dbItem.Code,
		dbItem.Description,
		dbItem.DefaultUOM,
		dbItem.Model,
		brandID,
		categoryID,
		dbItem.Variation,
		dbItem.Remarks,
	)
	if err != nil {
		return 0, err
	}

	// Fetch UomSettings DTOs
	var dbUomSettings []UomSetting
	err = db.Select(&dbUomSettings, "SELECT * FROM uom_settings WHERE item_id = ?", itemID)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// Map to domain.UomSetting
	var dUomSettings []domain.UomSetting
	for _, u := range dbUomSettings {
		du, errU := domain.NewUomSetting(u.MUOM, u.ConversionFactor)
		if errU == nil {
			dUomSettings = append(dUomSettings, du)
		}
	}

	// Fetch ReceivingLog DTOs
	var dbReceivingLogs []ReceivingLog
	err = db.Select(&dbReceivingLogs, "SELECT * FROM receiving_logs WHERE item_id = ? AND supplier = ?", itemID, supplierName)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// Map to domain.ReceivingLog
	var dReceivingLogs []domain.ReceivingLog
	for _, r := range dbReceivingLogs {
		dr, errR := domain.NewReceivingLog(
			r.ID,
			r.Supplier,
			r.Date,
			r.PLNo,
			r.ItemID,
			r.Qty,
			r.UOM,
			r.UnitPrice,
			r.Cost,
		)
		if errR == nil {
			dReceivingLogs = append(dReceivingLogs, dr)
		}
	}

	// Fetch SalesDetail DTOs
	var dbSalesDetails []SalesDetail
	err = db.Select(&dbSalesDetails, "SELECT * FROM sales_details WHERE item_id = ? AND supplier = ?", itemID, supplierName)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// Map to domain.SalesDetail
	var dSalesDetails []domain.SalesDetail
	for _, s := range dbSalesDetails {
		ds, errS := domain.NewSalesDetail(
			s.ID,
			s.DocType,
			s.DocStatus,
			s.DocDate,
			s.DocNumber,
			s.CustomerName,
			s.Supplier,
			s.ItemID,
			s.Qty,
			s.UOM,
			s.Price,
			s.Cost,
		)
		if errS == nil {
			dSalesDetails = append(dSalesDetails, ds)
		}
	}

	// Fetch InventoryAdjustment DTOs
	var dbAdjustments []InventoryAdjustment
	err = db.Select(&dbAdjustments, "SELECT * FROM inventory_adjustments WHERE item_id = ?", itemID)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// Map to domain.InventoryAdjustment
	var dAdjustments []domain.InventoryAdjustment
	for _, a := range dbAdjustments {
		da, errA := domain.NewInventoryAdjustment(
			a.ID,
			a.AdjustmentID,
			a.Date,
			a.ItemID,
			a.UOM,
			a.AdjustmentQty,
			a.Cost,
			a.Remarks,
		)
		if errA == nil {
			dAdjustments = append(dAdjustments, da)
		}
	}

	// Calculate stock using domain aggregate
	stock := domain.NewItemStock(dItem, dUomSettings, dReceivingLogs, dSalesDetails, dAdjustments)
	return stock.CalculateOnHand(supplierName), nil
}
