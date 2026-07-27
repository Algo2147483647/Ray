package factory

import (
	"math"
	"sync"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
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

func TestParseShapeTriangleUsesPointsDirectly(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "triangle",
		"p1":    []interface{}{0, 0, 0},
		"p2":    []interface{}{1, 0, 0},
		"p3":    []interface{}{0, 1, 0},
	})
	if err != nil {
		t.Fatalf("parse triangle: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}
	triangle, ok := shapes[0].(*shape.Triangle)
	if !ok {
		t.Fatalf("expected *shape.Triangle, got %T", shapes[0])
	}
	if triangle.P1.AtVec(0) != 0 || triangle.P1.AtVec(1) != 0 || triangle.P1.AtVec(2) != 0 {
		t.Fatalf("unexpected p1: %v", triangle.P1.RawVector().Data)
	}
	if triangle.P2.AtVec(0) != 1 || triangle.P2.AtVec(1) != 0 || triangle.P2.AtVec(2) != 0 {
		t.Fatalf("unexpected p2: %v", triangle.P2.RawVector().Data)
	}
	if triangle.P3.AtVec(0) != 0 || triangle.P3.AtVec(1) != 1 || triangle.P3.AtVec(2) != 0 {
		t.Fatalf("unexpected p3: %v", triangle.P3.RawVector().Data)
	}
}

func TestParseShapeRejectsHypercube(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "hypercube",
		"pmin":  []interface{}{-1, -1, -1},
		"pmax":  []interface{}{1, 1, 1},
	})
	if err == nil {
		t.Fatal("expected engine to reject studio-only hypercube")
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

func TestParseShapeKleinBottle4D(t *testing.T) {
	oldDim := utils.Dimension
	utils.SetDimension(4)
	t.Cleanup(func() { utils.SetDimension(oldDim) })

	shapes, err := ParseShape(map[string]interface{}{
		"shape":     "klein_bottle",
		"center":    []interface{}{0, 1, 2, 3},
		"r_major":   1.5,
		"r_minor":   0.5,
		"thickness": 0.06,
	})
	if err != nil {
		t.Fatalf("parse Klein bottle: %v", err)
	}
	if len(shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(shapes))
	}

	klein, ok := shapes[0].(*shape.KleinBottle4D)
	if !ok {
		t.Fatalf("expected *shape.KleinBottle4D, got %T", shapes[0])
	}
	if klein.Center.AtVec(3) != 3 || klein.R != 1.5 || klein.Minor != 0.5 || klein.Thickness != 0.06 {
		t.Fatalf("unexpected Klein bottle data: center=%v major=%f minor=%f thickness=%f", klein.Center.RawVector().Data, klein.R, klein.Minor, klein.Thickness)
	}
}

func TestParseShapeKleinBottle4DRequiresDimension4(t *testing.T) {
	oldDim := utils.Dimension
	utils.SetDimension(3)
	t.Cleanup(func() { utils.SetDimension(oldDim) })

	_, err := ParseShape(map[string]interface{}{
		"shape":     "klein_bottle",
		"center":    []interface{}{0, 0, 0, 0},
		"r_major":   1.5,
		"r_minor":   0.5,
		"thickness": 0.06,
	})
	if err == nil {
		t.Fatal("expected Klein bottle to require render dimension 4")
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

	interaction, ok := cubic.Intersect(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		shape.NewIntersectOptions(0, math.MaxFloat64),
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

	interaction, ok := quartic.Intersect(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		shape.NewIntersectOptions(0, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected sparse four-order equation to hit")
	}
	if math.Abs(interaction.Distance-2) > 1e-8 {
		t.Fatalf("expected hit at distance 2, got %f", interaction.Distance)
	}
}

func TestParseShapeFourOrderEquationIgnoresAuthoringTransform(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "four-order equation",
		"a": map[string]interface{}{
			"1, 1, 1, 1": 1,
			"0, 0, 0, 0": -1,
		},
		"center": []interface{}{2, 0, 0},
		"scale":  []interface{}{3, 1, 1},
		"basis": []interface{}{
			[]interface{}{0, 0, 1},
			[]interface{}{0, 1, 0},
			[]interface{}{-1, 0, 0},
		},
	})
	if err != nil {
		t.Fatalf("parse four-order equation basis: %v", err)
	}
	quartic, ok := shapes[0].(*shape.FourOrderEquation)
	if !ok {
		t.Fatalf("expected *shape.FourOrderEquation, got %T", shapes[0])
	}

	interaction, ok := quartic.Intersect(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		shape.NewIntersectOptions(0, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected canonical four-order equation to hit")
	}
	if math.Abs(interaction.Distance-1) > 1e-8 {
		t.Fatalf("expected hit at distance 1, got %f", interaction.Distance)
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

func TestParseShapeCubicEquationUsesBakedCoefficientsDirectly(t *testing.T) {
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

	interaction, ok := cubic.Intersect(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		shape.NewIntersectOptions(0, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected cubic to hit")
	}
	if math.Abs(interaction.Distance-1) > 1e-8 {
		t.Fatalf("expected engine to use coefficients directly and hit at x=1, got distance %f", interaction.Distance)
	}
}

func TestParseShapePolynomialSurface(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape":     "polynomial surface",
		"input_dim": 3,
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

func TestParseShapePolynomialSurfaceTransform(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape":     "polynomial surface",
		"input_dim": 3,
		"transform": []interface{}{
			[]interface{}{1, 0, 0, 0},
			[]interface{}{0, math.Sqrt(3) / 2, 0, 0.5},
			[]interface{}{0, 0, 1, 0},
			[]interface{}{0, -0.5, 0, math.Sqrt(3) / 2},
		},
		"coefficients": map[string]interface{}{
			"format": "coo",
			"terms": []interface{}{
				map[string]interface{}{"index": []interface{}{0, 0, 1}, "value": 1},
			},
		},
		"material_id": "unused",
	})
	if err != nil {
		t.Fatalf("parse polynomial surface basis: %v", err)
	}
	surface, ok := shapes[0].(*shape.PolynomialSurface)
	if !ok {
		t.Fatalf("expected polynomial surface, got %T", shapes[0])
	}
	if math.Abs(surface.Transform[3][1]+0.5) > 1e-12 {
		t.Fatalf("expected parsed transform to be preserved, got %v", surface.Transform)
	}
}

func TestParseShapeRejectsInvalidPolynomialSurfaceTransform(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":     "polynomial surface",
		"input_dim": 3,
		"transform": []interface{}{
			[]interface{}{1, 0, 0, 0},
			[]interface{}{0, 1, 0, 0},
			[]interface{}{0, 0, 1},
			[]interface{}{0, 0, 0, 1},
		},
		"coefficients": map[string]interface{}{
			"format": "coo",
			"terms": []interface{}{
				map[string]interface{}{"index": []interface{}{0, 0, 1}, "value": 1},
			},
		},
		"material_id": "unused",
	})
	if err == nil {
		t.Fatal("expected non-orthogonal polynomial surface basis to fail")
	}
}

func TestParseShapeRejectsExplicitPolynomialSurfaceMode(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":         "polynomial surface",
		"mode":          "explicit",
		"input_dim":     2,
		"explicit_axis": 2,
		"coefficients": map[string]interface{}{
			"format": "coo",
			"terms": []interface{}{
				map[string]interface{}{"index": []interface{}{2, 0}, "value": 1},
			},
		},
	})
	if err == nil {
		t.Fatal("expected explicit polynomial surface mode to fail")
	}
}

func TestParseShapeImplicitEquationRejectsBuiltInField(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
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
	if err == nil {
		t.Fatal("expected built-in implicit field to fail")
	}
}

func TestParseShapeImplicitEquationRejectsLegacyFunctionField(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":     "implicit equation",
		"function":  "gyroid",
		"frequency": 1,
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err == nil {
		t.Fatal("expected legacy implicit function field to fail")
	}
}

func TestParseShapeParametricEquationRejectsBuiltInFunction(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":    "parametric equation",
		"function": "spiral_flower",
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err == nil {
		t.Fatal("expected built-in parametric function to fail")
	}
}

func TestParseShapeParametricEquationExprPlane(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "parametric equation",
		"surface": map[string]interface{}{
			"type": "expr",
			"x":    "u",
			"y":    "v",
			"z":    "0",
		},
		"u_range":   []interface{}{-1, 1},
		"v_range":   []interface{}{-1, 1},
		"samples_u": 8,
		"samples_v": 8,
	})
	if err != nil {
		t.Fatalf("parse parametric expr plane: %v", err)
	}
	parametric, ok := shapes[0].(*shape.ParametricEquation)
	if !ok {
		t.Fatalf("expected *shape.ParametricEquation, got %T", shapes[0])
	}

	interaction, ok := parametric.Intersect(
		mat.NewVecDense(3, []float64{0.25, -0.5, -2}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		shape.NewIntersectOptions(1e-6, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected parametric expr plane hit")
	}
	if math.Abs(interaction.Distance-2) > 1e-6 {
		t.Fatalf("expected hit at distance 2, got %.12f", interaction.Distance)
	}
	if interaction.DPDU == nil || math.Abs(interaction.DPDU.AtVec(0)-1) > 1e-9 {
		t.Fatalf("expected autodiff dpdu, got %v", interaction.DPDU)
	}
}

func TestParseShapeParametricEquationExprExplicitDerivative(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "parametric equation",
		"surface": map[string]interface{}{
			"type": "expr",
			"x":    "u*u",
			"y":    "v",
			"z":    "0",
			"derivative": map[string]interface{}{
				"du": map[string]interface{}{"x": "2*u", "y": "0", "z": "0"},
				"dv": map[string]interface{}{"x": "0", "y": "1", "z": "0"},
			},
		},
	})
	if err != nil {
		t.Fatalf("parse parametric expr explicit derivative: %v", err)
	}
	parametric := shapes[0].(*shape.ParametricEquation)
	du, dv := parametric.Derivative(0.5, 0.25, mat.NewVecDense(3, nil), mat.NewVecDense(3, nil))
	if du == nil || dv == nil {
		t.Fatal("expected explicit derivative")
	}
	if math.Abs(du.AtVec(0)-1) > 1e-9 || math.Abs(dv.AtVec(1)-1) > 1e-9 {
		t.Fatalf("unexpected derivative: du=%v dv=%v", du.RawVector().Data, dv.RawVector().Data)
	}
}

func TestParseShapeParametricEquationExprConstantsTorus(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "parametric equation",
		"surface": map[string]interface{}{
			"type": "expr",
			"x":    "(R + r*cos(v))*cos(u)",
			"y":    "(R + r*cos(v))*sin(u)",
			"z":    "r*sin(v)",
			"constants": map[string]interface{}{
				"R": 2.0,
				"r": 0.5,
			},
		},
		"u_range":   []interface{}{0, 2 * math.Pi},
		"v_range":   []interface{}{0, 2 * math.Pi},
		"samples_u": 48,
		"samples_v": 24,
	})
	if err != nil {
		t.Fatalf("parse parametric expr torus: %v", err)
	}
	parametric := shapes[0].(*shape.ParametricEquation)
	interaction, ok := parametric.Intersect(
		mat.NewVecDense(3, []float64{0, -3, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		shape.NewIntersectOptions(1e-6, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected parametric expr torus hit")
	}
	if math.Abs(interaction.Distance-0.5) > 1e-5 {
		t.Fatalf("expected nearest hit at 0.5, got %.12f", interaction.Distance)
	}
}

func TestParseShapeParametricEquationExprNumericalDerivativeFallback(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "parametric equation",
		"surface": map[string]interface{}{
			"type": "expr",
			"x":    "floor(u) + u",
			"y":    "v",
			"z":    "0",
		},
	})
	if err != nil {
		t.Fatalf("parse parametric expr with numerical derivative fallback: %v", err)
	}
	parametric := shapes[0].(*shape.ParametricEquation)
	if parametric.Derivative != nil {
		t.Fatal("expected unsupported autodiff expression to use runtime numerical derivative fallback")
	}
}

func TestParseShapeParametricEquationExprRejectsReservedConstant(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "parametric equation",
		"surface": map[string]interface{}{
			"type": "expr",
			"x":    "u",
			"y":    "v",
			"z":    "0",
			"constants": map[string]interface{}{
				"u": 1,
			},
		},
	})
	if err == nil {
		t.Fatal("expected reserved parametric constant to fail")
	}
}

func TestParseShapeParametricCurveExpr(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "parametric curve",
		"curve": map[string]interface{}{
			"type":   "expr",
			"x":      "0",
			"y":      "t",
			"z":      "0",
			"radius": "r",
			"constants": map[string]interface{}{
				"r": 0.25,
			},
			"derivative": map[string]interface{}{
				"x": "0",
				"y": "1",
				"z": "0",
			},
		},
		"t_range": []interface{}{-1, 1},
		"samples": 64,
	})
	if err != nil {
		t.Fatalf("parse parametric expr curve: %v", err)
	}
	curve, ok := shapes[0].(*shape.ParametricCurve)
	if !ok {
		t.Fatalf("expected *shape.ParametricCurve, got %T", shapes[0])
	}

	interaction, ok := curve.Intersect(
		mat.NewVecDense(3, []float64{-1, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		shape.NewIntersectOptions(1e-6, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected parametric expr curve hit")
	}
	if math.Abs(interaction.Distance-0.75) > 1e-5 {
		t.Fatalf("expected hit at distance 0.75, got %.12f", interaction.Distance)
	}
	if interaction.DPDU == nil || math.Abs(interaction.DPDU.AtVec(1)-1) > 1e-9 {
		t.Fatalf("expected explicit derivative tangent, got %v", interaction.DPDU)
	}
}

func TestParseShapeParametricCurveAppliesCenterAndUniformScale(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape":  "parametric curve",
		"center": []interface{}{1.72, 0, 2.72},
		"scale":  2.0,
		"curve": map[string]interface{}{
			"type":   "expr",
			"x":      "t",
			"y":      "0",
			"z":      "0",
			"radius": 0.05,
			"derivative": map[string]interface{}{
				"x": "1",
				"y": "0",
				"z": "0",
			},
		},
	})
	if err != nil {
		t.Fatalf("parse placed parametric curve: %v", err)
	}
	curve := shapes[0].(*shape.ParametricCurve)
	point := curve.Function(1)
	if math.Abs(point.AtVec(0)-3.72) > 1e-12 || math.Abs(point.AtVec(1)) > 1e-12 || math.Abs(point.AtVec(2)-2.72) > 1e-12 {
		t.Fatalf("unexpected placed curve point: %v", point)
	}
	if math.Abs(curve.Radius(0)-0.1) > 1e-12 {
		t.Fatalf("expected scaled radius 0.1, got %.12f", curve.Radius(0))
	}
	tangent := curve.Derivative(0, mat.NewVecDense(3, nil))
	if math.Abs(tangent.AtVec(0)-2) > 1e-12 {
		t.Fatalf("expected scaled tangent, got %v", tangent)
	}
}

func TestParseShapeParametricCurveExprAutoDerivativeAndConstantRadius(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "parametric curve",
		"curve": map[string]interface{}{
			"type":   "expr",
			"x":      "sin(t)",
			"y":      "cos(t)",
			"z":      "t",
			"radius": 0.1,
		},
	})
	if err != nil {
		t.Fatalf("parse autodiff parametric expr curve: %v", err)
	}
	curve := shapes[0].(*shape.ParametricCurve)
	if curve.Derivative == nil {
		t.Fatal("expected autodiff derivative")
	}
	tangent := curve.Derivative(0, mat.NewVecDense(3, nil))
	if tangent == nil || math.Abs(tangent.AtVec(0)-1) > 1e-9 || math.Abs(tangent.AtVec(1)) > 1e-9 || math.Abs(tangent.AtVec(2)-1) > 1e-9 {
		t.Fatalf("unexpected autodiff tangent: %v", tangent)
	}
}

func TestParseShapeParametricCurveExprRejectsReservedConstant(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "parametric curve",
		"curve": map[string]interface{}{
			"type":   "expr",
			"x":      "t",
			"y":      "0",
			"z":      "0",
			"radius": 0.1,
			"constants": map[string]interface{}{
				"t": 1,
			},
		},
	})
	if err == nil {
		t.Fatal("expected reserved parametric curve constant to fail")
	}
}

func TestParseShapeImplicitEquationExprField(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "x*x + y*y + z*z - r*r",
			"constants": map[string]interface{}{
				"r": 1.0,
			},
			"gradient": map[string]interface{}{
				"x": "2*x",
				"y": "2*y",
				"z": "2*z",
			},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1.2, -1.2, -1.2},
			"pmax": []interface{}{1.2, 1.2, 1.2},
		},
		"step": 0.01,
	})
	if err != nil {
		t.Fatalf("parse expr implicit equation: %v", err)
	}
	implicit, ok := shapes[0].(*shape.ImplicitEquation)
	if !ok {
		t.Fatalf("expected *shape.ImplicitEquation, got %T", shapes[0])
	}

	interaction, ok := implicit.Intersect(
		mat.NewVecDense(3, []float64{0, 0, -1.2}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		shape.NewIntersectOptions(0, 3),
	)
	if !ok {
		t.Fatal("expected ray to hit expr sphere")
	}
	if math.Abs(interaction.Distance-0.2) > 1e-3 {
		t.Fatalf("expected hit near distance 0.2, got %f", interaction.Distance)
	}
	if math.Abs(interaction.GeometricNormal.AtVec(2)+1) > 1e-6 {
		t.Fatalf("expected analytic expr normal to face negative z, got %v", interaction.GeometricNormal.RawVector().Data)
	}
}

func TestParseShapeImplicitEquationMetaballsField(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "metaballs",
			"k":    2.0,
			"iso":  0.5,
			"balls": []interface{}{
				map[string]interface{}{
					"weight": 1.0,
					"center": []interface{}{0, 0, 0},
				},
			},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-2, -2, -2},
			"pmax": []interface{}{2, 2, 2},
		},
		"step": 0.01,
	})
	if err != nil {
		t.Fatalf("parse metaballs implicit equation: %v", err)
	}
	implicit, ok := shapes[0].(*shape.ImplicitEquation)
	if !ok {
		t.Fatalf("expected *shape.ImplicitEquation, got %T", shapes[0])
	}

	centerValue := implicit.Function(mat.NewVecDense(3, []float64{0, 0, 0}))
	if math.Abs(centerValue-0.5) > 1e-12 {
		t.Fatalf("expected center field value 0.5, got %f", centerValue)
	}
	gradient := implicit.Gradient(
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, nil),
	)
	wantGX := -4 * math.Exp(-2)
	if gradient == nil || math.Abs(gradient.AtVec(0)-wantGX) > 1e-12 || math.Abs(gradient.AtVec(1)) > 1e-12 || math.Abs(gradient.AtVec(2)) > 1e-12 {
		t.Fatalf("unexpected metaballs gradient: %v", gradient)
	}

	interaction, ok := implicit.Intersect(
		mat.NewVecDense(3, []float64{-2, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		shape.NewIntersectOptions(0, 4),
	)
	if !ok {
		t.Fatal("expected ray to hit metaball surface")
	}
	wantDistance := 2 - math.Sqrt(math.Log(2)/2)
	if math.Abs(interaction.Distance-wantDistance) > 0.02 {
		t.Fatalf("expected hit near distance %f, got %f", wantDistance, interaction.Distance)
	}
}

func TestParseShapeImplicitEquationGyroidField(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type":      "gyroid",
			"frequency": 2.0,
			"offset":    0.25,
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-2, -2, -2},
			"pmax": []interface{}{2, 2, 2},
		},
		"step": 0.01,
	})
	if err != nil {
		t.Fatalf("parse gyroid implicit equation: %v", err)
	}
	implicit, ok := shapes[0].(*shape.ImplicitEquation)
	if !ok {
		t.Fatalf("expected *shape.ImplicitEquation, got %T", shapes[0])
	}

	point := mat.NewVecDense(3, []float64{0.2, -0.3, 0.4})
	x, y, z := 2.0*point.AtVec(0), 2.0*point.AtVec(1), 2.0*point.AtVec(2)
	sx, cx := math.Sincos(x)
	sy, cy := math.Sincos(y)
	sz, cz := math.Sincos(z)
	wantValue := sx*cy + sy*cz + sz*cx - 0.25
	if got := implicit.Function(point); math.Abs(got-wantValue) > 1e-12 {
		t.Fatalf("expected gyroid value %f, got %f", wantValue, got)
	}

	gradient := implicit.Gradient(point, mat.NewVecDense(3, nil))
	wantGradient := []float64{
		2.0 * (cx*cy - sz*sx),
		2.0 * (-sx*sy + cy*cz),
		2.0 * (-sy*sz + cz*cx),
	}
	if gradient == nil {
		t.Fatal("expected gyroid analytic gradient")
	}
	for axis, want := range wantGradient {
		if math.Abs(gradient.AtVec(axis)-want) > 1e-12 {
			t.Fatalf("unexpected gyroid gradient: got %v, want %v", gradient.RawVector().Data, wantGradient)
		}
	}
}

func TestParseShapeImplicitEquationLPPowerSumField(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type":   "lp_power_sum",
			"power":  2.0 / 3.0,
			"radius": 1.0,
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-2, -2, -2},
			"pmax": []interface{}{2, 2, 2},
		},
		"step": 0.01,
	})
	if err != nil {
		t.Fatalf("parse lp power sum implicit equation: %v", err)
	}
	implicit, ok := shapes[0].(*shape.ImplicitEquation)
	if !ok {
		t.Fatalf("expected *shape.ImplicitEquation, got %T", shapes[0])
	}

	point := mat.NewVecDense(3, []float64{0.125, -0.216, 0.343})
	wantValue := math.Pow(0.125, 2.0/3.0) + math.Pow(0.216, 2.0/3.0) + math.Pow(0.343, 2.0/3.0) - 1
	if got := implicit.Function(point); math.Abs(got-wantValue) > 1e-12 {
		t.Fatalf("expected lp power sum value %f, got %f", wantValue, got)
	}

	gradient := implicit.Gradient(point, mat.NewVecDense(3, nil))
	wantGradient := []float64{
		(2.0 / 3.0) * math.Pow(0.125, -1.0/3.0),
		-(2.0 / 3.0) * math.Pow(0.216, -1.0/3.0),
		(2.0 / 3.0) * math.Pow(0.343, -1.0/3.0),
	}
	if gradient == nil {
		t.Fatal("expected lp power sum analytic gradient")
	}
	for axis, want := range wantGradient {
		if math.Abs(gradient.AtVec(axis)-want) > 1e-12 {
			t.Fatalf("unexpected lp power sum gradient: got %v, want %v", gradient.RawVector().Data, wantGradient)
		}
	}
}

func TestParseShapeImplicitEquationRejectsLPNormAlias(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type":   "lp_norm",
			"power":  1,
			"radius": 1,
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err == nil {
		t.Fatal("expected engine factory to reject studio-only lp_norm alias")
	}
}

func TestParseShapeImplicitEquationTransformMapsWorldToLocal(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "x",
			"gradient": map[string]interface{}{
				"x": "1",
				"y": "0",
				"z": "0",
			},
		},
		"transform": []interface{}{
			[]interface{}{1, 0, 0, 0},
			[]interface{}{0, 0, 1, 0},
			[]interface{}{0, 1, 0, 0},
			[]interface{}{0, 0, 0, 1},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-2, -2, -2},
			"pmax": []interface{}{2, 2, 2},
		},
		"step": 0.01,
	})
	if err != nil {
		t.Fatalf("parse transformed expr implicit equation: %v", err)
	}
	implicit := shapes[0].(*shape.ImplicitEquation)

	interaction, ok := implicit.Intersect(
		mat.NewVecDense(3, []float64{1, -1, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		shape.NewIntersectOptions(0, 3),
	)
	if !ok {
		t.Fatal("expected transformed implicit plane to hit")
	}
	if math.Abs(interaction.Distance-1) > 1e-3 {
		t.Fatalf("expected transformed plane hit at distance 1, got %f", interaction.Distance)
	}
	if math.Abs(interaction.GeometricNormal.AtVec(1)-1) > 1e-6 {
		t.Fatalf("expected transformed normal to face positive y, got %v", interaction.GeometricNormal.RawVector().Data)
	}
}

func TestParseShapeImplicitEquationBasisBuildsTransform(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "x",
		},
		"center": []interface{}{2, 0, 0},
		"scale":  []interface{}{2, 1, 1},
		"basis": []interface{}{
			[]interface{}{0, 0, 1},
			[]interface{}{0, 1, 0},
			[]interface{}{-1, 0, 0},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-3, -3, -3},
			"pmax": []interface{}{3, 3, 3},
		},
	})
	if err != nil {
		t.Fatalf("parse implicit equation basis: %v", err)
	}
	implicit := shapes[0].(*shape.ImplicitEquation)
	if math.Abs(implicit.Transform[1][3]-0.5) > 1e-12 {
		t.Fatalf("expected local x to use scaled world z, got %v", implicit.Transform)
	}
	if math.Abs(implicit.Transform[3][0]-2) > 1e-12 || math.Abs(implicit.Transform[3][1]+1) > 1e-12 {
		t.Fatalf("expected transformed local z row, got %v", implicit.Transform)
	}
}

func TestParseShapeImplicitEquationExprFieldUsesNumericalGradientFallback(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "floor(x) + x + y*y + z*z - 2",
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1.2, -1.2, -1.2},
			"pmax": []interface{}{1.2, 1.2, 1.2},
		},
	})
	if err != nil {
		t.Fatalf("parse expr implicit equation: %v", err)
	}
	implicit := shapes[0].(*shape.ImplicitEquation)
	if implicit.Gradient(mat.NewVecDense(3, []float64{1.5, 0, 0}), mat.NewVecDense(3, nil)) != nil {
		t.Fatal("expected unsupported autodiff expression to return nil analytic gradient")
	}

	normal := implicit.GetNormalVector(
		mat.NewVecDense(3, []float64{1.5, 0, 0}),
		mat.NewVecDense(3, nil),
	)
	if math.Abs(normal.AtVec(0)-1) > 1e-5 {
		t.Fatalf("expected numerical expr normal to face positive x, got %v", normal.RawVector().Data)
	}
}

func TestParseShapeImplicitEquationExprFieldAutoDiffGradient(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "pow(x, 2) + sin(y) + sqrt(z + 2) - r",
			"constants": map[string]interface{}{
				"r": 2.0,
			},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-2, -2, -2},
			"pmax": []interface{}{2, 2, 2},
		},
	})
	if err != nil {
		t.Fatalf("parse autodiff expr implicit equation: %v", err)
	}
	implicit := shapes[0].(*shape.ImplicitEquation)
	gradient := implicit.Gradient(
		mat.NewVecDense(3, []float64{3, 0, 2}),
		mat.NewVecDense(3, nil),
	)
	if gradient == nil {
		t.Fatal("expected autodiff gradient")
	}
	if math.Abs(gradient.AtVec(0)-6) > 1e-9 {
		t.Fatalf("expected d/dx = 6, got %v", gradient.RawVector().Data)
	}
	if math.Abs(gradient.AtVec(1)-1) > 1e-9 {
		t.Fatalf("expected d/dy = 1, got %v", gradient.RawVector().Data)
	}
	if math.Abs(gradient.AtVec(2)-0.25) > 1e-9 {
		t.Fatalf("expected d/dz = 0.25, got %v", gradient.RawVector().Data)
	}
}

func TestParseShapeImplicitEquationExprFieldAutoDiffAtan2Gradient(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "atan2(y, x) + atan2(z, sqrt(x*x + y*y) - r)",
			"constants": map[string]interface{}{
				"r": 0.5,
			},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-5, -5, -5},
			"pmax": []interface{}{5, 5, 5},
		},
	})
	if err != nil {
		t.Fatalf("parse atan2 autodiff expr implicit equation: %v", err)
	}
	implicit := shapes[0].(*shape.ImplicitEquation)

	x, y, z, r := 2.0, 3.0, 4.0, 0.5
	rho := math.Sqrt(x*x + y*y)
	q := rho - r
	thetaDenom := x*x + y*y
	phiDenom := z*z + q*q
	wantGradient := []float64{
		-y/thetaDenom - z*(x/rho)/phiDenom,
		x/thetaDenom - z*(y/rho)/phiDenom,
		q / phiDenom,
	}

	gradient := implicit.Gradient(
		mat.NewVecDense(3, []float64{x, y, z}),
		mat.NewVecDense(3, nil),
	)
	if gradient == nil {
		t.Fatal("expected atan2 autodiff gradient")
	}
	for axis, want := range wantGradient {
		if math.Abs(gradient.AtVec(axis)-want) > 1e-9 {
			t.Fatalf("unexpected atan2 gradient: got %v, want %v", gradient.RawVector().Data, wantGradient)
		}
	}
}

func TestParseShapeImplicitEquationExprFieldAutoDiffCommonFunctions(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "abs(x) + asin(y) + acos(z/2) + atan(x*y) + sinh(z) + cosh(x) + tanh(y) + log10(x+3)",
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-5, -5, -5},
			"pmax": []interface{}{5, 5, 5},
		},
	})
	if err != nil {
		t.Fatalf("parse common autodiff expr implicit equation: %v", err)
	}
	implicit := shapes[0].(*shape.ImplicitEquation)

	x, y, z := 0.5, 0.25, 0.4
	atanDenom := 1 + x*x*y*y
	wantGradient := []float64{
		1 + y/atanDenom + math.Sinh(x) + 1/((x+3)*math.Log(10)),
		1/math.Sqrt(1-y*y) + x/atanDenom + 1/math.Pow(math.Cosh(y), 2),
		-0.5/math.Sqrt(1-math.Pow(z/2, 2)) + math.Cosh(z),
	}

	gradient := implicit.Gradient(
		mat.NewVecDense(3, []float64{x, y, z}),
		mat.NewVecDense(3, nil),
	)
	if gradient == nil {
		t.Fatal("expected common function autodiff gradient")
	}
	for axis, want := range wantGradient {
		if math.Abs(gradient.AtVec(axis)-want) > 1e-9 {
			t.Fatalf("unexpected common function gradient: got %v, want %v", gradient.RawVector().Data, wantGradient)
		}
	}
}

func TestParseShapeImplicitEquationExprFieldConcurrentEvaluation(t *testing.T) {
	shapes, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "x*1000000 + y*1000 + z + bias",
			"constants": map[string]interface{}{
				"bias": 7.0,
			},
			"gradient": map[string]interface{}{
				"x": "1000000",
				"y": "1000",
				"z": "1",
			},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err != nil {
		t.Fatalf("parse expr implicit equation: %v", err)
	}
	implicit := shapes[0].(*shape.ImplicitEquation)

	const workers = 32
	const iterations = 200
	errCh := make(chan string, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		worker := worker
		go func() {
			defer wg.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				x := float64(worker)
				y := float64(iteration)
				z := float64(worker + iteration)
				point := mat.NewVecDense(3, []float64{x, y, z})
				expected := x*1000000 + y*1000 + z + 7
				if got := implicit.Function(point); got != expected {
					errCh <- "concurrent expr evaluate returned mixed env values"
					return
				}
				gradient := implicit.Gradient(point, mat.NewVecDense(3, nil))
				if gradient == nil || gradient.AtVec(0) != 1000000 || gradient.AtVec(1) != 1000 || gradient.AtVec(2) != 1 {
					errCh <- "concurrent expr gradient returned mixed env values"
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for msg := range errCh {
		t.Fatal(msg)
	}
}

func TestParseShapeImplicitEquationExprFieldRejectsInvalidExpression(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "x + missing",
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err == nil {
		t.Fatal("expected invalid expr field to fail")
	}
}

func TestParseShapeImplicitEquationExprFieldRejectsReservedConstant(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape": "implicit equation",
		"field": map[string]interface{}{
			"type": "expr",
			"expr": "x",
			"constants": map[string]interface{}{
				"sin": 1.0,
			},
		},
		"bounds": map[string]interface{}{
			"pmin": []interface{}{-1, -1, -1},
			"pmax": []interface{}{1, 1, 1},
		},
	})
	if err == nil {
		t.Fatal("expected reserved expr constant to fail")
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

func TestParseShapeRejectsAuthoringBoundsCenterSize(t *testing.T) {
	_, err := ParseShape(map[string]interface{}{
		"shape":    "sphere",
		"position": []interface{}{0, 0, 0},
		"r":        1,
		"bounds": map[string]interface{}{
			"center": []interface{}{0, 0, 0},
			"size":   []interface{}{2, 2, 2},
		},
	})
	if err == nil {
		t.Fatal("expected engine to reject studio authoring bounds")
	}
}
