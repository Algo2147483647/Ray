package adapt

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

func flattenArrayObject(object map[string]interface{}, ctx groupContext, index, dimension int) ([]map[string]interface{}, error) {
	arrayObjects, err := requiredArrayObjectMap(object)
	if err != nil {
		return nil, fmt.Errorf("array %s: %w", objectLabel(object, index), err)
	}
	childContexts, err := deriveArrayCellContexts(ctx, object, index, dimension)
	if err != nil {
		return nil, fmt.Errorf("array %s: %w", objectLabel(object, index), err)
	}

	flattened := []map[string]interface{}{}
	for _, cellKey := range sortedArrayCellKeys(arrayObjects) {
		childContext, ok := childContexts[cellKey]
		if !ok {
			return nil, fmt.Errorf("array %s: cell %q is outside counts", objectLabel(object, index), cellKey)
		}
		children, err := flattenObjects(arrayObjects[cellKey], childContext, dimension)
		if err != nil {
			return nil, err
		}
		flattened = append(flattened, children...)
	}
	return flattened, nil
}

func deriveArrayCellContexts(parent groupContext, object map[string]interface{}, index, dimension int) (map[string]groupContext, error) {
	origin, err := vectorField(object, "origin", dimension)
	if err != nil {
		return nil, err
	}
	localScale, err := optionalScale(object, "scale", dimension, unitVector(dimension))
	if err != nil {
		return nil, err
	}
	counts, err := requiredPositiveIntSlice(object, "counts", 3)
	if err != nil {
		return nil, err
	}
	delta, err := requiredArrayDelta(object, "delta", len(counts), dimension)
	if err != nil {
		return nil, err
	}

	base := groupContext{
		dimension: dimension,
		idPrefix:  joinID(parent.idPrefix, objectID(object, index)),
		center:    make([]float64, dimension),
		scale:     make([]float64, dimension),
		fields:    cloneMap(parent.fields),
	}
	for axis := 0; axis < dimension; axis++ {
		base.center[axis] = parent.center[axis] + parent.scale[axis]*origin[axis]
		base.scale[axis] = parent.scale[axis] * localScale[axis]
	}
	inheritGroupFields(base.fields, object)

	result := map[string]groupContext{}
	indices := make([]int, len(counts))
	var visit func(axis int)
	visit = func(axis int) {
		if axis == len(counts) {
			cellKey := arrayCellKey(indices)
			cell := groupContext{
				dimension: dimension,
				idPrefix:  joinID(base.idPrefix, arrayCellID(indices)),
				center:    append([]float64(nil), base.center...),
				scale:     append([]float64(nil), base.scale...),
				fields:    cloneMap(base.fields),
			}
			for dimAxis, indexValue := range indices {
				offsetScale := float64(indexValue - 1)
				for worldAxis := 0; worldAxis < dimension; worldAxis++ {
					cell.center[worldAxis] += base.scale[worldAxis] * delta[dimAxis][worldAxis] * offsetScale
				}
			}
			result[cellKey] = cell
			return
		}
		for value := 1; value <= counts[axis]; value++ {
			indices[axis] = value
			visit(axis + 1)
		}
	}
	visit(0)
	return result, nil
}

func requiredArrayObjectMap(object map[string]interface{}) (map[string][]map[string]interface{}, error) {
	raw, ok := object["objects"]
	if !ok {
		return nil, fmt.Errorf(`missing required field "objects"`)
	}
	items, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected object, got %T", "objects", raw)
	}
	counts, err := requiredPositiveIntSlice(object, "counts", 3)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]map[string]interface{}, len(items))
	for key, rawList := range items {
		index, err := parseArrayCellKey(key)
		if err != nil {
			return nil, fmt.Errorf("objects cell %q: %w", key, err)
		}
		if len(index) != len(counts) {
			return nil, fmt.Errorf("objects cell %q dimension must be %d, got %d", key, len(counts), len(index))
		}
		for axis, value := range index {
			if value > counts[axis] {
				return nil, fmt.Errorf("objects cell %q index %d exceeds counts[%d]=%d", key, value, axis, counts[axis])
			}
		}
		objectList, err := objectListValue(rawList, fmt.Sprintf("objects.%s", key))
		if err != nil {
			return nil, err
		}
		result[arrayCellKey(index)] = objectList
	}
	return result, nil
}

func objectListValue(raw interface{}, key string) ([]map[string]interface{}, error) {
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array, got %T", key, raw)
	}
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		mapped, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %q index %d: expected object, got %T", key, i, item)
		}
		result[i] = mapped
	}
	return result, nil
}

func requiredPositiveIntSlice(object map[string]interface{}, key string, maxLen int) ([]int, error) {
	raw, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}
	values, err := toFloat64Slice(raw)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	if len(values) == 0 || len(values) > maxLen {
		return nil, fmt.Errorf("field %q must contain 1 to %d values, got %d", key, maxLen, len(values))
	}
	result := make([]int, len(values))
	for i, value := range values {
		if value <= 0 || math.Trunc(value) != value {
			return nil, fmt.Errorf("%s index %d must be a positive integer", key, i)
		}
		result[i] = int(value)
	}
	return result, nil
}

func requiredArrayDelta(object map[string]interface{}, key string, count, dimension int) ([][]float64, error) {
	raw, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}
	rows, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array, got %T", key, raw)
	}
	if len(rows) != count {
		return nil, fmt.Errorf("field %q must contain %d vectors, got %d", key, count, len(rows))
	}
	result := make([][]float64, len(rows))
	for i, row := range rows {
		values, err := vectorValue(fmt.Sprintf("%s[%d]", key, i), row, dimension)
		if err != nil {
			return nil, err
		}
		result[i] = values
	}
	return result, nil
}

func parseArrayCellKey(key string) ([]int, error) {
	parts := strings.Split(key, ",")
	if len(parts) == 0 || len(parts) > 3 {
		return nil, fmt.Errorf("must contain 1 to 3 comma-separated indices")
	}
	result := make([]int, len(parts))
	for i, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("index %d must be a positive integer", i)
		}
		result[i] = value
	}
	return result, nil
}

func sortedArrayCellKeys(items map[string][]map[string]interface{}) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, _ := parseArrayCellKey(keys[i])
		right, _ := parseArrayCellKey(keys[j])
		for axis := 0; axis < len(left) && axis < len(right); axis++ {
			if left[axis] != right[axis] {
				return left[axis] < right[axis]
			}
		}
		return len(left) < len(right)
	})
	return keys
}

func arrayCellKey(index []int) string {
	parts := make([]string, len(index))
	for i, value := range index {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

func arrayCellID(index []int) string {
	labels := []string{"i", "j", "k"}
	parts := make([]string, len(index))
	for i, value := range index {
		parts[i] = labels[i] + strconv.Itoa(value)
	}
	return strings.Join(parts, "-")
}
