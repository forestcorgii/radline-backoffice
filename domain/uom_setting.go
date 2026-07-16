package domain

import "errors"

// UomSetting represents a conversion factor value object/entity.
type UomSetting struct {
	MUOM             string
	ConversionFactor float64
}

// NewUomSetting creates a UomSetting ensuring UOM and factor constraints.
func NewUomSetting(muom string, factor float64) (UomSetting, error) {
	if muom == "" {
		return UomSetting{}, errors.New("UOM code cannot be empty")
	}
	if factor <= 0 {
		return UomSetting{}, errors.New("conversion factor must be greater than zero")
	}
	return UomSetting{
		MUOM:             muom,
		ConversionFactor: factor,
	}, nil
}

// ConvertToBase converts multiple units quantity to base units quantity.
func (u UomSetting) ConvertToBase(qty float64) float64 {
	return qty * u.ConversionFactor
}

// ConvertToMultiple converts base units quantity to multiple units quantity.
func (u UomSetting) ConvertToMultiple(qty float64) float64 {
	if u.ConversionFactor == 0 {
		return 0
	}
	return qty / u.ConversionFactor
}
