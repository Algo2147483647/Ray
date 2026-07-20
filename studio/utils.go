package main

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func cloneNestedStringMap(value map[string]map[string]interface{}) map[string]map[string]interface{} {
	if len(value) == 0 {
		return nil
	}
	result := make(map[string]map[string]interface{}, len(value))
	for key, item := range value {
		result[key] = cloneMap(item)
	}
	return result
}

func cloneMapSlice(items []map[string]interface{}) []map[string]interface{} {
	if len(items) == 0 {
		return nil
	}
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		result[i] = cloneMap(item)
	}
	return result
}

func cloneMap(value map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(value))
	for key, item := range value {
		result[key] = deepClone(item)
	}
	return result
}

func deepClone(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return cloneMap(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = deepClone(item)
		}
		return result
	case []float64:
		return append([]float64(nil), v...)
	case []string:
		return append([]string(nil), v...)
	default:
		return v
	}
}

func stringField(object map[string]interface{}, key string) (string, bool) {
	raw, ok := object[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	return value, ok && value != ""
}

func objectID(object map[string]interface{}, index int) string {
	if id, ok := stringField(object, "id"); ok {
		return id
	}
	return "#" + strconv.Itoa(index)
}

func objectLabel(object map[string]interface{}, index int) string {
	if id, ok := stringField(object, "id"); ok {
		return strconv.Quote(id)
	}
	return "#" + strconv.Itoa(index)
}

func joinID(prefix, id string) string {
	if prefix == "" {
		return id
	}
	if id == "" {
		return prefix
	}
	return prefix + "/" + id
}

func zeroVector(dimension int) []float64 {
	return make([]float64, dimension)
}

func unitVector(dimension int) []float64 {
	values := make([]float64, dimension)
	for i := range values {
		values[i] = 1
	}
	return values
}

func toArray3(values []float64) [3]float64 {
	return [3]float64{values[0], values[1], values[2]}
}

func toFloat64Slice(value interface{}) ([]float64, error) {
	switch v := value.(type) {
	case []float64:
		return append([]float64(nil), v...), nil
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
		return nil, fmt.Errorf("expected array, got %T", value)
	}
}

func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
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
