package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestSphereIntersectReturnsNearestCompleteInteraction(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{0, 0, 0}), 1)

	interaction, ok := sphere.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, -3}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected sphere interaction")
	}
	if math.Abs(interaction.Distance-2) > 1e-9 {
		t.Fatalf("expected nearest distance 2, got %f", interaction.Distance)
	}
	if interaction.Point == nil || interaction.GeometricNormal == nil {
		t.Fatal("expected complete interaction")
	}
}

func TestSphereIntersectReturnsCompleteInteraction(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{0, 0, 0}), 1)

	interaction, ok := sphere.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, -3}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected sphere hit")
	}
	if interaction.Point == nil || interaction.GeometricNormal == nil || interaction.ShadingNormal == nil {
		t.Fatal("expected complete sphere interaction")
	}
	if math.Abs(interaction.Point.AtVec(2)+1) > 1e-9 {
		t.Fatalf("unexpected hit point: %v", interaction.Point.RawVector().Data)
	}
}

func TestHypersphereIntersect4D(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(4, []float64{0, 0, 0, 0}), 1)

	interaction, ok := sphere.IntersectAffine(
		mat.NewVecDense(4, []float64{-3, 0, 0, 0}),
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)

	if !ok {
		t.Fatal("expected 4D hypersphere hit")
	}
	if interaction.Point.Len() != 4 || interaction.GeometricNormal.Len() != 4 {
		t.Fatalf("expected 4D interaction, got point=%d normal=%d", interaction.Point.Len(), interaction.GeometricNormal.Len())
	}
	if math.Abs(interaction.Distance-2) > 1e-9 {
		t.Fatalf("expected nearest distance 2, got %f", interaction.Distance)
	}
}
