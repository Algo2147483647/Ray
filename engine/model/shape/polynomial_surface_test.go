package shape

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
)

func TestPolynomialSurfaceImplicitSphereIntersection(t *testing.T) {
	coefficients, err := maths.NewSparseTensorFromEntries([]int{3, 3, 3}, maths.SparseTensorHash, []maths.SparseTensorEntry[float64]{
		{Index: []int{2, 0, 0}, Value: 1},
		{Index: []int{0, 2, 0}, Value: 1},
		{Index: []int{0, 0, 2}, Value: 1},
		{Index: []int{0, 0, 0}, Value: -1},
	})
	if err != nil {
		t.Fatalf("create coefficients: %v", err)
	}

	surface := NewPolynomialSurface(PolynomialSurfaceImplicit, 3, 1, 2, coefficients)
	interaction, ok := surface.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, -3}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		0,
		10,
	)
	if !ok {
		t.Fatal("expected ray to hit polynomial sphere")
	}
	if math.Abs(interaction.Distance-2) > 1e-9 {
		t.Fatalf("expected distance 2, got %.12f", interaction.Distance)
	}
	if math.Abs(interaction.GeometricNormal.AtVec(2)+1) > 1e-9 {
		t.Fatalf("expected normal to face negative z, got %v", interaction.GeometricNormal.RawVector().Data)
	}
}

func TestPolynomialSurfaceExplicitParaboloid(t *testing.T) {
	coefficients, err := maths.NewSparseTensorFromEntries([]int{3, 3}, maths.SparseTensorHash, []maths.SparseTensorEntry[float64]{
		{Index: []int{2, 0}, Value: 1},
		{Index: []int{0, 2}, Value: 1},
	})
	if err != nil {
		t.Fatalf("create coefficients: %v", err)
	}

	surface := NewPolynomialSurface(PolynomialSurfaceExplicit, 2, 1, 2, coefficients)
	if got := surface.Evaluate([]float64{2, 3}); got != 13 {
		t.Fatalf("expected polynomial value 13, got %v", got)
	}

	gradient := surface.Gradient([]float64{2, 3})
	if gradient[0] != 4 || gradient[1] != 6 {
		t.Fatalf("expected gradient [4 6], got %v", gradient)
	}

	interaction, ok := surface.IntersectRange(
		mat.NewVecDense(3, []float64{1, 1, 5}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
		0,
		10,
	)
	if !ok {
		t.Fatal("expected ray to hit explicit polynomial surface")
	}
	if math.Abs(interaction.Distance-3) > 1e-9 {
		t.Fatalf("expected distance 3, got %.12f", interaction.Distance)
	}
}
