package factory

import (
	"fmt"
	"math"
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
	"expr":   parseImplicitExprField,
	"torus":  parseImplicitTorusField,
	"gyroid": parseImplicitGyroidField,
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

	functionName, err := utils.RequiredStringField(objDef, "function")
	if err != nil {
		return nil, "", err
	}
	return objDef, functionName, nil
}

func parseImplicitTorusField(
	fieldDef map[string]interface{},
	center, scale [3]float64,
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	majorRadius, minorRadius, err := parseImplicitTorusRadii(fieldDef)
	if err != nil {
		return nil, nil, err
	}
	return implicitTorusFunction(center, scale, majorRadius, minorRadius),
		implicitTorusGradient(center, scale, majorRadius),
		nil
}

func parseImplicitGyroidField(
	fieldDef map[string]interface{},
	center, scale [3]float64,
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	frequency, offset, err := parseImplicitGyroidParameters(fieldDef)
	if err != nil {
		return nil, nil, err
	}
	return implicitGyroidFunction(center, scale, frequency, offset),
		implicitGyroidGradient(center, scale, frequency),
		nil
}

func parseImplicitTorusRadii(objDef map[string]interface{}) (float64, float64, error) {
	majorRadius := 0.58
	minorRadius := 0.22

	if value, ok, err := utils.OptionalFloat64Field(objDef, "major_radius"); err != nil {
		return 0, 0, err
	} else if ok {
		majorRadius = value
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "minor_radius"); err != nil {
		return 0, 0, err
	} else if ok {
		minorRadius = value
	}
	if majorRadius <= 0 || minorRadius <= 0 {
		return 0, 0, fmt.Errorf("torus radii must be > 0")
	}
	if minorRadius >= majorRadius {
		return 0, 0, fmt.Errorf("minor_radius must be smaller than major_radius")
	}
	return majorRadius, minorRadius, nil
}

func parseImplicitGyroidParameters(objDef map[string]interface{}) (float64, float64, error) {
	frequency := 3.2
	offset := 0.0

	if value, ok, err := utils.OptionalFloat64Field(objDef, "frequency"); err != nil {
		return 0, 0, err
	} else if ok {
		frequency = value
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "offset"); err != nil {
		return 0, 0, err
	} else if ok {
		offset = value
	}
	if frequency <= 0 {
		return 0, 0, fmt.Errorf("frequency must be > 0")
	}
	return frequency, offset, nil
}

func implicitTorusFunction(center, scale [3]float64, majorRadius, minorRadius float64) func(*mat.VecDense) float64 {
	return func(point *mat.VecDense) float64 {
		x, y, z := implicitLocalPoint(point, center, scale)
		ringDistance := math.Sqrt(x*x+y*y) - majorRadius
		return ringDistance*ringDistance + z*z - minorRadius*minorRadius
	}
}

func implicitTorusGradient(center, scale [3]float64, majorRadius float64) func(*mat.VecDense, *mat.VecDense) *mat.VecDense {
	return func(point, res *mat.VecDense) *mat.VecDense {
		if res == nil || res.Len() != point.Len() {
			res = mat.NewVecDense(point.Len(), nil)
		} else {
			res.Zero()
		}

		x, y, z := implicitLocalPoint(point, center, scale)
		q := math.Sqrt(x*x + y*y)
		if q > utils.EPS {
			common := 2 * (q - majorRadius)
			res.SetVec(0, common*x/q/scale[0])
			res.SetVec(1, common*y/q/scale[1])
		}
		res.SetVec(2, 2*z/scale[2])
		return res
	}
}

func implicitGyroidFunction(center, scale [3]float64, frequency, offset float64) func(*mat.VecDense) float64 {
	return func(point *mat.VecDense) float64 {
		x, y, z := implicitLocalPoint(point, center, scale)
		x *= frequency
		y *= frequency
		z *= frequency
		return math.Sin(x)*math.Cos(y) + math.Sin(y)*math.Cos(z) + math.Sin(z)*math.Cos(x) - offset
	}
}

func implicitGyroidGradient(center, scale [3]float64, frequency float64) func(*mat.VecDense, *mat.VecDense) *mat.VecDense {
	return func(point, res *mat.VecDense) *mat.VecDense {
		if res == nil || res.Len() != point.Len() {
			res = mat.NewVecDense(point.Len(), nil)
		} else {
			res.Zero()
		}

		x, y, z := implicitLocalPoint(point, center, scale)
		x *= frequency
		y *= frequency
		z *= frequency

		res.SetVec(0, frequency*(math.Cos(x)*math.Cos(y)-math.Sin(z)*math.Sin(x))/scale[0])
		res.SetVec(1, frequency*(-math.Sin(x)*math.Sin(y)+math.Cos(y)*math.Cos(z))/scale[1])
		res.SetVec(2, frequency*(-math.Sin(y)*math.Sin(z)+math.Cos(z)*math.Cos(x))/scale[2])
		return res
	}
}

func implicitLocalPoint(point *mat.VecDense, center, scale [3]float64) (float64, float64, float64) {
	return (point.AtVec(0) - center[0]) / scale[0],
		(point.AtVec(1) - center[1]) / scale[1],
		(point.AtVec(2) - center[2]) / scale[2]
}
