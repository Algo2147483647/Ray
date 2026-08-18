package adapt

import (
	"fmt"
	"math"
)

func optionalVector(object map[string]interface{}, key string, dimension int, fallback []float64) ([]float64, error) {
	raw, ok := object[key]
	if !ok {
		return append([]float64(nil), fallback...), nil
	}
	values, err := toFloat64Slice(raw)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	if len(values) != dimension {
		return nil, fmt.Errorf("field %q must contain %d values, got %d", key, dimension, len(values))
	}
	return values, nil
}

func vectorField(object map[string]interface{}, key string, dimension int) ([]float64, error) {
	raw, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}
	return vectorValue(key, raw, dimension)
}

func vectorValue(key string, raw interface{}, dimension int) ([]float64, error) {
	values, err := toFloat64Slice(raw)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	if len(values) != dimension {
		return nil, fmt.Errorf("field %q must contain %d values, got %d", key, dimension, len(values))
	}
	return values, nil
}

func objectCenter(object map[string]interface{}, dimension int) ([]float64, error) {
	if center, ok := object["center"]; ok {
		return vectorValue("center", center, dimension)
	}
	if position, ok := object["position"]; ok {
		return vectorValue("position", position, dimension)
	}
	return nil, fmt.Errorf(`missing required field "center"`)
}

func optionalObjectCenter(object map[string]interface{}, dimension int, fallback []float64) ([]float64, error) {
	if center, ok := object["center"]; ok {
		return vectorValue("center", center, dimension)
	}
	if position, ok := object["position"]; ok {
		return vectorValue("position", position, dimension)
	}
	return append([]float64(nil), fallback...), nil
}

func floatField(object map[string]interface{}, key string) (float64, error) {
	raw, ok := object[key]
	if !ok {
		return 0, fmt.Errorf("missing required field %q", key)
	}
	value, err := toFloat64(raw)
	if err != nil {
		return 0, fmt.Errorf("field %q: %w", key, err)
	}
	return value, nil
}

func optionalScale(object map[string]interface{}, key string, dimension int, fallback []float64) ([]float64, error) {
	raw, ok := object[key]
	if !ok {
		return append([]float64(nil), fallback...), nil
	}
	if values, err := toFloat64Slice(raw); err == nil {
		if len(values) != dimension {
			return nil, fmt.Errorf("field %q must contain %d values, got %d", key, dimension, len(values))
		}
		for i, value := range values {
			if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("scale index %d must be a finite positive number", i)
			}
		}
		return values, nil
	}

	value, err := toFloat64(raw)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("scale index 0 must be a finite positive number")
	}
	values := make([]float64, dimension)
	for i := range values {
		values[i] = value
	}
	return values, nil
}

func applyPlacement(ctx groupContext, point []float64) []float64 {
	result := make([]float64, len(point))
	for worldAxis := range point {
		result[worldAxis] = ctx.center[worldAxis]
		for localAxis := range point {
			result[worldAxis] += ctx.basis[worldAxis][localAxis] * ctx.scale[localAxis] * point[localAxis]
		}
	}
	return result
}

func applyDirection(ctx groupContext, direction []float64) []float64 {
	result := make([]float64, len(direction))
	for worldAxis := range direction {
		for localAxis := range direction {
			result[worldAxis] += ctx.basis[worldAxis][localAxis] * direction[localAxis]
		}
	}
	return result
}

func multiplyBasis(a, b [][]float64) [][]float64 {
	result := make([][]float64, len(a))
	for row := range result {
		result[row] = make([]float64, len(a))
		for col := range result[row] {
			for k := range a[row] {
				result[row][col] += a[row][k] * b[k][col]
			}
		}
	}
	return result
}

func basisIsIdentity(basis [][]float64) bool {
	for row := range basis {
		for col := range basis[row] {
			expected := 0.0
			if row == col {
				expected = 1
			}
			if !nearlyEqual(basis[row][col], expected) {
				return false
			}
		}
	}
	return true
}

func uniformScaleVector(scale []float64) bool {
	for axis := 1; axis < len(scale); axis++ {
		if !nearlyEqual(scale[axis], scale[0]) {
			return false
		}
	}
	return true
}

func addVectors(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

func groupPlacementIsIdentity(ctx groupContext) bool {
	for i := 0; i < ctx.dimension; i++ {
		if ctx.center[i] != 0 || ctx.scale[i] != 1 {
			return false
		}
	}
	return basisIsIdentity(ctx.basis)
}
