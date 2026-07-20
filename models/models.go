package models

import "time"

type Brand struct {
	ID        int    `db:"id"`
	Code      string `db:"code"`
	Name      string `db:"name"`
	ItemCount int    `db:"item_count"`
}

type Category struct {
	ID        int    `db:"id"`
	Code      string `db:"code"`
	Name      string `db:"name"`
	ItemCount int    `db:"item_count"`
}

type Item struct {
	ID          int    `db:"id"`
	Code        string `db:"code"`
	Description string `db:"description"`
	DefaultUOM  string `db:"default_uom"`
	Model       string `db:"model"`
	BrandID     *int   `db:"brand_id"`
	CategoryID  *int   `db:"category_id"`
	Variation   string `db:"variation"`
	Remarks     string `db:"remarks"`
}

type ItemWithRelations struct {
	ID           int    `db:"id"`
	Code         string `db:"code"`
	Description  string `db:"description"`
	DefaultUOM   string `db:"default_uom"`
	Model        string `db:"model"`
	BrandID      *int   `db:"brand_id"`
	CategoryID   *int   `db:"category_id"`
	Variation    string `db:"variation"`
	Remarks      string `db:"remarks"`
	BrandName    string `db:"brand_name"`
	CategoryName string `db:"category_name"`
}

type UomSetting struct {
	ID               int     `db:"id"`
	ItemID           int     `db:"item_id"`
	MUOM             string  `db:"muom"`
	ConversionFactor float64 `db:"conversion_factor"`
}

type ReceivingLog struct {
	ID           int       `db:"id"`
	Supplier     string    `db:"supplier"`
	Date         time.Time `db:"date"`
	PLNo         string    `db:"pl_no"`
	ItemID       int       `db:"item_id"`
	Qty          float64   `db:"qty"`
	UOM          string    `db:"uom"`
	UnitPrice    float64   `db:"unit_price"`
	Less1        float64   `db:"less1"`
	Less2        float64   `db:"less2"`
	Cost         float64   `db:"cost"`
	TotalCost    float64   `db:"total_cost"`
	Markup       float64   `db:"markup"`
	SellingPrice float64   `db:"selling_price"`
	Remarks      string    `db:"remarks"`
}

type ReceivingLogWithItem struct {
	ID           int       `db:"id"`
	Supplier     string    `db:"supplier"`
	Date         time.Time `db:"date"`
	PLNo         string    `db:"pl_no"`
	ItemID       int       `db:"item_id"`
	Qty          float64   `db:"qty"`
	UOM          string    `db:"uom"`
	UnitPrice    float64   `db:"unit_price"`
	Less1        float64   `db:"less1"`
	Less2        float64   `db:"less2"`
	Cost         float64   `db:"cost"`
	TotalCost    float64   `db:"total_cost"`
	Markup       float64   `db:"markup"`
	SellingPrice float64   `db:"selling_price"`
	Remarks      string    `db:"remarks"`
	ItemCode     string    `db:"item_code"`
	Description  string    `db:"item_description"`
}

type SalesDetail struct {
	ID           int       `db:"id"`
	DocType      string    `db:"doc_type"`
	DocStatus    string    `db:"doc_status"`
	DocDate      time.Time `db:"doc_date"`
	DocNumber    string    `db:"doc_number"`
	CustomerName string    `db:"customer_name"`
	Supplier     string    `db:"supplier"`
	ItemID       int       `db:"item_id"`
	Qty          float64   `db:"qty"`
	UOM          string    `db:"uom"`
	Price        float64   `db:"price"`
	TotalSales   float64   `db:"total_sales"`
	Cost         float64   `db:"cost"`
	TotalCost    float64   `db:"total_cost"`
	Patong       float64   `db:"patong"`
	POSCharge    float64   `db:"pos_charge"`
	WT2307       float64   `db:"wt_2307"`
	TotalRemit   float64   `db:"total_remit"`
	Profit       float64   `db:"profit"`
	ProfitMargin float64   `db:"profit_margin"`
	Remarks      string    `db:"remarks"`
	RefPL        *string   `db:"ref_pl"`
}

type SalesDetailWithItem struct {
	ID           int       `db:"id"`
	DocType      string    `db:"doc_type"`
	DocStatus    string    `db:"doc_status"`
	DocDate      time.Time `db:"doc_date"`
	DocNumber    string    `db:"doc_number"`
	CustomerName string    `db:"customer_name"`
	Supplier     string    `db:"supplier"`
	ItemID       int       `db:"item_id"`
	Qty          float64   `db:"qty"`
	UOM          string    `db:"uom"`
	Price        float64   `db:"price"`
	TotalSales   float64   `db:"total_sales"`
	Cost         float64   `db:"cost"`
	TotalCost    float64   `db:"total_cost"`
	Patong       float64   `db:"patong"`
	POSCharge    float64   `db:"pos_charge"`
	WT2307       float64   `db:"wt_2307"`
	TotalRemit   float64   `db:"total_remit"`
	Profit       float64   `db:"profit"`
	ProfitMargin float64   `db:"profit_margin"`
	Remarks      string    `db:"remarks"`
	ItemCode     string    `db:"item_code"`
	Description  string    `db:"item_description"`
	RefPL        *string   `db:"ref_pl"`
}

type InventoryAdjustment struct {
	ID            int       `db:"id"`
	AdjustmentID  *int      `db:"adjustment_id"`
	Date          time.Time `db:"date"`
	ItemID        int       `db:"item_id"`
	UOM           string    `db:"uom"`
	AdjustmentQty float64   `db:"adjustment_qty"`
	Cost          float64   `db:"cost"`
	Remarks       string    `db:"remarks"`
}

type InventoryAdjustmentWithItem struct {
	ID            int       `db:"id"`
	AdjustmentID  *int      `db:"adjustment_id"`
	Date          time.Time `db:"date"`
	ItemID        int       `db:"item_id"`
	UOM           string    `db:"uom"`
	AdjustmentQty float64   `db:"adjustment_qty"`
	Cost          float64   `db:"cost"`
	Remarks       string    `db:"remarks"`
	ItemCode      string    `db:"item_code"`
	Description   string    `db:"item_description"`
}

// ItemStockView is a read model for the inventory stock list view
type ItemStockView struct {
	ItemID        int     `db:"item_id"`
	ItemCode      string  `db:"item_code"`
	Description   string  `db:"description"`
	DefaultUOM    string  `db:"default_uom"`
	TotalReceived float64 `db:"total_received"`
	TotalSold     float64 `db:"total_sold"`
	TotalAdjusted float64 `db:"total_adjusted"`
	OnHand        float64 `db:"on_hand"`
	CurrentCost   float64 `db:"current_cost"`
	CurrentPrice  float64 `db:"current_price"`
}

type UomSettingWithItem struct {
	ID               int     `db:"id"`
	ItemID           int     `db:"item_id"`
	ItemCode         string  `db:"item_code"`
	ItemDescription  string  `db:"item_description"`
	UOM              string  `db:"uom"`
	MUOM             string  `db:"muom"`
	ConversionFactor float64 `db:"conversion_factor"`
}

type Uom struct {
	ID   int    `db:"id"`
	Code string `db:"code"`
}
