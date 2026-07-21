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
	for i := range point {
		result[i] = ctx.center[i] + ctx.scale[i]*point[i]
	}
	return result
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
	return true
}
