package domain

import "errors"

// Category represents a category domain entity.
type Category struct {
	ID   int
	Code string
	Name string
}

// NewCategory creates a new Category entity validating code and name constraints.
func NewCategory(id int, code, name string) (Category, error) {
	if code == "" {
		return Category{}, errors.New("category code cannot be empty")
	}
	if name == "" {
		return Category{}, errors.New("category name cannot be empty")
	}
	return Category{
		ID:   id,
		Code: code,
		Name: name,
	}, nil
}
