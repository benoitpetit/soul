// Package util provides common utility functions for SOUL.
package util

// Min returns the smaller of two float64 values.
func Min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of two float64 values.
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Abs returns the absolute value of a float64.
func Abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

// MaxInt returns the larger of two int values.
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the smaller of two int values.
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxFloat returns the larger of two float64 values.
func MaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
