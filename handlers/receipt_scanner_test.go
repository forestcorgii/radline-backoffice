package handlers

import (
	"testing"
)

func TestParseOCRText_SampleReceipt(t *testing.T) {
	// Sample OCR output from engine 2
	sampleText := `RADLINE INDUSTRIAL TOOLS SUPPLIES
Hardware Store
Contact No.: 0931-219-0782 / 0930-943-3568
QUOTATION SLIP
No.
3907
IVAN
Date
7/14,2026
Customer_
Address
Quantity
I PC
DESCRIPTION
WADEOU WPB2915
P. BRUSH
Unit Price
Amount
30,
By:_
Customer's Signature`

	docType, docNumber, docDate, supplier, customer, items := parseOCRText(sampleText)

	if docType != "SALES" {
		t.Errorf("Expected DocType SALES, got %s", docType)
	}
	if docNumber != "3907" {
		t.Errorf("Expected DocNumber 3907, got %s", docNumber)
	}
	if docDate != "2026-07-14" {
		t.Errorf("Expected DocDate 2026-07-14, got %s", docDate)
	}
	if supplier != "RADLINE INDUSTRIAL TOOLS SUPPLIES" {
		t.Errorf("Expected supplier RADLINE INDUSTRIAL TOOLS SUPPLIES, got %s", supplier)
	}
	if customer != "IVAN" {
		t.Errorf("Expected customer IVAN, got %s", customer)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].ItemDescription != "WADFOW WPB2915 P. BRUSH" {
		t.Errorf("Expected description 'WADFOW WPB2915 P. BRUSH', got %s", items[0].ItemDescription)
	}
	if items[0].Qty != 1.0 {
		t.Errorf("Expected Qty 1.0, got %f", items[0].Qty)
	}
	if items[0].Price != 30.00 {
		t.Errorf("Expected Price 30.00, got %f", items[0].Price)
	}
}

func TestParseOCRText_GenericReceipt(t *testing.T) {
	genericText := `ACME CORP
No. 98765
Date: 12/25/2026
Customer: John Doe
INVOICE`

	docType, docNumber, docDate, supplier, customer, _ := parseOCRText(genericText)

	if docType != "SI" {
		t.Errorf("Expected DocType SI, got %s", docType)
	}
	if docNumber != "98765" {
		t.Errorf("Expected DocNumber 98765, got %s", docNumber)
	}
	if docDate != "2026-12-25" {
		t.Errorf("Expected DocDate 2026-12-25, got %s", docDate)
	}
	if supplier != "ACME CORP" {
		t.Errorf("Expected supplier ACME CORP, got %s", supplier)
	}
	if customer != "John Doe" {
		t.Errorf("Expected customer John Doe, got %s", customer)
	}
}
