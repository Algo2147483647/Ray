package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

func OptionalMapField(data map[string]interface{}, key string) (map[string]interface{}, bool, error) {
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

func ValidateNonNegativeSlice(name string, values []float64) error {
	for i, value := range values {
		if value < 0 {
			return fmt.Errorf("%s index %d must be >= 0", name, i)
		}
	}
	return nil
}

func ValidateStrictlyIncreasing(name string, values []float64) error {
	for i := 1; i < len(values); i++ {
		if values[i] <= values[i-1] {
			return fmt.Errorf("%s must be strictly increasing", name)
		}
	}
	return nil
}

func RequiredStringField(data map[string]interface{}, key string) (string, error) {
	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("missing required field %q", key)
	}

	text, err := toString(value)
	if err != nil {
		return "", fmt.Errorf("field %q: %w", key, err)
	}
	if text == "" {
		return "", fmt.Errorf("field %q must not be empty", key)
	}
	return text, nil
}

func OptionalStringField(data map[string]interface{}, key string) (string, bool, error) {
	value, ok := data[key]
	if !ok {
		return "", false, nil
	}

	text, err := toString(value)
	if err != nil {
		return "", true, fmt.Errorf("field %q: %w", key, err)
	}
	return text, true, nil
}

func OptionalBoolField(data map[string]interface{}, key string) (bool, bool, error) {
	value, ok := data[key]
	if !ok {
		return false, false, nil
	}

	boolean, err := toBool(value)
	if err != nil {
		return false, true, fmt.Errorf("field %q: %w", key, err)
	}
	return boolean, true, nil
}

func OptionalFloat64Field(data map[string]interface{}, key string) (float64, bool, error) {
	value, ok := data[key]
	if !ok {
		return 0, false, nil
	}

	number, err := toFloat64(value)
	if err != nil {
		return 0, true, fmt.Errorf("field %q: %w", key, err)
	}
	return number, true, nil
}

func RequiredFloat64Field(data map[string]interface{}, key string) (float64, error) {
	value, ok := data[key]
	if !ok {
		return 0, fmt.Errorf("missing required field %q", key)
	}

	number, err := toFloat64(value)
	if err != nil {
		return 0, fmt.Errorf("field %q: %w", key, err)
	}
	return number, nil
}

func RequiredFloat64SliceField(data map[string]interface{}, key string, expectedLengths ...int) ([]float64, error) {
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}

	values, err := ToFloat64Slice(value)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	if err := RequireSliceLength(key, values, expectedLengths...); err != nil {
		return nil, err
	}
	return values, nil
}

func OptionalFloat64SliceField(data map[string]interface{}, key string, expectedLengths ...int) ([]float64, bool, error) {
	value, ok := data[key]
	if !ok {
		return nil, false, nil
	}

	values, err := ToFloat64Slice(value)
	if err != nil {
		return nil, true, fmt.Errorf("field %q: %w", key, err)
	}
	if err := RequireSliceLength(key, values, expectedLengths...); err != nil {
		return nil, true, err
	}
	return values, true, nil
}

func RequireSliceLength(key string, values []float64, expectedLengths ...int) error {
	if len(expectedLengths) == 0 {
		return nil
	}

	for _, expected := range expectedLengths {
		if len(values) == expected {
			return nil
		}
	}

	return fmt.Errorf("field %q must contain %s values, got %d", key, formatAllowedLengths(expectedLengths), len(values))
}

func formatAllowedLengths(lengths []int) string {
	if len(lengths) == 1 {
		return strconv.Itoa(lengths[0])
	}

	result := ""
	for i, length := range lengths {
		if i > 0 {
			if i == len(lengths)-1 {
				result += " or "
			} else {
				result += ", "
			}
		}
		result += strconv.Itoa(length)
	}
	return result
}

func toString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("expected string, got %T", value)
	}
}

func toBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case float64:
		if v == 0 {
			return false, nil
		}
		if v == 1 {
			return true, nil
		}
	case int:
		if v == 0 {
			return false, nil
		}
		if v == 1 {
			return true, nil
		}
	case json.Number:
		number, err := v.Float64()
		if err != nil {
			return false, fmt.Errorf("invalid boolean number %q", v.String())
		}
		if number == 0 {
			return false, nil
		}
		if number == 1 {
			return true, nil
		}
	case string:
		if v == "true" {
			return true, nil
		}
		if v == "false" {
			return false, nil
		}
	}

	return false, fmt.Errorf("expected boolean-compatible value, got %v (%T)", value, value)
}

func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("must be a finite number")
		}
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		number, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf("invalid number %q", v.String())
		}
		return number, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", value)
	}
}

func ToFloat64Slice(value interface{}) ([]float64, error) {
	switch v := value.(type) {
	case []float64:
		result := make([]float64, len(v))
		copy(result, v)
		return result, nil
	case []interface{}:
		result := make([]float64, len(v))
		for i, item := range v {
			number, err := toFloat64(item)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			result[i] = number
		}
		return result, nil
	default:
		rv := reflect.ValueOf(value)
		if rv.IsValid() && rv.Kind() == reflect.Slice {
			result := make([]float64, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				number, err := toFloat64(rv.Index(i).Interface())
				if err != nil {
					return nil, fmt.Errorf("index %d: %w", i, err)
				}
				result[i] = number
			}
			return result, nil
		}
		return nil, fmt.Errorf("expected array, got %T", value)
	}
}
