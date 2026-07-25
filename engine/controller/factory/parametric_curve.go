package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
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

	curve := shape.NewParametricCurve(function, radius, tRange)
	curve.Derivative = derivative
	if err := applyParametricCurveOptions(curve, objDef); err != nil {
		return nil, err
	}
	return wrapSingleShapeWithBounds(curve, objDef)
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
