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
	if !basisIsIdentity(ctx.basis) && !rotationAwareShape(shapeName) {
		return nil, fmt.Errorf("shape %q does not support rotated group placement", shapeName)
	}
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
	case strings.EqualFold(shapeName, "plane"):
		return adaptPlane(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "quadratic"),
		strings.EqualFold(shapeName, "quadric"),
		strings.EqualFold(shapeName, "quadratic surface"):
		return adaptQuadraticSurface(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "quadratic equation"),
		strings.EqualFold(shapeName, "cubic equation"),
		strings.EqualFold(shapeName, "four-order equation"),
		strings.EqualFold(shapeName, "polynomial surface"):
		return nil, fmt.Errorf("legacy polynomial shape %q is not supported; use shape %q with degree, terms, and transform", shapeName, "polynomial")
	case strings.EqualFold(shapeName, "implicit equation"):
		return adaptImplicitEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "parametric equation"):
		return adaptParametricEquation(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "parametric curve"):
		return adaptParametricCurve(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "polynomial"):
		return adaptPolynomial(adapted, ctx, dimension)
	case strings.EqualFold(shapeName, "stl"):
		return adaptSTL(adapted, ctx, dimension)
	}
	return adapted, nil
}

func rotationAwareShape(shapeName string) bool {
	for _, supported := range []string{"triangle", "sphere", "hypersphere", "circle", "cylinder", "finite cylinder", "cuboid", "hypercuboid", "hypercube", "plane", "quadratic", "quadric", "quadratic surface"} {
		if strings.EqualFold(shapeName, supported) {
			return true
		}
	}
	return false
}

func adaptPlane(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("plane adapter requires dimension 3, got %d", dimension)
	}
	normal, err := vectorField(object, "normal", dimension)
	if err != nil {
		return nil, err
	}
	lengthSquared := 0.0
	for _, value := range normal {
		lengthSquared += value * value
	}
	if lengthSquared <= 0 || math.IsNaN(lengthSquared) || math.IsInf(lengthSquared, 0) {
		return nil, fmt.Errorf("field %q must be a finite non-zero vector", "normal")
	}

	_, hasPoint := object["point"]
	_, hasOffset := object["offset"]
	if hasPoint && hasOffset {
		return nil, fmt.Errorf("plane fields %q and %q cannot both be provided", "point", "offset")
	}
	constant := 0.0
	if hasPoint {
		point, err := vectorField(object, "point", dimension)
		if err != nil {
			return nil, err
		}
		for axis := range normal {
			constant -= normal[axis] * point[axis]
		}
	} else if hasOffset {
		offset, err := floatField(object, "offset")
		if err != nil {
			return nil, err
		}
		constant = -offset
	}

	terms := make([]map[string]interface{}, 0, 4)
	for axis, coefficient := range normal {
		if coefficient == 0 {
			continue
		}
		exponents := []int{0, 0, 0}
		exponents[axis] = 1
		terms = append(terms, map[string]interface{}{"exponents": exponents, "coefficient": coefficient})
	}
	if constant != 0 {
		terms = append(terms, map[string]interface{}{"exponents": []int{0, 0, 0}, "coefficient": constant})
	}

	adapted := cloneMap(object)
	adapted["shape"] = "polynomial"
	adapted["degree"] = 1
	adapted["terms"] = terms
	delete(adapted, "normal")
	delete(adapted, "point")
	delete(adapted, "offset")
	return adaptPolynomial(adapted, ctx, dimension)
}

func adaptQuadraticSurface(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("quadratic surface adapter requires dimension 3, got %d", dimension)
	}
	matrix, err := quadraticMatrix(object, "matrix", dimension)
	if err != nil {
		return nil, err
	}
	linear, err := optionalVector(object, "linear", dimension, zeroVector(dimension))
	if err != nil {
		return nil, err
	}
	constant := 0.0
	if _, ok := object["constant"]; ok {
		constant, err = floatField(object, "constant")
		if err != nil {
			return nil, err
		}
	}

	terms := make([]map[string]interface{}, 0, 10)
	hasQuadratic := false
	for first := 0; first < dimension; first++ {
		for second := first; second < dimension; second++ {
			coefficient := matrix[first][second]
			if first != second {
				coefficient *= 2
			}
			if coefficient == 0 {
				continue
			}
			hasQuadratic = true
			exponents := []int{0, 0, 0}
			exponents[first]++
			exponents[second]++
			terms = append(terms, map[string]interface{}{"exponents": exponents, "coefficient": coefficient})
		}
	}
	if !hasQuadratic {
		return nil, fmt.Errorf("field %q must contain a non-zero quadratic coefficient", "matrix")
	}
	for axis, coefficient := range linear {
		if coefficient == 0 {
			continue
		}
		exponents := []int{0, 0, 0}
		exponents[axis] = 1
		terms = append(terms, map[string]interface{}{"exponents": exponents, "coefficient": coefficient})
	}
	if constant != 0 {
		terms = append(terms, map[string]interface{}{"exponents": []int{0, 0, 0}, "coefficient": constant})
	}

	adapted := cloneMap(object)
	adapted["shape"] = "polynomial"
	adapted["degree"] = 2
	adapted["terms"] = terms
	delete(adapted, "matrix")
	delete(adapted, "linear")
	delete(adapted, "constant")
	return adaptPolynomial(adapted, ctx, dimension)
}

func quadraticMatrix(object map[string]interface{}, key string, dimension int) ([][]float64, error) {
	raw, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}
	rows, ok := raw.([]interface{})
	if !ok || len(rows) != dimension {
		return nil, fmt.Errorf("field %q must be a %dx%d matrix", key, dimension, dimension)
	}
	matrix := make([][]float64, dimension)
	for row, rawRow := range rows {
		values, err := toFloat64Slice(rawRow)
		if err != nil || len(values) != dimension {
			return nil, fmt.Errorf("field %q row %d must contain %d numbers", key, row, dimension)
		}
		for col, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("field %q[%d][%d] must be finite", key, row, col)
			}
		}
		matrix[row] = values
	}
	for row := 0; row < dimension; row++ {
		for col := row + 1; col < dimension; col++ {
			if !nearlyEqual(matrix[row][col], matrix[col][row]) {
				return nil, fmt.Errorf("field %q must be symmetric", key)
			}
		}
	}
	return matrix, nil
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
	worldPmin := make([]float64, len(pmin))
	worldPmax := make([]float64, len(pmin))
	for axis := range worldPmin {
		worldPmin[axis] = math.Inf(1)
		worldPmax[axis] = math.Inf(-1)
	}
	for mask := 0; mask < 1<<len(pmin); mask++ {
		corner := make([]float64, len(pmin))
		for axis := range corner {
			corner[axis] = pmin[axis]
			if mask&(1<<axis) != 0 {
				corner[axis] = pmax[axis]
			}
		}
		world := applyPlacement(ctx, corner)
		for axis, value := range world {
			worldPmin[axis] = math.Min(worldPmin[axis], value)
			worldPmax[axis] = math.Max(worldPmax[axis], value)
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
	if !basisIsIdentity(ctx.basis) {
		return nil, fmt.Errorf("cuboid does not support rotated group placement; use triangles for an oriented box")
	}
	shapeName, _ := stringField(object, "shape")
	isHypercube := strings.EqualFold(shapeName, "hypercube")

	if _, hasPmin := object["pmin"]; hasPmin {
		if _, hasPmax := object["pmax"]; hasPmax && !isHypercube && groupPlacementIsIdentity(ctx) {
			adapted := cloneMap(object)
			adapted["shape"] = "cuboid"
			return adapted, nil
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

	worldPmin, worldPmax := placedMinMax(ctx, pmin, pmax)

	adapted := cloneMap(object)
	adapted["shape"] = "cuboid"
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
	adapted["shape"] = "sphere"
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

	adapted["shape"] = "polynomial"
	adapted["degree"] = 2
	adapted["terms"] = spherePolynomialTerms()
	adapted["transform"] = transformToSlices(spherePolynomialTransform(ctx, center, radius))
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
	normal, err := vectorField(object, "normal", dimension)
	if err != nil {
		return nil, err
	}
	scale, ok := uniformPlacementScale(ctx)
	if !ok {
		return nil, fmt.Errorf("circle does not support non-uniform group scale")
	}

	adapted := cloneMap(object)
	adapted["center"] = applyPlacement(ctx, center)
	adapted["normal"] = applyDirection(ctx, normal)
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
	axis, err := vectorField(object, "axis", dimension)
	if err != nil {
		return nil, err
	}
	scale, ok := uniformPlacementScale(ctx)
	if !ok {
		return nil, fmt.Errorf("finite cylinder does not support non-uniform group scale")
	}

	adapted := cloneMap(object)
	adapted["shape"] = "cylinder"
	adapted["center"] = applyPlacement(ctx, center)
	adapted["axis"] = applyDirection(ctx, axis)
	adapted["r"] = radius * scale
	adapted["height"] = height * scale
	delete(adapted, "position")
	return adapted, nil
}

func adaptImplicitEquation(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("implicit equation adapter requires dimension 3, got %d", dimension)
	}

	adapted := cloneMap(object)
	normalizeImplicitFieldAlias(adapted)
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

func normalizeImplicitFieldAlias(object map[string]interface{}) {
	rawField, ok := object["field"]
	if !ok {
		return
	}
	field, ok := rawField.(map[string]interface{})
	if !ok {
		return
	}
	fieldType, ok := stringField(field, "type")
	if !ok {
		return
	}
	if strings.EqualFold(fieldType, "lp_norm") {
		field["type"] = "lp_power_sum"
	}
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

func adaptParametricCurve(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("parametric curve adapter requires dimension 3, got %d", dimension)
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

func adaptPolynomial(object map[string]interface{}, ctx groupContext, dimension int) (map[string]interface{}, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("polynomial adapter requires dimension 3, got %d", dimension)
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

func spherePolynomialTerms() []map[string]interface{} {
	return []map[string]interface{}{
		{"exponents": []int{2, 0, 0}, "coefficient": 1.0},
		{"exponents": []int{0, 2, 0}, "coefficient": 1.0},
		{"exponents": []int{0, 0, 2}, "coefficient": 1.0},
		{"exponents": []int{0, 0, 0}, "coefficient": -1.0},
	}
}

func spherePolynomialTransform(ctx groupContext, localCenter []float64, radius float64) [4][4]float64 {
	worldCenter := applyPlacement(ctx, localCenter)
	transform := [4][4]float64{{1, 0, 0, 0}}
	for localAxis := 0; localAxis < 3; localAxis++ {
		axisScale := ctx.scale[localAxis] * radius
		for worldAxis := 0; worldAxis < 3; worldAxis++ {
			coefficient := ctx.basis[worldAxis][localAxis] / axisScale
			transform[localAxis+1][worldAxis+1] = coefficient
			transform[localAxis+1][0] -= coefficient * worldCenter[worldAxis]
		}
	}
	return transform
}
