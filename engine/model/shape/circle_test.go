package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestCircleIntersectHitsDisk(t *testing.T) {
	circle := NewCircle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		2,
	)

	distance := circle.Intersect(
		mat.NewVecDense(3, []float64{0.5, 0.5, 3}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
	)

	if math.Abs(distance-3) > 1e-9 {
		t.Fatalf("unexpected hit distance: got %f want 3", distance)
	}
}

func TestCircleIntersectRejectsOutsideDisk(t *testing.T) {
	circle := NewCircle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		2,
	)

	distance := circle.Intersect(
		mat.NewVecDense(3, []float64{3, 0, 3}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
	)

	if distance != math.MaxFloat64 {
		t.Fatalf("expected miss, got distance %f", distance)
	}
}

func TestCircleBuildBoundingBoxForTiltedDisk(t *testing.T) {
	circle := NewCircle(
		mat.NewVecDense(3, []float64{1, 2, 3}),
		mat.NewVecDense(3, []float64{0, 0, 2}),
		2,
	)

	pmin, pmax := circle.BuildBoundingBox()
	wantMin := []float64{-1, 0, 3}
	wantMax := []float64{3, 4, 3}

	for i := 0; i < 3; i++ {
		if math.Abs(pmin.AtVec(i)-wantMin[i]) > 1e-9 || math.Abs(pmax.AtVec(i)-wantMax[i]) > 1e-9 {
			t.Fatalf("unexpected bbox axis %d: got [%f, %f] want [%f, %f]", i, pmin.AtVec(i), pmax.AtVec(i), wantMin[i], wantMax[i])
		}
	}
}
