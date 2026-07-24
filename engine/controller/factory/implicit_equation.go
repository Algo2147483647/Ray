package factory

import (
	"fmt"
	"math"
	"strings"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type implicitFieldFactory func(map[string]interface{}) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
)

var implicitFieldRegistry = map[string]implicitFieldFactory{
	"expr": parseImplicitExprField,
}

func parseImplicitEquation(objDef map[string]interface{}) ([]shape.Shape, error) {
	transform, err := parseImplicitTransform(objDef)
	if err != nil {
		return nil, err
	}

	bounds, ok, err := parseShapeBounds(objDef)
	if err != nil {
		return nil, err
	}

	function, gradient, err := buildImplicitField(objDef)
	if err != nil {
		return nil, err
	}

	var implicitRange [2]*mat.VecDense
	if ok {
		implicitRange = [2]*mat.VecDense{bounds.Pmin, bounds.Pmax}
	}
	equation := shape.NewImplicitEquationWithGradient(
		function,
		gradient,
		implicitRange,
	)
	equation.Transform = transform
	if step, ok, err := utils.OptionalFloat64Field(objDef, "step"); err != nil {
		return nil, err
	} else if ok {
		if step <= 0 {
			return nil, fmt.Errorf("step must be > 0")
		}
		equation.Step = step
	}
	if valueTol, ok, err := utils.OptionalFloat64Field(objDef, "value_tol"); err != nil {
		return nil, err
	} else if ok {
		if valueTol <= 0 {
			return nil, fmt.Errorf("value_tol must be > 0")
		}
		equation.ValueTol = valueTol
	}

	return []shape.Shape{equation}, nil
}

func buildImplicitField(
	objDef map[string]interface{},
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	fieldDef, fieldType, err := implicitFieldDefinition(objDef)
	if err != nil {
		return nil, nil, err
	}

	factory, ok := implicitFieldRegistry[strings.ToLower(fieldType)]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported implicit field %q", fieldType)
	}
	return factory(fieldDef)
}

func implicitFieldDefinition(objDef map[string]interface{}) (map[string]interface{}, string, error) {
	if fieldDef, ok, err := utils.OptionalMapField(objDef, "field"); err != nil {
		return nil, "", err
	} else if ok {
		fieldType, err := utils.RequiredStringField(fieldDef, "type")
		if err != nil {
			return nil, "", err
		}
		return fieldDef, fieldType, nil
	}

	if _, ok := objDef["function"]; ok {
		return nil, "", fmt.Errorf(`field "function" is no longer supported; use "field" with type "expr"`)
	}
	return nil, "", fmt.Errorf(`implicit equation requires "field" with type "expr"`)
}

func parseImplicitTransform(objDef map[string]interface{}) ([4][4]float64, error) {
	if _, ok := objDef["transform"]; ok {
		return parsePolynomialSurfaceTransform(objDef)
	}

	center, scale, err := parsePolynomialCenterScale(objDef)
	if err != nil {
		return [4][4]float64{}, err
	}
	basis, err := parseImplicitBasis(objDef)
	if err != nil {
		return [4][4]float64{}, err
	}

	transform := identityTransform4()
	for localAxis := 0; localAxis < 3; localAxis++ {
		transform[localAxis+1][0] = 0
		for worldAxis := 0; worldAxis < 3; worldAxis++ {
			transform[localAxis+1][0] -= basis[localAxis][worldAxis] * center[worldAxis] / scale[localAxis]
			transform[localAxis+1][worldAxis+1] = basis[localAxis][worldAxis] / scale[localAxis]
		}
	}
	return transform, nil
}

func parseImplicitBasis(objDef map[string]interface{}) ([3][3]float64, error) {
	basis := [3][3]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	raw, ok := objDef["basis"]
	if !ok {
		return basis, nil
	}
	rows, ok := raw.([]interface{})
	if !ok {
		return [3][3]float64{}, fmt.Errorf("field %q: expected array, got %T", "basis", raw)
	}
	if len(rows) != 3 {
		return [3][3]float64{}, fmt.Errorf("field %q must contain 3 vectors, got %d", "basis", len(rows))
	}
	for row, rawRow := range rows {
		values, err := utils.ToFloat64Slice(rawRow)
		if err != nil {
			return [3][3]float64{}, fmt.Errorf("basis[%d]: %w", row, err)
		}
		if len(values) != 3 {
			return [3][3]float64{}, fmt.Errorf("basis[%d] must contain 3 values, got %d", row, len(values))
		}
		copy(basis[row][:], values)
	}
	if err := validateImplicitBasis(basis); err != nil {
		return [3][3]float64{}, err
	}
	return basis, nil
}

func validateImplicitBasis(basis [3][3]float64) error {
	const tol = 1e-6
	for row := 0; row < 3; row++ {
		lengthSquared := 0.0
		for axis := 0; axis < 3; axis++ {
			value := basis[row][axis]
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("basis[%d][%d] must be finite", row, axis)
			}
			lengthSquared += value * value
		}
		if math.Abs(lengthSquared-1) > tol {
			return fmt.Errorf("basis[%d] must be unit length", row)
		}
		for other := row + 1; other < 3; other++ {
			dot := 0.0
			for axis := 0; axis < 3; axis++ {
				dot += basis[row][axis] * basis[other][axis]
			}
			if math.Abs(dot) > tol {
				return fmt.Errorf("basis[%d] and basis[%d] must be orthogonal", row, other)
			}
		}
	}
	return nil
}
