package factory

import (
	"fmt"
	"strings"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type implicitFieldFactory func(map[string]interface{}, [3]float64, [3]float64) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
)

var implicitFieldRegistry = map[string]implicitFieldFactory{
	"expr": parseImplicitExprField,
}

func parseImplicitEquation(objDef map[string]interface{}) ([]shape.Shape, error) {
	center, scale, err := parsePolynomialCenterScale(objDef)
	if err != nil {
		return nil, err
	}

	bounds, ok, err := parseShapeBounds(objDef)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("implicit equation requires bounds")
	}

	function, gradient, err := buildImplicitField(objDef, center, scale)
	if err != nil {
		return nil, err
	}

	equation := shape.NewImplicitEquationWithGradient(
		function,
		gradient,
		[2]*mat.VecDense{bounds.Pmin, bounds.Pmax},
	)
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
	center, scale [3]float64,
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
	return factory(fieldDef, center, scale)
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

func implicitLocalPoint(point *mat.VecDense, center, scale [3]float64) (float64, float64, float64) {
	return (point.AtVec(0) - center[0]) / scale[0],
		(point.AtVec(1) - center[1]) / scale[1],
		(point.AtVec(2) - center[2]) / scale[2]
}
