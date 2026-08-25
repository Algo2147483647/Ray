package factory

import (
	"encoding/json"
	"fmt"
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
	return normalizePolynomialCenterScale(center, scale, dimension)
}
