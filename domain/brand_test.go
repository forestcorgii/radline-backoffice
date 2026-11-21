package domain_test

import (
	"testing"

	"rav/domain"
)

func TestNewBrand(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		code     string
		brandNm  string
		wantCode string
		wantName string
		wantErr  bool
	}{
		{"valid brand", 1, "BRD1", "Brand One", "BRD1", "Brand One", false},
		{"trims whitespace", 2, "  BRD2  ", "  Brand Two  ", "BRD2", "Brand Two", false},
		{"empty code", 1, "", "Brand One", "", "", true},
		{"whitespace-only code", 1, "   \t\n  ", "Brand One", "", "", true},
		{"empty name", 1, "BRD1", "", "", "", true},
		{"whitespace-only name", 1, "BRD1", "   \t\n  ", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewBrand(tt.id, tt.code, tt.brandNm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewBrand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Code != tt.wantCode || got.Name != tt.wantName || got.ID != tt.id {
					t.Errorf("NewBrand() = %+v, want Code=%q Name=%q ID=%d", got, tt.wantCode, tt.wantName, tt.id)
				}
			}
		})
	}
}
