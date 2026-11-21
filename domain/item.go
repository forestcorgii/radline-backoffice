package domain

import (
	"errors"
	"strings"
)

// Item represents an item domain aggregate root/entity.
type Item struct {
	ID          int
	Code        string
	Description string
	DefaultUOM  string
	Model       string
	BrandID     int
	CategoryID  int
	Variation   string
	Remarks     string
}

// NewItem creates a new Item entity validating core constraints.
func NewItem(id int, code, description, defaultUOM, model string, brandID, categoryID int, variation, remarks string) (Item, error) {
	code = strings.TrimSpace(code)
	description = strings.TrimSpace(description)
	defaultUOM = strings.TrimSpace(defaultUOM)
	model = strings.TrimSpace(model)
	variation = strings.TrimSpace(variation)
	remarks = strings.TrimSpace(remarks)

	if code == "" {
		return Item{}, errors.New("item code cannot be empty")
	}
	if description == "" {
		return Item{}, errors.New("item description cannot be empty")
	}
	if defaultUOM == "" {
		return Item{}, errors.New("item default UOM cannot be empty")
	}
	if brandID < 0 {
		return Item{}, errors.New("brand ID cannot be negative")
	}
	if categoryID < 0 {
		return Item{}, errors.New("category ID cannot be negative")
	}
	return Item{
		ID:          id,
		Code:        code,
		Description: description,
		DefaultUOM:  defaultUOM,
		Model:       model,
		BrandID:     brandID,
		CategoryID:  categoryID,
		Variation:   variation,
		Remarks:     remarks,
	}, nil
}
