package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

func parseParametricEquation(objDef map[string]interface{}) ([]shape.Shape, error) {
	if utils.Dimension != 3 {
		return nil, fmt.Errorf("shape %q requires render dimension 3, got %d", ShapeParametricEquation, utils.Dimension)
	}

	surfaceDef, surfaceType, err := parametricSurfaceDefinition(objDef)
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

	uRange, err := optionalRange(objDef, "u_range", [2]float64{0, 1})
	if err != nil {
		return nil, err
	}
	vRange, err := optionalRange(objDef, "v_range", [2]float64{0, 1})
	if err != nil {
		return nil, err
	}
	function, derivative, err = applyParametricSurfacePlacement(function, derivative, objDef)
	if err != nil {
		return nil, err
	}

	equation := shape.NewParametricEquation(function, uRange, vRange)
	equation.Derivative = derivative
	if err := applyParametricOptions(equation, objDef); err != nil {
		return nil, err
	}
	return wrapSingleShapeWithBounds(equation, objDef)
}

func applyParametricSurfacePlacement(
	function shape.ParametricFunction,
	derivative shape.ParametricDerivative,
	objDef map[string]interface{},
) (shape.ParametricFunction, shape.ParametricDerivative, error) {
	center, scale, err := parsePolynomialCenterScale(objDef)
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

func parametricSurfaceDefinition(objDef map[string]interface{}) (map[string]interface{}, string, error) {
	if surfaceDef, ok, err := utils.OptionalMapField(objDef, "surface"); err != nil {
		return nil, "", err
	} else if ok {
		surfaceType, err := utils.RequiredStringField(surfaceDef, "type")
		if err != nil {
			return nil, "", err
		}
		return surfaceDef, surfaceType, nil
	}

	if _, ok := objDef["function"]; ok {
		return nil, "", fmt.Errorf(`field "function" is no longer supported; use "surface"`)
	}
	return nil, "", fmt.Errorf(`parametric equation requires "surface"`)
}

func applyParametricOptions(equation *shape.ParametricEquation, objDef map[string]interface{}) error {
	var err error
	if equation.SamplesU, err = optionalPositiveIntField(objDef, "samples_u", equation.SamplesU); err != nil {
		return err
	}
	if equation.SamplesV, err = optionalPositiveIntField(objDef, "samples_v", equation.SamplesV); err != nil {
		return err
	}
	if equation.NewtonMaxIter, err = optionalPositiveIntField(objDef, "newton_max_iter", equation.NewtonMaxIter); err != nil {
		return err
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "newton_tol"); err != nil {
		return err
	} else if ok {
		if value <= 0 {
			return fmt.Errorf("newton_tol must be > 0")
		}
		equation.NewtonTol = value
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "derivative_eps"); err != nil {
		return err
	} else if ok {
		if value <= 0 {
			return fmt.Errorf("derivative_eps must be > 0")
		}
		equation.DerivativeEps = value
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "bounds_padding"); err != nil {
		return err
	} else if ok {
		if value < 0 {
			return fmt.Errorf("bounds_padding must be >= 0")
		}
		equation.BoundsPadding = value
	}
	if value, ok, err := utils.OptionalFloat64Field(objDef, "residual_tol"); err != nil {
		return err
	} else if ok {
		if value <= 0 {
			return fmt.Errorf("residual_tol must be > 0")
		}
		equation.ResidualTol = value
	}
	return nil
}

func optionalRange(objDef map[string]interface{}, key string, fallback [2]float64) ([2]float64, error) {
	values, ok, err := utils.OptionalFloat64SliceField(objDef, key, 2)
	if err != nil || !ok {
		return fallback, err
	}
	if values[0] >= values[1] {
		return [2]float64{}, fmt.Errorf("%s must be increasing", key)
	}
	return [2]float64{values[0], values[1]}, nil
}
