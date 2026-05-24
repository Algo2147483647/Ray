package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestBoundedShapeFindsLaterRootInsideBounds(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{0, 0, 0}), 1)
	bounds := NewCuboid(
		mat.NewVecDense(3, []float64{0.2, -2, -2}),
		mat.NewVecDense(3, []float64{2, 2, 2}),
	)
	bounded := NewBoundedShape(sphere, bounds)

	interaction, ok := bounded.IntersectRange(
		mat.NewVecDense(3, []float64{-2, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected bounded sphere to hit the later root inside bounds")
	}
	if math.Abs(interaction.Distance-3) > 1e-9 {
		t.Fatalf("expected hit at distance 3, got %f", interaction.Distance)
	}
}

func TestBoundedShapeRejectsHitOutsideBounds(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{0, 0, 0}), 1)
	bounds := NewCuboid(
		mat.NewVecDense(3, []float64{1.2, -2, -2}),
		mat.NewVecDense(3, []float64{2, 2, 2}),
	)
	bounded := NewBoundedShape(sphere, bounds)

	_, ok := bounded.IntersectRange(
		mat.NewVecDense(3, []float64{-2, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
	)
	if ok {
		t.Fatal("expected bounded sphere to reject roots outside bounds")
	}
}

func TestBoundedShapeBuildBoundingBoxUsesBounds(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{0, 0, 0}), 10)
	bounds := NewCuboid(
		mat.NewVecDense(3, []float64{-1, -2, -3}),
		mat.NewVecDense(3, []float64{4, 5, 6}),
	)
	bounded := NewBoundedShape(sphere, bounds)

	pmin, pmax := bounded.BuildBoundingBox()
	if pmin.AtVec(0) != -1 || pmin.AtVec(2) != -3 || pmax.AtVec(0) != 4 || pmax.AtVec(2) != 6 {
		t.Fatalf("unexpected bounded box: pmin=%v pmax=%v", pmin.RawVector().Data, pmax.RawVector().Data)
	}
}
