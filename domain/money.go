package domain

import "math"

// Round2 rounds a monetary or numeric value to 2 decimal places.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// IsFinite returns true if v is neither NaN nor ±Inf.
func IsFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
