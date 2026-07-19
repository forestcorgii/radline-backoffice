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
			r.Less1,
			r.Less2,
			r.Cost,
			r.Markup,
			r.Remarks,
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
		refPL := ""
		if s.RefPL != nil {
			refPL = *s.RefPL
		}
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
			refPL,
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
		adjID := 0
		if a.AdjustmentID != nil {
			adjID = *a.AdjustmentID
		}
		da, errA := domain.NewInventoryAdjustment(
			a.ID,
			adjID,
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

// FetchItemStock loads all transaction data for a specific item and constructs the domain.ItemStock aggregate.
func FetchItemStock(db *sqlx.DB, itemID int) (domain.ItemStock, error) {
	// Fetch item DTO
	var dbItem Item
	err := db.Get(&dbItem, "SELECT * FROM items WHERE id = ?", itemID)
	if err != nil {
		return domain.ItemStock{}, err
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
		return domain.ItemStock{}, err
	}

	// Fetch UomSettings DTOs
	var dbUomSettings []UomSetting
	err = db.Select(&dbUomSettings, "SELECT * FROM uom_settings WHERE item_id = ?", itemID)
	if err != nil && err != sql.ErrNoRows {
		return domain.ItemStock{}, err
	}

	// Map to domain.UomSetting
	var dUomSettings []domain.UomSetting
	for _, u := range dbUomSettings {
		du, errU := domain.NewUomSetting(u.MUOM, u.ConversionFactor)
		if errU == nil {
			dUomSettings = append(dUomSettings, du)
		}
	}

	// Fetch ReceivingLog DTOs (without supplier filter)
	var dbReceivingLogs []ReceivingLog
	err = db.Select(&dbReceivingLogs, "SELECT * FROM receiving_logs WHERE item_id = ?", itemID)
	if err != nil && err != sql.ErrNoRows {
		return domain.ItemStock{}, err
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
			r.Less1,
			r.Less2,
			r.Cost,
			r.Markup,
			r.Remarks,
		)
		if errR == nil {
			dReceivingLogs = append(dReceivingLogs, dr)
		}
	}

	// Fetch SalesDetail DTOs (without supplier filter)
	var dbSalesDetails []SalesDetail
	err = db.Select(&dbSalesDetails, "SELECT * FROM sales_details WHERE item_id = ?", itemID)
	if err != nil && err != sql.ErrNoRows {
		return domain.ItemStock{}, err
	}

	// Map to domain.SalesDetail
	var dSalesDetails []domain.SalesDetail
	for _, s := range dbSalesDetails {
		refPL := ""
		if s.RefPL != nil {
			refPL = *s.RefPL
		}
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
			refPL,
		)
		if errS == nil {
			dSalesDetails = append(dSalesDetails, ds)
		}
	}

	// Fetch InventoryAdjustment DTOs
	var dbAdjustments []InventoryAdjustment
	err = db.Select(&dbAdjustments, "SELECT * FROM inventory_adjustments WHERE item_id = ?", itemID)
	if err != nil && err != sql.ErrNoRows {
		return domain.ItemStock{}, err
	}

	// Map to domain.InventoryAdjustment
	var dAdjustments []domain.InventoryAdjustment
	for _, a := range dbAdjustments {
		adjID := 0
		if a.AdjustmentID != nil {
			adjID = *a.AdjustmentID
		}
		da, errA := domain.NewInventoryAdjustment(
			a.ID,
			adjID,
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

	return domain.NewItemStock(dItem, dUomSettings, dReceivingLogs, dSalesDetails, dAdjustments), nil
}

// FetchItemsStockBatch loads all transaction data for multiple items and constructs domain.ItemStock aggregates in batch.
func FetchItemsStockBatch(db *sqlx.DB, itemIDs []int) (map[int]domain.ItemStock, error) {
	if len(itemIDs) == 0 {
		return make(map[int]domain.ItemStock), nil
	}

	// 1. Fetch items
	query, args, err := sqlx.In("SELECT * FROM items WHERE id IN (?)", itemIDs)
	if err != nil {
		return nil, err
	}
	var dbItems []Item
	err = db.Select(&dbItems, query, args...)
	if err != nil {
		return nil, err
	}

	itemsMap := make(map[int]Item)
	for _, item := range dbItems {
		itemsMap[item.ID] = item
	}

	// 2. Fetch UomSettings
	query, args, err = sqlx.In("SELECT * FROM uom_settings WHERE item_id IN (?)", itemIDs)
	if err != nil {
		return nil, err
	}
	var dbUomSettings []UomSetting
	err = db.Select(&dbUomSettings, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	uomMap := make(map[int][]UomSetting)
	for _, u := range dbUomSettings {
		uomMap[u.ItemID] = append(uomMap[u.ItemID], u)
	}

	// 3. Fetch ReceivingLogs
	query, args, err = sqlx.In("SELECT * FROM receiving_logs WHERE item_id IN (?)", itemIDs)
	if err != nil {
		return nil, err
	}
	var dbReceivingLogs []ReceivingLog
	err = db.Select(&dbReceivingLogs, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	recMap := make(map[int][]ReceivingLog)
	for _, r := range dbReceivingLogs {
		recMap[r.ItemID] = append(recMap[r.ItemID], r)
	}

	// 4. Fetch SalesDetails
	query, args, err = sqlx.In("SELECT * FROM sales_details WHERE item_id IN (?)", itemIDs)
	if err != nil {
		return nil, err
	}
	var dbSalesDetails []SalesDetail
	err = db.Select(&dbSalesDetails, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	salesMap := make(map[int][]SalesDetail)
	for _, s := range dbSalesDetails {
		salesMap[s.ItemID] = append(salesMap[s.ItemID], s)
	}

	// 5. Fetch InventoryAdjustments
	query, args, err = sqlx.In("SELECT * FROM inventory_adjustments WHERE item_id IN (?)", itemIDs)
	if err != nil {
		return nil, err
	}
	var dbAdjustments []InventoryAdjustment
	err = db.Select(&dbAdjustments, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	adjMap := make(map[int][]InventoryAdjustment)
	for _, a := range dbAdjustments {
		adjMap[a.ItemID] = append(adjMap[a.ItemID], a)
	}

	result := make(map[int]domain.ItemStock)
	for _, itemID := range itemIDs {
		dbItem, exists := itemsMap[itemID]
		if !exists {
			continue
		}

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
			continue
		}

		var dUomSettings []domain.UomSetting
		for _, u := range uomMap[itemID] {
			du, errU := domain.NewUomSetting(u.MUOM, u.ConversionFactor)
			if errU == nil {
				dUomSettings = append(dUomSettings, du)
			}
		}

		var dReceivingLogs []domain.ReceivingLog
		for _, r := range recMap[itemID] {
			dr, errR := domain.NewReceivingLog(
				r.ID,
				r.Supplier,
				r.Date,
				r.PLNo,
				r.ItemID,
				r.Qty,
				r.UOM,
				r.UnitPrice,
				r.Less1,
				r.Less2,
				r.Cost,
				r.Markup,
				r.Remarks,
			)
			if errR == nil {
				dReceivingLogs = append(dReceivingLogs, dr)
			}
		}

		var dSalesDetails []domain.SalesDetail
		for _, s := range salesMap[itemID] {
			refPL := ""
			if s.RefPL != nil {
				refPL = *s.RefPL
			}
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
				refPL,
			)
			if errS == nil {
				dSalesDetails = append(dSalesDetails, ds)
			}
		}

		var dAdjustments []domain.InventoryAdjustment
		for _, a := range adjMap[itemID] {
			adjID := 0
			if a.AdjustmentID != nil {
				adjID = *a.AdjustmentID
			}
			da, errA := domain.NewInventoryAdjustment(
				a.ID,
				adjID,
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

		result[itemID] = domain.NewItemStock(dItem, dUomSettings, dReceivingLogs, dSalesDetails, dAdjustments)
	}

	return result, nil
}
