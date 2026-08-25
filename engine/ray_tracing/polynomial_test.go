package ray_tracing

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/shape"
)

func testLinearPolynomial(t *testing.T, linear [3]float64, constant float64) *shape.Polynomial {
	t.Helper()
	entries := make([]maths.SparseTensorEntry[float64], 0, 4)
	for axis, coefficient := range linear {
		if coefficient == 0 {
			continue
		}
		exponents := []int{0, 0, 0}
		exponents[axis] = 1
		entries = append(entries, maths.SparseTensorEntry[float64]{Index: exponents, Value: coefficient})
	}
	if constant != 0 {
		entries = append(entries, maths.SparseTensorEntry[float64]{Index: []int{0, 0, 0}, Value: constant})
	}
	coefficients, err := maths.NewSparseTensorFromEntries([]int{2, 2, 2}, maths.SparseTensorHash, entries)
	if err != nil {
		t.Fatalf("create linear polynomial: %v", err)
	}
	return shape.NewPolynomial(coefficients)
}
