package maths

import (
	"math"
	"testing"
)

func TestSolvePolynomialRealUsesDescendingPowerOrder(t *testing.T) {
	roots, err := SolvePolynomialReal([]float64{1, -6, 11, -6})
	if err != nil {
		t.Fatalf("solve polynomial: %v", err)
	}

	expected := []float64{1, 2, 3}
	if len(roots) != len(expected) {
		t.Fatalf("expected roots %v, got %v", expected, roots)
	}

	for i := range expected {
		if math.Abs(roots[i]-expected[i]) > 1e-8 {
			t.Fatalf("expected roots %v, got %v", expected, roots)
		}
	}
}
