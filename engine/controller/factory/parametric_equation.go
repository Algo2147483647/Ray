package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

func parseParametricEquation(spec *parser.ParametricEquationSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("shape %q requires scene dimension 3, got %d", parser.ShapeParametricEquation.Kind, dimension)
	}

	surfaceDef, err := decodeRawObject(spec.Surface, "surface")
	if err != nil {
		return nil, err
	}
	surfaceType, err := requiredLeafType(surfaceDef)
	if err != nil {
		return nil, err
	}

	var function shape.ParametricFunction
	var derivative shape.ParametricDerivative
	switch surfaceType {
	case "expr":
		function, derivative, err = parseParametricExprSurface(surfaceDef)
		if err != nil {
			return nil, err
		}
	case "spherical_harmonic":
		function, derivative, err = parseParametricSphericalHarmonicSurface(surfaceDef)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported parametric surface %q", surfaceType)
	}

	uRange, err := optionalRangeValues(spec.URange, "u_range", [2]float64{0, 1})
	if err != nil {
		return nil, err
	}
	vRange, err := optionalRangeValues(spec.VRange, "v_range", [2]float64{0, 1})
	if err != nil {
		return nil, err
	}
	function, derivative, err = applyParametricSurfacePlacement(function, derivative, spec, dimension)
	if err != nil {
		return nil, err
	}

	equation := shape.NewParametricEquation(function, uRange, vRange)
	equation.Derivative = derivative
	if err := applyParametricOptions(equation, spec); err != nil {
		return nil, err
	}
	return wrapSingleShapeWithBounds(equation, bounds, dimension)
}

func applyParametricSurfacePlacement(
	function shape.ParametricFunction,
	derivative shape.ParametricDerivative,
	spec *parser.ParametricEquationSpec,
	dimension int,
) (shape.ParametricFunction, shape.ParametricDerivative, error) {
	center, scale, err := parsePlacement(spec.Center, spec.Scale, dimension)
	if err != nil {
		return nil, nil, err
	}

	placedFunction := func(u, v float64) *mat.VecDense {
		point := function(u, v)
		if point == nil || point.Len() < 3 {
			return point
		}
		return mat.NewVecDense(3, []float64{
			center[0] + scale[0]*point.AtVec(0),
			center[1] + scale[1]*point.AtVec(1),
			center[2] + scale[2]*point.AtVec(2),
		})
	}

	var placedDerivative shape.ParametricDerivative
	if derivative != nil {
		placedDerivative = func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
			du, dv = derivative(u, v, du, dv)
			if du == nil || dv == nil || du.Len() < 3 || dv.Len() < 3 {
				return du, dv
			}
			for axis := 0; axis < 3; axis++ {
				du.SetVec(axis, scale[axis]*du.AtVec(axis))
				dv.SetVec(axis, scale[axis]*dv.AtVec(axis))
			}
			return du, dv
		}
	}
	return placedFunction, placedDerivative, nil
}

func applyParametricOptions(equation *shape.ParametricEquation, spec *parser.ParametricEquationSpec) error {
	if err := assignPositiveInt("samples_u", spec.SamplesU, &equation.SamplesU); err != nil {
		return err
	}
	if err := assignPositiveInt("samples_v", spec.SamplesV, &equation.SamplesV); err != nil {
		return err
	}
	if err := assignPositiveInt("newton_max_iter", spec.NewtonMaxIter, &equation.NewtonMaxIter); err != nil {
		return err
	}
	if spec.NewtonTol != nil {
		if *spec.NewtonTol <= 0 {
			return fmt.Errorf("newton_tol must be > 0")
		}
		equation.NewtonTol = *spec.NewtonTol
	}
	if spec.DerivativeEps != nil {
		if *spec.DerivativeEps <= 0 {
			return fmt.Errorf("derivative_eps must be > 0")
		}
		equation.DerivativeEps = *spec.DerivativeEps
	}
	if spec.BoundsPadding != nil {
		if *spec.BoundsPadding < 0 {
			return fmt.Errorf("bounds_padding must be >= 0")
		}
		equation.BoundsPadding = *spec.BoundsPadding
	}
	if spec.ResidualTol != nil {
		if *spec.ResidualTol <= 0 {
			return fmt.Errorf("residual_tol must be > 0")
		}
		equation.ResidualTol = *spec.ResidualTol
	}
	return nil
}

func optionalRangeValues(values []float64, key string, fallback [2]float64) ([2]float64, error) {
	if values == nil {
		return fallback, nil
	}
	if _, err := requiredVector(key, values, 2); err != nil {
		return [2]float64{}, err
	}
	if values[0] >= values[1] {
		return [2]float64{}, fmt.Errorf("%s must be increasing", key)
	}
	return [2]float64{values[0], values[1]}, nil
}

func assignPositiveInt(name string, value *int, target *int) error {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return fmt.Errorf("field %q must be > 0", name)
	}
	*target = *value
	return nil
}

func requiredLeafType(def map[string]interface{}) (string, error) {
	value, ok := def["type"].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("missing required field %q", "type")
	}
	return value, nil
}
