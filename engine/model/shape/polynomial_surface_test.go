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

func TestPolynomialSurfaceTaubinHeartRayPolynomialMatchesEvaluation(t *testing.T) {
	surface := newTaubinHeartSurface(t)

	rays := []struct {
		start []float64
		dir   []float64
		ts    []float64
	}{
		{
			start: []float64{0.11, -2.7, -0.23},
			dir:   []float64{0.03, 1.0, 0.17},
			ts:    []float64{0, 0.5, 1.25, 2.8},
		},
		{
			start: []float64{-0.41, -1.8, 0.72},
			dir:   []float64{0.12, 0.9, -0.08},
			ts:    []float64{-0.2, 0.4, 1.1, 2.0},
		},
	}

	for _, ray := range rays {
		coeffs, err := surface.rayPolynomial(
			mat.NewVecDense(3, ray.start),
			mat.NewVecDense(3, ray.dir),
		)
		if err != nil {
			t.Fatalf("ray polynomial: %v", err)
		}

		for _, tt := range ray.ts {
			point := []float64{
				ray.start[0] + tt*ray.dir[0],
				ray.start[1] + tt*ray.dir[1],
				ray.start[2] + tt*ray.dir[2],
			}
			got := evalDescendingPolynomial(coeffs, tt)
			want := surface.Evaluate(point)
			if math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
				t.Fatalf("ray polynomial mismatch at t=%v: got %.15g want %.15g", tt, got, want)
			}
		}
	}
}

func TestPolynomialSurfaceTaubinHeartIntersection(t *testing.T) {
	surface := newTaubinHeartSurface(t)

	interaction, ok := surface.IntersectRange(
		mat.NewVecDense(3, []float64{0, -3, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		0,
		10,
	)
	if !ok {
		t.Fatal("expected ray through center of Taubin heart to hit")
	}
	if math.Abs(interaction.Distance-7.0/3.0) > 1e-7 {
		t.Fatalf("expected first center hit distance 7/3, got %.12f", interaction.Distance)
	}

	_, ok = surface.IntersectRange(
		mat.NewVecDense(3, []float64{1.4, -3, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		0,
		10,
	)
	if ok {
		t.Fatal("did not expect ray outside Taubin heart silhouette to hit")
	}
}

func newTaubinHeartSurface(t *testing.T) *PolynomialSurface {
	t.Helper()

	terms := map[[3]int]float64{}
	base := map[[3]int]float64{
		{2, 0, 0}: 1,
		{0, 2, 0}: 9.0 / 4.0,
		{0, 0, 2}: 1,
		{0, 0, 0}: -1,
	}

	cubed := multiplyTermMaps(multiplyTermMaps(base, base), base)
	for index, value := range cubed {
		terms[index] += value
	}
	terms[[3]int{2, 0, 3}] -= 1
	terms[[3]int{0, 2, 3}] -= 9.0 / 80.0

	entries := make([]maths.SparseTensorEntry[float64], 0, len(terms))
	for index, value := range terms {
		if value == 0 {
			continue
		}
		entries = append(entries, maths.SparseTensorEntry[float64]{
			Index: []int{index[0], index[1], index[2]},
			Value: value,
		})
	}

	coefficients, err := maths.NewSparseTensorFromEntries([]int{7, 7, 7}, maths.SparseTensorHash, entries)
	if err != nil {
		t.Fatalf("create Taubin heart coefficients: %v", err)
	}
	return NewPolynomialSurface(PolynomialSurfaceImplicit, 3, 1, 6, coefficients)
}

func multiplyTermMaps(a, b map[[3]int]float64) map[[3]int]float64 {
	result := map[[3]int]float64{}
	for ia, va := range a {
		for ib, vb := range b {
			index := [3]int{ia[0] + ib[0], ia[1] + ib[1], ia[2] + ib[2]}
			result[index] += va * vb
		}
	}
	return result
}

func evalDescendingPolynomial(coeffs []float64, x float64) float64 {
	result := 0.0
	for _, coefficient := range coeffs {
		result = result*x + coefficient
	}
	return result
}
