package factory

import (
	"fmt"
	"math"
	"strings"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type parametricSurfaceFactory func(map[string]interface{}, [3]float64, [3]float64) (shape.ParametricFunction, shape.ParametricDerivative, [2]float64, [2]float64, error)

var parametricSurfaceRegistry = map[string]parametricSurfaceFactory{
	"plane_patch":   parseParametricPlanePatch,
	"sphere":        parseParametricSphere,
	"spiral_flower": parseParametricSpiralFlower,
	"torus":         parseParametricTorus,
}

func parseParametricEquation(objDef map[string]interface{}) ([]shape.Shape, error) {
	if utils.Dimension != 3 {
		return nil, fmt.Errorf("shape %q requires render dimension 3, got %d", ShapeParametricEquation, utils.Dimension)
	}

	center, scale, err := parsePolynomialCenterScale(objDef)
	if err != nil {
		return nil, err
	}

	surfaceDef, surfaceType, err := parametricSurfaceDefinition(objDef)
	if err != nil {
		return nil, err
	}
	factory, ok := parametricSurfaceRegistry[strings.ToLower(surfaceType)]
	if !ok {
		return nil, fmt.Errorf("unsupported parametric surface %q", surfaceType)
	}

	function, derivative, defaultURange, defaultVRange, err := factory(surfaceDef, center, scale)
	if err != nil {
		return nil, err
	}
	uRange, err := optionalRange(objDef, "u_range", defaultURange)
	if err != nil {
		return nil, err
	}
	vRange, err := optionalRange(objDef, "v_range", defaultVRange)
	if err != nil {
		return nil, err
	}

	equation := shape.NewParametricEquation(function, uRange, vRange)
	equation.Derivative = derivative
	if err := applyParametricOptions(equation, objDef); err != nil {
		return nil, err
	}
	if err := equation.Validate(); err != nil {
		return nil, err
	}
	return wrapSingleShapeWithBounds(equation, objDef)
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

	surfaceType, err := utils.RequiredStringField(objDef, "function")
	if err != nil {
		return nil, "", err
	}
	return objDef, surfaceType, nil
}

func parseParametricPlanePatch(
	objDef map[string]interface{},
	center, scale [3]float64,
) (shape.ParametricFunction, shape.ParametricDerivative, [2]float64, [2]float64, error) {
	function := func(u, v float64) *mat.VecDense {
		return mat.NewVecDense(3, []float64{
			center[0] + scale[0]*u,
			center[1] + scale[1]*v,
			center[2],
		})
	}
	derivative := func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
		du.SetVec(0, scale[0])
		du.SetVec(1, 0)
		du.SetVec(2, 0)
		dv.SetVec(0, 0)
		dv.SetVec(1, scale[1])
		dv.SetVec(2, 0)
		return du, dv
	}
	return function, derivative, [2]float64{-1, 1}, [2]float64{-1, 1}, nil
}

func parseParametricSphere(
	objDef map[string]interface{},
	center, scale [3]float64,
) (shape.ParametricFunction, shape.ParametricDerivative, [2]float64, [2]float64, error) {
	radius, err := optionalPositiveFloatField(objDef, "r", 1)
	if err != nil {
		return nil, nil, [2]float64{}, [2]float64{}, err
	}
	function := func(u, v float64) *mat.VecDense {
		cosV := math.Cos(v)
		return mat.NewVecDense(3, []float64{
			center[0] + scale[0]*radius*cosV*math.Cos(u),
			center[1] + scale[1]*radius*cosV*math.Sin(u),
			center[2] + scale[2]*radius*math.Sin(v),
		})
	}
	derivative := func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
		cosV := math.Cos(v)
		sinV := math.Sin(v)
		du.SetVec(0, -scale[0]*radius*cosV*math.Sin(u))
		du.SetVec(1, scale[1]*radius*cosV*math.Cos(u))
		du.SetVec(2, 0)
		dv.SetVec(0, -scale[0]*radius*sinV*math.Cos(u))
		dv.SetVec(1, -scale[1]*radius*sinV*math.Sin(u))
		dv.SetVec(2, scale[2]*radius*cosV)
		return du, dv
	}
	return function, derivative, [2]float64{0, 2 * math.Pi}, [2]float64{-0.5 * math.Pi, 0.5 * math.Pi}, nil
}

func parseParametricTorus(
	objDef map[string]interface{},
	center, scale [3]float64,
) (shape.ParametricFunction, shape.ParametricDerivative, [2]float64, [2]float64, error) {
	majorRadius, minorRadius, err := parseImplicitTorusRadii(objDef)
	if err != nil {
		return nil, nil, [2]float64{}, [2]float64{}, err
	}
	function := func(u, v float64) *mat.VecDense {
		ring := majorRadius + minorRadius*math.Cos(v)
		return mat.NewVecDense(3, []float64{
			center[0] + scale[0]*ring*math.Cos(u),
			center[1] + scale[1]*ring*math.Sin(u),
			center[2] + scale[2]*minorRadius*math.Sin(v),
		})
	}
	derivative := func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
		ring := majorRadius + minorRadius*math.Cos(v)
		du.SetVec(0, -scale[0]*ring*math.Sin(u))
		du.SetVec(1, scale[1]*ring*math.Cos(u))
		du.SetVec(2, 0)
		dv.SetVec(0, -scale[0]*minorRadius*math.Sin(v)*math.Cos(u))
		dv.SetVec(1, -scale[1]*minorRadius*math.Sin(v)*math.Sin(u))
		dv.SetVec(2, scale[2]*minorRadius*math.Cos(v))
		return du, dv
	}
	return function, derivative, [2]float64{0, 2 * math.Pi}, [2]float64{0, 2 * math.Pi}, nil
}

func parseParametricSpiralFlower(
	objDef map[string]interface{},
	center, scale [3]float64,
) (shape.ParametricFunction, shape.ParametricDerivative, [2]float64, [2]float64, error) {
	function := func(r, theta float64) *mat.VecDense {
		edgePhase := math.Mod(3.6*theta, 2*math.Pi)
		edge := 1 - 0.5*math.Pow(1-edgePhase/math.Pi, 4) + math.Sin(15*theta)/150
		f2 := 2 * math.Pow(r*r-r, 2)
		alpha := 0.5 * math.Pi * math.Exp(-theta/(8*math.Pi))
		sinAlpha := math.Sin(alpha)
		cosAlpha := math.Cos(alpha)
		h := f2 * sinAlpha
		radius := sinAlpha*r + cosAlpha*h
		height := cosAlpha*r - sinAlpha*h

		return mat.NewVecDense(3, []float64{
			center[0] + scale[0]*edge*radius*math.Cos(theta),
			center[1] + scale[1]*edge*radius*math.Sin(theta),
			center[2] + scale[2]*edge*height,
		})
	}
	return function, nil, [2]float64{0, 1}, [2]float64{4 * math.Pi, 24 * math.Pi}, nil
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

func optionalPositiveFloatField(objDef map[string]interface{}, key string, fallback float64) (float64, error) {
	value, ok, err := utils.OptionalFloat64Field(objDef, key)
	if err != nil || !ok {
		return fallback, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("field %q must be > 0", key)
	}
	return value, nil
}
