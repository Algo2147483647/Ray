package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

func parseParametricCurve(spec *parser.ParametricCurveSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("shape %q requires scene dimension 3, got %d", ShapeParametricCurve, dimension)
	}

	curveDef, err := decodeRawObject(spec.Curve, "curve")
	if err != nil {
		return nil, err
	}
	curveType, err := requiredLeafType(curveDef)
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

	tRange, err := optionalRangeValues(spec.TRange, "t_range", [2]float64{0, 1})
	if err != nil {
		return nil, err
	}
	function, derivative, radius, err = applyParametricCurvePlacement(function, derivative, radius, spec, dimension)
	if err != nil {
		return nil, err
	}

	curve := shape.NewParametricCurve(function, radius, tRange)
	curve.Derivative = derivative
	if err := applyParametricCurveOptions(curve, spec); err != nil {
		return nil, err
	}
	return wrapSingleShapeWithBounds(curve, bounds, dimension)
}

// Parametric curves are represented as capsules, so their circular cross-section
// permits translation and uniform scale but not a non-uniform affine scale.
func applyParametricCurvePlacement(
	function shape.ParametricCurveFunction,
	derivative shape.ParametricCurveDerivative,
	radius shape.ParametricCurveRadius,
	spec *parser.ParametricCurveSpec,
	dimension int,
) (shape.ParametricCurveFunction, shape.ParametricCurveDerivative, shape.ParametricCurveRadius, error) {
	center, scale, err := parsePlacement(spec.Center, spec.Scale, dimension)
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

func applyParametricCurveOptions(curve *shape.ParametricCurve, spec *parser.ParametricCurveSpec) error {
	if err := assignPositiveInt("samples", spec.Samples, &curve.Samples); err != nil {
		return err
	}
	if err := assignPositiveInt("refine_iter", spec.RefineIter, &curve.RefineIter); err != nil {
		return err
	}
	if spec.DerivativeEps != nil {
		if *spec.DerivativeEps <= 0 {
			return fmt.Errorf("derivative_eps must be > 0")
		}
		curve.DerivativeEps = *spec.DerivativeEps
	}
	if spec.BoundsPadding != nil {
		if *spec.BoundsPadding < 0 {
			return fmt.Errorf("bounds_padding must be >= 0")
		}
		curve.BoundsPadding = *spec.BoundsPadding
	}
	return nil
}
