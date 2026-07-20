package factory

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Algo2147483647/ray/engine/utils"
)

func requiredPolynomialCoefficients(objDef map[string]interface{}, order int) ([]float64, error) {
	value, fieldName, err := requiredPolynomialCoefficientValue(objDef)
	if err != nil {
		return nil, err
	}

	total := 1
	for i := 0; i < order; i++ {
		total *= 4
	}

	if values, err := utils.ToFloat64Slice(value); err == nil {
		if err := utils.RequireSliceLength(fieldName, values, total); err != nil {
			return nil, err
		}
		return values, nil
	}

	sparse, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array or object, got %T", fieldName, value)
	}

	return parseSparsePolynomialCoefficients(fieldName, sparse, order, total)
}

func requiredPolynomialCoefficientValue(objDef map[string]interface{}) (interface{}, string, error) {
	lower, hasLower := objDef["a"]
	upper, hasUpper := objDef["A"]

	if hasLower && hasUpper {
		return nil, "", fmt.Errorf(`fields "a" and "A" cannot both be provided`)
	}
	if hasLower {
		return lower, "a", nil
	}
	if hasUpper {
		return upper, "A", nil
	}
	return nil, "", fmt.Errorf(`missing required field "a"`)
}

func parseSparsePolynomialCoefficients(fieldName string, sparse map[string]interface{}, order, total int) ([]float64, error) {
	coeffs := make([]float64, total)
	keyStyle := ""

	for key, rawValue := range sparse {
		index, style, err := sparsePolynomialIndex(key, order, total)
		if err != nil {
			return nil, fmt.Errorf("field %q key %q: %w", fieldName, key, err)
		}
		if keyStyle == "" {
			keyStyle = style
		} else if keyStyle != style {
			return nil, fmt.Errorf("field %q cannot mix flat and coordinate sparse keys", fieldName)
		}

		value, err := utils.RequiredFloat64Field(map[string]interface{}{"value": rawValue}, "value")
		if err != nil {
			return nil, fmt.Errorf("field %q key %q: %w", fieldName, key, err)
		}
		coeffs[index] = value
	}

	return coeffs, nil
}

func sparsePolynomialIndex(key string, order, total int) (int, string, error) {
	if strings.Contains(key, ",") {
		index, err := sparsePolynomialCoordinateIndex(key, order)
		return index, "coordinate", err
	}

	index, err := strconv.Atoi(strings.TrimSpace(key))
	if err != nil {
		return 0, "", fmt.Errorf("expected integer flat index")
	}
	if index < 0 || index >= total {
		return 0, "", fmt.Errorf("flat index must be in [0,%d], got %d", total-1, index)
	}
	return index, "flat", nil
}

func sparsePolynomialCoordinateIndex(key string, order int) (int, error) {
	parts := strings.Split(key, ",")
	if len(parts) != order {
		return 0, fmt.Errorf("coordinate key must contain %d indices, got %d", order, len(parts))
	}

	index := 0
	for position, part := range parts {
		coordinate, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return 0, fmt.Errorf("coordinate %d must be an integer", position)
		}
		if coordinate < 0 || coordinate >= 4 {
			return 0, fmt.Errorf("coordinate %d must be in [0,3], got %d", position, coordinate)
		}
		index = index*4 + coordinate
	}
	return index, nil
}

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
				return normalizedCenter, normalizedScale, fmt.Errorf("scale index %d must be a finite positive number", i)
			}
			normalizedScale[i] = value
		}
	}
	return normalizedCenter, normalizedScale, nil
}
