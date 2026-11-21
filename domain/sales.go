package domain

import (
	"errors"
	"strings"
	"time"
)

// SalesDetail represents a sales transaction record.
type SalesDetail struct {
	ID           int
	DocType      string
	DocStatus    string
	DocDate      time.Time
	DocNumber    string
	CustomerName string
	TINNo        string
	Address      string
	Supplier     string
	ItemID       int
	Qty          float64
	UOM          string
	Price        float64
	TotalSales   float64
	Cost         float64
	TotalCost    float64
	Patong       float64
	POSCharge    float64
	WT2307       float64
	TotalRemit   float64
	Profit       float64
	ProfitMargin float64
	Remarks      string
	RefPL        string
}

// SaleItem represents an item line in a sale.
type SaleItem struct {
	ItemID       int
	Qty          float64
	UOM          string
	Price        float64
	Cost         float64
	TotalSales   float64
	TotalCost    float64
	Patong       float64
	POSCharge    float64
	WT2307       float64
	TotalRemit   float64
	Profit       float64
	ProfitMargin float64
	Remarks      string
	RefPL        string
}

// Sale represents a sale transaction aggregate.
type Sale struct {
	DocType      string
	DocStatus    string
	DocDate      time.Time
	DocNumber    string
	CustomerName string
	TINNo        string
	Address      string
	Supplier     string
	Items        []SaleItem
}

// SetCustomerDetails updates the customer TIN and Address on the Sale aggregate.
func (s *Sale) SetCustomerDetails(tinNo, address string) {
	s.TINNo = strings.TrimSpace(tinNo)
	s.Address = strings.TrimSpace(address)
}

func isValidStatus(status string) bool {
	switch status {
	case "Active", "ACTIVE", "Posted", "POSTED", "Cancelled/Return", "CANCELLED/RETURN":
		return true
	}
	return false
}

// NewSale constructs and validates a Sale aggregate.
func NewSale(docType, docStatus string, docDate time.Time, docNumber, customerName, supplier string, items []SaleItem) (Sale, error) {
	docType = strings.TrimSpace(docType)
	docNumber = strings.TrimSpace(docNumber)
	supplier = strings.TrimSpace(supplier)
	docStatus = strings.TrimSpace(docStatus)
	customerName = strings.TrimSpace(customerName)

	if docType == "" {
		return Sale{}, errors.New("doc type cannot be empty")
	}
	if docNumber == "" {
		return Sale{}, errors.New("doc number cannot be empty")
	}
	if supplier == "" {
		return Sale{}, errors.New("supplier cannot be empty")
	}
	if docDate.IsZero() {
		return Sale{}, errors.New("doc date must be valid")
	}
	if docStatus == "" {
		return Sale{}, errors.New("doc status cannot be empty")
	}
	if !isValidStatus(docStatus) {
		return Sale{}, errors.New("invalid doc status: " + docStatus)
	}
	if len(items) == 0 {
		return Sale{}, errors.New("sale must have at least one item")
	}

	validatedItems := make([]SaleItem, len(items))
	for i, item := range items {
		itemUOM := strings.TrimSpace(item.UOM)
		if item.ItemID <= 0 {
			return Sale{}, errors.New("item ID must be valid")
		}
		if !IsFinite(item.Qty) || item.Qty <= 0 {
			return Sale{}, errors.New("quantity must be greater than zero")
		}
		if itemUOM == "" {
			return Sale{}, errors.New("UOM cannot be empty")
		}
		if !IsFinite(item.Price) || item.Price < 0 {
			return Sale{}, errors.New("price cannot be negative")
		}
		if !IsFinite(item.Cost) || item.Cost < 0 {
			return Sale{}, errors.New("cost cannot be negative")
		}
		if !IsFinite(item.Patong) || item.Patong < 0 {
			return Sale{}, errors.New("patong cannot be negative")
		}
		if !IsFinite(item.POSCharge) || item.POSCharge < 0 {
			return Sale{}, errors.New("POS charge cannot be negative")
		}
		if !IsFinite(item.WT2307) || item.WT2307 < 0 {
			return Sale{}, errors.New("WT 2307 cannot be negative")
		}

		price := Round2(item.Price)
		cost := Round2(item.Cost)
		patong := Round2(item.Patong)
		posCharge := Round2(item.POSCharge)
		wt2307 := Round2(item.WT2307)

		totalSales := Round2(item.Qty * price)
		totalCost := Round2(item.Qty * cost)
		deductions := Round2(patong + posCharge + wt2307)
		if deductions > totalSales {
			return Sale{}, errors.New("total deductions cannot exceed total sales")
		}
		totalRemit := Round2(totalSales - deductions)
		profit := Round2(totalSales - totalCost - posCharge - patong - wt2307)
		profitMargin := 0.0
		if totalSales > 0 {
			profitMargin = Round2((profit / totalSales) * 100)
		}

		validatedItems[i] = SaleItem{
			ItemID:       item.ItemID,
			Qty:          item.Qty,
			UOM:          itemUOM,
			Price:        price,
			Cost:         cost,
			TotalSales:   totalSales,
			TotalCost:    totalCost,
			Patong:       patong,
			POSCharge:    posCharge,
			WT2307:       wt2307,
			TotalRemit:   totalRemit,
			Profit:       profit,
			ProfitMargin: profitMargin,
			Remarks:      strings.TrimSpace(item.Remarks),
			RefPL:        strings.TrimSpace(item.RefPL),
		}
	}

	return Sale{
		DocType:      docType,
		DocStatus:    docStatus,
		DocDate:      docDate,
		DocNumber:    docNumber,
		CustomerName: customerName,
		Supplier:     supplier,
		Items:        validatedItems,
	}, nil
}

// ToSalesDetails maps the Sale aggregate to a slice of flat SalesDetail records.
func (s Sale) ToSalesDetails() []SalesDetail {
	details := make([]SalesDetail, len(s.Items))
	for i, item := range s.Items {
		details[i] = SalesDetail{
			DocType:      s.DocType,
			DocStatus:    s.DocStatus,
			DocDate:      s.DocDate,
			DocNumber:    s.DocNumber,
			CustomerName: s.CustomerName,
			TINNo:        s.TINNo,
			Address:      s.Address,
			Supplier:     s.Supplier,
			ItemID:       item.ItemID,
			Qty:          item.Qty,
			UOM:          item.UOM,
			Price:        item.Price,
			TotalSales:   item.TotalSales,
			Cost:         item.Cost,
			TotalCost:    item.TotalCost,
			Patong:       item.Patong,
			POSCharge:    item.POSCharge,
			WT2307:       item.WT2307,
			TotalRemit:   item.TotalRemit,
			Profit:       item.Profit,
			ProfitMargin: item.ProfitMargin,
			Remarks:      item.Remarks,
			RefPL:        item.RefPL,
		}
	}
	return details
}

// NewSalesDetail creates a sales transaction, validating constraints and calculating total sales, cost, total remit, profit, and profit margin.
func NewSalesDetail(id int, docType, docStatus string, docDate time.Time, docNumber, customerName, supplier string, itemID int, qty float64, uom string, price, cost, patong, posCharge, wt2307 float64, remarks, refPL string) (SalesDetail, error) {
	docType = strings.TrimSpace(docType)
	docNumber = strings.TrimSpace(docNumber)
	supplier = strings.TrimSpace(supplier)
	docStatus = strings.TrimSpace(docStatus)
	customerName = strings.TrimSpace(customerName)
	uom = strings.TrimSpace(uom)

	if docType == "" {
		return SalesDetail{}, errors.New("doc type cannot be empty")
	}
	if docNumber == "" {
		return SalesDetail{}, errors.New("doc number cannot be empty")
	}
	if supplier == "" {
		return SalesDetail{}, errors.New("supplier cannot be empty")
	}
	if docDate.IsZero() {
		return SalesDetail{}, errors.New("doc date must be valid")
	}
	if docStatus == "" {
		return SalesDetail{}, errors.New("doc status cannot be empty")
	}
	if !isValidStatus(docStatus) {
		return SalesDetail{}, errors.New("invalid doc status: " + docStatus)
	}
	if itemID <= 0 {
		return SalesDetail{}, errors.New("item ID must be valid")
	}
	if !IsFinite(qty) || qty <= 0 {
		return SalesDetail{}, errors.New("quantity must be greater than zero")
	}
	if uom == "" {
		return SalesDetail{}, errors.New("UOM cannot be empty")
	}
	if !IsFinite(price) || price < 0 {
		return SalesDetail{}, errors.New("price cannot be negative")
	}
	if !IsFinite(cost) || cost < 0 {
		return SalesDetail{}, errors.New("cost cannot be negative")
	}
	if !IsFinite(patong) || patong < 0 {
		return SalesDetail{}, errors.New("patong cannot be negative")
	}
	if !IsFinite(posCharge) || posCharge < 0 {
		return SalesDetail{}, errors.New("POS charge cannot be negative")
	}
	if !IsFinite(wt2307) || wt2307 < 0 {
		return SalesDetail{}, errors.New("WT 2307 cannot be negative")
	}

	price = Round2(price)
	cost = Round2(cost)
	patong = Round2(patong)
	posCharge = Round2(posCharge)
	wt2307 = Round2(wt2307)

	totalSales := Round2(qty * price)
	totalCost := Round2(qty * cost)
	deductions := Round2(patong + posCharge + wt2307)
	if deductions > totalSales {
		return SalesDetail{}, errors.New("total deductions cannot exceed total sales")
	}
	totalRemit := Round2(totalSales - deductions)
	profit := Round2(totalSales - totalCost - posCharge - patong - wt2307)
	profitMargin := 0.0
	if totalSales > 0 {
		profitMargin = Round2((profit / totalSales) * 100)
	}

	return SalesDetail{
		ID:           id,
		DocType:      docType,
		DocStatus:    docStatus,
		DocDate:      docDate,
		DocNumber:    docNumber,
		CustomerName: customerName,
		Supplier:     supplier,
		ItemID:       itemID,
		Qty:          qty,
		UOM:          uom,
		Price:        price,
		TotalSales:   totalSales,
		Cost:         cost,
		TotalCost:    totalCost,
		Patong:       patong,
		POSCharge:    posCharge,
		WT2307:       wt2307,
		TotalRemit:   totalRemit,
		Profit:       profit,
		ProfitMargin: profitMargin,
		Remarks:      strings.TrimSpace(remarks),
		RefPL:        strings.TrimSpace(refPL),
	}, nil
}
