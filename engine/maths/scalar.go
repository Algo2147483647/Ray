package maths

import "math"

func IsFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func Clamp(value, minimum, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func ClampUnit(value float64) float64 {
	return Clamp(value, 0, 1)
}

func Lerp(minimum, maximum, t float64) float64 {
	return minimum + (maximum-minimum)*t
}

func PositiveMod(value, period float64) float64 {
	value = math.Mod(value, period)
	if value < 0 {
		value += period
	}
	return value
}

func SignChanged(a, b float64) bool {
	return (a < 0 && b > 0) || (a > 0 && b < 0)
}
