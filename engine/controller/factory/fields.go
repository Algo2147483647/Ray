package factory

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/controller/parser"
)

var (
	requiredStringField       = parser.RequiredStringField
	optionalStringField       = parser.OptionalStringField
	optionalBoolField         = parser.OptionalBoolField
	optionalFloat64Field      = parser.OptionalFloat64Field
	requiredFloat64Field      = parser.RequiredFloat64Field
	requiredFloat64SliceField = parser.RequiredFloat64SliceField
	optionalFloat64SliceField = parser.OptionalFloat64SliceField
	requireSliceLength        = parser.RequireSliceLength
	toFloat64Slice            = parser.ToFloat64Slice
)

func optionalMapField(data map[string]interface{}, key string) (map[string]interface{}, bool, error) {
	value, ok := data[key]
	if !ok {
		return nil, false, nil
	}
	mapped, ok := value.(map[string]interface{})
	if !ok {
		return nil, true, fmt.Errorf("field %q: expected object, got %T", key, value)
	}
	return mapped, true, nil
}

func validateNonNegativeSlice(name string, values []float64) error {
	for i, value := range values {
		if value < 0 {
			return fmt.Errorf("%s index %d must be >= 0", name, i)
		}
	}
	return nil
}

func validateStrictlyIncreasing(name string, values []float64) error {
	for i := 1; i < len(values); i++ {
		if values[i] <= values[i-1] {
			return fmt.Errorf("%s must be strictly increasing", name)
		}
	}
	return nil
}
