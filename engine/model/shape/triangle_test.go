package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestTriangleIntersectRangeRejectsNearAndBehindHits(t *testing.T) {
	triangle := NewTriangle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
	)

	_, ok := triangle.IntersectRange(
		mat.NewVecDense(3, []float64{0.25, 0.25, 1e-8}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
		1e-6,
		math.MaxFloat64,
	)
	if ok {
		t.Fatal("expected near-zero hit to be rejected by tMin")
	}

	_, ok = triangle.IntersectRange(
		mat.NewVecDense(3, []float64{0.25, 0.25, 1}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1e-6,
		math.MaxFloat64,
	)
	if ok {
		t.Fatal("expected hit behind ray to be rejected")
	}
}

func TestTriangleSurfaceInteractionCarriesUVAndDerivatives(t *testing.T) {
	triangle := NewTriangle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
	)

	interaction, ok := triangle.IntersectRange(
		mat.NewVecDense(3, []float64{0.25, 0.5, 1}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
		1e-6,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected triangle hit")
	}
	if math.Abs(interaction.UV[0]-0.25) > 1e-9 || math.Abs(interaction.UV[1]-0.5) > 1e-9 {
		t.Fatalf("unexpected UV: %v", interaction.UV)
	}
	if interaction.DPDU == nil || interaction.DPDV == nil {
		t.Fatal("expected triangle derivatives")
	}
}
