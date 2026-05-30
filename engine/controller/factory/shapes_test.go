package factory

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

func TestParseShapeCircle(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape":    "circle",
		"position": []interface{}{0, 1, 2},
		"normal":   []interface{}{0, 0, 2},
		"r":        3,
	})
	if err != nil {
		t.Fatalf("parse circle: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}
	circle, ok := shapes[0].(*shape.Circle)
	if !ok {
		t.Fatalf("expected *shape.Circle, got %T", shapes[0])
	}
	if circle.R != 3 {
		t.Fatalf("unexpected radius: %f", circle.R)
	}
	if circle.Center.AtVec(1) != 1 || circle.Normal.AtVec(2) != 1 {
		t.Fatalf("unexpected circle data: center=%v normal=%v", circle.Center.RawVector().Data, circle.Normal.RawVector().Data)
	}
}

func TestParseShapeCircleRejectsZeroNormal(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":    "circle",
		"position": []interface{}{0, 0, 0},
		"normal":   []interface{}{0, 0, 0},
		"r":        1,
	})
	if err == nil {
		t.Fatal("expected zero normal to fail")
	}
}

func TestParseShapeFiniteCylinder(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape":    "finite cylinder",
		"position": []interface{}{0, 1, 2},
		"axis":     []interface{}{0, 0, 2},
		"r":        3,
		"height":   4,
	})
	if err != nil {
		t.Fatalf("parse finite cylinder: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}
	cylinder, ok := shapes[0].(*shape.FiniteCylinder)
	if !ok {
		t.Fatalf("expected *shape.FiniteCylinder, got %T", shapes[0])
	}
	if cylinder.R != 3 || cylinder.Height != 4 {
		t.Fatalf("unexpected cylinder dimensions: r=%f height=%f", cylinder.R, cylinder.Height)
	}
	if cylinder.Center.AtVec(1) != 1 || cylinder.Axis.AtVec(2) != 1 {
		t.Fatalf("unexpected cylinder data: center=%v axis=%v", cylinder.Center.RawVector().Data, cylinder.Axis.RawVector().Data)
	}
}

func TestParseShapeFiniteCylinderRejectsInvalidAxis(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":    "finite cylinder",
		"position": []interface{}{0, 0, 0},
		"axis":     []interface{}{0, 0, 0},
		"r":        1,
		"height":   2,
	})
	if err == nil {
		t.Fatal("expected zero axis to fail")
	}
}

func TestParseShapeWrapsOptionalBounds(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "quadratic equation",
		"a": []interface{}{
			1, 0, 0,
			0, 1, 0,
			0, 0, 1,
		},
		"b": []interface{}{0, 0, 0},
		"c": -1,
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-0.5, -0.25, -0.75},
			"pmax": []interface{}{0.5, 0.25, 0.75},
		},
	})
	if err != nil {
		t.Fatalf("parse bounded quadratic: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}
	bounded, ok := shapes[0].(*shape.BoundedShape)
	if !ok {
		t.Fatalf("expected *shape.BoundedShape, got %T", shapes[0])
	}
	if _, ok := bounded.Shape.(*shape.QuadraticEquation); !ok {
		t.Fatalf("expected bounded quadratic equation, got %T", bounded.Shape)
	}

	pmin, pmax := bounded.BuildBoundingBox()
	if pmin.AtVec(0) != -0.5 || pmin.AtVec(2) != -0.75 || pmax.AtVec(0) != 0.5 || pmax.AtVec(2) != 0.75 {
		t.Fatalf("unexpected bounds: pmin=%v pmax=%v", pmin.RawVector().Data, pmax.RawVector().Data)
	}
}

func TestParseShapeCubicEquation(t *testing.T) {
	coeffs := make([]interface{}, 64)
	for i := range coeffs {
		coeffs[i] = 0
	}
	coeffs[(1*4+1)*4+1] = 1
	coeffs[0] = -1

	shapes, err := ParseShape(map[string]interface{}{
		"shape": "cubic equation",
		"a":     coeffs,
	})
	if err != nil {
		t.Fatalf("parse cubic equation: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}
	if _, ok := shapes[0].(*shape.CubicEquation); !ok {
		t.Fatalf("expected *shape.CubicEquation, got %T", shapes[0])
	}
}

func TestParseShapeCubicEquationSparseFlatCoefficients(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "cubic equation",
		"A": map[string]interface{}{
			"21": 1,
			"0":  -1,
		},
	})
	if err != nil {
		t.Fatalf("parse sparse cubic equation: %v", err)
	}
	cubic, ok := shapes[0].(*shape.CubicEquation)
	if !ok {
		t.Fatalf("expected *shape.CubicEquation, got %T", shapes[0])
	}

	interaction, ok := cubic.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected sparse cubic to hit")
	}
	if math.Abs(interaction.Distance-1) > 1e-8 {
		t.Fatalf("expected hit at distance 1, got %f", interaction.Distance)
	}
}

func TestParseShapeFourOrderEquationSparseCoordinateCoefficients(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "four-order equation",
		"a": map[string]interface{}{
			"1, 1, 1, 1": 1,
			"1, 1, 1, 0": -12,
			"1, 1, 0, 0": 54,
			"1, 0, 0, 0": -108,
			"0, 0, 0, 0": 80,
		},
	})
	if err != nil {
		t.Fatalf("parse sparse four-order equation: %v", err)
	}
	quartic, ok := shapes[0].(*shape.FourOrderEquation)
	if !ok {
		t.Fatalf("expected *shape.FourOrderEquation, got %T", shapes[0])
	}

	interaction, ok := quartic.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected sparse four-order equation to hit")
	}
	if math.Abs(interaction.Distance-2) > 1e-8 {
		t.Fatalf("expected hit at distance 2, got %f", interaction.Distance)
	}
}

func TestParseShapeRejectsInvalidSparsePolynomialCoefficientKey(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "cubic equation",
		"a": map[string]interface{}{
			"8, 4, 5": 123,
		},
	})
	if err == nil {
		t.Fatal("expected invalid sparse coordinate key to fail")
	}
}

func TestParseShapeRejectsDuplicatePolynomialCoefficientFields(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "cubic equation",
		"a":     map[string]interface{}{"0": -1},
		"A":     map[string]interface{}{"21": 1},
	})
	if err == nil {
		t.Fatal("expected duplicate coefficient fields to fail")
	}
}

func TestParseShapeCubicEquationBakesCenterAndScalarScale(t *testing.T) {
	coeffs := make([]interface{}, 64)
	for i := range coeffs {
		coeffs[i] = 0
	}
	coeffs[(1*4+1)*4+1] = 1
	coeffs[0] = -1

	shapes, err := ParseShape(map[string]interface{}{
		"shape":  "cubic equation",
		"a":      coeffs,
		"center": []interface{}{2, 0, 0},
		"scale":  3,
	})
	if err != nil {
		t.Fatalf("parse transformed cubic equation: %v", err)
	}
	cubic, ok := shapes[0].(*shape.CubicEquation)
	if !ok {
		t.Fatalf("expected *shape.CubicEquation, got %T", shapes[0])
	}

	interaction, ok := cubic.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected transformed cubic to hit")
	}
	if math.Abs(interaction.Distance-5) > 1e-8 {
		t.Fatalf("expected hit at world x=5, got distance %f", interaction.Distance)
	}
}

func TestParseShapeRejectsInvalidPolynomialScale(t *testing.T) {
	coeffs := make([]interface{}, 64)
	for i := range coeffs {
		coeffs[i] = 0
	}

	_, err := ParseShape(map[string]interface{}{
		"shape": "cubic equation",
		"a":     coeffs,
		"scale": []interface{}{1, 0, 1},
	})
	if err == nil {
		t.Fatal("expected invalid polynomial scale to fail")
	}
}

func TestParseShapePolynomialSurface(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape":     "polynomial surface",
		"mode":      "implicit",
		"input_dim": 3,
		"degree":    2,
		"coefficients": map[string]interface{}{
			"format": "coo",
			"terms": []interface{}{
				map[string]interface{}{"index": []interface{}{2, 0, 0}, "value": 1},
				map[string]interface{}{"index": []interface{}{0, 2, 0}, "value": 1},
				map[string]interface{}{"index": []interface{}{0, 0, 2}, "value": 1},
				map[string]interface{}{"index": []interface{}{0, 0, 0}, "value": -1},
			},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err != nil {
		t.Fatalf("parse polynomial surface: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}

	bounded, ok := shapes[0].(*shape.BoundedShape)
	if !ok {
		t.Fatalf("expected bounded polynomial surface, got %T", shapes[0])
	}
	if _, ok := bounded.Shape.(*shape.PolynomialSurface); !ok {
		t.Fatalf("expected polynomial surface, got %T", bounded.Shape)
	}
}

func TestParseShapeImplicitEquationFieldRegistry(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type":         "torus",
			"major_radius": 0.6,
			"minor_radius": 0.2,
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err != nil {
		t.Fatalf("parse implicit equation field: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}
	implicit, ok := shapes[0].(*shape.ImplicitEquation)
	if !ok {
		t.Fatalf("expected *shape.ImplicitEquation, got %T", shapes[0])
	}

	interaction, ok := implicit.IntersectRange(
		mat.NewVecDense(3, []float64{0.6, 0, -1}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		0,
		2,
	)
	if !ok {
		t.Fatal("expected ray to hit configured implicit torus")
	}
	if math.Abs(interaction.Distance-0.8) > 1e-3 {
		t.Fatalf("expected hit near distance 0.8, got %f", interaction.Distance)
	}
}

func TestParseShapeImplicitEquationLegacyFunctionField(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape":     "implicit equation",
		"function":  "gyroid",
		"frequency": 1,
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err != nil {
		t.Fatalf("parse legacy implicit equation function: %v", err)
	}
	if _, ok := shapes[0].(*shape.ImplicitEquation); !ok {
		t.Fatalf("expected *shape.ImplicitEquation, got %T", shapes[0])
	}
}

func TestParseShapeImplicitEquationRejectsUnknownField(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "unknown",
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err == nil {
		t.Fatal("expected unknown implicit field to fail")
	}
}

func TestParseShapeRejectsInvalidBounds(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":    "sphere",
		"position": []interface{}{0, 0, 0},
		"r":        1,
		"bounds": map[string]interface{}{
			"pmin": []interface{}{1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err == nil {
		t.Fatal("expected invalid bounds to fail")
	}
}
