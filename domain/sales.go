package domain

import (
	"errors"
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
	Supplier     string
	ItemID       int
	Qty          float64
	UOM          string
	Price        float64
	TotalSales   float64
	Cost         float64
	TotalCost    float64
	Profit       float64
}

// SaleItem represents an item line in a sale.
type SaleItem struct {
	ItemID     int
	Qty        float64
	UOM        string
	Price      float64
	Cost       float64
	TotalSales float64
	TotalCost  float64
	Profit     float64
}

// Sale represents a sale transaction aggregate.
type Sale struct {
	DocType      string
	DocStatus    string
	DocDate      time.Time
	DocNumber    string
	CustomerName string
	Supplier     string
	Items        []SaleItem
}

// NewSale constructs and validates a Sale aggregate.
func NewSale(docType, docStatus string, docDate time.Time, docNumber, customerName, supplier string, items []SaleItem) (Sale, error) {
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
	if len(items) == 0 {
		return Sale{}, errors.New("sale must have at least one item")
	}

	validatedItems := make([]SaleItem, len(items))
	for i, item := range items {
		if item.ItemID <= 0 {
			return Sale{}, errors.New("item ID must be valid")
		}
		if item.Qty <= 0 {
			return Sale{}, errors.New("quantity must be greater than zero")
		}
		if item.UOM == "" {
			return Sale{}, errors.New("UOM cannot be empty")
		}
		if item.Price < 0 {
			return Sale{}, errors.New("price cannot be negative")
		}
		if item.Cost < 0 {
			return Sale{}, errors.New("cost cannot be negative")
		}

		totalSales := item.Qty * item.Price
		totalCost := item.Qty * item.Cost
		profit := totalSales - totalCost

		validatedItems[i] = SaleItem{
			ItemID:     item.ItemID,
			Qty:        item.Qty,
			UOM:        item.UOM,
			Price:      item.Price,
			Cost:       item.Cost,
			TotalSales: totalSales,
			TotalCost:  totalCost,
			Profit:     profit,
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
			Supplier:     s.Supplier,
			ItemID:       item.ItemID,
			Qty:          item.Qty,
			UOM:          item.UOM,
			Price:        item.Price,
			TotalSales:   item.TotalSales,
			Cost:         item.Cost,
			TotalCost:    item.TotalCost,
			Profit:       item.Profit,
		}
	}
	return details
}

// NewSalesDetail creates a sales transaction, validating constraints and calculating total sales, cost, and profit.
func NewSalesDetail(id int, docType, docStatus string, docDate time.Time, docNumber, customerName, supplier string, itemID int, qty float64, uom string, price, cost float64) (SalesDetail, error) {
	if docType == "" {
		return SalesDetail{}, errors.New("doc type cannot be empty")
	}
	if docNumber == "" {
		return SalesDetail{}, errors.New("doc number cannot be empty")
	}
	if supplier == "" {
		return SalesDetail{}, errors.New("supplier cannot be empty")
	}
	if itemID <= 0 {
		return SalesDetail{}, errors.New("item ID must be valid")
	}
	if qty <= 0 {
		return SalesDetail{}, errors.New("quantity must be greater than zero")
	}
	if uom == "" {
		return SalesDetail{}, errors.New("UOM cannot be empty")
	}
	if price < 0 {
		return SalesDetail{}, errors.New("price cannot be negative")
	}
	if cost < 0 {
		return SalesDetail{}, errors.New("cost cannot be negative")
	}

	totalSales := qty * price
	totalCost := qty * cost
	profit := totalSales - totalCost

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
		Profit:       profit,
	}, nil
}
