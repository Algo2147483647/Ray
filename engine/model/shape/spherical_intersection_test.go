package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestSphereSphericalCandidateHitsGreatCircle(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(4, []float64{0, 1, 0, 0}), 0.1)

	candidate, ok := sphere.IntersectSphericalCandidate(
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		1e-6,
		math.Pi,
	)
	if !ok {
		t.Fatal("expected S^3 sphere hit")
	}
	if math.Abs(candidate.ArcLength-math.Asin(1-0.1*0.1/2)) > 1e-6 {
		t.Fatalf("unexpected S^3 sphere arc length: %.15f", candidate.ArcLength)
	}
	if math.Abs(mat.Norm(candidate.Point, 2)-1) > 1e-9 {
		t.Fatalf("S^3 hit point left unit sphere: norm=%f", mat.Norm(candidate.Point, 2))
	}
}

func TestPlaneSphericalCandidateSolvesLinearGreatCircle(t *testing.T) {
	plane := &Plane{A: mat.NewVecDense(4, []float64{0, 1, 0, 0}), B: -0.5}

	candidate, ok := plane.IntersectSphericalCandidate(
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		1e-6,
		math.Pi,
	)
	if !ok {
		t.Fatal("expected S^3 plane hit")
	}
	if math.Abs(candidate.ArcLength-math.Pi/6) > 1e-6 {
		t.Fatalf("S^3 plane arc length = %.15f, want pi/6", candidate.ArcLength)
	}
	if math.Abs(candidate.Point.AtVec(1)-0.5) > 1e-6 {
		t.Fatalf("S^3 plane hit y = %.15f, want 0.5", candidate.Point.AtVec(1))
	}
}

func TestCuboidSphericalCandidateFindsFirstCoordinateSlab(t *testing.T) {
	cuboid := NewCuboid(
		mat.NewVecDense(4, []float64{0.7, 0.4, -0.2, -0.2}),
		mat.NewVecDense(4, []float64{1.0, 0.6, 0.2, 0.2}),
	)

	candidate, ok := cuboid.IntersectSphericalCandidate(
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		1e-6,
		math.Pi,
	)
	if !ok {
		t.Fatal("expected S^3 cuboid hit")
	}
	if math.Abs(candidate.ArcLength-math.Asin(0.4)) > 1e-6 {
		t.Fatalf("S^3 cuboid arc length = %.15f, want asin(0.4)", candidate.ArcLength)
	}
}
