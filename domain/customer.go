package domain

import (
	"errors"
	"strings"
)

// Customer represents a customer domain entity.
type Customer struct {
	ID            int
	Code          string
	Name          string
	TINNo         string
	Address       string
	ContactPerson string
	Phone         string
	Email         string
	PaymentTerms  string
	Notes         string
}

// NewCustomer creates a validated Customer entity.
func NewCustomer(id int, code, name, tinNo, address, contactPerson, phone, email, paymentTerms, notes string) (Customer, error) {
	trimmedCode := strings.TrimSpace(code)
	trimmedName := strings.TrimSpace(name)

	if trimmedCode == "" {
		return Customer{}, errors.New("customer code cannot be empty")
	}
	if trimmedName == "" {
		return Customer{}, errors.New("customer name cannot be empty")
	}

	terms := strings.TrimSpace(paymentTerms)
	if terms == "" {
		terms = "Net 30 Days"
	}

	return Customer{
		ID:            id,
		Code:          trimmedCode,
		Name:          trimmedName,
		TINNo:         strings.TrimSpace(tinNo),
		Address:       strings.TrimSpace(address),
		ContactPerson: strings.TrimSpace(contactPerson),
		Phone:         strings.TrimSpace(phone),
		Email:         strings.TrimSpace(email),
		PaymentTerms:  terms,
		Notes:         strings.TrimSpace(notes),
	}, nil
}
