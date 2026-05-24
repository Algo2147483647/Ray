package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
)

func parsePolynomialSurface(objDef map[string]interface{}) ([]shape.Shape, error) {
	modeText, ok, err := utils.OptionalStringField(objDef, "mode")
	if err != nil {
		return nil, err
	}
	if !ok {
		modeText = string(shape.PolynomialSurfaceImplicit)
	}
	mode := shape.PolynomialSurfaceMode(modeText)
	if mode != shape.PolynomialSurfaceImplicit && mode != shape.PolynomialSurfaceExplicit {
		return nil, fmt.Errorf("field %q must be %q or %q", "mode", shape.PolynomialSurfaceImplicit, shape.PolynomialSurfaceExplicit)
	}

	inputDim, err := requiredPositiveIntField(objDef, "input_dim")
	if err != nil {
		return nil, err
	}
	degree, err := requiredNonNegativeIntField(objDef, "degree")
	if err != nil {
		return nil, err
	}
	outputDim, err := optionalPositiveIntField(objDef, "output_dim", 1)
	if err != nil {
		return nil, err
	}
	explicitAxis, err := optionalNonNegativeIntField(objDef, "explicit_axis", 2)
	if err != nil {
		return nil, err
	}

	coefficients, effectiveDegree, err := parsePolynomialSurfaceCoefficients(objDef, inputDim, outputDim, degree)
	if err != nil {
		return nil, err
	}
	if effectiveDegree > degree {
		degree = effectiveDegree
	}

	surface := shape.NewPolynomialSurface(mode, inputDim, outputDim, degree, coefficients)
	surface.ExplicitAxis = explicitAxis

	center, scale, err := parsePolynomialSurfaceCenterScale(objDef, maxInt(inputDim, 3))
	if err != nil {
		return nil, err
	}
	surface.Center = center
	surface.Scale = scale

	return wrapSingleShapeWithBounds(surface, objDef)
}

func parsePolynomialSurfaceCoefficients(
	objDef map[string]interface{},
	inputDim, outputDim, degree int,
) (*maths.SparseTensor[float64], int, error) {
	coeffDef, ok, err := utils.OptionalMapField(objDef, "coefficients")
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		coeffDef = objDef
	}

	formatText, ok, err := utils.OptionalStringField(coeffDef, "format")
	if err != nil {
		return nil, 0, err
	}
	format := maths.SparseTensorHash
	if ok {
		format = maths.SparseTensorFormat(formatText)
	}
	if format != maths.SparseTensorCOO && format != maths.SparseTensorHash {
		return nil, 0, fmt.Errorf("field %q supports %q or %q for polynomial surface coefficients", "format", maths.SparseTensorCOO, maths.SparseTensorHash)
	}

	tensorShape, err := polynomialSurfaceTensorShape(coeffDef, inputDim, outputDim, degree)
	if err != nil {
		return nil, 0, err
	}

	rawTerms, ok := coeffDef["terms"]
	if !ok {
		return nil, 0, fmt.Errorf(`missing required field "terms"`)
	}
	terms, ok := rawTerms.([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("field %q: expected array, got %T", "terms", rawTerms)
	}

	degreePolicy, ok, err := utils.OptionalStringField(coeffDef, "degree_policy")
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		degreePolicy = "total"
	}
	if degreePolicy != "total" && degreePolicy != "per_axis" {
		return nil, 0, fmt.Errorf("field %q must be %q or %q", "degree_policy", "total", "per_axis")
	}

	entries := make([]maths.SparseTensorEntry[float64], 0, len(terms))
	maxTermDegree := degree
	for i, rawTerm := range terms {
		term, ok := rawTerm.(map[string]interface{})
		if !ok {
			return nil, 0, fmt.Errorf("terms[%d]: expected object, got %T", i, rawTerm)
		}

		index, err := requiredIntSliceField(term, "index")
		if err != nil {
			return nil, 0, fmt.Errorf("terms[%d]: %w", i, err)
		}
		if len(index) != len(tensorShape) {
			return nil, 0, fmt.Errorf("terms[%d]: index rank %d does not match coefficient rank %d", i, len(index), len(tensorShape))
		}

		alpha := index
		if outputDim > 1 {
			if index[0] < 0 || index[0] >= outputDim {
				return nil, 0, fmt.Errorf("terms[%d]: output index must be in [0,%d)", i, outputDim)
			}
			alpha = index[1:]
		}
		totalDegree := 0
		for axis, exponent := range alpha {
			if exponent < 0 || exponent > degree {
				return nil, 0, fmt.Errorf("terms[%d]: exponent on axis %d must be in [0,%d]", i, axis, degree)
			}
			totalDegree += exponent
		}
		if degreePolicy == "total" && totalDegree > degree {
			return nil, 0, fmt.Errorf("terms[%d]: total degree %d exceeds declared degree %d", i, totalDegree, degree)
		}
		if totalDegree > maxTermDegree {
			maxTermDegree = totalDegree
		}

		value, err := utils.RequiredFloat64Field(term, "value")
		if err != nil {
			return nil, 0, fmt.Errorf("terms[%d]: %w", i, err)
		}

		entries = append(entries, maths.SparseTensorEntry[float64]{
			Index: index,
			Value: value,
		})
	}

	tensor, err := maths.NewSparseTensorFromEntries(tensorShape, format, entries)
	if err != nil {
		return nil, 0, err
	}
	return tensor, maxTermDegree, nil
}

func polynomialSurfaceTensorShape(coeffDef map[string]interface{}, inputDim, outputDim, degree int) ([]int, error) {
	if rawShape, ok := coeffDef["shape"]; ok {
		values, err := requiredIntSliceValue("shape", rawShape)
		if err != nil {
			return nil, err
		}
		expectedRank := inputDim
		if outputDim > 1 {
			expectedRank++
		}
		if len(values) != expectedRank {
			return nil, fmt.Errorf("field %q rank must be %d, got %d", "shape", expectedRank, len(values))
		}
		for i, value := range values {
			if value <= 0 {
				return nil, fmt.Errorf("shape index %d must be > 0", i)
			}
		}
		return values, nil
	}

	tensorShape := make([]int, inputDim)
	for i := range tensorShape {
		tensorShape[i] = degree + 1
	}
	if outputDim > 1 {
		tensorShape = append([]int{outputDim}, tensorShape...)
	}
	return tensorShape, nil
}

func parsePolynomialSurfaceCenterScale(objDef map[string]interface{}, dimension int) ([]float64, []float64, error) {
	center := make([]float64, dimension)
	if values, ok, err := utils.OptionalFloat64SliceField(objDef, "center"); err != nil {
		return nil, nil, err
	} else if ok {
		if len(values) != dimension {
			return nil, nil, fmt.Errorf("field %q must contain %d values, got %d", "center", dimension, len(values))
		}
		copy(center, values)
	}

	scale := make([]float64, dimension)
	for i := range scale {
		scale[i] = 1
	}
	if value, ok := objDef["scale"]; ok {
		if values, err := utils.ToFloat64Slice(value); err == nil {
			if len(values) != dimension {
				return nil, nil, fmt.Errorf("field %q must contain %d values, got %d", "scale", dimension, len(values))
			}
			copy(scale, values)
		} else {
			scalar, err := utils.RequiredFloat64Field(map[string]interface{}{"scale": value}, "scale")
			if err != nil {
				return nil, nil, err
			}
			for i := range scale {
				scale[i] = scalar
			}
		}
	}
	for i, value := range scale {
		if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, nil, errInvalidScale(i)
		}
	}
	return center, scale, nil
}

func requiredPositiveIntField(data map[string]interface{}, key string) (int, error) {
	value, err := requiredNonNegativeIntField(data, key)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("field %q must be > 0", key)
	}
	return value, nil
}

func requiredNonNegativeIntField(data map[string]interface{}, key string) (int, error) {
	value, err := utils.RequiredFloat64Field(data, key)
	if err != nil {
		return 0, err
	}
	return nonNegativeWholeNumber(key, value)
}

func optionalPositiveIntField(data map[string]interface{}, key string, fallback int) (int, error) {
	value, err := optionalNonNegativeIntField(data, key, fallback)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("field %q must be > 0", key)
	}
	return value, nil
}

func optionalNonNegativeIntField(data map[string]interface{}, key string, fallback int) (int, error) {
	value, ok, err := utils.OptionalFloat64Field(data, key)
	if err != nil || !ok {
		return fallback, err
	}
	return nonNegativeWholeNumber(key, value)
}

func requiredIntSliceField(data map[string]interface{}, key string) ([]int, error) {
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}
	return requiredIntSliceValue(key, value)
}

func requiredIntSliceValue(key string, value interface{}) ([]int, error) {
	values, err := utils.ToFloat64Slice(value)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}

	result := make([]int, len(values))
	for i, value := range values {
		integer, err := nonNegativeWholeNumber(fmt.Sprintf("%s[%d]", key, i), value)
		if err != nil {
			return nil, err
		}
		result[i] = integer
	}
	return result, nil
}

func nonNegativeWholeNumber(key string, value float64) (int, error) {
	if value < 0 || math.Trunc(value) != value {
		return 0, fmt.Errorf("field %q must be a non-negative integer", key)
	}
	return int(value), nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
