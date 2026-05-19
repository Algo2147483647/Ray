package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestFiniteCylinderIntersectHitsSide(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	distance := cylinder.Intersect(
		mat.NewVecDense(3, []float64{2, 0, 0}),
		mat.NewVecDense(3, []float64{-1, 0, 0}),
	)

	if math.Abs(distance-1) > 1e-9 {
		t.Fatalf("unexpected side hit distance: got %f want 1", distance)
	}
}

func TestFiniteCylinderIntersectHitsCap(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	distance := cylinder.Intersect(
		mat.NewVecDense(3, []float64{0.5, 0, 4}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
	)

	if math.Abs(distance-2) > 1e-9 {
		t.Fatalf("unexpected cap hit distance: got %f want 2", distance)
	}
}

func TestFiniteCylinderIntersectRejectsBeyondHeight(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	distance := cylinder.Intersect(
		mat.NewVecDense(3, []float64{2, 0, 3}),
		mat.NewVecDense(3, []float64{-1, 0, 0}),
	)

	if distance != math.MaxFloat64 {
		t.Fatalf("expected miss beyond cylinder height, got %f", distance)
	}
}

func TestFiniteCylinderNormal(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	sideNormal := cylinder.GetNormalVector(mat.NewVecDense(3, []float64{1, 0, 0}), mat.NewVecDense(3, nil))
	topNormal := cylinder.GetNormalVector(mat.NewVecDense(3, []float64{0.5, 0, 2}), mat.NewVecDense(3, nil))

	if math.Abs(sideNormal.AtVec(0)-1) > 1e-9 || math.Abs(sideNormal.AtVec(2)) > 1e-9 {
		t.Fatalf("unexpected side normal: %v", sideNormal.RawVector().Data)
	}
	if math.Abs(topNormal.AtVec(2)-1) > 1e-9 {
		t.Fatalf("unexpected top normal: %v", topNormal.RawVector().Data)
	}
}

func TestFiniteCylinderBuildBoundingBox(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{1, 2, 3}),
		mat.NewVecDense(3, []float64{0, 0, 2}),
		1,
		4,
	)

	pmin, pmax := cylinder.BuildBoundingBox()
	wantMin := []float64{0, 1, 1}
	wantMax := []float64{2, 3, 5}

	for i := 0; i < 3; i++ {
		if math.Abs(pmin.AtVec(i)-wantMin[i]) > 1e-9 || math.Abs(pmax.AtVec(i)-wantMax[i]) > 1e-9 {
			t.Fatalf("unexpected bbox axis %d: got [%f, %f] want [%f, %f]", i, pmin.AtVec(i), pmax.AtVec(i), wantMin[i], wantMax[i])
		}
	}
}
