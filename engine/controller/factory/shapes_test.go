package factory

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/model/shape"
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
