package domain_test

import (
	"testing"

	"rav/domain"
)

func TestNewItem(t *testing.T) {
	tests := []struct {
		name        string
		id          int
		code        string
		description string
		defaultUOM  string
		model       string
		brandID     int
		categoryID  int
		variation   string
		remarks     string
		wantErr     bool
	}{
		{"valid item", 1, "ITM1", "Item One", "PCS", "M1", 1, 1, "V1", "Rem", false},
		{"valid item with zero brand/category", 2, "  ITM2 ", " Item Two ", " BOX ", " M2 ", 0, 0, " V2 ", " Rem2 ", false},
		{"empty code", 1, "", "Item One", "PCS", "", 0, 0, "", "", true},
		{"whitespace code", 1, "   ", "Item One", "PCS", "", 0, 0, "", "", true},
		{"empty description", 1, "ITM1", "", "PCS", "", 0, 0, "", "", true},
		{"whitespace description", 1, "ITM1", "   \t ", "PCS", "", 0, 0, "", "", true},
		{"empty default UOM", 1, "ITM1", "Item One", "", "", 0, 0, "", "", true},
		{"whitespace default UOM", 1, "ITM1", "Item One", "   ", "", 0, 0, "", "", true},
		{"negative brand ID", 1, "ITM1", "Item One", "PCS", "", -1, 0, "", "", true},
		{"negative category ID", 1, "ITM1", "Item One", "PCS", "", 0, -2, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewItem(tt.id, tt.code, tt.description, tt.defaultUOM, tt.model, tt.brandID, tt.categoryID, tt.variation, tt.remarks)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewItem() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.Code == "" {
				t.Errorf("expected trimmed non-empty Code, got %+v", got)
			}
		})
	}
}
