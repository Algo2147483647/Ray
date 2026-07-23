package adapt

import (
	"fmt"
	"math"
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
	if err := adaptBounds(adapted, ctx, dimension); err != nil {
		return nil, err
	}

	shapeName, _ := stringField(adapted, "shape")
	switch {
	case strings.EqualFold(shapeName, "cuboid"),
		strings.EqualFold(shapeName, "hypercuboid"),
		strings.EqualFold(shapeName, "hypercube"):
		return adaptCuboid(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "triangle"):
		return adaptTriangle(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "sphere"),
		strings.EqualFold(shapeName, "hypersphere"):
		return adaptSphere(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "circle"):
		return adaptCircle(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "cylinder"),
		strings.EqualFold(shapeName, "finite cylinder"):
		return adaptFiniteCylinder(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "quadratic equation"):
		return adaptQuadraticEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "cubic equation"):
		return adaptCubicEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "four-order equation"):
		return adaptFourOrderEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "implicit equation"):
		return adaptImplicitEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "parametric equation"):
		return adaptParametricEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "polynomial surface"):
		return adaptPolynomialSurface(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "stl"):
		return adaptSTL(adapted, ctx, dimension)
	}
	return adapted, nil
}

func adaptBounds(object map[string]interface{}, ctx groupContext, dimension int) error {
	rawBounds, ok := object["bounds"]
	if !ok {
		return nil
	}
	bounds, ok := rawBounds.(map[string]interface{})
	if !ok {
		return fmt.Errorf("field %q: expected object, got %T", "bounds", rawBounds)
	}

	if _, hasPmin := bounds["pmin"]; hasPmin {
		if _, hasPmax := bounds["pmax"]; !hasPmax {
			return fmt.Errorf(`bounds missing required field "pmax"`)
		}
		pmin, err := vectorValue("bounds.pmin", bounds["pmin"], dimension)
		if err != nil {
			return err
		}
		pmax, err := vectorValue("bounds.pmax", bounds["pmax"], dimension)
		if err != nil {
			return err
		}
		if err := validateBoundsMinMax(pmin, pmax); err != nil {
			return err
		}
		pmin, pmax = placedMinMax(ctx, pmin, pmax)
		bounds["pmin"] = pmin
		bounds["pmax"] = pmax
		return nil
	}

	center, err := boundsCenter(bounds, dimension)
	if err != nil {
		return fmt.Errorf("bounds requires either center+size or pmin+pmax: %w", err)
	}
	size, err := vectorField(bounds, "size", dimension)
	if err != nil {
		return err
	}

	pmin := make([]float64, dimension)
	pmax := make([]float64, dimension)
	for i := 0; i < dimension; i++ {
		if size[i] <= 0 {
			return fmt.Errorf("bounds size index %d must be > 0", i)
		}
		half := size[i] * 0.5
		pmin[i] = center[i] - half
		pmax[i] = center[i] + half
	}
	pmin, pmax = placedMinMax(ctx, pmin, pmax)
	object["bounds"] = map[string]interface{}{
		"pmin": pmin,
		"pmax": pmax,
	}
	return nil
}

func placedMinMax(ctx groupContext, pmin, pmax []float64) ([]float64, []float64) {
	worldPmin := applyPlacement(ctx, pmin)
	worldPmax := applyPlacement(ctx, pmax)
	for i := range worldPmin {
		if worldPmin[i] > worldPmax[i] {
			worldPmin[i], worldPmax[i] = worldPmax[i], worldPmin[i]
		}
	}
	return worldPmin, worldPmax
}

func boundsCenter(bounds map[string]interface{}, dimension int) ([]float64, error) {
	if center, ok := bounds["center"]; ok {
		return vectorValue("bounds.center", center, dimension)
	}
	if position, ok := bounds["position"]; ok {
		return vectorValue("bounds.position", position, dimension)
	}
	return nil, fmt.Errorf(`missing required field "center"`)
}

func validateBoundsMinMax(pmin, pmax []float64) error {
	for i := range pmin {
		if pmin[i] >= pmax[i] {
			return fmt.Errorf("bounds pmin index %d must be < pmax", i)
		}
	}
	return nil
}

func adaptCuboid(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	shapeName, _ := stringField(object, "shape")
	isHypercube := strings.EqualFold(shapeName, "hypercube")

	if _, hasPmin := object["pmin"]; hasPmin {
		if _, hasPmax := object["pmax"]; hasPmax && !isHypercube && groupPlacementIsIdentity(ctx) {
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
	if isHypercube {
		if err := validateHypercubeExtents(pmin, pmax); err != nil {
			return nil, err
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
	if isHypercube {
		adapted["shape"] = "cuboid"
	}
	adapted["pmin"] = worldPmin
	adapted["pmax"] = worldPmax
	delete(adapted, "center")
	delete(adapted, "position")
	delete(adapted, "size")
	return adapted, nil
}

func validateHypercubeExtents(pmin, pmax []float64) error {
	side := pmax[0] - pmin[0]
	if side <= 0 {
		return fmt.Errorf("hypercube side length must be > 0")
	}
	for axis := 1; axis < len(pmin); axis++ {
		diff := pmax[axis] - pmin[axis]
		if diff <= 0 {
			return fmt.Errorf("hypercube side length axis %d must be > 0", axis)
		}
		if !nearlyEqual(diff, side) {
			return fmt.Errorf("hypercube requires equal side lengths, axis %d has %g instead of %g", axis, diff, side)
		}
	}
	return nil
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
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

func adaptSphere(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	center, err := optionalObjectCenter(object, dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}
	radius, err := floatField(object, "r")
	if err != nil {
		return nil, err
	}

	adapted := cloneMap(object)
	worldCenter := applyPlacement(ctx, center)
	if scale, ok := uniformPlacementScale(ctx); ok || dimension != 3 {
		if !ok {
			return nil, fmt.Errorf("hypersphere does not support non-uniform group scale")
		}
		adapted["center"] = worldCenter
		adapted["r"] = radius * scale
		delete(adapted, "position")
		return adapted, nil
	}

	a, b, c := ellipsoidQuadratic(worldCenter, ctx.scale, radius)
	adapted["shape"] = "quadratic equation"
	adapted["a"] = a
	adapted["b"] = b
	adapted["c"] = c
	delete(adapted, "center")
	delete(adapted, "position")
	delete(adapted, "r")
	delete(adapted, "scale")
	delete(adapted, "basis")
	return adapted, nil
}

func adaptCircle(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	center, err := optionalObjectCenter(object, dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}
	radius, err := floatField(object, "r")
	if err != nil {
		return nil, err
	}
	if _, err := vectorField(object, "normal", dimension); err != nil {
		return nil, err
	}
	scale, ok := uniformPlacementScale(ctx)
	if !ok {
		return nil, fmt.Errorf("circle does not support non-uniform group scale")
	}

	adapted := cloneMap(object)
	adapted["center"] = applyPlacement(ctx, center)
	adapted["r"] = radius * scale
	delete(adapted, "position")
	return adapted, nil
}

func adaptFiniteCylinder(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	center, err := optionalObjectCenter(object, dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}
	radius, err := floatField(object, "r")
	if err != nil {
		return nil, err
	}
	height, err := floatField(object, "height")
	if err != nil {
		return nil, err
	}
	if _, err := vectorField(object, "axis", dimension); err != nil {
		return nil, err
	}
	scale, ok := uniformPlacementScale(ctx)
	if !ok {
		return nil, fmt.Errorf("finite cylinder does not support non-uniform group scale")
	}

	adapted := cloneMap(object)
	adapted["center"] = applyPlacement(ctx, center)
	adapted["r"] = radius * scale
	adapted["height"] = height * scale
	delete(adapted, "position")
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

func adaptFourOrderEquation(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("four-order equation adapter requires dimension 3, got %d", dimension)
	}

	localCenter, err := optionalVector(object, "center", dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}
	localScale, err := optionalScale(object, "scale", dimension, unitVector(dimension))
	if err != nil {
		return nil, err
	}
	basis, err := optionalBasis(object, dimension)
	if err != nil {
		return nil, err
	}

	coefficients, err := requiredPolynomialCoefficients(object, 4)
	if err != nil {
		return nil, err
	}

	adapted := cloneMap(object)
	adapted["a"] = bakeFourOrderCoefficients(coefficients, ctx, localCenter, localScale, basis)
	delete(adapted, "A")
	delete(adapted, "center")
	delete(adapted, "scale")
	delete(adapted, "basis")
	return adapted, nil
}

func adaptImplicitEquation(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("implicit equation adapter requires dimension 3, got %d", dimension)
	}

	adapted := cloneMap(object)
	transform, hasTransform, err := optionalTransform(adapted)
	if err != nil {
		return nil, err
	}

	if hasTransform {
		adapted["transform"] = transformToSlices(composeWithGroupInverse(transform, ctx))
	} else {
		localCenter, err := optionalVector(object, "center", dimension, zeroVector(dimension))
		if err != nil {
			return nil, err
		}
		localScale, err := optionalScale(object, "scale", dimension, unitVector(dimension))
		if err != nil {
			return nil, err
		}
		basis, err := optionalBasis(object, dimension)
		if err != nil {
			return nil, err
		}
		adapted["transform"] = transformToSlices(worldToLocalTransformMatrix(ctx, localCenter, localScale, basis))
	}

	delete(adapted, "center")
	delete(adapted, "scale")
	delete(adapted, "basis")
	return adapted, nil
}

func adaptParametricEquation(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("parametric equation adapter requires dimension 3, got %d", dimension)
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

	adapted := cloneMap(object)
	adapted["center"] = worldCenter
	adapted["scale"] = worldScale
	return adapted, nil
}

func adaptPolynomialSurface(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("polynomial surface adapter requires dimension 3, got %d", dimension)
	}

	adapted := cloneMap(object)
	transform, hasTransform, err := optionalTransform(adapted)
	if err != nil {
		return nil, err
	}

	if hasTransform {
		adapted["transform"] = transformToSlices(composeWithGroupInverse(transform, ctx))
	} else {
		localCenter, err := optionalVector(object, "center", dimension, zeroVector(dimension))
		if err != nil {
			return nil, err
		}
		localScale, err := optionalScale(object, "scale", dimension, unitVector(dimension))
		if err != nil {
			return nil, err
		}
		basis, err := optionalBasis(object, dimension)
		if err != nil {
			return nil, err
		}
		adapted["transform"] = transformToSlices(worldToLocalTransformMatrix(ctx, localCenter, localScale, basis))
	}

	delete(adapted, "center")
	delete(adapted, "scale")
	delete(adapted, "basis")
	return adapted, nil
}

func adaptSTL(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("stl adapter requires dimension 3, got %d", dimension)
	}
	center, err := objectCenter(object, dimension)
	if err != nil {
		return nil, err
	}
	localScale, err := vectorField(object, "scale", dimension)
	if err != nil {
		return nil, err
	}
	groupScale, ok := uniformPlacementScale(ctx)
	if !ok {
		return nil, fmt.Errorf("stl does not support non-uniform group scale")
	}

	worldScale := make([]float64, dimension)
	for i := range worldScale {
		worldScale[i] = groupScale * localScale[i]
	}

	adapted := cloneMap(object)
	adapted["center"] = applyPlacement(ctx, center)
	adapted["scale"] = worldScale
	delete(adapted, "position")
	return adapted, nil
}

func uniformPlacementScale(ctx groupContext) (float64, bool) {
	scale := ctx.scale[0]
	for i := 1; i < ctx.dimension; i++ {
		if !nearlyEqual(ctx.scale[i], scale) {
			return 0, false
		}
	}
	return scale, true
}

func ellipsoidQuadratic(center, scale []float64, radius float64) ([]float64, []float64, float64) {
	a := make([]float64, 9)
	b := make([]float64, 3)
	c := -1.0
	for axis := 0; axis < 3; axis++ {
		axisScale := scale[axis] * radius
		coefficient := 1 / (axisScale * axisScale)
		a[axis*3+axis] = coefficient
		b[axis] = -2 * center[axis] * coefficient
		c += center[axis] * center[axis] * coefficient
	}
	return a, b, c
}
