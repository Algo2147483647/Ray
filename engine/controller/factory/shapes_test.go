package factory

import (
	"math"
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

	interaction, ok := quartic.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
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

	interaction, ok := cubic.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
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
		"mode":      "implicit",
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
		"mode":      "implicit",
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
		"mode":      "implicit",
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
