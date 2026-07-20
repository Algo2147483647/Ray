package main

import (
	"fmt"
	"strings"
)

func adaptObject(object map[string]interface{}, ctx groupContext, index, dimension int) (map[string]interface{}, error) {
	adapted := cloneMap(object)
	delete(adapted, "objects")

	if id, ok := stringField(adapted, "id"); ok {
		adapted["id"] = joinID(ctx.idPrefix, id)
	} else if ctx.idPrefix != "" {
		adapted["id"] = joinID(ctx.idPrefix, objectID(object, index))
	}
	applyInheritedFields(adapted, ctx.fields)

	shapeName, _ := stringField(adapted, "shape")
	switch {
	case strings.EqualFold(shapeName, "cuboid"),
		strings.EqualFold(shapeName, "hypercuboid"),
		strings.EqualFold(shapeName, "hypercube"):
		return adaptCuboid(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "triangle"):
		return adaptTriangle(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "quadratic equation"):
		return adaptQuadraticEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "cubic equation"):
		return adaptCubicEquation(adapted, ctx, dimension)
	}
	return adapted, nil
}

func adaptCuboid(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if _, hasPmin := object["pmin"]; hasPmin {
		if _, hasPmax := object["pmax"]; hasPmax && groupPlacementIsIdentity(ctx) {
			return object, nil
		}
	}

	var pmin []float64
	var pmax []float64
	if rawPmin, hasPmin := object["pmin"]; hasPmin {
		rawPmax, hasPmax := object["pmax"]
		if !hasPmax {
			return nil, fmt.Errorf(`missing required field "pmax"`)
		}
		var err error
		pmin, err = vectorValue("pmin", rawPmin, dimension)
		if err != nil {
			return nil, err
		}
		pmax, err = vectorValue("pmax", rawPmax, dimension)
		if err != nil {
			return nil, err
		}
	} else {
		center, err := objectCenter(object, dimension)
		if err != nil {
			return nil, err
		}
		size, err := vectorField(object, "size", dimension)
		if err != nil {
			return nil, err
		}
		pmin = make([]float64, dimension)
		pmax = make([]float64, dimension)
		for i := 0; i < dimension; i++ {
			if size[i] <= 0 {
				return nil, fmt.Errorf("size index %d must be > 0", i)
			}
			half := size[i] * 0.5
			pmin[i] = center[i] - half
			pmax[i] = center[i] + half
		}
	}

	worldPmin := applyPlacement(ctx, pmin)
	worldPmax := applyPlacement(ctx, pmax)
	for i := 0; i < dimension; i++ {
		if worldPmin[i] > worldPmax[i] {
			worldPmin[i], worldPmax[i] = worldPmax[i], worldPmin[i]
		}
	}

	adapted := cloneMap(object)
	adapted["pmin"] = worldPmin
	adapted["pmax"] = worldPmax
	delete(adapted, "center")
	delete(adapted, "position")
	delete(adapted, "size")
	return adapted, nil
}

func adaptTriangle(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	p1, err := vectorField(object, "p1", dimension)
	if err != nil {
		return nil, err
	}
	p2, err := vectorField(object, "p2", dimension)
	if err != nil {
		return nil, err
	}
	p3, err := vectorField(object, "p3", dimension)
	if err != nil {
		return nil, err
	}
	center, err := optionalVector(object, "center", dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}

	adapted := cloneMap(object)
	adapted["p1"] = applyPlacement(ctx, addVectors(p1, center))
	adapted["p2"] = applyPlacement(ctx, addVectors(p2, center))
	adapted["p3"] = applyPlacement(ctx, addVectors(p3, center))
	delete(adapted, "center")
	return adapted, nil
}

func adaptQuadraticEquation(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("quadratic equation adapter requires dimension 3, got %d", dimension)
	}

	localCenter, err := optionalVector(object, "center", dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}
	localScale, err := optionalScale(object, "scale", dimension, unitVector(dimension))
	if err != nil {
		return nil, err
	}

	worldCenter := make([]float64, dimension)
	worldScale := make([]float64, dimension)
	for i := 0; i < dimension; i++ {
		worldCenter[i] = ctx.center[i] + ctx.scale[i]*localCenter[i]
		worldScale[i] = ctx.scale[i] * localScale[i]
	}

	a, err := vectorField(object, "a", 9)
	if err != nil {
		return nil, err
	}
	b, err := vectorField(object, "b", dimension)
	if err != nil {
		return nil, err
	}
	c, err := floatField(object, "c")
	if err != nil {
		return nil, err
	}
	worldA, worldB, worldC := bakeQuadraticCoefficients(a, b, c, toArray3(worldCenter), toArray3(worldScale))

	adapted := cloneMap(object)
	adapted["a"] = worldA
	adapted["b"] = worldB
	adapted["c"] = worldC
	delete(adapted, "center")
	delete(adapted, "scale")
	return adapted, nil
}

func adaptCubicEquation(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("cubic equation adapter requires dimension 3, got %d", dimension)
	}

	localCenter, err := optionalVector(object, "center", dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}
	localScale, err := optionalScale(object, "scale", dimension, unitVector(dimension))
	if err != nil {
		return nil, err
	}

	worldCenter := make([]float64, dimension)
	worldScale := make([]float64, dimension)
	for i := 0; i < dimension; i++ {
		worldCenter[i] = ctx.center[i] + ctx.scale[i]*localCenter[i]
		worldScale[i] = ctx.scale[i] * localScale[i]
	}

	coefficients, err := requiredPolynomialCoefficients(object, 3)
	if err != nil {
		return nil, err
	}

	adapted := cloneMap(object)
	adapted["a"] = bakeCubicCoefficients(coefficients, toArray3(worldCenter), toArray3(worldScale))
	delete(adapted, "A")
	delete(adapted, "center")
	delete(adapted, "scale")
	return adapted, nil
}
