package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/utils"
)

func normalizePolynomialCenterScale(center, scale []float64) ([3]float64, [3]float64, error) {
	normalizedCenter := [3]float64{}
	normalizedScale := [3]float64{1, 1, 1}

	if center != nil {
		if err := utils.RequireSliceLength("center", center, utils.Dimension); err != nil {
			return normalizedCenter, normalizedScale, err
		}
		copy(normalizedCenter[:], center)
	}
	if scale != nil {
		if err := utils.RequireSliceLength("scale", scale, utils.Dimension); err != nil {
			return normalizedCenter, normalizedScale, err
		}
		for i, value := range scale {
			if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return normalizedCenter, normalizedScale, errInvalidScale(i)
			}
			normalizedScale[i] = value
		}
	}
	return normalizedCenter, normalizedScale, nil
}

func errInvalidScale(index int) error {
	return fmt.Errorf("scale index %d must be a finite positive number", index)
}
