package domain

import (
	"testing"
)

func TestNewCustomer(t *testing.T) {
	t.Run("valid customer with default payment terms", func(t *testing.T) {
		cust, err := NewCustomer(1, "CUST-001", "Acme Corporation", "123-456-789-000", "123 Main St", "John Doe", "09123456789", "john@acme.com", "", "Important client")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cust.Code != "CUST-001" {
			t.Errorf("expected code CUST-001, got %s", cust.Code)
		}
		if cust.Name != "Acme Corporation" {
			t.Errorf("expected name Acme Corporation, got %s", cust.Name)
		}
		if cust.PaymentTerms != "Net 30 Days" {
			t.Errorf("expected default terms 'Net 30 Days', got %s", cust.PaymentTerms)
		}
	})

	t.Run("valid customer with explicit payment terms", func(t *testing.T) {
		cust, err := NewCustomer(2, "CUST-002", "Globex Inc", "", "", "", "", "", "Cash on Delivery (COD)", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cust.PaymentTerms != "Cash on Delivery (COD)" {
			t.Errorf("expected 'Cash on Delivery (COD)', got %s", cust.PaymentTerms)
		}
	})

	t.Run("empty code returns error", func(t *testing.T) {
		_, err := NewCustomer(0, "   ", "Acme", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error for empty code, got nil")
		}
	})

	t.Run("empty name returns error", func(t *testing.T) {
		_, err := NewCustomer(0, "CUST-003", "", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})
}
