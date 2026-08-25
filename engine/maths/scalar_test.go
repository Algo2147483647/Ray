package maths

import (
	"math"
	"testing"
)

func TestScalarUtilities(t *testing.T) {
	if !IsFinite(1) || IsFinite(math.Inf(1)) || IsFinite(math.NaN()) {
		t.Fatal("IsFinite returned an unexpected result")
	}
	if got := Clamp(2, -1, 1); got != 1 {
		t.Fatalf("Clamp returned %v", got)
	}
	if got := ClampUnit(-0.5); got != 0 {
		t.Fatalf("ClampUnit returned %v", got)
	}
	if got := Lerp(2, 6, 0.25); got != 3 {
		t.Fatalf("Lerp returned %v", got)
	}
	if got := PositiveMod(-1, 4); got != 3 {
		t.Fatalf("PositiveMod returned %v", got)
	}
	if !SignChanged(-1, 1) || SignChanged(0, 1) {
		t.Fatal("SignChanged returned an unexpected result")
	}
}

func TestIdentityTransform4(t *testing.T) {
	transform := IdentityTransform4()
	for row := range transform {
		for column := range transform[row] {
			want := 0.0
			if row == column {
				want = 1
			}
			if transform[row][column] != want {
				t.Fatalf("transform[%d][%d] = %v, want %v", row, column, transform[row][column], want)
			}
		}
	}
}
