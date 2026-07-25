package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

func parseParametricCurve(objDef map[string]interface{}) ([]shape.Shape, error) {
	if utils.Dimension != 3 {
		return nil, fmt.Errorf("shape %q requires render dimension 3, got %d", ShapeParametricCurve, utils.Dimension)
	}

	curveDef, curveType, err := parametricCurveDefinition(objDef)
	if err != nil {
		return nil, err
	}

	var function shape.ParametricCurveFunction
	var derivative shape.ParametricCurveDerivative
	var radius shape.ParametricCurveRadius
	switch curveType {
	case "expr":
		function, derivative, radius, err = parseParametricExprCurve(curveDef)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported parametric curve %q", curveType)
	}

	tRange, err := optionalRange(objDef, "t_range", [2]float64{0, 1})
	if err != nil {
		return nil, err
	}
	function, derivative, radius, err = applyParametricCurvePlacement(function, derivative, radius, objDef)
	if err != nil {
		return nil, err
	}

	curve := shape.NewParametricCurve(function, radius, tRange)
	curve.Derivative = derivative
	if err := applyParametricCurveOptions(curve, objDef); err != nil {
		return nil, err
	}
	return wrapSingleShapeWithBounds(curve, objDef)
}

// Parametric curves are represented as capsules, so their circular cross-section
// permits translation and uniform scale but not a non-uniform affine scale.
func applyParametricCurvePlacement(
	function shape.ParametricCurveFunction,
	derivative shape.ParametricCurveDerivative,
	radius shape.ParametricCurveRadius,
	objDef map[string]interface{},
) (shape.ParametricCurveFunction, shape.ParametricCurveDerivative, shape.ParametricCurveRadius, error) {
	center, scale, err := parsePolynomialCenterScale(objDef)
	if err != nil {
		return nil, nil, nil, err
	}
	if math.Abs(scale[0]-scale[1]) > 1e-12 || math.Abs(scale[0]-scale[2]) > 1e-12 {
		return nil, nil, nil, fmt.Errorf("parametric curve scale must be uniform")
	}

	uniformScale := scale[0]
	placedFunction := func(t float64) *mat.VecDense {
		point := function(t)
		if point == nil || point.Len() < 3 {
			return point
		}
		return mat.NewVecDense(3, []float64{
			center[0] + uniformScale*point.AtVec(0),
			center[1] + uniformScale*point.AtVec(1),
			center[2] + uniformScale*point.AtVec(2),
		})
	}

	var placedDerivative shape.ParametricCurveDerivative
	if derivative != nil {
		placedDerivative = func(t float64, res *mat.VecDense) *mat.VecDense {
			tangent := derivative(t, res)
			if tangent == nil || tangent.Len() < 3 {
				return tangent
			}
			for axis := 0; axis < 3; axis++ {
				tangent.SetVec(axis, uniformScale*tangent.AtVec(axis))
			}
			return tangent
		}
	}
	placedRadius := func(t float64) float64 { return uniformScale * radius(t) }
	return placedFunction, placedDerivative, placedRadius, nil
}

func parametricCurveDefinition(objDef map[string]interface{}) (map[string]interface{}, string, error) {
	if curveDef, ok, err := utils.OptionalMapField(objDef, "curve"); err != nil {
		return nil, "", err
	} else if ok {
		curveType, err := utils.RequiredStringField(curveDef, "type")
		if err != nil {
			return nil, "", err
		}
		return curveDef, curveType, nil
	}

	if _, ok := objDef["function"]; ok {
		return nil, "", fmt.Errorf(`field "function" is no longer supported; use "curve"`)
	}
	return nil, "", fmt.Errorf(`parametric curve requires "curve"`)
}

func applyParametricCurveOptions(curve *shape.ParametricCurve, objDef map[string]interface{}) error {
	var err error
	if curve.Samples, err = optionalPositiveIntField(objDef, "samples", curve.Samples); err != nil {
		return err
	}
	if curve.RefineIter, err = optionalPositiveIntField(objDef, "refine_iter", curve.RefineIter); err != nil {
		return err
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "derivative_eps"); err != nil {
		return err
	} else if ok {
		if value <= 0 {
			return fmt.Errorf("derivative_eps must be > 0")
		}
		curve.DerivativeEps = value
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "bounds_padding"); err != nil {
		return err
	} else if ok {
		if value < 0 {
			return fmt.Errorf("bounds_padding must be >= 0")
		}
		curve.BoundsPadding = value
	}
	return nil
}
