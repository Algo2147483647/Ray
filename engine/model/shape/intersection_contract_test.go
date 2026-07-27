package shape

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
)

func TestIntersectRejectsInvalidRange(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{0, 0, 0}), 1)

	_, ok := sphere.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, -3}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		IntersectOptions{Range: Interval{Min: math.NaN(), Max: 10}},
	)
	if ok {
		t.Fatal("expected NaN range to be rejected")
	}

	_, ok = sphere.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, -3}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		NewIntersectOptions(10, 1),
	)
	if ok {
		t.Fatal("expected inverted range to be rejected")
	}
}

func TestIntersectGeodesicReturnsMissForUnsupportedShape(t *testing.T) {
	triangle := NewTriangle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
	)

	_, ok := triangle.IntersectGeodesic(
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		geometry.Spherical(),
		NewIntersectOptions(1e-6, math.Pi),
	)
	if ok {
		t.Fatal("expected unsupported great-circle triangle query to miss")
	}
}

func TestIntersectGeodesicRejectsUnsupportedGeometry(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{0, 0, 0}), 1)

	_, ok := sphere.IntersectGeodesic(
		mat.NewVecDense(3, []float64{0, 0, -3}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		geometry.Euclidean(),
		NewIntersectOptions(1e-6, 10),
	)
	if ok {
		t.Fatal("expected Euclidean geometry to be rejected by IntersectGeodesic")
	}
}
