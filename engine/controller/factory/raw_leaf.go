package factory

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/utils"
)

// decodeRawObject is restricted to intentionally polymorphic leaf protocols
// (expression fields, curves, surfaces, and coefficient encodings).
func decodeRawObject(raw json.RawMessage, field string) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing required field %q", field)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("field %q: expected object: %w", field, err)
	}
	return result, nil
}

func parsePlacement(center []float64, scaleRaw json.RawMessage, dimension int) ([3]float64, [3]float64, error) {
	var scale []float64
	if len(scaleRaw) > 0 {
		if err := json.Unmarshal(scaleRaw, &scale); err != nil {
			var scalar float64
			if scalarErr := json.Unmarshal(scaleRaw, &scalar); scalarErr != nil {
				return [3]float64{}, [3]float64{}, fmt.Errorf("field %q: expected number or array", "scale")
			}
			scale = make([]float64, dimension)
			for i := range scale {
				scale[i] = scalar
			}
		}
	}
	return normalizePlacement(center, scale, dimension)
}

func normalizePlacement(center, scale []float64, dimension int) ([3]float64, [3]float64, error) {
	normalizedCenter := [3]float64{}
	normalizedScale := [3]float64{1, 1, 1}

	if center != nil {
		if err := utils.RequireSliceLength("center", center, dimension); err != nil {
			return normalizedCenter, normalizedScale, err
		}
		copy(normalizedCenter[:], center)
	}
	if scale != nil {
		if err := utils.RequireSliceLength("scale", scale, dimension); err != nil {
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
