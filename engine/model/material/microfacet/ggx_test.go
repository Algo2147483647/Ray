package microfacet

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
)

func TestGGXMaskingVanishesAtGrazingIncidence(t *testing.T) {
	ggx := NewGGX(0.3)
	grazing := maths.NewDirection(1, 0, math.SmallestNonzeroFloat64)
	if lambda := ggx.Lambda(grazing); !math.IsInf(lambda, 1) {
		t.Fatalf("Lambda at grazing incidence = %g, want +Inf", lambda)
	}
	if g1 := ggx.G1(grazing); g1 != 0 {
		t.Fatalf("G1 at grazing incidence = %g, want 0", g1)
	}
}
