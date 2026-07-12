package domain

import "errors"

// Brand represents a brand domain entity.
type Brand struct {
	ID   int
	Code string
	Name string
}

// NewBrand creates a new Brand entity validating code and name constraints.
func NewBrand(id int, code, name string) (Brand, error) {
	if code == "" {
		return Brand{}, errors.New("brand code cannot be empty")
	}
	if name == "" {
		return Brand{}, errors.New("brand name cannot be empty")
	}
	return Brand{
		ID:   id,
		Code: code,
		Name: name,
	}, nil
}
