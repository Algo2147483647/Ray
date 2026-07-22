package maths

import (
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestDistance(t *testing.T) {
	got := Distance(
		mat.NewVecDense(3, []float64{1, 2, 3}),
		mat.NewVecDense(3, []float64{4, 6, 3}),
	)
	assertNear(t, got, 5)
}
