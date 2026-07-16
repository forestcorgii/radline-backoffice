package domain_test

import (
	"testing"

	"radline/domain"
)

func TestUomSetting_ValidationRules(t *testing.T) {
	tests := []struct {
		name    string
		muom    string
		factor  float64
		wantErr bool
	}{
		{
			name:    "valid setting",
			muom:    "BOX",
			factor:  10.0,
			wantErr: false,
		},
		{
			name:    "empty MUOM",
			muom:    "",
			factor:  10.0,
			wantErr: true,
		},
		{
			name:    "zero factor",
			muom:    "BOX",
			factor:  0,
			wantErr: true,
		},
		{
			name:    "negative factor",
			muom:    "BOX",
			factor:  -5.5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewUomSetting(tt.muom, tt.factor)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUomSetting() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUomSetting_Conversions(t *testing.T) {
	setting, err := domain.NewUomSetting("BOX", 12.0)
	if err != nil {
		t.Fatalf("failed to create valid setting: %v", err)
	}

	// Test ConvertToBase
	t.Run("ConvertToBase", func(t *testing.T) {
		got := setting.ConvertToBase(5.0)
		want := 60.0
		if got != want {
			t.Errorf("ConvertToBase(5) = %v; want %v", got, want)
		}
	})

	// Test ConvertToMultiple
	t.Run("ConvertToMultiple", func(t *testing.T) {
		got := setting.ConvertToMultiple(24.0)
		want := 2.0
		if got != want {
			t.Errorf("ConvertToMultiple(24) = %v; want %v", got, want)
		}
	})
}
