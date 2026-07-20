package main

import (
	"fmt"
	"strings"
)

type groupContext struct {
	dimension int
	idPrefix  string
	center    []float64
	scale     []float64
	fields    map[string]interface{}
}

func newRootContext(dimension int) groupContext {
	scale := make([]float64, dimension)
	for i := range scale {
		scale[i] = 1
	}
	return groupContext{
		dimension: dimension,
		center:    make([]float64, dimension),
		scale:     scale,
		fields:    map[string]interface{}{},
	}
}

func flattenObjects(objects []map[string]interface{}, ctx groupContext, dimension int) ([]map[string]interface{}, error) {
	flattened := make([]map[string]interface{}, 0, len(objects))
	for index, object := range objects {
		shapeName, _ := stringField(object, "shape")
		if strings.EqualFold(shapeName, "group") {
			groupObjects, err := requiredObjectList(object, "objects")
			if err != nil {
				return nil, fmt.Errorf("group %s: %w", objectLabel(object, index), err)
			}
			childContext, err := deriveGroupContext(ctx, object, index, dimension)
			if err != nil {
				return nil, fmt.Errorf("group %s: %w", objectLabel(object, index), err)
			}
			children, err := flattenObjects(groupObjects, childContext, dimension)
			if err != nil {
				return nil, err
			}
			flattened = append(flattened, children...)
			continue
		}

		adapted, err := adaptObject(object, ctx, index, dimension)
		if err != nil {
			return nil, fmt.Errorf("object %s: %w", objectLabel(object, index), err)
		}
		flattened = append(flattened, adapted)
	}
	return flattened, nil
}

func deriveGroupContext(parent groupContext, object map[string]interface{}, index, dimension int) (groupContext, error) {
	localCenter, err := optionalVector(object, "center", dimension, zeroVector(dimension))
	if err != nil {
		return groupContext{}, err
	}
	localScale, err := optionalScale(object, "scale", dimension, unitVector(dimension))
	if err != nil {
		return groupContext{}, err
	}

	ctx := groupContext{
		dimension: dimension,
		idPrefix:  joinID(parent.idPrefix, objectID(object, index)),
		center:    make([]float64, dimension),
		scale:     make([]float64, dimension),
		fields:    cloneMap(parent.fields),
	}
	for i := 0; i < dimension; i++ {
		ctx.center[i] = parent.center[i] + parent.scale[i]*localCenter[i]
		ctx.scale[i] = parent.scale[i] * localScale[i]
	}
	inheritGroupFields(ctx.fields, object)
	return ctx, nil
}

func inheritGroupFields(fields map[string]interface{}, object map[string]interface{}) {
	for _, key := range []string{"material_id", "medium_id", "emission_id", "bounds"} {
		if value, ok := object[key]; ok {
			fields[key] = deepClone(value)
		}
	}
}

func applyInheritedFields(object, fields map[string]interface{}) {
	for key, value := range fields {
		if _, ok := object[key]; !ok {
			object[key] = deepClone(value)
		}
	}
}

func requiredObjectList(object map[string]interface{}, key string) ([]map[string]interface{}, error) {
	raw, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}
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
