package exprdiff

import "testing"

func TestDerivativesSupportsCommonFunctions(t *testing.T) {
	derivatives, ok := Derivatives("abs(x) + atan2(y, x) + sinh(z) + log10(x+3)", "x", "y", "z")
	if !ok {
		t.Fatal("expected derivative")
	}
	if len(derivatives) != 3 {
		t.Fatalf("expected three derivatives, got %d", len(derivatives))
	}
	for i, derivative := range derivatives {
		if derivative == "" {
			t.Fatalf("derivative %d is empty", i)
		}
	}
}

func TestDerivativesRejectsUnsupportedDiscontinuousFunction(t *testing.T) {
	if _, ok := Derivatives("floor(x) + x", "x"); ok {
		t.Fatal("expected floor derivative to be unsupported")
	}
}
