package factory

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/Algo2147483647/ray/engine/controller/parser"
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
	"expr":         parseImplicitExprField,
	"gyroid":       parseImplicitGyroidField,
	"lp_power_sum": parseImplicitLPPowerSumField,
	"metaballs":    parseImplicitMetaballsField,
}

func parseImplicitEquation(spec *parser.ImplicitEquationSpec, boundsSpec *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	transform, err := parseImplicitTransform(spec, dimension)
	if err != nil {
		return nil, err
	}

	bounds, ok, err := parseShapeBounds(boundsSpec, dimension)
	if err != nil {
		return nil, err
	}

	function, gradient, err := buildImplicitField(spec.Field)
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
	equation.Dimension = dimension
	equation.Transform = transform
	if spec.Step != nil {
		if *spec.Step <= 0 {
			return nil, fmt.Errorf("step must be > 0")
		}
		equation.Step = *spec.Step
	}
	if spec.ValueTol != nil {
		if *spec.ValueTol <= 0 {
			return nil, fmt.Errorf("value_tol must be > 0")
		}
		equation.ValueTol = *spec.ValueTol
	}

	return []shape.Shape{equation}, nil
}

func buildImplicitField(
	raw json.RawMessage,
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	fieldDef, err := decodeRawObject(raw, "field")
	if err != nil {
		return nil, nil, err
	}
	fieldType, err := utils.RequiredStringField(fieldDef, "type")
	if err != nil {
		return nil, nil, err
	}

	factory, ok := implicitFieldRegistry[strings.ToLower(fieldType)]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported implicit field %q", fieldType)
	}
	return factory(fieldDef)
}

func parseImplicitTransform(spec *parser.ImplicitEquationSpec, dimension int) ([4][4]float64, error) {
	if len(spec.Transform) > 0 {
		return parsePolynomialSurfaceTransform(spec.Transform)
	}
	center, scale, err := parsePlacement(spec.Center, spec.Scale, dimension)
	if err != nil {
		return [4][4]float64{}, err
	}
	basis, err := parseImplicitBasis(spec.Basis)
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

func parseImplicitBasis(raw json.RawMessage) ([3][3]float64, error) {
	basis := [3][3]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	if len(raw) == 0 {
		return basis, nil
	}
	var rows [][]float64
	if err := json.Unmarshal(raw, &rows); err != nil {
		return [3][3]float64{}, fmt.Errorf("field %q: expected array: %w", "basis", err)
	}
	if len(rows) != 3 {
		return [3][3]float64{}, fmt.Errorf("field %q must contain 3 vectors, got %d", "basis", len(rows))
	}
	for row, values := range rows {
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
