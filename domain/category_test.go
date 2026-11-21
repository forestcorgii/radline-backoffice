package domain_test

import (
	"testing"

	"rav/domain"
)

func TestNewCategory(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		code     string
		catNm    string
		wantCode string
		wantName string
		wantErr  bool
	}{
		{"valid category", 1, "CAT1", "Category One", "CAT1", "Category One", false},
		{"trims whitespace", 2, "  CAT2  ", "  Category Two  ", "CAT2", "Category Two", false},
		{"empty code", 1, "", "Category One", "", "", true},
		{"whitespace-only code", 1, "   \t\n  ", "Category One", "", "", true},
		{"empty name", 1, "CAT1", "", "", "", true},
		{"whitespace-only name", 1, "CAT1", "   \t\n  ", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewCategory(tt.id, tt.code, tt.catNm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewCategory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Code != tt.wantCode || got.Name != tt.wantName || got.ID != tt.id {
					t.Errorf("NewCategory() = %+v, want Code=%q Name=%q ID=%d", got, tt.wantCode, tt.wantName, tt.id)
				}
			}
		})
	}
}
